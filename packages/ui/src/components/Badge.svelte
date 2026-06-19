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
    border-radius: 9999px;
    font-size: 0.75rem;
    font-weight: 500;
  }
  .badge--default { background: #f1f5f9; color: #475569; }
  .badge--success { background: #dcfce7; color: #166534; }
  .badge--warning { background: #fef9c3; color: #854d0e; }
  .badge--danger  { background: #fee2e2; color: #991b1b; }
  .badge--info    { background: #dbeafe; color: #1d4ed8; }
</style>
