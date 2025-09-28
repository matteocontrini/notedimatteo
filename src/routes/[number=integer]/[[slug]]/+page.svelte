<script lang="ts">
	import Post from '$lib/Post.svelte';

	let { data } = $props();
	const canEdit = $derived(data.isLoggedIn);
</script>

<div class="lg:w-7/12">
	<Post post={data.post} canEdit={canEdit} />
</div>
