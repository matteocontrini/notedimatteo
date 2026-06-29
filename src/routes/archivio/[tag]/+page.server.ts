import { db } from '$lib/server/db';
import { renderPostBody } from '$lib/server/markdown';
import type { Post } from '$lib/types';
import { error } from '@sveltejs/kit';

export async function load({ params, url }) {
	const tag = params.tag?.trim();

	if (!tag) {
		error(404);
	}

	const limit = 20;
	const page = parseInt(url.searchParams.get('page') || '1', 10);
	const offset = (page - 1) * limit;

	// Get total count for pagination
	const totalPosts = await db.post.count({
		where: {
			publishedAt: {
				not: null
			},
			postsToTags: {
				some: {
					tagName: tag
				}
			}
		}
	});

	if (totalPosts === 0) {
		error(404);
	}

	const totalPages = Math.ceil(totalPosts / limit);

	const postsWithTags = await db.post.findMany({
		take: limit,
		skip: offset,
		orderBy: { publishedAt: 'desc' },
		where: {
			publishedAt: {
				not: null
			},
			postsToTags: {
				some: {
					tagName: tag
				}
			}
		},
		include: {
			htmlCache: true,
			postsToTags: {
				select: {
					tag: { select: { name: true } }
				}
			}
		}
	});

	const posts: Post[] = [];

	for (const { postsToTags, htmlCache, bodyRevision, ...post } of postsWithTags) {
		posts.push({
			...post,
			body: await renderPostBody({ ...post, bodyRevision, htmlCache }),
			tags: postsToTags.map((entry) => entry.tag.name)
		});
	}

	return {
		posts,
		tag,
		pagination: {
			currentPage: page,
			totalPages,
			hasNextPage: page < totalPages,
			hasPreviousPage: page > 1,
			totalPosts
		},
		seo: {
			title: `#${tag}`
		}
	};
}
