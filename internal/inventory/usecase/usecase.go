package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fekuna/omnipos-pkg/cache"
	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/fekuna/omnipos-product-service/internal/inventory"
	"github.com/fekuna/omnipos-product-service/internal/inventory/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type inventoryUseCase struct {
	repo   inventory.Repository
	cache  *cache.RedisClient
	logger logger.ZapLogger
}

func NewInventoryUseCase(repo inventory.Repository, cache *cache.RedisClient, log logger.ZapLogger) inventory.UseCase {
	return &inventoryUseCase{
		repo:   repo,
		cache:  cache,
		logger: log,
	}
}

func (uc *inventoryUseCase) GetProductInventory(ctx context.Context, merchantID, productID string, storeID *string) (*model.Inventory, error) {
	inv, err := uc.repo.GetByProduct(ctx, merchantID, productID, storeID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		// Return zero inventory if not found, or nil depending on requirement.
		// Usually good to return a zero object.
		return &model.Inventory{
			MerchantID:        merchantID,
			StoreID:           storeID,
			ProductID:         productID,
			Quantity:          0,
			AvailableQuantity: 0,
		}, nil
	}
	return inv, nil
}

func (uc *inventoryUseCase) ListLowStock(ctx context.Context, merchantID string, storeID *string, page, pageSize int) ([]model.Inventory, int, error) {
	return uc.repo.FindAll(ctx, &dto.InventoryFilters{
		MerchantID: merchantID,
		StoreID:    storeID,
		LowStock:   true,
		Page:       page,
		PageSize:   pageSize,
	})
}

func (uc *inventoryUseCase) AdjustInventory(ctx context.Context, input *dto.AdjustInventoryInput) (*model.Inventory, error) {
	// 0. Acquire Lock
	// Key: lock:inventory:{merchantID}:{productID}:{variantID}:{storeID} // simplied to lock per product/variant
	lockKey := fmt.Sprintf("lock:inventory:%s:%s", input.MerchantID, input.ProductID)
	if input.VariantID != nil {
		lockKey += ":" + *input.VariantID
	}
	if input.StoreID != nil {
		lockKey += ":" + *input.StoreID
	}

	lockValue := uuid.New().String()
	// Try to acquire lock with retry mechanism or simple fail
	// Simple retry loop (e.g. 3 attempts)
	acquired := false
	for i := 0; i < 3; i++ {
		ok, err := uc.cache.AcquireLock(ctx, lockKey, lockValue, 5*time.Second)
		if err != nil {
			uc.logger.Error("failed to acquire lock redis error", zap.Error(err))
		}
		if ok {
			acquired = true
			break
		}
		time.Sleep(100 * time.Millisecond) // wait before retry
	}

	if !acquired {
		return nil, errors.New("system busy, please try again later (lock)")
	}

	defer uc.cache.ReleaseLock(ctx, lockKey, lockValue)

	// 1. Get current inventory
	inv, err := uc.repo.GetByProduct(ctx, input.MerchantID, input.ProductID, input.StoreID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if inv == nil {
		// Create new
		inv = &model.Inventory{
			ID:         uuid.New().String(),
			MerchantID: input.MerchantID,
			StoreID:    input.StoreID,
			ProductID:  input.ProductID,
			VariantID:  input.VariantID,
			Quantity:   0,
			UpdatedAt:  now,
		}
	}

	quantityBefore := inv.Quantity
	inv.Quantity += input.QuantityChange
	inv.UpdatedAt = now

	if inv.Quantity < 0 {
		return nil, errors.New("insufficient inventory")
	}

	var refID *string
	if input.ReferenceID != "" {
		refID = &input.ReferenceID
	}
	var refType *string
	if input.ReferenceType != "" {
		refType = &input.ReferenceType
	}

	var createdBy *string
	if input.UserID != "" && input.UserID != "unknown" {
		createdBy = &input.UserID
	}

	// 2. Prepare Movement Log
	movement := &model.InventoryMovement{
		ID:             uuid.New().String(),
		MerchantID:     input.MerchantID,
		StoreID:        input.StoreID,
		ProductID:      input.ProductID,
		VariantID:      input.VariantID,
		MovementType:   "adjustment", // or input derived
		QuantityChange: input.QuantityChange,
		QuantityBefore: quantityBefore,
		QuantityAfter:  inv.Quantity,
		ReferenceType:  refType,
		ReferenceID:    refID,
		Notes:          input.Reason,
		CreatedBy:      createdBy,
		CreatedAt:      now,
	}

	err = uc.repo.AdjustStockWithMovement(ctx, inv, movement)
	if err != nil {
		return nil, err
	}

	return inv, nil
}

func (uc *inventoryUseCase) TransferInventory(ctx context.Context, input *dto.TransferInventoryInput) error {
	// This requires transactional adjustment of two store inventories.
	// Since our repo only supports single 'AdjustStockWithMovement' transactionally,
	// we ideally need a Transaction Manager to wrap two repo calls.
	// TODO: Implement Transaction Manager
	return errors.New("transfer not implemented yet")
}

func (uc *inventoryUseCase) ListMovements(ctx context.Context, filters *dto.MovementFilters) ([]model.InventoryMovement, int, error) {
	return uc.repo.ListMovements(ctx, filters)
}
