-- +goose Up
-- Audit logs are append-only: the app role may insert and read but never rewrite history.
REVOKE UPDATE, DELETE ON auth.audit_logs FROM pos_app;

-- +goose Down
GRANT UPDATE, DELETE ON auth.audit_logs TO pos_app;
