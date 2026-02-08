CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL,
    parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    image_url VARCHAR(500),
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(merchant_id, parent_id, name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_categories_merchant_id ON categories(merchant_id);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_active ON categories(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_categories_sort_order ON categories(merchant_id, sort_order);

-- Sample data: Create default "Uncategorized" category
INSERT INTO categories (id, merchant_id, name, description, sort_order, is_active)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '14045670-dd76-416d-b798-436757cef4b6',
    'Uncategorized',
    'Default category for products without a specific category',
    0,
    TRUE
) ON CONFLICT DO NOTHING;
