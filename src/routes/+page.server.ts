import type { Post } from '$lib/types';
import { db } from '$lib/server/db';

export async function load() {
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
			publishedAt: {
				not: null
			}
		}
	});

	const posts: Post[] = postsWithTags.map(({ postsToTags, ...post }) => ({
		...post,
		tags: postsToTags.map((entry) => entry.tag.name)
	}));

	return {
		posts
	};
}
