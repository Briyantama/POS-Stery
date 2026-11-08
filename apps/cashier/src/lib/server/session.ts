import { dev } from '$app/environment';
import { createSessionStore } from '@stery/kit/sveltekit/session';

/** Server-only cashier session: the JWT lives in an HttpOnly cookie and never reaches client JS. */
export const session = createSessionStore({ cookie: 'pos_cashier_session', secure: !dev, requireStore: true });
