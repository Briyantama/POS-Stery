import { env } from '$env/dynamic/private';
import { createApiProxy } from '@stery/kit/sveltekit/proxy';
import { session } from '$lib/server/session';

/** BFF proxy: /api/<path> goes to the Go API's /api/v1/<path> with the session's bearer token. */
export const { GET, POST, PUT, PATCH, DELETE } = createApiProxy({
	upstream: () => env.GATEWAY_URL ?? 'http://localhost:8000',
	accessCookie: session.cookie,
	onUnauthorized: ({ cookies }) => session.clear(cookies)
});
