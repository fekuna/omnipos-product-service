DROP INDEX IF EXISTS idx_inventory_low_stock;
DROP INDEX IF EXISTS idx_inventory_variant_id;
DROP INDEX IF EXISTS idx_inventory_product_id;
DROP INDEX IF EXISTS idx_inventory_store_id;
DROP INDEX IF EXISTS idx_inventory_merchant_id;

DROP TABLE IF EXISTS inventory CASCADE;
