// Renders every .md file in a directory with the SvelteKit app's Markdown
// pipeline (src/lib/server/markdown.ts), writing the expected .html next to it.
// The Go compatibility test compares the Go renderer's output with these files.
//
// Usage: node internal/markdown/testdata/generate.mjs [directory]
// The default directory is testdata/compat. Keep the plugins in sync with markdown.ts.
import { readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import rehypeStringify from 'rehype-stringify';
import { rehypeImgPlugin } from '../../../src/lib/server/rehype-img-plugin.ts';
import remarkStrikethroughOnly from '../../../src/lib/server/markdown-strikethrough.ts';

// Must match storageBaseURL in compat_test.go
const storageBaseUrl = 'https://storage.example.com';

const processor = unified()
	.use(remarkParse)
	.use(remarkStrikethroughOnly)
	.use(remarkRehype)
	.use(rehypeImgPlugin, { storageBaseUrl })
	.use(rehypeStringify);

const dir = process.argv[2] ?? join(import.meta.dirname, 'compat');

let count = 0;
for (const name of readdirSync(dir).filter((f) => f.endsWith('.md'))) {
	const markdown = readFileSync(join(dir, name), 'utf8');
	const html = String(await processor.process(markdown));
	writeFileSync(join(dir, name.replace(/\.md$/, '.html')), html);
	count++;
}

console.log(`Rendered ${count} files in ${dir}`);
