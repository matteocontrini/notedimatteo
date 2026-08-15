<script lang="ts">
	import type { PostForEdit } from '$lib/types';

	let { post }: { post: PostForEdit } = $props();

	const formatDatetimeLocal = (value: Date | null) => {
		if (!value) return '';

		// 2000-01-01T00:00:00
		const date = value.toLocaleDateString('sv-SE', { timeZone: 'Europe/Rome' });
		const time = value.toLocaleTimeString('sv-SE', { hour12: false, timeZone: 'Europe/Rome' });
		return `${date}T${time}`;
	};
</script>

<section class="grid gap-4">
	<label class="block">
		<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">Created at</span
		>
		<input
			class="h-9 w-full"
			type="datetime-local"
			name="createdAt"
			readonly
			value={formatDatetimeLocal(post.createdAt)}
		/>
	</label>

	{#if post.updatedAt != null}
		<label class="block">
			<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300"
				>Updated at</span
			>
			<input
				class="h-9 w-full"
				type="datetime-local"
				name="updatedAt"
				readonly
				value={formatDatetimeLocal(post.updatedAt)}
			/>
		</label>
	{/if}

	{#if post.publishedAt !== null}
		<label class="block">
			<span class="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300"
				>Published at</span
			>
			<input
				class="h-9 w-full"
				type="datetime-local"
				name="publishedAt"
				readonly
				value={formatDatetimeLocal(post.publishedAt)}
			/>
		</label>
	{/if}
</section>
