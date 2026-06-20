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
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-lg, 0.5rem);
    padding: var(--space-5, 1.25rem);
    display: flex;
    flex-direction: column;
    gap: var(--space-2, 0.5rem);
  }
  .statcard__title {
    font-size: var(--text-sm, 0.875rem);
    color: var(--color-muted, #7a7060);
    font-weight: var(--weight-medium, 500);
    text-transform: uppercase;
    letter-spacing: var(--tracking-wide, 0.08em);
    font-size: var(--text-xs, 0.75rem);
  }
  .statcard__value {
    font-family: var(--font-mono);
    font-size: var(--text-3xl, 1.875rem);
    font-weight: var(--weight-bold, 700);
    color: var(--color-primary, #1b3b8f);
    line-height: var(--leading-tight, 1.15);
    letter-spacing: var(--tracking-tight, -0.02em);
    font-variant-numeric: tabular-nums;
  }
  .statcard__sub {
    font-size: var(--text-xs, 0.75rem);
    color: var(--color-muted, #7a7060);
  }
</style>
