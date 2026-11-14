<script lang="ts">
  interface Props {
    title: string;
    sub?: string;
    children?: import('svelte').Snippet;
  }
  let { title, sub, children }: Props = $props();
</script>

<div class="page-header">
  <div class="page-header__text">
    <h1 class="page-header__title">{title}</h1>
    {#if sub}<p class="page-header__sub">{sub}</p>{/if}
  </div>
  {#if children}
    <div class="page-header__actions">{@render children()}</div>
  {/if}
</div>

<style>
  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: var(--space-4, 1rem);
    margin-bottom: var(--space-6, 1.5rem);
  }
  .page-header__title {
    font-size: var(--text-2xl, 1.5rem);
    font-weight: 700;
    color: var(--color-text, #0f172a);
  }
  .page-header__sub {
    margin: 0.125rem 0 0;
    font-size: var(--text-sm, 0.875rem);
    color: var(--color-muted, #64748b);
  }
  .page-header__actions {
    display: flex;
    align-items: center;
    gap: var(--space-3, 0.75rem);
    flex-shrink: 0;
  }
</style>
