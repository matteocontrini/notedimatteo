<script lang="ts">
	import type { PostForEdit } from '$lib/types';
	import { superForm } from 'sveltekit-superforms/client';
	import type { SuperValidated } from 'sveltekit-superforms';
	import type { PostFormValues } from './schema';

	let { data } = $props<{ data: { post: PostForEdit; form: SuperValidated<PostFormValues> } }>();

	const postForm = superForm<PostFormValues>(data.form, {
		resetForm: false
	});

	const { form: formData, errors: formErrors, enhance } = postForm;

	let formElement: HTMLFormElement;

	const handleKeydown = (event: KeyboardEvent) => {
		if ((event.metaKey || event.ctrlKey) && (event.key === 's' || event.key === 'S')) {
			event.preventDefault();
			formElement?.requestSubmit();
		}
	};

	const formatDatetimeLocal = (value: Date | null) =>
		value ? new Date(value).toISOString().slice(0, 16) : '';

	const TEXTAREA_BUFFER_PX = 12;

	const autoResize = (node: HTMLTextAreaElement, value?: string) => {
		const resize = () => {
			node.style.height = 'auto';
			node.style.height = `${node.scrollHeight + TEXTAREA_BUFFER_PX}px`;
		};

		resize();

		node.addEventListener('input', resize);

		return {
			destroy() {
				node.removeEventListener('input', resize);
			},
			update(newValue?: string) {
				// ensure resize after content changes programmatically
				if (newValue !== value) {
					value = newValue;
					resize();
				}
			}
		};
	};
</script>

<svelte:window on:keydown={handleKeydown} />

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

		{#if $formErrors.title}
			<p class="mt-1 text-sm text-red-600">{$formErrors.title[0]}</p>
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
			></textarea>
		</label>

		{#if $formErrors.body}
			<p class="mt-1 text-sm text-red-600">{$formErrors.body[0]}</p>
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
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Number</span>
				<input class="w-full h-9" type="number" name="number" bind:value={$formData.number} />

				{#if $formErrors.number}
					<p class="mt-1 text-sm text-red-600">{$formErrors.number[0]}</p>
				{/if}
			</label>

			<label class="block md:col-span-2">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Category</span>
				<input class="w-full h-9" name="category" bind:value={$formData.category} />

				{#if $formErrors.category}
					<p class="mt-1 text-sm text-red-600">{$formErrors.category[0]}</p>
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
			<button type="submit" class="w-full" name="intent" value="save">
				Save
			</button>

			{#if data.post.publishedAt === null}
				<button type="submit" class="w-full" name="intent" value="publish">
					Save & publish
				</button>
			{:else}
				<button type="submit" class="w-full" name="intent" value="unpublish">
					Save & unpublish
				</button>
			{/if}
		</section>
	</aside>
</form>
