package model

type Category struct {
	BaseModel
	MerchantID  string     `db:"merchant_id"`
	ParentID    *string    `db:"parent_id"` // Nullable
	Name        string     `db:"name"`
	Description *string    `db:"description"`
	ImageURL    *string    `db:"image_url"`
	SortOrder   int        `db:"sort_order"`
	IsActive    bool       `db:"is_active"`
	Children    []Category `db:"-"` // For tree structure, not in DB
}
