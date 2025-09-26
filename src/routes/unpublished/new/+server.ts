import { db } from '$lib/server/db';
import { redirect } from '@sveltejs/kit';

const DEFAULT_CATEGORY = 'note';

export async function GET({ locals }) {
	if (!locals.session) {
		return redirect(303, '/login');
	}

	const lastPost = await db.post.findFirst({
		select: { number: true },
		orderBy: { number: 'desc' }
	});

	const nextNumber = (lastPost?.number ?? 0) + 1;

	await db.post.create({
		data: {
			number: nextNumber,
			category: DEFAULT_CATEGORY,
			title: null,
			slug: null,
			hideTitle: false,
			body: '',
			publishedAt: null,
			updatedAt: null
		}
	});

	redirect(303, `/${nextNumber}/edit`);
}
