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
    font-weight: 500;
    cursor: pointer;
    transition: background-color 150ms, opacity 150ms;
  }
  .btn:focus-visible {
    outline: 2px solid var(--color-primary, #2563eb);
    outline-offset: 2px;
  }
  .btn:disabled,
  .btn--loading { opacity: 0.6; cursor: not-allowed; }

  .btn--sm  { padding: 0.25rem 0.75rem; font-size: var(--text-xs,  0.75rem); }
  .btn--md  { padding: 0.5rem 1.25rem;  font-size: var(--text-sm,  0.875rem); }
  .btn--lg  { padding: 0.75rem 1.5rem;  font-size: var(--text-base, 1rem); }

  .btn--primary   { background: var(--color-primary, #2563eb); color: #fff; }
  .btn--primary:hover:not(:disabled) { background: var(--color-primary-hover, #1d4ed8); }
  .btn--secondary { background: var(--color-surface-2, #f1f5f9); color: var(--color-text, #0f172a); }
  .btn--secondary:hover:not(:disabled) { background: var(--color-surface-hover, #e2e8f0); }
  .btn--danger    { background: var(--color-danger, #dc2626); color: #fff; }
  .btn--danger:hover:not(:disabled) { opacity: 0.9; }
  .btn--ghost     { background: transparent; color: var(--color-primary, #2563eb); }
  .btn--ghost:hover:not(:disabled) { background: var(--color-primary-light, #dbeafe); }

  .btn__spinner {
    width: 1em; height: 1em;
    border: 2px solid currentColor;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
    flex-shrink: 0;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
