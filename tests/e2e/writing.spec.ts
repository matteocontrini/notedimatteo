import type { Page } from '@playwright/test';
import { expect, logIn, test } from './fixtures.ts';

// The markup the editor writes for an uploaded image, with the preview's size
const imageMarkup = (width: number, height: number) =>
	new RegExp(
		String.raw`\[img width=${width} height=${height}\](\d{4}/\d{2}/[0-9a-f]{12})\.jpg\[/img\]`,
	);

// pngDataURL draws a red PNG in the page and returns it as a data URL.
function pngDataURL(page: Page, width: number, height: number) {
	return page.evaluate(
		([width, height]) => {
			const canvas = Object.assign(document.createElement('canvas'), { width, height });
			const context = canvas.getContext('2d')!;
			context.fillStyle = '#c83c28';
			context.fillRect(0, 0, width, height);
			return canvas.toDataURL('image/png');
		},
		[width, height],
	);
}

test('write, publish, and delete a note', async ({ page, request }) => {
	test.setTimeout(60_000);
	const body = page.locator('textarea[name=body]');
	const saved = page.getByText('Saved!');

	await logIn(page);

	const number = await test.step('create a draft', async () => {
		await page.getByRole('button', { name: 'new' }).click();
		await expect(page).toHaveURL(/^http:\/\/localhost:\d+\/\d+\/edit$/);
		await expect(body).toBeFocused();
		return Number(new URL(page.url()).pathname.split('/')[1]);
	});

	await test.step('an empty body is a field error', async () => {
		await page.getByLabel('Title', { exact: true }).fill('Nota di prova');
		await page.getByRole('button', { name: 'Save' }).click();
		await expect(page.getByText('Body is required')).toBeVisible();
	});

	await test.step('save with the keyboard', async () => {
		await body.fill('Una nota di prova');
		await body.press('ControlOrMeta+a');
		await body.press('ControlOrMeta+b');
		await expect(body).toHaveValue('**Una nota di prova**');

		await body.press('ControlOrMeta+s');
		await expect(saved).toBeVisible();
		await expect(page.getByText('Body is required')).toBeHidden();
		await expect(page.getByRole('button', { name: 'Save' })).toBeDisabled();
	});

	await test.step('preview', async () => {
		await body.press('ControlOrMeta+p');
		await expect(page.locator('.post-content strong')).toHaveText('Una nota di prova');
		// The button is inside the Body label, which gives it the name "Body"
		await page.getByRole('button').filter({ hasText: 'Back to editing' }).click();
		await expect(body).toBeFocused();
	});

	await test.step('tag suggestions', async () => {
		const tags = page.getByLabel('Tags (comma separated)');
		await tags.fill('sv');
		await tags.press('Tab');
		await expect(tags).toHaveValue('svelte');
	});

	const pastedKey = await test.step('paste an image', async () => {
		const dataURL = await pngDataURL(page, 1600, 900);
		await body.evaluate(async (textarea: HTMLTextAreaElement, dataURL) => {
			const blob = await (await fetch(dataURL)).blob();
			const data = new DataTransfer();
			data.items.add(new File([blob], 'incollata.png', { type: 'image/png' }));
			textarea.focus();
			textarea.setSelectionRange(textarea.value.length, textarea.value.length);
			textarea.dispatchEvent(new ClipboardEvent('paste', { clipboardData: data, bubbles: true }));
		}, dataURL);

		await expect(body).toHaveValue(imageMarkup(1200, 675), { timeout: 15_000 });
		return (await body.inputValue()).match(imageMarkup(1200, 675))![1];
	});

	await test.step('upload a chosen image', async () => {
		const dataURL = await pngDataURL(page, 300, 200);
		await page.locator('input[type=file]').setInputFiles({
			name: 'scelta.png',
			mimeType: 'image/png',
			buffer: Buffer.from(dataURL.split(',')[1], 'base64'),
		});
		await page.getByRole('button', { name: 'Upload file' }).click();
		await expect(body).toHaveValue(imageMarkup(300, 200), { timeout: 15_000 });
	});

	await test.step('the storage has both versions of the pasted image', async () => {
		for (const key of [pastedKey, `${pastedKey}_preview`]) {
			const response = await request.get(`${process.env.E2E_STORAGE_URL}/${key}.jpg`);
			expect(response.ok(), key).toBe(true);
			expect(response.headers()['content-type']).toBe('image/jpeg');
		}
	});

	await test.step('publish with the keyboard', async () => {
		await body.press('ControlOrMeta+Shift+s');
		await expect(saved).toBeVisible();
		await expect(page.getByRole('button', { name: 'Unpublish' })).toBeVisible();
		await expect(page.getByText('Published at')).toBeVisible();
	});

	await test.step('the note is on the home page and in the feed', async () => {
		await page.goto('/');
		const note = page.locator('article').first();
		await expect(note.getByRole('heading', { name: 'Nota di prova' })).toBeVisible();
		await expect(note.getByRole('link', { name: '#svelte' })).toBeVisible();

		const image = note.locator('img').first();
		await expect(image).toHaveAttribute('src', new RegExp(`/${pastedKey}_preview\\.jpg$`));
		await expect.poll(() => image.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBe(1200);

		expect(await (await request.get('/feed.xml')).text()).toContain('Nota di prova');
	});

	await test.step('a number that another post has is a field error', async () => {
		await page.goto(`/${number}/edit`);
		const numberField = page.locator('input[name=number]');
		await numberField.fill('1');
		await page.getByRole('button', { name: 'Save' }).click();
		await expect(page.getByText('Post #1 already exists')).toBeVisible();
		await numberField.fill(String(number));
	});

	await test.step('unpublish', async () => {
		await page.getByRole('button', { name: 'Unpublish' }).click();
		await expect(saved).toBeVisible();
		await expect(page.getByRole('button', { name: 'Publish', exact: true })).toBeVisible();

		await page.goto('/');
		await expect(page.getByRole('heading', { name: 'Nota di prova' })).toHaveCount(0);
	});

	await test.step('delete the draft', async () => {
		await page.goto('/unpublished');
		const editLink = page.locator(`a[href="/${number}/edit"]`);
		await expect(editLink).toBeVisible();

		page.once('dialog', (dialog) => dialog.accept());
		await page.locator(`form[action="/${number}/delete"]`).getByRole('button').click();
		await expect(page).toHaveURL('/unpublished');
		await expect(editLink).toHaveCount(0);
	});
});
