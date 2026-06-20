<script lang="ts">
  interface Props {
    title: string;
    value: string | number;
    sub?: string;
  }
  let { title, value, sub }: Props = $props();
</script>

<div class="statcard">
  <span class="statcard__title">{title}</span>
  <span class="statcard__value">{value}</span>
  {#if sub}<span class="statcard__sub">{sub}</span>{/if}
</div>

<style>
  .statcard {
    background: var(--color-surface, #fff);
    border: 1px solid var(--color-border, #e2e8f0);
    border-radius: var(--radius-lg, 0.5rem);
    padding: var(--space-5, 1.25rem);
    display: flex;
    flex-direction: column;
    gap: var(--space-2, 0.5rem);
  }
  .statcard__title {
    font-size: var(--text-sm, 0.875rem);
    color: var(--color-muted, #64748b);
    font-weight: 500;
  }
  .statcard__value {
    font-size: var(--text-3xl, 1.875rem);
    font-weight: 700;
    color: var(--color-primary, #2563eb);
    line-height: 1;
  }
  .statcard__sub {
    font-size: var(--text-xs, 0.75rem);
    color: var(--color-muted, #64748b);
  }
</style>
