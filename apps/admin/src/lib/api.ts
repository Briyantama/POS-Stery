const BASE = "/api";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public errors?: Record<string, string[]>,
  ) {
    super(message);
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  token?: string,
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new ApiError(res.status, data.message ?? res.statusText, data.errors);
  }

  return res.json() as T;
}

export const api = {
  get: <T>(path: string, token?: string) =>
    request<T>("GET", path, undefined, token),
  post: <T>(path: string, body: unknown, token?: string) =>
    request<T>("POST", path, body, token),
  put: <T>(path: string, body: unknown, token?: string) =>
    request<T>("PUT", path, body, token),
  delete: <T>(path: string, token?: string) =>
    request<T>("DELETE", path, undefined, token),
};
