<script lang="ts">
	import type { posts } from '$lib/server/db/schema';

	interface Props {
		post: typeof posts.$inferSelect;
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

		<a href="/2"> #cloud </a>

		/

		<span class="text-slate-500"> 16:41 </span>
	</div>
</article>
