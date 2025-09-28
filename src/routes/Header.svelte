<script lang="ts">
	import { resolve } from '$app/paths';
	import Category from './Category.svelte';
	import { page } from '$app/state';

	let { categories } = $props();

	const searchQuery = $derived(page.url.searchParams.get('q') ?? '');
</script>

<div class="container max-w-6xl mt-6">
	<h1 class="text-4xl font-bold">
		<a class="hover:underline" href={resolve('/')}>
			Note di Matteo
		</a>
	</h1>

	<div class="mt-4 grid gap-x-20 gap-y-4 lg:grid-cols-12 items-center">
		<div class="lg:col-span-7 flex gap-3 flex-wrap">
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
				class="w-full h-9"
			/>
		</form>
	</div>
</div>
