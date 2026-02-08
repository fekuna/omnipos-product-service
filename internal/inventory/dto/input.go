package dto

type AdjustInventoryInput struct {
	MerchantID     string
	StoreID        *string
	ProductID      string
	VariantID      *string
	QuantityChange float64
	Reason         string
	ReferenceID    string
	ReferenceType  string // 'manual_adjustment', 'sale', 'return'
	UserID         string
}

type TransferInventoryInput struct {
	MerchantID    string
	SourceStoreID *string
	TargetStoreID *string
	ProductID     string
	VariantID     *string
	Quantity      float64
	Reason        string
	UserID        string
}
