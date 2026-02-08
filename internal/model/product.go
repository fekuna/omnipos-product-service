package model

type Product struct {
	BaseModel
	MerchantID     string           `db:"merchant_id" json:"merchant_id"`
	CategoryID     *string          `db:"category_id" json:"category_id"` // Nullable
	SKU            string           `db:"sku" json:"sku"`
	Barcode        *string          `db:"barcode" json:"barcode"` // Nullable
	Name           string           `db:"name" json:"name"`
	Description    *string          `db:"description" json:"description"`
	BasePrice      float64          `db:"base_price" json:"base_price"`
	CostPrice      *float64         `db:"cost_price" json:"cost_price"`
	TaxRate        float64          `db:"tax_rate" json:"tax_rate"`
	HasVariants    bool             `db:"has_variants" json:"has_variants"`
	TrackInventory bool             `db:"track_inventory" json:"track_inventory"`
	ImageURL       *string          `db:"image_url" json:"image_url"`
	IsActive       bool             `db:"is_active" json:"is_active"`
	Variants       []ProductVariant `db:"-" json:"variants"` // Not in DB table directly
	Category       *Category        `db:"-" json:"category"` // Joined data
}

type ProductVariant struct {
	BaseModel
	ProductID       string   `db:"product_id" json:"product_id"`
	SKU             string   `db:"sku" json:"sku"`
	Barcode         *string  `db:"barcode" json:"barcode"`
	VariantName     string   `db:"variant_name" json:"variant_name"`
	PriceAdjustment float64  `db:"price_adjustment" json:"price_adjustment"`
	CostPrice       *float64 `db:"cost_price" json:"cost_price"`
	IsActive        bool     `db:"is_active" json:"is_active"`
}
