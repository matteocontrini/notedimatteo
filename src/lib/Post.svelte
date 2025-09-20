<script lang="ts">
	import type { Post } from '$lib/types';

	let { post }: { post: Post } = $props();
</script>

<article class="mt-4">
	{#if post.title && !post.hideTitle}
		<h3 class="text-lg font-semibold">
			{post.title}
		</h3>
	{/if}

	<p class="mt-4">
		{@html post.body}
	</p>

	<div class="mt-4 flex gap-2 flex-wrap">
		<a href="/{post.category}/{post.number}">
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
