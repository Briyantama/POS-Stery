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
  .alert--error   { background: var(--color-danger-bg,  #fee2e2); color: #991b1b; border-color: var(--color-danger,   #dc2626); }
  .alert--warning { background: var(--color-warning-bg, #fef9c3); color: #854d0e; border-color: var(--color-warning,  #d97706); }
  .alert--success { background: var(--color-success-bg, #dcfce7); color: #166534; border-color: var(--color-success,  #16a34a); }
  .alert--info    { background: var(--color-info-bg,    #dbeafe); color: #1d4ed8; border-color: var(--color-primary,  #2563eb); }
</style>
