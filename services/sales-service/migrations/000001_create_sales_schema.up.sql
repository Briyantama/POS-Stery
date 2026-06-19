CREATE SCHEMA IF NOT EXISTS sales;
GRANT USAGE ON SCHEMA sales TO pos_app;
CREATE TABLE sales.sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    store_id UUID NOT NULL,
    cashier_id UUID NOT NULL,
    customer_id UUID,
    subtotal NUMERIC(12,2) NOT NULL,
    discount_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    total NUMERIC(12,2) NOT NULL,
    status TEXT NOT NULL DEFAULT 'COMPLETED',
    notes TEXT,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sales_tenant_store ON sales.sales(tenant_id, store_id);
CREATE INDEX idx_sales_completed_at ON sales.sales(tenant_id, store_id, completed_at DESC);
ALTER TABLE sales.sales ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales.sales FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sales.sales USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid) WITH CHECK (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);
CREATE TABLE sales.sale_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    sale_id UUID NOT NULL REFERENCES sales.sales(id),
    product_id UUID NOT NULL,
    quantity INT NOT NULL,
    unit_price NUMERIC(12,2) NOT NULL,
    discount NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_total NUMERIC(12,2) NOT NULL
);
ALTER TABLE sales.sale_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales.sale_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sales.sale_items USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);
CREATE TABLE sales.receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    store_id UUID NOT NULL,
    sale_id UUID NOT NULL UNIQUE REFERENCES sales.sales(id),
    snapshot JSONB NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE sales.receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales.receipts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sales.receipts USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA sales TO pos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA sales GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pos_app;
