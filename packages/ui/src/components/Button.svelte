<script lang="ts">
  interface Props {
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
    size?: 'sm' | 'md' | 'lg';
    disabled?: boolean;
    loading?: boolean;
    type?: 'button' | 'submit' | 'reset';
    onclick?: () => void;
    children: import('svelte').Snippet;
  }

  let {
    variant = 'primary',
    size = 'md',
    disabled = false,
    loading = false,
    type = 'button',
    onclick,
    children,
  }: Props = $props();
</script>

<button
  {type}
  disabled={disabled || loading}
  aria-disabled={disabled || loading}
  aria-busy={loading}
  onclick={onclick}
  class="btn btn--{variant} btn--{size}"
  class:btn--loading={loading}
>
  {#if loading}
    <span class="btn__spinner" aria-hidden="true"></span>
  {/if}
  {@render children()}
</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    border: none;
    border-radius: var(--radius-md, 0.375rem);
    font-family: var(--font-sans);
    font-weight: var(--weight-medium, 500);
    cursor: pointer;
    transition: background-color 140ms, border-color 140ms, opacity 140ms, transform 90ms;
  }
  .btn:active:not(:disabled):not(.btn--loading) {
    transform: translateY(1px);
  }
  .btn:focus-visible {
    outline: none;
    box-shadow: 0 0 0 3px var(--ring-color, rgb(27 59 143 / 0.18));
  }
  .btn:disabled,
  .btn--loading { opacity: 0.6; cursor: not-allowed; }

  .btn--sm  { padding: 0.25rem 0.75rem; font-size: var(--text-xs,  0.75rem); }
  .btn--md  { padding: 0.5rem 1.25rem;  font-size: var(--text-sm,  0.875rem); }
  .btn--lg  { padding: 0.75rem 1.5rem;  font-size: var(--text-base, 1rem); }

  .btn--primary   { background: var(--color-primary, #1b3b8f); color: var(--color-on-primary, #fff); }
  .btn--primary:hover:not(:disabled):not(.btn--loading) { background: var(--color-primary-hover, #142e73); }
  .btn--secondary { background: var(--stone-100, #f0ebe1); color: var(--color-text, #1a1611); border: 1px solid var(--color-border, #e2dbcd); }
  .btn--secondary:hover:not(:disabled):not(.btn--loading) { background: var(--color-surface-hover, #eee8dc); }
  .btn--danger    { background: var(--color-danger, #c0392b); color: #fff; }
  .btn--danger:hover:not(:disabled):not(.btn--loading) { background: var(--color-danger-fg, #8c2a20); }
  .btn--ghost     { background: transparent; color: var(--color-primary, #1b3b8f); }
  .btn--ghost:hover:not(:disabled):not(.btn--loading) { background: var(--color-primary-light, #e3e8f5); }

  .btn__spinner {
    width: 1em; height: 1em;
    border: 2px solid currentColor;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.65s linear infinite;
    flex-shrink: 0;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
