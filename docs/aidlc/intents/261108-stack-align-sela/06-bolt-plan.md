# 06 - Bolt plan

| Bolt | Unit | Definition of done | Evidence |
| --- | --- | --- | --- |
| B0 | - | Vault read, intent docs written, progress note created | this folder; vault `Projects/stery-suite-stack-alignment.md` |
| B1 | U1-U4 | Foundation packages compile and their tests pass | `go build ./... && go test ./internal/...` |
| B2 | U2, U5 | Every service package ported with `http.go` and `http_test.go` | `go test ./services/...` |
| B3 | U6 | One binary serves all routes and runs the consumers | `go build ./cmd/api`; smoke |
| B4 | U7, U8, U10 | Frontends on SELA toolchain; infra, CI, docs aligned | `npm run check && npm run build`; `bash -n scripts/check.sh` |
| B5 | U9 | Old tree removed, no stale references | grep for `nats`, `zap.`, `grpc`, `go.work` returns nothing |
| B6 | - | Verification 1-9 of the plan | command output recorded in `audit.md` |

## Human gates

- Plan approved 2026-11-08 (D1-D4).
- No commit or push without an explicit request.

## Bounded autonomy

AI may write code, tests and docs on the current branch. It asks before destructive operations outside this plan, real credentials, or pushing.
