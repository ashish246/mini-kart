-- Drop indexes
DROP INDEX IF EXISTS idx_orders_coupon_code;
DROP INDEX IF EXISTS idx_orders_created_at;

-- Drop orders table
DROP TABLE IF EXISTS orders;
