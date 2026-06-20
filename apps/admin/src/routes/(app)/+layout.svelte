<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { getToken, getRole, clearSession } from '$lib/auth';
  import { api } from '$lib/api';

  interface Props { children: import('svelte').Snippet; }
  let { children }: Props = $props();

  const navLinks = [
    { href: '/dashboard', label: 'Dashboard' },
    { href: '/products',  label: 'Products' },
    { href: '/inventory', label: 'Inventory' },
    { href: '/suppliers', label: 'Suppliers' },
    { href: '/customers', label: 'Customers' },
    { href: '/reports',   label: 'Reports' },
  ];

  onMount(() => {
    const role = getRole();
    if (role !== 'admin') goto('/login');
  });

  async function logout() {
    const token = getToken();
    if (token) await api.post('/logout', {}, token).catch(() => {});
    clearSession();
    goto('/login');
  }
</script>

<div class="shell">
  <nav class="sidebar" aria-label="Main navigation">
    <div class="sidebar__brand">
      <span class="sidebar__logo">POS-Stery</span>
      <span class="sidebar__dot" aria-hidden="true"></span>
    </div>

    <ul class="sidebar__nav" role="list">
      {#each navLinks as link}
        <li>
          <a
            href={link.href}
            class="sidebar__link"
            aria-current={$page.url.pathname.startsWith(link.href) ? 'page' : undefined}
          >{link.label}</a>
        </li>
      {/each}
    </ul>

    <button class="sidebar__logout" onclick={logout}>Sign out</button>
  </nav>

  <div class="main-area">
    <main class="content">
      {@render children()}
    </main>
  </div>
</div>

<style>
  .shell { display: flex; min-height: 100vh; }

  /* ---- Ink sidebar ---- */
  .sidebar {
    width: 232px;
    flex-shrink: 0;
    background: var(--color-primary-ink, #11224f);
    display: flex;
    flex-direction: column;
    padding: 1.5rem 0 1.25rem;
    gap: 0.5rem;
    position: sticky;
    top: 0;
    height: 100vh;
    overflow-y: auto;
  }

  .sidebar__brand {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0 1.25rem 1.25rem;
    border-bottom: 1px solid rgb(255 255 255 / 0.1);
    margin-bottom: 0.5rem;
  }
  .sidebar__logo {
    font-size: var(--text-base, 1rem);
    font-weight: var(--weight-black, 800);
    color: var(--color-on-ink, #f0ebe1);
    letter-spacing: var(--tracking-tight, -0.02em);
  }
  /* Marigold live dot */
  .sidebar__dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--color-accent, #e0992e);
    flex-shrink: 0;
  }

  .sidebar__nav {
    list-style: none;
    margin: 0;
    padding: 0 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    flex: 1;
  }

  .sidebar__link {
    display: block;
    padding: 0.5rem 0.75rem;
    border-radius: var(--radius-md, 0.375rem);
    font-size: var(--text-sm, 0.875rem);
    font-weight: var(--weight-medium, 500);
    color: rgb(240 235 225 / 0.65);
    transition: background 140ms, color 140ms;
    position: relative;
  }
  .sidebar__link:hover {
    background: rgb(255 255 255 / 0.07);
    color: var(--color-on-ink, #f0ebe1);
  }
  /* Active state: white text + marigold 3px left rail */
  .sidebar__link[aria-current="page"] {
    color: var(--color-on-ink, #f0ebe1);
    background: rgb(255 255 255 / 0.1);
  }
  .sidebar__link[aria-current="page"]::before {
    content: '';
    position: absolute;
    left: -0.75rem;
    top: 25%;
    bottom: 25%;
    width: 3px;
    border-radius: 0 2px 2px 0;
    background: var(--color-accent, #e0992e);
  }

  .sidebar__logout {
    margin: 0 0.75rem;
    background: none;
    border: 1px solid rgb(255 255 255 / 0.15);
    border-radius: var(--radius-md, 0.375rem);
    padding: 0.5rem 0.75rem;
    font-size: var(--text-sm, 0.875rem);
    cursor: pointer;
    color: rgb(240 235 225 / 0.65);
    text-align: left;
    transition: border-color 140ms, color 140ms;
  }
  .sidebar__logout:hover {
    color: var(--color-on-ink, #f0ebe1);
    border-color: rgb(255 255 255 / 0.3);
  }
  .sidebar__logout:focus-visible { outline: none; box-shadow: 0 0 0 3px rgb(255 255 255 / 0.3); }

  /* ---- Content area ---- */
  .main-area { flex: 1; min-width: 0; display: flex; flex-direction: column; }
  .content { flex: 1; padding: 2rem; overflow-y: auto; }
</style>
