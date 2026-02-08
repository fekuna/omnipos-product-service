DROP INDEX IF EXISTS idx_products_name_search;
DROP INDEX IF EXISTS idx_products_active;
DROP INDEX IF EXISTS idx_products_barcode;
DROP INDEX IF EXISTS idx_products_sku;
DROP INDEX IF EXISTS idx_products_category_id;
DROP INDEX IF EXISTS idx_products_merchant_id;

DROP TABLE IF EXISTS products CASCADE;
