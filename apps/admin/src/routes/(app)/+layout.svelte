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
      <span class="sidebar__subtitle">Admin</span>
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

  <main class="content">
    {@render children()}
  </main>
</div>

<style>
  .shell { display: flex; min-height: 100vh; }

  .sidebar {
    width: 220px;
    flex-shrink: 0;
    background: var(--color-surface);
    border-right: 1px solid var(--color-border);
    display: flex;
    flex-direction: column;
    padding: 1.5rem 1rem;
    gap: 1rem;
  }
  .sidebar__brand { padding: 0 0.5rem 1rem; border-bottom: 1px solid var(--color-border); }
  .sidebar__logo { display: block; font-weight: 700; font-size: 1rem; color: var(--color-primary); }
  .sidebar__subtitle { font-size: 0.75rem; color: var(--color-muted); }

  .sidebar__nav { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 0.25rem; flex: 1; }
  .sidebar__link {
    display: block;
    padding: 0.5rem 0.75rem;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    color: var(--color-muted);
    transition: background 150ms, color 150ms;
  }
  .sidebar__link:hover { background: var(--color-surface-hover); color: var(--color-text); }
  .sidebar__link[aria-current="page"] { background: var(--color-primary); color: #fff; }

  .sidebar__logout {
    background: none;
    border: 1px solid var(--color-border);
    border-radius: 0.375rem;
    padding: 0.5rem;
    font-size: 0.875rem;
    cursor: pointer;
    color: var(--color-muted);
  }
  .sidebar__logout:hover { color: var(--color-danger); border-color: var(--color-danger); }

  .content { flex: 1; padding: 2rem; overflow-y: auto; }
</style>
