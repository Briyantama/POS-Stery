-- +goose Up
CREATE INDEX idx_sale_items_sale_id ON sales.sale_items (sale_id);

-- +goose Down
DROP INDEX IF EXISTS sales.idx_sale_items_sale_id;
