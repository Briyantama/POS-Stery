# Product

<!-- impeccable:product-schema 1 -->

> Inferred from `PRD.md`, `docs/` and the app code only. No interview round was possible (subagent run, no answer mechanism); every fact below is repo-sourced, and open decisions are listed explicitly.

## Platform

web

## Users

- **Cashier** (`apps/cashier`): works a checkout counter in a retail store, often under queue pressure. Searches products or scans barcodes, builds a cart, edits quantities, applies allowed discounts, takes payment (Cash, QRIS, Debit), shows or prints the receipt. Landscape tablet/terminal PWA (manifest `orientation: landscape`, standalone).
- **Tenant Admin** (`apps/admin`): manages tenant settings, stores, products, inventory, suppliers, pricing, customers and reports; receives low-stock notifications. Desktop dashboard with a per-store switcher.
- **Stock Manager** (PRD 5.3): monitors inventory, creates purchase orders, confirms replenishment. No dedicated screen exists yet in `apps/admin`.
- **Customer** (PRD 5.4): tracked for loyalty points only; not a UI user.

## Product Purpose

Multi-tenant POS SaaS for retail store chains: one platform for stores, products, inventory, sales, suppliers, customers and receipts, with AI-ready capabilities (forecasting, pricing suggestions, chat support, product recognition) planned in later phases. Success: checkout is fast, stock is accurate across stores, every sale and receipt is recorded, tenants never see each other's data.

## Positioning

A retail operating system sized for chains (tenant, then stores) rather than a single till: tenant and store scoping on every record, atomic stock deduction on sale, and AI services that suggest but never force changes in phase 1 (PRD 8.2).

## Operating Context

- Cashier flow is three stages in one screen: cart, tender, receipt. Receipt shows sale ID, line items, discount, total, tender, change.
- Currency is IDR with `id-ID` number formatting; payment methods Cash, QRIS, Debit; receipt closes with "terima kasih". Product copy in the apps is English.
- Admin shell: sidebar with Dashboard, Products, Inventory, Suppliers, Customers, Reports, plus a Store select (cookie forwarded as `X-Store-ID`).
- Both apps sit behind a server-side `/api` proxy; JWT lives in an httpOnly cookie; role (admin or cashier) is enforced server-side.
- Shared code: `packages/ui` (`@pos-stery/ui`) on top of `@stery/kit` tokens.

## Capabilities and Constraints

- Stack: SvelteKit / Svelte 5 apps, scoped component CSS, no Tailwind. PWA/offline is optional but strongly preferred for the cashier (PRD 9.3).
- Business rules relevant to UI: stock cannot go negative; a sale creates a receipt; loyalty points update only after a successful sale; manual discounts are controlled.
- Spec-driven: the PRD forbids inventing workflows or screens not written in it (PRD 20). UI work stays inside documented modules.
- Non-goals: marketplace, ERP/accounting, complex promotions, warehouse management, direct payment gateway in phase 1.
- Undecided: barcode-scanner integration depth and offline behavior (phase 2); purchase-order and AI screens are not built yet.

## Brand Commitments

Name **POS-Stery**, part of the Stery suite. The apps already carry an established identity (ink-cobalt primary, marigold accent, warm paper neutrals, mono figures) recorded in `DESIGN.md`; keep it.

## Evidence on Hand

- `PRD.md`, `docs/BRD_POS.en.md`, `docs/FSD_POS.en.md`, `docs/adr/` (present; PRD is the source used here).
- Working screens: admin dashboard, products, inventory, suppliers, customers, reports, login; cashier POS and login.
- Absent: customer testimonials, benchmarks, pricing, store photos or logos beyond the wordmark. Do not fabricate them.

## Product Principles

1. **Checkout speed first.** Every cashier interaction is judged by taps and seconds under queue pressure.
2. **Numbers are the interface.** Money, quantities and stock are read at a glance; they must be exact, aligned and unambiguous.
3. **Operate, do not decorate.** Familiar patterns, dense but calm screens, brand expressed through precise details.
4. **State is never silent.** Low stock, change due, failed sale, loading and empty states are always visible and recoverable.
5. **Stay inside the PRD.** No invented workflows or claims.

## Accessibility & Inclusion

No formal standard recorded. Working target: WCAG 2.2 AA, with touch-sized controls on the cashier terminal and keyboard operation for both apps.
