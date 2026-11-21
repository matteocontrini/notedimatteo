// The data the server starts with. The tests compute what pages should show
// from these posts, so changing them doesn't break the tests' expectations.

export const ADMIN_USERNAME = 'admin';
export const ADMIN_PASSWORD = 'e2e-password';
export const BUCKET = 'uploads';

export type SeedPost = {
	number: number;
	category: string;
	title: string | null;
	slug: string | null;
	body: string;
	tags: string[];
	publishedAt: Date;
};

// posts are 24 published notes, more than the 20 that a page lists: posts 1–10
// in September 2026 and posts 11–24 in October.
export const posts: SeedPost[] = Array.from({ length: 24 }, (_, i) => {
	const number = i + 1;
	const tags = [];
	if (number % 2 === 0) tags.push('go');
	if (number % 5 === 0) tags.push('svelte');

	return {
		number,
		category: ['note', 'til', 'cit'][number % 3],
		title: number === 24 ? 'Il post più recente' : null,
		slug: number === 24 ? 'il-post-piu-recente' : null,
		body: number === 7 ? 'Una nota sullo zucchero.' : `Nota numero ${number}.`,
		tags,
		publishedAt:
			number <= 10
				? new Date(Date.UTC(2026, 8, number + 1, 10))
				: new Date(Date.UTC(2026, 9, number - 10, 10)),
	};
});

const quote = (value: string | null) =>
	value === null ? 'NULL' : `'${value.replaceAll("'", "''")}'`;

export function seedSQL() {
	const tags = [...new Set(posts.flatMap((post) => post.tags))];
	const rows = posts.map(
		(post) =>
			`(${post.number}, ${quote(post.category)}, ${quote(post.title)}, ${quote(post.slug)}, FALSE, ` +
			`${quote(post.body)}, '${post.publishedAt.toISOString()}', '${post.publishedAt.toISOString()}', NULL)`,
	);
	const links = posts.flatMap((post) =>
		post.tags.map((tag) => `((SELECT id FROM posts WHERE number = ${post.number}), ${quote(tag)})`),
	);

	return `
		INSERT INTO posts (number, category, title, slug, "hideTitle", body, "createdAt", "publishedAt", "updatedAt")
		VALUES ${rows.join(',\n')};
		INSERT INTO tags (name) VALUES ${tags.map((tag) => `(${quote(tag)})`).join(', ')};
		INSERT INTO posts_tags ("postId", "tagName") VALUES ${links.join(',\n')};
	`;
}
