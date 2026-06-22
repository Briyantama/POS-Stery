---
name: pos-stery-scaffold
description: >
  Scaffold the POS-Stery monorepo layer by layer in the correct dependency
  order: foundation → proto contracts → shared Go library → auth service →
  domain services → Laravel gateway → SvelteKit apps → CI. Use this skill
  whenever the user says "scaffold", "init a service", "create the proto
  files", "set up the gateway", "build sprint N", or asks to implement any
  part of the POS-Stery Phase 1 structure. Also use it when a specific file
  type is requested (Dockerfile, migration, go.mod, SKILL.md, buf.yaml) within
  the POS-Stery project directory.
---

# POS-Stery Scaffold Skill

Reference `CLAUDE.md` and `PRD.md` at the repo root before writing any file.
All work must trace to a PRD section.

---

## Sprint Dependency Order

Never scaffold a later sprint before its dependencies are complete.

```
Sprint 0 (Foundation)      ← no dependencies
Sprint 1 (Contracts)       ← Sprint 0
Sprint 2 (Shared library)  ← Sprint 1
Sprint 3 (Auth service)    ← Sprint 2
Sprint 4a–e (Services)     ← Sprint 2 (parallel); 4c,4e need 4b (inventory) running
Sprint 5 (Laravel gateway) ← Sprint 3
Sprint 6 (SvelteKit apps)  ← Sprint 5
Sprint 7 (CI + scripts)    ← all sprints
```

---

## Sprint 0 — Foundation

Files to create (all independent, can be parallel):

### `.gitignore`

Cover Go binaries, PHP vendor, Node modules, `.env`, `gen/` (proto output), IDE files, OS files.

### `deploy/docker/docker-compose.yml`

Services: `postgres:16-alpine` (port 5432), `redis:7-alpine` (port 6379),
`nats:2.10-alpine` with `nats.conf` mount (ports 4222, 8222),
`jaegertracing/all-in-one:1.58` (ports 16686, 4317, 4318),
`prom/prometheus:v2.53.0` (port 9090).
Single bridge network `pos-net`. Named volumes for postgres data and nats data.
**No application services here** — services run via `go run` locally.

### `deploy/docker/init-postgres.sql`

```sql
CREATE ROLE pos_admin WITH LOGIN PASSWORD '${POSTGRES_ADMIN_PASSWORD}' CREATEDB BYPASSRLS;
CREATE ROLE pos_app   WITH LOGIN PASSWORD '${POSTGRES_APP_PASSWORD}';

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Each service creates its own schema in its migration 001 file.
-- pos_app is granted USAGE on each schema by the migration that creates it.
```

### `deploy/docker/nats.conf`

```
port: 4222
http_port: 8222

jetstream {
  store_dir: /data/jetstream
}

accounts {
  POS {
    jetstream: enabled
    users: [{ user: pos, password: "${NATS_PASSWORD}" }]
  }
}
```

Define the `POS_EVENTS` stream programmatically in `_shared/nats/streams.go` (not in the config file), so services self-register on startup.

### `.env.example`

```
# Postgres
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=pos_db
POSTGRES_ADMIN_PASSWORD=changeme
POSTGRES_APP_PASSWORD=changeme

# Redis
REDIS_ADDR=localhost:6379

# NATS
NATS_URL=nats://localhost:4222
NATS_PASSWORD=changeme

# Auth service
JWT_RS256_PRIVATE_KEY_PATH=./keys/private.pem
JWT_RS256_PUBLIC_KEY_PATH=./keys/public.pem
SERVICE_TOKEN_SECRET=changeme

# Gateway
APP_URL=http://localhost:8000
GATEWAY_PORT=8000
ADMIN_APP_URL=http://localhost:5173
CASHIER_APP_URL=http://localhost:5174

# Service gRPC ports (gateway HTTP/JSON transcoding)
AUTH_SERVICE_URL=http://localhost:8081
PRODUCT_SERVICE_URL=http://localhost:8082
INVENTORY_SERVICE_URL=http://localhost:8083
SALES_SERVICE_URL=http://localhost:8084
SUPPLIER_SERVICE_URL=http://localhost:8085
CUSTOMER_SERVICE_URL=http://localhost:8086
```

### `Makefile`

See the Makefile Targets section below.

---

## Sprint 1 — Proto Contracts

### `proto/buf.yaml`

```yaml
version: v2
modules:
  - path: .
lint:
  use: [STANDARD]
  except: [PACKAGE_VERSION_SUFFIX]
breaking:
  use: [FILE]
deps:
  - buf.build/googleapis/googleapis
```

### `proto/buf.gen.yaml`

