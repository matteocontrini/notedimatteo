import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import rehypeStringify from 'rehype-stringify';
import { rehypeImgPlugin } from './rehype-img-plugin';
import { STORAGE_PUBLIC_BASE_URL } from '$env/static/private';
import remarkStrikethroughOnly from '$lib/server/markdown-strikethrough';
import { db } from '$lib/server/db';

export const MARKDOWN_RENDERER_VERSION = 1;

// prettier-ignore
const processor = unified()
	.use(remarkParse)
	.use(remarkStrikethroughOnly)
	.use(remarkRehype)
	.use(rehypeImgPlugin, { storageBaseUrl: STORAGE_PUBLIC_BASE_URL })
	.use(rehypeStringify);

export async function renderMarkdown(markdown: string): Promise<string> {
	const file = await processor.process(markdown);
	return file.toString();
}

export async function renderPostBody(post: {
	id: number;
	body: string;
	bodyRevision: number;
	htmlCache: {
		rendererVersion: number;
		bodyRevision: number;
		html: string;
	} | null;
}): Promise<string> {
	if (
		post.htmlCache?.rendererVersion === MARKDOWN_RENDERER_VERSION &&
		post.htmlCache.bodyRevision === post.bodyRevision
	) {
		return post.htmlCache.html;
	}

	const html = await renderMarkdown(post.body);

	await db.postHtmlCache.upsert({
		where: { postId: post.id },
		create: {
			postId: post.id,
			rendererVersion: MARKDOWN_RENDERER_VERSION,
			bodyRevision: post.bodyRevision,
			html
		},
		update: {
			rendererVersion: MARKDOWN_RENDERER_VERSION,
			bodyRevision: post.bodyRevision,
			html,
			renderedAt: new Date()
		}
	});

	return html;
}
