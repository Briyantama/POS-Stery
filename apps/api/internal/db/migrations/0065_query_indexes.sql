-- +goose Up
-- Newest-first list pages (ORDER BY created_at DESC LIMIT n) read the first rows of the index instead of
-- sorting every row of the tenant (or tenant+store) first.
CREATE INDEX IF NOT EXISTS idx_products_tenant_created
    ON product.products (tenant_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_tenant_created
    ON customer.customers (tenant_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_purchase_orders_tenant_store_created
    ON supplier.purchase_orders (tenant_id, store_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS supplier.idx_purchase_orders_tenant_store_created;
DROP INDEX IF EXISTS customer.idx_customers_tenant_created;
DROP INDEX IF EXISTS product.idx_products_tenant_created;
