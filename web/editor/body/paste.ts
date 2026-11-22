// Pasting. An image file uploads. Formatted text, as copied from a web page,
// becomes Markdown, keeping links, emphasis, code, lists, quotes, and
// headings. Anything else, and anything pasted inside a code block, pastes as
// plain text.

import { EditorView } from '@codemirror/view';
import TurndownService from 'turndown';
import { insideCodeBlock } from './blocks';

// formattedHTML reports whether the HTML has formatting that Markdown keeps;
// without it, the plain text that comes with it pastes the same.
const formattedHTML = (html: string) =>
	/<a\s[^>]*href/i.test(html) ||
	/<(b|strong|i|em|s|del|code|pre|li|blockquote|h[1-6])[\s>]/i.test(html) ||
	/font-weight:\s*(bold|[6-9]00)|font-style:\s*italic/i.test(html);

const turndown = new TurndownService({
	headingStyle: 'atx',
	codeBlockStyle: 'fenced',
	bulletListMarker: '-',
	emDelimiter: '*',
});

// Images would load from the other site in the post
turndown.addRule('image', { filter: 'img', replacement: () => '' });

// Google Docs wraps the copied text in a <b> that isn't bold, and marks bold
// and italic text with styles on <span>s
const style = (node: HTMLElement) => node.getAttribute('style') ?? '';
turndown.addRule('notBold', {
	filter: (node) => node.nodeName === 'B' && /font-weight:\s*(normal|400)/.test(style(node)),
	replacement: (content) => content,
});
turndown.addRule('boldSpan', {
	filter: (node) => node.nodeName === 'SPAN' && /font-weight:\s*(bold|[6-9]00)/.test(style(node)),
	replacement: (content) => (content.trim() ? `**${content}**` : content),
});
turndown.addRule('italicSpan', {
	filter: (node) => node.nodeName === 'SPAN' && /font-style:\s*italic/.test(style(node)),
	replacement: (content) => (content.trim() ? `*${content}*` : content),
});

// Turndown's own list items put three spaces after the marker
turndown.addRule('listItem', {
	filter: 'li',
	replacement: (content, node) => {
		const item = node as HTMLElement;
		const list = item.parentNode as HTMLElement;
		let prefix = '- ';
		if (list.nodeName === 'OL') {
			const start = Number(list.getAttribute('start') ?? 1);
			prefix = `${start + Array.prototype.indexOf.call(list.children, item)}. `;
		}
		const paragraph = /\n$/.test(content);
		const text = content.replace(/^\n+|\n+$/g, '') + (paragraph ? '\n' : '');
		const indented = text.replace(/\n/g, '\n' + ' '.repeat(prefix.length));
		return prefix + indented + (item.nextSibling ? '\n' : '');
	},
});

// codeText returns a code block's text with its line breaks. Syntax
// highlighters often put each line in its own <div> and end it with <br>,
// which textContent drops.
const codeText = (node: Node): string => {
	if (node.nodeType === node.TEXT_NODE) return node.textContent ?? '';
	if (node.nodeName === 'BR') return '\n';
	const text = Array.from(node.childNodes, codeText).join('');
	const block = node.nodeName === 'DIV' || node.nodeName === 'P';
	return block && text !== '' && !text.endsWith('\n') ? text + '\n' : text;
};

turndown.addRule('codeBlock', {
	filter: 'pre',
	replacement: (_content, node) => {
		const pre = node as HTMLElement;
		const code = codeText(pre).replace(/\n+$/, '');
		const classes = `${pre.className} ${pre.querySelector('code')?.className ?? ''}`;
		const language = /language-([\w+-]+)/.exec(classes)?.[1] ?? '';
		const longestRun = Math.max(0, ...(code.match(/`+/g) ?? []).map((run) => run.length));
		const fence = '`'.repeat(Math.max(3, longestRun + 1));
		return `\n\n${fence}${language}\n${code}\n${fence}\n\n`;
	},
});

// htmlToMarkdown converts formatted HTML to Markdown. It returns null when the
// HTML has no formatting to keep.
export const htmlToMarkdown = (html: string) => {
	if (!formattedHTML(html)) return null;
	const markdown = turndown.turndown(html).trim();
	return markdown === '' ? null : markdown;
};

// paste handles pasting in the editor. onImage receives a pasted image file.
export const paste = (onImage: (file: File) => void) =>
	EditorView.domEventHandlers({
		paste(event, view) {
			const data = event.clipboardData;
			const image = Array.from(data?.items ?? [])
				.find((item) => item.type.startsWith('image/'))
				?.getAsFile();
			if (image) {
				event.preventDefault();
				onImage(image);
				return true;
			}

			if (insideCodeBlock(view.state, view.state.selection.main.head)) return false;
			const markdown = htmlToMarkdown(data?.getData('text/html') ?? '');
			if (markdown === null) return false;
			event.preventDefault();
			view.dispatch(view.state.replaceSelection(markdown), {
				scrollIntoView: true,
				userEvent: 'input.paste',
			});
			return true;
		},
	});
