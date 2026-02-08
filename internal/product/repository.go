package product

import (
	"context"

	"github.com/fekuna/omnipos-product-service/internal/model"
	"github.com/fekuna/omnipos-product-service/internal/product/dto"
)

type Repository interface {
	Create(ctx context.Context, product *model.Product) error
	FindByID(ctx context.Context, id string) (*model.Product, error)
	FindAll(ctx context.Context, filters *dto.ProductFilters) ([]model.Product, int, error)
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id string) error

	// Check SKU/Barcode uniqueness
	IsSKUUnique(ctx context.Context, merchantID, sku, excludeID string) (bool, error)
	IsBarcodeUnique(ctx context.Context, merchantID, barcode, excludeID string) (bool, error)

	ReserveStock(ctx context.Context, items map[string]int32) error
}
