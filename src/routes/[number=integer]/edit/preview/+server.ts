import { renderMarkdown } from '$lib/server/markdown';
import { json, redirect } from '@sveltejs/kit';
import { z } from 'zod/v4';
import type { RequestHandler } from './$types';

const payloadSchema = z.object({
	body: z.string()
});

export const POST: RequestHandler = async ({ locals, request }) => {
	if (!locals.session) {
		return redirect(303, '/login');
	}

	let parsedBody: unknown;

	try {
		parsedBody = await request.json();
	} catch (error) {
		console.error('Error parsing JSON body:', error);
		return json({ error: 'Invalid JSON body' }, { status: 400 });
	}

	const result = payloadSchema.safeParse(parsedBody);

	if (!result.success) {
		return json({ error: 'Body is required' }, { status: 400 });
	}

	const html = await renderMarkdown(result.data.body);

	return json({ html });
};
