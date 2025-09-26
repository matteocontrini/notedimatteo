import { db } from '$lib/server/db';
import type { PostForEdit } from '$lib/types';
import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { superValidate } from 'sveltekit-superforms/server';
import { zod4 } from 'sveltekit-superforms/adapters';
import type { SuperValidated } from 'sveltekit-superforms';
import type { ZodValidationSchema } from 'sveltekit-superforms/adapters';
import { postFormSchema, type PostFormValues } from './schema';

const validationSchema = postFormSchema as unknown as ZodValidationSchema;

export const load = (async ({ params, locals }) => {
	if (!locals.session) {
		return redirect(303, '/login');
	}

	const number = Number(params.number);

	if (!Number.isInteger(number)) {
		error(404);
	}

	const postRecord = await db.post.findUnique({
		where: { number },
		include: {
			postsToTags: {
				select: {
					tag: { select: { name: true } }
				}
			}
		}
	});

	if (!postRecord) {
		error(404);
	}

	const post: PostForEdit = {
		id: postRecord.id,
		number: postRecord.number,
		category: postRecord.category,
		createdAt: postRecord.createdAt,
		updatedAt: postRecord.updatedAt,
		publishedAt: postRecord.publishedAt,
		title: postRecord.title,
		slug: postRecord.slug,
		hideTitle: postRecord.hideTitle,
		body: postRecord.body,
		tags: postRecord.postsToTags.map((entry) => entry.tag.name)
	};

	const form = await superValidate(
		{
			number: post.number,
			category: post.category,
			title: post.title ?? '',
			slug: post.slug ?? '',
			hideTitle: post.hideTitle,
			body: post.body,
			tags: post.tags.join(', ')
		},
		zod4(validationSchema)
	);

	return { post, form: form as SuperValidated<PostFormValues> };
}) satisfies PageServerLoad;

const parseTags = (value: string) =>
	Array.from(
		new Set(
			value
				.split(',')
				.map((tag) => tag.trim())
				.filter((tag) => tag.length > 0)
		)
	);

type PostMutationIntent = 'save' | 'publish' | 'unpublish';

const mutatePost = async ({
	params,
	form,
	intent
}: {
	params: { number: string };
	form: SuperValidated<PostFormValues>;
	intent: PostMutationIntent;
}) => {
	const currentNumber = Number(params.number);
	if (!Number.isInteger(currentNumber)) {
		error(404);
	}

	const postRecord = await db.post.findUnique({
		where: { number: currentNumber },
		select: { id: true, publishedAt: true }
	});

	if (!postRecord) {
		error(404);
	}

	const { number: nextNumber, category, title, slug, hideTitle, body, tags } = form.data;
	const normalisedTitle = title === '' ? null : title;
	const normalisedSlug = slug === '' ? null : slug;
	const parsedTags = parseTags(tags);
	const now = new Date();

	await db.$transaction(async (tx) => {
		await tx.post.update({
			where: { id: postRecord.id },
			data: {
				number: nextNumber,
				category,
				title: normalisedTitle,
				slug: normalisedSlug,
				hideTitle,
				body,
				updatedAt: now,
				publishedAt:
					intent === 'publish' ? now : intent === 'unpublish' ? null : postRecord.publishedAt
			}
		});

		await tx.postsToTags.deleteMany({ where: { postId: postRecord.id } });

		if (parsedTags.length > 0) {
			await Promise.all(
				parsedTags.map((name) =>
					tx.tag.upsert({
						where: { name },
						update: {},
						create: { name }
					})
				)
			);

			await tx.postsToTags.createMany({
				data: parsedTags.map((name) => ({
					postId: postRecord.id,
					tagName: name
				}))
			});
		}
	});

	return { number: nextNumber, slug: normalisedSlug };
};

export const actions = {
	save: async (event) => {
		if (!event.locals.session) {
			return redirect(303, '/login');
		}

		const rawFormData = await event.request.formData();
		const intentValue = rawFormData.get('intent');
		const form = await superValidate(rawFormData, zod4(validationSchema));
		if (!form.valid) {
			return fail(400, { form });
		}

		const intent = intentValue as PostMutationIntent;

		await mutatePost({
			params: event.params,
			form: form as SuperValidated<PostFormValues>,
			intent
		});

		return { form };
	}
} satisfies Actions;
