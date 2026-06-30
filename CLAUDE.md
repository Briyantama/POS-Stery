# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Multi-tenant POS SaaS platform. See `PRD.md` for the full spec. All work must be traceable to a PRD section — do not invent features outside it.

---

## Commands

All commands are run from the monorepo root unless noted.

### Infrastructure

```bash
make dev           # Start Postgres 16, Redis 7, NATS 2.10, Jaeger, Prometheus via Docker
make dev-down      # Stop infra containers
make dev-reset     # Destroy volumes + restart (DESTRUCTIVE)
```

### Protobuf / gRPC Contracts

```bash
make proto-gen     # buf generate → populates gen/go/ and gen/openapi/
make proto-lint    # buf lint
make proto-breaking # buf breaking against main branch
```

Regenerate after any `.proto` change. Never hand-edit files under `gen/`.

### Database Migrations

```bash
make migrate-up                           # Apply all pending migrations (all services)
make migrate-down                         # Roll back one step per service
make migrate-create NAME=xxx SERVICE=inventory  # Create migration pair for a service
make migrate-fresh                        # DESTRUCTIVE: drop all schemas + reapply
```

Migrations live in `services/<name>-service/migrations/`. Tool: `golang-migrate` v4.

### Go Services

```bash
make build-go          # Build all services (auto-discovers cmd/server/main.go)
make test-go           # go test -short ./... across all workspace members
make test-integration  # INTEGRATION=true go test ./...
make lint-go           # golangci-lint (if installed)
make tidy              # go mod tidy across all go.work members
make sqlc-gen          # sqlc generate for all services with sqlc.yaml
```

Run a single service test:
```bash
cd services/inventory-service && go test ./internal/application/commands/...
```

### Go API Gateway

```bash
make dev-gateway       # go run ./cmd/server/... (services/api-gateway, default :8000)
cd services/api-gateway && go test ./...
```

### SvelteKit Apps

```bash
make install-admin     # pnpm install in apps/admin
make dev-admin         # pnpm dev (admin dashboard, default :5173)
make install-cashier   # pnpm install in apps/cashier
make dev-cashier       # pnpm dev (cashier POS, default :5174)
```

### All-in-one bootstrap

```bash
make all    # proto-gen + migrate-up + build-go + install-admin + install-cashier
```

---

## Architecture

### Stack overview

```
Browser (SvelteKit)
    │  REST/JSON
    ▼
services/api-gateway  ← Go (chi router, RS256 JWT auth, Redis JTI blacklist, RBAC)
    │  gRPC (plaintext, internal network only)
    ▼
Go microservices  ← each owns a Postgres schema + publishes to NATS
    │  NATS JetStream (POS_EVENTS stream)
    ▼
Event consumers   ← services subscribe to Sale.Completed, Product.LowStock, etc.
```

The Go API gateway dials each backend service directly over gRPC using generated client stubs. All gRPC traffic is internal to the `pos-net` Docker bridge / Kubernetes overlay network.

### Go workspace

`go.work` at the repo root ties together `gen/go`, `services/_shared`, and all `services/*-service` modules. Each service has its own `go.mod` for independent builds. Use `make tidy` to sync all members.

### Tenant isolation (critical)

Every request carries `tenant_id`. Store-scoped requests also carry `store_id`. Isolation is enforced at four layers:

1. **Gateway** — `TenantResolver` and `StoreResolver` middleware validate JWT claims; `StoreResolver` confirms the `store_id` belongs to the authenticated tenant.
2. **gRPC contracts** — every request message includes `tenant_id` and `store_id` fields.
3. **Service layer** — command/query handlers pass both IDs through to the repository.
4. **Database** — PostgreSQL RLS policies on every tenant-scoped table: `USING (tenant_id = current_setting('app.tenant_id')::uuid)`. The `_shared/database` package injects `SET LOCAL app.tenant_id = '...'` and `SET LOCAL app.store_id = '...'` before every query via `WithTenantContext(ctx, tenantID, storeID)`.

Never skip the `WithTenantContext` call in a repository. Omitting it causes RLS to use an empty `app.tenant_id`, which silently returns zero rows or blocks writes.

### Database layout

Single Postgres instance, one schema per service:

| Schema | Owner service |
|---|---|
| `auth` | auth-service (tenants, stores, users, roles) |
| `product` | product-service |
| `inventory` | inventory-service |
| `sales` | sales-service |
| `supplier` | supplier-service |
| `customer` | customer-service |

