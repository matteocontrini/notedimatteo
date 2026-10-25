import type { Post } from '#lib/types.ts';
import { db } from '#lib/server/db/index.ts';
import { renderPostBody } from '#lib/server/markdown.ts';

const RESULTS_LIMIT = 20;

export async function load({ url }: { url: URL }) {
	const query = url.searchParams.get('q')?.trim() ?? '';

	if (!query) {
		return {
			query: '',
			posts: [] satisfies Post[],
			seo: {
				title: 'Cerca in Note di Matteo',
			},
		};
	}

	const postsWithTags = await db.post.findMany({
		take: RESULTS_LIMIT,
		orderBy: { publishedAt: 'desc' },
		include: {
			htmlCache: true,
			postsToTags: {
				select: {
					tag: { select: { name: true } },
				},
			},
		},
		where: {
			publishedAt: {
				not: null,
			},
			OR: [
				{ title: { contains: query, mode: 'insensitive' } },
				{ body: { contains: query, mode: 'insensitive' } },
				{
					postsToTags: {
						some: {
							tag: { name: { contains: query, mode: 'insensitive' } },
						},
					},
				},
			],
		},
	});

	const posts: Post[] = [];

	for (const { postsToTags, htmlCache, bodyRevision, ...post } of postsWithTags) {
		posts.push({
			...post,
			body: await renderPostBody({ ...post, bodyRevision, htmlCache }),
			tags: postsToTags.map((entry) => entry.tag.name),
		});
	}

	return {
		query,
		posts,
		seo: {
			title: `Risultati per "${query}"`,
		},
	};
}
