<script lang="ts">
	let { children } = $props();
</script>

<kbd
	class="rounded border border-slate-300 bg-slate-100 px-1.5 py-0.5 dark:border-slate-600 dark:bg-slate-800"
>
	{@render children()}
</kbd>
