CREATE SCHEMA IF NOT EXISTS supplier;
GRANT USAGE ON SCHEMA supplier TO pos_app;

CREATE TABLE supplier.suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    contact TEXT,
    phone TEXT,
    email TEXT,
    address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_suppliers_tenant_id ON supplier.suppliers(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE supplier.suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier.suppliers FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON supplier.suppliers
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE TABLE supplier.purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    store_id UUID NOT NULL,
    supplier_id UUID NOT NULL REFERENCES supplier.suppliers(id),
    status TEXT NOT NULL DEFAULT 'PENDING',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_purchase_orders_tenant_store ON supplier.purchase_orders(tenant_id, store_id);

ALTER TABLE supplier.purchase_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier.purchase_orders FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON supplier.purchase_orders
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE TABLE supplier.purchase_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    purchase_order_id UUID NOT NULL REFERENCES supplier.purchase_orders(id),
    product_id UUID NOT NULL,
    quantity_ordered INT NOT NULL,
    quantity_received INT NOT NULL DEFAULT 0,
    unit_cost NUMERIC(12,2) NOT NULL
);

ALTER TABLE supplier.purchase_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier.purchase_order_items FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON supplier.purchase_order_items
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA supplier TO pos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA supplier GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pos_app;
