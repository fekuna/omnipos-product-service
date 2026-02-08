package dto

type CategoryFilters struct {
	MerchantID      string
	ParentID        *string // Nil means ignore, Empty string means root categories
	IsActive        *bool
	IncludeChildren bool
	Page            int
	PageSize        int
}
