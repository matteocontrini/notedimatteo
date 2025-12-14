<script lang="ts">
	import '@fontsource-variable/inter';
	import '../app.css';
	import favicon from '$lib/assets/favicon.png';
	import { ModeWatcher } from 'mode-watcher';
	import { resolve } from '$app/paths';
	import MetaTags from './MetaTags.svelte';
	import TopBar from './TopBar.svelte';
	import Header from './Header.svelte';
	import Footer from './Footer.svelte';

	let { children, data } = $props();
	const isLoggedIn = $derived(data.isLoggedIn);
	const unpublishedCount = $derived(data.unpublishedCount ?? 0);
</script>

<svelte:head>
	<MetaTags />
	<link rel="icon" href={favicon} />
	<link rel="alternate" type="application/rss+xml" title="Note di Matteo" href={resolve('/feed.xml')} />
</svelte:head>

<ModeWatcher />

<TopBar {isLoggedIn} {unpublishedCount} />

<Header categories={data.categories} />

<hr class="mt-6" />

<div class="container max-w-6xl mt-10">
	{@render children?.()}
</div>

<hr class="mt-16">

<Footer />
