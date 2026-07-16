<script lang="ts">
	import { resolve } from '$app/paths';
	import Category from './Category.svelte';
	import { page } from '$app/state';

	let { categories } = $props();

	const searchQuery = $derived(page.url.searchParams.get('q') ?? '');
</script>

<div class="container mt-6 max-w-6xl">
	<h1 class="text-4xl font-bold">
		<a class="hover:underline" href={resolve('/')}> Note di Matteo </a>
	</h1>

	<div class="mt-4 grid items-center gap-x-20 gap-y-4 lg:grid-cols-12">
		<div class="flex flex-wrap gap-3 lg:col-span-7">
			{#each categories as category (category.slug)}
				<Category slug={category.slug} label={category.label} count={category.count} />
			{/each}
		</div>

		<form
			class="lg:col-span-5"
			method="GET"
			action={resolve('/cerca')}
			role="search"
			data-sveltekit-keepfocus
		>
			<input
				type="search"
				name="q"
				value={searchQuery}
				placeholder="Cerca..."
				aria-label="Cerca nel sito"
				class="h-9 w-full"
			/>
		</form>
	</div>
</div>
