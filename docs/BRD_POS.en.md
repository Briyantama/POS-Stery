# BRD POS-Stery

*Business Requirements Document: multi-tenant POS SaaS for retail stores (the `POS` product in the Stery Suite portfolio)*

| | |
|---|---|
| **Document Name** | BRD POS-Stery (Business Requirements Document) |
| **Version** | 1.0 (Draft) |
| **Date** | 6 November 2026 |
| **Prepared by** | Product & Engineering Team |
| **Status** | Draft for review |
| **Related Documents** | `PRD.md` (POS-Stery PRD, in English, source of product requirements); `FSD_POS.id.md` (functional specification, contains the BR -> FR matrix in 11.2); `00_Analisis_Dampak_Risiko` (Stery Suite impact and risk analysis); `SELA/CLAUDE.md` and `SELA/docs/FSD_Sela.md` (Stery Suite stack standard) |
| **Changes from previous** | Changes from previous: First issue |

> **Method note.** This document is the first BRD for POS-Stery. Product requirements come from `PRD.md` (402 lines, in English), which is not changed; this BRD only references its sections (e.g. PRD §4) and records where the code deviates from the PRD. The source of truth for status is the code: every statement about the current state carries a status from {Done, Partial, Not started, Pending Verification, Deferred} and a file path in the `Evidence` column (paths relative to the `POS-Stery` repo root). Business values not found in the repo (prices, SLAs, quotas, KPI targets, competitor data) are written as the literal `tbd` and recorded as open decisions `D-POS-NN` (section 9). There are no price figures in this document. No build, lint, or test was run to produce it; all findings come from static reading of the code.

---

## 1. Executive Summary

POS-Stery is a multi-tenant Point of Sale (POS) SaaS platform for retail stores: one company (tenant) can own several stores, cashiers sell from the cashier interface, and admins manage products, stock, suppliers, customers, and reports (`PRD.md` §1-§3). In the Stery Suite portfolio, POS is the `POS` SKU, sold standalone, together with one other product, or as part of the full `SUITE` package (see the Stery Suite BRD).

**Current state (code = truth).** The Go backend consists of six gRPC services (auth, product, inventory, sales, supplier, customer), all 28 RPCs of which have real handlers, plus a Go `chi` REST gateway with 14 business routes (Partial: 14 of 28 operations are exposed over REST). There are two SvelteKit applications (admin and cashier), both SSR with a BFF proxy. The multi-tenant foundation (`tenant_id` column, RLS `ENABLE` + `FORCE` on all 15 policies) is in place. However, the core business rules of PRD §13 are not yet fully met:

- loyalty points are never added after a sale (`AddLoyaltyPoints` has no caller);
- stock deduction and sale persistence are not one transaction (gRPC saga with best-effort compensation) and there is no idempotency key;
- the "stock must not go negative" rule is race-prone because stock updates have no guard in SQL;
- the admin dashboard fabricates a 55/30/15 payment mix and the top-product name is always empty;
- there is no VAT (PPN) on the receipt, no stored payment method, no offline cashier queue;
- there is no code to create a tenant, store, or first user;
- CI is stale (Laravel job, Go 1.24 versus 1.25), there are no Postgres integration tests, and secrets are written in two git-tracked files (`ecosystem.config.cjs`, `deploy/docker/init-postgres.sql`; only the paths are recorded, not the values).

**Direction.** Owner decision (analysis `00_Analisis_Dampak_Risiko`): Stery Suite is built on the principle of composition, not consolidation; SELA's stack, patterns, and architecture become the standard for the whole suite. POS will be aligned to that stack (Go modules in one process, stdlib REST `/api/v1`, PostgreSQL 18 with goose, opaque sessions, Postgres outbox, Dragonfly, RustFS, a single `apps/web` SvelteKit SPA) in workstream S2 and ends at the **POS stabilisation gate** (section 12). Operating modes: `STERY_MODE=standalone` (standalone behaviour) or `suite` (identity, entitlement, billing, notification, audit delegated to the platform).

**Intended business outcomes:** cashiers complete sales atomically and dependably; stock is accurate per store; reports use real data; customer data is protected; packages (number of stores and devices) can be sold and limited via the dimensions `pos.stores.max` and `pos.devices.max` (values `tbd`).

