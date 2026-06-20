<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { getToken, clearSession, requireCashier } from '$lib/session';
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
    <div class="topbar__brand">
      <span class="topbar__logo">POS-Stery</span>
      <span class="topbar__dot" aria-hidden="true"></span>
    </div>

    <nav class="topbar__nav" aria-label="Cashier navigation">
      <a
        href="/pos"
        class="topbar__link"
        aria-current={$page.url.pathname.startsWith('/pos') ? 'page' : undefined}
      >New Sale</a>
      <a
        href="/receipts"
        class="topbar__link"
        aria-current={$page.url.pathname.startsWith('/receipts') ? 'page' : undefined}
      >Receipts</a>
    </nav>

    <button class="topbar__logout" onclick={logout}>Sign out</button>
  </header>

  <main class="content">
    {@render children()}
  </main>
</div>

<style>
  .shell {
    display: flex;
    flex-direction: column;
    height: 100dvh;
    overflow: hidden;
  }

  /* ---- Ink topbar ---- */
  .topbar {
    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 0 1.5rem;
    height: 52px;
    flex-shrink: 0;
    background: var(--color-primary-ink, #11224f);
    color: #fff;
  }

  .topbar__brand {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin-right: 0.5rem;
  }
  .topbar__logo {
    font-weight: var(--weight-black, 800);
    font-size: var(--text-sm, 0.875rem);
    color: var(--color-on-ink, #f0ebe1);
    letter-spacing: var(--tracking-tight, -0.02em);
  }
  /* Marigold live dot */
  .topbar__dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--color-accent, #e0992e);
    flex-shrink: 0;
  }

  .topbar__nav {
    display: flex;
    gap: 0;
    flex: 1;
    height: 100%;
  }

  .topbar__link {
    display: flex;
    align-items: center;
    padding: 0 1rem;
    height: 100%;
    color: rgb(240 235 225 / 0.65);
    font-size: var(--text-sm, 0.875rem);
    font-weight: var(--weight-medium, 500);
    transition: color 140ms;
    position: relative;
    border-bottom: 3px solid transparent;
    box-sizing: border-box;
  }
  .topbar__link:hover { color: var(--color-on-ink, #f0ebe1); }
  /* Active: white text + marigold underline */
  .topbar__link[aria-current="page"] {
    color: #fff;
    border-bottom-color: var(--color-accent, #e0992e);
  }

  .topbar__logout {
    background: rgb(255 255 255 / 0.1);
    border: 1px solid rgb(255 255 255 / 0.15);
    border-radius: var(--radius-md, 0.375rem);
    padding: 0.375rem 0.875rem;
    color: rgb(240 235 225 / 0.8);
    font-size: var(--text-sm, 0.875rem);
    cursor: pointer;
    transition: background 140ms, color 140ms;
  }
  .topbar__logout:hover {
    background: rgb(255 255 255 / 0.18);
    color: #fff;
  }

  /* ---- Content (full remaining height, no padding — POS page owns its own layout) ---- */
  .content {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }
</style>
