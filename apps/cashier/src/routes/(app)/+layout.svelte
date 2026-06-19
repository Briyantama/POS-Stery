<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { getToken, getStoreId, clearSession, requireCashier } from '$lib/session';
  import { api } from '$lib/api';

  interface Props { children: import('svelte').Snippet; }
  let { children }: Props = $props();

  onMount(() => requireCashier());

  async function logout() {
    const token = getToken();
    if (token) await api.post('/logout', {}, token).catch(() => {});
    clearSession();
    goto('/login');
  }
</script>

<div class="shell">
  <header class="topbar">
    <span class="topbar__brand">POS-Stery Cashier</span>
    <nav class="topbar__nav" aria-label="Cashier navigation">
      <a href="/pos" class="topbar__link">New Sale</a>
      <a href="/receipts" class="topbar__link">Receipts</a>
    </nav>
    <button class="topbar__logout" onclick={logout}>Sign out</button>
  </header>

  <main class="content">
    {@render children()}
  </main>
</div>

<style>
  .shell { display: flex; flex-direction: column; min-height: 100vh; }

  .topbar {
    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 0.75rem 1.5rem;
    background: var(--color-primary);
    color: #fff;
  }
  .topbar__brand { font-weight: 700; font-size: 1rem; }
  .topbar__nav   { display: flex; gap: 1rem; flex: 1; }
  .topbar__link  { color: rgba(255,255,255,0.85); font-size: 0.875rem; transition: color 150ms; }
  .topbar__link:hover { color: #fff; }
  .topbar__logout {
    background: rgba(255,255,255,0.15);
    border: none;
    border-radius: 0.375rem;
    padding: 0.375rem 0.75rem;
    color: #fff;
    font-size: 0.875rem;
    cursor: pointer;
  }
  .topbar__logout:hover { background: rgba(255,255,255,0.25); }

  .content { flex: 1; padding: 1.5rem; }
</style>
