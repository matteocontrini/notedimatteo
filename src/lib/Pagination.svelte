<script lang="ts">
	import { page } from '$app/state';

	interface PaginationData {
		currentPage: number;
		totalPages: number;
		hasNextPage: boolean;
		hasPreviousPage: boolean;
		totalPosts: number;
	}

	let { pagination }: { pagination: PaginationData } = $props();

	const previousPageLink = $derived(
		pagination.currentPage - 1 > 1
			? `${page.url.pathname}?page=${pagination.currentPage - 1}`
			: page.url.pathname
	);

	const nextPageLink = $derived(`${page.url.pathname}?page=${pagination.currentPage + 1}`);
</script>

{#if pagination && pagination.totalPages > 1}
	<div class="flex justify-center items-center gap-4 mt-12 mb-8">
		{#if pagination.hasPreviousPage}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="link" href={previousPageLink}>
				← Precedente
			</a>
		{/if}

		<span class="py-2 text-gray-600">
			Pagina {pagination.currentPage} di {pagination.totalPages}
		</span>

		{#if pagination.hasNextPage}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="link" href={nextPageLink}>
				Successiva →
			</a>
		{/if}
	</div>
{/if}