```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/pos-stery/pos-stery/gen/go
plugins:
  - remote: buf.build/protocolbuffers/go
    out: ../gen/go
    opt: [paths=source_relative]
  - remote: buf.build/grpc/go
    out: ../gen/go
    opt: [paths=source_relative]
  - remote: buf.build/grpc-ecosystem/gateway
    out: ../gen/go
    opt: [paths=source_relative, generate_unbound_methods=true]
  - remote: buf.build/grpc-ecosystem/openapiv2
    out: ../gen/openapi
```

### Proto file structure (applies to all 6 services)

```proto
syntax = "proto3";
package pos.<service>.v1;

import "google/api/annotations.proto";

option go_package = "github.com/pos-stery/pos-stery/gen/go/pos/<service>/v1;<service>v1";

service <Name>Service {
  rpc MethodName(MethodNameRequest) returns (MethodNameResponse) {
    option (google.api.http) = {
      post: "/v1/<resource>"
      body: "*"
    };
  }
}

message MethodNameRequest {
  string tenant_id = 1;
  string store_id  = 2;  // omit for tenant-level operations
  // ... domain fields
}
```

Key fields per service (all request messages start with tenant_id + store_id):

**auth.proto** — `Login(email, password) → (token, expires_at)`, `Validate(token) → (claims)`
**product.proto** — `CreateProduct`, `UpdateProduct`, `SearchProducts(query, category, limit, offset)`
**inventory.proto** — `AddItem(product_id, quantity, threshold)`, `UpdateStock(product_id, quantity_delta, reason: SALE|ADJUSTMENT|RECEIPT)`, `ListItems`, `GetItem`, `SetThreshold`
**sales.proto** — `CreateSale(items: [{product_id, quantity, unit_price}], customer_id?, discount_amount?)`, `GetSalesReport(date, granularity: DAY|STORE)`
**supplier.proto** — `AddSupplier`, `CreatePurchaseOrder(supplier_id, items)`, `ReceiveStock(purchase_order_id, items_received)`
**customer.proto** — `CreateCustomer`, `GetCustomer`, `AddLoyaltyPoints(sale_id, points)`

After writing all .proto files:

```bash
cd proto && buf dep update && buf generate
```

### `gen/go/go.mod`

```
module github.com/pos-stery/pos-stery/gen/go
go 1.24
```

Add generated proto dependencies after running `buf generate`.

### `go.work`

```
go 1.24

use (
    ./gen/go
    ./services/_shared
    ./services/auth-service
    ./services/product-service
    ./services/inventory-service
    ./services/sales-service
    ./services/supplier-service
    ./services/customer-service
)
```

---

## Sprint 2 — Shared Library (`services/_shared`)

Module path: `github.com/pos-stery/pos-stery/services/_shared`

### `database/pool.go`

`NewPool(cfg Config) (*pgxpool.Pool, error)` — configures pgx pool, sets `default_transaction_isolation` to `read committed`.

### `database/tenant_context.go` ← CRITICAL

```go
// WithTenantContext wraps fn in a transaction that sets app.tenant_id and
// app.store_id as local parameters before execution. RLS policies depend on
// these settings — never call repository methods without this wrapper.
func WithTenantContext(ctx context.Context, pool *pgxpool.Pool,
    tenantID, storeID string, fn func(pgx.Tx) error) error
```

The transaction executes:

```sql
SET LOCAL app.tenant_id = '<tenantID>';
SET LOCAL app.store_id  = '<storeID>';
```

before calling `fn`. If `storeID` is empty (admin-level operations), set it to `''`.

### `middleware/grpc_interceptors.go`

Unary and stream interceptors for:

- **TenantEnforcer** — rejects requests where `tenant_id` field is empty or doesn't match the JWT claim extracted by the gateway.
- **ServiceAuthValidator** — validates `X-Service-Token` HMAC-SHA256 header for service-to-service calls.
- **RequestLogger** — zap structured logging with `tenant_id`, `store_id`, method, duration.
- **MetricsRecorder** — Prometheus `grpc_server_handled_total` and `grpc_server_handling_seconds`.
- **OtelTracer** — OpenTelemetry span creation.

### `nats/jetstream.go`

`NewClient(url, password string) (nats.JetStreamContext, error)`

### `nats/event_envelope.go`

```go
type EventEnvelope struct {
    EventType  string    `json:"event_type"`
    TenantID   string    `json:"tenant_id"`
    StoreID    string    `json:"store_id"`
    OccurredAt time.Time `json:"occurred_at"`
    Payload    any       `json:"payload"`
}
```

All NATS publishes wrap their payload in `EventEnvelope`.

### `nats/streams.go`

