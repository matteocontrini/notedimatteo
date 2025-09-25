import { db } from '$lib/server/db';
import type { PostForEdit } from '$lib/types';
import { error, redirect, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { superValidate } from 'sveltekit-superforms/server';
import { zod4 } from 'sveltekit-superforms/adapters';
import type { SuperValidated } from 'sveltekit-superforms';
import type { ZodValidationSchema } from 'sveltekit-superforms/adapters';
import { postFormSchema, type PostFormValues } from './schema';

const validationSchema = postFormSchema as unknown as ZodValidationSchema;

export const load = (async ({ params }) => {
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

const mutatePost = async ({
	params,
	form,
	publish
}: {
	params: { number: string };
	form: SuperValidated<PostFormValues>;
	publish: boolean;
}) => {
	const currentNumber = Number(params.number);
	if (!Number.isInteger(currentNumber)) {
		error(404);
	}

	const postRecord = await db.post.findUnique({
		where: { number: currentNumber },
		select: { id: true }
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
				...(publish ? { publishedAt: now } : {})
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
		const form = await superValidate(event.request, zod4(validationSchema));
		if (!form.valid) {
			return fail(400, { form });
		}

		const result = await mutatePost({
			params: event.params,
			form: form as SuperValidated<PostFormValues>,
			publish: false
		});
		throw redirect(303, `/${result.number}/edit`);
	},
	publish: async (event) => {
		const form = await superValidate(event.request, zod4(validationSchema));
		if (!form.valid) {
			return fail(400, { form });
		}

		const result = await mutatePost({
			params: event.params,
			form: form as SuperValidated<PostFormValues>,
			publish: true
		});
		const target = result.slug ? `/${result.number}/${result.slug}` : `/${result.number}`;
		throw redirect(303, target);
	}
} satisfies Actions;
