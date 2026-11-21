import { afterEach, describe, expect, it, vi } from 'vitest';
import { formFromPost, sameForm, savePost, toFormData, type EditorPost } from './form';

const post: EditorPost = {
	id: 7,
	number: 5,
	category: 'note',
	title: '',
	slug: 'a-slug',
	hideTitle: false,
	body: 'Body',
	tags: ['go', 'svelte'],
	createdAt: '2026-11-21T10:00:00+01:00',
	updatedAt: null,
	publishedAt: null,
	urls: {
		post: '/5/a-slug',
		edit: '/5/edit',
		preview: '/5/edit/preview',
		upload: '/5/edit/upload',
	},
};

const respond = (status: number, body: unknown) =>
	vi.spyOn(globalThis, 'fetch').mockResolvedValue(Response.json(body, { status }));

afterEach(() => {
	vi.restoreAllMocks();
});

describe('post form', () => {
	it('starts from the post, with tags separated by commas', () => {
		expect(formFromPost(post)).toEqual({
			number: 5,
			category: 'note',
			title: '',
			slug: 'a-slug',
			hideTitle: false,
			body: 'Body',
			tags: 'go, svelte',
		});
	});

	it('compares every field', () => {
		const form = formFromPost(post);
		expect(sameForm(form, { ...form })).toBe(true);
		expect(sameForm(form, { ...form, hideTitle: true })).toBe(false);
		expect(sameForm(form, { ...form, number: null })).toBe(false);
	});

	it('sends the fields the server reads', () => {
		const data = toFormData({ ...formFromPost(post), number: null, hideTitle: true }, 'publish');
		expect(Object.fromEntries(data)).toEqual({
			intent: 'publish',
			number: '',
			category: 'note',
			title: '',
			slug: 'a-slug',
			hideTitle: 'true',
			body: 'Body',
			tags: 'go, svelte',
		});
	});
});

describe('savePost', () => {
	it('returns the saved post', async () => {
		const fetch = respond(200, { post });
		expect(await savePost('/5/edit', formFromPost(post), 'save')).toEqual({ post });
		expect(fetch).toHaveBeenCalledWith('/5/edit', expect.objectContaining({ method: 'POST' }));
	});

	it('returns field errors', async () => {
		respond(400, { errors: { number: 'Post #6 already exists' } });
		expect(await savePost('/5/edit', formFromPost(post), 'save')).toEqual({
			errors: { number: 'Post #6 already exists' },
		});
	});

	it('throws other errors', async () => {
		respond(401, { error: 'Unauthorized' });
		await expect(savePost('/5/edit', formFromPost(post), 'save')).rejects.toThrow('Unauthorized');

		vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('<html>', { status: 502 }));
		await expect(savePost('/5/edit', formFromPost(post), 'save')).rejects.toThrow('502');
	});
});
