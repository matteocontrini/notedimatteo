import { db } from '$lib/server/db';
import { posts } from '$lib/server/db/schema';
import { count } from 'drizzle-orm';

export async function load() {
	const categories = await db
		.select({
			name: posts.category,
			count: count()
		})
		.from(posts)
		.groupBy(posts.category);

	return {
		categories
	};
}
