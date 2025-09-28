<script lang="ts">
	import Post from '$lib/Post.svelte';
	import Sidebar from '$lib/Sidebar.svelte';
	import { page } from '$app/state';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	const canEdit = $derived(data.isLoggedIn);
	const postGroups = $derived(data.postGroups ?? []);
	const pagination = $derived(data.pagination);
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
		{:else}
			Non c'è niente qui.
		{/each}

		<!-- Pagination -->
		{#if pagination && pagination.totalPages > 1}
			<div class="flex justify-center items-center gap-4 mt-12 mb-8">
				{#if pagination.hasPreviousPage}
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a class="link" href="{page.url.pathname}?page={pagination.currentPage - 1}">
						← Precedente
					</a>
				{/if}

				<span class="py-2 text-gray-600">
					Pagina {pagination.currentPage} di {pagination.totalPages}
				</span>

				{#if pagination.hasNextPage}
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a class="link" href="{page.url.pathname}?page={pagination.currentPage + 1}">
						Successiva →
					</a>
				{/if}
			</div>
		{/if}
	</div>

	<Sidebar archive={data.archive} tags={data.tags} />
</div>
