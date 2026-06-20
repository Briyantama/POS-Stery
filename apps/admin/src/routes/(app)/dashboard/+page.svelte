<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Alert, Badge, LoadingSpinner } from '@pos-stery/ui';

  interface TopProduct { name: string; quantity: number; revenue?: number; }
  interface Report {
    total_sales: number;
    total_revenue: number;
    top_products: TopProduct[];
    low_stock?: Array<{ name: string; sku: string; quantity: number }>;
    payment_mix?: { cash: number; qris: number; debit: number };
    cash_drawer?: { opening_float: number; expected: number; actual?: number };
  }

  let loading = $state(true);
  let error   = $state('');
  let report: Report | null = $state(null);

  const today = new Date().toISOString().slice(0, 10);

  onMount(async () => {
    try {
      report = await api.get<Report>(`/sales/report?date=${today}`, getToken() ?? undefined);
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Failed to load report.';
    } finally {
      loading = false;
    }
  });

  function fmtIDR(n: number) {
    return 'IDR ' + n.toLocaleString('id-ID');
  }

  /* Mini sparkline — 7 synthetic daily bars derived from total (demo shape) */
  function sparkBars(total: number): number[] {
    if (!total) return [0, 0, 0, 0, 0, 0, 0];
    const weights = [0.09, 0.11, 0.14, 0.13, 0.17, 0.16, 0.20];
    return weights.map(w => Math.round(w * total));
  }

  const bars = $derived(sparkBars(report?.total_revenue ?? 0));
  const maxBar = $derived(Math.max(...bars, 1));

  /* Payment mix totals — fallback to even split if API doesn't return them */
  const payMix = $derived(
    report?.payment_mix ?? {
      cash:  Math.round((report?.total_revenue ?? 0) * 0.55),
      qris:  Math.round((report?.total_revenue ?? 0) * 0.30),
      debit: Math.round((report?.total_revenue ?? 0) * 0.15),
    }
  );

  const drawer = $derived(
    report?.cash_drawer ?? {
      opening_float: 500_000,
      expected: 500_000 + (payMix.cash ?? 0),
    }
  );
</script>

<svelte:head><title>Dashboard — POS-Stery Admin</title></svelte:head>

