<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, ApiError } from '$lib/api';
  import { setSession } from '$lib/session';
  import { Button, Input, LoginCard } from '@pos-stery/ui';

  let email    = $state('');
  let password = $state('');
  let storeId  = $state('');
  let error    = $state('');
  let loading  = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;
    try {
      const res = await api.post<{ token: string; role: string; store_id: string }>(
        '/login',
        { email, password, store_id: storeId },
      );
      if (res.role !== 'cashier') { error = 'Cashier access only.'; return; }
      setSession(res.token, res.role, res.store_id ?? storeId);
      goto('/pos');
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Login failed.';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head><title>Sign in — POS-Stery Cashier</title></svelte:head>

<LoginCard subtitle="Cashier Terminal" {error}>
  <form onsubmit={handleSubmit}>
    <Input label="Email" type="email" bind:value={email} required />
    <Input label="Password" type="password" bind:value={password} required />
    <Input label="Store ID" bind:value={storeId} placeholder="UUID" required />
    <Button type="submit" {loading} disabled={loading}>Sign in</Button>
  </form>
</LoginCard>

<style>
  form { display: flex; flex-direction: column; gap: var(--space-4, 1rem); }
</style>
