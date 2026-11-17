-- +goose Up
-- FK checks bypass RLS, so the single-column FK let loyalty rows reference another tenant's customer.
-- Fails if existing rows already cross tenants; fix that data before migrating.
ALTER TABLE customer.customers ADD CONSTRAINT customers_tenant_id_id_key UNIQUE (tenant_id, id);
ALTER TABLE customer.loyalty_points DROP CONSTRAINT loyalty_points_customer_id_fkey;
ALTER TABLE customer.loyalty_points ADD CONSTRAINT loyalty_points_tenant_customer_fkey
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customer.customers (tenant_id, id);

-- +goose Down
ALTER TABLE customer.loyalty_points DROP CONSTRAINT loyalty_points_tenant_customer_fkey;
ALTER TABLE customer.loyalty_points ADD CONSTRAINT loyalty_points_customer_id_fkey
    FOREIGN KEY (customer_id) REFERENCES customer.customers (id);
ALTER TABLE customer.customers DROP CONSTRAINT customers_tenant_id_id_key;
