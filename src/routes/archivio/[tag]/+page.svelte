<script lang="ts">
	import Post from '$lib/Post.svelte';
	import { formatMonthName, formatMonthParam } from '$lib/date';

	let { data } = $props();
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		<h1 class="font-serif text-3xl font-medium">Tag · #{data.tag}</h1>

		{#if data.posts.length === 0}
			<p class="mt-4 text-slate-500">Nessun post con questo tag.</p>
		{:else}
			{#each data.posts as post (post.id)}
				<Post {post} />
				<hr class="my-6" />
			{/each}
		{/if}
	</div>

	<div class="md:col-span-5">
		<h2 class="font-serif text-2xl font-medium">Archivio</h2>

		<ul class="mt-4">
			{#each data.archive as month, index (month.year + '-' + month.month)}
				{#if index === 0 || data.archive[index - 1].year !== month.year}
					<li class="mt-2 font-semibold">{month.year}</li>
				{/if}
				<li class="mt-1">
					<a href={`/archivio/${month.year}/${formatMonthParam(month.month)}`} class="link">
						{formatMonthName(month.year, month.month)}
					</a>
					({month.postsCount})
				</li>
			{/each}
		</ul>

		<h2 class="mt-8 font-serif text-2xl font-medium">Tag</h2>

		<ul class="mt-4 space-y-1">
			{#each data.tags as tagItem(tagItem.name)}
				<li>
					<a href="/archivio/{tagItem.name}" class="link">#{tagItem.name}</a>
					({tagItem.count})
				</li>
			{/each}
		</ul>
	</div>
</div>
