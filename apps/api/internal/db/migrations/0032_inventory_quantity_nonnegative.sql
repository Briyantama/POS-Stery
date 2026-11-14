-- +goose Up
-- Rows driven negative by the old check-then-update race are clamped to 0 so the
-- constraint can be added; those shortfalls need a manual stock count to reconcile.
UPDATE inventory.stock_items SET quantity = 0, updated_at = NOW() WHERE quantity < 0;

ALTER TABLE inventory.stock_items
    ADD CONSTRAINT stock_items_quantity_nonnegative CHECK (quantity >= 0);

-- +goose Down
ALTER TABLE inventory.stock_items DROP CONSTRAINT IF EXISTS stock_items_quantity_nonnegative;
