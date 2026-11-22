<script lang="ts">
	import { untrack } from 'svelte';
	import {
		formFromPost,
		sameForm,
		savePost,
		type EditorData,
		type FieldErrors,
		type Intent,
	} from './form';
	import PostContentEditor from './PostContentEditor.svelte';
	import PostDates from './PostDates.svelte';
	import PostMetadata from './PostMetadata.svelte';
	import PublishActions from './PublishActions.svelte';
	import SaveStatus from './SaveStatus.svelte';
	import ShortcutHelp from './ShortcutHelp.svelte';

	let { data }: { data: EditorData } = $props();

	// main.ts mounts the editor once, so data never changes
	const initial = untrack(() => data);
	// post holds the last saved version, with its dates and URLs
	let post = $state(initial.post);
	let form = $state(formFromPost(initial.post));
	// saved holds the fields as the last successful save sent them
	let saved = $state(formFromPost(initial.post));
	let errors = $state<FieldErrors>({});
	const dirty = $derived(!sameForm(form, saved));

	let saving = false;
	// The status bar says "Saving..." only for saves slower than 200 ms
	let slow = $state(false);
	let showSaved = $state(false);
	let savedTimer: ReturnType<typeof setTimeout> | undefined;

	const save = async (intent: Intent) => {
		if (saving) return;
		saving = true;
		const slowTimer = setTimeout(() => (slow = true), 200);
		const sent = $state.snapshot(form);

		try {
			const result = await savePost(post.urls.edit, sent, intent);
			if (result.errors) {
				errors = result.errors;
				return;
			}

			post = result.post;
			saved = sent;
			errors = {};
			// A new number moves the post, and its editor, to new URLs
			if (location.pathname !== post.urls.edit) {
				history.replaceState(history.state, '', post.urls.edit);
			}

			showSaved = true;
			clearTimeout(savedTimer);
			savedTimer = setTimeout(() => (showSaved = false), 1000);
		} catch (error) {
			console.error(error);
			alert(error instanceof Error ? `Error: ${error.message}` : 'Unknown error');
		} finally {
			clearTimeout(slowTimer);
			slow = false;
			saving = false;
		}
	};

	const publishIntent = $derived<Intent>(post.publishedAt === null ? 'publish' : 'unpublish');

	const handleSubmit = (event: SubmitEvent) => {
		event.preventDefault();
		// Enter in a field submits the form as if the user clicked Save, the first button
		const button = event.submitter as HTMLButtonElement | null;
		void save((button?.value as Intent | undefined) ?? 'save');
	};

	const handleKeydown = (event: KeyboardEvent) => {
		if (!(event.metaKey || event.ctrlKey) || event.key.toLowerCase() !== 's') return;

		event.preventDefault();
		// Cmd+Shift+S does what the second button does: publish or unpublish
		void save(event.shiftKey ? publishIntent : 'save');
	};
</script>

<svelte:window onkeydown={handleKeydown} />

<SaveStatus saving={slow} saved={showSaved} />

<form class="grid gap-12 md:grid-cols-12" onsubmit={handleSubmit}>
	<PostContentEditor bind:form {errors} urls={post.urls} imageBaseURL={initial.imageBaseURL} />

	<aside class="space-y-6 md:col-span-5">
		<PostMetadata
			{post}
			bind:form
			{errors}
			allTags={initial.tags}
			categories={initial.categories}
		/>
		<PostDates {post} />
		<PublishActions published={post.publishedAt !== null} saveDisabled={!dirty} />
		<ShortcutHelp />
	</aside>
</form>
