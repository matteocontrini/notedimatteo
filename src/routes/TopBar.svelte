<script lang="ts">
	import { resolve } from '$app/paths';

	let { isLoggedIn, unpublishedCount = 0 } = $props();
</script>

<div class="h-4.5 bg-[#0067B9]">
	<div class="container flex h-full max-w-6xl items-center justify-end gap-4">
		{#if isLoggedIn}
			<a class="text-xs text-white/80 uppercase hover:text-white" href={resolve('/unpublished')}>
				unpublished ({unpublishedCount})
			</a>

			<a
				class="text-xs text-white/80 uppercase hover:text-white"
				href={resolve('/unpublished/new')}
			>
				new
			</a>

			<a class="text-xs text-white/80 uppercase hover:text-white" href={resolve('/logout')}>
				logout
			</a>
		{:else}
			<a class="text-xs text-white/80 uppercase hover:text-white" href="https://matteosonoio.it">
				blog
				<span class="font-[system-ui]">↗︎</span>
			</a>

			<a class="text-xs text-white/80 uppercase hover:text-white" href={resolve('/about')}>
				about
			</a>
		{/if}
	</div>
</div>
