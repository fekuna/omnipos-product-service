package dto

type ProductFilters struct {
	MerchantID  string
	CategoryID  string
	IsActive    *bool
	SearchQuery string // For name, sku, barcode search
	SortBy      string // name, price, created_at
	SortOrder   string // asc, desc
	Page        int
	PageSize    int
}
