<script lang="ts">
  interface Props { size?: 'sm' | 'md' | 'lg'; label?: string; }
  let { size = 'md', label = 'Loading…' }: Props = $props();
</script>

<div class="spinner spinner--{size}" role="status" aria-label={label}>
  <span class="sr-only">{label}</span>
</div>

<style>
  .spinner {
    border: 3px solid var(--color-border, #e2e8f0);
    border-top-color: var(--color-primary, #2563eb);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }
  .spinner--sm { width: 1rem;  height: 1rem; }
  .spinner--md { width: 1.5rem; height: 1.5rem; }
  .spinner--lg { width: 2.5rem; height: 2.5rem; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; }
</style>
