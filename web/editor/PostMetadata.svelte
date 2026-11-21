<script lang="ts">
	import Fuse from 'fuse.js';
	import { normalizeSlug } from './slug';
	import { completeCurrentTag, getCurrentTag } from './tags';
	import type { Category, EditorPost, FieldErrors, PostForm } from './form';

	let {
		post,
		form = $bindable(),
		errors,
		allTags,
		categories,
	}: {
		post: EditorPost;
		form: PostForm;
		errors: FieldErrors;
		allTags: string[];
		categories: Category[];
	} = $props();

	// Initialize Fuse for fuzzy matching
	const tagsFuse = $derived(new Fuse(allTags, { threshold: 0.3 }));
	// Tags autocomplete state
	let tagSuggestions = $state<string[]>([]);

	// Update tag suggestions based on cursor position
	const updateTagSuggestions = (input: HTMLInputElement) => {
		// Get the current tag being typed at cursor position
		const currentTag = getCurrentTag(input.value, input.selectionStart ?? 0);
		// Fuzzy search and show top 10 matches
		tagSuggestions = currentTag
			? tagsFuse
					.search(currentTag)
					.map((result) => result.item)
					.slice(0, 10)
			: [];
	};

	// Handle tag input and cursor movement
	const handleTagsInput = (event: Event) => {
		updateTagSuggestions(event.currentTarget as HTMLInputElement);
	};

	// Handle Tab key for autocomplete
	const handleTagsKeydown = (event: KeyboardEvent) => {
		// Check if Tab was pressed and we have suggestions
		if (event.key !== 'Tab' || tagSuggestions.length === 0) return;

		event.preventDefault();
		const input = event.currentTarget as HTMLInputElement;
		const completion = completeCurrentTag(
			input.value,
			input.selectionStart ?? 0,
			tagSuggestions[0],
		);

		form.tags = completion.value;
		// Position cursor after the completed tag
		setTimeout(() => {
			input.setSelectionRange(completion.cursorPosition, completion.cursorPosition);
			updateTagSuggestions(input);
		}, 0);
	};

	const handleSlugInput = (event: Event) => {
		const input = event.currentTarget as HTMLInputElement;
		const normalized = normalizeSlug(input.value);

		if (normalized !== input.value) {
			form.slug = normalized;
		}
	};
</script>

<section class="grid gap-4 md:grid-cols-2">
	<label class="block">
		<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">ID</span>
		<input class="h-9 w-full" name="id" value={post.id.toString()} readonly />
	</label>

	<label class="block md:col-start-2">
		<span class="mb-2 flex text-sm text-slate-700 dark:text-slate-300">
			<span class="font-medium">Number</span>
			<a href={post.urls.post} class="link ml-auto">open post</a>
		</span>
		<input class="h-9 w-full" type="number" name="number" bind:value={form.number} />

		{#if errors.number}
			<p class="mt-1 text-sm text-red-600">{errors.number}</p>
		{/if}
	</label>

	<label class="block md:col-span-2">
		<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">Category</span>
		<select class="h-9 w-full py-0" name="category" bind:value={form.category}>
			{#each categories as option (option.slug)}
				<option value={option.slug}>{option.label}</option>
			{/each}
		</select>

		{#if errors.category}
			<p class="mt-1 text-sm text-red-600">{errors.category}</p>
		{/if}
	</label>

	<label class="block md:col-span-2">
		<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">Slug</span>
		<input class="h-9 w-full" name="slug" bind:value={form.slug} oninput={handleSlugInput} />
	</label>

	<label class="block md:col-span-2">
		<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">
			Tags (comma separated)
		</span>
		<input
			class="h-9 w-full"
			name="tags"
			bind:value={form.tags}
			oninput={handleTagsInput}
			onclick={handleTagsInput}
			onkeyup={handleTagsInput}
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
