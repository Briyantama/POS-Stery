<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Alert, Badge, Card, PageHeader, Table } from '@pos-stery/ui';

  interface StockItem {
    product_id: string; product_name: string; quantity: number;
    min_quantity: number; is_low_stock: boolean;
  }

  let items: StockItem[] = $state([]);
  let loading      = $state(true);
  let error        = $state('');
  let lowStockOnly = $state(false);

  async function load() {
    loading = true;
    try {
      const qs = lowStockOnly ? '?low_stock_only=true' : '';
      const res = await api.get<{ items: StockItem[] }>(`/inventory${qs}`, getToken() ?? undefined);
      items = res.items ?? [];
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Failed to load inventory.';
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<svelte:head><title>Inventory — POS-Stery Admin</title></svelte:head>

<PageHeader title="Inventory">
  <label class="toggle">
    <input type="checkbox" bind:checked={lowStockOnly} onchange={load} />
    Low stock only
  </label>
</PageHeader>

{#if error}
  <Alert>{error}</Alert>
{:else}
  <Card>
    <Table
      headers={['Product', 'Stock', 'Min. Stock', 'Status']}
      {loading}
      isEmpty={items.length === 0}
      empty="No inventory records."
    >
      {#each items as item (item.product_id)}
        <tr>
          <td>{item.product_name}</td>
          <td>{item.quantity}</td>
          <td>{item.min_quantity}</td>
          <td><Badge variant={item.is_low_stock ? 'warning' : 'success'}>{item.is_low_stock ? 'Low Stock' : 'OK'}</Badge></td>
        </tr>
      {/each}
    </Table>
  </Card>
{/if}

<style>
  .toggle { display: flex; align-items: center; gap: var(--space-2, 0.5rem); font-size: var(--text-sm, 0.875rem); cursor: pointer; }
</style>
