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
  {disabled}
  aria-disabled={disabled || loading}
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
    border-radius: 0.375rem;
    font-weight: 500;
    cursor: pointer;
    transition: background-color 150ms, opacity 150ms;
  }
  .btn:disabled,
  .btn--loading { opacity: 0.6; cursor: not-allowed; }

  .btn--sm  { padding: 0.25rem 0.75rem; font-size: 0.75rem; }
  .btn--md  { padding: 0.5rem 1.25rem;  font-size: 0.875rem; }
  .btn--lg  { padding: 0.75rem 1.5rem;  font-size: 1rem; }

  .btn--primary   { background: var(--color-primary, #2563eb); color: #fff; }
  .btn--secondary { background: var(--color-surface-2, #f1f5f9); color: var(--color-text, #0f172a); }
  .btn--danger    { background: var(--color-danger, #dc2626); color: #fff; }
  .btn--ghost     { background: transparent; color: var(--color-primary, #2563eb); }

  .btn__spinner {
    width: 1em; height: 1em;
    border: 2px solid currentColor;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
