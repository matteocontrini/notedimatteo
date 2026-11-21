import { test as base, expect, type Page } from '@playwright/test';
import { ADMIN_PASSWORD, ADMIN_USERNAME } from './data.ts';

export { expect };

// test is Playwright's test with two changes to every page. Requests to other
// hosts fail, so GoatCounter's script never loads. An uncaught JavaScript error
// fails the test, because a broken script often shows nothing else.
export const test = base.extend({
	page: async ({ page }, use) => {
		await page.route(
			(url) => url.hostname !== 'localhost',
			(route) => route.abort(),
		);
		const errors: Error[] = [];
		page.on('pageerror', (error) => errors.push(error));

		await use(page);

		expect(errors, 'uncaught JavaScript errors').toEqual([]);
	},
});

export async function logIn(page: Page) {
	await page.goto('/login');
	await page.getByLabel('Username').fill(ADMIN_USERNAME);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Accedi' }).click();
	await expect(page).toHaveURL('/');
}
