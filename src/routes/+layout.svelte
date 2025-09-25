<script lang="ts">
	import '@fontsource-variable/inter';
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { ModeWatcher, userPrefersMode } from 'mode-watcher';
	import Category from './Category.svelte';
	import { resolve } from '$app/paths';

	let { children, data } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Note di Matteo</title>
</svelte:head>

<ModeWatcher />

<div class="bg-[#0067B9] h-[18px]">
	<div class="container max-w-6xl h-full flex items-center justify-end gap-4">
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
	</div>
</div>

<div class="container max-w-6xl mt-6">
	<div class="flex justify-between items-center">
		<h1 class="text-4xl font-bold">
			<a class="hover:underline" href={resolve('/')}>
				Note di Matteo
			</a>
		</h1>

		<ul>
			<li>
				<a class="link" href={resolve('/about')}>
					Cos'è?
				</a>
			</li>
		</ul>
	</div>

	<div class="mt-4 grid gap-20 md:grid-cols-12 items-center">
		<div class="md:col-span-7 flex gap-3 flex-wrap">
			{#each data.categories as category (category.slug)}
				<Category slug={category.slug} label={category.label} count={category.count} />
			{/each}
		</div>

		<div class="md:col-span-5">
			<input type="text" placeholder="Cerca..." class="w-full h-9" />
		</div>
	</div>
</div>

<hr class="mt-6 border-slate-300 dark:border-slate-700" />

<div class="container max-w-6xl mt-10">
	{@render children?.()}
</div>

<hr class="mt-16">

<footer class="container max-w-6xl py-12 flex flex-col sm:flex-row gap-4">
	<p>
		<a href="https://matteosonoio.it" class="link">
			Matteo Contrini
		</a>
	</p>

	<select class="pl-3 pr-10 py-0 h-9 sm:ml-auto"
					bind:value={userPrefersMode.current}>
		<option value="system">Automatico</option>
		<option value="light">Chiaro</option>
		<option value="dark">Scuro</option>
	</select>
</footer>
