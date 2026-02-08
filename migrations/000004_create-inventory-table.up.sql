CREATE TABLE IF NOT EXISTS inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL,
    store_id UUID, -- NULL = central/warehouse inventory
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity DECIMAL(15,3) NOT NULL DEFAULT 0,
    reserved_quantity DECIMAL(15,3) DEFAULT 0, -- quantity reserved for pending orders
    available_quantity DECIMAL(15,3) GENERATED ALWAYS AS (quantity - reserved_quantity) STORED,
    reorder_point DECIMAL(15,3) DEFAULT 0, -- trigger reorder when available_quantity <= this
    reorder_quantity DECIMAL(15,3) DEFAULT 0, -- suggested quantity to reorder
    last_counted_at TIMESTAMPTZ, -- last physical count date
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_inventory_location UNIQUE(merchant_id, store_id, product_id, variant_id),
    CONSTRAINT positive_quantity CHECK (quantity >= 0),
    CONSTRAINT positive_reserved CHECK (reserved_quantity >= 0),
    CONSTRAINT reserved_not_exceed_quantity CHECK (reserved_quantity <= quantity)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_inventory_merchant_id ON inventory(merchant_id);
CREATE INDEX IF NOT EXISTS idx_inventory_store_id ON inventory(store_id);
CREATE INDEX IF NOT EXISTS idx_inventory_product_id ON inventory(product_id);
CREATE INDEX IF NOT EXISTS idx_inventory_variant_id ON inventory(variant_id) WHERE variant_id IS NOT NULL;
-- Low stock alert index
CREATE INDEX IF NOT EXISTS idx_inventory_low_stock ON inventory(merchant_id, available_quantity) 
    WHERE available_quantity <= reorder_point AND reorder_point > 0;
