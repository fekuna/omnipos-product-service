package dto

type CreateCategoryInput struct {
	MerchantID  string
	ParentID    *string
	Name        string
	Description string
	ImageURL    string
	SortOrder   int
}

type UpdateCategoryInput struct {
	ID          string
	MerchantID  string
	ParentID    *string // If nil, no update? Or explicitly set to nil? Logic needs care. Ptr to string ok.
	Name        string
	Description string
	ImageURL    string
	SortOrder   int
	IsActive    bool
}
