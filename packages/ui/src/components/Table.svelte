<script lang="ts">
  interface Props {
    headers: string[];
    children: import('svelte').Snippet;
    empty?: string;
    isEmpty?: boolean;
    loading?: boolean;
  }
  let {
    headers,
    children,
    empty = 'No records found.',
    isEmpty = false,
    loading = false,
  }: Props = $props();
</script>

<div class="table-wrap">
  <table class="table">
    <thead>
      <tr>
        {#each headers as header}
          <th scope="col">{header}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#if loading}
        <tr><td colspan={headers.length} class="table__state">Loading…</td></tr>
      {:else if isEmpty}
        <tr><td colspan={headers.length} class="table__state">{empty}</td></tr>
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
    font-size: var(--text-sm, 0.875rem);
  }
  .table th {
    padding: 0.75rem 1rem;
    text-align: left;
    border-bottom: 1px solid var(--color-border, #e2e8f0);
    font-weight: 600;
    color: var(--color-muted, #64748b);
    font-size: var(--text-xs, 0.75rem);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .table :global(td) {
    padding: 0.75rem 1rem;
    text-align: left;
    border-bottom: 1px solid var(--color-border, #e2e8f0);
  }
  .table :global(tr:last-child td) { border-bottom: none; }
  .table :global(tr:hover td) { background: var(--color-surface-hover, #f8fafc); }
  .table__state {
    text-align: center;
    color: var(--color-muted, #64748b);
    padding: 2rem;
  }
</style>
