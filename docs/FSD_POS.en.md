# FSD POS-Stery

*Functional Specification Document: multi-tenant POS SaaS for retail stores (the `POS` product in Stery Suite): current functional behaviour (as-is) and the target architecture following STACK-SPEC*

| | |
|---|---|
| **Document Name** | FSD POS-Stery (Functional Specification Document) |
| **Version** | 1.0 (Draft) |
| **Date** | 6 November 2026 |
| **Prepared by** | Product & Engineering Team |
| **Status** | Draft for review |
| **Related Documents** | `PRD.md` (POS-Stery PRD, in English); `BRD_POS.id.md` (business requirements BR-POS-01 to BR-POS-21); `00_Analisis_Dampak_Risiko` (Stery Suite impact and risk analysis); `SELA/CLAUDE.md` and `SELA/docs/FSD_Sela.md` section 2 (STACK-SPEC source) |
| **Changes from previous** | Changes from previous: First issue |

---

## 1. Introduction

### 1.1 Purpose

This FSD is the technical reference for POS-Stery: (a) it documents the functional behaviour that **already exists in code**, with status and evidence, (b) it sets the target architecture identical to the Stery Suite STACK-SPEC (source: SELA) and decisions D1-D12, and (c) it lists the as-is -> to-be gaps in section 2.7 so the alignment (workstream S2) can be planned. Business requirements are in `BRD_POS.id.md`; product requirements are in `PRD.md`, which is not modified.

### 1.2 Scope

POS catalogue modules: 01 AUTH, 02 PRD (catalogue), 03 INV, 04 SAL, 05 SUP, 06 CUS, 07 RPT, 08 NTF (Not started), 09 AI (Not started). Plus cross-module: security (`FR-POS-SEC.n`) and non-functional (`NFR-POS-NN`, section 9). There is no new POS PRD; every reference to "PRD" means `PRD.md`.

### 1.3 Conventions

- **Status:** Done, Partial, Not started, Pending Verification, Deferred. **Priority:** M (Must), S (Should), C (Could).
- **Truth = code.** Paths in the `Evidence` column are relative to the `POS-Stery` repo root and exist on disk on the document date. Status documents that contradict the code are recorded in 11.3 and are not repeated as fact.
- **As-is vs to-be.** Section 4 describes current behaviour (status + evidence) and, under "Target", the behaviour required by the target stack. Target code paths appear only in the delta table 2.7, never in the `Evidence` column.
- **No business figures.** Prices, SLAs, quotas and performance targets are `tbd` together with the open decision `D-POS-NN` (defined in BRD section 9.3). Effort estimates are marked `[INFERENCE]`.
- **Secrets.** The document only names file paths that contain secrets, never their values.
- **Static reading.** No build, lint or test was run to compile this document; runtime behaviour findings are marked `[INFERENCE]` or Pending Verification.

### 1.4 Summary of code state (verified facts)

| Fact | Status | Evidence |
|---|---|---|
| Six gRPC services (auth, product, inventory, sales, supplier, customer), all 28 RPCs have handlers | Done | `proto/pos/auth/v1/auth.proto` |
| Go `chi` REST gateway with 14 business routes, not Laravel | Done | `services/api-gateway/cmd/server/main.go` |
| The `/v1/...` HTTP annotations in the proto are not mounted by any mux (no grpc-gateway) | Done | `proto/pos/sales/v1/sales.proto` |
| 79 unit test functions (stub-based); no Postgres integration tests | Done | `services/sales-service/internal/application/commands/create_sale_test.go` |
| Two SvelteKit SSR apps (admin, cashier), all admin pages read-only | Partial | `apps/admin/src/hooks.server.ts` |

---

## 2. Architecture and Stack

### 2.1 Stack (STACK-SPEC, source SELA)

The following table is the normative Stery Suite STACK-SPEC; the "Current POS state" column shows the starting point for alignment. The source column points to SELA files (the truth for the standard).

| # | Layer | Standard (POS target) | SELA source | Current POS state |
|---|---|---|---|---|
| 1 | Repo layout | per product: `apps/api`, `apps/web`, `deploy/`, `scripts/`, `docs/`, `Makefile`, `CLAUDE.md`, `AGENTS.md` | `SELA/CLAUDE.md` ("Layout"); `SELA/docs/FSD_Sela.md` 2.6 | monorepo `services/*` (6 services + gateway), `apps/admin`, `apps/cashier`, `packages/ui`, `gen/go`, `go.work`; no `AGENTS.md` |
| 2 | Language | Go >= 1.25.1, one module per product `github.com/<owner>/<product>` (owner handle `Briyantama` as in SELA; not verified for new repos) | `SELA/apps/api/go.mod` | `go.work` 1.25.0, `services/_shared/go.mod` 1.24.0, each service has its own `go.mod`; CI uses Go 1.24 |
| 3 | Decomposition | modules = Go packages `services/<name>` in one process: core `Service` + Postgres repository + HTTP adapter + optional gRPC adapter; boundaries through Go interfaces and proto contracts so they can be split later | `SELA/CLAUDE.md` ("Layout"); `SELA/docs/FSD_Sela.md` 2.2 | seven separate processes calling each other over gRPC |
| 4 | Transport | REST `/api/v1` on stdlib `net/http` ServeMux, envelope `{success,data,error}` via `kit/httpx`; gRPC (`api/proto/v1`, buf, committed `gen/go`, `scripts/proto-gen.sh --check`) for machine/internal clients with a deny-by-default interceptor; no gateway tier; Caddy only as reverse proxy | `SELA/apps/api/internal/httpx`; `SELA/scripts/proto-gen.sh`; `SELA/deploy/Caddyfile` | `chi` gateway `/api/...` without version, Laravel-style `{message}` errors; inter-service gRPC with fail-open `X-Service-Token` |
| 5 | Database | PostgreSQL 18 (`uuidv7()`), pgx/v5 raw SQL (no sqlc/ORM), embedded goose v3.26.0 + `cmd/migrate up\|down`, `testdb.NewMigrated` | `SELA/apps/api/internal/db`; `SELA/apps/api/internal/testdb`; `SELA/apps/api/cmd/migrate` | PostgreSQL 16, per-service golang-migrate, `gen_random_uuid()`, sqlc only in auth-service, pgx 5.6.0 |
| 6 | Tenant isolation | RLS mandatory (D8): `ENABLE` + `FORCE`, `tenant_isolation` policy with `USING` and `WITH CHECK`, `current_setting('app.tenant_id', true)::uuid` set per transaction by `kit/tenantdb.WithTenant`; POS adds `app.store_id`; app role NOBYPASSRLS, separate migration role; cross-tenant tests mandatory per module | Decision D8 (SELA has no tenant/RLS yet; pattern carried over from POS/HRIS) | `ENABLE`+`FORCE` on all tenant tables, 7 of 15 policies without `WITH CHECK`, no policy uses `app.store_id` |
| 7 | State store | Dragonfly via go-redis/v9: sessions (hash only), rate limits, quotas, queues, Lua scripts | `SELA/apps/api/services/auth/service.go` | Redis (JWT blacklist, gateway rate limit), go-redis 9.6.1 |
| 8 | Object storage | RustFS (S3) via aws-sdk-go-v2 behind `objstore.Store`; presigned URLs only; ClamAV `clamd` scan fail-closed on upload paths | `SELA/apps/api/internal/objstore`; `SELA/apps/api/services/media/scan` | not used (POS does not store files yet; receipt PDF in phase 2) |
| 9 | Authentication | opaque session in HttpOnly Secure cookie, hash in Dragonfly; RBAC tables (roles, permissions, actions, role_actions) + `RequirePermission(code)`; no JWT and no tokens in browser storage | `SELA/apps/api/services/auth/service.go`; `SELA/apps/api/services/rbac` | JWT RS256 (`tid`, `sid`, `role`), httpOnly cookie holding the JWT, only a `roles` table |
| 10 | Async | in-process ticker workers + Postgres outbox + idempotent consumers; no NATS/Kafka | `SELA/apps/api/cmd/api/main.go`; `SELA/apps/api/services/event/notify.go` | NATS JetStream stream `POS_EVENTS`, best-effort publishing, no subscribers |
| 11 | Observability | `log/slog`, `/healthz`; OpenTelemetry/Prometheus are not baseline (decision D11, open) | `SELA/apps/api/internal/health` | zap; `/healthz` and `/readyz` on the gateway; no OTel/Prometheus code |
| 12 | Configuration | environment variables, fail-fast validation, secrets not logged, `.env.example` | `SELA/apps/api/internal/config` | viper + `.env.example`; literal secrets in `ecosystem.config.cjs` |
| 13 | PII and audit | `kit/piicrypt` (AES-256-GCM, per-purpose AAD, blind index, key rotation); `kit/audit` append-only `audit_logs` with DB triggers | `SELA/apps/api/internal/piicrypt`; `SELA/apps/api/internal/audit` | customer PII in plain text; table `auth.audit_logs` exists but is never written and has no trigger |
| 14 | Payments | `payment.Provider`, Midtrans QRIS (Xendit removed), signed webhook, paid notification matched against amount | `SELA/apps/api/services/payment/provider.go`; `SELA/apps/api/services/payment/midtrans.go` | none; POS does not implement a payment gateway (PRD §4). POS subscription billing is done by the platform (`suite` mode) |
| 15 | Web | SvelteKit 2 + Svelte 5 + Vite 8 + TypeScript 6, `ssr=false`, `adapter-static`, npm, Node >= 20.19; `apiFetch` + envelope mirror; i18n `{id,en}` with parity test; Vitest 4 + Testing Library; Playwright (Chrome channel) | `SELA/apps/web/package.json`; `SELA/apps/web/src/lib/api/client.ts`; `SELA/apps/web/src/lib/i18n/messages/parity.test.ts` | two `adapter-node` SSR apps with a BFF proxy |
| 16 | Deploy | Caddy; static web (Vercel or Caddy); `deploy/docker-compose.yml` for Dragonfly, RustFS, ClamAV profile; standalone ports API 8080, gRPC 9090, web 5173 | `SELA/deploy/docker-compose.yml`; `SELA/deploy/Caddyfile` | PM2 `ecosystem.config.cjs`; compose contains redis, nats, jaeger, prometheus (no Postgres); gateway 8000 |
| 17 | Quality gate | `scripts/check.sh` (proto drift, gofmt, vet, golangci-lint, `go test -cover`, svelte-check, vitest, real e2e; `CHECK_STAGES`), GitHub Actions jobs go/web/e2e, 80% coverage floor, `-race` in CI | `SELA/scripts/check.sh`; `SELA/.github/workflows/ci.yml` | stale CI (see 11.3); no e2e |
| 18 | Process | TDD RED/GREEN, `docs/tracking/tracker.json` + `scripts/build_tracker.py`, FSD updated together with code, `docs/aidlc/` artefacts | `SELA/docs/tracking/tracker.json`; `SELA/scripts/build_tracker.py`; `SELA/docs/aidlc` | no tracker; PRD §20 requires the PRD to be updated before code |
| 19 | Locale | Indonesian default, English supported, IDR, WIB | `SELA/CLAUDE.md` ("Language") | UI in English |

Suite decisions that apply to POS (in full in the Stery Suite plan): D1 composition (POS stays a separate deployable), D2 `STERY_MODE=standalone|suite`, D3 opaque session `stery_session` in `suite` mode, D4 entitlement with snapshot and grace period, D6 outbox without NATS, D7 one `pos` database and one `pos` app role, D8 RLS, D9 one `apps/web`, D10 password login retained with an opaque session, D11 observability `tbd`, D12 alignment order (POS = S2).

### 2.2 Module decomposition