`EnsureStreams(js nats.JetStreamContext) error` — creates `POS_EVENTS` stream on startup if it doesn't exist.
Subjects: `Product.LowStock`, `Sale.Completed`, `Inventory.Replenished`, `Customer.New`, `AI.OrderForecasted`.

### `server/grpc_server.go`

`New(cfg Config, svc grpc.ServiceDesc, impl any) *grpc.Server` — builds a server with the standard interceptor chain from `middleware`.

### `server/shutdown.go`

`GracefulShutdown(grpcSrv *grpc.Server, httpSrv *http.Server, timeout time.Duration)`

---

## Sprint 3 — Auth Service

Migration `001_create_auth_schema.up.sql` must:

1. `CREATE SCHEMA auth;`
2. `GRANT USAGE ON SCHEMA auth TO pos_app;`
3. Create tables: `tenants`, `stores`, `users`, `roles`, `permissions`, `user_roles`
4. All tables: `tenant_id UUID NOT NULL`, `created_at TIMESTAMPTZ DEFAULT NOW()`, `updated_at TIMESTAMPTZ DEFAULT NOW()`, `deleted_at TIMESTAMPTZ NULL`
5. `stores` has `tenant_id` FK to `tenants.id`
6. Enable RLS on all tables except `tenants` (tenants is cross-tenant by design for the superadmin role)
7. Create RLS policy: `CREATE POLICY tenant_isolation ON auth.<table> USING (tenant_id = current_setting('app.tenant_id')::uuid)`
8. `CREATE TABLE auth.audit_logs (...)` — append-only, no RLS, logged by `pos_admin`

`jwt_signer.go` — RS256 signing with key loaded from `JWT_RS256_PRIVATE_KEY_PATH`. Issued claims: `sub` (user_id), `tid` (tenant_id), `sid` (store_id, empty for admin), `role` (admin|cashier|stock_manager), `exp`.

`cmd/server/main.go` pattern for all services:

```go
func main() {
    cfg := config.Load()
    log := observability.NewLogger(cfg)
    pool := database.NewPool(cfg.DB)
    js := nats.NewClient(cfg.NATS)
    nats.EnsureStreams(js)
    redis := redis.NewClient(cfg.Redis)
    // wire repos, handlers, publishers
    srv := server.New(cfg.GRPC, authv1.AuthService_ServiceDesc, grpcImpl)
    server.GracefulShutdown(srv, nil, 30*time.Second)
}
```

---

## Sprint 4 — Domain Services

Apply the same pattern for each service. Key service-specific notes:

### inventory-service (4b — build before sales and supplier)

`UpdateStockHandler` must enforce the no-negative rule before writing:

```go
if item.Quantity + cmd.QuantityDelta < 0 {
    return ErrStockWouldGoNegative
}
```

After a successful update, check against threshold and publish `Product.LowStock` if `item.Quantity <= threshold.MinQuantity`.

Migration must create `inventory.stock_thresholds(tenant_id, store_id, product_id, min_quantity)` with a unique index on `(tenant_id, store_id, product_id)`.

### sales-service (4c — depends on inventory running)

`CreateSaleHandler` synchronous saga:

```
1. call InventoryService.UpdateStock (quantity_delta: -qty) for each line item
2. if all deductions succeed → write sale + sale_items + receipt to DB
3. if DB write fails → for each deducted item, call InventoryService.UpdateStock (quantity_delta: +qty) to compensate
4. publish Sale.Completed on success
```

Receipt stored as `jsonb` snapshot: `{sale_id, tenant_id, store_id, items, total, discount, cashier_id, completed_at}`.

`grpc_clients/inventory_client.go` — wraps `inventoryv1.InventoryServiceClient` with the service auth header injected.

### supplier-service (4e — depends on inventory running)

`ReceiveStockHandler` calls `InventoryService.UpdateStock` (reason: RECEIPT) for each received item, then publishes `Inventory.Replenished`.

---

## Sprint 5 — Laravel API Gateway

```bash
cd apps && composer create-project laravel/laravel api-gateway
```

### Middleware chain (applied globally)

```php
// routes/api.php middleware group order:
// JwtValidation → TenantResolver → StoreResolver → CheckRole
```

`JwtValidation.php` — verifies RS256 JWT using the public key from `JWT_RS256_PUBLIC_KEY_PATH` env. Sets `request->tenantId`, `request->storeId`, `request->userId`, `request->role`.

`StoreResolver.php` — if `request->storeId` is empty, reads `X-Store-ID` header. Validates the store belongs to `request->tenantId` by calling `AuthServiceClient::validateStore`. Rejects with 403 if mismatch.

`CheckRole.php` — middleware constructor takes `string ...$allowed` roles. Route groups use `->middleware('role:admin')` or `->middleware('role:cashier')`.

