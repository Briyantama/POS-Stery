import { createClient } from '@stery/kit';

export { ApiError } from '@stery/kit';

/** Browser client for the BFF proxy (/api -> Go API /api/v1); returns the envelope's data. */
export const api = createClient('/api');
