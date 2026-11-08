<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiError } from '$lib/api';
  import { Alert, Badge, Card, Input, PageHeader, Table } from '@pos-stery/ui';

  interface Product {
    id: string; name: string; sku: string; base_price: number;
    sale_price: number | null; is_active: boolean; category_id: string | null;
  }

  let products: Product[] = $state([]);
  let loading = $state(true);
  let error   = $state('');
  let search  = $state('');

  async function load() {
    loading = true;
    try {
      const res = await api.get<{ items: Product[] }>(`/products?q=${encodeURIComponent(search)}`);
      products = res.items ?? [];
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Failed to load products.';
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<svelte:head><title>Products — POS-Stery Admin</title></svelte:head>

<PageHeader title="Products" />

<Card>
  <div class="toolbar">
    <Input placeholder="Search products…" bind:value={search} oninput={() => load()} />
  </div>

  {#if error}
    <Alert>{error}</Alert>
  {:else}
    <Table
      headers={['SKU', 'Name', 'Base Price', 'Sale Price', 'Status']}
      {loading}
      isEmpty={products.length === 0}
      empty="No products found."
    >
      {#each products as p (p.id)}
        <tr>
          <td>{p.sku}</td>
          <td>{p.name}</td>
          <td>IDR {p.base_price.toLocaleString()}</td>
          <td>{p.sale_price == null ? '-' : `IDR ${p.sale_price.toLocaleString()}`}</td>
          <td><Badge variant={p.is_active ? 'success' : 'default'}>{p.is_active ? 'Active' : 'Inactive'}</Badge></td>
        </tr>
      {/each}
    </Table>
  {/if}
</Card>

<style>
  .toolbar { margin-bottom: var(--space-4, 1rem); max-width: 320px; }
</style>
