<script lang="ts">
	let { label, count } = $props();
</script>

<div class="border border-slate-300 dark:border-slate-500 dark:bg-slate-700 px-2 py-1 text-sm font-semibold">
	{label}
	<span class="font-normal ml-1">
		{count}
	</span>
</div>