| Module | As-is: service (separate process) | To-be: Go package in one process | As-is status | Evidence |
|---|---|---|---|---|
| 01 AUTH | `auth-service` gRPC 8081; JWT verification in the gateway | package `auth` (opaque session, RBAC, tenant resolution) + port `Identity` | Partial | `services/auth-service/cmd/server/main.go` |
| 02 PRD | `product-service` gRPC 8082 | package `product` | Partial | `services/product-service/cmd/server/main.go` |
| 03 INV | `inventory-service` gRPC 8083 | package `inventory` | Partial | `services/inventory-service/cmd/server/main.go` |
| 04 SAL | `sales-service` gRPC 8084 (calls inventory over gRPC) | package `sales` calls `inventory` as a Go interface within one transaction | Partial | `services/sales-service/cmd/server/main.go` |
| 05 SUP | `supplier-service` gRPC 8085 (calls inventory over gRPC) | package `supplier` | Partial | `services/supplier-service/cmd/server/main.go` |
| 06 CUS | `customer-service` gRPC 8086 | package `customer` | Partial | `services/customer-service/cmd/server/main.go` |
| 07 RPT | `GetSalesReport` inside `sales-service` | package `sales` (reporting) or package `report` | Partial | `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| 08 NTF | none | `Notification` port adapter (platform hub in `suite` mode; outbox worker in `standalone`) | Not started | `services/_shared/nats/streams.go` |
| 09 AI | none | no package (PRD phase 2/3) | Not started | `services/_shared/nats/streams.go` |
| Gateway | `api-gateway` chi 8000 | removed; REST routes are mounted by `apps/api` | Done | `services/api-gateway/cmd/server/main.go` |

### 2.3 Interface components

| Component | Function | Status | Evidence |
|---|---|---|---|
| REST gateway (chi) | JWT -> Tenant -> Store -> `WithRole`, rate limit 300 per minute, 4 MB body, login 20 per minute per IP | Done | `services/api-gateway/cmd/server/main.go` |
| JWT verifier + blacklist | local RS256 verification with the public key; `jti` blacklist in Redis | Done | `services/api-gateway/internal/jwt/verifier.go` |
| Gateway gRPC clients | one client per service with `x-service-token` | Done | `services/api-gateway/internal/clients/base.go` |
| Admin web (SvelteKit SSR) | read pages: dashboard, products, inventory, suppliers, customers, reports | Partial | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| Cashier web (SvelteKit SSR + partial PWA) | cashier login, POS, receipt | Partial | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| BFF proxy | `src/routes/api/[...path]/+server.ts` forwards to the gateway with the token from the cookie | Done | `apps/admin/src/routes/api/[...path]/+server.ts` |
| Shared UI package | `@pos-stery/ui` | Done | `packages/ui` |

**Go ports (target, same as Stery Suite).** Each product uses six ports with two adapters (local for `standalone`, platform for `suite`) selected by `STERY_MODE`:

```go
Identity.Verify(ctx, token) (Principal, error)
Tenant.WithTenant(ctx, tenantID, scope, fn)          // scope: tenant, tenant+store
Entitlement.Allows(ctx, org, feature) (bool, error)
Notification.Enqueue(ctx, tx, msg)
Audit.Record(ctx, event)
Events.Publish(ctx, tx, envelope)
```

Mapping to POS: `Tenant.WithTenant` replaces `WithTenantContext` (adds `app.store_id`); `Entitlement.Allows` checks features and the dimensions `pos.stores.max`, `pos.devices.max`; `Events.Publish` writes an outbox row within the business transaction.

### 2.4 Dragonfly usage

| Usage | As-is (Redis) | Target (Dragonfly) | As-is evidence |
|---|---|---|---|
| Access token blacklist | `jti` key on logout, read by gateway and auth-service | removed: opaque sessions are revoked by deleting the session hash | `services/auth-service/internal/infrastructure/redisstore/token_blacklist.go` |
| Sessions | none (stateless JWT; refresh token in Postgres) | `session:<sha256>` -> attributes (`user_id`, `tenant_id`, `store_id`, `role`) TTL `tbd` (D-POS-11); only the hash is stored | `services/auth-service/internal/domain/refresh_token.go` |
| Rate limit | in-memory `httprate` in the gateway | Dragonfly counters per address/account (login 20 per minute per IP retained) | `services/api-gateway/cmd/server/main.go` |
| Login lockout key | none | `login:fail:<email-hash>` [INFERENCE: aligned with `SELA/CLAUDE.md` auth invariants] | `services/auth-service/internal/application/commands/login.go` |
| Active cashier device list (`pos.devices.max`) | none | device registry in Postgres + `device:<id>` heartbeat in Dragonfly [INFERENCE: design does not exist yet] | `apps/cashier/src/routes/(auth)/login/+page.svelte` |

Standalone port: Dragonfly 6379. Suite profile port: POS uses Dragonfly 6382 (one instance per product, D7).

### 2.5 Events and outbox

**As-is (NATS JetStream).** Stream `POS_EVENTS`, file storage, 7-day max age, 1 replica. Envelope: `{event_type, tenant_id, store_id, occurred_at, payload}` (without `event_id`, `actor_id`, `schema_version`). There is no subscriber anywhere in the code.

| Subject | Publisher | Consumer | Status | Evidence |
|---|---|---|---|---|
| `Product.LowStock` | inventory (`UpdateStock`, quantity <= threshold; publish failure swallowed) | none | Partial | `services/inventory-service/internal/infrastructure/nats_publisher.go` |
| `Sale.Completed` | sales (best-effort after commit) | none (should be customer/loyalty) | Partial | `services/sales-service/internal/infrastructure/nats_publisher.go` |
| `Inventory.Replenished` | supplier (per received item) | none | Partial | `services/supplier-service/internal/infrastructure/nats_publisher.go` |
| `Customer.New` | customer | none | Partial | `services/customer-service/internal/infrastructure/nats_publisher.go` |
| `AI.OrderForecasted` | none (constant only) | none | Not started | `services/_shared/nats/streams.go` |

**Target (D6).** In-process domain events + a Postgres outbox row within the business transaction; a ticker worker reads the outbox with `FOR UPDATE SKIP LOCKED`; idempotent consumers with a `processed_events` table. Cross-product envelope: `{event_id UUIDv7, event_type, org_id, store_id?, actor_id, occurred_at, schema_version, payload}`; at-least-once delivery to the platform `IntegrationService/Publish`; opt-in per organisation via the entitlement feature `integration.<a>_<b>`. Subject mapping: `Product.LowStock` -> `inventory.low_stock`, `Sale.Completed` -> `sales.sale.completed`, `Inventory.Replenished` -> `supplier.stock.received`, `Customer.New` -> `customer.created` (final names `tbd` in S2). First in-process subscribers: loyalty on `sales.sale.completed` (idempotent per `sale_id`) and the low-stock notification.

### 2.6 Code structure and contracts

**As-is.** Monorepo: `services/{auth,product,inventory,sales,supplier,customer}-service` (each with `go.mod`, `cmd/server`, `internal/{domain,application,infrastructure,interfaces/grpc}`, `migrations`), `services/api-gateway`, `services/_shared` (config, database, middleware, nats, redis, observability, server, errors), `proto/pos/<svc>/v1/*.proto`, `gen/go`, `apps/{admin,cashier}`, `packages/ui`, `deploy/docker`, `scripts`, `go.work`. Layered domain/application/infrastructure/interfaces architecture per service (except auth-service, which uses sqlc; the others use pgx).

| Contract | Location | Status | Evidence |
|---|---|---|---|
| Proto for 6 services, 28 RPCs | `proto/pos/*/v1/*.proto` | Done | `proto/pos/customer/v1/customer.proto` |
| Generated code | `gen/go/pos/*` (committed) | Done | `gen/go` |
| Migrations | one `000001_*` per service (auth: 000001-000003) | Partial | `services/auth-service/migrations/000003_add_login_tenant_resolver.up.sql` |
| Migration validation script | only counts up/down file pairs | Partial | `scripts/migrate-validate.sh` |

**Target.** `apps/api` (one Go module, `cmd/api`, `cmd/migrate`, `services/<name>`, `api/proto/v1`, `gen/go`, embedded goose migrations), `apps/web` (one SPA with route groups `(admin)` and `(cashier)`), `deploy/`, `scripts/check.sh`, `docs/tracking/tracker.json`, `AGENTS.md`; shared code is imported from `kit` (repo `stery-suite`). Target path details are in 2.7.

### 2.7 As-is -> to-be alignment delta

Reuse: the `domain` and `application` layers are kept (business logic and unit tests), `infrastructure` and `interfaces` are rewritten. Effort estimates S/M/L [INFERENCE] (S = days, M = weeks, L = several weeks; not validated).

| as-is -> to-be | modules affected | reuse (domain/application kept, infrastructure/interfaces rewritten) | effort S/M/L [INFERENCE] |
|---|---|---|---|
| Go `chi` gateway (`services/api-gateway`) -> REST `/api/v1` on stdlib ServeMux in `apps/api/cmd/api` with envelope `{success,data,error}` via `kit/httpx` | all | handlers remapped; Tenant/Store/Role middleware becomes `RequirePermission` | M |
| Six gRPC processes + inter-service saga -> Go packages `apps/api/services/<name>` in one process, between modules via Go interfaces | 03, 04, 05 | domain and commands (`create_sale.go`, `receive_stock.go`) kept; inter-service gRPC clients removed | L |
| Fail-open gRPC `X-Service-Token` (`services/_shared/middleware/grpc_interceptors.go`) -> deny-by-default interceptor with a public allowlist; gRPC only for machine clients | 01 | none (rewritten) | S |
| JWT RS256 (`tid`, `sid`, `role`) + refresh token -> opaque session cookie `stery_session`/POS session cookie with attribute `store_id`, hash in Dragonfly (D3, D10); bcrypt retained (`services/auth-service/internal/application/commands/login.go`) | 01 | domain `User`/`Store`/`Tenant`, tenant resolution `auth.resolve_login_tenant` kept; rotation/reuse detection replaced by session expiry | M |
| `golang-migrate` -> embedded goose v3.26.0 + `cmd/migrate`; consolidate per-service migrations into one series | all | table SQL kept; add migrations for fixes (stock CHECK, `WITH CHECK`, new columns) | M |
| sqlc (only `auth-service`) and pgx 5.6.0 -> pgx/v5 raw SQL without sqlc | 01 | sqlc queries rewritten as raw SQL | S |
| zap + viper -> `log/slog` + fail-fast env configuration (`kit/config`) | all | none | S |
| Redis -> Dragonfly (go-redis/v9) | 01 | go-redis client retained | S |
| NATS JetStream -> Postgres outbox + ticker worker + `processed_events` (D6) | 03, 04, 05, 06 | publishers (`nats_publisher.go`) replaced by `Events.Publish`; payload contracts kept | M |
| `WithTenantContext` (`SET LOCAL`, unverified) -> `kit/tenantdb.WithTenant` (`set_config(..., true)`, tenant + `app.store_id`) + cross-tenant tests | all | none | M |
| RLS: add `WITH CHECK` to 7 policies; add `app.store_id` policies on store tables; roles `pos_admin`/`pos_app` -> separate migration role + a single NOBYPASSRLS app role `pos` | all | schema kept | S |
| Sale: gRPC saga -> one DB transaction with idempotency key, server-side price, conditional stock decrement, stock movement ledger, outbox (7.1) | 04, 03 | validation and total calculation kept | M |
| Loyalty: no caller -> called in-process after commit through an outbox consumer, idempotent per `sale_id` | 06, 04 | `add_loyalty_points.go` kept | S |
| PO receiving: non-atomic, no partial -> one transaction, received quantity per item, partial status (7.2) | 05 | PO entity kept | M |
| Money `float64` in Go -> exact representation (integer rupiah units in Go; `NUMERIC(12,2)` columns stay) | 04, 05, 07 | types and tests adjusted | S |
| 14 REST routes -> full parity with module operations (6.2), `/api/v1` prefix, `PUT` product etc. | all | none | M |
| Admin + cashier SSR (`adapter-node`, BFF) -> one static SPA `apps/web`, route groups `(admin)` and `(cashier)`, `apiFetch`, i18n `{id,en}` | 10 | `packages/ui` and page components moved | L |
| Cashier PWA only caches assets -> IndexedDB offline queue, service worker scope `/cashier/` (7.3) | 04 | none | M |
| Dashboard: synthetic data -> real data with product names and payment mix | 07 | page kept as the skeleton | S |
| PM2 `ecosystem.config.cjs` + compose (redis, nats, jaeger, prometheus) -> Caddy + `deploy/docker-compose.yml` (Dragonfly, RustFS, ClamAV profile) | deploy | none | S |
| Go 1.24/1.25 and Postgres 16 -> Go >= 1.25.1 and PostgreSQL 18; CI -> `scripts/check.sh` + jobs go/web/e2e, `-race`, 80% floor | CI | unit tests kept | M |
| Tracked secrets (`ecosystem.config.cjs`, `deploy/docker/init-postgres.sql`) -> `.env.example` without secret values; rotation and git history purge | deploy, 01 | none | S |
| OpenTelemetry/Prometheus (dependencies only) -> D11 `tbd`; baseline `slog` + `/healthz` | all | none | S |
| `suite` mode adapters for six ports + entitlement `pos.stores.max`, `pos.devices.max` (S4) | 01, 05, 08 | ports introduced in S2 with local adapters | M |

**S2 workstream order** (each item closes one or more BRD 12.1 gate items): WS-1 security and hygiene fixes (G5, G7, G8 partial; can be done earlier), WS-2 `apps/api` foundation and database (goose, PG18, tenantdb, G4, G6), WS-3 move modules 01-03 and 05-07, WS-4 transactional sale and loyalty (G1), WS-5 full REST surface (G2), WS-6 single `apps/web` and offline, WS-7 CI `check.sh` (G3), WS-8 PPN/receipt decision (G8).

---

## 3. Actors and Access Rights Model

### 3.1 Actors

| Actor | Description | Login | Evidence |
|---|---|---|---|
| Tenant admin (`admin`) | manages products, stock, suppliers, customers, reports, prices | email + password | `services/auth-service/internal/domain/user.go` |
| Cashier (`cashier`) | sells; bound to one store; login requires `store_id` | email + password + `store_id` | `services/auth-service/internal/application/commands/login.go` |
| Stock manager (`stock_manager`) | inventory, PO, stock receiving | email + password | `services/api-gateway/cmd/server/main.go` |
| Store customer | recorded for loyalty; does not log in | none | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| Platform (`suite` mode) | issues sessions, entitlement, notifications, audit | service to service (HMAC) | does not exist in POS yet (Not started) |
| Operator (provisioning) | creates the tenant, stores, first user | no path yet (Not started) | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |

Code note: `User.PrimaryRole()` only takes the user's first role, so multiple roles per user are not effectively supported yet (Partial).

### 3.2 As-is access rights matrix (enforced in the gateway `WithRole`)

| Operation (route) | admin | cashier | stock_manager |
|---|---|---|---|
| `POST /api/logout` | yes | yes | no |
| `GET /api/products` | yes | yes | no |
| `POST /api/products`, `PUT /api/products/{id}` | yes | no | no |
| `GET /api/inventory`, `POST /api/inventory` | yes | no | yes |
| `POST /api/sales` | no | yes | no |
| `GET /api/sales/report` | yes | no | no |
| `GET /api/suppliers`, `POST /api/purchase-orders` | yes | no | yes |
| `POST /api/suppliers` | yes | no | no |
| `GET /api/customers`, `POST /api/customers` | yes | yes | no |

Finding: `stock_manager` cannot see products (`GET /api/products`) although it needs to pick products for a PO, and cannot call `POST /api/logout`; `admin` cannot create a sale. Evidence: `services/api-gateway/cmd/server/main.go`. Roles are stored in `auth.roles` (a roles table only; there is no `permissions` table although PRD §12 implies one). Target: a permission catalogue and `RequirePermission(code)` (FR-POS-01.9).

### 3.3 Target access rights model

- The fixed roles `admin`, `cashier`, `stock_manager` are mapped to permissions (`products:write`, `sales:create`, `reports:read`, etc.; final catalogue `tbd` in S2) in the tables `roles`, `permissions`, `actions`, `role_actions`.
- Session attributes: `user_id`, `tenant_id`, `store_id` (mandatory for cashiers), `role`; an admin selects the active store through a selector (this changes the session attribute `store_id`, not a free `X-Store-ID` header).
- Tenant-level routes (products, suppliers, customers) do not require a store; store-level routes (inventory, sales, PO) do.
- `suite` mode: the session and base permissions come from the platform (`Identity.Verify`); the product role is mapped from organisation membership (details `tbd`).

---

## 4. Functional Specification per Module

Each module: summary, FR table (`ID | Requirement | Prioritas (M/S/C) | Status | Bukti`), then business rules, state machine (if any), and events.

### 4.01 AUTH (identity, tenant, store, RBAC)

bcrypt password login with tenant resolution from the email (SECURITY DEFINER function `auth.resolve_login_tenant`; ambiguous emails are rejected), RS256 JWT with a 10-minute lifetime and claims `tid`, `sid`, `role`, `email`; a 30-day opaque refresh token with a SHA-256 hash, rotation within one family and family revocation when reuse is detected. The gateway verifies the JWT locally (public key) and checks the `jti` blacklist.

| ID | Requirement | Priority | Status | Evidence |
|---|---|---|---|---|
| FR-POS-01.1 | Email + password login (bcrypt); the tenant is resolved from the email via `auth.resolve_login_tenant`; ambiguous email fails closed | M | Done | `services/auth-service/internal/application/commands/login.go`; `services/auth-service/migrations/000003_add_login_tenant_resolver.up.sql` |
| FR-POS-01.2 | Cashier login must include a `store_id` equal to the assigned store; the token carries `sid` | M | Done | `services/auth-service/internal/application/commands/login.go` |
| FR-POS-01.3 | Admin selects the active store; the gateway validates `X-Store-ID` via `ValidateStore`; the admin store selector UI exists | M | Partial | `services/api-gateway/internal/middleware/store.go`; `apps/admin/src/lib/api.ts` |
| FR-POS-01.4 | Refresh token with rotation and family revocation on reuse; available over REST and used by the web | M | Partial | `services/auth-service/internal/application/commands/refresh.go`; `services/auth-service/internal/application/commands/refresh_test.go` |
| FR-POS-01.5 | Logout (`jti` blacklist + revoke refresh), logout of all devices, session list; available to all roles | S | Partial | `services/auth-service/internal/application/commands/logout.go`; `services/auth-service/internal/application/commands/logout_all.go`; `services/auth-service/internal/application/commands/sessions.go` |
| FR-POS-01.6 | Enforcement of the roles `admin`/`cashier`/`stock_manager` in the API (`WithRole`) | M | Done | `services/api-gateway/internal/middleware/role.go`; `services/api-gateway/internal/middleware/role_test.go` |
| FR-POS-01.7 | Login rate limit 20 per minute per IP and 300 per minute global; maximum request body 4 MB | M | Done | `services/api-gateway/cmd/server/main.go` |
| FR-POS-01.8 | Provisioning: create the tenant, stores, first user, and reset passwords without manual SQL | M | Not started | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |
| FR-POS-01.9 | Permission catalogue (`permissions`, `actions`, `role_actions`) and `RequirePermission(code)` replace the per-route role list | S | Not started | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| FR-POS-01.10 | Opaque session (HttpOnly Secure cookie, hash in Dragonfly) with attribute `store_id` replaces JWT; no tokens in browser storage | M | Not started | `services/auth-service/internal/infrastructure/jwt/signer.go` |
| FR-POS-01.11 | HttpOnly session cookie in the web apps, role enforcement in `hooks.server.ts` | S | Partial | `apps/admin/src/hooks.server.ts`; `apps/admin/src/lib/server/session.ts` |
| FR-POS-01.12 | Mode `STERY_MODE=standalone|suite` selects the adapters of the six ports; default `standalone` | M | Not started | `services/api-gateway/internal/config/config.go` |
| FR-POS-01.13 | Enforcement of entitlement `pos.stores.max` (store creation) and `pos.devices.max` (cashier device login) with snapshot and safe degradation | M | Not started | `services/auth-service/internal/infrastructure/postgres/store_repository.go` |

**Business rules**

- The login email must resolve to exactly one tenant; if ambiguous, login is rejected without hints (as-is, `auth.resolve_login_tenant`).
- A cashier may only log in to the assigned store; an admin may select any store belonging to their tenant (validated by `ValidateStore`).
- The tables `auth.tenants` and `auth.roles` have no RLS; `auth.refresh_tokens` deliberately has no RLS (read before the tenant context exists). In the target, tables without RLS MUST be registered and restricted (G4).
- One user has one effective role (`PrimaryRole` = first role).
- `suite` mode: `standalone` behaviour does not change; the session and entitlement come from the platform.

**State machine (refresh token)**

| From | Event | To |
|---|---|---|
| active | refresh used | revoked, new token in the same family (`replaced_by`) |
| revoked | used again | whole family revoked |
| active | logout / logout-all | revoked |
| active | after 30 days | expired |

Target: no refresh token; opaque session with a sliding lifetime (TTL `tbd`).

**Events.** Produced: none (as-is); target `auth.user.logged_in` for audit (`tbd`). Consumed: none.

### 4.02 PRD (product catalogue)

| ID | Requirement | Priority | Status | Evidence |
|---|---|---|---|---|
| FR-POS-02.1 | Create a product (name, `sku` unique per tenant, `barcode`, `base_price`, `sale_price`, `unit`, category) | M | Done | `services/product-service/internal/application/commands/create_product.go`; `services/product-service/internal/application/commands/create_product_test.go` |
| FR-POS-02.2 | Update a product (name, price, active status) without accidentally changing `is_active` | M | Partial | `services/product-service/internal/application/commands/update_product.go` |
| FR-POS-02.3 | Fetch a product by id over REST | S | Partial | `services/product-service/internal/interfaces/grpc/product_service.go` |
| FR-POS-02.4 | Search products by name, SKU and barcode with pagination (limit/offset passed through) | M | Partial | `services/product-service/internal/infrastructure/postgres/product_repository.go`; `services/product-service/internal/application/queries/search_products.go` |
| FR-POS-02.5 | Auditable price change history (who, when, old and new value) | S | Not started | `services/product-service/internal/application/commands/update_product.go` |
| FR-POS-02.6 | Category management (create, update, list) | S | Not started | `services/product-service/migrations/000001_create_product_schema.up.sql` |
| FR-POS-02.7 | Admin form to create and update products | M | Not started | `apps/admin/src/routes/(app)/products/+page.svelte` |

**Business rules**

- `UNIQUE(tenant_id, sku)`; products are soft deleted (`deleted_at`).
- As-is search: `name ILIKE` or an exact `barcode` match; SKU is not searched (the UI placeholder "name or SKU" is misleading). The gateway only forwards the `q` parameter.
- The `UpdateProduct` handler sets `is_active` as requested (false when omitted), so a partial update deactivates the product (as-is, read statically).
- Price: `base_price` and `sale_price` are optional (`NUMERIC(12,2)`); a sale uses the `unit_price` sent by the client (fixed in FR-POS-04.4).
- Price changes only by `admin`.

**State machine.** `aktif` <-> `nonaktif` (`is_active`); `dihapus` (soft delete). No other transitions.

**Events.** Produced: none (target `product.price_changed` for audit, `tbd`). Consumed: none.

### 4.03 INV (inventory)

| ID | Requirement | Priority | Status | Evidence |
|---|---|---|---|---|
| FR-POS-03.1 | Add a stock item per store with an initial quantity (`UNIQUE(tenant_id, store_id, product_id)`) | M | Done | `services/inventory-service/internal/application/commands/add_item.go` |
| FR-POS-03.2 | Change stock atomically and never negative, including under concurrency (conditional UPDATE + `CHECK (quantity >= 0)`) | M | Partial | `services/inventory-service/internal/application/commands/update_stock.go`; `services/inventory-service/internal/infrastructure/postgres/stock_repository.go` |
| FR-POS-03.3 | Stock movement ledger: every change records the delta, reason, type and reference id (sale, receipt, adjustment), actor | M | Not started | `services/inventory-service/internal/application/commands/update_stock.go` |
| FR-POS-03.4 | Set a stock threshold per product per store; list and get item report `min_quantity` and `is_low_stock` correctly | M | Partial | `services/inventory-service/internal/application/commands/set_threshold.go`; `services/inventory-service/internal/interfaces/grpc/inventory_service.go` |
| FR-POS-03.5 | Low-stock event published when the quantity crosses the threshold, consumed by notifications | M | Partial | `services/inventory-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-03.6 | List items per store with a low-stock filter; get item over REST | M | Partial | `services/inventory-service/internal/application/queries/list_items.go`; `services/inventory-service/internal/application/queries/get_item.go` |
| FR-POS-03.7 | Manual stock adjustment and threshold setting available over REST and the admin UI | M | Not started | `services/api-gateway/internal/handlers/inventory.go` |

**Business rules**

- Stock MUST NOT be negative (PRD §13). As-is: `update_stock.go` reads the item (`GetItem`) and then executes `UPDATE ... quantity = quantity + $1` without a `quantity + $1 >= 0` guard and without a table `CHECK`, so two simultaneous sales can produce negative stock (TOCTOU; read statically, not tested at runtime) [INFERENCE].
- The `reason` parameter and reference are ignored by the repository (no ledger).
- `AddItem`, `ListItems`, `GetItem` return `min_quantity = 0` and `is_low_stock = false` because the proto conversion uses fixed values; only `UpdateStock` fills `LowStockAlert` correctly (`services/inventory-service/internal/interfaces/grpc/inventory_service.go`).
- The low-stock event is published when quantity <= threshold; publish failures are swallowed.
- Target: stock reduction by a sale inside the sale transaction (7.1); every change writes a `stock_movements` row.

**State machine.** No status; quantity increases/decreases with a reason: `RECEIPT`, `SALE`, `ADJUSTMENT` (the as-is reason is sent but discarded).

**Events.** Produced: `Product.LowStock` (target `inventory.low_stock`). Consumed: target `sales.sale.completed` is not needed (reduction happens inside the transaction); `supplier.stock.received` in-process.

### 4.04 SAL (sale, discount, receipt)

| ID | Requirement | Priority | Status | Evidence |
|---|---|---|---|---|
| FR-POS-04.1 | Create a sale with items (product, quantity), validate input and compute subtotal, discount, total | M | Partial | `services/sales-service/internal/application/commands/create_sale.go`; `services/sales-service/internal/application/commands/create_sale_test.go` |
| FR-POS-04.2 | Stock reduction and storage of the sale + receipt in one database transaction | M | Partial | `services/sales-service/internal/application/commands/create_sale.go`; `services/sales-service/internal/infrastructure/grpc_clients/inventory_client.go` |
| FR-POS-04.3 | Idempotent sale creation: a client-supplied sale ID (`client_sale_id`) unique per tenant and store; a retry returns the same sale | M | Not started | `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.4 | Unit price looked up on the server from the catalogue; discount limited by role authority; a negative total must be rejected | M | Not started | `services/api-gateway/internal/handlers/sales.go`; `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.5 | PPN/tax is computed and stored on the sale and receipt according to decision D-POS-01 | M | Not started | `services/sales-service/internal/domain/sale.go` |
| FR-POS-04.6 | Payment method (cash, QRIS, debit as record-keeping) stored on the sale according to D-POS-04 | S | Not started | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| FR-POS-04.7 | Receipt stored as a JSONB snapshot unique per sale (items, total, cashier, store, customer, tax, payment method) | M | Partial | `services/sales-service/internal/infrastructure/postgres/sale_repository.go`; `services/sales-service/migrations/000001_create_sales_schema.up.sql` |
| FR-POS-04.8 | Fetch a sale and receipt by id over REST (`GetSale`) | M | Partial | `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| FR-POS-04.9 | `Sale.Completed` event written inside the transaction (outbox) and consumed by loyalty | M | Partial | `services/sales-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-04.10 | Cashier interface: search/scan, cart, checkout, receipt display and print, receipt history | M | Partial | `apps/cashier/src/routes/(app)/pos/+page.svelte`; `apps/cashier/src/routes/(app)/receipts/+page.svelte` |
| FR-POS-04.11 | Void and return of sales according to policy D-POS-06 | C | Not started | `services/sales-service/internal/domain/sale.go` |
| FR-POS-04.12 | Offline sale queue on the cashier device with automatic sync without duplication (7.3) | S | Not started | `apps/cashier/src/service-worker.ts` |

**Business rules**

- A sale MUST decrease stock atomically and create a receipt (PRD §13). As-is: sequence `validasi -> hitung -> inventory.DeductStock per item (gRPC) -> saleRepo.Create (sale + item + struk satu transaksi) -> PublishSaleCompleted (best-effort)`; if `Create` fails, `compensate()` calls `RestoreStock` per item and only logs a warning if restoration fails. The stock deduction commits separately from the sale (`services/sales-service/internal/application/commands/create_sale.go`).
- As-is defects: no idempotency key; `UnitPrice` and `Discount` come from the caller; `total` is clamped at 0; money is `float64` in Go while the column is `NUMERIC(12,2)`; no PPN/payment method/customer in the snapshot; no loyalty call; no discount authority check; `completed_at` and the report day boundary use the DB session time zone, not WIB.
- Gateway: `createSaleBody` only contains `customer_id`, `items[{product_id, quantity, unit_price}]`, `discount_amount`, `notes`; the `payment_method` from the cashier UI is dropped (`services/api-gateway/internal/handlers/sales.go`).
- Only the `cashier` role can create a sale (gateway).
- One receipt per sale (`sales.receipts.sale_id UNIQUE`).
- Target: see 7.1; PPN and payment method `tbd` (D-POS-01, D-POS-04); discount authority `tbd` (D-POS-07).

**State machine (sale).** As-is there is only one `status` value: `COMPLETED` (column default; no other value in the code). Target (after decision D-POS-06): `COMPLETED -> VOIDED` (admin), `COMPLETED -> RETURNED` (partial/full); the offline queue adds the client states `QUEUED -> SYNCING -> SYNCED | FAILED`.

**Events.** Produced: `Sale.Completed` (target `sales.sale.completed`), `Product.LowStock` indirectly. Consumed: target loyalty (06) and hypothesis INT-3 (platform).

### 4.05 SUP (supplier and PO)

| ID | Requirement | Priority | Status | Evidence |
|---|---|---|---|---|
| FR-POS-05.1 | Add a supplier (name, contact, phone, email, address) | M | Done | `services/supplier-service/internal/application/commands/add_supplier.go` |
| FR-POS-05.2 | List suppliers per tenant (excluding soft-deleted) | M | Done | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| FR-POS-05.3 | Create a PO for one supplier with items (product, quantity, unit cost); supplier validated, products validated as belonging to the tenant | M | Partial | `services/supplier-service/internal/application/commands/create_purchase_order.go` |
| FR-POS-05.4 | List POs and get a PO over REST | S | Partial | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| FR-POS-05.5 | Receive stock against a PO: add stock for each item exactly once, atomic, idempotent | M | Partial | `services/supplier-service/internal/application/commands/receive_stock.go`; `services/supplier-service/internal/application/commands/receive_stock_test.go` |
| FR-POS-05.6 | Partial receiving (`quantity_received` per item) and PO cancellation (`CANCELLED`) | S | Not started | `services/supplier-service/internal/application/commands/receive_stock.go` |
| FR-POS-05.7 | `Inventory.Replenished` event after receiving, consumed by notifications/analytics | S | Partial | `services/supplier-service/internal/infrastructure/nats_publisher.go` |

**Business rules**

- A new PO has status `PENDING`; it MUST have at least one item with quantity > 0 and a valid `product_id`. As-is does not check that the product exists or belongs to the tenant (`create_purchase_order.go`).
- As-is receiving (`receive_stock.go`): items absent from the request are skipped; per item it calls inventory `UpdateStock` (RECEIPT) over gRPC and then publishes `Inventory.Replenished` best-effort; receiving any item marks the whole PO `RECEIVED`. A failure mid-loop leaves stock partially increased and the PO still `PENDING`, so a retry adds twice (not idempotent).
- PO status in the schema: `PENDING`, `RECEIVED`, `CANCELLED`; there is no cancel RPC.
- The `quantity_received` column exists but does not form a partial status.
- `unit_cost` is `NUMERIC(12,2)`.

**State machine (PO).**

| From | Event | To | As-is status |
|---|---|---|---|
| `PENDING` | receive goods | `RECEIVED` | Done (no partial) |
| `PENDING` | cancel | `CANCELLED` | Not started (no RPC) |
| `PENDING` | receive partially | `PARTIAL` (target) | Not started |
| `PARTIAL` | receive remainder | `RECEIVED` (target) | Not started |

**Events.** Produced: `Inventory.Replenished` (target `supplier.stock.received`). Consumed: none.

### 4.06 CUS (customer and loyalty)

| ID | Requirement | Priority | Status | Evidence |
|---|---|---|---|---|
| FR-POS-06.1 | Create a customer (name, phone, email) | M | Done | `services/customer-service/internal/application/commands/create_customer.go`; `services/customer-service/internal/application/commands/create_customer_test.go` |
| FR-POS-06.2 | Fetch and list customers over REST | S | Partial | `services/customer-service/internal/interfaces/grpc/customer_service.go`; `services/customer-service/internal/application/queries/get_customer.go` |
| FR-POS-06.3 | Loyalty points are added automatically after a sale with a customer is committed, exactly once per `sale_id` | M | Not started | `services/customer-service/internal/application/commands/add_loyalty_points.go`; `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| FR-POS-06.4 | Earning rule (rate per spend value) and point redemption according to D-POS-05 | S | Not started | `services/customer-service/internal/application/commands/add_loyalty_points.go` |
| FR-POS-06.5 | `Customer.New` event published after a customer is created | C | Partial | `services/customer-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-06.6 | Cashier attaches a customer to the sale and receipt in the UI | S | Not started | `apps/cashier/src/routes/(app)/pos/+page.svelte` |

**Business rules**

- Points only increase after a successful sale (PRD §13). **As-is not met:** `AddLoyaltyPoints` (`customer_service.go`, comment "called by sales-service after Sale.Completed") has no caller; sales-service does not call it and there is no NATS subscriber anywhere in the code.
- The loyalty handler reads the total and then inserts a row in a separate transaction (race); `loyalty_points.sale_id` is not `UNIQUE`, so it is not idempotent (read statically) [INFERENCE].
- The earning rule is undefined (the caller supplies the number of points); there is no redemption, expiry or tier.
- Customer data (name, phone, email) is personal data, stored in plain text (section 8).
- Target: an in-process outbox consumer applies points in its own transaction, key `UNIQUE(tenant_id, sale_id)`.

**State machine.** Customer: `aktif` / `nonaktif` / `dihapus` (soft delete). Points: append-only ledger (`points`, `total_points`).

**Events.** Produced: `Customer.New` (target `customer.created`). Consumed: target `sales.sale.completed`.

### 4.07 RPT (reports)

| ID | Requirement | Priority | Status | Evidence |
|---|---|---|---|---|
| FR-POS-07.1 | Sales report per date range: total revenue and transaction count, per store and per tenant | M | Partial | `services/sales-service/internal/infrastructure/postgres/sale_repository.go`; `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| FR-POS-07.2 | Top products with product name and units sold | M | Partial | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.3 | Payment method mix computed from real sales data | S | Not started | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| FR-POS-07.4 | The dashboard shows only real data; field names match the API contract | M | Partial | `apps/admin/src/routes/(app)/dashboard/+page.svelte`; `apps/admin/src/routes/(app)/reports/+page.svelte` |
| FR-POS-07.5 | Report day boundaries follow the WIB time zone (or the store's, D-POS-14) | S | Not started | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.6 | Tenant-level report scope selectable by admin without a store context | S | Partial | `services/api-gateway/internal/handlers/sales.go` |

**Business rules**

- All report queries respect tenant and store scope (PRD §13); the repository queries support both store and tenant scope, but the gateway `Store` middleware requires a store context for all routes so tenant scope cannot be used over REST (read statically) [INFERENCE].
- `product_name` in top products is always `''`: the SQL writes `'' AS product_name` for the store and tenant queries without a join to products (`sale_repository.go`).
- Contract mismatch: the dashboard UI expects `total_sales`, `top_products[].name`, `.quantity`; the backend returns `transaction_count`, `product_id`, `product_name`, `units_sold` (`gen/go/pos/sales/v1/sales.pb.go`).
- The dashboard fabricates data: payment mix (falls back to 55/30/15 from `total_revenue`), synthetic sparkline bars (seven fixed-weight bars), and a default cash float (`opening_float` 500.000) (`apps/admin/src/routes/(app)/dashboard/+page.svelte`, lines 44-68).
- `payment_method` is not stored, so the payment mix cannot be computed.

**Events.** Produced: none. Consumed: none (target: reports read directly from the sales tables).

### 4.08 NTF (notification): Not started

There is no notification service (PRD §10.2 lists `NotificationService`, §14.3 requires the admin to receive low-stock notifications). The `Product.LowStock` event is published but has no consumer (Not started; `services/_shared/nats/streams.go`). The `notifications` table in PRD §12 is not in the migrations.

Reason for deferral: PRD §18 places notifications in Phase 2, and `suite` mode delegates delivery to the platform notification hub through the port `Notification.Enqueue(ctx, tx, msg)`. In `standalone` mode, the local adapter writes `notifications`/outbox rows that a ticker worker delivers (channel `tbd`). There are no POS FRs in this document; business requirement BR-POS-17 maps to FR-POS-03.5 and this module. When it is built, FRs will be added in an FSD revision.

### 4.09 AI (demand forecasting): Not started

There is no AI service (`ai-service` does not exist); the subject `AI.OrderForecasted` is only a constant without a publisher (`services/_shared/nats/streams.go`). `POST /api/ai/forecast-demand` in PRD §11.2 does not exist in the gateway.

Reason for deferral: PRD §18 places forecasting and recommendations in Phase 2 and dynamic pricing, image recognition and chat in Phase 3 (a correction to the assumption that all AI = Phase 3, see 11.3); its prerequisite is a reliable sales history (section 7.1) and the stock ledger (FR-POS-03.3). There are no POS FRs in this document; BR-POS-18 is marked Could.

### 4.SEC Cross-module security

| ID | Requirement | Priority | Status | Evidence |
|---|---|---|---|---|
| FR-POS-SEC.1 | RLS `ENABLE` + `FORCE` on all tenant tables with a `tenant_isolation` policy | M | Done | `services/sales-service/migrations/000001_create_sales_schema.up.sql`; `services/inventory-service/migrations/000001_create_inventory_schema.up.sql` |
| FR-POS-SEC.2 | Every RLS policy has `WITH CHECK` (7 of 15 do not yet) | M | Partial | `services/customer-service/migrations/000001_create_customer_schema.up.sql`; `services/supplier-service/migrations/000001_create_supplier_schema.up.sql` |
| FR-POS-SEC.3 | Store scope enforced in RLS via `app.store_id` for tables with a `store_id` column | S | Not started | `services/_shared/database/tenant_context.go` |
| FR-POS-SEC.4 | The per-transaction tenant context works on real PostgreSQL and is proven by cross-tenant integration tests | M | Pending Verification | `services/_shared/database/tenant_context.go` |
| FR-POS-SEC.5 | Deny-by-default service-to-service authentication (no token = rejected), no gRPC reflection in production | M | Partial | `services/_shared/middleware/grpc_interceptors.go` |
| FR-POS-SEC.6 | No secrets in git-tracked files; secrets rotated and purged from history | M | Not started | `ecosystem.config.cjs`; `deploy/docker/init-postgres.sql` |
| FR-POS-SEC.7 | Append-only audit trail for sensitive actions (login, price change, stock adjustment, void, users) | S | Not started | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| FR-POS-SEC.8 | Customer personal data (name, phone, email) is protected according to the UU PDP decision (`kit/piicrypt` encryption + blind index if decided) | S | Not started | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| FR-POS-SEC.9 | Body size limit and rate limit on the API | S | Done | `services/api-gateway/cmd/server/main.go` |
| FR-POS-SEC.10 | Web session only in HttpOnly cookies; no token in `localStorage` | S | Partial | `apps/admin/src/lib/auth.ts`; `apps/admin/src/lib/server/session.ts` |

**Business rules**

- All business data access happens inside a transaction with tenant context; only the `WithTenantContext` mechanism exists: `SET LOCAL app.tenant_id = $1; SET LOCAL app.store_id = $2` (`services/_shared/database/tenant_context.go`). PostgreSQL does not accept bound parameters in `SET` and rejects multiple commands in one prepared statement, so this pattern probably fails on a real connection [INFERENCE]; there are no integration tests, so the status is Pending Verification and the target uses `set_config(..., true)`.
- No RLS policy references `app.store_id`; store scope exists only in the application layer (contradicts README/CLAUDE.md, 11.3).
- Seven policies without `WITH CHECK`: `customer.loyalty_points`, `inventory.stock_thresholds`, `product.categories`, `sales.sale_items`, `sales.receipts`, `supplier.purchase_orders`, `supplier.purchase_order_items`. [INFERENCE] PostgreSQL uses `USING` as the check for `FOR ALL` policies when `WITH CHECK` is omitted, so the gap is mainly explicitness, not a proven leak.
- `auth.audit_logs` exists but no code writes to it and there is no append-only trigger.
- `server.New` registers gRPC reflection (as-is).

**State machine.** Not applicable.

**Events.** Produced: target `audit` via `Audit.Record`. Consumed: none.

---

## 5. Data Model

As-is schemas: `auth`, `product`, `inventory`, `sales`, `supplier`, `customer`. As-is database roles `pos_admin` (BYPASSRLS, used by migrations) and `pos_app` (`services/auth-service/migrations/000001_create_auth_schema.up.sql`; roles are created in `deploy/docker/init-postgres.sql`). All primary keys are UUID with `gen_random_uuid()` (target `uuidv7()`). PII column classes: `PII` = personal data, `Rahasia` = credentials, `-` = not PII.

### 5.1 As-is entities

| Schema.Table | Key columns | RLS | PII class |
|---|---|---|---|
| `auth.tenants` | `id`, `name`, `slug` (UNIQUE), `is_active`, `deleted_at` | no RLS (migration comment: access controlled by the application layer through the role `pos_superadmin`; that role is only mentioned in comments, not defined in `services`, `deploy`, or `scripts`) | - |
| `auth.stores` | `id`, `tenant_id`, `name`, `address`, `phone`, `is_active`, `deleted_at` | `ENABLE`+`FORCE`, `tenant_isolation` with `WITH CHECK` | store address/phone: business data |
| `auth.roles` | `id`, `name` (UNIQUE; seeded `admin`, `cashier`, `stock_manager`), `description` | no RLS | - |
| `auth.users` | `id`, `tenant_id`, `email`, `password_hash` (bcrypt), `name`, `is_active`, `deleted_at`; `UNIQUE(tenant_id, email)` | `ENABLE`+`FORCE` with `WITH CHECK` | email and name: PII; `password_hash`: Secret |
| `auth.user_roles` | `user_id`, `role_id`, `store_id` (NULL = all stores, non-NULL = store-bound, e.g. cashier), `tenant_id`; unique per (user, role, tenant, store) | `ENABLE`+`FORCE` with `WITH CHECK` | - |
| `auth.audit_logs` | `id`, `tenant_id`, `user_id`, `action`, `entity`, `entity_id`, `metadata JSONB`, `created_at` | no RLS; the comment says "append-only" but there is no trigger; no code writes to it | `metadata`: `tbd` |
| `auth.refresh_tokens` | `id`, `user_id`, `tenant_id`, `family_id`, `token_hash` (UNIQUE, SHA-256), `issued_at`, `expires_at`, `revoked_at`, `replaced_by`, `user_agent`, `ip_address` | no RLS (deliberate, lookup by hash before the tenant is known) | `ip_address` and `user_agent`: related PII; token hash: Secret |
| `product.categories` | `id`, `tenant_id`, `name` | `ENABLE`+`FORCE`, without `WITH CHECK` | - |
| `product.products` | `id`, `tenant_id`, `category_id`, `name`, `sku`, `barcode`, `base_price`, `sale_price`, `unit`, `is_active`, `deleted_at`; `UNIQUE(tenant_id, sku)` | `ENABLE`+`FORCE` | - |
| `inventory.stock_items` | `tenant_id`, `store_id`, `product_id`, `quantity BIGINT`; `UNIQUE(tenant_id, store_id, product_id)` | `ENABLE`+`FORCE` | - |
| `inventory.stock_thresholds` | `tenant_id`, `store_id`, `product_id`, `min_quantity` | `ENABLE`+`FORCE`, without `WITH CHECK` | - |
| `sales.sales` | `id`, `tenant_id`, `store_id`, `cashier_id`, `customer_id`, `subtotal`, `discount_amount`, `total`, `status` (default `COMPLETED`), `notes`, `completed_at` | `ENABLE`+`FORCE` | - (`customer_id` is a reference) |
| `sales.sale_items` | `sale_id`, `tenant_id`, `product_id`, `quantity INT`, `unit_price`, `discount`, `line_total` | `ENABLE`+`FORCE`, without `WITH CHECK` | - |
| `sales.receipts` | `sale_id UNIQUE`, `tenant_id`, `store_id`, `snapshot JSONB`, `issued_at` | `ENABLE`+`FORCE`, without `WITH CHECK` | the snapshot may contain the cashier name `tbd` |
| `supplier.suppliers` | `id`, `tenant_id`, `name`, `contact`, `phone`, `email`, `address`, `deleted_at` | `ENABLE`+`FORCE` | business PII (supplier contact) |
| `supplier.purchase_orders` | `id`, `tenant_id`, `store_id`, `supplier_id`, `status` (`PENDING`/`RECEIVED`/`CANCELLED`), `notes` | `ENABLE`+`FORCE`, without `WITH CHECK` | - |
| `supplier.purchase_order_items` | `purchase_order_id`, `tenant_id`, `product_id`, `quantity_ordered`, `quantity_received`, `unit_cost` | `ENABLE`+`FORCE`, without `WITH CHECK` | - |
| `customer.customers` | `id`, `tenant_id`, `name`, `phone`, `email`, `is_active`, `deleted_at` | `ENABLE`+`FORCE` | PII (name, phone, email) |
| `customer.loyalty_points` | `customer_id`, `tenant_id`, `points`, `total_points`, `sale_id` (not `UNIQUE`) | `ENABLE`+`FORCE`, without `WITH CHECK` | - |

PRD §12 tables that do not exist: `notifications`, `ai_forecasts`, `permissions`. Column evidence: per-service migrations in `services/*/migrations/`; function `auth.resolve_login_tenant` in `services/auth-service/migrations/000003_add_login_tenant_resolver.up.sql`.

### 5.2 Target additions

| Entity | Columns/change | Reason |
|---|---|---|
| `sales.sales` | `client_sale_id UUID`, `UNIQUE(tenant_id, store_id, client_sale_id)`, `tax_amount`, `payment_method`, `tax_basis` (`tbd`) | idempotency, PPN, payment method |
| `sales.sale_items` | `store_id`, `tax_amount` (`tbd`), foreign key to `sales.sales` within the `app.store_id` policy | store RLS |
| `inventory.stock_items` | `CHECK (quantity >= 0)` | non-negative stock |
| `inventory.stock_movements` | `id`, `tenant_id`, `store_id`, `product_id`, `delta`, `reason`, `ref_type`, `ref_id`, `actor_id`, `created_at` (append-only) | ledger |
| `customer.loyalty_points` | `UNIQUE(tenant_id, sale_id)` | idempotent |
| `outbox`, `processed_events` | `event_id`, `event_type`, `payload`, `tenant_id`, `store_id`, `created_at`, `published_at`/`processed_at` | D6 |
| `audit_logs` | `kit/audit`, triggers reject UPDATE/DELETE/TRUNCATE | FR-POS-SEC.7 |
| `auth.devices` | `id`, `tenant_id`, `store_id`, `label`, `registered_at`, `last_seen_at`, `revoked_at` | `pos.devices.max` [INFERENCE] |
| `auth.stores` | `external_ref` (suite), `org_id` (`suite` mode) | platform organisation link |
| `auth.users` | `external_subject` (`suite` mode) | platform identity link |
| all tenant tables | `WITH CHECK` on every policy, `app.store_id` policy on tables with a `store_id` column | G4 |

---

## 6. API Specification

### 6.1 As-is REST (chi gateway, 14 business routes)

As-is responses do not use the `{success,data,error}` envelope; a successful body is the proto object directly, errors are `{"message": ...}`, and validation 422 is Laravel-style `{message, errors}` (`services/api-gateway/internal/handlers/render.go`). The JWT is read from the `Authorization: Bearer` header or the `pos_token` cookie; store context comes from the `sid` claim or the `X-Store-ID` header.

| As-is route | RPC | Role | Status | Evidence |
|---|---|---|---|---|
| `POST /api/login` | `AuthService.Login` | public | Done | `services/api-gateway/internal/handlers/auth.go` |
| `POST /api/logout` | `AuthService.Logout` | admin, cashier | Done | `services/api-gateway/internal/handlers/auth.go` |
| `GET /api/products` | `ProductService.SearchProducts` (only `q`) | admin, cashier | Partial | `services/api-gateway/internal/handlers/product.go` |
| `POST /api/products` | `CreateProduct` | admin | Done | `services/api-gateway/internal/handlers/product.go` |
| `PUT /api/products/{id}` | `UpdateProduct` | admin | Done | `services/api-gateway/internal/handlers/product.go` |
| `GET /api/inventory` | `ListItems` | admin, stock_manager | Done | `services/api-gateway/internal/handlers/inventory.go` |
| `POST /api/inventory` | `AddItem` | admin, stock_manager | Done | `services/api-gateway/internal/handlers/inventory.go` |
| `POST /api/sales` | `CreateSale` | cashier | Partial | `services/api-gateway/internal/handlers/sales.go` |
| `GET /api/sales/report` | `GetSalesReport` | admin | Partial | `services/api-gateway/internal/handlers/sales.go` |
| `GET /api/suppliers` | `ListSuppliers` | admin, stock_manager | Done | `services/api-gateway/internal/handlers/supplier.go` |
| `POST /api/suppliers` | `AddSupplier` | admin | Done | `services/api-gateway/internal/handlers/supplier.go` |
| `POST /api/purchase-orders` | `CreatePurchaseOrder` | admin, stock_manager | Done | `services/api-gateway/internal/handlers/supplier.go` |
| `GET /api/customers` | `ListCustomers` | admin, cashier | Done | `services/api-gateway/internal/handlers/customer.go` |
| `POST /api/customers` | `CreateCustomer` | admin, cashier | Done | `services/api-gateway/internal/handlers/customer.go` |
| `GET /healthz`, `GET /readyz` | - | public | Done | `services/api-gateway/internal/handlers/health.go` |

The `google.api.http` annotations in the proto declare `/v1/...` routes (e.g. `GET /v1/sales/{sale_id}`), but no grpc-gateway mounts them: these routes exist only as annotations (Done as a contract, Not started as a live API; `proto/pos/sales/v1/sales.proto`). PRD §11.2 lists `POST /api/ai/forecast-demand`, which does not exist in the code.

### 6.2 REST parity with module operations (gate item G2)

28 RPCs; 14 exposed, 14 not. Target route = proto annotation with the `/api/v1` prefix (replacing `/v1`).

| Module | Operation (RPC) | As-is REST | Target route `/api/v1` | Gate G2 |
|---|---|---|---|---|
| 01 | `Login` | `POST /api/login` | `POST /api/v1/auth/login` | met (route move) |
| 01 | `Logout` | `POST /api/logout` | `POST /api/v1/auth/logout` | met (route move) |
| 01 | `Refresh` | none | removed (opaque session, FR-POS-01.10) | replaced |
| 01 | `LogoutAll` | none | `POST /api/v1/auth/logout-all` | not yet |
| 01 | `ListSessions` | none | `GET /api/v1/auth/sessions` | not yet |
| 01 | `Validate` | none (internal) | internal (`Identity.Verify`) | n/a |
| 01 | `ValidateStore` | none (used by the gateway) | internal (session attribute `store_id`) | n/a |
| 01 | tenant/store/user provisioning | none | `POST /api/v1/stores`, `POST /api/v1/users` etc. (`tbd`, D-POS-12) | not yet |
| 02 | `CreateProduct`, `UpdateProduct`, `SearchProducts` | exists (some parameters) | `POST/PUT/GET /api/v1/products` | route move |
| 02 | `GetProduct` | none | `GET /api/v1/products/{product_id}` | not yet |
| 03 | `AddItem`, `ListItems` | exists | `POST/GET /api/v1/inventory` | route move |
| 03 | `GetItem` | none | `GET /api/v1/inventory/{product_id}` | not yet |
| 03 | `UpdateStock` | none | `PATCH /api/v1/inventory/{product_id}/stock` | not yet |
| 03 | `SetThreshold` | none | `PUT /api/v1/inventory/{product_id}/threshold` | not yet |
| 04 | `CreateSale` | exists (limited fields) | `POST /api/v1/sales` | route move |
| 04 | `GetSale` | none | `GET /api/v1/sales/{sale_id}` | not yet |
| 07 | `GetSalesReport` | exists | `GET /api/v1/sales/report` | route move |
| 05 | `AddSupplier`, `ListSuppliers`, `CreatePurchaseOrder` | exists | `POST/GET /api/v1/suppliers`, `POST /api/v1/purchase-orders` | route move |
| 05 | `ListPurchaseOrders` | none | `GET /api/v1/purchase-orders` | not yet |
| 05 | `ReceiveStock` | none | `POST /api/v1/purchase-orders/{purchase_order_id}/receive` | not yet |
| 06 | `CreateCustomer`, `ListCustomers` | exists | `POST/GET /api/v1/customers` | route move |
| 06 | `GetCustomer` | none | `GET /api/v1/customers/{customer_id}` | not yet |
| 06 | `AddLoyaltyPoints` | none | not exposed: called in-process (FR-POS-06.3); points view `GET /api/v1/customers/{customer_id}` | replaced |

### 6.3 Target REST contract

- `/api/v1` prefix on stdlib ServeMux; envelope `{success, data, error}` (`kit/httpx`); internal errors keep a fixed generic message; restricted with `RequirePermission(code)`.
- Session via HttpOnly cookie; no CORS (web and API share one origin through the reverse proxy, as in SELA).
- `POST /api/v1/sales` accepts `client_sale_id` (header `Idempotency-Key` or body; final form `tbd`), `customer_id?`, `items[{product_id, quantity, discount?}]`, `payment_method` (`tbd`), `notes?`; `unit_price` is NOT accepted from the client. Response: sale + receipt.
- `limit`/`offset` pagination consistent across all lists (as-is only `q` is forwarded in product search).
- Store-level routes require the session attribute `store_id`.

### 6.4 gRPC contracts

| Service | Proto file | RPCs | Status | Evidence |
|---|---|---|---|---|
| `pos.auth.v1.AuthService` | `proto/pos/auth/v1/auth.proto` | 7: `Login`, `Validate`, `Logout`, `Refresh`, `LogoutAll`, `ListSessions`, `ValidateStore` | Done | `services/auth-service/internal/interfaces/grpc/auth_service.go` |
| `pos.product.v1.ProductService` | `proto/pos/product/v1/product.proto` | 4 | Done | `services/product-service/internal/interfaces/grpc/product_service.go` |
| `pos.inventory.v1.InventoryService` | `proto/pos/inventory/v1/inventory.proto` | 5: `AddItem`, `UpdateStock`, `ListItems`, `GetItem`, `SetThreshold` | Done | `services/inventory-service/internal/interfaces/grpc/inventory_service.go` |
| `pos.sales.v1.SalesService` | `proto/pos/sales/v1/sales.proto` | 3 | Done | `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| `pos.supplier.v1.SupplierService` | `proto/pos/supplier/v1/supplier.proto` | 5 | Done | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| `pos.customer.v1.CustomerService` | `proto/pos/customer/v1/customer.proto` | 4 | Done | `services/customer-service/internal/interfaces/grpc/customer_service.go` |

