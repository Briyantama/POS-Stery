---
name: POS-Stery
description: "Counter craft: warm paper, ink-cobalt, marigold signal, mono-figure ledger."
colors:
  primary: "#1b3b8f"
  primary-hover: "#142e73"
  primary-light: "#e3e8f5"
  primary-ink: "#11224f"
  accent: "#e0992e"
  accent-hover: "#c8821c"
  accent-light: "#f8ebd2"
  paper-canvas: "#f4f0e8"
  surface: "#ffffff"
  surface-hover: "#eee8dc"
  stone-50: "#f7f4ee"
  stone-100: "#f0ebe1"
  stone-200: "#e2dbcd"
  stone-300: "#cfc6b5"
  stone-400: "#a89e8b"
  stone-500: "#7a7060"
  stone-600: "#5c5345"
  stone-700: "#423b30"
  stone-800: "#2a2520"
  stone-900: "#1a1611"
  text: "#1a1611"
  muted: "#7a7060"
  on-ink: "#f0ebe1"
  success: "#2e7d52"
  success-bg: "#e1efe5"
  success-fg: "#1c5436"
  warning: "#b5710e"
  warning-bg: "#f8e9cb"
  warning-fg: "#7a4b08"
  danger: "#c0392b"
  danger-bg: "#f6e1dd"
  danger-fg: "#8c2a20"
typography:
  display:
    fontFamily: "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace"
    fontSize: "2.5rem"
    fontWeight: 800
    lineHeight: 1.15
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 700
    lineHeight: 1.15
    letterSpacing: "-0.02em"
  title:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif"
    fontSize: "1rem"
    fontWeight: 600
    lineHeight: 1.5
  body:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 600
    letterSpacing: "0.08em"
rounded:
  sm: "4px"
  md: "6px"
  lg: "8px"
  xl: "12px"
  full: "9999px"
spacing:
  1: "4px"
  2: "8px"
  3: "12px"
  4: "16px"
  5: "20px"
  6: "24px"
  8: "32px"
  10: "40px"
  12: "48px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "#ffffff"
    rounded: "{rounded.md}"
    padding: "8px 20px"
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
  button-secondary:
    backgroundColor: "{colors.stone-100}"
    textColor: "{colors.text}"
    rounded: "{rounded.md}"
    padding: "8px 20px"
  button-danger:
    backgroundColor: "{colors.danger}"
    textColor: "#ffffff"
    rounded: "{rounded.md}"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text}"
    rounded: "{rounded.md}"
    padding: "8px 12px"
  badge-success:
    backgroundColor: "{colors.success-bg}"
    textColor: "{colors.success-fg}"
    rounded: "{rounded.full}"
    padding: "2px 8px"
  card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
    padding: "20px"
  sidebar:
    backgroundColor: "{colors.primary-ink}"
    textColor: "{colors.on-ink}"
---

# Design System: POS-Stery

Source of truth: `@stery/kit` variables (`--primary`, `--accent`, `--background`, ...) are given POS brand values in each app's `src/app.css`; `packages/ui/src/tokens.css` aliases them to `--color-*` names and adds the stone ramp, type scale, spacing, radius and shadows. Components in `packages/ui/src/components` and the app pages consume only the `--color-*` / `--space-*` / `--text-*` aliases. Light theme only (kit's `data-theme="dark"` is never set).

## Overview

**Creative North Star: "Counter Craft"**

A well-kept shop counter: warm paper, an ink-blue ledger, a brass-marigold bell. Screens read like a printed till roll and day book. Figures are set in monospace with tabular numerals so columns of money line up, labels are small tracked uppercase, and chrome stays quiet so the numbers carry the page.

Density is moderate and calm. Operators glance, tap, and move on; nothing animates for its own sake. Depth comes from 1px stone borders and tonal layers (white panels on warm canvas), not shadows.

**Key Characteristics:**
- Warm stone neutrals instead of cool grey; ink-cobalt for action and totals; marigold only as a rare signal.
- Mono tabular figures for every price, quantity, SKU and total.
- Uppercase 12px tracked micro-labels for panel titles and stat captions.
- Borders over shadows; dashed rules evoke receipt tape.
- System font stack (no webfont load) for speed at the counter.

## Colors

Warm paper and ink with one signal color.

