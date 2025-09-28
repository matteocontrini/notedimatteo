<script lang="ts">
	import Post from '$lib/Post.svelte';
	import Sidebar from '$lib/Sidebar.svelte';
	import { page } from '$app/state';

	let { data } = $props();
	const canEdit = $derived(data.isLoggedIn);
	const pagination = $derived(data.pagination);
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		<h2 class="text-2xl font-semibold mb-8">
			{data.category.label}
		</h2>

		{#if data.posts.length === 0}
			<p class="mt-4 text-slate-500">
				Nessun post pubblicato in questa categoria.
			</p>
		{:else}
			{#each data.posts as post (post.id)}
				<Post {post} canEdit={canEdit} />
				<hr class="my-6" />
			{/each}
		{/if}

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
