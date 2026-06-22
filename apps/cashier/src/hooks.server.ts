import { redirect, type Handle } from "@sveltejs/kit";
import { readSession } from "$lib/server/session";

const REQUIRED_ROLE = "cashier";

/**
 * Server-edge role guard. CLAUDE.md requires hooks.server.ts to enforce the
 * cashier role for this app. localStorage cannot be read here, so the session
 * lives in an httpOnly cookie (see $lib/server/session).
 *
 * /api/*  — BFF proxy; manages its own auth (injects bearer, surfaces 401).
 * /auth/* — session endpoint; sets/clears the cookie itself.
 * Everything else is a page load and requires a valid cashier session.
 */
export const handle: Handle = async ({ event, resolve }) => {
  const session = readSession(event.cookies);
  event.locals.session = session;

  const { pathname } = event.url;

  if (pathname.startsWith("/api") || pathname.startsWith("/auth")) {
    return resolve(event);
  }

  const isCashier = session?.role === REQUIRED_ROLE;

  if (pathname === "/") {
    throw redirect(303, isCashier ? "/pos" : "/login");
  }
  if (!isCashier && pathname !== "/login") {
    throw redirect(303, "/login");
  }
  if (isCashier && pathname === "/login") {
    throw redirect(303, "/pos");
  }

  return resolve(event);
};
