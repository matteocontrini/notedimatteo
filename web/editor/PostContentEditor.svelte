<script lang="ts">
	import type { EditorView } from '@codemirror/view';
	import { onMount, tick } from 'svelte';
	import { createBodyEditor, finishUpload, startUpload } from './codemirror';
	import type { EditorURLs, FieldErrors, PostForm } from './form';
	import { getImageMarkup, uploadImage } from './uploads';

	let {
		form = $bindable(),
		errors,
		urls,
		imageBaseURL,
	}: { form: PostForm; errors: FieldErrors; urls: EditorURLs; imageBaseURL: string } = $props();
	let editorParent: HTMLDivElement;
	let view: EditorView | undefined;
	let fileInput: HTMLInputElement;
	let showPreview = $state(false);
	let previewLoading = $state(false);
	let previewHtml = $state('');
	let previewError = $state<string | null>(null);
	let uploads = $state(0);

	onMount(() => {
		view = createBodyEditor({
			parent: editorParent,
			doc: form.body,
			imageBaseURL,
			onChange: (body) => (form.body = body),
			onPasteImage: (file) => void uploadFile(file),
		});
		// The user opens the editor to write the body, so the body gets the focus
		view.focus();
		return () => view?.destroy();
	});

	const loadPreview = async () => {
		previewLoading = true;
		previewError = null;
		showPreview = true;

		try {
			const response = await fetch(urls.preview, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ body: form.body }),
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
		// The editor stays in the page while hidden, so it keeps its undo history
		void tick().then(() => view?.focus());
	};

	// uploadFile uploads an image and puts its markup where the cursor was when
	// the upload started.
	const uploadFile = async (file: File) => {
		if (!view) return false;
		const id = startUpload(view);
		uploads++;

		try {
			const image = await uploadImage(urls.upload, file);
			finishUpload(view, id, getImageMarkup(image));
			return true;
		} catch (error) {
			finishUpload(view, id);
			console.error('Upload error:', error);
			alert('Upload failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
			return false;
		} finally {
			uploads--;
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

	// The editor handles the formatting shortcuts; Cmd+P works anywhere on the page
	const handleKeydown = (event: KeyboardEvent) => {
		if (!(event.metaKey || event.ctrlKey) || event.key.toLowerCase() !== 'p') return;

		event.preventDefault();
		if (showPreview) closePreview();
		else void loadPreview();
	};
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="md:col-span-7">
	<label class="block">
		<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">Title</span>
		<input class="h-9 w-full" name="title" bind:value={form.title} />
	</label>

	<label class="mt-3 flex items-center gap-3">
		<input
			class="accent-slate-800 dark:accent-slate-200"
			type="checkbox"
			name="hideTitle"
			bind:checked={form.hideTitle}
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

	<div class:hidden={showPreview} bind:this={editorParent}></div>

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

	{#if errors.body}
		<p class="mt-1 text-sm text-red-600">{errors.body}</p>
	{/if}

	<!-- Uploader -->
	<div class="mt-4">
		<label class="block">
			<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">Upload</span>
			<input class="w-full px-2 py-1" type="file" accept="image/*" bind:this={fileInput} />
		</label>

		<button type="button" class="mt-2" onclick={uploadSelectedFile} disabled={uploads > 0}>
			{uploads > 0 ? 'Uploading...' : 'Upload file'}
		</button>
	</div>
</div>
