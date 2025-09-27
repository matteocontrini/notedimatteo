import type { Post } from '$lib/types';
import { db } from '$lib/server/db';
import { renderMarkdown } from '$lib/server/markdown';

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

	const posts: Post[] = await Promise.all(
		postsWithTags.map(async ({ postsToTags, body, ...post }) => ({
			...post,
			body: await renderMarkdown(body),
			tags: postsToTags.map((entry) => entry.tag.name)
		}))
	);

	const headingFormatter = new Intl.DateTimeFormat('it-IT', {
		timeZone: 'Europe/Rome',
		day: 'numeric',
		month: 'long',
		year: 'numeric'
	});

	const keyFormatter = new Intl.DateTimeFormat('en-CA', {
		timeZone: 'Europe/Rome',
		year: 'numeric',
		month: '2-digit',
		day: '2-digit'
	});

	const postGroups = posts.reduce<
		{
			dateKey: string;
			heading: string;
			posts: Post[];
		}[]
	>((groups, post) => {
		const publishedAt = post.publishedAt ?? post.createdAt;
		const date = new Date(publishedAt);
		const dateKey = keyFormatter.format(date);
		let group = groups.length ? groups[groups.length - 1] : null;

		if (!group || group.dateKey !== dateKey) {
			group = {
				dateKey,
				heading: headingFormatter.format(date),
				posts: []
			};
			groups.push(group);
		}

		group.posts.push(post);
		return groups;
	}, []);

	return {
		postGroups
	};
}
