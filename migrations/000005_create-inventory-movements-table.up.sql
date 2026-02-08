CREATE TABLE IF NOT EXISTS inventory_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL,
    store_id UUID,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id) ON DELETE CASCADE,
    movement_type VARCHAR(50) NOT NULL, -- 'purchase', 'sale', 'adjustment', 'transfer_in', 'transfer_out', 'return'
    quantity_change DECIMAL(15,3) NOT NULL, -- positive for increase, negative for decrease
    quantity_before DECIMAL(15,3) NOT NULL,
    quantity_after DECIMAL(15,3) NOT NULL,
    reference_type VARCHAR(50), -- 'order', 'adjustment', 'transfer', 'purchase_order'
    reference_id UUID, -- ID of the related entity
    notes TEXT,
    created_by UUID, -- user_id who performed the action
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_movement_type CHECK (movement_type IN ('purchase', 'sale', 'adjustment', 'transfer_in', 'transfer_out', 'return'))
);

-- Indexes for performance and audit trail queries
CREATE INDEX IF NOT EXISTS idx_inventory_movements_merchant_id ON inventory_movements(merchant_id);
CREATE INDEX IF NOT EXISTS idx_inventory_movements_product_id ON inventory_movements(product_id);
CREATE INDEX IF NOT EXISTS idx_inventory_movements_variant_id ON inventory_movements(variant_id) WHERE variant_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_movements_created_at ON inventory_movements(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_movements_reference ON inventory_movements(reference_type, reference_id) WHERE reference_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_movements_store_id ON inventory_movements(store_id) WHERE store_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_movements_type ON inventory_movements(merchant_id, movement_type, created_at DESC);
