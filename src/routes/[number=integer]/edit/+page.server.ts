import { db } from '$lib/server/db';
import type { PostForEdit } from '$lib/types';
import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load = (async ({ params }) => {
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

	if (!postRecord) {
		error(404);
	}

	const post: PostForEdit = {
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

	return { post };
}) satisfies PageServerLoad;
