<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Card, Table, Input, Button } from '@pos-stery/ui';

  interface TopProduct { product_id: string; name: string; quantity: number; revenue: number; }
  interface Report { total_sales: number; total_revenue: number; top_products: TopProduct[]; }

  let date    = $state(new Date().toISOString().slice(0, 10));
  let report: Report | null = $state(null);
  let loading = $state(false);
  let error   = $state('');

  async function load() {
    loading = true;
    error   = '';
    try {
      report = await api.get<Report>(`/sales/report?date=${date}`, getToken() ?? undefined);
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Failed to load report.';
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<svelte:head><title>Reports — POS-Stery Admin</title></svelte:head>

<div class="page-header">
  <h1 class="page-title">Sales Report</h1>
  <div class="toolbar">
    <Input type="date" bind:value={date} />
    <Button onclick={load} {loading}>Load</Button>
  </div>
</div>

{#if error}
  <p class="error-msg" role="alert">{error}</p>
{:else if report}
  <div class="stats-grid">
    <Card title="Total Sales"><p class="stat">{report.total_sales}</p></Card>
    <Card title="Total Revenue"><p class="stat">IDR {report.total_revenue.toLocaleString()}</p></Card>
  </div>

  <Card title="Top Products">
    <Table headers={['Product', 'Units Sold', 'Revenue']}>
      {#each report.top_products ?? [] as p (p.product_id)}
        <tr>
          <td>{p.name}</td>
          <td>{p.quantity}</td>
          <td>IDR {p.revenue.toLocaleString()}</td>
        </tr>
      {:else}
        <tr><td colspan="3" style="text-align:center;padding:2rem;color:var(--color-muted)">No data.</td></tr>
      {/each}
    </Table>
  </Card>
{/if}

<style>
  .page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.5rem; flex-wrap: wrap; gap: 1rem; }
  .page-title  { font-size: 1.5rem; font-weight: 700; }
  .toolbar     { display: flex; gap: 0.75rem; align-items: flex-end; }
  .stats-grid  { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 1rem; margin-bottom: 1.5rem; }
  .stat        { font-size: 2rem; font-weight: 700; color: var(--color-primary); margin: 0; }
  .error-msg   { color: var(--color-danger); }
</style>
