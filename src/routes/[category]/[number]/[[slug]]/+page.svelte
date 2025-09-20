<script lang="ts">
	import Post from '$lib/Post.svelte';

	let { data } = $props();
</script>

<div class="lg:w-7/12">
	<Post post={data.post} />
</div>
