package usecase

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fekuna/omnipos-pkg/cache"
	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/fekuna/omnipos-pkg/search"
	"github.com/fekuna/omnipos-product-service/internal/model"
	"github.com/fekuna/omnipos-product-service/internal/product"
	"github.com/fekuna/omnipos-product-service/internal/product/dto"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type productUseCase struct {
	repo   product.Repository
	cache  *cache.RedisClient
	es     *search.Client
	logger logger.ZapLogger
}

func NewProductUseCase(repo product.Repository, cache *cache.RedisClient, es *search.Client, log logger.ZapLogger) product.UseCase {
	return &productUseCase{
		repo:   repo,
		cache:  cache,
		es:     es,
		logger: log,
	}
}

func (uc *productUseCase) CreateProduct(ctx context.Context, input *dto.CreateProductInput) (*model.Product, error) {
	unique, err := uc.repo.IsSKUUnique(ctx, input.MerchantID, input.SKU, "")
	if err != nil {
		return nil, err
	}
	if !unique {
		return nil, errors.New("SKU already exists")
	}

	if input.Barcode != "" {
		unique, err := uc.repo.IsBarcodeUnique(ctx, input.MerchantID, input.Barcode, "")
		if err != nil {
			return nil, err
		}
		if !unique {
			return nil, errors.New("Barcode already exists")
		}
	}

	id := uuid.New().String()
	now := time.Now()

	costPrice := input.CostPrice
	categoryID := &input.CategoryID
	if input.CategoryID == "" {
		categoryID = nil
	}
	barcode := &input.Barcode
	if input.Barcode == "" {
		barcode = nil
	}

	p := &model.Product{
		BaseModel:      model.BaseModel{ID: id, CreatedAt: now, UpdatedAt: now},
		MerchantID:     input.MerchantID,
		CategoryID:     categoryID,
		SKU:            input.SKU,
		Barcode:        barcode,
		Name:           input.Name,
		Description:    &input.Description,
		BasePrice:      input.BasePrice,
		CostPrice:      &costPrice,
		TaxRate:        input.TaxRate,
		HasVariants:    input.HasVariants,
		TrackInventory: input.TrackInventory,
		ImageURL:       &input.ImageURL,
		IsActive:       true,
	}

	err = uc.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	// Invalidate Cache
	go uc.invalidateProductCache(context.Background(), input.MerchantID)

	// Sync to Elastic
	go uc.syncToElastic(context.Background(), p)

	return p, nil
}

func (uc *productUseCase) syncToElastic(ctx context.Context, p *model.Product) {
	if uc.es == nil {
		return
	}
	// Use a simplified model or the same model for indexing
	// Alternatively use a global alias "products" with routing
	// For simplicity, let's use "products" index for all and filter by merchant_id in query
	const indexName = "products"

	// Ensure index exists (maybe do this on startup, but doing lazily here for resilience)
	// In production, run migration script. Here, we try to be helpful.
	// Mapping def:
	mapping := `{
		"mappings": {
			"properties": {
				"merchant_id": { "type": "keyword" },
				"name": { "type": "text" },
				"description": { "type": "text" },
				"sku": { "type": "keyword" },
				"barcode": { "type": "keyword" },
				"base_price": { "type": "double" },
				"created_at": { "type": "date" }
			}
		}
	}`
	_ = uc.es.CreateIndex(ctx, indexName, mapping)

	if err := uc.es.Index(ctx, indexName, p.ID, p); err != nil {
		uc.logger.Error("failed to index product", zap.Error(err))
	}
}

func (uc *productUseCase) GetProduct(ctx context.Context, id string) (*model.Product, error) {
	return uc.repo.FindByID(ctx, id)
}