Two roles: `pos_admin` (BYPASSRLS — migrations only) and `pos_app` (subject to RLS — all application code). No cross-schema foreign keys. Services reference other domains by UUID only.

### JWT claims

- Cashier tokens include `tenant_id` + `store_id` (embedded at login).
- Admin tokens include `tenant_id` only; admins pass `X-Store-ID` header per request.

`StoreResolver` validates the header value belongs to the tenant on every admin request. `CheckRole` enforces `admin` vs `cashier` separation.

### Service-to-service auth

Go services validate an `X-Service-Token` HMAC-SHA256 header on inbound calls from other services (sales→inventory, supplier→inventory). The shared secret is in `SERVICE_TOKEN` env var. The `_shared/middleware` interceptor handles this transparently.

### Inventory atomicity (sale flow)

`SalesService.CreateSale` uses a synchronous saga:
1. Call `InventoryService.UpdateStock` (deduct) via gRPC.
2. If successful, write sale + receipt records to DB.
3. If the DB write fails, call `InventoryService.UpdateStock` (compensate +quantity) as a best-effort rollback.

Stock can never go negative — enforced in `inventory-service`'s `UpdateStockHandler` before any DB write. A `Product.LowStock` event is published when stock falls to or below the per-product-per-store threshold (`inventory.stock_thresholds`).

### NATS events

Stream `POS_EVENTS`, file storage, 7-day retention. Subjects:

- `Product.LowStock` — published by inventory-service
- `Sale.Completed` — published by sales-service
- `Inventory.Replenished` — published by supplier-service
- `Customer.New` — published by customer-service
- `AI.OrderForecasted` — published by ai-service (Phase 2)

Use `_shared/nats` `EventEnvelope` for all publish/subscribe calls.

### Go service internal layout

Each service follows the same hexagonal pattern:

```
internal/
  domain/          — aggregates, value objects, domain events, repository interfaces
  application/
    commands/      — write path: Command struct + Handler
    queries/       — read path: Query struct + Handler
    ports.go       — EventPublisher interface
  interfaces/grpc/ — implements the generated gRPC server interface
  infrastructure/
    postgres/      — pgx repository implementation + sqlc-generated queries
    grpc_clients/  — downstream gRPC clients (only in sales, supplier)
    nats_publisher.go
```

Business rules live in `domain/` and `application/commands/`. Infrastructure adapters implement the ports. Never put business logic in `interfaces/grpc/` or `infrastructure/`.

### Proto conventions

Package namespace: `pos.<service>.v1`. All request messages that are store-scoped include `string tenant_id = 1` and `string store_id = 2` as the first two fields. HTTP annotations (`google.api.http`) are on every RPC to enable grpc-gateway transcoding. After editing a `.proto`, always run `make proto-gen && make proto-lint`.

### SvelteKit apps

Two separate apps share `packages/ui` (Svelte component library, pnpm workspace):

- `apps/admin` — tenant admin dashboard; `hooks.server.ts` enforces admin role.
- `apps/cashier` — cashier POS; `hooks.server.ts` enforces cashier role; includes `service-worker.ts` for PWA offline support.

API calls go to the Laravel gateway at `PUBLIC_GATEWAY_URL` (set in `.env`).

---

## Key constraints from PRD

- Stock must never go negative (enforced in inventory-service domain layer).
- Every completed sale must produce a receipt record (JSON snapshot in `sales.receipts`).
- Customer loyalty points are updated only after a fully committed sale.
- All report queries must be scoped by both `tenant_id` and `store_id`.
- Phase 1 has no public tenant signup — tenants are seeded via console command.
- Elasticsearch and MinIO are optional and excluded from Phase 1.

---

## PM2 Services

> Prerequisite: `make dev` must be running (infra). Install PM2 once: `npm install -g pm2`

| Port | Name | Type |
|------|------|------|
| 5173 | pos-admin-5173 | SvelteKit (admin dashboard) |
| 5174 | pos-cashier-5174 | SvelteKit (cashier POS) |
| 8000 | pos-gateway-8000 | Go API gateway |

**Terminal Commands:**
```bash
pm2 start ecosystem.config.cjs   # First time (loads config)
pm2 start all                    # After first time
pm2 stop all / pm2 restart all
pm2 start pos-admin-5173 / pm2 stop pos-admin-5173
pm2 logs / pm2 status / pm2 monit
pm2 save                         # Save process list
pm2 resurrect                    # Restore saved list
```