Note: `DeductStock` and `RestoreStock` are not RPCs but client wrapper methods in sales-service that call `InventoryService.UpdateStock` with a negative or positive `quantity_delta` (`services/sales-service/internal/infrastructure/grpc_clients/inventory_client.go`). Target: gRPC only for machine clients (`api/proto/v1`, `buf`, committed `gen/go`, `scripts/proto-gen.sh --check`) with a deny-by-default interceptor; between modules, Go interfaces are used.

---

## 7. Critical Process Flows

### 7.1 Sale as a single database transaction (replaces the gRPC saga)

**As-is (saga).**

1. The gateway verifies the JWT, determines `tenant_id`, `store_id`, `cashier_id`, and then calls `SalesService.CreateSale` (role `cashier`).
2. `create_sale.go` validates the input and computes subtotal, discount, total (total clamped to 0) from the price and discount sent by the client.
3. For each item, `inventory.DeductStock` is called over gRPC; a failure midway triggers `compensate()` (`RestoreStock`, only a log warning if it fails).
4. `saleRepo.Create` writes the sale, items, and receipt in one transaction.
5. `PublishSaleCompleted` to NATS, best-effort.
6. Loyalty is not called. There is no idempotency key.

Possible failures: the process dies between steps 3 and 4 (stock reduced without a sale); a retry after a timeout (duplicate sale or stock deducted twice); `RestoreStock` fails (stock lost).

