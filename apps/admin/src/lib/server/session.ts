import { dev } from '$app/environment';
import { createSessionStore } from '@stery/kit/sveltekit/session';

/** Server-only admin session: the JWT lives in an HttpOnly cookie and never reaches client JS. */
export const session = createSessionStore({ cookie: 'pos_admin_session', secure: !dev });

/** Store the admin is working in; sent upstream as X-Store-ID. Set by the store selector. */
export const STORE_COOKIE = 'pos_admin_store';
