import { db } from '$lib/server/db';
import type { Post } from '$lib/types';
import { error, redirect } from '@sveltejs/kit';

function normaliseCategory(input: string) {
	return input.toLowerCase().replace(/[^a-z]+/g, '');
}

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

	if (normaliseCategory(postRecord.category) !== normaliseCategory(params.category)) {
		error(404);
	}

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
		body: postRecord.body,
		tags: postRecord.postsToTags.map((entry) => entry.tag.name)
	};

	if (post.slug && params.slug !== post.slug) {
		redirect(307, `/${normaliseCategory(post.category)}/${post.number}/${post.slug}`);
	}

	return { post };
}
