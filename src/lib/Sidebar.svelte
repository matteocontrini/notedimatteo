<script lang="ts">
	import { formatMonthName, formatMonthParam } from '$lib/date';
	import type { CalendarItem } from '$lib/types';
	import { resolve } from '$app/paths';

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

	let alphabeticalSort = $state(false);

	const sortedTags = $derived(
		alphabeticalSort
			? [...tags].sort((a, b) => a.name.localeCompare(b.name))
			: tags
	);
</script>

<div class="md:col-span-5">
	<h2 class="text-2xl font-semibold">Archivio</h2>

	<ul class="mt-4">
		{#each archive as month, index (month.year + '-' + month.month)}
			{#if index === 0 || archive[index - 1].year !== month.year}
				<li class="mt-2 font-semibold">{month.year}</li>
			{/if}
			<li class="mt-1">
				<a class="link" href={resolve('/archivio/[year]/[month]', {
					year: month.year.toString(),
					month: formatMonthParam(month.month)
				})}>
					{formatMonthName(month.year, month.month)}
				</a>
				({month.postsCount})
			</li>
		{/each}
	</ul>

	<div class="mt-8 flex items-center gap-4">
		<h2 class="text-2xl font-medium">Tag</h2>

		<button
			onclick={() => alphabeticalSort = !alphabeticalSort}
			class="transition-colors cursor-pointer border-0 bg-transparent p-0 mt-1.5 h-fit"
			class:text-gray-400={!alphabeticalSort}
			class:text-sky-700={alphabeticalSort}
			class:dark:text-gray-500={!alphabeticalSort}
			class:dark:text-sky-500={alphabeticalSort}
			aria-label="Ordina alfabeticamente"
			title="Ordina alfabeticamente"
		>
			<svg width="32" height="20" viewBox="0 0 32 20" fill="none" xmlns="http://www.w3.org/2000/svg">
				<text x="0" y="13" font-family="system-ui, sans-serif" font-size="11" font-weight="600" fill="currentColor">
					A-Z
				</text>
				<path d="M28 6v8m0 0l-2-2m2 2l2-2" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"
							stroke-linejoin="round" />
			</svg>
		</button>
	</div>

	<ul class="mt-4 space-y-1">
		{#each sortedTags as tag(tag.name)}
			<li>
				<a class="link" href={resolve('/archivio/[tag]', { tag: tag.name })}>#{tag.name}</a>
				({tag.count})
			</li>
		{/each}
	</ul>

	<h2 class="mt-8 text-2xl font-medium">Segui</h2>

	<ul class="mt-4 space-y-1">
		<li>
			<a class="link" href={resolve('/feed.xml')}>Feed RSS</a>
		</li>
	</ul>
</div>
