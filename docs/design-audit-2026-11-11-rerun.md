# Design audit and critique RE-RUN, POS-Stery (apps/admin, apps/cashier, packages/ui)

Date: 2026-11-11. Skill: impeccable 4.8.2. Mode: Operate. Baseline: `docs/design-audit-2026-11-11.md` (not modified). Read-only run: no source edits, no build/lint/tests.

Run notes:
- `impeccable.cmd context` ran once (cwd = repo); existing PRODUCT.md and DESIGN.md used. (The launcher printed a harmless stale-engine warning for `windows-x64/impeccable.exe`.)
- Detector: `impeccable.cmd detect --json apps/admin/src apps/cashier/src packages/ui/src` exit 0, `[]` (also `[]` with `--no-config --no-design-system`). Not confirmed that the detector parses `.svelte` templates, so "0 findings" is weak evidence; all findings below are manual.
- No browser render and no touch/gesture test: contrast values are computed from tokens; layout and touch behavior are from source.
- Critique method: `Method: dual-agent (A: AuditPos.CritiqueA design review, no detector · B: AuditPos.CritiqueB detector)`. A and B ran as isolated sub-agents; A finished before detector output entered this context. Browser overlay not run (no servers started).

## Scores

| Measure | Baseline | Re-run | Delta |
|---|---|---|---|
| Audit health | 14/20 | **15/20** (Good) | +1 |
| Nielsen | 28/40 | **22/40** (Acceptable) | see note |

Nielsen note: the drop is method variance, not regression. The baseline was a same-context pass by the polisher; the re-run is an independent assessor that did not see the baseline, scored harder on unfixed items (no scanner path, sticky errors, no retry) and found two new functional defects. No heuristic got worse because of the polish pass.

## Part 1: Audit

| # | Dimension | Base | Now | Key finding |
|---|-----------|------|-----|-------------|
| 1 | Accessibility | 2 | 3 | Muted/placeholder/labels/aria-pressed fixed. Open: input edge contrast 1.2:1, no cart live region, ambiguous "Decrease/Increase" labels |
| 2 | Performance | 4 | 3 | Search fires an API call per keystroke, no debounce or stale-response guard (cashier search, admin lists) |
| 3 | Theming | 3 | 3 | Slate fallbacks (11 lines, 5 shared files) still present; PWA colors fixed |
| 4 | Responsive | 2 | 3 | Cashier scroll and admin narrow sidebar fixed; touch at 40px (not 44); manifest landscape-only |
| 5 | Implementation Integrity | 3 | 3 | Coherent, product-specific; isolated drifts (locale, Table alignment) |
| | **Total** | 14 | **15/20** | **Good (address weak dimensions)** |

Performance 4 to 3 is a stricter reading (per-keystroke request fan-out with out-of-order responses can show stale results at the till), not a regression.

### Implementation Integrity Verdict
Pass. Counter-craft system (warm stone ramp, ink cobalt, marigold marker, mono tabular figures, receipt tape) is applied through `--color-*` aliases; not interchangeable with a generic admin template. Detector: 0 findings.

### Findings (open and new)

**[P1] NEW. Admin load error is sticky: one failure hides the table until reload.**
- Location: `apps/admin/src/routes/(app)/products/+page.svelte:13,22-24` (`error` set, never reset in `load()`); same pattern `customers/+page.svelte:~19`, `inventory/+page.svelte:~23`, `suppliers/+page.svelte:~17`, `dashboard/+page.svelte:~32`.
- Impact: typing in the search retries the fetch, but the `{#if error}` branch still wins, so a transient failure kills the page. Principle 4 (state never silent, always recoverable).
- Fix: `error = ''` at the start of `load()`; add a Retry action next to `Alert`. Suggested: `/impeccable harden`.

**[P1] NEW. "Today" is computed in UTC, so it is yesterday before 07:00 WIB.**
- Location: `dashboard/+page.svelte:24`, `reports/+page.svelte:9` (`new Date().toISOString().slice(0, 10)`).
- Impact: day-open totals and the Reports default date are wrong for Indonesian stores in the morning; wrong money at day close.
- Fix: format the local date (`toLocaleDateString('en-CA')` or manual Y-M-D from local getters). Suggested: `/impeccable harden`.

