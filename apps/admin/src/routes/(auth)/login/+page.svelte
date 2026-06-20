<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, ApiError } from '$lib/api';
  import { setSession } from '$lib/auth';
  import { Button, Input, LoginCard } from '@pos-stery/ui';

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
      if (res.role !== 'admin') { error = 'Admin access only.'; return; }
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

<LoginCard subtitle="Admin Dashboard" {error}>
  <form onsubmit={handleSubmit}>
    <Input label="Email" type="email" bind:value={email} required />
    <Input label="Password" type="password" bind:value={password} required />
    <Button type="submit" {loading} disabled={loading}>Sign in</Button>
  </form>
</LoginCard>

<style>
  form { display: flex; flex-direction: column; gap: var(--space-4, 1rem); }
</style>
