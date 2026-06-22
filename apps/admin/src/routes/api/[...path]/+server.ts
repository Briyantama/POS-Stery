import { error, type RequestHandler } from "@sveltejs/kit";
import { env } from "$env/dynamic/private";
import { readSession } from "$lib/server/session";

const GATEWAY = env.GATEWAY_URL ?? "http://localhost:8080";

// Headers that must not be forwarded verbatim across the proxy hop.
const STRIP = new Set([
  "connection",
  "keep-alive",
  "transfer-encoding",
  "upgrade",
  "host",
  "content-length",
]);

/**
 * BFF proxy. The browser keeps calling /api/* (see $lib/api), but the bearer
 * token is held server-side in the httpOnly session cookie and injected here,
 * so the JWT never reaches client JS. Admin store scoping (X-Store-ID) is passed
 * through from the client and validated by the gateway's StoreResolver.
 */
const proxy: RequestHandler = async ({ request, params, url, cookies }) => {
  const session = readSession(cookies);
  const path = params.path ?? "";
  const target = `${GATEWAY}/api/${path}${url.search}`;

  const headers = new Headers();
  request.headers.forEach((value, key) => {
    if (!STRIP.has(key.toLowerCase())) headers.set(key, value);
  });
  if (session?.token) headers.set("Authorization", `Bearer ${session.token}`);

  const init: RequestInit = { method: request.method, headers };
  if (request.method !== "GET" && request.method !== "HEAD") {
    init.body = await request.arrayBuffer();
  }

  let res: Response;
  try {
    res = await fetch(target, init);
  } catch {
    throw error(502, "Upstream gateway unavailable.");
  }

  const resHeaders = new Headers();
  res.headers.forEach((value, key) => {
    if (!STRIP.has(key.toLowerCase())) resHeaders.set(key, value);
  });

  return new Response(res.body, {
    status: res.status,
    statusText: res.statusText,
    headers: resHeaders,
  });
};

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
