import { getCategoryLabel, categorySlugs } from '$lib/categories';
import { db } from '$lib/server/db';
import { renderMarkdown } from '$lib/server/markdown';
import type { Post } from '$lib/types';
import { error } from '@sveltejs/kit';

export async function load({ params }) {
	const { category } = params;

	if (!category) {
		error(404);
	}

	if (!categorySlugs.includes(category)) {
		error(404);
	}

	const limit = 20;
	const offset = 0;

	const postsWithTags = await db.post.findMany({
		take: limit,
		skip: offset,
		orderBy: { publishedAt: 'desc' },
		include: {
			postsToTags: {
				select: {
					tag: { select: { name: true } }
				}
			}
		},
		where: {
			category,
			publishedAt: {
				not: null
			}
		}
	});

	const posts: Post[] = await Promise.all(
		postsWithTags.map(async ({ postsToTags, body, ...post }) => ({
			...post,
			body: await renderMarkdown(body),
			tags: postsToTags.map((entry) => entry.tag.name)
		}))
	);

	const label = getCategoryLabel(category);

	return {
		posts,
		category: {
			slug: category,
			label
		},
		seo: {
			title: `${label} in Note di Matteo`
		}
	};
}
