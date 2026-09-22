import type { Handle, ServerInit } from '@sveltejs/kit';
import { building } from '$app/environment';
import { db } from '$lib/server/db';
import * as logger from '$lib/server/logger';

const SESSION_COOKIE = 'notedimatteo_session';

export const init: ServerInit = async () => {
	if (building) return;

	await db.$connect();
};

export const handle: Handle = async ({ event, resolve }) => {
	const ip =
		event.request.headers.get('x-forwarded-for')?.split(',')[0] || event.getClientAddress();
	logger.info(`${ip} - ${event.request.method} ${event.url.pathname + event.url.search}`);

	const token = event.cookies.get(SESSION_COOKIE);

	event.locals.session = null;

	if (token) {
		const session = await db.session.findFirst({
			where: { token },
		});

		if (session) {
			event.locals.session = session.token;

			// Update last activity time at most once a minute
			if (Date.now() - session.lastActivityAt.getTime() > 60 * 1000) {
				await db.session.update({
					where: { id: session.id },
					data: { lastActivityAt: new Date() },
				});
			}
		} else {
			event.cookies.delete(SESSION_COOKIE, { path: '/' });
		}
	}

	return resolve(event);
};
