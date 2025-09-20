<script lang="ts">
	import { formatMonthName, formatMonthParam } from '$lib/date';
	import type { CalendarItem } from '$lib/types';

	interface TagSummary {
		name: string;
		count: number;
	}

	let {
		archive,
		tags
	}: {
		archive: CalendarItem[];
		tags: TagSummary[];
	} = $props();
</script>

<div class="md:col-span-5">
	<h2 class="font-serif text-2xl font-medium">Archivio</h2>

	<ul class="mt-4">
		{#each archive as month, index (month.year + '-' + month.month)}
			{#if index === 0 || archive[index - 1].year !== month.year}
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
		{#each tags as tag(tag.name)}
			<li>
				<a href={`/archivio/${tag.name}`} class="link">#{tag.name}</a>
				({tag.count})
			</li>
		{/each}
	</ul>
</div>
