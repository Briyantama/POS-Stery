CREATE SCHEMA IF NOT EXISTS product;
GRANT USAGE ON SCHEMA product TO pos_app;

CREATE TABLE product.categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE product.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    category_id UUID REFERENCES product.categories(id),
    name TEXT NOT NULL,
    sku TEXT NOT NULL,
    barcode TEXT,
    base_price NUMERIC(12,2) NOT NULL,
    sale_price NUMERIC(12,2),
    description TEXT,
    unit TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, sku)
);

CREATE INDEX idx_products_tenant_id ON product.products(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_barcode ON product.products(tenant_id, barcode) WHERE barcode IS NOT NULL;

-- Enable RLS on both tables
ALTER TABLE product.categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE product.categories FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON product.categories
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

ALTER TABLE product.products ENABLE ROW LEVEL SECURITY;
ALTER TABLE product.products FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON product.products
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA product TO pos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA product GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pos_app;
