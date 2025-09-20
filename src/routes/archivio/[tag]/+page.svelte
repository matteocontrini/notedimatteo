<script lang="ts">
	import Post from '$lib/Post.svelte';
	import Sidebar from '$lib/Sidebar.svelte';

	let { data } = $props();
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		<h2 class="font-serif text-3xl font-medium mb-8">#{data.tag}</h2>

		{#if data.posts.length === 0}
			<p class="mt-4 text-slate-500">Nessun post con questo tag.</p>
		{:else}
			{#each data.posts as post (post.id)}
				<Post {post} />
				<hr class="my-6" />
			{/each}
		{/if}
	</div>

	<Sidebar archive={data.archive} tags={data.tags} />
</div>
