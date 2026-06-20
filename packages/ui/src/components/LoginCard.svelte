<script lang="ts">
  interface Props {
    subtitle: string;
    error?: string;
    children: import('svelte').Snippet;
  }
  let { subtitle, error = '', children }: Props = $props();
</script>

<main class="login-page">
  <div class="login-card">
    <div class="login-brand">
      <strong class="login-brand__name">POS-Stery</strong>
      <span class="login-brand__sub">{subtitle}</span>
    </div>

    {@render children()}

    {#if error}
      <p class="login-error" role="alert">{error}</p>
    {/if}
  </div>
</main>

<style>
  .login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-surface-2, #f8fafc);
  }
  .login-card {
    background: var(--color-surface, #fff);
    border: 1px solid var(--color-border, #e2e8f0);
    border-radius: var(--radius-xl, 0.75rem);
    padding: var(--space-10, 2.5rem);
    width: 100%;
    max-width: 400px;
    display: flex;
    flex-direction: column;
    gap: var(--space-6, 1.5rem);
  }
  .login-brand { text-align: center; display: flex; flex-direction: column; gap: var(--space-1, 0.25rem); }
  .login-brand__name { font-size: var(--text-2xl, 1.5rem); font-weight: 700; color: var(--color-primary, #2563eb); }
  .login-brand__sub  { font-size: var(--text-sm, 0.875rem); color: var(--color-muted, #64748b); }
  .login-error { color: var(--color-danger, #dc2626); font-size: var(--text-sm, 0.875rem); margin: 0; }
</style>
