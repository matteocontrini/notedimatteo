import { ADMIN_USERNAME } from './data.ts';
import { expect, logIn, test } from './fixtures.ts';

test('logged-out visitors are sent to the login page', async ({ page }) => {
	for (const path of ['/unpublished', '/1/edit']) {
		await page.goto(path);
		await expect(page).toHaveURL('/login');
	}
});

test('a wrong password shows an error', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Username').fill(ADMIN_USERNAME);
	await page.getByLabel('Password').fill('wrong');
	await page.getByRole('button', { name: 'Accedi' }).click();

	await expect(page.getByText('Credenziali non valide.')).toBeVisible();
	await expect(page.getByLabel('Username')).toHaveValue(ADMIN_USERNAME);
});

test('log in and log out', async ({ page }) => {
	await logIn(page);
	await expect(page.getByRole('link', { name: 'unpublished (0)' })).toBeVisible();

	await page.getByRole('button', { name: 'logout' }).click();
	await expect(page).toHaveURL('/');
	await expect(page.getByRole('link', { name: 'about' })).toBeVisible();

	await page.goto('/unpublished');
	await expect(page).toHaveURL('/login');
});
