<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Card, Badge, LoadingSpinner } from '@pos-stery/ui';

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

<h1 class="page-title">Dashboard</h1>
<p class="page-subtitle">Today — {today}</p>

{#if loading}
  <div class="center"><LoadingSpinner size="lg" /></div>
{:else if error}
  <p class="error-msg" role="alert">{error}</p>
{:else if report}
  <div class="stats-grid">
    <Card title="Today's Sales">
      <p class="stat">{report.total_sales ?? 0}</p>
    </Card>
    <Card title="Today's Revenue">
      <p class="stat">IDR {(report.total_revenue ?? 0).toLocaleString()}</p>
    </Card>
  </div>

  {#if report.top_products?.length}
    <Card title="Top Products Today">
      <ul class="top-list">
        {#each report.top_products as product}
          <li class="top-list__item">
            <span>{product.name}</span>
            <Badge variant="info">{product.quantity} sold</Badge>
          </li>
        {/each}
      </ul>
    </Card>
  {/if}
{/if}

<style>
  .page-title    { font-size: 1.5rem; font-weight: 700; margin-bottom: 0.25rem; }
  .page-subtitle { color: var(--color-muted); margin: 0 0 1.5rem; }
  .center        { display: flex; justify-content: center; padding: 3rem; }
  .error-msg     { color: var(--color-danger); }
  .stats-grid    { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 1rem; margin-bottom: 1.5rem; }
  .stat          { font-size: 2rem; font-weight: 700; color: var(--color-primary); margin: 0; }
  .top-list      { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 0.5rem; }
  .top-list__item { display: flex; align-items: center; justify-content: space-between; }
</style>
