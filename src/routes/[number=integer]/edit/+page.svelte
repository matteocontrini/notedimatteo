<script lang="ts">
	import type { PostForEdit } from '$lib/types';

	let { data } = $props<{ data: { post: PostForEdit } }>();

	const optionalText = (value: string | null) => value ?? '';
	const formatDatetimeLocal = (value: Date | null) =>
		value ? new Date(value).toISOString().slice(0, 16) : '';

	const TEXTAREA_BUFFER_PX = 12;

	const autoResize = (node: HTMLTextAreaElement) => {
		const resize = () => {
			node.style.height = 'auto';
			node.style.height = `${node.scrollHeight + TEXTAREA_BUFFER_PX}px`;
		};

		resize();

		node.addEventListener('input', resize);

		return {
			destroy() {
				node.removeEventListener('input', resize);
			},
			update: resize
		};
	};
</script>
<form class="grid gap-12 md:grid-cols-12">
	<div class="md:col-span-7">
		<label class="block">
			<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Title</span>
			<input class="w-full h-9" name="title" value={optionalText(data.post.title)} />
		</label>

		<label class="mt-3 flex items-center gap-3">
			<input class="accent-slate-800 dark:accent-slate-200" type="checkbox" name="hideTitle"
						 checked={data.post.hideTitle} />
			<span class="text-sm text-slate-700 dark:text-slate-300">Hide title</span>
		</label>

		<label class="block mt-6">
			<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Body</span>
			<textarea
				class="w-full resize-none overflow-hidden"
				use:autoResize
				name="body"
				rows="16"
			>{data.post.body}</textarea>
		</label>
	</div>

	<aside class="md:col-span-5 space-y-6">
		<section class="grid gap-4 md:grid-cols-2">
			<label class="block">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">ID</span>
				<input
					class="w-full h-9"
					name="id"
					value={data.post.id.toString()}
					readonly
				/>
			</label>

			<label class="block md:col-start-2">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Number</span>
				<input class="w-full h-9" type="number" name="number" value={data.post.number.toString()} />
			</label>

			<label class="block md:col-span-2">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Category</span>
				<input class="w-full h-9" name="category" value={data.post.category} />
			</label>

			<label class="block md:col-span-2">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Slug</span>
				<input class="w-full h-9" name="slug" value={optionalText(data.post.slug)} />
			</label>

			<label class="block md:col-span-2">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Tags (comma separated)</span>
				<input class="w-full h-9" name="tags" value={data.post.tags.join(', ')} />
			</label>
		</section>

		<section class="grid gap-4">
			<label class="block">
				<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Created at</span>
				<input class="w-full h-9" type="datetime-local" name="createdAt" readonly
							 value={formatDatetimeLocal(data.post.createdAt)} />
			</label>

			{#if data.post.updatedAt != null}
				<label class="block">
					<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Updated at</span>
					<input class="w-full h-9" type="datetime-local" name="updatedAt" readonly
								 value={formatDatetimeLocal(data.post.updatedAt)} />
				</label>
			{/if}

			{#if data.post.publishedAt !== null}
				<label class="block">
					<span class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Published at</span>
					<input class="w-full h-9" type="datetime-local" name="publishedAt" readonly
								 value={formatDatetimeLocal(data.post.publishedAt)} />
				</label>
			{/if}
		</section>

		<section class="space-y-4">
			<button type="button" class="w-full">
				Save
			</button>

			{#if data.post.publishedAt === null}
				<button type="button" class="w-full">
					Save & publish
				</button>
			{/if}
		</section>
	</aside>
</form>
