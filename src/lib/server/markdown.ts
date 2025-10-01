import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import rehypeStringify from 'rehype-stringify';
import { rehypeImgPlugin } from './rehype-img-plugin';
import { STORAGE_PUBLIC_BASE_URL } from '$env/static/private';

// prettier-ignore
const processor = unified()
	.use(remarkParse)
	.use(remarkRehype)
	.use(rehypeImgPlugin, { storageBaseUrl: STORAGE_PUBLIC_BASE_URL })
	.use(rehypeStringify);

export async function renderMarkdown(markdown: string): Promise<string> {
	const file = await processor.process(markdown);
	return file.toString();
}
