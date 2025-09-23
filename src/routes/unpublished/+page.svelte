<script lang="ts">
	import Sidebar from '$lib/Sidebar.svelte';
	import { resolve } from '$app/paths';

	let { data } = $props();

	const formatDateTime = (value: Date | string) =>
		new Date(value).toLocaleString('it-IT', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		<h2 class="text-3xl font-semibold mb-8">Unpublished</h2>

		{#each data.posts as post(post.id)}
			<article class="mt-4">
				<header class="space-y-1">
					<div class="text-sm text-slate-500">
						#{post.number}
						/
						{post.title}
						{#if post.hideTitle} (hidden){/if}
					</div>

					<div class="text-sm text-slate-500">
						Created at: {formatDateTime(post.createdAt)}
					</div>

					<div class="text-sm text-slate-500">
						Updated at: {post.updatedAt ? formatDateTime(post.updatedAt) : ''}
					</div>
				</header>

				<div class="post-content mt-4">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -->
					{@html post.body}
				</div>

				<div class="mt-4">
					{#each post.tags as tag(tag)}
						<a href={resolve('/archivio/[tag]', { tag })}>
							#{tag}
						</a>
					{/each}
				</div>
			</article>

			<hr class="my-6" />
		{/each}
	</div>

	<Sidebar archive={data.archive} tags={data.tags} />
</div>
