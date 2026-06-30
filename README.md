# POS-Stery

Multi-tenant Point of Sale SaaS platform for retail store chains. Supports multi-store tenants, cashier checkout flows, inventory management, supplier orders, customer loyalty, and daily reporting.

---

## Architecture

```
Browser (SvelteKit)
    │  REST/JSON
    ▼
api-gateway  ← Go · chi router · RS256 JWT · Redis JTI blacklist · RBAC
    │  gRPC (internal network)
    ▼
Go microservices  ← each owns a Postgres schema · publishes to NATS JetStream
    │  NATS JetStream (POS_EVENTS stream)
    ▼
Event consumers  ← Sale.Completed · Product.LowStock · Inventory.Replenished
```

### Services

| Service | Port | Responsibility |
|---|---|---|
| `api-gateway` | 8000 | HTTP gateway, auth middleware, RBAC, route proxy to gRPC backends |
| `auth-service` | 8081 | Tenants, stores, users, roles, JWT issue/refresh/revoke |
| `product-service` | 8082 | Product catalog, SKU management |
| `inventory-service` | 8083 | Stock levels, thresholds, low-stock events |
| `sales-service` | 8084 | Sales, receipts, saga with inventory deduction |
| `supplier-service` | 8085 | Suppliers, purchase orders, replenishment |
| `customer-service` | 8086 | Customer records, loyalty points |

### Frontend Apps

| App | Port | Description |
|---|---|---|
| `apps/admin` | 5173 | Tenant admin dashboard (SvelteKit) |
| `apps/cashier` | 5174 | Cashier POS with offline PWA support (SvelteKit) |

### Observability

Jaeger (tracing), Prometheus (metrics, per-service ports 9101–9106), structured logging via OpenTelemetry.

---

## Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Docker + Docker Compose | latest | Required for infra |
| Go | 1.22+ | For backend services |
| Node.js | 20+ | For SvelteKit apps |
| pnpm | 9+ | Package manager for frontend |
| `buf` | latest | Protobuf toolchain |
| `golang-migrate` | v4 | Database migrations |
| `sqlc` | latest | SQL code generation |
| PM2 | latest | Process manager for local dev |

---

## Quick Start

### 1. Clone and configure

```bash
git clone <repo-url>
cd POS-Stery
cp .env.example .env
```

Edit `.env` and set passwords. Generate RS256 keys for JWT:

```bash
mkdir -p keys
openssl genrsa -out keys/private.pem 2048
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
```

### 2. Start infrastructure

```bash
make dev
```

Starts Postgres 16, Redis 7, NATS 2.10, Jaeger, and Prometheus via Docker Compose.

### 3. Initialize database

```bash
make db-init       # Create pos_admin and pos_app roles
make migrate-up    # Apply all migrations
```

### 4. Build and run

```bash
make all           # proto-gen + migrate-up + build-go + install frontend deps
```

Then start all services with PM2:

```bash
npm install -g pm2          # install once globally
pm2 start ecosystem.config.cjs
pm2 status
```

| Process | Port | URL |
|---|---|---|
| `pos-admin-5173` | 5173 | <http://localhost:5173> — Admin dashboard |
| `pos-cashier-5174` | 5174 | <http://localhost:5174> — Cashier POS |
| `pos-gateway-8000` | 8000 | <http://localhost:8000> — API gateway |
| `pos-auth-8081` | 8081 | Auth service (gRPC) |
| `pos-product-8082` | 8082 | Product service (gRPC) |
| `pos-inventory-8083` | 8083 | Inventory service (gRPC) |
| `pos-sales-8084` | 8084 | Sales service (gRPC) |
| `pos-supplier-8085` | 8085 | Supplier service (gRPC) |
| `pos-customer-8086` | 8086 | Customer service (gRPC) |

---

## Development

### Infrastructure lifecycle

```bash
make dev           # Start infra containers
make dev-down      # Stop infra containers
make dev-reset     # DESTRUCTIVE: destroy volumes + restart
```

### Protobuf

```bash
make proto-gen      # buf generate → gen/go/ and gen/openapi/
make proto-lint     # buf lint
make proto-breaking # Check for breaking changes against main
```

Always run `make proto-gen && make proto-lint` after editing any `.proto` file. Never hand-edit files under `gen/`.

### Database migrations

```bash
make migrate-up                                     # Apply all pending
make migrate-down                                   # Roll back one step per service
make migrate-create NAME=add_index SERVICE=product  # Create new migration pair
make migrate-fresh                                  # DESTRUCTIVE: drop + reapply all
```

Migrations live in `services/<name>-service/migrations/`.

### Go services

```bash
make build-go          # Build all services
make test-go           # go test -short ./...
make test-integration  # INTEGRATION=true go test ./...
make lint-go           # golangci-lint
make tidy              # go mod tidy across all workspace members
make sqlc-gen          # sqlc generate for all services
```

Run tests for a single service:

