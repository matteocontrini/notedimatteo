import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

// Builds the assets served by the Go server. The SvelteKit app keeps using vite.config.ts.
export default defineConfig({
	root: 'web',
	plugins: [tailwindcss()],
	build: {
		outDir: 'dist',
		emptyOutDir: true,
		manifest: true,
		rollupOptions: {
			input: 'assets/main.css',
		},
	},
});
