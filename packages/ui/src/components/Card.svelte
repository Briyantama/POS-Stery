<script lang="ts">
  interface Props {
    title?: string;
    children: import('svelte').Snippet;
    actions?: import('svelte').Snippet;
  }
  let { title, children, actions }: Props = $props();
</script>

<div class="card">
  {#if title || actions}
    <div class="card__header">
      {#if title}<h3 class="card__title">{title}</h3>{/if}
      {#if actions}<div class="card__actions">{@render actions()}</div>{/if}
    </div>
  {/if}
  <div class="card__body">{@render children()}</div>
</div>

<style>
  .card {
    background: var(--color-surface, #fff);
    border: 1px solid var(--color-border, #e2e8f0);
    border-radius: 0.5rem;
    overflow: hidden;
  }
  .card__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid var(--color-border, #e2e8f0);
  }
  .card__title { margin: 0; font-size: 1rem; font-weight: 600; }
  .card__body  { padding: 1.25rem; }
</style>
