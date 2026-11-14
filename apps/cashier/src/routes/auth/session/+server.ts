import { env } from '$env/dynamic/private';
import { createSessionHandlers } from '@stery/kit/sveltekit/session';
import { session } from '$lib/server/session';

/** POST logs in and sets the cashier session cookie; DELETE logs out and clears it. */
export const { POST, DELETE } = createSessionHandlers({
	upstream: () => env.GATEWAY_URL ?? 'http://localhost:8000',
	store: session,
	requiredRole: 'cashier',
	forbiddenMessage: 'Cashier access only.'
});
