-- +goose Up
-- FK checks bypass RLS, so a single-column FK let a product reference another tenant's category.
-- Fails if existing rows already cross tenants; fix that data before migrating.
ALTER TABLE product.categories ADD CONSTRAINT categories_tenant_id_id_key UNIQUE (tenant_id, id);
ALTER TABLE product.products DROP CONSTRAINT products_category_id_fkey;
ALTER TABLE product.products ADD CONSTRAINT products_tenant_category_fkey
    FOREIGN KEY (tenant_id, category_id) REFERENCES product.categories (tenant_id, id);

-- +goose Down
ALTER TABLE product.products DROP CONSTRAINT products_tenant_category_fkey;
ALTER TABLE product.products ADD CONSTRAINT products_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES product.categories (id);
ALTER TABLE product.categories DROP CONSTRAINT categories_tenant_id_id_key;
