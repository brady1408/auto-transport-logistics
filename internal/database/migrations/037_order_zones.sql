-- +goose Up

-- Rename zone to origin_zone on orders
ALTER TABLE orders RENAME COLUMN zone TO origin_zone;

-- Add destination zone
ALTER TABLE orders ADD COLUMN destination_zone VARCHAR(20);

-- Update indexes that reference the old column name
DROP INDEX IF EXISTS idx_orders_zone;
CREATE INDEX idx_orders_origin_zone ON orders (origin_zone);
CREATE INDEX idx_orders_destination_zone ON orders (destination_zone);

-- +goose Down

DROP INDEX IF EXISTS idx_orders_destination_zone;
DROP INDEX IF EXISTS idx_orders_origin_zone;
ALTER TABLE orders DROP COLUMN IF EXISTS destination_zone;
ALTER TABLE orders RENAME COLUMN origin_zone TO zone;
CREATE INDEX idx_orders_zone ON orders (zone, order_number);
