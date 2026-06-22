import { json, error, type RequestHandler } from "@sveltejs/kit";
import { env } from "$env/dynamic/private";
import { writeSession, clearSession, readSession } from "$lib/server/session";

const GATEWAY = env.GATEWAY_URL ?? "http://localhost:8080";
const REQUIRED_ROLE = "admin";

// Shape returned by the gateway, which transcodes auth.proto LoginResponse.
// Token is `access_token`; role/store live under `claims`. We tolerate both
// snake_case and camelCase JSON, and a flat fallback, to avoid coupling to the
// grpc-gateway marshaler's casing config.
interface UserClaims {
  role?: string;
  store_id?: string;
  storeId?: string;
}
interface LoginResponse {
  access_token?: string;
  accessToken?: string;
  token?: string;
  role?: string;
  claims?: UserClaims;
  message?: string;
}

/** POST /auth/session — exchange credentials for an httpOnly admin session. */
export const POST: RequestHandler = async ({ request, cookies }) => {
  let payload: { email?: string; password?: string };
  try {
    payload = await request.json();
  } catch {
    throw error(400, "Invalid request body.");
  }

  const email = payload.email?.trim();
  const password = payload.password;
  if (!email || !password) throw error(400, "Email and password are required.");

  let res: Response;
  try {
    res = await fetch(`${GATEWAY}/api/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ email, password }),
    });
  } catch {
    throw error(502, "Cannot reach authentication service.");
  }

  const data = (await res.json().catch(() => ({}))) as LoginResponse;
  if (!res.ok) {
    throw error(res.status === 401 ? 401 : 400, data.message ?? "Login failed.");
  }

  const token = data.access_token ?? data.accessToken ?? data.token;
  const role = data.claims?.role ?? data.role;
  if (role !== REQUIRED_ROLE) throw error(403, "Admin access only.");
  if (!token) throw error(502, "Invalid login response from gateway.");

  writeSession(cookies, { token, role });
  return json({ role });
};

/** DELETE /auth/session — best-effort gateway logout, then clear the cookie. */
export const DELETE: RequestHandler = async ({ cookies }) => {
  const session = readSession(cookies);
  if (session?.token) {
    try {
      await fetch(`${GATEWAY}/api/logout`, {
        method: "POST",
        headers: { Authorization: `Bearer ${session.token}`, Accept: "application/json" },
      });
    } catch {
      // Clear the local session regardless of upstream availability.
    }
  }
  clearSession(cookies);
  return json({ ok: true });
};
