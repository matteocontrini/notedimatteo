import { posts } from './data.ts';
import { expect, test } from './fixtures.ts';

const pageSize = 20;

test('home page and pagination', async ({ page }) => {
	await page.goto('/');
	await expect(page.locator('article')).toHaveCount(pageSize);
	await expect(page.getByText(`Pagina 1 di ${Math.ceil(posts.length / pageSize)}`)).toBeVisible();

	await page.getByRole('link', { name: 'Successiva →' }).click();
	await expect(page).toHaveURL('/?page=2');
	await expect(page.locator('article')).toHaveCount(posts.length - pageSize);
});

test('a post redirects to its slug', async ({ page }) => {
	await page.goto('/24');
	await expect(page).toHaveURL('/24/il-post-piu-recente');
	await expect(page).toHaveTitle(/Il post più recente/);
	await expect(page.getByRole('heading', { name: 'Il post più recente' })).toBeVisible();
});

test('categories, tags, months, and search', async ({ page }) => {
	await test.step('category', async () => {
		const count = posts.filter((post) => post.category === 'til').length;
		await page.goto('/');
		await page.getByRole('link', { name: `TIL ${count}` }).click();
		await expect(page).toHaveURL('/til');
		await expect(page.locator('article')).toHaveCount(count);
	});

	await test.step('tag', async () => {
		await page.goto('/archivio/go');
		await expect(page.getByRole('heading', { name: '#go' })).toBeVisible();
		await expect(page.locator('article')).toHaveCount(
			posts.filter((post) => post.tags.includes('go')).length,
		);
	});

	await test.step('month', async () => {
		await page.goto('/archivio/2026/09');
		await expect(page.getByRole('heading', { name: 'Settembre 2026' })).toBeVisible();
		await expect(page.locator('article')).toHaveCount(
			posts.filter((post) => post.publishedAt.getUTCMonth() === 8).length,
		);
	});

	await test.step('search', async () => {
		await page.goto('/');
		await page.getByRole('searchbox', { name: 'Cerca nel sito' }).fill('zucchero');
		await page.keyboard.press('Enter');
		await expect(page).toHaveURL('/cerca?q=zucchero');
		await expect(page.getByText('1 risultato')).toBeVisible();
		await expect(page.locator('article')).toContainText('Una nota sullo zucchero.');
	});
});

test('feed', async ({ request }) => {
	const response = await request.get('/feed.xml');
	expect(response.headers()['content-type']).toContain('application/xml');

	const feed = await response.text();
	expect(feed.match(/<item>/g)).toHaveLength(posts.length);
	expect(feed).toContain('Il post più recente');
});

test('missing pages', async ({ page }) => {
	const response = await page.goto('/9999');
	expect(response?.status()).toBe(404);
	await expect(page.getByText('Not Found')).toBeVisible();
});

test('the theme choice stays after a reload', async ({ page }) => {
	await page.emulateMedia({ colorScheme: 'light' });
	await page.goto('/');
	await expect(page.locator('html')).not.toHaveClass(/dark/);

	await page.getByRole('combobox').selectOption('Scuro');
	await expect(page.locator('html')).toHaveClass(/dark/);
	await page.reload();
	await expect(page.locator('html')).toHaveClass(/dark/);
});
