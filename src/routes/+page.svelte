<script lang="ts">
	import Post from '../lib/Post.svelte';

	const monthFormatter = new Intl.DateTimeFormat('it-IT', { month: 'long' });
	const formatMonthName = (year: number, month: number) =>
		monthFormatter.format(new Date(Date.UTC(year, month - 1, 1)));

	let { data } = $props();
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		<h2 class="font-serif text-2xl font-medium">18 settembre 2025</h2>

		{#each data.posts as post(post.id)}
			<Post {post} />
			<hr class="my-6" />
		{/each}
	</div>

	<div class="md:col-span-5">
		<h2 class="font-serif text-2xl font-medium">Archivio</h2>

		<ul class="mt-4">
			{#each data.archive as month, index (month.year + '-' + month.month)}
				{#if index === 0 || data.archive[index - 1].year !== month.year}
					<li class="mt-2 font-semibold">{month.year}</li>
				{/if}
				<li class="mt-1">
					<a href="/archivio/{month.year}/{month.month}" class="link">
						{formatMonthName(month.year, month.month)}
					</a>
					({month.postsCount})
				</li>
			{/each}
		</ul>

		<h2 class="mt-8 font-serif text-2xl font-medium">Tag</h2>

		<ul class="mt-4 space-y-1">
			{#each data.tags as tag(tag.name)}
				<li>
					<a href="/archivio/{tag.name}" class="link">#{tag.name}</a>
					({tag.count})
				</li>
			{/each}
		</ul>
	</div>
</div>