### Primary
- **Ink Cobalt** (#1b3b8f): primary buttons, totals, prices, links to action, focus ring base. Hover is **Deep Cobalt** (#142e73). Tint **Cobalt Wash** (#e3e8f5) for selected states and the "total due" panel.
- **Night Ink** (#11224f): sidebar (admin) and topbar (cashier) fill only.

### Secondary
- **Marigold** (#e0992e): live dot in the wordmark and the active-nav rail/underline. Tint #f8ebd2, hover #c8821c.

### Neutral
- **Paper Canvas** (#f4f0e8): app background. **Surface** (#ffffff): panels, cards, inputs.
- **Stone ramp** 50 #f7f4ee (receipt card) to 900 #1a1611 (text ink). Borders use stone-200 (#e2dbcd); strong borders and input edges stone-300 (#cfc6b5).
- **Muted** (#7a7060): secondary text and captions. **On Ink** (#f0ebe1): text on Night Ink, dimmed to 65% for inactive nav.

### Status (each with bg/fg pair)
- Success #2e7d52 / #e1efe5 / #1c5436; Warning #b5710e / #f8e9cb / #7a4b08; Danger #c0392b / #f6e1dd / #8c2a20; Info = primary.

### Named Rules
**The One Signal Rule.** Marigold appears only as a navigation or presence marker. It is never a button fill, a large surface, or body text.
**The Pair Rule.** A status color on a tinted surface uses the matching `-fg` for text and the saturated hue only for borders, icons and large figures.

## Typography

**Sans (UI):** system stack `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif`.
**Mono (figures):** `ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace`.

**Character:** plain, fast, neutral UI voice; the mono face is the ledger's voice and is reserved for money, quantities, SKUs, IDs.

### Hierarchy
- **Display** (800, 2.5rem mono, 1.15, -0.02em): the dashboard revenue figure.
- **Headline** (700, 1.5rem, tight): page titles (`PageHeader`), login wordmark (800).
- **Stat/Total** (700-800, 1.875rem mono): StatCard values, tender "total due".
- **Title** (600, 1rem): card titles, receipt store name.
- **Body** (400-500, 0.875rem, 1.5): UI default; tables, inputs, buttons.
- **Label** (500-600, 0.75rem, +0.08em, uppercase): panel titles, stat captions, table headers (0.05em), badges.

### Named Rules
**The Ledger Rule.** Every number a user might compare is mono with `font-variant-numeric: tabular-nums`.

## Layout

- **Admin:** fixed 232px Night Ink sidebar (sticky, full height) + fluid content with 2rem padding; dashboard stacks hero panel, a two-up operations row (collapses at 640px), and a two-up ledger row (collapses at 768px). Tables scroll horizontally inside their card.
- **Cashier:** full-height shell (`100dvh`): 52px topbar over a two-column POS grid, products `1fr` and cart `384px`; collapses to one column at 768px. The POS page owns its own padding.
- Spacing is a 4px scale (`--space-1` to `--space-12`); panels use 16-20px padding, groups 8-12px gaps.

## Elevation & Depth

Flat by default. Hierarchy is tonal (white panel on paper canvas) plus 1px stone-200 borders; `--shadow-sm` and `--shadow-md` exist but only the login card uses a shadow. Focus is a 3px soft cobalt ring (`--ring-color`: rgb(27 59 143 / 0.18)).

**The Flat-By-Default Rule.** Surfaces carry no shadow at rest; only floating or isolated surfaces (login card) earn one.

## Shapes

Small, quiet radii: 4px chips and steppers, 6px buttons and inputs, 8px cards and panels, 12px login card, full pill for badges and quick-cash chips. Borders are 1px; the totals block uses a 2px top rule and receipt lines use dashed rules. No colored side stripes on cards; the only rails are the 4px cobalt hero rail and the 3px marigold active-nav marker.

## Components

### Buttons
Medium radius (6px), weight 500. Primary Ink Cobalt fill with white text; secondary stone-100 with a stone-200 border; danger #c0392b; ghost cobalt text with wash hover. Sizes sm/md/lg (0.75/0.875/1rem). Hover darkens a step; active nudges down 1px; focus is the cobalt ring; disabled and loading drop to 60% opacity with a spinner.

### Inputs / Fields
Surface fill, stone-200 border, 6px radius, 14px text, label above in 500 weight. Focus: cobalt border + ring. Error: danger border, `-fg` message below.

### Badges
Pill, 12px uppercase tracked, tinted bg + `-fg` text (success, warning, danger, info, default stone).

### Cards / Containers
`Card`, `StatCard`, ledger cards: white, 1px stone-200, 8px radius, 20px padding, no shadow. StatCard: tracked caption over mono 30px cobalt value.

### Navigation
Admin sidebar and cashier topbar on Night Ink; links 14px medium at 65% On Ink, full On Ink + subtle white wash when active, marigold 3px rail (sidebar) or underline (topbar). Wordmark "POS-Stery" 800 weight with a marigold dot. Sign out is an outlined ghost button.

### Tables
`Table`: uppercase 12px muted headers, 1px row rules, warm hover row, centered empty/loading row.

### Receipt tape (signature)
The cashier's cart rows (dashed dividers, mono figures) and receipt card (stone-50, centered store header, dashed `- - -` rules, PAID stamp in success green, "terima kasih" footer) are the product's signature: they should feel like thermal-paper output.

### Cashier tender controls
Method buttons (Cash/QRIS/Debit) as an equal-width segmented group with cobalt-wash selected state; quick-cash pill chips; "Change due" row in success tint, switching to warning tint while tendered cash is short.

## Do's and Don'ts

### Do:
- **Do** use the `--color-*`, `--space-*`, `--text-*`, `--radius-*` aliases; add new values to `packages/ui/src/tokens.css`, never inline hex.
- **Do** set money, quantities and SKUs in mono with tabular numerals.
- **Do** keep touch controls on the cashier at 40px or larger.
- **Do** give every state (loading, empty, error, disabled, change due) a visible treatment.

### Don't:
- **Don't** introduce Tailwind, utility classes, or a second styling system; scoped component CSS only.
- **Don't** rename `--color-*` aliases or the kit variables they read; HRIS, SELA and kit consumers depend on the shared names.
- **Don't** use marigold as a fill or for text on white.
- **Don't** add shadows to panels, colored side-borders to cards, or gradient text.
- **Don't** use cool slate greys; all neutrals come from the warm stone ramp.
