package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fekuna/omnipos-product-service/internal/model"
	"github.com/fekuna/omnipos-product-service/internal/product/dto"
	"github.com/jmoiron/sqlx"
)

type PGRepository struct {
	DB *sqlx.DB
}

func NewPGRepository(db *sqlx.DB) *PGRepository {
	return &PGRepository{DB: db}
}

func (r *PGRepository) Create(ctx context.Context, p *model.Product) error {
	query := `
        INSERT INTO products (
            id, merchant_id, category_id, sku, barcode, name, description, 
            base_price, cost_price, tax_rate, has_variants, track_inventory, 
            image_url, is_active, created_at, updated_at
        )
        VALUES (
            :id, :merchant_id, :category_id, :sku, :barcode, :name, :description, 
            :base_price, :cost_price, :tax_rate, :has_variants, :track_inventory, 
            :image_url, :is_active, :created_at, :updated_at
        )
    `
	// Note: Transaction handling for variants should normally be done in UseCase using a transaction manager.
	// Here we just insert the product.
	_, err := r.DB.NamedExecContext(ctx, query, p)
	return err
}

func (r *PGRepository) FindByID(ctx context.Context, id string) (*model.Product, error) {
	var product model.Product
	query := `SELECT * FROM products WHERE id = $1 LIMIT 1`
	err := r.DB.GetContext(ctx, &product, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *PGRepository) FindAll(ctx context.Context, f *dto.ProductFilters) ([]model.Product, int, error) {
	var products []model.Product
	var count int

	conditions := []string{}
	args := map[string]interface{}{}

	if f.MerchantID != "" {
		conditions = append(conditions, "merchant_id = :merchant_id")
		args["merchant_id"] = f.MerchantID
	}
	if f.CategoryID != "" {
		conditions = append(conditions, "category_id = :category_id")
		args["category_id"] = f.CategoryID
	}
	if f.IsActive != nil {
		conditions = append(conditions, "is_active = :is_active")
		args["is_active"] = *f.IsActive
	}
	if f.SearchQuery != "" {
		// Search by name (full text or ilike), sku, or barcode
		// Using simple ILIKE for broad compatibility, or TSVECTOR if configured
		conditions = append(conditions, "(name ILIKE :search OR sku ILIKE :search OR barcode ILIKE :search)")
		args["search"] = "%" + f.SearchQuery + "%"
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count
	countQuery := "SELECT count(*) FROM products" + whereClause
	rows, err := r.DB.NamedQueryContext(ctx, countQuery, args)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&count)
	}

	// List
	orderBy := "created_at DESC"
	if f.SortBy != "" {
		// Prevent SQL injection by whitelisting fields
		switch f.SortBy {
		case "name":
			orderBy = "name"
		case "price":
			orderBy = "base_price"
		case "created_at":
			orderBy = "created_at"
		}
		if strings.ToLower(f.SortOrder) == "asc" {
			orderBy += " ASC"
		} else {
			orderBy += " DESC"
		}
	}

	query := fmt.Sprintf("SELECT * FROM products%s ORDER BY %s", whereClause, orderBy)

	if f.PageSize > 0 {
		offset := (f.Page - 1) * f.PageSize
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.PageSize, offset)
	}

	nstmt, err := r.DB.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer nstmt.Close()

	err = nstmt.SelectContext(ctx, &products, args)
	if err != nil {
		return nil, 0, err
	}

	return products, count, nil
}

func (r *PGRepository) Update(ctx context.Context, p *model.Product) error {
	query := `
        UPDATE products 
        SET category_id = :category_id, 
            sku = :sku, 
            barcode = :barcode, 
            name = :name, 
            description = :description, 
            base_price = :base_price, 
            cost_price = :cost_price, 
            tax_rate = :tax_rate, 
            has_variants = :has_variants, 
            track_inventory = :track_inventory, 
            image_url = :image_url, 
            is_active = :is_active, 
            updated_at = :updated_at
        WHERE id = :id AND merchant_id = :merchant_id
    `
	_, err := r.DB.NamedExecContext(ctx, query, p)
	return err
}

func (r *PGRepository) Delete(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, "DELETE FROM products WHERE id = $1", id)
	return err
}

func (r *PGRepository) IsSKUUnique(ctx context.Context, merchantID, sku, excludeID string) (bool, error) {
	var count int
	query := `SELECT count(*) FROM products WHERE merchant_id = $1 AND sku = $2`
	args := []interface{}{merchantID, sku}
	if excludeID != "" {
		query += ` AND id != $3`
		args = append(args, excludeID)
	}

	err := r.DB.GetContext(ctx, &count, query, args...)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (r *PGRepository) IsBarcodeUnique(ctx context.Context, merchantID, barcode, excludeID string) (bool, error) {
	if barcode == "" {
		return true, nil
	}
	var count int
	query := `SELECT count(*) FROM products WHERE merchant_id = $1 AND barcode = $2`
	args := []interface{}{merchantID, barcode}
	if excludeID != "" {
		query += ` AND id != $3`
		args = append(args, excludeID)
	}

	err := r.DB.GetContext(ctx, &count, query, args...)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (r *PGRepository) ReserveStock(ctx context.Context, items map[string]int32) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare statement for efficiency? Not explicitly needed for simple query but good practice.
	// We handle variants later. For now, assuming products have inventory.
	// Note: If HasVariants=true, we should probably update Variant table.
	// But let's assume simple Product inventory for this iteration or check logic.
	// The implementation plan says "UPDATE products SET inventory_count = inventory_count - $1...".
	// Let's stick to Product for now. Variants logic can be added if item map has VariantID.

	query := `
		UPDATE inventory 
		SET quantity = quantity - $1, updated_at = NOW() 
		WHERE product_id = $2 AND quantity >= $1
	`

	for productID, qty := range items {
		// Validate inputs
		if qty <= 0 {
			continue
		}

		res, err := tx.ExecContext(ctx, query, qty, productID)
		if err != nil {
			return err
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}

		if rows == 0 {
			// Failed to update (Out of Stock or ID not found)
			// Return specific error?
			return fmt.Errorf("insufficient stock for product %s", productID)
		}
	}

	return tx.Commit()
}
