<script lang="ts">
	import type { Post } from '$lib/types';
	import { resolve } from '$app/paths';

	let {
		post,
		canEdit = false
	}: {
		post: Post;
		canEdit?: boolean;
	} = $props();
</script>

<article class="mt-4">
	{#if post.title && !post.hideTitle}
		<div>
			<h3 class="inline font-semibold">
				{post.title}
			</h3>
		</div>
	{/if}

	<div class="mt-4 post-content">
		<!-- eslint-disable-next-line svelte/no-at-html-tags -->
		{@html post.body}
	</div>

	<div class="mt-4 flex gap-2 flex-wrap text-sm">
		<a
			href={resolve('/[number=integer]/[[slug]]', {
				number: post.number.toString(),
				slug: post.slug ?? undefined
			})}
		>
			#{post.number}
		</a>
		/
		<div class="text-slate-500">
			{new Date(post.publishedAt ?? post.createdAt).toLocaleTimeString('it-IT', {
				hour: '2-digit',
				minute: '2-digit'
			})}
		</div>

		{#if post.tags.length}
			<span class="text-slate-500">/</span>

			{#each post.tags as tag(tag)}
				<a href={resolve('/archivio/[tag]', { tag })}>
					#{tag}
				</a>
			{/each}
		{/if}

		{#if canEdit}
			<span class="text-slate-500">/</span>

			<a
				href={resolve('/[number=integer]/edit', { number: post.number.toString() })}
				class="text-slate-500 hover:underline"
			>
				modifica
			</a>
		{/if}
	</div>
</article>
