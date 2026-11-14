# CLAUDE.md

Guidance for Claude Code in this repository.

## Project

Multi-tenant POS SaaS. `PRD.md` is the product spec; `docs/FSD_POS.en.md` is the technical reference. Work must trace to a PRD section. Architecture decisions are in `docs/adr/`.

## Layout

```
apps/api/                     one Go module github.com/Briyantama/POS-Stery (Go 1.27.3)
  cmd/api/                    HTTP server: main.go (run(ctx, getenv)), routes.go (newMux)
  cmd/migrate/                goose up|down, reads DATABASE_URL
  internal/config/            Load(getenv) with fail-fast validation
  internal/db/migrations/     goose SQL files NNNN_name.sql (0001 bootstrap, auth 0011-0013, product 0021,
                              inventory 0031, sales 0041, supplier 0051, customer 0061)
  services/<name>/            auth product inventory sales supplier customer
    domain/ application/{commands,queries}/ infrastructure/   business logic, ports, adapters
    http.go                   HTTPHandler.Register(mux): routes "VERB /api/v1/..."
apps/admin/                   SvelteKit admin dashboard (adapter-node, :5173)
apps/cashier/                 SvelteKit cashier POS + PWA service worker (adapter-node, :5174)
packages/ui/                  shared Svelte components
deploy/docker-compose.yml     Dragonfly (Postgres runs outside compose)
scripts/check.sh              quality gate, stages go and web
```

Shared Go and web code lives in the sibling repo `../stery-kit` (D5, ADR-0001): Go module `github.com/Briyantama/stery-kit/go` (httpx, health, objstore, migrate, testdb, pgtenant, events, authn, ratelimit) and npm package `@stery/kit` (`stery-kit/web`). `apps/api/go.mod` points at it with `replace ... => ../../../stery-kit/go`. Build and test need that checkout; `stery-kit` is read-only from this repo.

## Commands

```bash
make up        # compose up (Dragonfly), migrate up, start the API in the background (.run/api.pid)
make down      # stop the API and compose
make status    # API pid + curl /healthz
make build     # go build -o .run/api ./cmd/api
make check     # bash scripts/check.sh (needs TEST_DATABASE_URL)
```

In `apps/api`:

```bash
go run ./cmd/migrate up          # or down
go run ./cmd/api                 # :8000
go test ./services/inventory/... # one package
gofmt -l . && go vet ./...
```

Frontends (`apps/admin`, `apps/cashier`): `npm install`, `npm run dev`, `npm run check`, `npm run test:unit`. Node >= 20.19.

The full `scripts/check.sh` runs in GitHub CI. Locally run targeted Go tests, `npm run check` and Vitest. `go test -race` needs cgo and runs in CI only.

### PM2 (local dev)

`ecosystem.config.cjs` defines three processes and reads secrets from `.env`:

| Port | Name | Type |
|------|------|------|
| 8000 | pos-api-8000 | Go API (`go run ./cmd/api` in `apps/api`) |
| 5173 | pos-admin-5173 | SvelteKit admin |
| 5174 | pos-cashier-5174 | SvelteKit cashier |

`pm2 start ecosystem.config.cjs`, `pm2 restart all`, `pm2 logs pos-api-8000`, `pm2 status`. Port 8080 on this machine belongs to an unrelated Apache.

## Conventions

- Handlers respond only through `httpx.WriteSuccess` / `httpx.WriteError`; envelope `{success,data,error}`. Error text is a short fixed sentence; unexpected errors return 500 `internal error`, never `err.Error()`.
- Tenant and actor ids come from `authn.FromContext`, never from the request body.
- Every route is public on purpose (`POST /api/v1/login`, `GET /healthz`) or wrapped by `authn.Authenticate`; roles via `authn.WithRole`. Authenticated GETs set `Cache-Control: no-store`.
- Status codes: 200 read/update/action, 201 create with `Location`, 400 bad JSON, 422 validation, 409 conflict, 404 missing or cross-tenant, 401, 403, 429 with `Retry-After`.
- JSON fields are `snake_case`. Paginated lists use `?page=&per_page=` (default 20, max 100) and return `{items,page,per_page,total}`.
- Logging: `log/slog` with `InfoContext`/`ErrorContext`; add `tenant_id` where known. Never log tokens, passwords or PII.
- Errors wrapped with `%w`. Tests in external `_test` packages, table-driven or Arrange/Act/Assert; DB tests use the template-clone test database and skip when `TEST_DATABASE_URL` is unset.
- Business rules live in `domain/` and `application/commands/`; `infrastructure/` and `http.go` hold no business logic.
- Migrations: one goose file per change with `-- +goose Up` and `-- +goose Down`, numbered inside the module's range. Never edit an applied migration.
- Events: publish to Dragonfly stream `pos:events` after the DB commit (ADR-0002). Types: `Product.LowStock`, `Sale.Completed`, `Inventory.Replenished`, `Customer.New`. No consumers yet.
- Frontends reach the API only through their SvelteKit server proxy (`/api/[...path]` to `${GATEWAY_URL}/api/v1/...`, default `http://localhost:8000`); tokens stay in server-side httpOnly cookies.
- Secrets go in `.env` as `${VAR}` references, never in tracked files.

## Invariants

- The tenant transaction helper (`WithTenantTx`, `set_config('app.tenant_id'|'app.store_id', ..., true)`) is never skipped in a tenant repository. Skipping it makes RLS return zero rows silently.
- `auth.refresh_tokens` has NO RLS (lookup by hash happens before the tenant is known): never wrap it in the tenant helper. Always call `Rotate(ctx, oldID, newRT)`, never a separate Create + Revoke.
- Refresh tokens: opaque 32 random bytes, SHA-256 hash stored, `family_id` reuse detection revokes the whole family. Access TTL 10 min, refresh TTL 30 days, jti blacklist in Dragonfly (`pos:token:blacklist:<sha256(jti)>`).
- A login email present in two tenants fails closed (`auth.resolve_login_tenant` returns NULL).
- Stock never goes negative; `UpdateStockHandler` checks before writing.
- `CreateSale` order: deduct stock, write sale + items + receipt in one transaction, on failure best-effort compensating `+qty`. Every completed sale has a receipt JSON snapshot.
- Loyalty points change only after the sale is fully committed.
- `Product.LowStock` fires at or below the per-product-per-store threshold.
- Reports are scoped by `tenant_id` and `store_id`.
- No public tenant signup in Phase 1 (tenants are console-seeded).
- No cross-schema foreign keys; other domains are referenced by UUID only.
- DB roles: `pos_admin` (BYPASSRLS, migrations only), `pos_app` (RLS-bound, all application code).
- Rate limits: 300 requests/min per client IP globally, 20/min on `POST /api/v1/login`.
