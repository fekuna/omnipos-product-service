package model

import "time"

type Inventory struct {
	ID                string     `db:"id"`
	MerchantID        string     `db:"merchant_id"`
	StoreID           *string    `db:"store_id"`
	ProductID         string     `db:"product_id"`
	VariantID         *string    `db:"variant_id"`
	Quantity          float64    `db:"quantity"`
	ReservedQuantity  float64    `db:"reserved_quantity"`
	AvailableQuantity float64    `db:"available_quantity"` // Generated column
	ReorderPoint      float64    `db:"reorder_point"`
	ReorderQuantity   float64    `db:"reorder_quantity"`
	LastCountedAt     *time.Time `db:"last_counted_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

type InventoryMovement struct {
	ID             string    `db:"id"`
	MerchantID     string    `db:"merchant_id"`
	StoreID        *string   `db:"store_id"`
	ProductID      string    `db:"product_id"`
	VariantID      *string   `db:"variant_id"`
	MovementType   string    `db:"movement_type"`
	QuantityChange float64   `db:"quantity_change"`
	QuantityBefore float64   `db:"quantity_before"`
	QuantityAfter  float64   `db:"quantity_after"`
	ReferenceType  *string   `db:"reference_type"`
	ReferenceID    *string   `db:"reference_id"`
	Notes          string    `db:"notes"`
	CreatedBy      *string   `db:"created_by"`
	CreatedAt      time.Time `db:"created_at"`
}
