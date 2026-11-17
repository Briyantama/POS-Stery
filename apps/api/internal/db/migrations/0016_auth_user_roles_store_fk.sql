-- +goose Up
-- user_roles.store_id had no FK; a role could point at a missing store or another tenant's store.
-- FK checks bypass RLS, so the key includes tenant_id. A NULL store_id (all stores) skips the check.
-- Fails if existing rows already reference a foreign or missing store; fix that data before migrating.
ALTER TABLE auth.stores ADD CONSTRAINT stores_tenant_id_id_key UNIQUE (tenant_id, id);
ALTER TABLE auth.user_roles ADD CONSTRAINT user_roles_tenant_store_fkey
    FOREIGN KEY (tenant_id, store_id) REFERENCES auth.stores (tenant_id, id);

-- +goose Down
ALTER TABLE auth.user_roles DROP CONSTRAINT user_roles_tenant_store_fkey;
ALTER TABLE auth.stores DROP CONSTRAINT stores_tenant_id_id_key;
