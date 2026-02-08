DROP INDEX IF EXISTS idx_product_variants_active;
DROP INDEX IF EXISTS idx_product_variants_barcode;
DROP INDEX IF EXISTS idx_product_variants_sku;
DROP INDEX IF EXISTS idx_product_variants_product_id;

DROP TABLE IF EXISTS product_variants CASCADE;
