<script lang="ts">
	let { children } = $props();
</script>

<kbd class="px-1.5 py-0.5 bg-slate-100 dark:bg-slate-800 rounded border border-slate-300 dark:border-slate-600">
	{@render children()}
</kbd>
