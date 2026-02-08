package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fekuna/omnipos-product-service/internal/inventory/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
	"github.com/jmoiron/sqlx"
)

type PGRepository struct {
	DB *sqlx.DB
}

func NewPGRepository(db *sqlx.DB) *PGRepository {
	return &PGRepository{DB: db}
}

func (r *PGRepository) GetByProduct(ctx context.Context, merchantID, productID string, storeID *string) (*model.Inventory, error) {
	var inv model.Inventory
	query := `SELECT * FROM inventory WHERE merchant_id = $1 AND product_id = $2`
	args := []interface{}{merchantID, productID}

	if storeID != nil && *storeID != "" {
		query += ` AND store_id = $3`
		args = append(args, *storeID)
	} else {
		query += ` AND store_id IS NULL`
	}

	err := r.DB.GetContext(ctx, &inv, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil if no record found (caller handles creating defaults)
		}
		return nil, err
	}
	return &inv, nil
}

func (r *PGRepository) BatchGetByProducts(ctx context.Context, merchantID string, productIDs []string, storeID *string) ([]model.Inventory, error) {
	if len(productIDs) == 0 {
		return []model.Inventory{}, nil
	}

	query, args, err := sqlx.In(`
        SELECT * FROM inventory 
        WHERE merchant_id = ? AND product_id IN (?)
    `, merchantID, productIDs)
	if err != nil {
		return nil, err
	}

	if storeID != nil && *storeID != "" {
		query += ` AND store_id = ?`
		args = append(args, *storeID)
	} else {
		query += ` AND store_id IS NULL`
	}

	// Rebind for Postgres ($1, $2...)
	query = r.DB.Rebind(query)

	var items []model.Inventory
	err = r.DB.SelectContext(ctx, &items, query, args...)
	return items, err
}

func (r *PGRepository) FindAll(ctx context.Context, f *dto.InventoryFilters) ([]model.Inventory, int, error) {
	var items []model.Inventory
	var count int

	conditions := []string{}
	args := map[string]interface{}{}

	if f.MerchantID != "" {
		conditions = append(conditions, "merchant_id = :merchant_id")
		args["merchant_id"] = f.MerchantID
	}
	if f.ProductID != "" {
		conditions = append(conditions, "product_id = :product_id")
		args["product_id"] = f.ProductID
	}
	if f.StoreID != nil {
		if *f.StoreID == "" {
			conditions = append(conditions, "store_id IS NULL")
		} else {
			conditions = append(conditions, "store_id = :store_id")
			args["store_id"] = *f.StoreID
		}
	}
	if f.LowStock {
		conditions = append(conditions, "available_quantity <= reorder_point AND reorder_point > 0")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT count(*) FROM inventory" + whereClause
	rows, err := r.DB.NamedQueryContext(ctx, countQuery, args)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&count)
	}

	query := "SELECT * FROM inventory" + whereClause + " ORDER BY updated_at DESC"
	if f.PageSize > 0 {
		offset := (f.Page - 1) * f.PageSize
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.PageSize, offset)
	}

	nstmt, err := r.DB.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer nstmt.Close()

	err = nstmt.SelectContext(ctx, &items, args)
	return items, count, err
}

func (r *PGRepository) CreateOrUpdate(ctx context.Context, inv *model.Inventory) error {
	query := `
        INSERT INTO inventory (
            id, merchant_id, store_id, product_id, variant_id, 
            quantity, reserved_quantity, reorder_point, reorder_quantity, 
            last_counted_at, updated_at
        ) 
        VALUES (
            :id, :merchant_id, :store_id, :product_id, :variant_id, 
            :quantity, :reserved_quantity, :reorder_point, :reorder_quantity, 
            :last_counted_at, :updated_at
        )
        ON CONFLICT (merchant_id, store_id, product_id, variant_id) 
        DO UPDATE SET 
            quantity = EXCLUDED.quantity,
            reserved_quantity = EXCLUDED.reserved_quantity,
            reorder_point = EXCLUDED.reorder_point,
            reorder_quantity = EXCLUDED.reorder_quantity,
            last_counted_at = EXCLUDED.last_counted_at,
            updated_at = EXCLUDED.updated_at
    `
	// Note: available_quantity is generated column, so we don't insert/update it
	_, err := r.DB.NamedExecContext(ctx, query, inv)
	return err
}

