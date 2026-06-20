<script lang="ts">
  import { onMount } from 'svelte';
  import { getToken } from '$lib/auth';
  import { api, ApiError } from '$lib/api';
  import { Alert, Button, Card, PageHeader, Table } from '@pos-stery/ui';

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

<PageHeader title="Suppliers">
  <Button onclick={() => {}}>+ Add Supplier</Button>
</PageHeader>

{#if error}
  <Alert>{error}</Alert>
{:else}
  <Card>
    <Table
      headers={['Name', 'Contact', 'Phone', 'Email']}
      {loading}
      isEmpty={suppliers.length === 0}
      empty="No suppliers yet."
    >
      {#each suppliers as s (s.supplier_id)}
        <tr>
          <td>{s.name}</td>
          <td>{s.contact}</td>
          <td>{s.phone}</td>
          <td>{s.email}</td>
        </tr>
      {/each}
    </Table>
  </Card>
{/if}