---

## 2. Background and Problem Statement

### 2.1 Background

- The owner decided to sell HRIS-Stery, POS-Stery, and SELA as packages (all three, any two products, or one product), each able to run standalone or integrated.
- POS-Stery was born as a microservice monorepo: gateway, six Go services, two web applications, NATS JetStream, Redis, PostgreSQL 16 with RLS (`PRD.md` §10, `CLAUDE.md`). The repo has 21 commits; the last commit is 30 June 2026 on the `development` branch.
- The SELA stack (single Go module, REST + gRPC without a gateway, Postgres 18, goose, Dragonfly, RustFS, SvelteKit SPA, `scripts/check.sh`) has been set as the suite standard; POS must be aligned, not rewritten from scratch: the domain and application layers are kept, infrastructure and interfaces are rewritten (FSD 2.7).

### 2.2 Problem statement

| # | Problem | Business impact | Status | Evidence |
|---|---|---|---|---|
| P1 | Loyalty points never increase after a sale | The loyalty programme (PRD §7.5, §13) does not work; the promise to customers is not met | Not started | `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| P2 | Sales are neither atomic nor idempotent: stock is deducted via gRPC and then the sale is written, compensation is only best-effort | Stock and sales can go out of sync; a double click or retry creates a duplicate sale or deducts stock twice | Partial | `services/sales-service/internal/application/commands/create_sale.go` |
| P3 | Stock updates have no SQL guard; there is no stock movement ledger | Negative stock is possible under concurrency; there is no trail of the reason for a change | Partial | `services/inventory-service/internal/infrastructure/postgres/stock_repository.go` |
| P4 | The receipt contains no VAT (PPN), payment method, or customer | The receipt cannot yet serve as proof of a taxable transaction; the payment method chosen by the cashier is lost | Not started | `services/sales-service/internal/domain/sale.go` |
| P5 | The admin dashboard fabricates data (55/30/15 payment mix, synthetic sparkline bars, a default cash float of 500.000) and the top-product name is empty | The owner makes decisions from fake numbers | Partial | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| P6 | There is no path to create a tenant, store, or user; no password reset | New customers cannot be onboarded without manual SQL; cannot be sold as SaaS | Not started | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |
| P7 | The cashier has no offline mode; the service worker only caches build assets | The store stops selling when the internet drops (PRD §9.3 says PWA/offline is "highly preferred") | Not started | `apps/cashier/src/service-worker.ts` |
| P8 | Secrets written in git-tracked files; stale CI; no Postgres integration tests | Security risk and undetected regressions | Partial | `ecosystem.config.cjs` |
| P9 | The REST surface covers only 14 of 28 operations: no `GetSale`, `ReceiveStock`, PO list, loyalty, stock threshold, `GetItem`, `UpdateStock`, refresh token | The web interfaces cannot reach features that already exist in the backend | Partial | `services/api-gateway/cmd/server/main.go` |
| P10 | The store selector for admins is absent from the UI, although the gateway requires a store context for every authenticated route | By static reading, the admin pages cannot load data [INFERENCE] | Not started | `services/api-gateway/internal/middleware/store.go` |
| P11 | There is no UU PDP statement for customer data (name, phone, email) | Compliance risk not yet assessed | Pending Verification | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |

---

## 3. Business Objectives

| Objective ID | Objective | Measure of success (target `tbd`, see section 11) |
|---|---|---|
| T1 | Fast, atomic, and dependable cashier sales | Sale completion time; sale success rate; zero duplicate sales |
| T2 | Accurate stock per store | System vs physical stock variance; zero negative stock |
| T3 | Reports use real data | Zero synthetic numbers on the dashboard; complete product names |
| T4 | Ready to sell as SaaS (tenant, store, user, package, limits) | Time to onboard a new tenant; dimensions `pos.stores.max` and `pos.devices.max` enforced |
| T5 | Sellable standalone and within the suite | `standalone` and `suite` modes pass the same tests |
| T6 | Secure and compliant | All RLS policies have `WITH CHECK`; secrets rotated and purged; UU PDP decision recorded |
| T7 | Maintainable by a small team | CI green with the `check.sh` pattern; one stack, one convention |

---

## 4. Scope

### 4.1 Segments and packages

- **Target segment** (`PRD.md` §1, §6): retail stores and store chains (one tenant, several stores). Market size and more detailed segmentation are not in the repo: `tbd` (open question RQ-1 in section 8).
- **Product SKU:** `POS`. **Suite bundles:** `HRS+POS`, `POS+SEL`, `SUITE` (bundle discount rule, billing cycle, trial period: `tbd`; defined in the Stery Suite BRD).
- **POS limit dimensions** (names fixed, values `tbd`): `pos.stores.max` (number of stores per tenant), `pos.devices.max` (number of cashier devices per tenant). Package tiers (name and value): `tbd` (D-POS-09).
- **Modes:** `standalone` (default; users, sessions, and packages owned by POS itself) and `suite` (delegated to the platform). A POS tenant in `suite` mode is linked to a platform organisation via UUID and the `external_ref` column.

### 4.2 In scope

Derived from `PRD.md` §3, §5, §7 and §18 Phase 1, with code status in section 13:

1. Tenant and store hierarchy, data isolation per `tenant_id` and `store_id` (PRD §6).
2. Authentication and RBAC for the roles `admin`, `cashier`, `stock_manager` (PRD §5, §17).
3. Product catalogue (create, update, search) and basic pricing/discounts (PRD §7.1, §7.6).
4. Inventory per store, adjustments, stock thresholds, low-stock alerts (PRD §7.2).
5. Cashier sales, controlled discounts, digital receipt and snapshot (PRD §7.3, §7.6, §7.7).
6. Suppliers, purchase orders (PO), stock receipt (PRD §7.4).
7. Customers and loyalty points (PRD §7.5).
8. Daily sales reports per store and per tenant (PRD §7.8).
9. Alignment to the Stery Suite stack, `standalone`/`suite` modes, package limits (sections 6 and 12).
10. Onboarding of tenants, stores, and users (does not exist yet; BR-POS-21).
11. Offline cashier queue and receipt printing (open decisions D-POS-02, D-POS-03).

### 4.3 Deferred

| Item | Reason for deferral | Reference |
|---|---|---|
| Notification service and low-stock alerts to admins (NTF module) | No service yet; the `Product.LowStock` event is published but has no consumer. Deferred to PRD Phase 2 and to the platform notification hub (`suite` mode) | `PRD.md` §10.2, §14.3, §18 Phase 2 |
| AI demand forecasting (AI module) | There is no AI service in the repo; the PRD places forecasting and recommendations in Phase 2 (§18), dynamic pricing, image recognition, and chat in Phase 3 | `PRD.md` §8, §18 |
| Advanced analytics, dynamic pricing, product image recognition, chat | PRD Phase 3 | `PRD.md` §18 Phase 3 |
| POS integration with SELA (INT-3: a store event creates a SELA event and invites opted-in customers) | Suite hypothesis, phase S5; validate demand first | Stery Suite plan |
| HRIS staff <-> POS user integration (INT-1) | Suite hypothesis, phase S5 | Stery Suite plan |
| Hardware barcode scanner integration | PRD Phase 2 | `PRD.md` §18 Phase 2 |

### 4.4 Out of scope

Repeating `PRD.md` §4 (Non-Goals):

- No marketplace or multi-vendor commerce.
- No accounting/ERP replacement in phase 1.
- No complex promotion engine beyond basic discounts in the first release.
- No advanced warehouse management system (WMS) in phase 1.
- No direct payment gateway implementation unless explicitly added later. (Recording the payment method on a sale is not a payment gateway; see D-POS-04.)

---

## 5. Stakeholders

| Role | Interest Description | Level of Influence |
|---|---|---|
| Product owner / business sponsor | Package, price, alignment order (S2), VAT and receipt decisions | High |
| Tenant admin (`admin`) | Manages stores, products, stock, suppliers, prices, reports; receives low-stock alerts | High |
| Cashier (`cashier`) | Searches products, sells, applies permitted discounts, prints/views receipts; needs speed and offline resilience | High |
| Stock manager (`stock_manager`) | Monitors stock, creates POs, confirms receipts | Medium |
| Store customers | Tracked for loyalty points; their personal data is protected | Medium |
| Suppliers | Recipients of POs (no digital channel yet) | Low |
| Engineering team (solo/small) | Aligns the stack, maintains quality; limited WIP capacity | High |
| Stery Suite platform operator | Entitlement, billing, SSO in `suite` mode | High (`suite` mode) |
| Legal / compliance advisor | UU PDP decision for customer data; VAT and receipt decisions | Medium (`tbd`: not yet appointed) |

---

## 6. Business Requirements

MoSCoW priority: M = Must, S = Should, C = Could. The "PRD Reference" column points to a `PRD.md` section; implementation status per BR is in FSD 11.2 and 11.1.

| **ID** | **Business Requirement** | **Priority** | **PRD Reference** |
|---|---|---|---|
| BR-POS-01 | The cashier can complete a sale (search/scan product, cart, permitted discounts, completion) and the system stores the history and receipt. Stock deduction and sale persistence must be atomic and must not be duplicated by retries | M | §7.3, §13, §14.2 |
| BR-POS-02 | All business data is isolated per tenant (mandatory) and per store (for store data); no cross-tenant leakage in the API, services, repositories, or events | M | §6.2, §13, §17 |
| BR-POS-03 | Role-based access: `admin` (tenant), `cashier`, `stock_manager` with authority limits enforced in the backend; the cashier is bound to the assigned store | M | §5, §17 |
| BR-POS-04 | Admins can create and change products, categories, and prices; price changes can be audited | M | §7.1, §7.6 |
| BR-POS-05 | Stock per store is always accurate: it must not go negative, every change records its reason and reference, stock thresholds can be configured, and a low-stock event is published when the threshold is crossed | M | §7.2, §13, §14.1, §14.3 |
| BR-POS-06 | The stock manager can manage suppliers, create POs, and receive goods so that stock increases exactly once per receipt | M | §7.4, §14.4 |
| BR-POS-07 | Customers can be recorded and loyalty points increase only after a successful sale, exactly once per sale; the rules for earning and redeeming points are defined (D-POS-05) | S | §7.5, §13, §14.5 |
| BR-POS-08 | Daily sales reports per store and per tenant use real data (revenue, transaction count, top products with names, payment method mix) with the day boundary following the store time zone (WIB) | M | §7.8, §14.6 |
| BR-POS-09 | VAT/tax treatment is defined (D-POS-01) and the receipt and reports reflect it; the decision is recorded before the stabilisation gate is closed | M | §7.7 (does not mention tax: PRD gap) |
| BR-POS-10 | The digital receipt can be displayed, printed, and re-retrieved from history; the print medium is defined (D-POS-02) | S | §7.7, §9.2 |
| BR-POS-11 | The cashier can still sell when the internet is disconnected (offline queue, automatic sync, no duplicate sales); the offline scope is defined (D-POS-03) | S | §9.3, §18 Phase 2 |
| BR-POS-12 | The payment method on a sale (cash, QRIS, debit as recording only; no payment gateway) is defined and stored on the sale and in reports (D-POS-04) | S | §4 (gateway non-goal), §7.3 |
| BR-POS-13 | Security hygiene: no secrets in the tracked repo (rotation and history purge), internal transport secured, audit trail available for sensitive actions | M | §17 |
| BR-POS-14 | Automated quality gate: CI green following the SELA `check.sh` pattern, Postgres integration tests (including cross-tenant and RLS tests) run, test coverage measured | M | §15, §19 |
| BR-POS-15 | POS is aligned to the Stery Suite standard stack (STACK-SPEC) with functional behaviour preserved, ending at the POS stabilisation gate | M | Suite decisions (D1-D12) |
| BR-POS-16 | POS can run `standalone` or `suite` per customer (`STERY_MODE`), with package limits `pos.stores.max` and `pos.devices.max` enforced through entitlement and safe degradation when entitlement expires | M | Suite decisions (D2, D4) |
| BR-POS-17 | Admins receive low-stock notifications and important sales events (NTF module) | S | §10.2, §14.3, §18 Phase 2 |
| BR-POS-18 | Demand forecasting: reorder quantity suggestions based on sales history per product per store | C | §8.1, §14.7, §18 Phase 2 |
| BR-POS-19 | Customer personal data (name, phone, email) is treated in accordance with UU PDP: the decision on application, processing basis, retention, and data subject rights is defined and recorded (D-POS-08) | S | §17 (does not mention UU PDP: PRD gap) |
| BR-POS-20 | Hypothetical integration with SELA (INT-3): a store event can create a SELA event for the organisation and invite consenting customers; only built if demand is validated | C | Stery Suite plan (S5) |
| BR-POS-21 | Onboarding of new customers: tenant, store, first user, and password reset can be done without manual SQL (`standalone` mode: operator console or self-service, D-POS-12; `suite` mode: by the platform) | M | §5.1 ("Creates stores"), §6.1 |

---

## 7. Business Model and Monetisation

All commercial values for POS are `tbd`; no prices, quotas, or SLAs were found in the `POS-Stery` repo (confirmation: there is no pricing, billing, or subscription file). This document only sets the structure.

| Aspect | Provision | Value |
|---|---|---|
| Unit of sale | SKU `POS`; bundles `HRS+POS`, `POS+SEL`, `SUITE` | Bundle discount rule: `tbd` |
| Limit dimensions | `pos.stores.max` (stores per tenant); `pos.devices.max` (cashier devices per tenant) | Value per tier: `tbd` |
| Package tiers | Tier names and contents | `tbd` (D-POS-09) |
| Subscription price | Per tier | `tbd` |
| Billing cycle, trial period, late fees | Determined by the platform (suite billing uses SELA's `payment.Provider` abstraction, Midtrans QRIS) | `tbd` |
| Availability SLA, support | Per tier | `tbd` |
| Behaviour when entitlement expires | The last entitlement snapshot is used until the grace period (default 24 hours, configurable, `tbd`), then paid features fail closed; cashier sales in `standalone` mode are never blocked | Grace period: `tbd` |
| Store customer payments (QRIS/debit at the cashier) | Not a payment gateway; recording only (D-POS-04) | — |

Open questions: price per tier, whether devices are limited per store or per tenant, whether there is a free tier (D-POS-09).

---

## 8. Alternatives and Competitor Analysis

### 8.1 Architecture alternatives (summary; full analysis in `00_Analisis_Dampak_Risiko` section 5)

| Option | Description | Decision for POS |
|---|---|---|
| A. Polyrepo + platform repo | POS stays a separate repo and deployable, separate database; the `stery-suite` platform holds identity, entitlement, billing, notification, audit | **Chosen** |
| B. Monorepo | All products in one repo | Not chosen |
| C. Single codebase/runtime | POS, HRIS, SELA merged | Rejected: violates standalone sales and increases blast radius |

### 8.2 Internal alternatives for POS

| Option | Notes | Decision |
|---|---|---|
| Keep gRPC microservices + chi gateway + NATS | Matches the current code | Not chosen: conflicts with the suite stack decision; the cross-service saga complicates sale atomicity |
| Align to the SELA stack (modules in one process, Postgres outbox) | A sale becomes a single database transaction | **Chosen** (BR-POS-15) |

### 8.3 Competitors

There is no competitor data in any repo. Everything below is `tbd`; do not fill it in without research.

| Competitor | Segment | Price | Key features | Notes |
|---|---|---|---|---|
| `tbd` | `tbd` | `tbd` | `tbd` | `tbd` |

Open research questions:

- RQ-1: Who is the main target segment (single store, minimarket, small chain) and how large is it?
- RQ-2: Which POS product do prospective customers use today and how much do they pay?
- RQ-3: Do customers require thermal receipt printing and barcode scanner integration from the first release?
- RQ-4: How often does the internet connection at target stores drop; is offline mode a purchase requirement?
- RQ-5: Which payment methods must be recorded (cash, QRIS, debit, e-wallet)?
- RQ-6: Is there a mandatory VAT/e-faktur requirement that the receipt must meet?

---

## 9. Assumptions and Constraints

### 9.1 Assumptions

| # | Assumption | Status |
|---|---|---|
| A1 | This document is documentation only; no code is changed. The stack migration is workstream S2, specified in FSD 2.7 | Established |
| A2 | The business rules of PRD §13 apply unchanged | Established |
| A3 | There are no production POS customers whose data must be migrated | `tbd` (no evidence in the repo of whether a production deployment exists; D-POS-13) |
| A4 | One tenant can have many stores; one cashier is bound to one store | Based on the cashier login code |
| A5 | Currency IDR, time zone WIB, Indonesian default and English supported (STACK-SPEC 19) | Established by the suite standard |

### 9.2 Constraints

- Small team: only one product alignment runs at a time (D12); POS (S2) runs after the SELA gate is closed and after the `kit` extraction (S1).
- The technical standard is non-negotiable per product: STACK-SPEC (FSD 2.1).
- No payment gateway (PRD §4).
- The document must not contain prices or business figures that are not in the repo.

### 9.3 Open decisions

| ID | Decision | Options / question | Interim default |
|---|---|---|---|
| D-POS-01 | VAT/tax on the receipt | Rate, inclusive or exclusive, per product or per store, e-faktur obligation | `tbd` |
| D-POS-02 | Receipt print medium | Thermal printer (ESC/POS), browser print, PDF, send via WhatsApp/email | `tbd` |
| D-POS-03 | Cashier offline scope | Sales queue only or also local product search; stock handling at sync; maximum offline duration | `tbd` |
| D-POS-04 | Payment methods | Set of methods, whether QRIS is recorded only; no payment gateway | `tbd` |
| D-POS-05 | Loyalty rules | Earn rate per spend value, redemption, expiry | `tbd` |
| D-POS-06 | Sale void/return | Void and return policy, who is authorised | `tbd` |
| D-POS-07 | Cashier discount authority | Percent/nominal limit per role, admin approval | `tbd` |
| D-POS-08 | UU PDP for customer data | Processing basis, retention, data subject rights, appointment of a responsible person | `tbd` (legal review) |
| D-POS-09 | Package tiers and prices | Values of `pos.stores.max`, `pos.devices.max`, free tier | `tbd` |
| D-POS-10 | Observability | OpenTelemetry/Prometheus are not in the SELA baseline (D11); optional `kit` module or not | `tbd` |
| D-POS-11 | Cookie domain and session TTL | Cookie `stery_session` (`suite` mode) and POS session (`standalone` mode) | `tbd` |
| D-POS-12 | Tenant onboarding in `standalone` mode | Operator console, self-service, or script | `tbd` |
| D-POS-13 | Migration of existing POS customer data | Whether production data exists; migration strategy if so | `tbd` |
| D-POS-14 | Report day boundary | Per-store time zone or fixed WIB | WIB (STACK-SPEC 19 standard), per store `tbd` |

---

## 10. Risks and Mitigation

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-POS-01 | Alignment to the SELA stack rewrites POS infrastructure and interfaces (gateway, inter-service gRPC, NATS, JWT, migrations, two web applications) | Long schedule; functional regressions | Keep domain and application; characterisation tests before the rewrite; one alignment in flight; effort S/M/L in FSD 2.7 [INFERENCE] |
| RISK-POS-02 | RLS regression when moving to `kit/tenantdb` (the current `SET LOCAL` mechanism is not yet proven to work because there are no integration tests) | Cross-tenant leakage or failure of all queries | Mandatory cross-tenant tests per module; `set_config(..., true)` per transaction; gate "Postgres integration tests run" |
| RISK-POS-03 | Duplicate sales or wrong stock because of a saga without idempotency | Financial and stock discrepancies | Sale as one DB transaction with an idempotency key (FSD 7.1) |
| RISK-POS-04 | Secrets in git-tracked files (`ecosystem.config.cjs`, `deploy/docker/init-postgres.sql`) and a service-token interceptor that fails open | Credential compromise; internal services exposed | Rotation, git history purge, deny-by-default (gate) |
| RISK-POS-05 | UI conversion from SSR (adapter-node, BFF) to a static SPA and merging two applications | UI, session, PWA regressions | Route groups `(admin)` and `(cashier)`; Playwright e2e |
| RISK-POS-06 | NATS JetStream semantics (durable, 7 days) replaced by a Postgres outbox | Lost or duplicate events | Outbox `FOR UPDATE SKIP LOCKED`, idempotent consumers, `processed_events` |
| RISK-POS-07 | VAT/receipt and UU PDP decisions delayed | The gate cannot be closed; compliance | Schedule legal review before S2 completes; D-POS-01, D-POS-08 |
| RISK-POS-08 | Solo team capacity | Excess WIP | D12 sequence; one alignment in flight |
| RISK-POS-09 | Status document drift (README, CLAUDE.md, PRD §10.1/§17, gRPC security document) versus code | Decisions based on stale claims | FSD 11.3 lists contradictions; code = truth |
| RISK-POS-10 | Prices and package tiers not yet set | Cannot be sold and limited | D-POS-09; dimensions fixed, values `tbd` |
| RISK-POS-11 | Dependency on the platform as a single point of failure (`suite` mode) | Cashier cannot log in or entitlement cannot be checked | 24-hour entitlement snapshot; `standalone` mode unaffected; offline cashier session `tbd` |
| RISK-POS-12 | Offline cashier causes sales to sync late with stale stock or prices | Negative stock or price differences | Conflict policy D-POS-03; sale ID from the client for dedup |
| RISK-POS-13 | Admin cannot use the UI because the store selector is absent and the gateway requires a store context on all routes | Admin features unusable [INFERENCE: static reading] | Store selector in `apps/web (admin)` in S2; tenant-level routes do not require a store |
| RISK-POS-14 | The dashboard shows fake numbers to the owner | Wrong business decisions | Gate: synthetic data removed |

---

## 11. KPIs

Definitions are fixed; all targets are `tbd` (the owner sets them; D-POS-09 for commercial ones). A baseline is listed only where it can be computed from the code.

| KPI | Definition | Current baseline | Target |
|---|---|---|---|
| Sale completion time | Median time from the "Pay" button to the receipt being displayed, per store | Not measured | `tbd` |
| Sale success rate | Completed sales / sale attempts, per day | Not measured | `tbd` |
| Duplicate sales | Number of sales with the same idempotency key stored more than once | No idempotency key | 0 (target value `tbd` if the owner changes it) |
| Stock accuracy | System stock vs physical count variance, per store per month | Not measured | `tbd` |
| Negative stock | Number of stock rows with quantity < 0 | Not guarded in SQL | 0 (`tbd`) |
| Loyalty consistency | Customer sales that produce exactly one points entry / all customer sales | 0 callers | `tbd` |
| Offline sync success | Queued sales synced without duplication / all queued sales | No queue | `tbd` |
| RLS WITH CHECK coverage | Policies with `WITH CHECK` / all tenant policies | 8 of 15 (static analysis) | 15 of 15 |
| CI health | Percentage of green CI runs on the main branch | Stale CI | `tbd` |
| Test coverage | Go and web statement coverage | Only stub-based unit tests (79 test functions) | 80% floor (STACK-SPEC 17 standard) |
| Tenant onboarding time | Time from contract until tenant, store, and first user are active | No path | `tbd` |
| Package adoption | Active tenants per tier; usage against `pos.stores.max` and `pos.devices.max` | No tracking | `tbd` |

---

## 12. Timeline and Phases

Calendar dates: `tbd` (the owner sets them; no date can be derived from the repo). The order follows decision D12 of Stery Suite.

| Phase | Contents | Prerequisite | Output |
|---|---|---|---|
| S0 | Documents (this document, FSD POS, analysis, suite BRD/PRD/FSD) and closing the SELA gate | — | Documents approved |
| S1 | Extraction of `kit` from SELA and platform IAM/ENT; POS not yet touched | S0 | `kit` available |
| S2 | **POS alignment** to STACK-SPEC (workstream in FSD 2.7) | S1 | POS stabilisation gate (below) |
| S3 | HRIS alignment (does not touch POS) | S2 | — |
| S4 | `suite` mode adapters in POS (identity, entitlement, notification, audit, events), billing and console | S3 | POS can run in `suite` mode |
| S5 | Cross-product integrations (INT-3 hypothesis) and new products | S4 | Only if demand is validated |

### 12.1 POS stabilisation gate (S2 exit criteria)

Eight items, all mandatory. Items that do not depend on the new stack (secret rotation, removal of fake data, recording decisions) can be done earlier on the current stack [INFERENCE].

| # | Gate item | Related BR | Verification method |
|---|---|---|---|
| G1 | Loyalty wired in-process after the sale commit, idempotent per sale | BR-POS-07 | Integration test: a customer sale produces exactly one points entry, even when repeated |
| G2 | REST surface parity with module operations (every business operation has a `/api/v1` route) | BR-POS-01, BR-POS-10 | FSD 6 parity table with no "Not started" rows |
| G3 | CI green with the SELA `check.sh` pattern (proto, gofmt, vet, golangci-lint, tests with coverage, svelte-check, vitest, real e2e; `-race` in CI) | BR-POS-14 | Green CI run on the gate commit |
| G4 | RLS `WITH CHECK` on every policy | BR-POS-02 | Migration test listing policies without `WITH CHECK` = 0 |
| G5 | Git-tracked secrets rotated and purged from history | BR-POS-13 | Secret scan on history is clean; old credentials no longer valid |
| G6 | Postgres integration tests run (including cross-tenant tests) | BR-POS-14, BR-POS-02 | Tests run in CI without skips |
| G7 | Fake dashboard data removed | BR-POS-08 | No synthetic constants on the dashboard page; complete product names |
| G8 | VAT and receipt decisions recorded | BR-POS-09, BR-POS-10 | D-POS-01 and D-POS-02 have status decided in the decision log |

---

## 13. Current Implementation Status

Summary per POS catalogue module (detail per FR in FSD 11.1). Status: Done, Partial, Not started, Pending Verification, Deferred. Readiness percentages are not stated unless marked `[INFERENCE]`.

| Module | Coverage | Status | Evidence |
|---|---|---|---|
| 01 AUTH | bcrypt password login with tenant resolution, 10-minute RS256 JWT, rotating refresh token with reuse detection, logout, three-role RBAC at the gateway. Not yet: tenant/store/user creation, admin store selector, refresh in REST and web | Partial | `services/auth-service/internal/application/commands/login.go` |
| 02 PRD (catalogue) | Create, update, get, search products. Not yet: SKU search, categories (table exists, no RPC), price history, UI form | Partial | `services/product-service/internal/infrastructure/postgres/product_repository.go` |
| 03 INV (inventory) | Add item, update stock, list, threshold, low-stock event. Not yet: non-negative stock guard, movement ledger, REST routes for update stock/threshold/get item | Partial | `services/inventory-service/internal/application/commands/update_stock.go` |
| 04 SAL (sales) | Sale with items, item discounts, total, snapshot receipt, `Sale.Completed` event. Not yet: atomicity, idempotency, VAT (PPN), payment method, receipt in UI, receipt history | Partial | `services/sales-service/internal/application/commands/create_sale.go` |
| 05 SUP (supplier/PO) | Add/list suppliers, create PO, receive stock. Not yet: partial and atomic receipt, PO cancellation, REST routes for receive and PO list | Partial | `services/supplier-service/internal/application/commands/receive_stock.go` |
| 06 CUS (customer/loyalty) | Create, get, list customers; `AddLoyaltyPoints` RPC. Not yet: caller of points after a sale, earn and redeem rules | Partial | `services/customer-service/internal/application/commands/add_loyalty_points.go` |
| 07 RPT (reports) | Sales report (revenue, transaction count, top products). Not yet: product name (always empty), payment mix, WIB day boundary; the dashboard fabricates data | Partial | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| 08 NTF (notification) | No service; the low-stock event is published without a consumer | Not started | `services/_shared/nats/streams.go` |
| 09 AI (forecasting) | No AI service; the `AI.OrderForecasted` subject is only a constant with no publisher | Not started | `services/_shared/nats/streams.go` |
| Cross-module: security | RLS `ENABLE` + `FORCE` on all tenant tables; 7 of 15 policies without `WITH CHECK`; service-token interceptor fails open; secrets in two tracked files | Partial | `services/_shared/middleware/grpc_interceptors.go` |
| Cross-module: quality | 79 unit test functions (stub-based), no integration tests; stale CI | Partial | `.github/workflows/ci.yml` |

---

## 14. Approval

| Name | Title | Signature | Date |
|---|---|---|---|
| | Product Owner | | |
| | Business Sponsor | | |
| | Engineering Lead | | |
