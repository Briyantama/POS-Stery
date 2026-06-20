<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Alert, Button, Card, Input, PageHeader, StatCard, Table } from '@pos-stery/ui';

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

<PageHeader title="Sales Report">
  <div class="toolbar">
    <Input type="date" bind:value={date} />
    <Button onclick={load} {loading}>Load</Button>
  </div>
</PageHeader>

{#if error}
  <Alert>{error}</Alert>
{:else if report}
  <div class="stats-grid">
    <StatCard title="Total Sales" value={report.total_sales} />
    <StatCard title="Total Revenue" value="IDR {report.total_revenue.toLocaleString()}" />
  </div>

  <Card title="Top Products">
    <Table
      headers={['Product', 'Units Sold', 'Revenue']}
      isEmpty={(report.top_products ?? []).length === 0}
      empty="No data for this date."
    >
      {#each report.top_products ?? [] as p (p.product_id)}
        <tr>
          <td>{p.name}</td>
          <td>{p.quantity}</td>
          <td>IDR {p.revenue.toLocaleString()}</td>
        </tr>
      {/each}
    </Table>
  </Card>
{/if}

<style>
  .toolbar    { display: flex; gap: var(--space-3, 0.75rem); align-items: flex-end; }
  .stats-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: var(--space-4, 1rem); margin-bottom: var(--space-6, 1.5rem); }
</style>