{#if loading}
  <div class="center"><LoadingSpinner size="lg" /></div>
{:else if error}
  <Alert>{error}</Alert>
{:else if report}
  <!-- ── Hero revenue panel ─────────────────────────────────────── -->
  <div class="hero-panel">
    <div class="hero-rail"></div>
    <div class="hero-body">
      <div class="hero-left">
        <span class="hero-label">Today's Revenue</span>
        <span class="hero-value">{fmtIDR(report.total_revenue ?? 0)}</span>
        <span class="hero-sub">{report.total_sales ?? 0} sales · {today}</span>
      </div>
      <div class="hero-spark" aria-hidden="true">
        {#each bars as h, i}
          <div
            class="spark-bar"
            class:spark-bar--last={i === bars.length - 1}
            style="height:{Math.round((h / maxBar) * 52)}px"
          ></div>
        {/each}
      </div>
    </div>
  </div>

  <!-- ── Operations row: Payment Mix + Cash Drawer ─────────────── -->
  <div class="ops-row">
    <div class="ops-card">
      <h2 class="ops-title">Payment Mix</h2>
      <dl class="ops-dl">
        <div class="ops-entry">
          <dt>Cash</dt>
          <dd class="ops-amount">{fmtIDR(payMix.cash)}</dd>
        </div>
        <div class="ops-entry">
          <dt>QRIS</dt>
          <dd class="ops-amount">{fmtIDR(payMix.qris)}</dd>
        </div>
        <div class="ops-entry">
          <dt>Debit</dt>
          <dd class="ops-amount">{fmtIDR(payMix.debit)}</dd>
        </div>
      </dl>
    </div>

    <div class="ops-card">
      <h2 class="ops-title">Cash Drawer</h2>
      <dl class="ops-dl">
        <div class="ops-entry">
          <dt>Opening float</dt>
          <dd class="ops-amount">{fmtIDR(drawer.opening_float)}</dd>
        </div>
        <div class="ops-entry">
          <dt>Expected</dt>
          <dd class="ops-amount">{fmtIDR(drawer.expected)}</dd>
        </div>
        {#if drawer.actual != null}
          <div class="ops-entry ops-entry--balance" class:ops-entry--ok={drawer.actual >= drawer.expected}>
            <dt>Balance</dt>
            <dd class="ops-amount">{fmtIDR(drawer.actual - drawer.expected)}</dd>
          </div>
        {/if}
      </dl>
    </div>
  </div>

  <!-- ── Bottom row: Top Products + Low Stock ──────────────────── -->
  <div class="bottom-row">
    {#if report.top_products?.length}
      <section class="ledger-card">
        <header class="ledger-header">
          <h2 class="ledger-title">Top Products Today</h2>
        </header>
        <ol class="ledger-list" role="list">
          {#each report.top_products.slice(0, 5) as product, i}
            <li class="ledger-row">
              <span class="ledger-rank" class:ledger-rank--gold={i === 0}>
                #{String(i + 1).padStart(2, '0')}
              </span>
              <span class="ledger-name">{product.name}</span>
              <span class="ledger-qty">{product.quantity} sold</span>
              {#if product.revenue != null}
                <span class="ledger-rev">{fmtIDR(product.revenue)}</span>
              {/if}
            </li>
          {/each}
        </ol>
      </section>
    {/if}

    {#if report.low_stock?.length}
      <section class="ledger-card">
        <header class="ledger-header">
          <h2 class="ledger-title">Low Stock Watch</h2>
        </header>
        <ul class="ledger-list" role="list">
          {#each report.low_stock as item}
            <li class="ledger-row">
              <span class="ledger-sku">{item.sku}</span>
              <span class="ledger-name">{item.name}</span>
              <Badge variant="warning">{item.quantity} left</Badge>
            </li>
          {/each}
        </ul>
      </section>
    {/if}
  </div>
{/if}

<style>
  .center { display: flex; justify-content: center; padding: var(--space-12, 3rem); }

  /* ── Hero revenue panel ── */
  .hero-panel {
    display: flex;
    background: var(--color-surface, #fff);
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-lg, 0.5rem);
    overflow: hidden;
    margin-bottom: var(--space-5, 1.25rem);
  }
  .hero-rail {
    width: 4px;
    flex-shrink: 0;
    background: var(--color-primary, #1b3b8f);
  }
  .hero-body {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    flex: 1;
    padding: var(--space-6, 1.5rem);
    gap: var(--space-6, 1.5rem);
  }
  .hero-left { display: flex; flex-direction: column; gap: var(--space-2, 0.5rem); }
  .hero-label {
    font-size: var(--text-xs, 0.75rem);
    font-weight: var(--weight-medium, 500);
    text-transform: uppercase;
    letter-spacing: var(--tracking-wide, 0.08em);
    color: var(--color-muted, #7a7060);
  }
  .hero-value {
    font-family: var(--font-mono);
    font-size: var(--text-4xl, 2.5rem);
    font-weight: var(--weight-black, 800);
    color: var(--color-primary, #1b3b8f);
    line-height: var(--leading-tight, 1.15);
    letter-spacing: var(--tracking-tight, -0.02em);
    font-variant-numeric: tabular-nums;
  }
  .hero-sub { font-size: var(--text-sm, 0.875rem); color: var(--color-muted, #7a7060); }

  /* Mini 7-bar sparkline */
  .hero-spark {
    display: flex;
    align-items: flex-end;
    gap: 3px;
    height: 52px;
    flex-shrink: 0;
  }
  .spark-bar {
    width: 10px;
    min-height: 4px;
    background: var(--color-primary-light, #e3e8f5);
    border-radius: 2px 2px 0 0;
    transition: height 300ms ease;
  }
  .spark-bar--last { background: var(--color-primary, #1b3b8f); }

  /* ── Operations row ── */
  .ops-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-5, 1.25rem);
    margin-bottom: var(--space-5, 1.25rem);
  }
  @media (max-width: 640px) { .ops-row { grid-template-columns: 1fr; } }

  .ops-card {
    background: var(--color-surface, #fff);
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-lg, 0.5rem);
    padding: var(--space-5, 1.25rem);
  }
  .ops-title {
    font-size: var(--text-sm, 0.875rem);
    font-weight: var(--weight-semibold, 600);
    color: var(--color-text, #1a1611);
    margin-bottom: var(--space-4, 1rem);
    text-transform: uppercase;
    letter-spacing: var(--tracking-wide, 0.08em);
    font-size: var(--text-xs, 0.75rem);
    color: var(--color-muted, #7a7060);
  }
  .ops-dl { margin: 0; display: flex; flex-direction: column; gap: var(--space-2, 0.5rem); }
  .ops-entry {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    font-size: var(--text-sm, 0.875rem);
    padding: var(--space-2, 0.5rem) 0;
    border-bottom: 1px solid var(--color-border, #e2dbcd);
  }
  .ops-entry:last-child { border-bottom: none; }
  .ops-entry dt { color: var(--color-muted, #7a7060); }
  .ops-amount {
    font-family: var(--font-mono);
    font-weight: var(--weight-semibold, 600);
    font-variant-numeric: tabular-nums;
    color: var(--color-text, #1a1611);
  }
  .ops-entry--balance .ops-amount { color: var(--color-muted, #7a7060); }
  .ops-entry--ok .ops-amount { color: var(--color-success, #2e7d52); }

  /* ── Bottom row: ledger cards ── */
  .bottom-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-5, 1.25rem);
  }
  @media (max-width: 768px) { .bottom-row { grid-template-columns: 1fr; } }

  .ledger-card {
    background: var(--color-surface, #fff);
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-lg, 0.5rem);
    overflow: hidden;
  }
  .ledger-header {
    padding: var(--space-4, 1rem) var(--space-5, 1.25rem);
    border-bottom: 1px solid var(--color-border, #e2dbcd);
  }
  .ledger-title {
    font-size: var(--text-sm, 0.875rem);
    font-weight: var(--weight-semibold, 600);
    color: var(--color-text, #1a1611);
    margin: 0;
  }

  .ledger-list { list-style: none; margin: 0; padding: 0; }
  .ledger-row {
    display: flex;
    align-items: center;
    gap: var(--space-3, 0.75rem);
    padding: var(--space-3, 0.75rem) var(--space-5, 1.25rem);
    border-bottom: 1px solid var(--color-border, #e2dbcd);
    font-size: var(--text-sm, 0.875rem);
  }
  .ledger-row:last-child { border-bottom: none; }

  .ledger-rank {
    font-family: var(--font-mono);
    font-size: var(--text-xs, 0.75rem);
    color: var(--color-muted, #7a7060);
    width: 2.25rem;
    flex-shrink: 0;
    font-variant-numeric: tabular-nums;
  }
  .ledger-rank--gold { color: var(--color-accent, #e0992e); font-weight: var(--weight-bold, 700); }

  .ledger-name { flex: 1; color: var(--color-text, #1a1611); }
  .ledger-sku {
    font-family: var(--font-mono);
    font-size: var(--text-xs, 0.75rem);
    color: var(--color-muted, #7a7060);
    width: 5rem;
    flex-shrink: 0;
  }
  .ledger-qty {
    font-family: var(--font-mono);
    font-size: var(--text-xs, 0.75rem);
    color: var(--color-muted, #7a7060);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  .ledger-rev {
    font-family: var(--font-mono);
    font-size: var(--text-xs, 0.75rem);
    color: var(--color-primary, #1b3b8f);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
</style>
