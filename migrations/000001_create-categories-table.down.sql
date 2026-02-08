DROP INDEX IF EXISTS idx_categories_sort_order;
DROP INDEX IF EXISTS idx_categories_active;
DROP INDEX IF EXISTS idx_categories_parent_id;
DROP INDEX IF EXISTS idx_categories_merchant_id;

DROP TABLE IF EXISTS categories CASCADE;
