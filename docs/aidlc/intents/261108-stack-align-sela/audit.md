# Audit log (append-only)

| Date | Actor | Event | Rationale |
| --- | --- | --- | --- |
| 2026-11-08 | Owner | Asked to align POS-Stery with the SELA stack | One stack across the three products |
| 2026-11-08 | Owner | D1 everything incl. architecture; D2 Go ServeMux; D3 Dragonfly Streams; D4 frontend toolchain only | Answers to the planning questions |
| 2026-11-08 | Owner | Approved the plan | Plan stored as session artifact `stack-align-sela-plan.md` |
| 2026-11-08 | AI | Wrote intent docs 00-06, state and audit; A1-A8 recorded as Open assumptions | No assumption taken silently |
| 2026-11-08 | Owner | D5 shared kit for backend and frontend, SELA migrated too, consumed by local path until pushed; remote git@github.com:Briyantama/stery-kit.git | Remove duplicate infrastructure across the three products |
| 2026-11-08 | AI | Route table set in `05-units-of-work.md`; roles for routes without a gateway rule chosen conservatively (writes `hr_admin` in HRIS); gateway-only validate RPCs get no route; POS refresh is public | Old gateways did not define them; owner can override |
| 2026-11-08 | Owner | Kit remote moved to git@github.com:Briyantama/stery-kit.git; Go module path is `github.com/Briyantama/stery-kit/go` | Owner choice: replace origin and module path |
| 2026-11-08 | AI | Admin store selector (`X-Store-ID`) and inventory list fields (`product_name`, `min_quantity`, `is_low_stock`) recorded as open gaps, not built | Pre-existing gaps outside the stack change |
