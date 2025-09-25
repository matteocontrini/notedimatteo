<script lang="ts">
	import Post from '$lib/Post.svelte';
	import Sidebar from '$lib/Sidebar.svelte';

	let { data } = $props();
	const hasQuery = $derived(data.query.length > 0);
	const hasResults = $derived(data.posts.length > 0);
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		{#if !hasResults || !hasQuery}
			<h2 class="text-2xl font-semibold mb-2">Nessun risultato per "{data.query}"</h2>
		{:else}
			<h2 class="text-2xl font-semibold mb-2">
				{data.posts.length === 1 ? '1 risultato' : `${data.posts.length} risultati`}
				per "{data.query}"
			</h2>

			<hr class="my-6" />

			{#each data.posts as post (post.id)}
				<Post {post} />
				<hr class="my-6" />
			{/each}
		{/if}
	</div>

	<Sidebar archive={data.archive} tags={data.tags} />
</div>
