-- +goose Up
-- A sale awards loyalty points once. Keep the earliest row per (tenant_id, sale_id);
-- balances are SUM(points), so dropping duplicate awards is the intended correction.
DELETE FROM customer.loyalty_points a
USING customer.loyalty_points b
WHERE a.tenant_id = b.tenant_id
  AND a.sale_id = b.sale_id
  AND (a.created_at, a.id) > (b.created_at, b.id);

ALTER TABLE customer.loyalty_points
    ADD CONSTRAINT loyalty_points_tenant_sale_key UNIQUE (tenant_id, sale_id);

-- +goose Down
ALTER TABLE customer.loyalty_points DROP CONSTRAINT IF EXISTS loyalty_points_tenant_sale_key;
