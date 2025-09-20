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
			hideTitle: true,
			body: `Dolore sint occaecat dolore. Lorem magna dolore sit non. Tempor et eiusmod cupidatat est magna reprehenderit quis fugiat. Mollit eu mollit consectetur magna qui incididunt ad commodo ea.
	
Id enim et velit aliquip elit aute. Irure duis aute aliquip labore culpa laborum. Nostrud fugiat cupidatat veniam incididunt in fugiat aute officia irure consectetur Lorem culpa incididunt amet excepteur. Ut sit dolore nisi anim in sunt reprehenderit dolore tempor voluptate sit duis mollit excepteur exercitation. Elit fugiat sunt dolore ad sunt deserunt occaecat tempor ut sint dolore veniam qui. Voluptate nostrud dolore enim tempor aute. Aliqua cillum fugiat cupidatat aute culpa aliquip reprehenderit laborum. Nisi ad veniam sint ullamco reprehenderit adipisicing deserunt cupidatat eiusmod aliqua.

Est proident aliqua dolore deserunt sint nulla laboris aute elit laboris ad minim. Laboris pariatur ad officia voluptate mollit duis nulla exercitation incididunt cupidatat pariatur ex. Incididunt culpa nisi voluptate fugiat anim nulla velit fugiat do incididunt in veniam. Consequat consequat exercitation enim nulla minim magna ipsum. Anim adipisicing qui consequat ad nostrud esse reprehenderit consequat.

Lorem anim sunt officia velit velit culpa Lorem amet. Amet ullamco nostrud sit laboris est velit enim Lorem cillum. Qui est ea do laborum magna qui quis do duis minim non. Esse esse aliquip mollit tempor sit adipisicing consequat adipisicing veniam officia nulla deserunt labore excepteur. Irure Lorem velit qui ad minim adipisicing consequat. Aute cupidatat laborum incididunt cupidatat proident ex culpa in aute sint qui aliqua veniam non. Eu est sunt deserunt aliquip est sint ea non do officia adipisicing ipsum aute.
`
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
	];

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
