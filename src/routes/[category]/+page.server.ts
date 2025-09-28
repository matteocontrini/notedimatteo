import { getCategoryLabel, categorySlugs } from '$lib/categories';
import { db } from '$lib/server/db';
import { renderMarkdown } from '$lib/server/markdown';
import type { Post } from '$lib/types';
import { error } from '@sveltejs/kit';

export async function load({ params, url }) {
	const { category } = params;

	if (!category) {
		error(404);
	}

	if (!categorySlugs.includes(category)) {
		error(404);
	}

	const limit = 20;
	const page = parseInt(url.searchParams.get('page') || '1', 10);
	const offset = (page - 1) * limit;

	// Get total count for pagination
	const totalPosts = await db.post.count({
		where: {
			category,
			publishedAt: {
				not: null
			}
		}
	});

	const totalPages = Math.ceil(totalPosts / limit);

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
		pagination: {
			currentPage: page,
			totalPages,
			hasNextPage: page < totalPages,
			hasPreviousPage: page > 1,
			totalPosts
		},
		seo: {
			title: `${label} in Note di Matteo`
		}
	};
}
