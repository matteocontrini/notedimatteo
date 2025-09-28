<script lang="ts">
	import Post from '$lib/Post.svelte';
	import Sidebar from '$lib/Sidebar.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	const canEdit = $derived(data.isLoggedIn);
	const postGroups = $derived(data.postGroups ?? []);
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		{#each postGroups as group, index (group.dateKey)}
			<h2 class="text-2xl font-semibold mb-8"
					class:mt-10={index !== 0}>
				{group.heading}
			</h2>

			{#each group.posts as post (post.id)}
				<Post {post} canEdit={canEdit} showDate={false} />
				<hr class="my-6" />
			{/each}
		{/each}
	</div>

	<Sidebar archive={data.archive} tags={data.tags} />
</div>
