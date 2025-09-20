import { db } from '$lib/server/db';

export async function load() {
	// Extract tags
	const tagsResults = await db.tag.findMany({
		select: {
			name: true,
			_count: { select: { postsToTags: true } }
		},
		orderBy: { name: 'asc' }
	});

	const tags = tagsResults.map((tag) => ({
		name: tag.name,
		count: tag._count.postsToTags
	}));

	return {
		categories: [],
		archive: [],
		tags
	};
}
