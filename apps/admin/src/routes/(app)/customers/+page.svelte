<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Card, Table, Input } from '@pos-stery/ui';

  interface Customer { customer_id: string; name: string; phone: string; email: string; loyalty_points: number; }

  let customers: Customer[] = $state([]);
  let loading = $state(true);
  let error   = $state('');
  let search  = $state('');

  async function load() {
    loading = true;
    try {
      const res = await api.get<{ customers: Customer[] }>(`/customers?q=${encodeURIComponent(search)}`, getToken() ?? undefined);
      customers = res.customers ?? [];
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Failed to load customers.';
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<svelte:head><title>Customers — POS-Stery Admin</title></svelte:head>

<h1 class="page-title">Customers</h1>

<Card>
  <div class="toolbar"><Input placeholder="Search customers…" bind:value={search} oninput={() => load()} /></div>

  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else}
    <Table headers={['Name', 'Phone', 'Email', 'Loyalty Points']} {loading}>
      {#each customers as c (c.customer_id)}
        <tr>
          <td>{c.name}</td>
          <td>{c.phone ?? '—'}</td>
          <td>{c.email ?? '—'}</td>
          <td>{c.loyalty_points ?? 0}</td>
        </tr>
      {:else}
        {#if !loading}<tr><td colspan="4" style="text-align:center;padding:2rem;color:var(--color-muted)">No customers.</td></tr>{/if}
      {/each}
    </Table>
  {/if}
</Card>

<style>
  .page-title { font-size: 1.5rem; font-weight: 700; margin-bottom: 1.5rem; }
  .toolbar    { margin-bottom: 1rem; max-width: 320px; }
  .error-msg  { color: var(--color-danger); }
</style>
