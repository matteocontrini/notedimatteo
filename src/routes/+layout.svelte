<script lang="ts">
	import '@fontsource-variable/open-sans';
	import '@fontsource-variable/noto-serif';
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { ModeWatcher, userPrefersMode } from 'mode-watcher';
	import Badge from './Badge.svelte';

	let { children, data } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Note di Matteo</title>
</svelte:head>

<ModeWatcher />

<div class="bg-[#0067B9] h-[18px]">
</div>

<div class="container max-w-6xl mt-6">
	<div class="flex justify-between items-center">
		<h1 class="text-4xl font-serif">
			<a href="/" class="hover:underline">
				Note di Matteo
			</a>
		</h1>

		<ul>
			<li>
				<a href="/about" class="link">
					Cos'è?
				</a>
			</li>
		</ul>
	</div>

	<div class="mt-4 flex flex-col sm:flex-row gap-4 justify-between items-center">
		<div class="flex gap-3 flex-wrap">
			{#each data.categories as category(category.name)}
				<Badge label={category.name} count={category.count} />
			{/each}
		</div>

		<input type="text" placeholder="Cerca..." class="w-full max-w-[400px] h-9" />
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

	<select class="px-2 py-1 sm:ml-auto pr-10"
					bind:value={userPrefersMode.current}>
		<option value="system">Automatico</option>
		<option value="light">Chiaro</option>
		<option value="dark">Scuro</option>
	</select>
</footer>
