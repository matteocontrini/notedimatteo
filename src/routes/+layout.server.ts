import { getCategoryLabel } from '$lib/categories';
import { db } from '$lib/server/db';
import type { CalendarItem } from '$lib/types';

export async function load() {
	const categoriesResults = await db.post.groupBy({
		by: ['category'],
		_count: { _all: true },
		orderBy: { category: 'asc' },
		where: {
			publishedAt: {
				not: null
			}
		}
	});

	const categories = categoriesResults.map((category) => ({
		slug: category.category,
		label: getCategoryLabel(category.category),
		count: category._count._all
	}));

	const archive = await db.$queryRaw<CalendarItem[]>`
		SELECT EXTRACT(YEAR FROM "publishedAt")::int AS "year",
		       EXTRACT(MONTH FROM "publishedAt")::int AS "month",
		       COUNT(*)::int AS "postsCount"
		FROM "posts"
		WHERE "publishedAt" IS NOT NULL
		GROUP BY 1, 2
		ORDER BY 1 DESC, 2 DESC
	`;

	const tagsResults = await db.postsToTags.groupBy({
		by: ['tagName'],
		_count: { _all: true },
		where: {
			post: {
				publishedAt: {
					not: null
				}
			}
		},
		orderBy: { tagName: 'asc' }
	});

	const tags = tagsResults.map((tag) => ({
		name: tag.tagName,
		count: tag._count._all
	}));

	return {
		categories,
		archive,
		tags
	};
}
