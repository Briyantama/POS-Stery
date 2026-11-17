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
    background: var(--color-surface-2, #f4f0e8);
  }
  .login-card {
    background: var(--color-surface, #fff);
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-xl, 0.75rem);
    box-shadow: var(--shadow-md);
    padding: var(--space-10, 2.5rem);
    width: 100%;
    max-width: 400px;
    display: flex;
    flex-direction: column;
    gap: var(--space-6, 1.5rem);
  }
  .login-brand { text-align: center; display: flex; flex-direction: column; gap: var(--space-1, 0.25rem); }
  .login-brand__name {
    font-size: var(--text-2xl, 1.5rem);
    font-weight: var(--weight-black, 800);
    color: var(--color-primary-ink, #11224f);
    letter-spacing: var(--tracking-tight, -0.02em);
  }
  .login-brand__sub  { font-size: var(--text-sm, 0.875rem); color: var(--color-muted, #7a7060); }
  .login-error { color: var(--color-danger-fg, #8c2a20); font-size: var(--text-sm, 0.875rem); margin: 0; }
</style>
