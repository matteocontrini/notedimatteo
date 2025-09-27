import { db } from '$lib/server/db';
import { renderMarkdown } from '$lib/server/markdown';
import type { Post } from '$lib/types';
import { error, redirect } from '@sveltejs/kit';

export async function load({ params }) {
	const number = Number(params.number);

	if (!Number.isInteger(number)) {
		error(404);
	}

	const postRecord = await db.post.findUnique({
		where: { number },
		include: {
			postsToTags: {
				select: {
					tag: { select: { name: true } }
				}
			}
		}
	});

	if (!postRecord || !postRecord.publishedAt) {
		error(404);
	}

	const body = await renderMarkdown(postRecord.body);

	const post: Post = {
		id: postRecord.id,
		number: postRecord.number,
		category: postRecord.category,
		createdAt: postRecord.createdAt,
		updatedAt: postRecord.updatedAt,
		publishedAt: postRecord.publishedAt,
		title: postRecord.title,
		slug: postRecord.slug,
		hideTitle: postRecord.hideTitle,
		body,
		tags: postRecord.postsToTags.map((entry) => entry.tag.name)
	};

	const targetBase = `/${post.number}`;

	if (post.slug) {
		if (params.slug !== post.slug) {
			redirect(307, `${targetBase}/${post.slug}`);
		}
	} else if (params.slug) {
		redirect(307, targetBase);
	}

	return {
		post,
		seo: {
			title: postTitle(post),
			description: body.slice(0, 160) + '...',
			ogType: 'article',
			jsonLd: {
				'@context': 'https://schema.org',
				'@type': 'NewsArticle',
				headline: postTitle(post),
				datePublished: post.publishedAt!.toISOString(),
				dateModified: post.updatedAt?.toISOString()
			}
		}
	};
}

function postTitle(post: Post): string {
	if (post.title) {
		return post.title;
	} else {
		const date = new Date(post.publishedAt!).toLocaleDateString('it-IT', {
			timeZone: 'Europe/Rome'
		});
		return `Nota #${post.number} del ${date}`;
	}
}
