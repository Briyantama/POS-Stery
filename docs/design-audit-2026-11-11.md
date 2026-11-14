# Design audit and critique, POS-Stery (apps/admin, apps/cashier, packages/ui)

Date: 2026-11-11. Skill: impeccable 4.8.2. Mode: Operate. Scope: both SvelteKit apps and the shared `@pos-stery/ui` components. Source-level review only: no browser render was run, so contrast values are computed from tokens, and touch-gesture behavior is untested.

Detector: `impeccable detect --json apps/admin/src apps/cashier/src packages/ui/src` returned `[]` before and after polish (no deterministic findings). Findings below are manual.

## Part 1: Audit (technical)

### Audit Health Score

| # | Dimension | Score | Key Finding |
|---|-----------|-------|-------------|
| 1 | Accessibility | 2 | Muted text 4.1-4.3:1 on paper surfaces; placeholder 2.65:1; unlabelled search inputs; sub-30px cashier tap targets |
| 2 | Performance | 4 | System fonts, no heavy effects, no images |
| 3 | Theming | 3 | Strong token use; off-brand slate fallbacks and PWA colors |
| 4 | Responsive Design | 2 | Cashier single-column mode cannot scroll; admin sidebar has no narrow layout |
| 5 | Implementation Integrity | 3 | Coherent, product-specific system; small isolated drifts |
| **Total** | | **14/20** | **Good (address weak dimensions)** |

### Implementation Integrity Verdict

Pass. The "Counter craft" system (warm stone ramp, ink cobalt, marigold marker, mono tabular figures, receipt-tape cart and receipt) is applied consistently through `--color-*` aliases over kit variables, and is not interchangeable with a generic admin template. Detector: 0 findings.

### Findings

Status: FIXED (polish applied), SKIPPED (reason given).

