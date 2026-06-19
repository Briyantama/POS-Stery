import { browser } from '$app/environment';
import { goto } from '$app/navigation';

const TOKEN_KEY = 'pos_admin_token';
const ROLE_KEY  = 'pos_admin_role';

export function getToken(): string | null {
  if (!browser) return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function getRole(): string | null {
  if (!browser) return null;
  return localStorage.getItem(ROLE_KEY);
}

export function setSession(token: string, role: string): void {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(ROLE_KEY, role);
}

export function clearSession(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(ROLE_KEY);
}

export function requireAdmin(): void {
  const role = getRole();
  if (role !== 'admin') {
    goto('/login');
  }
}
