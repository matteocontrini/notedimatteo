import type { Post } from '$lib/types';
import { db } from '$lib/server/db';
import { renderMarkdown } from '$lib/server/markdown';
import { fail, redirect } from '@sveltejs/kit';
import type { Actions } from './$types';

export async function load({ locals }) {
	if (!locals.session) {
		return redirect(303, '/login');
	}

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

export const actions = {
	delete: async (event) => {
		if (!event.locals.session) {
			return redirect(303, '/login');
		}

		const formData = await event.request.formData();
		const postIdValue = formData.get('postId');
		const postId = Number(postIdValue);

		if (!postIdValue || Number.isNaN(postId)) {
			return fail(400, { error: 'Invalid post id' });
		}

		try {
			await db.post.delete({ where: { id: postId } });
		} catch (error) {
			console.error('Error deleting post:', error);
			return fail(500, { error: 'Unable to delete post' });
		}

		return { success: true };
	}
} satisfies Actions;
