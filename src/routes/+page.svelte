<script lang="ts">
	import Post from '$lib/Post.svelte';
	import Sidebar from '$lib/Sidebar.svelte';

	let { data } = $props();
	const canEdit = $derived(data.isLoggedIn);
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		<h2 class="text-2xl font-semibold mb-8">18 settembre 2025</h2>

		{#each data.posts as post(post.id)}
			<Post {post} canEdit={canEdit} showDate={false} />
			<hr class="my-6" />
		{/each}
	</div>

	<Sidebar archive={data.archive} tags={data.tags} />
</div>
