<script lang="ts">
	let { form } = $props();
	const message = $derived(form?.message ?? '');
	const username = $derived(form?.values?.username ?? '');
</script>

<h1 class="text-2xl font-semibold mb-6">Accedi</h1>

<form method="POST" class="space-y-4">
	<div class="flex flex-col gap-2">
		<label class="text-sm font-medium" for="username">Username</label>
		<input
			id="username"
			name="username"
			type="text"
			autocomplete="username"
			required
			value={username}
			class="h-9"
		/>
	</div>

	<div class="flex flex-col gap-2">
		<label class="text-sm font-medium" for="password">Password</label>
		<input
			id="password"
			name="password"
			type="password"
			autocomplete="current-password"
			required
			class="h-9"
		/>
	</div>

	{#if message}
		<p class="text-sm text-red-600">{message}</p>
	{/if}

	<button type="submit" class="w-full h-9">
		Accedi
	</button>
</form>