**Target.**

1. The client sends `POST /api/v1/sales` with `client_sale_id` (UUID created on the device), items (without price), `customer_id?`, `payment_method?`.
2. `kit/tenantdb.WithTenant(ctx, tenantID, scope tenant+store, fn)` opens a transaction and sets `app.tenant_id` and `app.store_id` via `set_config(..., true)`.
3. Inside the transaction: `INSERT ... ON CONFLICT (tenant_id, store_id, client_sale_id) DO NOTHING`; on conflict, return the existing sale (idempotent).
4. The unit price is read from `product.products`; the discount is validated against role authority (D-POS-07); PPN is computed (D-POS-01); the total is computed with an exact money representation.
5. For each item: `UPDATE inventory.stock_items SET quantity = quantity - $q WHERE ... AND quantity - $q >= 0`; zero rows = insufficient stock -> the transaction is aborted with a 409 error; write `stock_movements` (`reason = SALE`, `ref_id = sale_id`).
6. Write `sales.sales`, `sales.sale_items`, `sales.receipts` (snapshot).
7. Write the outbox row `sales.sale.completed` (and `inventory.low_stock` if the threshold is crossed) in the same transaction; `COMMIT`.
8. The in-process outbox worker hands `sales.sale.completed` to the loyalty consumer (idempotent per `sale_id`, own transaction, `UNIQUE(tenant_id, sale_id)`); a loyalty failure does not fail the sale and is retried.

