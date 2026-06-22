import { dev } from "$app/environment";
import type { Cookies } from "@sveltejs/kit";

/**
 * Server-only cashier session. The JWT lives in an httpOnly cookie so it is
 * never exposed to client JS. Cashier tokens embed tenant_id + store_id in their
 * claims, so the gateway derives store scope from the token; storeId is kept here
 * only for display. hooks.server.ts enforces the cashier role at the edge.
 */
export const SESSION_COOKIE = "pos_cashier_session";

// Matches the refresh-token TTL on auth-service (30 days).
const MAX_AGE_SECONDS = 60 * 60 * 24 * 30;

export interface Session {
  token: string;
  role: string;
  storeId: string;
}

export function readSession(cookies: Cookies): Session | null {
  const raw = cookies.get(SESSION_COOKIE);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<Session>;
    if (
      typeof parsed.token !== "string" ||
      typeof parsed.role !== "string" ||
      typeof parsed.storeId !== "string"
    ) {
      return null;
    }
    return { token: parsed.token, role: parsed.role, storeId: parsed.storeId };
  } catch {
    return null;
  }
}

export function writeSession(cookies: Cookies, session: Session): void {
  cookies.set(SESSION_COOKIE, JSON.stringify(session), {
    path: "/",
    httpOnly: true,
    sameSite: "lax",
    secure: !dev,
    maxAge: MAX_AGE_SECONDS,
  });
}

export function clearSession(cookies: Cookies): void {
  cookies.delete(SESSION_COOKIE, { path: "/" });
}
