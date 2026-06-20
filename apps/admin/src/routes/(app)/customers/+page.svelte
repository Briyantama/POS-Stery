<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Alert, Card, Input, PageHeader, Table } from '@pos-stery/ui';

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

<PageHeader title="Customers" />

<Card>
  <div class="toolbar">
    <Input placeholder="Search customers…" bind:value={search} oninput={() => load()} />
  </div>

  {#if error}
    <Alert>{error}</Alert>
  {:else}
    <Table
      headers={['Name', 'Phone', 'Email', 'Loyalty Points']}
      {loading}
      isEmpty={customers.length === 0}
      empty="No customers found."
    >
      {#each customers as c (c.customer_id)}
        <tr>
          <td>{c.name}</td>
          <td>{c.phone ?? '—'}</td>
          <td>{c.email ?? '—'}</td>
          <td>{c.loyalty_points ?? 0}</td>
        </tr>
      {/each}
    </Table>
  {/if}
</Card>

<style>
  .toolbar { margin-bottom: var(--space-4, 1rem); max-width: 320px; }
</style>
