<script lang="ts">
	import type { Post } from '$lib/types';

	let { post }: { post: Post } = $props();
	let postHref = $derived(post.slug ? `/${post.number}/${post.slug}` : `/${post.number}`);
</script>

<article class="mt-4">
	{#if post.title && !post.hideTitle}
		<h3 class="text-lg font-semibold">
			{post.title}
		</h3>
	{/if}

	<div class="mt-4 post-content">
		<!-- eslint-disable-next-line svelte/no-at-html-tags -->
		{@html post.body}
	</div>

	<div class="mt-4 flex gap-2 flex-wrap">
		<a href={postHref}>
			#{post.number}
		</a>

		/

		{#each post.tags as tag(tag)}
			<a href="/archivio/{tag}">
				#{tag}
			</a>
		{/each}

		/

		<span class="text-slate-500">
			{new Date(post.publishedAt ?? post.createdAt).toLocaleTimeString('it-IT', {
				hour: '2-digit',
				minute: '2-digit'
			})}
		</span>
	</div>
</article>
