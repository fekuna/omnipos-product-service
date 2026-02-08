package dto

type CreateProductInput struct {
	MerchantID     string
	CategoryID     string // Optional?
	SKU            string
	Barcode        string
	Name           string
	Description    string
	BasePrice      float64
	CostPrice      float64
	TaxRate        float64
	HasVariants    bool
	TrackInventory bool
	ImageURL       string
}

type UpdateProductInput struct {
	ID             string
	MerchantID     string // For authz check usually
	CategoryID     string
	SKU            string
	Barcode        string
	Name           string
	Description    string
	BasePrice      float64
	CostPrice      float64
	TaxRate        float64
	TrackInventory bool
	ImageURL       string
	IsActive       bool
}

type CreateVariantInput struct {
	ProductID       string
	SKU             string
	Barcode         string
	VariantName     string
	PriceAdjustment float64
	CostPrice       float64
}

type UpdateVariantInput struct {
	ID              string
	ProductID       string
	SKU             string
	Barcode         string
	VariantName     string
	PriceAdjustment float64
	CostPrice       float64
	IsActive        bool
}
