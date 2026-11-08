-- +goose Up
CREATE INDEX idx_purchase_order_items_order ON supplier.purchase_order_items (tenant_id, purchase_order_id);
CREATE INDEX idx_purchase_orders_supplier_id ON supplier.purchase_orders (supplier_id);

-- +goose Down
DROP INDEX IF EXISTS supplier.idx_purchase_orders_supplier_id;
DROP INDEX IF EXISTS supplier.idx_purchase_order_items_order;
