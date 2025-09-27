<script lang="ts">
	import Post from '$lib/Post.svelte';

	let { data } = $props();
	const canEdit = $derived(data.isLoggedIn);

	const fallbackTitle = $derived(
		`Nota del ${new Date(data.post.publishedAt!).toLocaleDateString('it-IT', {
			timeZone: 'Europe/Rome',
			day: '2-digit',
			month: 'long',
			year: 'numeric'
		})}`
	);
	const pageTitle = $derived(data.post.title?.trim() || fallbackTitle);
</script>

<div class="lg:w-7/12">
	<Post post={data.post} canEdit={canEdit} />
</div>