Failure points: the single transaction removes half-finished states; retry is safe via `client_sale_id`. Business rules: PRD §13 (non-negative stock, atomic, receipt required, points only after a sale). Required tests: two simultaneous sales on stock 1, retry with the same `client_sale_id`, cross-tenant.

### 7.2 PO receiving

**As-is** (`receive_stock.go`): fetch the PO (`PENDING`); for each item in the request (items outside the request are skipped) call inventory `UpdateStock` (RECEIPT) over gRPC, then publish `Inventory.Replenished`; mark the whole PO `RECEIVED`. A failure midway -> stock partially increased and the PO stays `PENDING`; a retry adds twice.

**Target.**

1. `POST /api/v1/purchase-orders/{id}/receive` with a list of `{product_id, quantity_received}` and `Idempotency-Key`.
2. In one tenant+store transaction: lock the PO (`SELECT ... FOR UPDATE`); validate that the status is not `RECEIVED`/`CANCELLED`; each quantity <= `quantity_ordered - quantity_received`.
3. Per item: increase `quantity_received`; increase stock (`stock_items` upsert) and write `stock_movements` (`reason = RECEIPT`, `ref_id = po_id`).
4. PO status: all items complete -> `RECEIVED`, partial -> `PARTIAL`.
5. Write outbox `supplier.stock.received`; `COMMIT`.

