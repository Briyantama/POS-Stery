<script lang="ts">
  import { Card, PageHeader } from '@pos-stery/ui';
</script>

<svelte:head><title>Receipts — POS-Stery Cashier</title></svelte:head>

<PageHeader title="Receipts" />
<Card>
  <p class="placeholder">Receipt history will appear here. PDF export available in Phase 2.</p>
</Card>

<style>
  .placeholder { color: var(--color-muted); margin: 0; }
</style>
