<script lang="ts">
	import { userPrefersMode } from 'mode-watcher';
</script>

<footer class="container flex max-w-6xl flex-col gap-4 py-12 sm:flex-row">
	<select class="h-9 py-0 pr-10 pl-3 sm:ml-auto" bind:value={userPrefersMode.current}>
		<option value="system">Automatico</option>
		<option value="light">Chiaro</option>
		<option value="dark">Scuro</option>
	</select>
</footer>
