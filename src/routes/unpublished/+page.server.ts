import type { Post } from '$lib/types';
import { db } from '$lib/server/db';
import { renderMarkdown } from '$lib/server/markdown';

export async function load() {
	const postsWithTags = await db.post.findMany({
		orderBy: { createdAt: 'desc' },
		where: {
			publishedAt: null
		},
		include: {
			postsToTags: {
				select: {
					tag: { select: { name: true } }
				}
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

	return {
		posts
	};
}
