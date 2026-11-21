<script lang="ts">
	let { saving, saved }: { saving: boolean; saved: boolean } = $props();
</script>

{#if saving || saved}
	<div
		class="fixed bottom-0 left-0 flex h-9 w-full items-center"
		class:bg-white={!saved}
		class:dark:bg-slate-900={!saved}
		class:dark:text-white={!saved}
		class:border-t={!saved}
		class:border-slate-200={!saved}
		class:dark:border-slate-700={!saved}
		class:bg-green-500={saved}
		class:text-white={saved}
	>
		<div class="container max-w-6xl">
			{#if saved}
				<span class="font-medium">Saved!</span>
			{:else}
				Saving...
			{/if}
		</div>
	</div>
{/if}
