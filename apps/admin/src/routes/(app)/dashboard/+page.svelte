<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Alert, Badge, LoadingSpinner, PageHeader, StatCard } from '@pos-stery/ui';

  let loading = $state(true);
  let error   = $state('');
  let report: { total_sales: number; total_revenue: number; top_products: Array<{ name: string; quantity: number }> } | null = $state(null);

  const today = new Date().toISOString().slice(0, 10);

  onMount(async () => {
    try {
      report = await api.get(`/sales/report?date=${today}`, getToken() ?? undefined);
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Failed to load report.';
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head><title>Dashboard — POS-Stery Admin</title></svelte:head>

<PageHeader title="Dashboard" sub="Today — {today}" />

{#if loading}
  <div class="center"><LoadingSpinner size="lg" /></div>
{:else if error}
  <Alert>{error}</Alert>
{:else if report}
  <div class="stats-grid">
    <StatCard title="Today's Sales" value={report.total_sales ?? 0} />
    <StatCard title="Today's Revenue" value="IDR {(report.total_revenue ?? 0).toLocaleString()}" />
  </div>

  {#if report.top_products?.length}
    <section class="top-section">
      <h2 class="section-title">Top Products Today</h2>
      <ul class="top-list" role="list">
        {#each report.top_products as product}
          <li class="top-list__item">
            <span>{product.name}</span>
            <Badge variant="info">{product.quantity} sold</Badge>
          </li>
        {/each}
      </ul>
    </section>
  {/if}
{/if}

<style>
  .center      { display: flex; justify-content: center; padding: var(--space-12, 3rem); }
  .stats-grid  { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: var(--space-4, 1rem); margin-bottom: var(--space-6, 1.5rem); }

  .top-section {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg, 0.5rem);
    padding: var(--space-5, 1.25rem);
  }
  .section-title { font-size: var(--text-base, 1rem); font-weight: 600; margin-bottom: var(--space-4, 1rem); }
  .top-list      { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--space-2, 0.5rem); }
  .top-list__item { display: flex; align-items: center; justify-content: space-between; }
</style>
