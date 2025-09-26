import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import rehypeStringify from 'rehype-stringify';
import preview, { textFormatter } from 'remark-preview';

const processor = unified()
	.use(remarkParse)
	.use(
		preview(
			textFormatter({
				length: 250,
				maxBlocks: 2
			})
		)
	)
	.use(remarkRehype)
	.use(rehypeStringify);

async function processMarkdown(markdown: string) {
	const file = await processor.process(markdown);
	const data = file.data as PreviewData;
	return {
		html: file.toString(),
		preview: data.fm?.preview ?? ''
	};
}

export async function renderMarkdown(markdown: string): Promise<string> {
	const { html } = await processMarkdown(markdown);
	return html;
}

export async function renderMarkdownWithPreview(markdown: string): Promise<{
	html: string;
	preview: string;
}> {
	return processMarkdown(markdown);
}

type PreviewData = {
	fm?: {
		preview?: string;
	};
};
