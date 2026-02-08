DROP INDEX IF EXISTS idx_inventory_movements_type;
DROP INDEX IF EXISTS idx_inventory_movements_store_id;
DROP INDEX IF EXISTS idx_inventory_movements_reference;
DROP INDEX IF EXISTS idx_inventory_movements_created_at;
DROP INDEX IF EXISTS idx_inventory_movements_variant_id;
DROP INDEX IF EXISTS idx_inventory_movements_product_id;
DROP INDEX IF EXISTS idx_inventory_movements_merchant_id;

DROP TABLE IF EXISTS inventory_movements CASCADE;
