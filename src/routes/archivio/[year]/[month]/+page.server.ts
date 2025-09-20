import { db } from '$lib/server/db';
import type { Post } from '$lib/types';
import { error, redirect } from '@sveltejs/kit';

export async function load({ params, url }) {
	const yearParam = params.year;
	const monthParam = params.month;

	const year = Number(yearParam);
	const month = Number(monthParam);

	if (!Number.isInteger(year) || !Number.isInteger(month)) {
		error(404);
	}

	if (month < 1 || month > 12) {
		error(404);
	}

	const canonicalMonth = month.toString().padStart(2, '0');

	if (monthParam !== canonicalMonth) {
		throw redirect(307, `/archivio/${year}/${canonicalMonth}${url.search}`);
	}

	const monthStart = new Date(Date.UTC(year, month - 1, 1));
	const monthEnd = new Date(Date.UTC(month === 12 ? year + 1 : year, month === 12 ? 0 : month, 1));

	const postsWithTags = await db.post.findMany({
		orderBy: { publishedAt: 'desc' },
		where: {
			publishedAt: {
				not: null,
				gte: monthStart,
				lt: monthEnd
			}
		},
		include: {
			postsToTags: {
				select: {
					tag: { select: { name: true } }
				}
			}
		}
	});

	if (postsWithTags.length === 0) {
		error(404);
	}

	const posts: Post[] = postsWithTags.map(({ postsToTags, ...post }) => ({
		...post,
		tags: postsToTags.map((entry) => entry.tag.name)
	}));

	return {
		posts,
		year,
		month
	};
}
