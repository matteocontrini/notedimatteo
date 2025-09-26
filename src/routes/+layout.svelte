<script lang="ts">
	import '@fontsource-variable/inter';
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { ModeWatcher, userPrefersMode } from 'mode-watcher';
	import Category from './Category.svelte';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	let { children, data } = $props();
	const searchQuery = $derived(page.url.searchParams.get('q') ?? '');
	const isLoggedIn = $derived(data.isLoggedIn);
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<link rel="alternate" type="application/rss+xml" title="Note di Matteo" href={resolve('/feed.xml')} />
	<title>Note di Matteo</title>
</svelte:head>

<ModeWatcher />

<div class="bg-[#0067B9] h-[18px]">
	<div class="container max-w-6xl h-full flex items-center justify-end gap-4">
		<a
			class="text-white/80 text-xs uppercase hover:text-white"
			href="https://matteosonoio.it"
		>
			blog
			<span class="font-[system-ui]">↗︎</span>
		</a>

		<a
			class="text-white/80 text-xs uppercase hover:text-white"
			href={resolve('/about')}
		>
			about
		</a>

		{#if isLoggedIn}
			<a
				class="text-white/80 text-xs uppercase hover:text-white"
				href={resolve('/unpublished')}
			>
				unpublished
			</a>

			<a
				class="text-white/80 text-xs uppercase hover:text-white"
				href={resolve('/unpublished/new')}
			>
				new
			</a>
		{:else}
			<a
				class="text-white/80 text-xs uppercase hover:text-white"
				href={resolve('/login')}
			>
				login
			</a>
		{/if}
	</div>
</div>

<div class="container max-w-6xl mt-6">
	<h1 class="text-4xl font-bold">
		<a class="hover:underline" href={resolve('/')}>
			Note di Matteo
		</a>
	</h1>

	<div class="mt-4 grid gap-x-20 gap-y-4 lg:grid-cols-12 items-center">
		<div class="lg:col-span-7 flex gap-3 flex-wrap">
			{#each data.categories as category (category.slug)}
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

<hr class="mt-6 border-slate-300 dark:border-slate-700" />

<div class="container max-w-6xl mt-10">
	{@render children?.()}
</div>

<hr class="mt-16">

<footer class="container max-w-6xl py-12 flex flex-col sm:flex-row gap-4">
	<select class="pl-3 pr-10 py-0 h-9 sm:ml-auto"
					bind:value={userPrefersMode.current}>
		<option value="system">Automatico</option>
		<option value="light">Chiaro</option>
		<option value="dark">Scuro</option>
	</select>
</footer>
