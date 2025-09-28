<script lang="ts">
	import { resolve } from '$app/paths';

	let { isLoggedIn } = $props();
</script>

<div class="bg-[#0067B9] h-[18px]">
	<div class="container max-w-6xl h-full flex items-center justify-end gap-4">
		{#if isLoggedIn}
			<a
				class="text-white/80 text-xs uppercase hover:text-white"
				href={resolve('/unpublished')}
			>
				unpublished
			</a>

			<a
				class="text-white/80 text-xs uppercase hover:text-white"
				href={resolve('/unpublished/new')}
			>
				new
			</a>

			<a
				class="text-white/80 text-xs uppercase hover:text-white"
				href={resolve('/logout')}
			>
				logout
			</a>
		{:else}
			<a
				class="text-white/80 text-xs uppercase hover:text-white"
				href="https://matteosonoio.it"
			>
				blog
				<span class="font-[system-ui]">↗︎</span>
			</a>

			<a
				class="text-white/80 text-xs uppercase hover:text-white"
				href={resolve('/about')}
			>
				about
			</a>
		{/if}
	</div>
</div>
