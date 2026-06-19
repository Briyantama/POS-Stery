<script lang="ts">
  import { Card } from '@pos-stery/ui';
</script>

<svelte:head><title>Receipts — POS-Stery Cashier</title></svelte:head>

<h1 class="page-title">Receipts</h1>
<Card>
  <p class="placeholder">Receipt history will appear here. PDF export available in Phase 2.</p>
</Card>

<style>
  .page-title   { font-size: 1.25rem; font-weight: 700; margin-bottom: 1rem; }
  .placeholder  { color: var(--color-muted); }
</style>
