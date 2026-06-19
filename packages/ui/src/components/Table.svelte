<script lang="ts">
  interface Props {
    headers: string[];
    children: import('svelte').Snippet;
    empty?: string;
    loading?: boolean;
  }
  let { headers, children, empty = 'No records found.', loading = false }: Props = $props();
</script>

<div class="table-wrap">
  <table class="table">
    <thead>
      <tr>
        {#each headers as header}
          <th>{header}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#if loading}
        <tr><td colspan={headers.length} class="table__state">Loading…</td></tr>
      {:else}
        {@render children()}
      {/if}
    </tbody>
  </table>
</div>

<style>
  .table-wrap { overflow-x: auto; }
  .table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.875rem;
  }
  .table th,
  .table :global(td) {
    padding: 0.75rem 1rem;
    text-align: left;
    border-bottom: 1px solid var(--color-border, #e2e8f0);
  }
  .table th {
    font-weight: 600;
    color: var(--color-muted, #64748b);
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .table :global(tr:hover td) { background: var(--color-surface-hover, #f8fafc); }
  .table__state { text-align: center; color: var(--color-muted, #64748b); padding: 2rem; }
</style>
