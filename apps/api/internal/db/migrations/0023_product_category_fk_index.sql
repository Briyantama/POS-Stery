-- +goose Up
CREATE INDEX idx_products_category_id ON product.products (category_id) WHERE category_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS product.idx_products_category_id;
