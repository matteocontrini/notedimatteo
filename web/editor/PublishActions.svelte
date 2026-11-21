<script lang="ts">
	let { published, saveDisabled }: { published: boolean; saveDisabled: boolean } = $props();
</script>

<section class="space-y-4">
	<button type="submit" class="w-full" name="intent" value="save" disabled={saveDisabled}>
		Save
	</button>

	{#if !published}
		<button type="submit" class="w-full" name="intent" value="publish"> Publish </button>
	{:else}
		<button type="submit" class="w-full" name="intent" value="unpublish"> Unpublish </button>
	{/if}
</section>
