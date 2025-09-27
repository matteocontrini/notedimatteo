<script lang="ts">
	import type { PostForEdit } from '$lib/types';
	import { superForm } from 'sveltekit-superforms/client';
	import type { SuperValidated } from 'sveltekit-superforms';
	import type { PostFormValues } from './schema';
	import { categorySlugs, getCategoryLabel } from '$lib/categories';
	import { autoResize } from '$lib/actions/auto-resize';
	import { resolve } from '$app/paths';

	let { data } = $props<{ data: { post: PostForEdit; form: SuperValidated<PostFormValues> } }>();

	let showSaved = $state(false);
	let savedTimer: ReturnType<typeof setTimeout> | undefined;

	const postForm = superForm<PostFormValues>(data.form, {
		resetForm: false,
		delayMs: 200,
		onResult({ result }) {
			if (result.type === 'success') {
				showSaved = true;
				clearTimeout(savedTimer);
				savedTimer = setTimeout(() => (showSaved = false), 1000);
			}
		},
		onError({ result }) {
			console.error(result);
			if (result.error.message) {
				alert('Error: ' + result.error.message);
			} else {
				alert('Unknown error');
			}
		}
	});

	const { form: formData, errors: errors, enhance, delayed, tainted, isTainted } = postForm;

	const categoryOptions = categorySlugs.map(slug => ({ slug, label: getCategoryLabel(slug) }));

	let formElement: HTMLFormElement;

	const handleKeydown = (event: KeyboardEvent) => {
		if ((event.metaKey || event.ctrlKey) && (event.key === 's' || event.key === 'S')) {
			event.preventDefault();
			formElement?.requestSubmit();
		}
	};

	const formatDatetimeLocal = (value: Date | null) =>
		value ? new Date(value).toISOString().slice(0, 16) : '';

</script>

<svelte:window on:keydown={handleKeydown} />

{#if $delayed || showSaved}
	<div
		class="fixed left-0 top-0 h-9 bg-white dark:bg-slate-900 w-full flex items-center border-b border-slate-200 dark:border-slate-700">
		<div class="container max-w-6xl dark:text-white">
			{#if $delayed}
				Saving...
			{:else}
				<span class="text-green-500 font-medium">Saved!</span>
			{/if}
		</div>
	</div>
{/if}

<form
	class="grid gap-12 md:grid-cols-12"
	bind:this={formElement}
	method="POST"
	action="?/save"
	use:enhance
>
	<div class="md:col-span-7">
		<label class="block">
			<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Title</span>
			<input class="w-full h-9" name="title" bind:value={$formData.title} />
		</label>

		{#if $errors.title}
			<p class="mt-1 text-sm text-red-600">{$errors.title[0]}</p>
		{/if}

		<label class="mt-3 flex items-center gap-3">
			<input
				class="accent-slate-800 dark:accent-slate-200"
				type="checkbox"
				name="hideTitle"
				bind:checked={$formData.hideTitle}
			/>
			<span class="text-sm text-slate-700 dark:text-slate-300">Hide title</span>
		</label>

		<label class="block mt-6">
			<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Body</span>
			<textarea
				class="w-full resize-none overflow-hidden"
				use:autoResize={$formData.body}
				name="body"
				rows="16"
				bind:value={$formData.body}
				autofocus
			></textarea>
		</label>

		{#if $errors.body}
			<p class="mt-1 text-sm text-red-600">{$errors.body[0]}</p>
		{/if}
	</div>

	<aside class="md:col-span-5 space-y-6">
		<section class="grid gap-4 md:grid-cols-2">
			<label class="block">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">ID</span>
				<input
					class="w-full h-9"
					name="id"
					value={data.post.id.toString()}
					readonly
				/>
			</label>

			<label class="block md:col-start-2">
				<span class="flex text-sm text-slate-700 dark:text-slate-300 mb-2">
					<span class="font-medium">
						Number
					</span>
					<a
						href={resolve('/[number=integer]/[[slug]]', {number: data.post.number.toString(), slug: data.post.slug ?? undefined})}
						class="link ml-auto">
						open post
					</a>
				</span>
				<input class="w-full h-9" type="number" name="number" bind:value={$formData.number} />

				{#if $errors.number}
					<p class="mt-1 text-sm text-red-600">{$errors.number[0]}</p>
				{/if}
			</label>

			<label class="block md:col-span-2">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Category</span>
				<select class="w-full h-9 py-0" name="category" bind:value={$formData.category}>
					{#each categoryOptions as option (option.slug)}
						<option value={option.slug}>{option.label}</option>
					{/each}
				</select>

				{#if $errors.category}
					<p class="mt-1 text-sm text-red-600">{$errors.category[0]}</p>
				{/if}
			</label>

			<label class="block md:col-span-2">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Slug</span>
				<input class="w-full h-9" name="slug" bind:value={$formData.slug} />
			</label>

			<label class="block md:col-span-2">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Tags (comma separated)</span>
				<input class="w-full h-9" name="tags" bind:value={$formData.tags} />
			</label>
		</section>

		<section class="grid gap-4">
			<label class="block">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Created at</span>
				<input class="w-full h-9" type="datetime-local" name="createdAt" readonly
							 value={formatDatetimeLocal(data.post.createdAt)} />
			</label>

			{#if data.post.updatedAt != null}
				<label class="block">
					<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Updated at</span>
					<input class="w-full h-9" type="datetime-local" name="updatedAt" readonly
								 value={formatDatetimeLocal(data.post.updatedAt)} />
				</label>
			{/if}

			{#if data.post.publishedAt !== null}
				<label class="block">
					<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Published at</span>
					<input class="w-full h-9" type="datetime-local" name="publishedAt" readonly
								 value={formatDatetimeLocal(data.post.publishedAt)} />
				</label>
			{/if}
		</section>

		<section class="space-y-4">
			<button type="submit" class="w-full" name="intent" value="save" disabled={!isTainted($tainted)}>
				Save
			</button>

			{#if data.post.publishedAt === null}
				<button type="submit" class="w-full" name="intent" value="publish">
					Publish
				</button>
			{:else}
				<button type="submit" class="w-full" name="intent" value="unpublish">
					Unpublish
				</button>
			{/if}
		</section>
	</aside>
</form>
