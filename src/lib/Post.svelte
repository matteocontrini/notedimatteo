<script lang="ts">
	import type { Post } from '$lib/types';

	interface Props {
		post: Post;
	}

	let { post }: Props = $props();
</script>

<article class="mt-4">
	{#if post.title}
		<h3 class="text-lg font-semibold">
			{post.title}
		</h3>
	{/if}

	<p class="mt-4">
		{@html post.body}
	</p>

	<div class="mt-4">
		<a href="/{post.category.toLowerCase().replace(/[^a-z]+/g, '')}/{post.number}">
			#{post.number}
		</a>

		/

		{#each post.tags as tag(tag)}
			<a href="/archivio/{tag}" class="mr-1">
				#{tag}
			</a>
		{/each}

		<a href="/2"> #cloud </a>

		/

		<span class="text-slate-500"> 16:41 </span>
	</div>
</article>
