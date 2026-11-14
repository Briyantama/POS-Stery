# ADR-0001: Single-module Go API on the SELA stack

**Status:** Accepted  
**Date:** 2026-11-08  
**Deciders:** Owner (decisions D1 to D5, 2026-11-08)  
**Supersedes:** `docs/security/grpc-transport-security.md` (inter-service gRPC transport)

---

## Context

POS-Stery ran as eight Go processes: a `chi` REST gateway (`services/api-gateway`) and six gRPC services (auth, product, inventory, sales, supplier, customer), tied together by `go.work`, protobuf/buf code in `gen/go`, an HMAC `X-Service-Token` between services, zap, viper, golang-migrate per service and Postgres 16. Sales and supplier changed stock by calling inventory over gRPC.

SELA, the reference product in the same suite, runs one Go module with one binary on stdlib `net/http`, `log/slog`, embedded goose migrations, Postgres 18 and Dragonfly. The owner decided to align HRIS-Stery and POS-Stery with it (D1: scope covers architecture too).

POS has no external gRPC client. Every caller is one of the two SvelteKit apps, through their server-side proxy.

## Decision

- `apps/api` is one Go module, `github.com/Briyantama/POS-Stery`, Go 1.25.1. It builds two binaries: `cmd/api` (listens on `PORT`, default 8000, health `GET /healthz`) and `cmd/migrate` (`up|down`, reads `DATABASE_URL`).
- Each former service is a package under `apps/api/services/<name>`. The `domain` and `application` layers moved unchanged; gRPC servers, `cmd/server` and per-service `go.mod` files are deleted.
- Routes are mounted on one `http.ServeMux` (D2). The old gateway paths keep their shape with `/v1` inserted (`POST /api/v1/login`, `/api/v1/products`, ...). Responses use the envelope `{success,data,error}`.
- Sales and supplier call inventory's `UpdateStockHandler` in-process. The deduct (`-qty`, reason SALE) and restore (`+qty`, reason ADJUSTMENT) semantics stay the same.
- JWT RS256 with the existing claims (`sub`, `tid`, `sid`, `role`, `email`, `jti`) stays, verified by middleware with the Dragonfly jti blacklist. Store validation is an in-process call to the auth service.
- Migrations are goose files in `apps/api/internal/db/migrations` (goose v3.26.0, embedded). Tenant transactions set `app.tenant_id` and `app.store_id` with `set_config(..., true)`.
- Logging is `log/slog`. OpenTelemetry, Prometheus and Jaeger are removed.
- Protobuf, buf, `gen/`, `go.work` and the service token are removed. No gRPC server runs.
- Shared code (D5): HTTP helpers, health, object storage, migrate, test database, tenant transactions, events, authentication and rate limiting live in the sibling repo `stery-kit` (Go module `github.com/Briyantama/stery-kit/go`, npm package `@stery/kit`). POS consumes it with `replace github.com/Briyantama/stery-kit/go => ../../../stery-kit/go` and `file:../../../stery-kit/web`. Docker builds of `apps/api` use the parent of the repo as build context, and CI checks out `stery-kit` next to the repo.

## Consequences

- One deployable (`api`) instead of eight. Calls between modules are function calls, so the service token and its fail-open check are gone.
- A sale deducts stock in the same process, but still not in the same database transaction as the sale record. The compensating `+qty` on failure stays.
- The modules can no longer be scaled or deployed independently.
- No metrics or traces exist until a later decision adds them. Logs and `/healthz` are the only signals.
- Building or testing the API requires a `stery-kit` checkout next to the repo. CI fails until `Briyantama/stery-kit` is pushed.

## Alternatives considered

- **Keep the microservices and only swap libraries.** Rejected by D1: the owner asked for SELA's architecture, not just its dependencies.
- **Keep gRPC between packages for a later split.** Rejected: no external gRPC consumer exists, and Go interfaces keep the module boundaries.
- **Copy the shared packages into each product.** Rejected by D5: the same code would drift between HRIS and POS.
