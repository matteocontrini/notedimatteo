<script lang="ts">
	import type { Post } from '$lib/types';
	import { resolve } from '$app/paths';

	let { post }: { post: Post } = $props();
</script>

<article class="mt-4">
	<div>
		<a href={resolve('/[number=integer]/[[slug]]', { number: post.number.toString(), slug: post.slug })}>
			#{post.number}
		</a>

		{#if post.title && !post.hideTitle}
			/
			<h3 class="inline font-semibold">
				{post.title}
			</h3>
		{/if}
	</div>


	<div class="mt-4 post-content">
		<!-- eslint-disable-next-line svelte/no-at-html-tags -->
		{@html post.body}
	</div>

	<div class="mt-4 flex gap-2 flex-wrap">
		<div class="text-slate-500">
			{new Date(post.publishedAt ?? post.createdAt).toLocaleTimeString('it-IT', {
				hour: '2-digit',
				minute: '2-digit'
			})}
		</div>

		<span class="text-slate-500">/</span>

		{#each post.tags as tag(tag)}
			<a href="/archivio/{tag}">
				#{tag}
			</a>
		{/each}
	</div>
</article>