**[P1] Muted text fails AA on paper surfaces. FIXED**
- Location: `--muted-foreground` (#7a7060) in `apps/admin/src/app.css`, `apps/cashier/src/app.css`; consumed as `--color-muted` everywhere (stat captions, SKU, table headers, hints).
- Impact: 4.28:1 on canvas #f4f0e8, 4.10:1 on stone-100, 3.99:1 on hover rows; 12px captions are small text. WCAG 1.4.3.
- Fix applied: value changed to #6e6455 (5.11 / 4.89 / 4.76:1). Variable name unchanged. `stone-500` in the ramp keeps #7a7060.

**[P1] Placeholder contrast 2.65:1. FIXED**
- Location: `--color-placeholder` (#a89e8b) in `packages/ui/src/tokens.css`.
- Fix applied: #7a7060 (4.87:1 on white).

**[P1] Search/date inputs have no accessible name. FIXED**
- Location: cashier product search; admin products, customers search; reports date input. Placeholder is the only label; the date input has none.
- Standard: WCAG 3.3.2 / 4.1.2.
- Fix applied: `Input` gained an optional `ariaLabel` prop (applied only when no visible `label`); used on the four inputs. Visible copy unchanged.

**[P1] Cashier single-column layout cannot scroll. FIXED**
- Location: `apps/cashier/src/routes/(app)/+layout.svelte` (`.content { overflow: hidden }`) vs `pos/+page.svelte` (`@media (max-width: 768px) { height: auto; overflow: auto }`).
- Impact: below 768px the cart and products are clipped; the checkout is unreachable.
- Fix applied: `.content { overflow-y: auto }` inside the same breakpoint.

**[P1] Cashier touch targets too small for a counter terminal. FIXED**
- Location: `pos/+page.svelte` `.qty-btn` (26px), `.receipt-line__remove` (about 20x20), `.quick-chip` (about 28px tall), `.back-btn` (text only); layout `.topbar__logout` (about 30px).
- Impact: mis-taps on quantity and removal change a sale. WCAG 2.5.8 (24px) was met only by spacing; 44px is the recommended touch size.
- Fix applied: 40px (2.5rem) hit areas for all five. Remove button uses negative margin so row height is unchanged. Not raised to 44px to keep the 384px cart column from wrapping; residual P3.

**[P2] Payment method group exposes no selected state to assistive tech. FIXED**
- Location: `pos/+page.svelte` `.method-btn`. Selection was visual only.
- Fix applied: `aria-pressed` on each button.

**[P2] Admin sidebar has no narrow-screen layout and no focus ring on links. FIXED**
- Location: `apps/admin/src/routes/(app)/+layout.svelte`. Fixed 232px sticky sidebar at all widths; nav links and store select rely on the browser outline on a dark fill.
- Fix applied: at 768px and below the sidebar becomes a static top block with a wrapping row of links and 1rem content padding; links and select get the same white focus ring as Sign out.

**[P2] "Gold" rank text 2.4:1. FIXED**
- Location: `dashboard/+page.svelte` `.ledger-rank--gold` (marigold #e0992e on white, 12px). Breaks the One Signal Rule in DESIGN.md.
- Fix applied: text uses `--color-warning-fg` (#7a4b08), bold.

**[P2] Cash drawer shortfall is not flagged. FIXED**
- Location: `dashboard/+page.svelte` `.ops-entry--balance`. A negative balance was rendered in muted grey, the same as neutral data; only a non-negative balance turned green.
- Fix applied: balance defaults to `--color-danger-fg`; the existing `--ok` rule still wins for balanced or over. No markup or copy change.

**[P2] Off-brand PWA colors. FIXED**
- Location: `apps/cashier/src/app.html` `theme-color` (#2563eb) and `static/manifest.json` (`theme_color` #2563eb, `background_color` #f8fafc): cool slate blue that appears nowhere in the UI.
- Fix applied: #11224f (Night Ink, matches the topbar) and #f4f0e8 (paper canvas).

**[P3] Reduced-motion not honored. FIXED**
- Location: both `app.css`. All transitions are 90-140ms hover/press feedback; spinners are unaffected.
- Fix applied: `@media (prefers-reduced-motion: reduce)` shortens transition duration to 1ms (color and border state changes remain), disables smooth scrolling. Spinners keep rotating, since they are status, not decoration.

### SKIPPED findings

| Finding | Reason |
|---|---|
| **[P3] Slate fallback hexes in `packages/ui` components** (`#e2e8f0`, `#2563eb`, `#0f172a`, `#64748b`, `#f8fafc` in Card, Table, PageHeader, LoadingSpinner) | Inert: tokens are always defined by app.css, so fallbacks never render. Cosmetic cleanup of 11 lines in 5 shared files; skipped to avoid churn. |
| **[P2] Numeric table columns not right-aligned / mono** (Base Price, Sale Price, Stock, Revenue in `Table` users) | Needs a `Table` API change (per-column alignment for headers and cells). Contradicts "keep behavior"; propose as `/impeccable layout` follow-up. |
| **[P2] Number formatting inconsistent** (`toLocaleString()` with no locale in products/reports vs `id-ID` in dashboard and cashier) | Changes displayed output (separator characters), so it is a behavior/copy change, not polish. |
| **[P3] Admin dashboard hides Payment Mix / Cash Drawer entirely when the API omits them; cashier search shows nothing when a query has zero results** | Requires new empty-state copy; copy is out of scope. |
| **[P3] `aria-label` on non-interactive `.qty-val` span** (ignored by AT) | Behavior-neutral but needs markup decision (live region vs removing); low impact. |
| **[P3] Touch targets stay at 40px, not 44px** | Deliberate: wider targets would wrap the 384px cart column. |
| **[P3] Cashier `dist` icon is a single 192px `favicon.png` in the manifest** | Needs a new asset; not available. |
| **Dark theme** | Kit ships `data-theme="dark"`, POS never sets it; adding it is a design decision, not polish. |

### Patterns and systemic issues

- Muted/placeholder contrast was a token-level miss, fixed once at the source rather than per component.
- Interactive controls built as raw buttons in the cashier page (qty, chip, back, remove) did not share the `Button` sizing, so each drifted below touch size.

### Positive findings

- Token discipline is high: pages and components consume `--color-*`, `--space-*`, `--text-*`, `--radius-*` with matching fallbacks; no Tailwind, no inline hex in markup.
- Mono tabular figures used for all cashier money values and admin stats.
- Good state coverage: `Button` loading/disabled/focus, `Table` loading/empty, `Alert` role, change-due tint switching, `aria-current` on nav, server-side role guard.
- Borders over shadows, receipt-tape metaphor, and restrained use of marigold give the product a recognizable voice.

## Part 2: Critique (design review)

Method: `⚠️ DEGRADED: single-context (delegated worker; assessments A and B run inline, B = detector output above, which was reviewed after the manual pass)`.

### Design Health Score (Nielsen)

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | Processing label and spinner on charge; admin pages show only a bare "Loading..." row |
| 2 | Match System / Real World | 3 | Till-roll metaphor and IDR formatting fit; "Customer ID (optional)" asks for a UUID, which a cashier cannot know |
| 3 | User Control and Freedom | 3 | Back from tender, remove item; no way to cancel a sale or hold a cart |
| 4 | Consistency and Standards | 3 | Shared components strong; number formatting differs between pages |
| 5 | Error Prevention | 3 | Confirm disabled until cash covers total; discount can exceed subtotal (clamped silently) |
| 6 | Recognition Rather Than Recall | 2 | Store ID is a raw UUID text field at cashier login; customer lookup is by UUID |
| 7 | Flexibility and Efficiency | 2 | No keyboard shortcut or Enter-to-add for scanner input; first search hit is not auto-selectable |
| 8 | Aesthetic and Minimalist Design | 4 | Quiet, purposeful, no decoration |
| 9 | Error Recovery | 3 | Sale errors shown above Confirm with retry; list-page load errors replace the table with no retry |
| 10 | Help and Documentation | 2 | Search hint only; no empty-state guidance |
| **Total** | | **28/40** | **Good** |

### Design Specificity Verdict

LLM assessment: product-specific. The ledger and receipt vocabulary, ink sidebar with marigold marker, and mono figures would not transfer to an unrelated product unchanged. The weakest generic area is the admin list pages (card + table + search), which are functional but have no ledger treatment of their numbers.
Deterministic scan: 0 findings.

### Overall Impression

A confident, quiet system with real identity. The biggest opportunity is cashier speed: scan-and-go entry, larger touch affordances, and replacing UUID fields with lookups.

### What's Working
- The three-stage cart/tender/receipt flow keeps total due, method and change in one place, with change-due tinting that signals short cash.
- The receipt card (dashed rules, PAID stamp, mono figures) makes the end of a sale feel finished.
- Admin dashboard puts revenue first, then payment mix and cash drawer, then top products and low stock: the day-close order.

### Priority Issues

1. **[P1] Cashier login and cart use UUID text fields (Store ID, Customer ID).** Why: recall burden at the counter, high error rate. Fix: store picker populated from the cashier's assigned stores; customer lookup by phone. Needs API/product decision. Suggested: `/impeccable clarify`, `/impeccable harden`.
2. **[P1] Barcode scanner flow is not first-class.** Why: PRD names scanning as core. Fix: Enter in the search field adds the exact SKU/barcode match; keep focus in the field after add. Behavior change, out of polish. Suggested: `/impeccable harden`.
3. **[P2] Admin list pages lack ledger treatment and retry.** Why: numbers unaligned, errors dead-end. Fix: right-aligned mono numeric columns, Retry action in `Alert`. Suggested: `/impeccable layout`.
4. **[P2] Zero-result and empty-state guidance missing in cashier search and dashboard sections.** Why: silent blank panels read as bugs. Suggested: `/impeccable onboard`, `/impeccable clarify`.
5. **[P3] Discount entered in raw IDR only.** Why: percent discounts are common; PRD calls for controlled discounts, so a policy decision is needed first. Suggested: `/impeccable shape`.

### Persona Red Flags

**Jordan (first-time cashier):** must paste a Store ID UUID to sign in; no hint where to find it. "Customer ID (optional)" invites skipping loyalty.
**Alex (power user at a busy till):** scanner input needs a click per item; no shortcut to Charge. 26px steppers (now 40px) caused mis-taps.
**Sam (accessibility):** search inputs had no accessible name; method selection was not announced; muted text was below AA (all fixed).

### Minor Observations
- Receipt "terima kasih" footer is Indonesian inside an English UI; confirm that is intended.
- `Button` renders `aria-disabled` plus `disabled`; redundant but harmless.
- Page `<title>`s are consistent ("X — POS-Stery Admin/Cashier").

### Questions to Consider
- Should the cashier be Indonesian-first (receipt footer, "Cash/QRIS/Debit" labels already local)?
- What would the cart look like if the first search result were always one keystroke from added?
- Does the admin need a store-scoped hero (selected store) instead of tenant-wide wording?

Questions skipped: none could be asked (no answer mechanism in this delegated run); priorities above are the inherited action list.

### Recommended actions (remaining)
1. `/impeccable harden`: scanner Enter-to-add, UUID field replacements, retry on list errors.
2. `/impeccable layout`: numeric column alignment, `Table` alignment API.
3. `/impeccable clarify` and `/impeccable onboard`: empty and zero-result states.
4. `/impeccable polish` after the above.

## Polish applied (files edited)

- `packages/ui/src/tokens.css`, `packages/ui/src/components/Input.svelte`
- `apps/admin/src/app.css`, `apps/admin/src/routes/(app)/+layout.svelte`, `.../dashboard/+page.svelte`, `.../products/+page.svelte`, `.../customers/+page.svelte`, `.../reports/+page.svelte`
- `apps/cashier/src/app.css`, `apps/cashier/src/app.html`, `apps/cashier/static/manifest.json`, `apps/cashier/src/routes/(app)/+layout.svelte`, `.../pos/+page.svelte`

No CSS variable renamed; no copy, behavior, or dependency change.
