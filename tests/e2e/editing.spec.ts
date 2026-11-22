import type { Locator, Page } from '@playwright/test';
import { expect, logIn, test } from './fixtures.ts';

// preview opens the server's rendering of the body, which is what the post
// will show, runs check on it, and goes back to editing.
async function preview(page: Page, check: (content: Locator) => Promise<void>) {
	await page.keyboard.press('ControlOrMeta+p');
	await check(page.locator('.post-content'));
	await page.getByRole('button').filter({ hasText: 'Back to editing' }).click();
}

test('the body editor hides blank lines and keeps paragraphs with Enter', async ({ page }) => {
	const body = page.getByRole('textbox', { name: 'Body' });

	await logIn(page);
	await page.getByRole('button', { name: 'new' }).click();
	await expect(body).toBeFocused();

	await test.step('Enter starts paragraphs, and blank lines are hidden', async () => {
		await body.pressSequentially('Primo');
		await body.press('Enter');
		await body.pressSequentially('Secondo');
		await body.press('Enter');
		await body.pressSequentially('Terzo');

		// Three lines and the two gaps between them, but no blank lines
		await expect(body.locator('.cm-line')).toHaveText(['Primo', 'Secondo', 'Terzo']);
		await expect(body.locator('.cm-md-gap')).toHaveCount(2);

		await preview(page, async (content) => {
			await expect(content.locator('p')).toHaveText(['Primo', 'Secondo', 'Terzo']);
		});
	});

	await test.step('the arrows cross a paragraph break in one step', async () => {
		await body.locator('.cm-line').nth(1).click();
		await body.press('Home');
		await body.press('ArrowLeft');
		await body.pressSequentially('!');
		await expect(body.locator('.cm-line')).toHaveText(['Primo!', 'Secondo', 'Terzo']);
	});

	await test.step('Backspace at the start of a paragraph joins it to the one above', async () => {
		await body.locator('.cm-line').nth(2).click();
		await body.press('Home');
		await body.press('Backspace');
		await expect(body.locator('.cm-line')).toHaveText(['Primo!', 'SecondoTerzo']);

		await preview(page, async (content) => {
			await expect(content.locator('p')).toHaveText(['Primo!', 'SecondoTerzo']);
		});
	});

	await test.step('Enter at the start of a paragraph adds an empty one above it', async () => {
		await body.locator('.cm-line').nth(1).click();
		await body.press('Home');
		await body.press('Enter');
		await expect(body.locator('.cm-line')).toHaveText(['Primo!', '', 'SecondoTerzo']);

		// The cursor stays with the paragraph
		await body.pressSequentially('X');
		await expect(body.locator('.cm-line')).toHaveText(['Primo!', '', 'XSecondoTerzo']);
		await body.press('Backspace');

		// The empty paragraph stays when the cursor moves away, and takes text
		await body.locator('.cm-line').first().click();
		await expect(body.locator('.cm-line')).toHaveText(['Primo!', '', 'SecondoTerzo']);
		await body.locator('.cm-line').nth(1).click();
		await body.pressSequentially('Nuovo');

		await preview(page, async (content) => {
			await expect(content.locator('p')).toHaveText(['Primo!', 'Nuovo', 'SecondoTerzo']);
		});
	});

	await test.step('Enter on an empty list item leaves the list', async () => {
		await body.press('ControlOrMeta+End');
		await body.press('Enter');
		await body.pressSequentially('- uno');
		await body.press('Enter');
		await body.pressSequentially('due');
		await body.press('Enter');
		await body.press('Enter');
		await body.pressSequentially('Dopo la lista');

		await preview(page, async (content) => {
			await expect(content.locator('li')).toHaveText(['uno', 'due']);
			await expect(content.locator('p').last()).toHaveText('Dopo la lista');
		});
	});

	await test.step('pasted formatted text becomes Markdown', async () => {
		await body.press('ControlOrMeta+End');
		await body.press('Enter');
		await body.evaluate((editor) => {
			const data = new DataTransfer();
			data.setData(
				'text/html',
				'<p>Un <a href="https://example.com/">collegamento</a> e <b>grassetto</b></p>',
			);
			data.setData('text/plain', 'Un collegamento e grassetto');
			editor.dispatchEvent(new ClipboardEvent('paste', { clipboardData: data, bubbles: true }));
		});
		await expect(body.locator('.cm-line').last()).toHaveText(
			'Un [collegamento](https://example.com/) e **grassetto**',
		);

		await preview(page, async (content) => {
			await expect(content.getByRole('link', { name: 'collegamento' })).toHaveAttribute(
				'href',
				'https://example.com/',
			);
		});
	});

	await test.step('Enter closes a code block, and Tab and Shift+Tab indent in it but move the focus elsewhere', async () => {
		await body.press('ControlOrMeta+End');
		await body.press('Enter');
		await body.pressSequentially('```');
		await body.press('Enter');
		// Enter after the opening fence adds the closing one
		const code = body.locator('.cm-md-code');
		await expect(code).toHaveText(['```', '', '```']);

		await body.press('Tab');
		await body.pressSequentially('rientrato');
		await expect(code.nth(1)).toHaveText('    rientrato');
		await body.press('Shift+Tab');
		await expect(code.nth(1)).toHaveText('rientrato');

		await body.locator('.cm-line').first().click();
		await body.press('Tab');
		await expect(body).not.toBeFocused();
	});
});
