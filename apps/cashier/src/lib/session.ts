import { browser } from '$app/environment';
import { goto } from '$app/navigation';

const TOKEN_KEY = 'pos_cashier_token';
const ROLE_KEY  = 'pos_cashier_role';
const STORE_KEY = 'pos_cashier_store';

export function getToken(): string | null {
  if (!browser) return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function getStoreId(): string | null {
  if (!browser) return null;
  return localStorage.getItem(STORE_KEY);
}

export function setSession(token: string, role: string, storeId: string): void {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(ROLE_KEY, role);
  localStorage.setItem(STORE_KEY, storeId);
}

export function clearSession(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(ROLE_KEY);
  localStorage.removeItem(STORE_KEY);
}

export function requireCashier(): void {
  const role = localStorage.getItem(ROLE_KEY);
  if (role !== 'cashier') goto('/login');
}
