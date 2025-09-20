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

	return { post };
}
