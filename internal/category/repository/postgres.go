package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fekuna/omnipos-product-service/internal/category/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
	"github.com/jmoiron/sqlx"
)

type PGRepository struct {
	DB *sqlx.DB
}

func NewPGRepository(db *sqlx.DB) *PGRepository {
	return &PGRepository{DB: db}
}

func (r *PGRepository) Create(ctx context.Context, c *model.Category) error {
	query := `
        INSERT INTO categories (id, merchant_id, parent_id, name, description, image_url, sort_order, is_active, created_at, updated_at)
        VALUES (:id, :merchant_id, :parent_id, :name, :description, :image_url, :sort_order, :is_active, :created_at, :updated_at)
    `
	_, err := r.DB.NamedExecContext(ctx, query, c)
	return err
}

func (r *PGRepository) FindByID(ctx context.Context, id string) (*model.Category, error) {
	var category model.Category
	query := `SELECT * FROM categories WHERE id = $1 LIMIT 1`
	err := r.DB.GetContext(ctx, &category, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *PGRepository) FindAll(ctx context.Context, f *dto.CategoryFilters) ([]model.Category, int, error) {
	var categories []model.Category
	var count int

	conditions := []string{}
	args := map[string]interface{}{}

	if f.MerchantID != "" {
		conditions = append(conditions, "merchant_id = :merchant_id")
		args["merchant_id"] = f.MerchantID
	}
	// ParentID filtering logic
	if f.ParentID != nil {
		if *f.ParentID == "" {
			// Find root categories (parent_id IS NULL)
			conditions = append(conditions, "parent_id IS NULL")
		} else {
			conditions = append(conditions, "parent_id = :parent_id")
			args["parent_id"] = *f.ParentID
		}
	}
	if f.IsActive != nil {
		conditions = append(conditions, "is_active = :is_active")
		args["is_active"] = *f.IsActive
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := "SELECT count(*) FROM categories" + whereClause

	// Check if we assume 'args' can be used with NamedQuery for count.
	// sqlx.Named doesn't directly support returning scalar easily without struct scan.
	// Simpler approach: Use NamedQuery and scan into struct or use rows.

	rows, err := r.DB.NamedQueryContext(ctx, countQuery, args)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&count)
	}

	// List query
	query := "SELECT * FROM categories" + whereClause + " ORDER BY sort_order ASC, name ASC"

	// Pagination
	if f.PageSize > 0 {
		offset := (f.Page - 1) * f.PageSize
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.PageSize, offset)
	}

	nstmt, err := r.DB.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer nstmt.Close()

	err = nstmt.SelectContext(ctx, &categories, args)
	if err != nil {
		return nil, 0, err
	}

	return categories, count, nil
}

func (r *PGRepository) Update(ctx context.Context, c *model.Category) error {
	query := `
        UPDATE categories 
        SET parent_id = :parent_id, 
            name = :name, 
            description = :description, 
            image_url = :image_url, 
            sort_order = :sort_order, 
            is_active = :is_active, 
            updated_at = :updated_at
        WHERE id = :id AND merchant_id = :merchant_id
    `
	_, err := r.DB.NamedExecContext(ctx, query, c)
	return err
}

func (r *PGRepository) Delete(ctx context.Context, id string) error {
	// Check if it has children? Database constraint (fk) is SET NULL, so children become root.
	// Or we could enforce check here. Simple delete for now.
	_, err := r.DB.ExecContext(ctx, "DELETE FROM categories WHERE id = $1", id)
	return err
}
