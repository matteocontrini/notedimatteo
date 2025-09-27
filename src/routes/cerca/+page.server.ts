import type { Post } from '$lib/types';
import { db } from '$lib/server/db';
import { renderMarkdown } from '$lib/server/markdown';

const RESULTS_LIMIT = 20;

export async function load({ url }: { url: URL }) {
	const query = url.searchParams.get('q')?.trim() ?? '';

	if (!query) {
		return {
			query: '',
			posts: [] satisfies Post[],
			seo: {
				title: 'Cerca in Note di Matteo'
			}
		};
	}

	const postsWithTags = await db.post.findMany({
		take: RESULTS_LIMIT,
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
			},
			OR: [
				{ title: { contains: query, mode: 'insensitive' } },
				{ body: { contains: query, mode: 'insensitive' } },
				{
					postsToTags: {
						some: {
							tag: { name: { contains: query, mode: 'insensitive' } }
						}
					}
				}
			]
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
		query,
		posts,
		seo: {
			title: `Risultati per "${query}"`
		}
	};
}
