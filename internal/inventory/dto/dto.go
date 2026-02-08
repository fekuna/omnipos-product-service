package dto

import "time"

type InventoryFilters struct {
	MerchantID string
	StoreID    *string // Nil for global/warehouse if we distinguish, or just filter
	ProductID  string
	LowStock   bool // If true, filter by available_quantity <= reorder_point
	Page       int
	PageSize   int
}

type MovementFilters struct {
	MerchantID   string
	ProductID    string
	StoreID      *string
	MovementType string
	StartDate    *time.Time
	EndDate      *time.Time
	Page         int
	PageSize     int
}
