# 01 - Context

## Starting point

POS-Stery before 2026-11-08: Go chi gateway calling six Go gRPC services, go.work with per-service modules, NATS JetStream, golang-migrate per service, zap, OpenTelemetry, Prometheus. Frontends: `apps/admin`, `apps/cashier`, `packages/ui`, behind chi gateway `services/api-gateway`.

Reference stack: `SELA/apps/api` (one module, `cmd/api`, `cmd/migrate`, `internal/{config,httpx,health,db,testdb,objstore}`, `services/<name>`).

## Invariants that must not weaken

- `db.WithTenantTx` is never skipped in a tenant repository; `auth.refresh_tokens` has no RLS and is never wrapped
- refresh tokens: opaque, SHA-256 hash stored, `family_id` reuse detection, access TTL 10 min, refresh TTL 30 days, Redis jti blacklist; always `Rotate`
- stock never negative; `CreateSale` deducts stock, writes sale + items + receipt in one transaction, compensates on failure
- loyalty points only after a committed sale; `Product.LowStock` at or below threshold
- no public tenant signup; no cross-schema FKs; roles `pos_admin` (migrations) and `pos_app` (RLS-bound)

## Behavior to preserve

Every invariant above, every existing domain and application unit test, the event subject strings and envelope fields, the JWT claims, and every route path the frontends call (with `/v1` inserted where the route table in `05-units-of-work.md` says so).
