# 02 - Decision gates

| # | Question | Options | Recommendation | Affects | Status | Decision |
| --- | --- | --- | --- | --- | --- | --- |
| D1 | Scope of alignment | Toolchain only / everything incl. architecture | Everything | all units | Decided | Everything: one Go module, services as packages (2026-11-08) |
| D2 | HTTP entry point | Keep gateway / Go ServeMux | Go ServeMux like SELA | U5, U6 | Decided | Go `net/http` ServeMux replaces the chi gateway `services/api-gateway` (2026-11-08) |
| D3 | Event backbone | NATS JetStream / Dragonfly Streams | Dragonfly Streams | U3, U5 | Decided | Dragonfly Streams, stream `pos:events` (2026-11-08) |
| D4 | Frontend | Rewrite to SELA libs / toolchain only | Toolchain only | U7 | Decided | Keep UI libraries, align toolchain versions (2026-11-08) |
| D5 | Shared infrastructure code | Copy per product / shared kit | Shared kit | U1, U3, U4, U7 | Decided | Shared repo `stery-kit` (Go module `github.com/Briyantama/stery-kit/go`, npm `@stery/kit`), used by SELA, HRIS-Stery and POS-Stery; local path dependency until pushed (2026-11-08) |
| A1 | Repository layer | Keep pgx/v5 + sqlc / rewrite to database/sql | Keep pgx/v5 | U5 | Open | Assumed: keep |
| A2 | Observability | Remove OTel/Prometheus / keep exporter config | Remove, SELA has none | U1, U8 | Open | Assumed: remove |
| A3 | gRPC and protobuf | Remove / keep | Remove, no external consumer | U5, U9 | Open | Assumed: remove |
| A4 | Frontend adapters and server session layer | Keep / static SPA with API cookies | Keep | U7 | Open | Assumed: keep |
| A5 | Module path | `github.com/Briyantama/POS-Stery` | as SELA convention | U1 | Open | Assumed |
| A6 | Commits | none unless asked | none | all | Open | Assumed: no commit or push |
| A7 | Full `scripts/check.sh` | CI only / local | CI only (SELA owner policy 2026-10-17) | Verification | Open | Assumed: CI only |
| A8 | API contract rules (pagination, status codes, snake_case, 404 for cross-tenant) | as plan step 5 | as plan | U5 | Open | Assumed |
