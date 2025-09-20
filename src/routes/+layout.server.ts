import { db } from '$lib/server/db';
import { posts, postsToTags, tags } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { count, desc, sql } from 'drizzle-orm';

export async function load() {
	const categories = await db
		.select({
			name: posts.category,
			count: count()
		})
		.from(posts)
		.groupBy(posts.category);

	const archive = await db
		.select({
			year: sql<number>`extract(year from ${posts.publishedAt})::int`,
			month: sql<number>`extract(month from ${posts.publishedAt})::int`,
			count: count()
		})
		.from(posts)
		.groupBy(
			sql`extract(year from ${posts.publishedAt})`,
			sql`extract(month from ${posts.publishedAt})`
		)
		.orderBy(
			desc(sql`extract(year from ${posts.publishedAt})`),
			desc(sql`extract(month from ${posts.publishedAt})`)
		);

	// Group archive by year and format month names
	const monthFormatter = new Intl.DateTimeFormat('it-IT', { month: 'long' });

	const grouped = archive.reduce((acc, row) => {
		const monthName = monthFormatter.format(new Date(row.year, row.month - 1, 1));
		const entry = { month: monthName, monthNumber: row.month, postCount: row.count };
		if (!acc.has(row.year)) acc.set(row.year, [entry]);
		else acc.get(row.year)!.push(entry);
		return acc;
	}, new Map<number, { month: string; monthNumber: number; postCount: number }[]>());

	const formattedArchive = Array.from(grouped.entries())
		.sort((a, b) => b[0] - a[0])
		.map(([year, months]) => ({
			year,
			months: months.sort((a, b) => b.monthNumber - a.monthNumber)
		}));

	const tagsList = await db
		.select({
			name: tags.name,
			count: sql<number>`count(*)`
		})
		.from(tags)
		.innerJoin(postsToTags, eq(postsToTags.tagName, tags.name))
		.innerJoin(posts, eq(posts.id, postsToTags.postId))
		.groupBy(tags.name)
		.orderBy(tags.name);

	return {
		categories,
		archive: formattedArchive,
		tags: tagsList
	};
}
