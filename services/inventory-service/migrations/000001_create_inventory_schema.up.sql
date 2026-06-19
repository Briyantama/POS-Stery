CREATE SCHEMA IF NOT EXISTS inventory;
GRANT USAGE ON SCHEMA inventory TO pos_app;

CREATE TABLE inventory.stock_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    store_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, store_id, product_id)
);

CREATE INDEX idx_stock_items_tenant_store ON inventory.stock_items(tenant_id, store_id);

ALTER TABLE inventory.stock_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.stock_items FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON inventory.stock_items
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE TABLE inventory.stock_thresholds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    store_id UUID NOT NULL,
    product_id UUID NOT NULL,
    min_quantity BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, store_id, product_id)
);

ALTER TABLE inventory.stock_thresholds ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.stock_thresholds FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON inventory.stock_thresholds
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA inventory TO pos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA inventory GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pos_app;
