<script lang="ts">
  interface Props {
    variant?: 'error' | 'warning' | 'success' | 'info';
    children: import('svelte').Snippet;
  }
  let { variant = 'error', children }: Props = $props();
</script>

<p class="alert alert--{variant}" role="alert">{@render children()}</p>

<style>
  .alert {
    margin: 0;
    padding: 0.625rem 1rem;
    border-radius: var(--radius-md, 0.375rem);
    font-size: var(--text-sm, 0.875rem);
    border-left: 3px solid transparent;
  }
  .alert--error   { background: var(--color-danger-bg,  #f6e1dd); color: var(--color-danger-fg,  #8c2a20); border-color: var(--color-danger,   #c0392b); }
  .alert--warning { background: var(--color-warning-bg, #f8e9cb); color: var(--color-warning-fg, #7a4b08); border-color: var(--color-warning,  #b5710e); }
  .alert--success { background: var(--color-success-bg, #e1efe5); color: var(--color-success-fg, #1c5436); border-color: var(--color-success,  #2e7d52); }
  .alert--info    { background: var(--color-info-bg,    #e3e8f5); color: var(--color-info-fg,    #142e73); border-color: var(--color-info,     #1b3b8f); }
</style>
