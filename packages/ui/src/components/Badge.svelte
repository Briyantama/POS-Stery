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
    font-weight: var(--weight-medium, 500);
    letter-spacing: var(--tracking-wide, 0.08em);
    text-transform: uppercase;
    white-space: nowrap;
  }
  .badge--default { background: var(--stone-100, #f0ebe1); color: var(--stone-600, #5c5345); }
  .badge--success { background: var(--color-success-bg, #e1efe5); color: var(--color-success-fg, #1c5436); }
  .badge--warning { background: var(--color-warning-bg, #f8e9cb); color: var(--color-warning-fg, #7a4b08); }
  .badge--danger  { background: var(--color-danger-bg,  #f6e1dd); color: var(--color-danger-fg,  #8c2a20); }
  .badge--info    { background: var(--color-info-bg,    #e3e8f5); color: var(--color-info-fg,    #142e73); }
</style>
