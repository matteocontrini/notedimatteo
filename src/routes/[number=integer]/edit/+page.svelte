<script lang="ts">
	import { page } from '$app/state';
	import type { PostForEdit } from '$lib/types';
	import type { SuperValidated } from 'sveltekit-superforms';
	import { superForm } from 'sveltekit-superforms/client';
	import PostContentEditor from './components/PostContentEditor.svelte';
	import PostDates from './components/PostDates.svelte';
	import PostMetadata from './components/PostMetadata.svelte';
	import PublishActions from './components/PublishActions.svelte';
	import SaveStatus from './components/SaveStatus.svelte';
	import ShortcutHelp from './components/ShortcutHelp.svelte';
	import type { PostFormValues } from './schema';

	let { data } = $props<{ data: { post: PostForEdit; form: SuperValidated<PostFormValues> } }>();

	let showSaved = $state(false);
	let savedTimer: ReturnType<typeof setTimeout> | undefined;
	let formElement: HTMLFormElement;
	let publishButton = $state<HTMLButtonElement>();

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
			alert(result.error.message ? `Error: ${result.error.message}` : 'Unknown error');
		},
	});

	const { enhance, delayed, tainted, isTainted } = postForm;
	// Get all available tags from page data (loaded in layout)
	const allTags = $derived(
		page.data.tags?.map((tag: { name: string; count: number }) => tag.name) ?? [],
	);

	const handleKeydown = (event: KeyboardEvent) => {
		if (!(event.metaKey || event.ctrlKey) || event.key.toLowerCase() !== 's') return;

		event.preventDefault();
		if (event.shiftKey) {
			// Cmd+Shift+S: Save and publish
			publishButton?.click();
		} else {
			// Cmd+S: Save
			formElement?.requestSubmit();
		}
	};
</script>

<svelte:window onkeydown={handleKeydown} />

<SaveStatus saving={$delayed} saved={showSaved} />

<form
	class="grid gap-12 md:grid-cols-12"
	bind:this={formElement}
	method="POST"
	action="?/save"
	use:enhance
>
	<PostContentEditor post={data.post} {postForm} />

	<aside class="space-y-6 md:col-span-5">
		<PostMetadata post={data.post} {postForm} {allTags} />
		<PostDates post={data.post} />
		<PublishActions post={data.post} saveDisabled={!isTainted($tainted)} bind:publishButton />
		<ShortcutHelp />
	</aside>
</form>
