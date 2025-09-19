<script lang="ts">
	import Post from './Post.svelte';
</script>


<div class="grid md:grid-cols-12 gap-20">

	<div class="md:col-span-7">

		<h2 class="text-2xl font-medium font-serif">
			18 settembre 2025
		</h2>

		<Post />

		<hr class="my-6" />

		<Post />

		<hr class="my-6" />

		<h2 class="mt-14 text-2xl font-medium font-serif">
			18 settembre 2025
		</h2>

		<Post />

		<hr class="my-6" />

		<Post />

	</div>

	<div class="md:col-span-5">
		<h2 class="text-2xl font-medium font-serif">
			Archivio
		</h2>

		<ul class="mt-4">
			<li class="font-semibold">
				2025
			</li>
			<li class="mt-1">
				<a href="" class="link">
					settembre
				</a>
				(30)
			</li>
		</ul>

		<h2 class="mt-8 text-2xl font-medium font-serif">
			Tag
		</h2>

		<ul class="mt-4">
			<li>
				<a href="" class="link">
					#cloud
				</a>
				(2)
			</li>
			<li class="mt-1">
				<a href="" class="link">
					#cloud
				</a>
			</li>
			<li class="mt-1">
				<a href="" class="link">
					#cloud
				</a>
			</li>
			<li class="mt-1">
				<a href="" class="link">
					#cloud
				</a>
			</li>
			<li class="mt-1">
				<a href="" class="link">
					#cloud
				</a>
			</li>
		</ul>
	</div>

</div>
