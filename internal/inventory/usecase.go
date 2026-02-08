package inventory

import (
	"context"

	"github.com/fekuna/omnipos-product-service/internal/inventory/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
)

type UseCase interface {
	GetProductInventory(ctx context.Context, merchantID, productID string, storeID *string) (*model.Inventory, error)
	ListLowStock(ctx context.Context, merchantID string, storeID *string, page, pageSize int) ([]model.Inventory, int, error)
	AdjustInventory(ctx context.Context, input *dto.AdjustInventoryInput) (*model.Inventory, error)
	TransferInventory(ctx context.Context, input *dto.TransferInventoryInput) error
	ListMovements(ctx context.Context, filters *dto.MovementFilters) ([]model.InventoryMovement, int, error)
}
