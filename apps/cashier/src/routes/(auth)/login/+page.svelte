<script lang="ts">
  import { goto } from '$app/navigation';
  import { Button, Input, LoginCard } from '@pos-stery/ui';

  let email    = $state('');
  let password = $state('');
  let storeId  = $state('');
  let error    = $state('');
  let loading  = $state(false);

  // Credentials are exchanged server-side (/auth/session), which validates the
  // cashier role and stores the JWT in an httpOnly cookie. The token never
  // touches client JS.
  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;
    try {
      const res = await fetch('/auth/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, store_id: storeId }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        error = data.message ?? 'Login failed.';
        return;
      }
      await goto('/pos');
    } catch {
      error = 'Login failed.';
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