Only the roles `admin` and `stock_manager` (permission `purchase_orders:receive`, catalogue `tbd`).

### 7.3 Offline cashier queue

As-is status: Not started. The service worker (`apps/cashier/src/service-worker.ts`) only stores build assets (stale-while-revalidate, bypassing non-GET and `/api/`); `manifest.json` has `start_url` `/pos` and one icon; there is no sale queue.

**Target** (following SELA's guest upload queue pattern: IndexedDB `queue`, uploader state machine, sync trigger, cache policy `policy.ts`; `SELA/apps/web/src/lib/pwa/policy.ts`):

1. A service worker with scope `/cashier/` (`D9`); the app assets and the last product catalogue (read-only) are cached.
2. While offline, the cashier completes sales into the IndexedDB queue with a UUID `client_sale_id`; a temporary receipt is marked "not yet synced".
3. When the network returns, the uploader sends the queue in order through `POST /api/v1/sales` with `client_sale_id` (idempotent, 7.1).
4. Status per queued sale: `QUEUED -> SYNCING -> SYNCED | FAILED` (a 409 insufficient-stock or 422 price rejection: shown in the cashier/admin "needs action" list).
5. Conflict policy: insufficient stock at sync -> the sale remains valid (the goods are already sold) and the stock is flagged as a discrepancy for the admin; or it is rejected (decision D-POS-03, not yet made).
6. Offline duration limit, queue capacity, and device registration (`pos.devices.max`): `tbd`.
7. Offline authentication: an existing cashier session stays valid until it expires; a new login requires network (`tbd`, D-POS-03).

---

## 8. Security, Privacy and UU PDP

### 8.1 Threat model and controls

| Threat | As-is control | Status | Evidence | Target |
|---|---|---|---|---|
| Cross-tenant leak | RLS `ENABLE`+`FORCE`, `tenant_id` filter on queries; 7 policies without `WITH CHECK`; no integration tests | Partial | `services/_shared/database/tenant_context.go` | complete RLS + `kit/tenantdb` + cross-tenant tests (G4, G6) |
| Internal service spoofing | HMAC `x-service-token` (30-second window) verified only when present; empty/`changeme`/no header accepted | Partial | `services/_shared/middleware/grpc_interceptors.go` | deny-by-default; gRPC reflection off in production |
| Credential theft from the repo | literal secrets in tracked files | Not started | `ecosystem.config.cjs` | rotation, purge history, `.env.example` (G5) |
| Session theft | 10-minute JWT in an httpOnly cookie; `jti` blacklist; 30-day refresh | Partial | `services/auth-service/internal/infrastructure/jwt/signer.go` | opaque session, hash in Dragonfly |
| Login brute force | 20 per minute per IP; no account lock | Partial | `services/api-gateway/cmd/server/main.go` | rate limit per account and address + lockout |
| Role escalation | `WithRole` per route; no permission catalogue | Partial | `services/api-gateway/internal/middleware/role.go` | `RequirePermission` |
| Price/discount fraud by the client | client-supplied price and discount accepted | Not started | `services/api-gateway/internal/handlers/sales.go` | server-side price, authority limits |
| Duplicate sale | no idempotency key | Not started | `services/sales-service/internal/application/commands/create_sale.go` | `client_sale_id` |
| Repudiation of actions | `auth.audit_logs` is not written | Not started | `services/auth-service/migrations/000001_create_auth_schema.up.sql` | append-only `kit/audit` |

Inter-service gRPC transport is deliberately plaintext (`docs/security/grpc-transport-security.md`); in the target there is no inter-service gRPC (one process), so this risk disappears for internal paths, and machine gRPC uses TLS at Caddy `tbd`.

### 8.2 Personal data and UU PDP

| Data | Location | Class | Protection status | Evidence |
|---|---|---|---|---|
| Customer name, phone, email | `customer.customers` | PII | plain text, no column encryption | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| User (staff) email | `auth.users` | PII | plain text | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| Password | `auth.users.password_hash` | Secret | bcrypt | `services/auth-service/internal/application/commands/login.go` |
| Supplier contact (name, phone, email) | `supplier.suppliers` | business PII | plain text | `services/supplier-service/migrations/000001_create_supplier_schema.up.sql` |
| Customer purchase history | `sales.sales.customer_id` | related PII | pseudonymous by id | `services/sales-service/migrations/000001_create_sales_schema.up.sql` |

- There is no UU PDP statement in the `POS-Stery` repo (thorough search; none). The application of Law No. 27/2022 to store customer data, the processing basis (consent/contract), retention, data subject rights (access, correction, deletion), incident notification, and appointment of a responsible person are open decision D-POS-08 and need legal review before S2 is closed. This document does not claim compliance.
- Prepared technical target: PII columns encrypted with `kit/piicrypt` (AES-256-GCM, per-purpose AAD) with a blind index for phone/email search if decided; customer deletion uses anonymisation (sales and points history kept without PII); append-only audit table.
- A receipt is a transaction document; whether the snapshot may contain customer data is determined by decisions D-POS-01 and D-POS-08 (see BRD 9.3).

### 8.3 Secret hygiene and supply chain

- Tracked files that contain secrets (paths only): `ecosystem.config.cjs` (service token, Redis, NATS, Postgres passwords) and `deploy/docker/init-postgres.sql` (passwords of the roles `pos_admin` and `pos_app`). `scripts/migrate-helper.sh` has a weak default admin password. `docs/security/grpc-transport-security.md` mentions the default value `changeme`.
- Not tracked by git but present on the working disk: `.env`, `keys/private.pem`, `keys/public.pem`, `server.exe`, `apps/*/build/` (ignored by `.gitignore`; a correction to the baseline claim, see 11.3). The private JWT key is used in local development; in the target JWT is removed.
- After rotation, git history must be purged and all old credentials considered leaked (G5).
- Go dependencies differ in version across services (grpc, pgx, nats.go, jwt/v5); target one `go.mod`.

---

## 9. Technical Non-Functional Requirements

All performance and capacity figures are `tbd` (not in the repo, including the PRD which only says "high concurrency" without figures); a baseline only where it can be derived from code.

| ID | Requirement | Category | Target | As-is status | Evidence |
|---|---|---|---|---|---|
| NFR-POS-01 | Sale completion time (p95) per store | Performance | `tbd` | Not measured | `services/sales-service/internal/application/commands/create_sale.go` |
| NFR-POS-02 | Concurrent cashiers per store and stores per tenant | Capacity | `tbd` (aligned with `pos.stores.max`, `pos.devices.max`) | Not measured | `services/api-gateway/cmd/server/main.go` |
| NFR-POS-03 | API service availability | Availability | `tbd` (SLA per tier) | No monitoring | `services/api-gateway/internal/handlers/health.go` |
| NFR-POS-04 | Atomic and idempotent sales | Integrity | no duplicate sales, no negative stock | Not met | `services/sales-service/internal/application/commands/create_sale.go` |
| NFR-POS-05 | Tenant and store isolation proven by cross-tenant tests per module | Security | 100% of modules | No integration tests yet | `services/_shared/database/tenant_context.go` |
| NFR-POS-06 | Minimum 80% test coverage (Go and web), `-race` in CI | Quality | 80% | 79 stub-based unit test functions, stale CI | `.github/workflows/ci.yml` |
| NFR-POS-07 | All migrations up -> down -> up on a fresh DB in CI | Quality | mandatory | The script only counts file pairs | `scripts/migrate-validate.sh` |
| NFR-POS-08 | Configuration via fail-fast env, secrets not logged, `.env.example` without secret values | Operations | mandatory | Partial | `services/_shared/config` |
| NFR-POS-09 | Structured `slog` JSON logging and `/healthz`; metrics/tracing = D11 `tbd` | Observability | `slog` + `/healthz` | zap; no actual metrics/tracing | `services/_shared/observability` |
| NFR-POS-10 | Offline cashier resilience: PWA scope `/cashier/`, IndexedDB queue | Resilience | offline duration `tbd` | Not started | `apps/cashier/src/service-worker.ts` |
| NFR-POS-11 | Postgres backup and recovery | Operations | RPO/RTO `tbd` | None | `deploy/docker/docker-compose.yml` |
| NFR-POS-12 | Retention of transaction data and personal data | Compliance | `tbd` (D-POS-08) | No policy | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| NFR-POS-13 | Localisation: Indonesian default, English supported, IDR, WIB; WIB day boundary | Usability | mandatory | English UI; DB session time | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| NFR-POS-14 | Accessibility and usability of the cashier screen (touch targets, contrast, keyboard/scanner) | Usability | `tbd` | Not assessed | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| NFR-POS-15 | One stack and one convention (STACK-SPEC) with `standalone` behaviour unchanged in `suite` mode | Maintainability | mandatory | Diverges | `go.work` |
| NFR-POS-16 | Safe degradation when the platform is unavailable: 24-hour entitlement snapshot (configurable, `tbd`), fail closed for paid features, `standalone` sales never blocked | Resilience | 24 hours (`tbd`) | No `suite` mode yet | `services/api-gateway/internal/config/config.go` |

---

## 10. Screen Structure

### 10.1 As-is screens

| App | Screen | Function | Status | Evidence |
|---|---|---|---|---|
| Admin | `(auth)/login` | admin/stock login | Done | `apps/admin/src/routes/(auth)/login/+page.svelte` |
| Admin | `(app)/dashboard` | summary: revenue, transaction count, top products; fake payment mix and sparkline; field names do not match | Partial | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| Admin | `(app)/products` | product list (read-only) | Partial | `apps/admin/src/routes/(app)/products/+page.svelte` |
| Admin | `(app)/inventory` | inventory list (read-only) | Partial | `apps/admin/src/routes/(app)/inventory/+page.svelte` |
| Admin | `(app)/suppliers` | supplier list (read-only) | Partial | `apps/admin/src/routes/(app)/suppliers/+page.svelte` |
| Admin | `(app)/customers` | customer list (read-only) | Partial | `apps/admin/src/routes/(app)/customers/+page.svelte` |
| Admin | `(app)/reports` | sales report | Partial | `apps/admin/src/routes/(app)/reports/+page.svelte` |
| Cashier | `(auth)/login` | cashier login; requires the Store ID (UUID) to be typed | Partial | `apps/cashier/src/routes/(auth)/login/+page.svelte` |
| Cashier | `(app)/pos` | search products, cart, tender (Cash/QRIS/Debit UI only), receipt | Partial | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| Cashier | `(app)/receipts` | placeholder: text "Receipt history will appear here. PDF export available in Phase 2." | Not started | `apps/cashier/src/routes/(app)/receipts/+page.svelte` |
| Both | BFF `api/[...path]`, `auth/session` | proxy to the gateway, session cookie | Done | `apps/cashier/src/routes/auth/session/+server.ts` |
| Cashier | Service worker + manifest | asset cache; no offline queue; `start_url` `/pos` | Partial | `apps/cashier/static/manifest.json` |

UI findings (read statically, `[INFERENCE]` for runtime impact):

- There is no admin store selector and no admin page sends `X-Store-ID`, while the gateway requires it (403 "Store context required.") for the admin role: admin pages probably cannot load data (`services/api-gateway/internal/middleware/store.go`).
- There is no refresh token flow in the web or the gateway: the JWT lives 10 minutes while the cookie lives 30 days.
- The `localStorage`-based `getToken`/`setSession` helpers exist but `setSession` is never called (dead code).
- The BFF URL fallback `http://localhost:8080` differs from the gateway default port 8000.
- The cashier receipt view expects `{sale_id, total_amount, items_count}`, the gateway returns `{sale: {...}, receipt: {...}}` so the receipt shows "—" and 0 [INFERENCE].
- The search placeholder mentions SKU, which is not searched.

### 10.2 Target structure (D9)

One `apps/web` (SvelteKit SPA, `ssr=false`, `adapter-static`):

| Route group | Pages | Role |
|---|---|---|
| `(auth)` | login (email + password; a cashier picks the store from the assigned list instead of typing a UUID) | all |
| `(admin)` | dashboard, products (create/update/categories/price history), inventory (adjustment, threshold, movements), suppliers, PO (create/receive/cancel), customers, reports, store and user settings, store selector | admin, stock_manager |
| `(cashier)` | POS, receipt history and printing, offline queue ("needs action") | cashier |

PWA scope `/cashier/`; `apiFetch` with `credentials: 'include'`; i18n `{id,en}` with a parity test; components from `packages/ui` are moved.

---

## 11. Traceability Matrix and Status

### 11.1 Status per FR

| ID | Priority | Status | Evidence |
|---|---|---|---|
| FR-POS-01.1 | M | Done | `services/auth-service/internal/application/commands/login.go` |
| FR-POS-01.2 | M | Done | `services/auth-service/internal/application/commands/login.go` |
| FR-POS-01.3 | M | Partial | `services/api-gateway/internal/middleware/store.go` |
| FR-POS-01.4 | M | Partial | `services/auth-service/internal/application/commands/refresh.go` |
| FR-POS-01.5 | S | Partial | `services/auth-service/internal/application/commands/logout.go` |
| FR-POS-01.6 | M | Done | `services/api-gateway/internal/middleware/role.go` |
| FR-POS-01.7 | M | Done | `services/api-gateway/cmd/server/main.go` |
| FR-POS-01.8 | M | Not started | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |
| FR-POS-01.9 | S | Not started | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| FR-POS-01.10 | M | Not started | `services/auth-service/internal/infrastructure/jwt/signer.go` |
| FR-POS-01.11 | S | Partial | `apps/admin/src/hooks.server.ts` |
| FR-POS-01.12 | M | Not started | `services/api-gateway/internal/config/config.go` |
| FR-POS-01.13 | M | Not started | `services/auth-service/internal/infrastructure/postgres/store_repository.go` |
| FR-POS-02.1 | M | Done | `services/product-service/internal/application/commands/create_product.go` |
| FR-POS-02.2 | M | Partial | `services/product-service/internal/application/commands/update_product.go` |
| FR-POS-02.3 | S | Partial | `services/product-service/internal/interfaces/grpc/product_service.go` |
| FR-POS-02.4 | M | Partial | `services/product-service/internal/infrastructure/postgres/product_repository.go` |
| FR-POS-02.5 | S | Not started | `services/product-service/internal/application/commands/update_product.go` |
| FR-POS-02.6 | S | Not started | `services/product-service/migrations/000001_create_product_schema.up.sql` |
| FR-POS-02.7 | M | Not started | `apps/admin/src/routes/(app)/products/+page.svelte` |
| FR-POS-03.1 | M | Done | `services/inventory-service/internal/application/commands/add_item.go` |
| FR-POS-03.2 | M | Partial | `services/inventory-service/internal/infrastructure/postgres/stock_repository.go` |
| FR-POS-03.3 | M | Not started | `services/inventory-service/internal/application/commands/update_stock.go` |
| FR-POS-03.4 | M | Partial | `services/inventory-service/internal/application/commands/set_threshold.go` |
| FR-POS-03.5 | M | Partial | `services/inventory-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-03.6 | M | Partial | `services/inventory-service/internal/application/queries/list_items.go` |
| FR-POS-03.7 | M | Not started | `services/api-gateway/internal/handlers/inventory.go` |
| FR-POS-04.1 | M | Partial | `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.2 | M | Partial | `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.3 | M | Not started | `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.4 | M | Not started | `services/api-gateway/internal/handlers/sales.go` |
| FR-POS-04.5 | M | Not started | `services/sales-service/internal/domain/sale.go` |
| FR-POS-04.6 | S | Not started | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| FR-POS-04.7 | M | Partial | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-04.8 | M | Partial | `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| FR-POS-04.9 | M | Partial | `services/sales-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-04.10 | M | Partial | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| FR-POS-04.11 | C | Not started | `services/sales-service/internal/domain/sale.go` |
| FR-POS-04.12 | S | Not started | `apps/cashier/src/service-worker.ts` |
| FR-POS-05.1 | M | Done | `services/supplier-service/internal/application/commands/add_supplier.go` |
| FR-POS-05.2 | M | Done | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| FR-POS-05.3 | M | Partial | `services/supplier-service/internal/application/commands/create_purchase_order.go` |
| FR-POS-05.4 | S | Partial | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| FR-POS-05.5 | M | Partial | `services/supplier-service/internal/application/commands/receive_stock.go` |
| FR-POS-05.6 | S | Not started | `services/supplier-service/internal/application/commands/receive_stock.go` |
| FR-POS-05.7 | S | Partial | `services/supplier-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-06.1 | M | Done | `services/customer-service/internal/application/commands/create_customer.go` |
| FR-POS-06.2 | S | Partial | `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| FR-POS-06.3 | M | Not started | `services/customer-service/internal/application/commands/add_loyalty_points.go` |
| FR-POS-06.4 | S | Not started | `services/customer-service/internal/application/commands/add_loyalty_points.go` |
| FR-POS-06.5 | C | Partial | `services/customer-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-06.6 | S | Not started | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| FR-POS-07.1 | M | Partial | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.2 | M | Partial | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.3 | S | Not started | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| FR-POS-07.4 | M | Partial | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| FR-POS-07.5 | S | Not started | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.6 | S | Partial | `services/api-gateway/internal/handlers/sales.go` |
| FR-POS-SEC.1 | M | Done | `services/sales-service/migrations/000001_create_sales_schema.up.sql` |
| FR-POS-SEC.2 | M | Partial | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| FR-POS-SEC.3 | S | Not started | `services/_shared/database/tenant_context.go` |
| FR-POS-SEC.4 | M | Pending Verification | `services/_shared/database/tenant_context.go` |
| FR-POS-SEC.5 | M | Partial | `services/_shared/middleware/grpc_interceptors.go` |
| FR-POS-SEC.6 | M | Not started | `ecosystem.config.cjs` |
| FR-POS-SEC.7 | S | Not started | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| FR-POS-SEC.8 | S | Not started | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| FR-POS-SEC.9 | S | Done | `services/api-gateway/cmd/server/main.go` |
| FR-POS-SEC.10 | S | Partial | `apps/admin/src/lib/server/session.ts` |

Modules 08 NTF and 09 AI have no FRs (Not started, see 4.08 and 4.09).

Summary of FR status counts: 68 FRs (13 + 7 + 7 + 12 + 7 + 6 + 6 + 10). Done 11, Partial 30, Not started 26, Pending Verification 1 (counted from the table above; not a readiness percentage).

### 11.2 BR -> FR matrix

All BRs are defined in `BRD_POS.id.md` section 6. The NFR column points to the non-functional requirements in section 9.

| BR | Related FR | Related NFR | Notes |
|---|---|---|---|
| BR-POS-01 | FR-POS-04.1, FR-POS-04.2, FR-POS-04.3, FR-POS-04.4, FR-POS-04.7, FR-POS-04.8, FR-POS-04.10 | NFR-POS-01, NFR-POS-04 | Flow 7.1 |
| BR-POS-02 | FR-POS-SEC.1, FR-POS-SEC.2, FR-POS-SEC.3, FR-POS-SEC.4 | NFR-POS-05 | Gates G4, G6 |
| BR-POS-03 | FR-POS-01.2, FR-POS-01.3, FR-POS-01.6, FR-POS-01.9 | - | Matrix 3.2 |
| BR-POS-04 | FR-POS-02.1, FR-POS-02.2, FR-POS-02.3, FR-POS-02.4, FR-POS-02.5, FR-POS-02.6, FR-POS-02.7 | - | - |
| BR-POS-05 | FR-POS-03.1, FR-POS-03.2, FR-POS-03.3, FR-POS-03.4, FR-POS-03.5, FR-POS-03.6, FR-POS-03.7 | NFR-POS-04 | - |
| BR-POS-06 | FR-POS-05.1, FR-POS-05.2, FR-POS-05.3, FR-POS-05.4, FR-POS-05.5, FR-POS-05.6, FR-POS-05.7 | - | Flow 7.2 |
| BR-POS-07 | FR-POS-06.1, FR-POS-06.2, FR-POS-06.3, FR-POS-06.4, FR-POS-06.6 | - | Gate G1 |
| BR-POS-08 | FR-POS-07.1, FR-POS-07.2, FR-POS-07.3, FR-POS-07.4, FR-POS-07.5, FR-POS-07.6 | NFR-POS-13 | Gate G7 |
| BR-POS-09 | FR-POS-04.5 | - | D-POS-01; gate G8 |
| BR-POS-10 | FR-POS-04.7, FR-POS-04.8, FR-POS-04.10 | - | D-POS-02; gate G8 |
| BR-POS-11 | FR-POS-04.12 | NFR-POS-10 | Flow 7.3; D-POS-03 |
| BR-POS-12 | FR-POS-04.6, FR-POS-07.3 | - | D-POS-04 |
| BR-POS-13 | FR-POS-SEC.5, FR-POS-SEC.6, FR-POS-SEC.7, FR-POS-SEC.10, FR-POS-01.5, FR-POS-01.7 | NFR-POS-08 | Gate G5 |
| BR-POS-14 | FR-POS-SEC.4 | NFR-POS-06, NFR-POS-07 | Gates G3, G6 |
| BR-POS-15 | FR-POS-01.10, FR-POS-01.11 | NFR-POS-09, NFR-POS-15 | Delta 2.7 |
| BR-POS-16 | FR-POS-01.12, FR-POS-01.13 | NFR-POS-16 | `suite` mode |
| BR-POS-17 | FR-POS-03.5 | - | Module 4.08 without FRs (Not started) |
| BR-POS-18 | (no POS FR) | - | Module 4.09 without FRs (Not started, PRD phase 2/3) |
| BR-POS-19 | FR-POS-SEC.8 | NFR-POS-12 | D-POS-08 |
| BR-POS-20 | (no POS FR) | - | Hypothesis INT-3, specified in FSD Stery Suite 4.07; Deferred |
| BR-POS-21 | FR-POS-01.8 | - | D-POS-12 |

### 11.3 Contradictions between status documents and code

**A. Document contradictions against code** (code = truth; the documents are not modified by this work).

| # | Document | Claim | Code shows | Evidence |
|---|---|---|---|---|
| K1 | `PRD.md` §10.1 | "Laravel gateway for auth and HTTP API" | Go `chi` gateway; there is no Laravel directory | `services/api-gateway/cmd/server/main.go` |
| K2 | `PRD.md` §17 | gateway authentication "JWT / Sanctum" | RS256 JWT; there is no Sanctum | `services/auth-service/internal/infrastructure/jwt/signer.go` |
| K3 | `PRD.md` §11.1 | `AuthService: Login, Validate` | 7 RPCs (`Logout`, `Refresh`, `LogoutAll`, `ListSessions`, `ValidateStore` added) | `proto/pos/auth/v1/auth.proto` |
| K4 | `PRD.md` §11.2 | `POST /api/ai/forecast-demand` exists | the route does not exist; the code adds customer routes and `PUT /api/products/{id}` | `services/api-gateway/cmd/server/main.go` |
| K5 | `PRD.md` §13 | "Customer points updated only after successful sale" | not enforced: `AddLoyaltyPoints` has no caller | `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| K6 | `PRD.md` §12.1 | tables `notifications`, `ai_forecasts`, `permissions` required | not in the migrations | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| K7 | `PRD.md` §7.1 | product search by name, SKU, barcode | only name (`ILIKE`) and exact barcode | `services/product-service/internal/infrastructure/postgres/product_repository.go` |
| K8 | `PRD.md` §14.3 | admin receives low-stock notifications | no event consumer | `services/inventory-service/internal/infrastructure/nats_publisher.go` |
| K9 | `README.md` and `CLAUDE.md` | four tenant isolation layers, including store-scope RLS | no RLS policy uses `app.store_id`; store scope exists only in the application layer | `services/_shared/database/tenant_context.go` |
| K10 | `docs/security/grpc-transport-security.md` | a pod without the secret cannot call any backend service | the interceptor accepts requests without a token header, without metadata, or with an empty/`changeme` secret (fail-open) | `services/_shared/middleware/grpc_interceptors.go` |
| K11 | interceptor comment | chain "recover -> otel -> metrics -> logging -> service-auth -> tenant-enforce" | only service-auth and logging; no tenant enforcement on gRPC | `services/_shared/middleware/grpc_interceptors.go` |
| K12 | `README.md` | "structured logging via OpenTelemetry" | logging uses zap; there is no OpenTelemetry/Prometheus code (only `go.sum`/viper defaults); `METRICS_PORT` is not read by code | `services/_shared/observability` |
| K13 | `CLAUDE.md` | `make dev` runs Postgres 16 | `deploy/docker/docker-compose.yml` contains redis, nats, jaeger, prometheus without Postgres | `deploy/docker/docker-compose.yml` |
| K14 | `CLAUDE.md` and `README.md` | tenants "seeded via console command" | no code or seed creates tenants, stores, or users | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |
| K15 | `CLAUDE.md` | the SvelteKit apps section mentions the "Laravel gateway" at `PUBLIC_GATEWAY_URL`, its own diagram says Go chi | Go chi | `CLAUDE.md` |
| K16 | `README.md` / `CLAUDE.md` | Go 1.22+ / pnpm | `go.work` 1.25.0, `Dockerfile.base` 1.25; a `package-lock.json` (npm) exists | `go.work` |
| K17 | `.github/workflows/ci.yml` | job "Laravel gateway tests" (php 8.3, composer, Pest, `apps/api-gateway`) | the `apps/api-gateway` directory does not exist; the gateway is Go | `.github/workflows/ci.yml` |
| K18 | `.github/workflows/ci.yml` | Go `'1.24'`; triggers `main`/`develop`; `npm --filter @pos-stery/admin check` | Go 1.25 (`Dockerfile.base`, `go.work`); active branch `development`; pnpm-style `--filter` under npm; no root `package.json`; no e2e | `.github/workflows/ci.yml` |
| K19 | `scripts/preflight.sh` | requires PHP/Composer/pnpm and Go 1.24+ | the stack no longer uses PHP/pnpm | `scripts/preflight.sh` |
| K20 | `AddLoyaltyPoints` comment | "called by sales-service after Sale.Completed" | no caller | `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| K21 | `PRD.md` §18 vs the suite plan | the plan calls AI forecasting Phase 3 | `PRD.md` §18 places forecasting and recommendations in Phase 2; dynamic pricing, imaging, chat in Phase 3 | `PRD.md` |
| K22 | admin dashboard UI | shows a 55/30/15 payment mix and sparkline as data | the numbers are synthesised from `total_revenue`; the backend does not store the payment method | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| K23 | cashier UI | `payment_method` Cash/QRIS/Debit is selected | the gateway drops `payment_method`; there is no column | `services/api-gateway/internal/handlers/sales.go` |

**B. Corrections to baseline claims re-read from the code** (claims from the [S] scanner that are not fully correct).

| # | Baseline claim [S] | Code shows | Evidence |
|---|---|---|---|
| C1 | `scripts/migrate-validate.sh` does not exist | the file exists; it only checks up/down pairs for six services | `scripts/migrate-validate.sh` |
| C2 | `keys/`, `.env`, `server.exe` are tracked by git | not tracked (ignored by `.gitignore`), but present on the working disk; only `ecosystem.config.cjs` and `deploy/docker/init-postgres.sql` are tracked | `.gitignore` |
| C3 | four RLS policies without `WITH CHECK` | seven: `customer.loyalty_points`, `inventory.stock_thresholds`, `product.categories`, `sales.sale_items`, `sales.receipts`, `supplier.purchase_orders`, `supplier.purchase_order_items` | `services/supplier-service/migrations/000001_create_supplier_schema.up.sql` |
| C4 | OpenTelemetry/Prometheus exist in POS | only indirect dependencies and configuration defaults; no code | `services/_shared/observability` |
| C5 | the service token interceptor protects gRPC | fail-open (see K10) | `services/_shared/middleware/grpc_interceptors.go` |
| C6 | sqlc is used in POS | only `auth-service`; other services use raw pgx (the `sqlc` directory in sales-service is empty) | `services/auth-service/sqlc.yaml` |
| C7 | Postgres integration tests "never run" | there are none at all (the Makefile target `test-integration` only sets an env variable) | `Makefile` |
| C8 | `extractIPAddr` returns `0.0.0.0` (auth) | the claim belongs to HRIS, not applicable to POS | `services/auth-service/internal/application/commands/login.go` |

**C. Not verified at runtime** (Pending Verification, [INFERENCE] from static reading): `SET LOCAL app.tenant_id = $1` with a bound parameter and two commands (FR-POS-SEC.4); an admin without a store selector receives 403; the cashier receipt shows empty values; an expired JWT without refresh; stock and loyalty races.

---

## 12. Closing Notes

- **Document status.** First draft; issued together with `BRD_POS.id.md`. Updates: every POS code change MUST update sections 4, 5, 6, and 11 (SELA FSD practice, STACK-SPEC 18); the version is bumped and the "Changes from previous" column is filled in.
- **Blocking open decisions** (defined in BRD 9.3): D-POS-01 (PPN) and D-POS-02 (receipt) close gate G8; D-POS-03 (offline) determines flow 7.3; D-POS-04 (payment method) determines the sale and report schema; D-POS-05 (loyalty) determines FR-POS-06.4; D-POS-08 (UU PDP) determines FR-POS-SEC.8; D-POS-09 (package) determines the limits `pos.stores.max` and `pos.devices.max`; D-POS-10 (observability); D-POS-11 (cookie/session TTL); D-POS-12 (tenant provisioning); D-POS-13 (data migration).
- **What was not run.** No build, lint, or test; the figures (28 RPCs, 14 routes, 79 test functions, 15 RLS policies, 7 without `WITH CHECK`) come from static reading and counting.
- **Business values.** Prices, SLAs, quotas, KPI targets, competitor data, and performance figures are all `tbd`; there are no `Rp` figures in this document.
- **Translation.** The English version `FSD_POS.en.md` was produced by the translation lane with an identical structure; this Indonesian version is the source of truth.
