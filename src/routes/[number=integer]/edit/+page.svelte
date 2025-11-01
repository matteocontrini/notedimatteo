<script lang="ts">
	import type { PostForEdit } from '$lib/types';
	import { superForm } from 'sveltekit-superforms/client';
	import type { SuperValidated } from 'sveltekit-superforms';
	import type { PostFormValues } from './schema';
	import { categorySlugs, getCategoryLabel } from '$lib/categories';
	import { autoResize } from '$lib/actions/auto-resize';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

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

	let showPreview = $state(false);
	let previewLoading = $state(false);
	let previewHtml = $state('');
	let previewError = $state<string | null>(null);

	let fileInput: HTMLInputElement;
	let uploadLoading = $state(false);

	const uploadFile = async (file: File, cursorPos?: number, textarea?: HTMLTextAreaElement) => {
		const formData = new FormData();
		formData.append('file', file);

		uploadLoading = true;

		try {
			const response = await fetch(`${page.url.pathname}/upload`, {
				method: 'POST',
				body: formData
			});

			if (!response.ok) {
				throw new Error('Upload failed');
			}

			const result = await response.json() as {
				main: string;
				preview: string;
				width: number;
				height: number;
			};

			// Add image markdown to the bottom of the text body with dimensions
			const imageMarkdown = `[img width=${result.width} height=${result.height}]${result.main}[/img]`;

			// Use the update method to modify the store value
			postForm.form.update(form => {
				// If cursor position is provided, insert at cursor with smart spacing
				if (cursorPos !== undefined) {
					const before = form.body.substring(0, cursorPos);
					const after = form.body.substring(cursorPos);

					// Smart spacing: only add newlines if needed
					let spacingBefore = '';
					let spacingAfter = '';

					// Check if we need spacing before the image
					if (before.length > 0 && !before.endsWith('\n\n')) {
						if (before.endsWith('\n')) {
							spacingBefore = '\n';
						} else {
							spacingBefore = '\n\n';
						}
					}

					// Check if we need spacing after the image
					if (after.length > 0 && !after.startsWith('\n\n')) {
						if (after.startsWith('\n')) {
							spacingAfter = '\n';
						} else {
							spacingAfter = '\n\n';
						}
					}

					const newBody = before + spacingBefore + imageMarkdown + spacingAfter + after;

					// Calculate new cursor position (after the inserted image)
					const newCursorPos = before.length + spacingBefore.length + imageMarkdown.length + spacingAfter.length;

					// Restore cursor position after state update
					if (textarea) {
						setTimeout(() => {
							textarea.setSelectionRange(newCursorPos, newCursorPos);
							textarea.focus();
						}, 0);
					}

					return {
						...form,
						body: newBody
					};
				} else {
					// Fallback: append to end (for file input upload)
					return {
						...form,
						body: form.body + '\n\n' + imageMarkdown
					};
				}
			});

			return true;
		} catch (error) {
			console.error('Upload error:', error);
			alert('Upload failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
			return false;
		} finally {
			uploadLoading = false;
		}
	};

	const upload = async () => {
		if (!fileInput.files || fileInput.files.length === 0) {
			alert('Please select a file to upload');
			return;
		}

		const file = fileInput.files[0];
		const success = await uploadFile(file);

		if (success) {
			fileInput.value = ''; // Clear the input
		}
	};

	const handlePaste = async (event: ClipboardEvent) => {
		const items = event.clipboardData?.items;
		if (!items) return;

		// Look for image files in the clipboard
		for (let i = 0; i < items.length; i++) {
			const item = items[i];
			if (item.type.startsWith('image/')) {
				event.preventDefault(); // Prevent default paste behavior for images

				const file = item.getAsFile();
				if (file) {
					// Capture cursor position before upload
					const textarea = event.target as HTMLTextAreaElement;
					const cursorPos = textarea.selectionStart;
					await uploadFile(file, cursorPos, textarea);
				}
				break; // Only handle the first image found
			}
		}
	};

	const insertMarkdownLink = (textarea: HTMLTextAreaElement) => {
		const selectionStart = textarea.selectionStart;
		const selectionEnd = textarea.selectionEnd;
		const selectedText = textarea.value.substring(selectionStart, selectionEnd);

		let newBody: string;
		let newCursorPos: number;

		if (selectedText.length === 0) {
			// Case 1: No selection - insert []() with cursor at link text position
			const before = textarea.value.substring(0, selectionStart);
			const after = textarea.value.substring(selectionEnd);
			newBody = before + '[]()' + after;
			newCursorPos = selectionStart + 1; // Position cursor inside brackets: [|]()
		} else if (selectedText.startsWith('http://') || selectedText.startsWith('https://')) {
			// Case 2: Selected text is URL - create [](url) with cursor at link text position
			const before = textarea.value.substring(0, selectionStart);
			const after = textarea.value.substring(selectionEnd);
			newBody = before + '[](' + selectedText + ')' + after;
			newCursorPos = selectionStart + 1; // Position cursor inside brackets: [|](url)
		} else {
			// Case 3: Selected text is regular text - create [text]() with cursor at URL position
			const before = textarea.value.substring(0, selectionStart);
			const after = textarea.value.substring(selectionEnd);
			newBody = before + '[' + selectedText + ']()' + after;
			newCursorPos = selectionStart + selectedText.length + 3; // Position cursor inside parentheses: [text](|)
		}

		// Update form data
		postForm.form.update(form => {
			return {
				...form,
				body: newBody
			};
		});

		// Restore cursor position after state update
		setTimeout(() => {
			textarea.setSelectionRange(newCursorPos, newCursorPos);
			textarea.focus();
		}, 0);
	};

	const loadPreview = async () => {
		previewLoading = true;
		previewError = null;
		showPreview = true;

		const previewEndpoint = resolve('/[number=integer]/edit/preview', {
			number: data.post.number.toString()
		});

		try {
			const response = await fetch(previewEndpoint, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ body: $formData.body })
			});

			if (!response.ok) {
				throw new Error('Unable to render preview');
			}

			const result = (await response.json()) as { html: string };
			previewHtml = result.html;
		} catch (error) {
			previewHtml = '';
			previewError = error instanceof Error ? error.message : 'Unexpected error while rendering preview';
		} finally {
			previewLoading = false;
		}
	};

	const closePreview = () => {
		showPreview = false;
		previewError = null;
	};

	const handleKeydown = (event: KeyboardEvent) => {
		if ((event.metaKey || event.ctrlKey) && (event.key === 's' || event.key === 'S')) {
			event.preventDefault();
			formElement?.requestSubmit();
		}

		// Cmd+K: Insert markdown link
		if ((event.metaKey || event.ctrlKey) && (event.key === 'k' || event.key === 'K')) {
			event.preventDefault();
			const textarea = event.target as HTMLTextAreaElement;
			if (textarea && textarea.tagName === 'TEXTAREA') {
				insertMarkdownLink(textarea);
			}
		}
	};

	const formatDatetimeLocal = (value: Date | null) => {
		if (!value) {
			return '';
		}

		// 2000-01-01T00:00:00
		const date = value.toLocaleDateString('sv-SE', { timeZone: 'Europe/Rome' });
		const time = value.toLocaleTimeString('sv-SE', { hour12: false, timeZone: 'Europe/Rome' });
		return `${date}T${time}`;
	};

	const normalizeSlug = (value: string) => {
		return value
			.toLowerCase()
			.normalize('NFD') // Decompose combined characters
			.replace(/[\u0300-\u036f]/g, '') // Remove diacritics (accents)
			.replace(/\s+/g, '-') // Replace spaces with hyphens
			.replace(/[^\w-]+/g, '') // Remove non-word chars except hyphens
			.replace(/--+/g, '-'); // Replace multiple hyphens with single hyphen
	};

	const handleSlugInput = (event: Event) => {
		const input = event.target as HTMLInputElement;
		const oldValue = input.value;
		const normalized = normalizeSlug(oldValue);

		if (normalized !== oldValue) {
			$formData.slug = normalized;
		}
	};

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
			<span class="flex items-center text-sm text-slate-700 dark:text-slate-300 mb-2">
				<span class="font-medium">Body</span>
				{#if showPreview}
					<button type="button" class="link ml-auto" onclick={closePreview}>
						Back to editing
					</button>
				{:else}
					<button type="button" class="link ml-auto" onclick={loadPreview}>
						Preview
					</button>
				{/if}
			</span>
		</label>

		<textarea
			class="w-full resize-none overflow-hidden"
			class:hidden={showPreview}
			use:autoResize={$formData.body}
			name="body"
			rows="16"
			bind:value={$formData.body}
			onpaste={handlePaste}
			autofocus
		></textarea>

		{#if showPreview}
			<div class="w-full border border-slate-300 dark:border-slate-500 px-3 py-2">
				{#if previewLoading}
					<p>Rendering preview…</p>
				{:else if previewError}
					<p class="text-red-600">{previewError}</p>
				{:else}
					<article class="post-content">
						<!-- eslint-disable-next-line svelte/no-at-html-tags -->
						{@html previewHtml}
					</article>
				{/if}
			</div>
		{/if}

		{#if $errors.body}
			<p class="mt-1 text-sm text-red-600">{$errors.body[0]}</p>
		{/if}

		<!-- Uploader -->
		<div class="mt-4">
			<label class="block">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Upload</span>
				<input
					class="w-full px-2 py-1"
					type="file"
					accept="image/*"
					bind:this={fileInput}
				/>
			</label>

			<button
				type="button"
				class="mt-2"
				onclick={() => upload()}
				disabled={uploadLoading}
			>
				{#if uploadLoading}
					Uploading...
				{:else}
					Upload file
				{/if}
			</button>
		</div>
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
				<input class="w-full h-9" name="slug" bind:value={$formData.slug} oninput={handleSlugInput} />
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
