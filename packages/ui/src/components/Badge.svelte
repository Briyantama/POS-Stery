<script lang="ts">
  interface Props {
    variant?: 'default' | 'success' | 'warning' | 'danger' | 'info';
    children: import('svelte').Snippet;
  }
  let { variant = 'default', children }: Props = $props();
</script>

<span class="badge badge--{variant}">{@render children()}</span>

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    padding: 0.125rem 0.5rem;
    border-radius: var(--radius-full, 9999px);
    font-size: var(--text-xs, 0.75rem);
    font-weight: 500;
  }
  .badge--default { background: var(--color-surface-hover, #f1f5f9); color: var(--color-muted, #475569); }
  .badge--success { background: var(--color-success-bg, #dcfce7); color: #166534; }
  .badge--warning { background: var(--color-warning-bg, #fef9c3); color: #854d0e; }
  .badge--danger  { background: var(--color-danger-bg,  #fee2e2); color: #991b1b; }
  .badge--info    { background: var(--color-info-bg,    #dbeafe); color: #1d4ed8; }
</style>