func (r *PGRepository) LogMovement(ctx context.Context, m *model.InventoryMovement) error {
	query := `
        INSERT INTO inventory_movements (
            id, merchant_id, store_id, product_id, variant_id, 
            movement_type, quantity_change, quantity_before, quantity_after, 
            reference_type, reference_id, notes, created_by, created_at
        )
        VALUES (
            :id, :merchant_id, :store_id, :product_id, :variant_id, 
            :movement_type, :quantity_change, :quantity_before, :quantity_after, 
            :reference_type, :reference_id, :notes, :created_by, :created_at
        )
    `
	_, err := r.DB.NamedExecContext(ctx, query, m)
	return err
}

func (r *PGRepository) ListMovements(ctx context.Context, f *dto.MovementFilters) ([]model.InventoryMovement, int, error) {
	var items []model.InventoryMovement
	var count int

	conditions := []string{}
	args := map[string]interface{}{}

	if f.MerchantID != "" {
		conditions = append(conditions, "merchant_id = :merchant_id")
		args["merchant_id"] = f.MerchantID
	}
	if f.ProductID != "" {
		conditions = append(conditions, "product_id = :product_id")
		args["product_id"] = f.ProductID
	}
	if f.MovementType != "" {
		conditions = append(conditions, "movement_type = :movement_type")
		args["movement_type"] = f.MovementType
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT count(*) FROM inventory_movements" + whereClause
	rows, err := r.DB.NamedQueryContext(ctx, countQuery, args)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&count)
	}

	query := "SELECT * FROM inventory_movements" + whereClause + " ORDER BY created_at DESC"
	if f.PageSize > 0 {
		offset := (f.Page - 1) * f.PageSize
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.PageSize, offset)
	}

	nstmt, err := r.DB.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer nstmt.Close()

	err = nstmt.SelectContext(ctx, &items, args)
	return items, count, err
}

func (r *PGRepository) AdjustStockWithMovement(ctx context.Context, inv *model.Inventory, movement *model.InventoryMovement) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Update Inventory
	upsertQuery := `
        INSERT INTO inventory (
            id, merchant_id, store_id, product_id, variant_id, 
            quantity, reserved_quantity, reorder_point, reorder_quantity, 
            last_counted_at, updated_at
        ) 
        VALUES (
            :id, :merchant_id, :store_id, :product_id, :variant_id, 
            :quantity, :reserved_quantity, :reorder_point, :reorder_quantity, 
            :last_counted_at, :updated_at
        )
        ON CONFLICT (merchant_id, store_id, product_id, variant_id) 
        DO UPDATE SET 
            quantity = EXCLUDED.quantity,
            reserved_quantity = EXCLUDED.reserved_quantity,
            last_counted_at = EXCLUDED.last_counted_at,
            updated_at = EXCLUDED.updated_at
    `
	// Only updating quantity fields for adjustment usually, but upsert overrides everything.
	// Ensure 'inv' has correct current values if it exists, or new values.

	_, err = tx.NamedExecContext(ctx, upsertQuery, inv)
	if err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	// 2. Log Movement
	insertLogQuery := `
        INSERT INTO inventory_movements (
            id, merchant_id, store_id, product_id, variant_id, 
            movement_type, quantity_change, quantity_before, quantity_after, 
            reference_type, reference_id, notes, created_by, created_at
        )
        VALUES (
            :id, :merchant_id, :store_id, :product_id, :variant_id, 
            :movement_type, :quantity_change, :quantity_before, :quantity_after, 
            :reference_type, :reference_id, :notes, :created_by, :created_at
        )
    `
	_, err = tx.NamedExecContext(ctx, insertLogQuery, movement)
	if err != nil {
		return fmt.Errorf("failed to log movement: %w", err)
	}

	return tx.Commit()
}
