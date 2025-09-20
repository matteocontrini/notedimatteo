import { db } from '$lib/server/db';
import { renderMarkdown } from '$lib/server/markdown';
import type { Post } from '$lib/types';
import { error } from '@sveltejs/kit';

export async function load({ params }) {
	const tag = params.tag?.trim();

	if (!tag) {
		error(404);
	}

	const postsWithTags = await db.post.findMany({
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

	const posts: Post[] = await Promise.all(
		postsWithTags.map(async ({ postsToTags, body, ...post }) => ({
			...post,
			body: await renderMarkdown(body),
			tags: postsToTags.map((entry) => entry.tag.name)
		}))
	);

	return {
		posts,
		tag
	};
}