**[P1] OPEN (was P1 critique #2). Barcode scanner / keyboard path missing.**
- Location: `apps/cashier/src/routes/(app)/pos/+page.svelte:143-150,56-66,68-77`.
- Impact: Enter does nothing; every scan needs a click; no debounce or stale-response guard (`searchProducts` per keystroke, `:147`); focus not returned to the field after add. PRD names scanning as core.
- Fix: Enter adds exact SKU match or first result; ~150ms debounce plus request token; refocus input after add and after New Sale. Suggested: `/impeccable harden`.

**[P1] OPEN (was P3/P1). Search failure and zero results are silent.**
- Location: `pos/+page.svelte:64` (`catch { searchResults = [] }`), `:152-166` (no branch for query with no match).
- Fix: "No match for 'x'" line and a retryable error line. Suggested: `/impeccable clarify`, `/impeccable onboard`.

**[P1] NEW. Change is the key output but the smallest figure on the receipt, and "Change due IDR 0" shows while cash is short.**
- Location: `pos/+page.svelte:366-369` and `.rct-tot-row--change` (`:671`, 12px); tender `:286-289`.
- Fix: large mono change line on the receipt; when short, show "Short by IDR n" and a reason under the disabled Confirm. Suggested: `/impeccable clarify`, `/impeccable typeset`.

**[P2] NEW. Text input edge is 1.2:1 (stone-200 on white); WCAG 1.4.11 needs 3:1 for field boundaries.**
- Location: `packages/ui/src/components/Input.svelte:63` (`border: 1px solid var(--color-border)`); DESIGN.md says input edges use stone-300 (`--color-border-strong`), which is itself 1.69:1.
- Fix: use `--color-border-strong` and darken it to about #8f8574 (3:1 on white) for inputs only, without renaming the variable. Suggested: `/impeccable polish`, `/impeccable colorize`.

**[P2] NEW. Cart changes not announced; stepper labels ambiguous.**
- Location: `pos/+page.svelte:197-199` (`aria-label="Decrease"`/`"Increase"` repeated per line, no item name; `aria-label` on non-interactive `.qty-val` span, still open from baseline P3), cart list has no live region (only the receipt has `aria-live`, `:313`).
- Fix: `aria-label="Decrease {item.name}"`, drop the span label or make it `role="status"`; visually hidden live total. Suggested: `/impeccable harden`.

**[P2] OPEN (baseline P2). Numeric table columns not right-aligned/mono; number locale inconsistent.**
- Location: `Table.svelte:56-60` (all cells left-aligned); figure cells `products/+page.svelte:53-54`, `inventory` Stock/Min, `reports:55-56`, `customers` Loyalty; `toLocaleString()` without locale `products:53-54`, `reports:43,56` vs `id-ID` in dashboard and cashier.
- Fix: numeric cell class (right, mono, tabular) and `id-ID` everywhere. Suggested: `/impeccable layout`.

**[P2] OPEN (baseline P1 critique #1). UUID text fields for Store ID and Customer ID.**
- Location: `apps/cashier/src/routes/(auth)/login/+page.svelte:44`, `pos/+page.svelte:234`.
- Needs API/product decision (store picker, phone lookup). Discount is unbounded and ungated (`:229-232`), total silently clamps to 0 (`:39`): cap at subtotal. Suggested: `/impeccable harden`, `/impeccable shape`.

**[P2] NEW. Destructive misses without undo: "−" at quantity 1 deletes the line; Sign out mid-sale drops the cart with no confirm.**
- Location: `pos:79-83`, `apps/cashier/src/routes/(app)/+layout.svelte:10-13`.
- Fix: confirm on logout when `cart.length`; hold removed line for undo. Suggested: `/impeccable harden`.

**[P3] OPEN. Slate fallback hexes** (`#e2e8f0`, `#2563eb`, `#0f172a`, `#64748b`, `#f8fafc`): `Card.svelte:23,32`, `LoadingSpinner.svelte:12,13`, `PageHeader.svelte:32,37`, `Table.svelte:49,51,59,62,65`. Inert while tokens load; replace with the warm values. `/impeccable polish`.

**[P3] OPEN. Touch targets 40px, not 44px; manifest has one 192px icon and landscape-only.** Residual from baseline (deliberate for the 384px cart column; icon needs a new asset).

**[P3] OPEN. Admin dashboard hides Payment Mix, Cash Drawer, Low Stock entirely when empty** (`dashboard:~64,116,138`): no "all stocked" reassurance.

**[P3] NEW. Cashier has no `<h1>`** (page starts at `h2`, `pos:138,176`); `Table` loading row has no `aria-busy`/live status (`Table.svelte:29`).

### False positive from the assessor
- "Placeholder 2.6:1" cited at `Input.svelte:72` refers to the CSS fallback `#a89e8b`; the live token `--color-placeholder` is `#7a7060` (4.87:1 on white), so not a defect.

### Positive findings
- Token discipline; no inline hex in markup (only `#fff` on the cashier topbar, `+layout.svelte:56`), no Tailwind.
- Warning/success large figures pass: change amount 20px bold at 3.28:1 (warning) and 4.24:1 (success), both above the 3:1 large-text bar; labels use `-fg`.
- Muted `#6e6455` is 5.81:1 on white, 4.76:1 on hover rows; Night Ink nav at 65% is 7.28:1.
- `aria-pressed` method group, `aria-current` nav, focus rings, reduced-motion rule (feedback kept), 40px cashier targets.

## Part 2: Critique

### Design Health Score (Nielsen)

| # | Heuristic | Base | Now | Key Issue |
|---|-----------|------|-----|-----------|
| 1 | Visibility of System Status | 3 | 2 | Search failure silent; no no-results state; cart changes not announced; admin error sticky |
| 2 | Match System / Real World | 3 | 3 | IDR, Cash/QRIS/Debit, "Charge" fit; cashier types UUIDs |
| 3 | User Control and Freedom | 3 | 2 | "−" at qty 1 deletes with no undo; logout drops cart without confirm |
| 4 | Consistency and Standards | 3 | 2 | `toLocaleString()` vs `id-ID`; two revenue treatments |
| 5 | Error Prevention | 3 | 2 | Discount unbounded/ungated; total clamps to 0 silently; Confirm disabled with no reason |
| 6 | Recognition Rather Than Recall | 2 | 3 | SKU/price/qty visible; UUID fields still hurt (scored on cart screen) |
| 7 | Flexibility and Efficiency | 2 | 1 | No Enter-to-add, no autofocus, no debounce, no shortcuts |
| 8 | Aesthetic and Minimalist Design | 4 | 3 | Calm and on brand; cart Total 16px vs 30px tender total |
| 9 | Error Recovery | 3 | 2 | Admin error never clears, no retry; dashboard no retry |
| 10 | Help and Documentation | 2 | 2 | Only a search hint |
| **Total** | | 28/40 | **22/40** | **Acceptable** |

### Design Specificity Verdict
LLM assessment: specific. Receipt tape, PAID stamp, dashed rules, quick-cash chips, change-due tint, mono tabular IDR could not transfer to an unrelated product. Generic seam: admin list pages (default tables, no ledger treatment).
Deterministic scan: 0 findings (see run notes; Svelte template coverage unconfirmed).

### Overall Impression
Identity and polish pass held; what remains is behavior, not paint. Biggest opportunity: make the till a till (scan, Enter, change large, recoverable errors).

### What's Working
1. Three-stage in-place checkout (cart, tender, receipt) keeps total, method and change together.
2. Receipt card as signature object.
3. Token and focus/aria discipline after the polish pass.

### Priority Issues
1. [P1] Scanner/keyboard path and per-keystroke search: `/impeccable harden`.
2. [P1] Silent search failure and zero results: `/impeccable clarify`.
3. [P1] Change figure undersized; false "Change due IDR 0" when short: `/impeccable clarify`, `/impeccable typeset`.
4. [P1] Admin sticky errors and UTC "today": `/impeccable harden`.
5. [P2] Admin tables break the Ledger Rule: `/impeccable layout`.

### Persona Red Flags
**Alex (queue-pressure cashier):** scan does nothing; mis-tapped "−" removes a line with no undo; change is 12px on the receipt; sign out has no confirm.
**Jordan (first-timer):** must paste a Store ID UUID; "Customer ID (optional)" UUID invites skipping loyalty.
**Sam (accessibility):** cart changes unannounced; three identical "Decrease" buttons; input boundary 1.2:1.
**Admin owner:** one failed fetch kills the page; morning reports show yesterday.

### Minor Observations
- Receipt "terima kasih" inside an English UI; confirm intent.
- Cashier nav has a single "New Sale" link that does not reset the sale; receipt has no print/share.
- Dashboard hero 4px rail (documented exception).

### Questions to Consider
- If the cashier cannot scan, see stock or print, is the screen a till or a form?
- Should the admin show the active store on every page header, since changing store reloads the page?

Questions skipped: delegated run with no answer mechanism; priorities above are the action list.

## Delta vs baseline

Baseline FIXED items verified present in current source (not re-opened): muted `#6e6455` (5.81:1 on white), placeholder token `#7a7060`, `ariaLabel` on search/date inputs (`pos:144`, `products`), cashier single-column scroll (`(app)/+layout.svelte:~131`), 40px cashier targets (`pos:480,494,581,535`; logout `min-height:2.5rem`), `aria-pressed` on methods, admin narrow-screen sidebar, gold rank contrast, cash-drawer shortfall color, PWA colors (`manifest.json` `#11224f`/`#f4f0e8`), reduced-motion in both `app.css`. All 10 baseline FIXED items hold.

| Baseline finding | Status |
|---|---|
| P1 muted text, placeholder, input names, cashier scroll, touch targets | FIXED, verified |
| P2 aria-pressed, admin sidebar, gold rank, drawer shortfall, PWA colors; P3 reduced motion | FIXED, verified |
| P3 slate fallbacks in `packages/ui` | STILL OPEN |
| P2 numeric columns / Table alignment | STILL OPEN |
| P2 number locale inconsistency | STILL OPEN |
| P3 empty states (dashboard, cashier zero results) | STILL OPEN (zero results raised to P1) |
| P3 `aria-label` on `.qty-val` | STILL OPEN |
| P3 40px vs 44px, 192px icon | STILL OPEN (deliberate / needs asset) |
| Critique P1 UUID fields, scanner flow | STILL OPEN |
| Critique P2 retry on admin list errors | STILL OPEN, broadened: error is sticky (new defect) |
| Critique P5 raw-IDR discount | STILL OPEN (now also: unbounded, ungated) |
| Dark theme | Still out of scope |

NEW this run: sticky admin error (P1), UTC "today" (P1), receipt change figure / false "Change due 0" (P1), input boundary contrast 1.2:1 (P2), cart not announced and ambiguous stepper labels (P2), no undo for line delete / logout without confirm (P2), per-keystroke search without debounce or stale guard (perf, within P1 above), no cashier `<h1>`, Table loading not announced (P3).

Counts now: P0 0, P1 5, P2 5, P3 4.

## Recommended actions
1. `/impeccable harden`: scanner Enter-to-add + debounce + refocus; clear admin `error` and add Retry; local-date "today"; logout confirm and line-delete undo; discount cap.
2. `/impeccable clarify`: search no-result/error lines; large change on receipt and "Short by IDR n"; Confirm-disabled reason.
3. `/impeccable layout`: numeric column alignment in `Table`, `id-ID` formatting.
4. `/impeccable onboard`: empty states for dashboard sections.
5. `/impeccable polish`: input boundary contrast (`Input.svelte`, `tokens.css` border-strong), slate fallbacks, stepper `aria-label` with item name, cart live region, cashier `<h1>`.

## Applied (2026-11-11)

| File | Change |
|---|---|
| `apps/admin/.../products`, `customers`, `inventory`, `suppliers`, `dashboard` `+page.svelte` | `load()` resets `error` first; `Retry` button next to the `Alert` (suppliers and dashboard refactored from inline `onMount` to a reusable `load`) |
| `apps/admin/.../dashboard`, `reports` `+page.svelte` | "Today" from `toLocaleDateString('en-CA')` (local) instead of `toISOString()` (UTC) |
| `apps/admin/.../products`, `customers`, `inventory`, `reports` `+page.svelte` | numeric columns via `numeric={[...]}` and `class="num"` cells; `id-ID` locale in products and reports |
| `packages/ui/src/components/Table.svelte` | optional `numeric?: number[]` prop: header right-aligned; `td.num` right, mono, tabular |
| `packages/ui/src/components/Input.svelte` | border uses new `--color-input-edge` (fallback `#8f8574`, 3:1 on white); new optional `onkeydown` prop |
| `packages/ui/src/tokens.css` | added `--color-input-edge: #8f8574`; existing `--color-*` aliases untouched (so `--color-border-strong` is unchanged) |
| `apps/cashier/.../pos/+page.svelte` | search: 150ms debounce + request token (stale responses dropped); Enter adds exact SKU match else first result; tender shows "Short by IDR n" (warning) instead of "Change due IDR 0" while cash is short; receipt Change line enlarged (20px mono bold, success-fg, ruled) |

## Skipped (2026-11-11)

- Cashier search no-result / error lines, refocus after add: not in the approved subset (copy decision).
- Cart live region, stepper `aria-label` with item name, `.qty-val` label, cashier `<h1>`, Table `aria-busy`: not in the approved subset.
- Logout confirm, line-delete undo, discount cap: product decisions.
- UUID Store/Customer ID fields: needs API / product decision.
- Slate fallback hexes, 44px targets, 192px icon, dashboard empty states: not in the approved subset (icon needs new asset).
