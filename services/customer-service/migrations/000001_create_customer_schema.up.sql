CREATE SCHEMA IF NOT EXISTS customer;
GRANT USAGE ON SCHEMA customer TO pos_app;

CREATE TABLE customer.customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    phone TEXT,
    email TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_customers_tenant_id ON customer.customers(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE customer.customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer.customers FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON customer.customers
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE TABLE customer.loyalty_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL REFERENCES customer.customers(id),
    points INT NOT NULL,
    total_points INT NOT NULL,
    sale_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loyalty_customer_id ON customer.loyalty_points(customer_id);

ALTER TABLE customer.loyalty_points ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer.loyalty_points FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON customer.loyalty_points
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA customer TO pos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA customer GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pos_app;
