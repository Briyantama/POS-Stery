<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, ApiError } from '$lib/api';
  import { setSession } from '$lib/auth';
  import { Button, Input } from '@pos-stery/ui';

  let email    = $state('');
  let password = $state('');
  let error    = $state('');
  let loading  = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;
    try {
      const res = await api.post<{ token: string; role: string }>('/login', { email, password });
      if (res.role !== 'admin') {
        error = 'Admin access only.';
        return;
      }
      setSession(res.token, res.role);
      goto('/dashboard');
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Login failed.';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head><title>Sign in — POS-Stery Admin</title></svelte:head>

<main class="login-page">
  <div class="login-card">
    <div class="login-brand">
      <h1>POS-Stery</h1>
      <p>Admin Dashboard</p>
    </div>

    <form onsubmit={handleSubmit}>
      <Input label="Email" type="email" bind:value={email} required />
      <Input label="Password" type="password" bind:value={password} required />

      {#if error}
        <p class="login-error" role="alert">{error}</p>
      {/if}

      <Button type="submit" {loading} disabled={loading}>Sign in</Button>
    </form>
  </div>
</main>

<style>
  .login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-surface-2);
  }
  .login-card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 0.75rem;
    padding: 2.5rem;
    width: 100%;
    max-width: 400px;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }
  .login-brand { text-align: center; }
  .login-brand h1 { font-size: 1.5rem; font-weight: 700; color: var(--color-primary); }
  .login-brand p  { color: var(--color-muted); margin: 0.25rem 0 0; }
  form { display: flex; flex-direction: column; gap: 1rem; }
  .login-error { color: var(--color-danger); font-size: 0.875rem; margin: 0; }
</style>