func (uc *productUseCase) ListProducts(ctx context.Context, filters *dto.ProductFilters) ([]model.Product, int, error) {
	// 1. Generate Cache Key
	cacheKey, err := uc.generateCacheKey(filters)
	if err == nil {
		// 2. Check Cache
		val, err := uc.cache.Client.Get(ctx, cacheKey).Result()
		if err == nil {
			var result struct {
				Products []model.Product
				Count    int
			}
			if err := json.Unmarshal([]byte(val), &result); err == nil {
				// Cache Hit
				return result.Products, result.Count, nil
			}
		}
	}

	// 3. Search via Elastic (if query present)
	if filters.SearchQuery != "" && uc.es != nil {
		// Try ES
		q := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []map[string]interface{}{
						{
							"query_string": map[string]interface{}{
								"query":  fmt.Sprintf("*%s*", filters.SearchQuery),
								"fields": []string{"name^3", "sku", "barcode", "description"},
							},
						},
						{
							"term": map[string]interface{}{
								"merchant_id": filters.MerchantID,
							},
						},
					},
				},
			},
			"from": (filters.Page - 1) * filters.PageSize,
		}
		if filters.PageSize > 0 {
			q["size"] = filters.PageSize
		}

		res, err := uc.es.Search(ctx, "products", q)
		if err == nil {
			// Map hits to products
			var esProducts []model.Product
			for _, hit := range res.Hits.Hits {
				var p model.Product
				if err := json.Unmarshal(hit.Source, &p); err == nil {
					// We might want to fill some runtime fields or just trust ES source
					// The ES source currently matches the model structure (mostly)
					esProducts = append(esProducts, p)
				}
			}
			return esProducts, res.Hits.Total.Value, nil
		}
		// If ES fails, fall through to DB
		uc.logger.Error("ES search failed, falling back to DB", zap.Error(err))
	}

	// 4. DB Query (Fallback or Standard List)
	products, count, err := uc.repo.FindAll(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	// 4. Set Cache
	if cacheKey != "" {
		cacheData := struct {
			Products []model.Product
			Count    int
		}{
			Products: products,
			Count:    count,
		}
		if data, err := json.Marshal(cacheData); err == nil {
			uc.cache.Client.Set(ctx, cacheKey, data, 5*time.Minute)
		}
	}

	return products, count, nil
}

func (uc *productUseCase) generateCacheKey(filters *dto.ProductFilters) (string, error) {
	data, err := json.Marshal(filters)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("products:list:%s:%x", filters.MerchantID, md5.Sum(data)), nil
}

func (uc *productUseCase) invalidateProductCache(ctx context.Context, merchantID string) {
	// Invalidate all list caches for this merchant
	pattern := fmt.Sprintf("products:list:%s:*", merchantID)
	keys, err := uc.cache.Client.Keys(ctx, pattern).Result()
	if err == nil && len(keys) > 0 {
		uc.cache.Client.Del(ctx, keys...)
	}
}

func (uc *productUseCase) UpdateProduct(ctx context.Context, input *dto.UpdateProductInput) (*model.Product, error) {
	p, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("product not found")
	}

	if p.SKU != input.SKU {
		unique, err := uc.repo.IsSKUUnique(ctx, input.MerchantID, input.SKU, p.ID)
		if err != nil {
			return nil, err
		}
		if !unique {
			return nil, errors.New("SKU already exists")
		}
	}

	// Update fields
	p.SKU = input.SKU
	p.Name = input.Name
	p.Description = &input.Description
	p.BasePrice = input.BasePrice
	cost := input.CostPrice
	p.CostPrice = &cost
	p.TaxRate = input.TaxRate
	p.TrackInventory = input.TrackInventory
	p.ImageURL = &input.ImageURL
	p.IsActive = input.IsActive
	if input.CategoryID != "" {
		catID := input.CategoryID
		p.CategoryID = &catID
	} else {
		p.CategoryID = nil
	}
	if input.Barcode != "" {
		bc := input.Barcode
		p.Barcode = &bc
	} else {
		p.Barcode = nil
	}

	p.UpdatedAt = time.Now()
	err = uc.repo.Update(ctx, p)
	if err != nil {
		return nil, err
	}

	// Invalidate Cache
	go uc.invalidateProductCache(context.Background(), p.MerchantID)
	// Sync ES
	go uc.syncToElastic(context.Background(), p)

	return p, nil
}

func (uc *productUseCase) DeleteProduct(ctx context.Context, id string) error {
	// Get product to know merchant ID for invalidation
	p, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return nil // Already deleted
	}

	err = uc.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate Cache
	go uc.invalidateProductCache(context.Background(), p.MerchantID)
	// Remove from ES
	if uc.es != nil {
		go func() {
			err := uc.es.Delete(context.Background(), "products", id)
			if err != nil {
				uc.logger.Error("failed to delete product from ES", zap.Error(err))
			}
		}()
	}

	return nil
}

func (uc *productUseCase) AddVariant(ctx context.Context, input *dto.CreateVariantInput) (*model.ProductVariant, error) {
	// Placeholder
	return nil, nil
}

func (uc *productUseCase) ListVariants(ctx context.Context, productID string) ([]model.ProductVariant, error) {
	// Placeholder
	return nil, nil
}

func (uc *productUseCase) ReserveStock(ctx context.Context, items map[string]int32) error {
	return uc.repo.ReserveStock(ctx, items)
}
