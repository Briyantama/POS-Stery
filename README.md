# POS-Stery

Multi-tenant Point of Sale SaaS platform for retail store chains. Supports multi-store tenants, cashier checkout flows, inventory management, supplier orders, customer loyalty, and daily reporting.

---

## Architecture

```
Browser
    │
    ▼
apps/admin, apps/cashier  ← SvelteKit server proxy, JWT in httpOnly cookies
    │  REST /api/v1, envelope {success,data,error}
    ▼
apps/api  ← one Go binary: net/http ServeMux, RS256 JWT, jti blacklist, roles, rate limits
    │  packages auth, product, inventory, sales, supplier, customer (in-process calls)
    ├── PostgreSQL 18, one schema per package, RLS
    └── Dragonfly: jti blacklist, rate limits, event stream pos:events
```

Decisions: `docs/adr/0001-single-module-go-api-on-sela-stack.md`, `docs/adr/0002-dragonfly-streams-replace-nats-jetstream.md`.

### API packages (`apps/api/services/<name>`)

| Package | Responsibility |
|---|---|
| `auth` | Tenants, stores, users, roles, JWT issue/refresh/revoke, store validation |
| `product` | Product catalog, SKU management |
| `inventory` | Stock levels, thresholds, low-stock events |
| `sales` | Sales, receipts, stock deduction through `inventory` |
| `supplier` | Suppliers, purchase orders, replenishment |
| `customer` | Customer records, loyalty points |

Shared Go and web code comes from the sibling repo `stery-kit` (Go module `github.com/Briyantama/stery-kit/go`, npm package `@stery/kit`). Clone it next to this repo: `../stery-kit`.

### Frontend Apps

| App | Port | Description |
|---|---|---|
| `apps/admin` | 5173 | Tenant admin dashboard (SvelteKit) |
| `apps/cashier` | 5174 | Cashier POS with offline PWA support (SvelteKit) |

### Observability

`log/slog` structured logs and `GET /healthz`. No metrics or tracing.

---

## Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Docker + Docker Compose | latest | Runs Dragonfly |
| PostgreSQL | 18 | Runs outside compose |
| Go | 1.25.1+ | API |
| Node.js | 20.19+ | SvelteKit apps (npm) |
| PM2 | latest | Optional, local process manager |

---

## Quick Start

### 1. Clone and configure

```bash
git clone <repo-url> POS-Stery
git clone git@github.com:Briyantama/stery-kit.git stery-kit
cd POS-Stery
cp .env.example .env
```

Edit `.env` and set passwords. Generate RS256 keys for JWT:

```bash
mkdir -p keys
openssl genrsa -out keys/private.pem 2048
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
```

### 2. Run

```bash
make up        # Dragonfly via compose, migrate up, API on :8000 in the background
make status    # API pid + /healthz
make down      # stop API and compose
```

Frontends:

```bash
cd apps/admin && npm install && npm run dev      # :5173
cd apps/cashier && npm install && npm run dev    # :5174
```

---

## Development

### API (`apps/api`)

```bash
go run ./cmd/migrate up          # or down; reads DATABASE_URL
go run ./cmd/api                 # :8000
go test ./...                    # DB tests need TEST_DATABASE_URL, skipped otherwise
gofmt -l . && go vet ./...
```

`make build` writes `.run/api`. `make check` runs `scripts/check.sh` (gofmt, vet, golangci-lint, tests, `npm run check`, Vitest); the full gate runs in GitHub CI.

Migrations are goose files in `apps/api/internal/db/migrations/NNNN_name.sql` with `-- +goose Up` and `-- +goose Down` sections.

### SvelteKit apps

`npm run dev`, `npm run check`, `npm run build`, `npm run test:unit` in `apps/admin` or `apps/cashier`.

### PM2

`ecosystem.config.cjs` defines three processes and reads secrets from `.env`:

| Name | Port | |
|---|---|---|
| `pos-api-8000` | 8000 | Go API (`go run ./cmd/api` in `apps/api`) |
| `pos-admin-5173` | 5173 | Admin dashboard |
| `pos-cashier-5174` | 5174 | Cashier POS |

```bash
pm2 start ecosystem.config.cjs
pm2 restart all
pm2 logs pos-api-8000
pm2 status
```

---

## Tenant Isolation

1. **Middleware**: the JWT is verified locally; tenant and store ids come from its claims, and an admin's `X-Store-ID` is checked against the tenant by the `auth` package.
2. **Handlers**: tenant and actor ids are read from the request context, never from the body.
3. **Database**: PostgreSQL RLS on every tenant-scoped table. Repositories run inside `WithTenantTx`, which sets `app.tenant_id` and `app.store_id` with `set_config(..., true)`.

---

## Database Layout

Single Postgres database, one schema per package:

| Schema | Package |
|---|---|
| `auth` | auth: tenants, stores, users, roles |
| `product` | product |
| `inventory` | inventory |
| `sales` | sales |
| `supplier` | supplier |
| `customer` | customer |

No cross-schema foreign keys. Packages reference other domains by UUID only. Roles: `pos_admin` (BYPASSRLS, migrations), `pos_app` (RLS-bound, application).

---

## Key Business Rules

- **Stock never goes negative**: checked in inventory's `UpdateStockHandler` before any DB write.
- **Every sale produces a receipt**: JSON snapshot stored in `sales.receipts`.
- **Loyalty points update only after a committed sale.**
- **All report queries are scoped by `tenant_id` and `store_id`.**
- **Phase 1 has no self-service signup**: tenants are seeded via console.

---

## Events

Dragonfly stream `pos:events`, published after commit. No consumers yet.

| Type | Publisher |
|---|---|
| `Product.LowStock` | inventory |
| `Sale.Completed` | sales |
| `Inventory.Replenished` | supplier |
| `Customer.New` | customer |

---

## Environment Variables

Copy `.env.example` to `.env`. Key variables:

| Variable | Description |
|---|---|
| `PORT` | API port (default 8000) |
| `DATABASE_URL` | Postgres connection URL for the API and `cmd/migrate` |
| `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_TLS` | Dragonfly connection |
| `JWT_RS256_PRIVATE_KEY_PATH`, `JWT_RS256_PUBLIC_KEY_PATH` | RS256 key files |
| `TRUSTED_PROXY_CIDRS` | Proxies whose forwarded client IP is trusted |
| `GATEWAY_URL` | API base URL for the SvelteKit server proxy (default `http://localhost:8000`) |
| `TEST_DATABASE_URL` | Postgres URL for DB tests |
