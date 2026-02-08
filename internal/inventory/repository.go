package inventory

import (
	"context"

	"github.com/fekuna/omnipos-product-service/internal/inventory/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
)

type Repository interface {
	// Inventory Items
	GetByProduct(ctx context.Context, merchantID, productID string, storeID *string) (*model.Inventory, error)
	BatchGetByProducts(ctx context.Context, merchantID string, productIDs []string, storeID *string) ([]model.Inventory, error)
	FindAll(ctx context.Context, filters *dto.InventoryFilters) ([]model.Inventory, int, error)

	// Core stock operations
	CreateOrUpdate(ctx context.Context, inv *model.Inventory) error

	// Movements / Audit
	LogMovement(ctx context.Context, movement *model.InventoryMovement) error
	ListMovements(ctx context.Context, filters *dto.MovementFilters) ([]model.InventoryMovement, int, error)

	// Transaction support
	AdjustStockWithMovement(ctx context.Context, inv *model.Inventory, movement *model.InventoryMovement) error
}
