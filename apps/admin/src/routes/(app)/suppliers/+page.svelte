<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Button, Card, Table } from '@pos-stery/ui';

  interface Supplier { supplier_id: string; name: string; contact: string; phone: string; email: string; }

  let suppliers: Supplier[] = $state([]);
  let loading = $state(true);
  let error   = $state('');

  onMount(async () => {
    try {
      const res = await api.get<{ suppliers: Supplier[] }>('/suppliers', getToken() ?? undefined);
      suppliers = res.suppliers ?? [];
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Failed to load suppliers.';
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head><title>Suppliers — POS-Stery Admin</title></svelte:head>

<div class="page-header">
  <h1 class="page-title">Suppliers</h1>
  <Button onclick={() => {}}>+ Add Supplier</Button>
</div>

{#if error}
  <p class="error-msg" role="alert">{error}</p>
{:else}
  <Card>
    <Table headers={['Name', 'Contact', 'Phone', 'Email']} {loading}>
      {#each suppliers as s (s.supplier_id)}
        <tr>
          <td>{s.name}</td>
          <td>{s.contact}</td>
          <td>{s.phone}</td>
          <td>{s.email}</td>
        </tr>
      {:else}
        {#if !loading}<tr><td colspan="4" style="text-align:center;padding:2rem;color:var(--color-muted)">No suppliers.</td></tr>{/if}
      {/each}
    </Table>
  </Card>
{/if}

<style>
  .page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.5rem; }
  .page-title  { font-size: 1.5rem; font-weight: 700; }
  .error-msg   { color: var(--color-danger); }
</style>
