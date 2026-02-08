package product

import (
	"context"

	"github.com/fekuna/omnipos-product-service/internal/model"
	"github.com/fekuna/omnipos-product-service/internal/product/dto"
)

type UseCase interface {
	CreateProduct(ctx context.Context, input *dto.CreateProductInput) (*model.Product, error)
	GetProduct(ctx context.Context, id string) (*model.Product, error)
	ListProducts(ctx context.Context, filters *dto.ProductFilters) ([]model.Product, int, error)
	UpdateProduct(ctx context.Context, input *dto.UpdateProductInput) (*model.Product, error)
	DeleteProduct(ctx context.Context, id string) error

	// Variant ops
	AddVariant(ctx context.Context, input *dto.CreateVariantInput) (*model.ProductVariant, error)
	ListVariants(ctx context.Context, productID string) ([]model.ProductVariant, error)

	ReserveStock(ctx context.Context, items map[string]int32) error
}
