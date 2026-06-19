<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Card, Table, Badge, Input, LoadingSpinner } from '@pos-stery/ui';

  interface StockItem {
    product_id: string; product_name: string; quantity: number;
    min_quantity: number; is_low_stock: boolean;
  }

  let items: StockItem[] = $state([]);
  let loading     = $state(true);
  let error       = $state('');
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

<div class="page-header">
  <h1 class="page-title">Inventory</h1>
  <label class="toggle">
    <input type="checkbox" bind:checked={lowStockOnly} onchange={load} />
    Low stock only
  </label>
</div>

{#if error}
  <p class="error-msg" role="alert">{error}</p>
{:else}
  <Card>
    <Table headers={['Product', 'Stock', 'Min. Stock', 'Status']} {loading}>
      {#each items as item (item.product_id)}
        <tr>
          <td>{item.product_name}</td>
          <td>{item.quantity}</td>
          <td>{item.min_quantity}</td>
          <td><Badge variant={item.is_low_stock ? 'warning' : 'success'}>{item.is_low_stock ? 'Low Stock' : 'OK'}</Badge></td>
        </tr>
      {:else}
        {#if !loading}<tr><td colspan="4" style="text-align:center;padding:2rem;color:var(--color-muted)">No inventory records.</td></tr>{/if}
      {/each}
    </Table>
  </Card>
{/if}

<style>
  .page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.5rem; }
  .page-title  { font-size: 1.5rem; font-weight: 700; }
  .toggle      { display: flex; align-items: center; gap: 0.5rem; font-size: 0.875rem; cursor: pointer; }
  .error-msg   { color: var(--color-danger); }
</style>
