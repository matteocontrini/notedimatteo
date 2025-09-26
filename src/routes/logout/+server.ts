import { redirect } from '@sveltejs/kit';
import { db } from '$lib/server/db';

const SESSION_COOKIE = 'notedimatteo_session';

export async function GET({ cookies }) {
	const token = cookies.get(SESSION_COOKIE);

	if (token) {
		await db.session.deleteMany({ where: { token } });
		cookies.delete(SESSION_COOKIE, { path: '/' });
	}

	return redirect(303, '/');
}
