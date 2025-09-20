<script lang="ts">
	import { page } from '$app/state';

	let { label, count } = $props();
	let link = `/${label.toLowerCase().replace(/[^a-z]+/g, '')}`;
	let isCurrent = $derived(page.url.pathname === link);
</script>

<a
	class="border border-slate-300 px-2 py-1 text-sm font-semibold hover:border-slate-700 dark:border-slate-500 dark:bg-slate-700"
	class:border-slate-700={isCurrent}
	href={link}
>
	{label}
	<span class="ml-1 font-normal">
		{count}
	</span>
</a>
