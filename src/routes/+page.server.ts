import { db } from '$lib/server/db';
import { posts, postsToTags } from '$lib/server/db/schema';
import { desc } from 'drizzle-orm';
import { eq } from 'drizzle-orm';

export async function load() {
	// const categoryValue = 'note';
	const limit = 20;
	const offset = 0;

	const postsList = await db.query.posts.findMany({
		limit,
		offset,
		// where: eq(posts.category, categoryValue),
		orderBy: [desc(posts.publishedAt)],
		with: {
			postsToTags: true
		}
	});

	return {
		posts: postsList
	};
}
