import 'dotenv/config';
import postgres from 'postgres';
import { drizzle } from 'drizzle-orm/postgres-js';
import { inArray } from 'drizzle-orm';
import { posts, tags, postsToTags } from '../src/lib/server/db/schema';

function slugify(input: string): string {
	return input
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/(^-|-$)/g, '');
}

async function main() {
	const { DATABASE_URL, RESET } = process.env;
	if (!DATABASE_URL) {
		console.error('DATABASE_URL is not set. Add it to your environment or a .env file.');
		process.exit(1);
	}

	// Create a dedicated postgres client so we can close it at the end
	const client = postgres(DATABASE_URL, { max: 1 });
	const db = drizzle(client);

	try {
		if (RESET === '1') {
			console.log('RESET=1 detected — clearing tables...');
			await db.delete(postsToTags);
			await db.delete(posts);
			await db.delete(tags);
		}

		// Sample content
		const seedPosts: typeof posts.$inferInsert[] = [
			{
				number: 1,
				category: 'Note',
				title: 'Hello, world',
				slug: slugify('Hello, world'),
				hideTitle: false,
				body: 'First note — getting the project started with SvelteKit and Drizzle ORM.'
			},
			{
				number: 2,
				category: 'Note',
				title: 'Database schema',
				slug: slugify('Database schema'),
				hideTitle: false,
				body: 'Sketching the posts/tags schema and relations with drizzle-orm/pg-core.'
			},
			{
				number: 3,
				category: 'TIL',
				title: 'Seeding data',
				slug: slugify('Seeding data'),
				hideTitle: false,
				body: 'How to seed a Postgres database using drizzle-orm with postgres-js.'
			},
			{
				number: 4,
				category: 'Cit.',
				title: 'On simplicity',
				slug: slugify('On simplicity'),
				hideTitle: true,
				body: 'Some reflections on keeping things simple in software projects.'
			},
			{
				number: 5,
				category: 'Cit.',
				title: 'Svelte 5 snapshot',
				slug: slugify('Svelte 5 snapshot'),
				hideTitle: false,
				body: 'A brief look at Svelte 5 features and how they fit this project.'
			}
		] as const;

		const tagMapping: Record<number, string[]> = {
			1: ['svelte', 'drizzle', 'postgres'],
			2: ['drizzle', 'typescript'],
			3: ['drizzle', 'guide'],
			4: ['thoughts'],
			5: ['svelte', 'webdev']
		};

		const allTags = Array.from(new Set(Object.values(tagMapping).flat())).map((name) => ({ name }));

		// 1) Ensure tags exist
		if (allTags.length) {
			await db.insert(tags).values(allTags).onConflictDoNothing();
		}

		// 2) Insert posts (idempotent on unique "number")
		await db.insert(posts).values(seedPosts).onConflictDoNothing();

		// 3) Fetch post IDs to create join rows
		const numbers = seedPosts.map((p) => p.number);
		const existingPosts = await db
			.select({ id: posts.id, number: posts.number })
			.from(posts)
			.where(inArray(posts.number, numbers));

		const idByNumber = new Map(existingPosts.map((p) => [p.number, p.id] as const));

		const joinRows: { postId: number; tagName: string }[] = [];
		for (const [num, tagList] of Object.entries(tagMapping)) {
			const postId = idByNumber.get(Number(num));
			if (!postId) continue;
			for (const tagName of tagList) {
				joinRows.push({ postId, tagName });
			}
		}

		if (joinRows.length) {
			await db.insert(postsToTags).values(joinRows).onConflictDoNothing();
		}

		console.log('Seed complete:', {
			posts: numbers.length,
			tags: allTags.length,
			links: joinRows.length
		});
	} finally {
		// Close DB connection
		await client.end({ timeout: 5 });
	}
}

main().catch((err) => {
	console.error('Seed failed:', err);
	process.exit(1);
});
