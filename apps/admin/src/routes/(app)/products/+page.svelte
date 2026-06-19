<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Button, Card, Table, Badge, Input, LoadingSpinner } from '@pos-stery/ui';

  interface Product {
    product_id: string; name: string; sku: string; base_price: number;
    sale_price: number; is_active: boolean; category_id: string;
  }

  let products: Product[] = $state([]);
  let loading = $state(true);
  let error   = $state('');
  let search  = $state('');

  async function load() {
    loading = true;
    try {
      const res = await api.get<{ products: Product[] }>(`/products?q=${encodeURIComponent(search)}`, getToken() ?? undefined);
      products = res.products ?? [];
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Failed to load products.';
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<svelte:head><title>Products — POS-Stery Admin</title></svelte:head>

<div class="page-header">
  <h1 class="page-title">Products</h1>
  <Button onclick={() => {}}>+ Add Product</Button>
</div>

<Card>
  <div class="toolbar">
    <Input placeholder="Search products…" bind:value={search} oninput={() => load()} />
  </div>

  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else}
    <Table headers={['SKU', 'Name', 'Base Price', 'Sale Price', 'Status']} {loading}>
      {#each products as p (p.product_id)}
        <tr>
          <td>{p.sku}</td>
          <td>{p.name}</td>
          <td>IDR {p.base_price.toLocaleString()}</td>
          <td>IDR {p.sale_price.toLocaleString()}</td>
          <td><Badge variant={p.is_active ? 'success' : 'default'}>{p.is_active ? 'Active' : 'Inactive'}</Badge></td>
        </tr>
      {:else}
        {#if !loading}<tr><td colspan="5" style="text-align:center;padding:2rem;color:var(--color-muted)">No products found.</td></tr>{/if}
      {/each}
    </Table>
  {/if}
</Card>

<style>
  .page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.5rem; }
  .page-title  { font-size: 1.5rem; font-weight: 700; }
  .toolbar     { margin-bottom: 1rem; max-width: 320px; }
  .error-msg   { color: var(--color-danger); }
</style>
