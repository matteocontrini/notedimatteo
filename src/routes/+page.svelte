<script lang="ts">
	import Post from '../lib/Post.svelte';

	let { data } = $props();
</script>

<div class="grid gap-20 md:grid-cols-12">
	<div class="md:col-span-7">
		<h2 class="font-serif text-2xl font-medium">18 settembre 2025</h2>

		{#each data.posts as post}
			<Post {post} />
			<hr class="my-6" />
		{/each}
	</div>

	<div class="md:col-span-5">
		<h2 class="font-serif text-2xl font-medium">Archivio</h2>

		<ul class="mt-4">
			{#each data.archive as group}
				<li class="mt-2 font-semibold">{group.year}</li>
				{#each group.months as month}
					<li class="mt-1">
						<a href="/archivio/{group.year}/{month.monthNumber}" class="link">
							{month.month}
						</a>
						({month.postCount})
					</li>
				{/each}
			{/each}
		</ul>

		<h2 class="mt-8 font-serif text-2xl font-medium">Tag</h2>

		<ul class="mt-4 space-y-1">
			{#each data.tags as tag}
				<li>
					<a href="/archivio/{tag.name}" class="link">#{tag.name}</a>
					({tag.count})
				</li>
			{/each}
		</ul>
	</div>
</div>
