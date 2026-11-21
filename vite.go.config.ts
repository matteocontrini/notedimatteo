import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

// Builds the assets served by the Go server. The SvelteKit app keeps using vite.config.ts.
export default defineConfig({
	root: 'web',
	plugins: [tailwindcss(), svelte()],
	build: {
		outDir: 'dist',
		emptyOutDir: true,
		manifest: true,
		rollupOptions: {
			// Rollup names each output file after its key here. The manifest still
			// lists the entries by source path, which the Go server looks up.
			input: { main: 'assets/main.css', site: 'assets/site.ts', editor: 'editor/main.ts' },
		},
	},
});
