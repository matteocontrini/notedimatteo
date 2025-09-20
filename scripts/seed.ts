import 'dotenv/config';
import { PrismaClient } from '@prisma/client';

const prisma = new PrismaClient();

function slugify(input: string): string {
	return input
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/(^-|-$)/g, '');
}

async function main() {
	const { RESET } = process.env;

	if (RESET === '1') {
		console.log('RESET=1 detected — clearing tables...');
		await prisma.postsToTags.deleteMany();
		await prisma.post.deleteMany();
		await prisma.tag.deleteMany();
	}

	const seedPosts = [
		{
			number: 1,
			category: 'note',
			title: 'Hello, world',
			slug: slugify('Hello, world'),
			hideTitle: false,
			body: 'First note — getting the project started with SvelteKit and Prisma.'
		},
		{
			number: 2,
			category: 'note',
			title: 'Database schema',
			slug: slugify('Database schema'),
			hideTitle: false,
			body: 'Sketching the posts/tags schema and relations.'
		},
		{
			number: 3,
			category: 'til',
			title: 'Seeding data',
			slug: slugify('Seeding data'),
			hideTitle: false,
			body: 'How to seed a Postgres database using Prisma.'
		},
		{
			number: 4,
			category: 'cit',
			title: 'On simplicity',
			slug: slugify('On simplicity'),
			hideTitle: true,
			body: 'Some reflections on keeping things simple in software projects.'
		},
		{
			number: 5,
			category: 'cit',
			title: 'Svelte 5 snapshot',
			slug: slugify('Svelte 5 snapshot'),
			hideTitle: false,
			body: 'A brief look at Svelte 5 features and how they fit this project.'
		}
	] as const;

	const tagMapping: Record<number, string[]> = {
		1: ['svelte', 'prisma', 'postgres'],
		2: ['prisma', 'typescript'],
		3: ['prisma', 'guide'],
		4: ['thoughts'],
		5: ['svelte', 'webdev']
	};

	const allTags = Array.from(new Set(Object.values(tagMapping).flat())).map((name) => ({ name }));

	if (allTags.length) {
		await prisma.tag.createMany({ data: allTags, skipDuplicates: true });
	}

	await prisma.post.createMany({ data: seedPosts, skipDuplicates: true });

	const numbers = seedPosts.map((p) => p.number);
	const existingPosts = await prisma.post.findMany({
		where: { number: { in: numbers } },
		select: { id: true, number: true }
	});

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
		await prisma.postsToTags.createMany({ data: joinRows, skipDuplicates: true });
	}

	console.log('Seed complete:', {
		posts: seedPosts.length,
		tags: allTags.length,
		links: joinRows.length
	});
}

main()
	.catch((err) => {
		console.error('Seed failed:', err);
		process.exit(1);
	})
	.finally(async () => {
		await prisma.$disconnect();
	});
