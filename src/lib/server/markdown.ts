import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import rehypeStringify from 'rehype-stringify';

// prettier-ignore
const processor = unified()
	.use(remarkParse)
	.use(remarkRehype)
	.use(rehypeStringify);

export async function renderMarkdown(markdown: string): Promise<string> {
	const file = await processor.process(markdown);
	return file.toString();
}
