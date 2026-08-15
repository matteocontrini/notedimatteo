<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { autoResize } from '$lib/actions/auto-resize';
	import type { PostForEdit } from '$lib/types';
	import { tick } from 'svelte';
	import type { SuperForm } from 'sveltekit-superforms';
	import {
		insertMarkdownLink,
		toggleMarkdownBold,
		toggleMarkdownItalic,
		toggleMarkdownQuote,
		type TextEdit,
	} from '../editor/markdown';
	import { getImageMarkup, insertImageMarkup, uploadImage } from '../editor/uploads';
	import type { PostFormValues } from '../schema';

	let { post, postForm }: { post: PostForEdit; postForm: SuperForm<PostFormValues> } = $props();

	const form = $derived(postForm.form);
	const errors = $derived(postForm.errors);
	let bodyTextarea: HTMLTextAreaElement;
	let fileInput: HTMLInputElement;
	let showPreview = $state(false);
	let previewLoading = $state(false);
	let previewHtml = $state('');
	let previewError = $state<string | null>(null);
	let uploadLoading = $state(false);

	const restoreSelection = async (
		textarea: HTMLTextAreaElement,
		selectionStart: number,
		selectionEnd = selectionStart,
	) => {
		await tick();
		textarea.setSelectionRange(selectionStart, selectionEnd);
		textarea.focus();
	};

	const applyTextEdit = (
		textarea: HTMLTextAreaElement,
		transform: (value: string, selectionStart: number, selectionEnd: number) => TextEdit,
	) => {
		const edit = transform(textarea.value, textarea.selectionStart, textarea.selectionEnd);
		$form.body = edit.value;
		void restoreSelection(textarea, edit.selectionStart, edit.selectionEnd);
	};

	const loadPreview = async () => {
		previewLoading = true;
		previewError = null;
		showPreview = true;

		const previewEndpoint = resolve('/[number=integer]/edit/preview', {
			number: post.number.toString(),
		});

		try {
			const response = await fetch(previewEndpoint, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ body: $form.body }),
			});

			if (!response.ok) throw new Error('Unable to render preview');

			const result = (await response.json()) as { html: string };
			previewHtml = result.html;
		} catch (error) {
			previewHtml = '';
			previewError =
				error instanceof Error ? error.message : 'Unexpected error while rendering preview';
		} finally {
			previewLoading = false;
		}
	};

	const closePreview = () => {
		showPreview = false;
		previewError = null;
		// Focus the textarea when returning to edit mode
		void tick().then(() => bodyTextarea?.focus());
	};

	const uploadFile = async (
		file: File,
		cursorPosition?: number,
		textarea?: HTMLTextAreaElement,
	) => {
		uploadLoading = true;

		try {
			const image = await uploadImage(`${page.url.pathname}/upload`, file);
			const insertion = insertImageMarkup($form.body, getImageMarkup(image), cursorPosition);
			$form.body = insertion.body;

			if (textarea && insertion.cursorPosition !== undefined) {
				// Restore cursor position after state update
				void restoreSelection(textarea, insertion.cursorPosition);
			}

			return true;
		} catch (error) {
			console.error('Upload error:', error);
			alert('Upload failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
			return false;
		} finally {
			uploadLoading = false;
		}
	};

	const uploadSelectedFile = async () => {
		const file = fileInput.files?.[0];

		if (!file) {
			alert('Please select a file to upload');
			return;
		}

		if (await uploadFile(file)) fileInput.value = '';
	};

	const handlePaste = async (event: ClipboardEvent) => {
		// Look for image files in the clipboard
		const imageItem = Array.from(event.clipboardData?.items ?? []).find((item) =>
			item.type.startsWith('image/'),
		);
		const file = imageItem?.getAsFile();

		if (!file) return;

		event.preventDefault();
		const textarea = event.currentTarget as HTMLTextAreaElement;
		// Capture cursor position before upload
		await uploadFile(file, textarea.selectionStart, textarea);
	};

	const handleKeydown = (event: KeyboardEvent) => {
		const hasCommandModifier = event.metaKey || event.ctrlKey;
		if (!hasCommandModifier) return;

		if (event.key.toLowerCase() === 'p') {
			// Cmd+P: Toggle preview
			event.preventDefault();
			if (showPreview) closePreview();
			else void loadPreview();
			return;
		}

		const textarea = event.target as HTMLTextAreaElement;
		if (textarea.tagName !== 'TEXTAREA') return;

		if (event.key.toLowerCase() === 'k') {
			// Cmd+K: Insert markdown link
			event.preventDefault();
			applyTextEdit(textarea, insertMarkdownLink);
		} else if (event.key.toLowerCase() === 'b') {
			// Cmd+B: Toggle bold
			event.preventDefault();
			applyTextEdit(textarea, toggleMarkdownBold);
		} else if (event.key.toLowerCase() === 'i') {
			// Cmd+I: Toggle italic
			event.preventDefault();
			applyTextEdit(textarea, toggleMarkdownItalic);
		} else if (event.shiftKey && event.key === "'") {
			// Cmd+Shift+': Toggle quote
			event.preventDefault();
			applyTextEdit(textarea, toggleMarkdownQuote);
		}
	};
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="md:col-span-7">
	<label class="block">
		<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">Title</span>
		<input class="h-9 w-full" name="title" bind:value={$form.title} />
	</label>

	{#if $errors.title}
		<p class="mt-1 text-sm text-red-600">{$errors.title[0]}</p>
	{/if}

	<label class="mt-3 flex items-center gap-3">
		<input
			class="accent-slate-800 dark:accent-slate-200"
			type="checkbox"
			name="hideTitle"
			bind:checked={$form.hideTitle}
		/>
		<span class="text-sm text-slate-700 dark:text-slate-300">Hide title</span>
	</label>

	<label class="mt-6 block">
		<span class="mb-2 flex items-center text-sm text-slate-700 dark:text-slate-300">
			<span class="font-medium">Body</span>
			{#if showPreview}
				<button type="button" class="link ml-auto" onclick={closePreview}>Back to editing</button>
			{:else}
				<button type="button" class="link ml-auto" onclick={loadPreview}>Preview</button>
			{/if}
		</span>
	</label>

	<textarea
		class="w-full resize-none overflow-hidden"
		class:hidden={showPreview}
		use:autoResize={$form.body}
		name="body"
		rows="16"
		bind:value={$form.body}
		bind:this={bodyTextarea}
		onpaste={handlePaste}
		autofocus></textarea>

	{#if showPreview}
		<div class="w-full border border-slate-300 px-3 py-2 dark:border-slate-500">
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
			<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">Upload</span>
			<input class="w-full px-2 py-1" type="file" accept="image/*" bind:this={fileInput} />
		</label>

		<button type="button" class="mt-2" onclick={uploadSelectedFile} disabled={uploadLoading}>
			{uploadLoading ? 'Uploading...' : 'Upload file'}
		</button>
	</div>
</div>
