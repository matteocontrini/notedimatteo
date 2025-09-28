<script lang="ts">
import { userPrefersMode } from 'mode-watcher';
</script>

<footer class="container max-w-6xl py-12 flex flex-col sm:flex-row gap-4">
	<select class="pl-3 pr-10 py-0 h-9 sm:ml-auto"
					bind:value={userPrefersMode.current}>
		<option value="system">Automatico</option>
		<option value="light">Chiaro</option>
		<option value="dark">Scuro</option>
	</select>
</footer>