```bash
cd services/inventory-service && go test ./internal/application/commands/...
```

### SvelteKit apps

```bash
make install-admin     # pnpm install in apps/admin
make dev-admin         # pnpm dev (admin, :5173)
make install-cashier   # pnpm install in apps/cashier
make dev-cashier       # pnpm dev (cashier, :5174)
```

### PM2

`ecosystem.config.cjs` defines all 9 processes (2 frontend, 1 gateway, 6 backend services). PM2 must be running while `make dev` infra containers are up.

**Start / stop all**

```bash
pm2 start ecosystem.config.cjs   # First run — loads config and starts everything
pm2 restart all                  # Restart all processes
pm2 stop all                     # Stop all (keeps processes in list)
pm2 delete all                   # Remove all from PM2 list
```

**Start / stop individual processes**

```bash
pm2 start pos-admin-5173
pm2 start pos-cashier-5174
pm2 start pos-gateway-8000
pm2 start pos-auth-8081
pm2 start pos-product-8082
pm2 start pos-inventory-8083
pm2 start pos-sales-8084
pm2 start pos-supplier-8085
pm2 start pos-customer-8086

pm2 stop pos-gateway-8000        # Stop one process by name
pm2 restart pos-auth-8081        # Restart one process by name
```

**Logs and monitoring**

```bash
pm2 logs                         # Tail logs from all processes
pm2 logs pos-auth-8081           # Tail logs for one process
pm2 logs --lines 200             # Show last 200 lines
pm2 status                       # Process table (pid, status, cpu, memory)
pm2 monit                        # Real-time dashboard in terminal
```

**Persist across reboots**

```bash
pm2 save                         # Save current process list
pm2 resurrect                    # Restore saved list
pm2 startup                      # Generate OS startup hook (run output as root/admin)
```

---

## Tenant Isolation

Isolation is enforced at four layers:

1. **Gateway** — `TenantResolver` and `StoreResolver` middleware validate JWT claims and confirm `store_id` belongs to the authenticated tenant.
2. **gRPC contracts** — every request message includes `tenant_id` and `store_id`.
3. **Service layer** — command/query handlers pass both IDs through to the repository.
4. **Database** — PostgreSQL RLS on every tenant-scoped table. The `_shared/database` package calls `WithTenantContext(ctx, tenantID, storeID)` before every query to inject `SET LOCAL app.tenant_id` and `SET LOCAL app.store_id`.

Two DB roles: `pos_admin` (BYPASSRLS — migrations only) and `pos_app` (subject to RLS — all application code).

---

## Database Layout

Single Postgres instance, one schema per service:

| Schema | Service |
|---|---|
| `auth` | auth-service — tenants, stores, users, roles |
| `product` | product-service |
| `inventory` | inventory-service |
| `sales` | sales-service |
| `supplier` | supplier-service |
| `customer` | customer-service |

No cross-schema foreign keys. Services reference other domains by UUID only.

---

## Key Business Rules

- **Stock never goes negative** — enforced in `inventory-service` domain layer before any DB write.
- **Every sale produces a receipt** — JSON snapshot stored in `sales.receipts`.
- **Loyalty points update only after a committed sale.**
- **All report queries are scoped by `tenant_id` and `store_id`.**
- **Phase 1 has no self-service signup** — tenants are seeded via console.

---

## Service Internal Layout

Each Go service follows hexagonal architecture:

```
internal/
  domain/           — aggregates, value objects, domain events, repository interfaces
  application/
    commands/       — write path: Command struct + Handler
    queries/        — read path: Query struct + Handler
    ports.go        — EventPublisher interface
  interfaces/grpc/  — gRPC server implementation
  infrastructure/
    postgres/       — pgx repository + sqlc-generated queries
    grpc_clients/   — downstream gRPC clients (sales, supplier only)
    nats_publisher.go
```

Business rules live only in `domain/` and `application/commands/`. Infrastructure adapters implement the ports.

---

## NATS Events

Stream `POS_EVENTS`, file storage, 7-day retention:

| Subject | Publisher |
|---|---|
| `Product.LowStock` | inventory-service |
| `Sale.Completed` | sales-service |
| `Inventory.Replenished` | supplier-service |
| `Customer.New` | customer-service |
| `AI.OrderForecasted` | ai-service (Phase 2) |

---

## Environment Variables

Copy `.env.example` to `.env`. Key variables:

| Variable | Description |
|---|---|
| `POSTGRES_APP_PASSWORD` | Password for `pos_app` role |
| `JWT_RS256_PRIVATE_KEY_PATH` | Path to RS256 private key |
| `SERVICE_TOKEN_SECRET` | HMAC-SHA256 secret for service-to-service auth |
| `PUBLIC_GATEWAY_URL` | Gateway URL for SvelteKit apps (default `http://localhost:8000`) |
| `NATS_URL` | NATS connection string |
| `REDIS_ADDR` | Redis address |
