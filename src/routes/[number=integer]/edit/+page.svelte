<script lang="ts">
	import type { PostForEdit } from '$lib/types';
	import { superForm } from 'sveltekit-superforms/client';
	import type { SuperValidated } from 'sveltekit-superforms';
	import type { PostFormValues } from './schema';
	import { categorySlugs, getCategoryLabel } from '$lib/categories';
	import { autoResize } from '$lib/actions/auto-resize';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Kbd from '$lib/Kbd.svelte';
	import Fuse from 'fuse.js';

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
	let publishButton: HTMLButtonElement;
	let bodyTextarea: HTMLTextAreaElement;

	// Tags autocomplete state
	let tagSuggestions = $state<string[]>([]);

	// Get all available tags from page data (loaded in layout)
	const allTags = $derived(page.data.tags?.map((tag: { name: string; count: number }) => tag.name) ?? []);

	// Initialize Fuse for fuzzy matching
	const tagsFuse = $derived(
		new Fuse(allTags, {
			threshold: 0.3
		})
	);

	// Get the current tag being typed at cursor position
	const getCurrentTag = (value: string, cursorPos: number): string => {
		const beforeCursor = value.substring(0, cursorPos);
		const afterCursor = value.substring(cursorPos);

		// Find where current tag starts (after last comma, or start)
		const lastComma = beforeCursor.lastIndexOf(',');
		const tagStart = lastComma === -1 ? 0 : lastComma + 1;

		// Find where current tag ends (before next comma, or end)
		const nextComma = afterCursor.indexOf(',');
		const tagEnd = nextComma === -1 ? value.length : cursorPos + nextComma;

		return value.substring(tagStart, tagEnd).trim();
	};

	// Update tag suggestions based on cursor position
	const updateTagSuggestions = (input: HTMLInputElement) => {
		const cursorPos = input.selectionStart ?? 0;
		const currentTag = getCurrentTag(input.value, cursorPos);

		if (currentTag.length === 0) {
			tagSuggestions = [];
			return;
		}

		// Fuzzy search and show top 10 matches
		const results = tagsFuse.search(currentTag);
		tagSuggestions = results
			.map(result => result.item as string)
			.slice(0, 10);
	};

	// Handle tag input - update suggestions as user types
	const handleTagsInput = (event: Event) => {
		const input = event.target as HTMLInputElement;
		updateTagSuggestions(input);
	};

	// Handle cursor movement - update suggestions when cursor moves
	const handleTagsCursorMove = (event: Event) => {
		const input = event.target as HTMLInputElement;
		updateTagSuggestions(input);
	};

	// Handle Tab key for autocomplete
	const handleTagsKeydown = (event: KeyboardEvent) => {
		// Check if Tab was pressed and we have suggestions
		if (event.key === 'Tab' && tagSuggestions.length > 0) {
			event.preventDefault(); // Prevent default Tab behavior

			const input = event.target as HTMLInputElement;
			const cursorPos = input.selectionStart ?? 0;
			const value = input.value;

			// Get info about current tag
			const beforeCursor = value.substring(0, cursorPos);
			const afterCursor = value.substring(cursorPos);

			const lastComma = beforeCursor.lastIndexOf(',');
			const tagStart = lastComma === -1 ? 0 : lastComma + 1;

			const nextComma = afterCursor.indexOf(',');
			const tagEnd = nextComma === -1 ? value.length : cursorPos + nextComma;

			// Replace current tag with first suggestion
			const before = value.substring(0, tagStart);
			const after = value.substring(tagEnd);
			const firstSuggestion = tagSuggestions[0];

			// Build new value
			let newValue = before + firstSuggestion + after;

			// Update form data
			$formData.tags = newValue;

			// Position cursor after the completed tag
			const newCursorPos = tagStart + firstSuggestion.length;
			setTimeout(() => {
				input.setSelectionRange(newCursorPos, newCursorPos);
				updateTagSuggestions(input);
			}, 0);
		}
	};

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

	const toggleMarkdownBold = (textarea: HTMLTextAreaElement) => {
		const selectionStart = textarea.selectionStart;
		const selectionEnd = textarea.selectionEnd;
		const selectedText = textarea.value.substring(selectionStart, selectionEnd);

		let newBody: string;
		let newSelectionStart: number;
		let newSelectionEnd: number;

		const before = textarea.value.substring(0, selectionStart);
		const after = textarea.value.substring(selectionEnd);

		if (selectedText.length === 0) {
			// No selection - insert **|** with cursor in the middle
			newBody = before + '****' + after;
			newSelectionStart = selectionStart + 2;
			newSelectionEnd = selectionStart + 2;
		} else {
			// Wrap selected text in **
			newBody = before + '**' + selectedText + '**' + after;
			newSelectionStart = selectionStart + 2;
			newSelectionEnd = selectionEnd + 2;
		}

		// Update form data
		postForm.form.update(form => {
			return {
				...form,
				body: newBody
			};
		});

		// Restore selection after state update
		setTimeout(() => {
			textarea.setSelectionRange(newSelectionStart, newSelectionEnd);
			textarea.focus();
		}, 0);
	};

	const toggleMarkdownItalic = (textarea: HTMLTextAreaElement) => {
		const selectionStart = textarea.selectionStart;
		const selectionEnd = textarea.selectionEnd;
		const selectedText = textarea.value.substring(selectionStart, selectionEnd);

		let newBody: string;
		let newSelectionStart: number;
		let newSelectionEnd: number;

		const before = textarea.value.substring(0, selectionStart);
		const after = textarea.value.substring(selectionEnd);

		if (selectedText.length === 0) {
			// No selection - insert *|* with cursor in the middle
			newBody = before + '**' + after;
			newSelectionStart = selectionStart + 1;
			newSelectionEnd = selectionStart + 1;
		} else {
			// Wrap selected text in *
			newBody = before + '*' + selectedText + '*' + after;
			newSelectionStart = selectionStart + 1;
			newSelectionEnd = selectionEnd + 1;
		}

		// Update form data
		postForm.form.update(form => {
			return {
				...form,
				body: newBody
			};
		});

		// Restore selection after state update
		setTimeout(() => {
			textarea.setSelectionRange(newSelectionStart, newSelectionEnd);
			textarea.focus();
		}, 0);
	};

	const toggleMarkdownQuote = (textarea: HTMLTextAreaElement) => {
		const selectionStart = textarea.selectionStart;
		const selectionEnd = textarea.selectionEnd;
		const fullText = textarea.value;

		let newBody: string;
		let newSelectionStart: number;
		let newSelectionEnd: number;

		if (selectionStart === selectionEnd) {
			// No selection: insert "> " at cursor position
			const before = fullText.substring(0, selectionStart);
			const after = fullText.substring(selectionStart);

			newBody = before + '> ' + after;
			newSelectionStart = selectionStart + 2; // Position after "> "
			newSelectionEnd = selectionStart + 2;
		} else {
			// Has selection: quote each line in the selection
			// Find the line boundaries
			const lineStartPos = fullText.lastIndexOf('\n', selectionStart - 1) + 1;
			let lineEndPos = fullText.indexOf('\n', selectionEnd);
			if (lineEndPos === -1) lineEndPos = fullText.length;

			const before = fullText.substring(0, lineStartPos);
			const textToQuote = fullText.substring(lineStartPos, lineEndPos);
			const after = fullText.substring(lineEndPos);

			// Add "> " to each line, including blank lines to maintain quote block
			const lines = textToQuote.split('\n');
			const quotedText = lines
				.map(line => line.length > 0 ? '> ' + line : '>')
				.join('\n');

			newBody = before + quotedText + after;

			// Calculate new cursor positions
			const totalAdded = lines.reduce((sum, line) => {
				return sum + (line.length > 0 ? 2 : 1); // "> " = 2 chars, ">" = 1 char
			}, 0);
			const firstLineAdded = lines[0].length > 0 ? 2 : 1;

			newSelectionStart = selectionStart + firstLineAdded;
			newSelectionEnd = selectionEnd + totalAdded;
		}

		// Update form data
		postForm.form.update(form => {
			return {
				...form,
				body: newBody
			};
		});

		// Restore selection after state update
		setTimeout(() => {
			textarea.setSelectionRange(newSelectionStart, newSelectionEnd);
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
		// Focus the textarea when returning to edit mode
		setTimeout(() => {
			bodyTextarea?.focus();
		}, 0);
	};

	const handleKeydown = (event: KeyboardEvent) => {
		// Cmd+Shift+S: Save and publish
		if ((event.metaKey || event.ctrlKey) && event.shiftKey && (event.key === 's' || event.key === 'S')) {
			event.preventDefault();
			publishButton?.click();
			return;
		}

		// Cmd+S: Save
		if ((event.metaKey || event.ctrlKey) && (event.key === 's' || event.key === 'S')) {
			event.preventDefault();
			formElement?.requestSubmit();
		}

		// Cmd+P: Toggle preview
		if ((event.metaKey || event.ctrlKey) && (event.key === 'p' || event.key === 'P')) {
			event.preventDefault();
			if (showPreview) {
				closePreview();
			} else {
				loadPreview();
			}
		}

		// Cmd+K: Insert markdown link
		if ((event.metaKey || event.ctrlKey) && (event.key === 'k' || event.key === 'K')) {
			event.preventDefault();
			const textarea = event.target as HTMLTextAreaElement;
			if (textarea && textarea.tagName === 'TEXTAREA') {
				insertMarkdownLink(textarea);
			}
		}

		// Cmd+B: Toggle bold
		if ((event.metaKey || event.ctrlKey) && (event.key === 'b' || event.key === 'B')) {
			event.preventDefault();
			const textarea = event.target as HTMLTextAreaElement;
			if (textarea && textarea.tagName === 'TEXTAREA') {
				toggleMarkdownBold(textarea);
			}
		}

		// Cmd+I: Toggle italic
		if ((event.metaKey || event.ctrlKey) && (event.key === 'i' || event.key === 'I')) {
			event.preventDefault();
			const textarea = event.target as HTMLTextAreaElement;
			if (textarea && textarea.tagName === 'TEXTAREA') {
				toggleMarkdownItalic(textarea);
			}
		}

		// Cmd+Shift+': Toggle quote
		if ((event.metaKey || event.ctrlKey) && event.shiftKey && event.key === "'") {
			event.preventDefault();
			const textarea = event.target as HTMLTextAreaElement;
			if (textarea && textarea.tagName === 'TEXTAREA') {
				toggleMarkdownQuote(textarea);
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
		class="fixed bottom-0 left-0 flex h-9 w-full items-center"
		class:bg-white={!showSaved}
		class:dark:bg-slate-900={!showSaved}
		class:dark:text-white={!showSaved}
		class:border-t={!showSaved}
		class:border-slate-200={!showSaved}
		class:dark:border-slate-700={!showSaved}
		class:bg-green-500={showSaved}
		class:text-white={showSaved}
	>
		<div class="container max-w-6xl">
			{#if !showSaved}
				Saving...
			{:else}
				<span class="font-medium">Saved!</span>
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
			bind:this={bodyTextarea}
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
				<input
					class="w-full h-9"
					name="tags"
					bind:value={$formData.tags}
					oninput={handleTagsInput}
					onclick={handleTagsCursorMove}
					onkeyup={handleTagsCursorMove}
					onkeydown={handleTagsKeydown}
					autocomplete="off"
				/>
				{#if tagSuggestions.length > 0}
					<div class="mt-2 text-xs text-slate-600 dark:text-slate-400">
						{tagSuggestions.join(', ')}
					</div>
				{/if}
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
				<button type="submit" class="w-full" name="intent" value="publish" bind:this={publishButton}>
					Publish
				</button>
			{:else}
				<button type="submit" class="w-full" name="intent" value="unpublish" bind:this={publishButton}>
					Unpublish
				</button>
			{/if}
		</section>

		<section class="text-xs text-slate-600 dark:text-slate-400 space-y-1.5">
			<div class="font-medium text-slate-700 dark:text-slate-300 mb-2">Keyboard shortcuts</div>
			<div><Kbd>⌘B</Kbd> Bold</div>
			<div><Kbd>⌘I</Kbd> Italic</div>
			<div><Kbd>⌘⇧'</Kbd> Quote</div>
			<div><Kbd>⌘K</Kbd> Insert link</div>
			<div><Kbd>⌘P</Kbd> Toggle preview</div>
			<div><Kbd>⌘S</Kbd> Save</div>
			<div><Kbd>⌘⇧S</Kbd> Publish</div>
		</section>
	</aside>
</form>
