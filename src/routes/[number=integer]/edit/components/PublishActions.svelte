<script lang="ts">
	import type { PostForEdit } from '$lib/types';

	let {
		post,
		saveDisabled,
		publishButton = $bindable(),
	}: {
		post: PostForEdit;
		saveDisabled: boolean;
		publishButton?: HTMLButtonElement;
	} = $props();
</script>

<section class="space-y-4">
	<button type="submit" class="w-full" name="intent" value="save" disabled={saveDisabled}>
		Save
	</button>

	{#if post.publishedAt === null}
		<button type="submit" class="w-full" name="intent" value="publish" bind:this={publishButton}>
			Publish
		</button>
	{:else}
		<button type="submit" class="w-full" name="intent" value="unpublish" bind:this={publishButton}>
			Unpublish
		</button>
	{/if}
</section>
