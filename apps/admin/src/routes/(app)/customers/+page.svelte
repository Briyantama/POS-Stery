<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiError } from '$lib/api';
  import { Alert, Button, Card, Input, PageHeader, Table } from '@pos-stery/ui';

  interface Customer { id: string; name: string; phone: string; email: string; loyalty_points: number; }

  let customers: Customer[] = $state([]);
  let loading = $state(true);
  let error   = $state('');
  let search  = $state('');

  async function load() {
    loading = true;
    error = '';
    try {
      const res = await api.get<{ items: Customer[] }>(`/customers?q=${encodeURIComponent(search)}`);
      customers = res.items ?? [];
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
    <Input ariaLabel="Search customers" placeholder="Search customers…" bind:value={search} oninput={() => load()} />
  </div>

  {#if error}
    <Alert>{error}</Alert>
    <Button variant="secondary" onclick={load}>Retry</Button>
  {:else}
    <Table
      headers={['Name', 'Phone', 'Email', 'Loyalty Points']}
      numeric={[3]}
      {loading}
      isEmpty={customers.length === 0}
      empty="No customers found."
    >
      {#each customers as c (c.id)}
        <tr>
          <td>{c.name}</td>
          <td>{c.phone ?? '—'}</td>
          <td>{c.email ?? '—'}</td>
          <td class="num">{c.loyalty_points ?? 0}</td>
        </tr>
      {/each}
    </Table>
  {/if}
</Card>

<style>
  .toolbar { margin-bottom: var(--space-4, 1rem); max-width: 320px; }
</style>
