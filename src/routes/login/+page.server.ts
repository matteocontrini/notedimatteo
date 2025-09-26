import { fail, redirect } from '@sveltejs/kit';
import { randomBytes } from 'node:crypto';
import { env } from '$env/dynamic/private';
import { dev } from '$app/environment';
import { db } from '$lib/server/db';

const USERNAME = env.ADMIN_USERNAME;
const PASSWORD = env.ADMIN_PASSWORD;

if (!USERNAME || !PASSWORD) {
	throw new Error('ADMIN_USERNAME and ADMIN_PASSWORD must be set to enable login.');
}

export const actions = {
	default: async ({ request, cookies }) => {
		const formData = await request.formData();
		const username = formData.get('username');
		const password = formData.get('password');

		if (typeof username !== 'string' || typeof password !== 'string') {
			return fail(400, {
				message: 'Compila tutti i campi.',
				values: { username: typeof username === 'string' ? username : '' }
			});
		}

		if (username !== USERNAME || password !== PASSWORD) {
			return fail(400, {
				message: 'Credenziali non valide.',
				values: { username }
			});
		}

		const token = randomBytes(32).toString('hex');

		await db.session.create({
			data: { token }
		});

		cookies.set('notedimatteo_session', token, {
			path: '/',
			httpOnly: true,
			sameSite: 'lax',
			secure: !dev,
			maxAge: 60 * 60 * 24 * 60 // 60 days
		});

		throw redirect(303, '/');
	}
};