### gRPC client pattern

```php
class BaseGrpcGatewayClient {
    // All clients call the grpc-gateway HTTP/JSON transcoding endpoint
    // using Guzzle. Auth header: Authorization: Bearer <service-token>
    protected function post(string $service, string $path, array $body): array
    protected function get(string $service, string $path, array $query = []): array
}
```

Each `*ServiceClient.php` extends `BaseGrpcGatewayClient` and wraps individual RPC methods. All calls automatically inject `tenant_id` and `store_id` from the current request.

### `routes/api.php` endpoints (PRD §11.2)

```
POST   /api/login                  AuthController@login       (public)
POST   /api/logout                 AuthController@logout      (jwt)
GET    /api/products               ProductController@index    (jwt, role:admin,cashier)
POST   /api/products               ProductController@store    (jwt, role:admin)
GET    /api/inventory              InventoryController@index  (jwt, role:admin,stock_manager)
POST   /api/inventory              InventoryController@store  (jwt, role:admin,stock_manager)
POST   /api/sales                  SalesController@store      (jwt, role:cashier)
GET    /api/sales/report           SalesController@report     (jwt, role:admin)
GET    /api/suppliers              SupplierController@index   (jwt, role:admin,stock_manager)
POST   /api/suppliers              SupplierController@store   (jwt, role:admin)
POST   /api/purchase-orders        SupplierController@order   (jwt, role:admin,stock_manager)
POST   /api/ai/forecast-demand     AIController@forecast      (jwt, role:admin)
```

---

## Sprint 6 — SvelteKit Apps

```bash
# root
echo "packages:\n  - 'apps/*'\n  - 'packages/*'" > pnpm-workspace.yaml

# admin
cd apps && npx sv create admin --template skeleton --types ts --no-add-ons
# cashier (add service-worker for PWA)
npx sv create cashier --template skeleton --types ts --no-add-ons
```

### `hooks.server.ts` pattern (both apps)

```ts
export const handle: Handle = async ({ event, resolve }) => {
  const token = event.cookies.get("pos_token");
  if (!token) return redirect(302, "/login");
  const claims = verifyJwt(token); // verify RS256 with PUBLIC_KEY env
  if (claims.role !== REQUIRED_ROLE) return redirect(302, "/login");
  event.locals.user = claims;
  return resolve(event);
};
```

Admin uses `REQUIRED_ROLE = 'admin'`. Cashier uses `REQUIRED_ROLE = 'cashier'`.

### Admin routes stubs

Each route exports a `+page.server.ts` that fetches from the Laravel gateway using the server-side token and a `+page.svelte` that renders the data. Implement as real pages (not empty shells) with at minimum a list view and a create form.

### Cashier POS route (`apps/cashier/src/routes/(app)/pos`)

This is the most latency-critical screen. Cart state is local (`$state` rune). Product search calls `GET /api/products?q=` with debounce. Checkout calls `POST /api/sales`. Show a receipt modal on success.

---

## Sprint 7 — CI + Scripts

### `.github/workflows/ci.yml`

Jobs (run in parallel where possible):

1. `proto-lint` — `buf lint && buf breaking --against .git#branch=main`
2. `go-test` — `go test -short ./...` across all workspace members
3. `migrate-validate` — verify every service has matching up/down pairs
4. `gateway-test` — `composer install && vendor/bin/pest`

### `scripts/migrate-helper.sh`

Usage: `./scripts/migrate-helper.sh up inventory` or `./scripts/migrate-helper.sh create inventory add_threshold`
Wraps `golang-migrate` with the correct database URL and migrations path per service.

### Complete Makefile targets

See CLAUDE.md §Commands for the full list. Key additions for Sprint 7:

- `make sqlc-gen` — discover all `sqlc.yaml` files and run `sqlc generate`
- `make test-integration` — `INTEGRATION=true go test ./...` (requires Docker infra running)
- `make ci` — proto-lint + migrate-validate + test-go + test-gateway (local CI gate)

---

## Patterns to Repeat Consistently

- Every migration `001_create_<schema>_schema.up.sql` must: create schema, grant usage to `pos_app`, enable RLS on every table with `tenant_id`, create the tenant isolation policy.
- Every `cmd/server/main.go` uses the same boot sequence: config → logger → pool → nats → redis → wire → serve → graceful shutdown.
- Every gRPC server implementation file has the signature `func New(deps *Deps) *ServiceServer` where `Deps` holds repos and the event publisher.
- Every `nats_publisher.go` wraps events in `_shared/nats.EventEnvelope` before publishing.
- `sqlc.yaml` in each service points schema to `./migrations/*.up.sql` and queries to `./internal/infrastructure/postgres/queries/*.sql`.
