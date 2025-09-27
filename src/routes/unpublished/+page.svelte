<script lang="ts">
	import Sidebar from '$lib/Sidebar.svelte';
	import { resolve } from '$app/paths';
	import { format } from 'timeago.js';

	let { data, form } = $props();

	const formatDateTime = (value: Date | string) =>
		new Date(value).toLocaleString('it-IT', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});

	const confirmDeletion = (event: MouseEvent) => {
		const button = event.currentTarget as HTMLButtonElement | null;
		const postNumber = button?.dataset.postNumber;
		const message = `Confirm deletion of post #${postNumber}?`;

		if (!confirm(message)) {
			event.preventDefault();
		}
	};
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		<h2 class="text-2xl font-semibold mb-8">Unpublished</h2>

		{#if form?.error}
			<p class="text-red-600 font-medium">{form.error}</p>

			<hr class="my-6" />
		{/if}

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
						Created at: {formatDateTime(post.createdAt)} ({format(post.createdAt)})
					</div>

					<div class="text-sm text-slate-500">
						Updated at:
						{#if post.updatedAt}
							{formatDateTime(post.updatedAt)} ({format(post.updatedAt)})
						{:else}
							-
						{/if}
					</div>

					<div class="flex items-center gap-3">
						<a
							class="text-sm text-slate-500 hover:underline"
							href={resolve('/[number=integer]/edit', { number: post.number.toString() })}
						>
							modifica
						</a>
						<form method="post" action="?/delete" class="inline">
							<input type="hidden" name="postId" value={post.id} />
							<button
								type="submit"
								class="link text-red-600"
								data-post-number={post.number.toString()}
								onclick={confirmDeletion}
							>
								elimina
							</button>
						</form>
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
