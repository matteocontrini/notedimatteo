import { describe, expect, it } from 'vitest';
import { htmlToMarkdown, textToPaste } from './paste';
import { stateOf } from './testing';

describe('pasted HTML', () => {
	it('keeps links, as copied from a page in Firefox', () => {
		const html =
			'<html><head><meta charset="utf-8"></head><body><p>Read the ' +
			'<a href="https://example.com/a" class="">first part</a> and the ' +
			'<a href="https://example.com/b">second</a>.</p></body></html>';
		expect(htmlToMarkdown(html)).toBe(
			'Read the [first part](https://example.com/a) and the [second](https://example.com/b).',
		);
	});

	it('keeps the lines and language of highlighted code blocks', () => {
		const line = (code: string) =>
			`<div class="token-line"><span class="token keyword">${code}</span><br></div>`;
		const html =
			'<p>Before:</p><div class="codeBlockContainer"><pre class="prism-code language-go">' +
			`<code>${line('fmt.Println(1)')}${line('fmt.Println(2)')}</code></pre></div><p>After.</p>`;
		expect(htmlToMarkdown(html)).toBe(
			'Before:\n\n```go\nfmt.Println(1)\nfmt.Println(2)\n```\n\nAfter.',
		);
	});

	it('uses a longer fence when the code contains one', () => {
		expect(htmlToMarkdown('<pre><code>```\ncode\n```</code></pre>')).toBe(
			'````\n```\ncode\n```\n````',
		);
	});

	it('writes lists with one space after the marker', () => {
		const html =
			'<ul><li>one <b>two</b></li><li>three<ul><li>nested</li></ul></li></ul>' +
			'<ol start="3"><li>a</li><li>b</li></ol>';
		expect(htmlToMarkdown(html)).toBe('- one **two**\n- three\n  - nested\n\n3. a\n4. b');
	});

	it('keeps lists tight when each item is a single paragraph', () => {
		const html = '<ol><li><p><strong>One</strong> first</p></li><li><p>Two</p></li></ol>';
		expect(htmlToMarkdown(html)).toBe('1. **One** first\n2. Two');
	});

	it('separates the items of a list with several paragraphs in an item', () => {
		const html = '<ul><li><p>One</p><p>More</p></li><li><p>Two</p></li></ul>';
		expect(htmlToMarkdown(html)).toBe('- One\n\n  More\n\n- Two');
	});

	it('keeps the bold and italic text of Google Docs, which marks them with styles', () => {
		const html =
			'<meta charset="utf-8"><b style="font-weight:normal;" id="docs-internal-guid-1">' +
			'<p dir="ltr"><span style="font-weight:400">Plain </span>' +
			'<span style="font-weight:700">bold</span>' +
			'<span style="font-style:italic"> italic</span></p></b>';
		expect(htmlToMarkdown(html)).toBe('Plain **bold** *italic*');
	});

	it('keeps quotes', () => {
		expect(htmlToMarkdown('<blockquote><p>Quoted</p></blockquote><p>After</p>')).toBe(
			'> Quoted\n\nAfter',
		);
	});

	it('drops images, which would load from the other site', () => {
		expect(
			htmlToMarkdown(
				'<p><a href="https://example.com">Link</a> <img src="https://x/y.png" alt="photo"></p>',
			),
		).toBe('[Link](https://example.com)');
	});

	it('leaves HTML without formatting to the plain text paste', () => {
		expect(htmlToMarkdown('<span>Just text</span>')).toBeNull();
		expect(htmlToMarkdown('<p>Text <img src="https://x/y.png"></p>')).toBeNull();
		expect(htmlToMarkdown('')).toBeNull();
	});
});

describe('pasting', () => {
	const list = '<ul><li>one</li><li>two</li></ul>';

	it('inside a quote keeps every line in it', () => {
		expect(textToPaste(stateOf('> |'), list, 'one\ntwo')).toBe('- one\n> - two');
		expect(textToPaste(stateOf('> > |'), list, '')).toBe('- one\n> > - two');
		expect(textToPaste(stateOf('> |'), '', 'one\r\n\r\ntwo')).toBe('one\n>\n> two');
	});

	it('outside a quote leaves plain text to CodeMirror', () => {
		expect(textToPaste(stateOf('|'), list, 'one\ntwo')).toBe('- one\n- two');
		expect(textToPaste(stateOf('|'), '', 'one\ntwo')).toBeNull();
		// Before the quote markers isn't inside the quote
		expect(textToPaste(stateOf('|> a'), '', 'one\ntwo')).toBeNull();
	});

	it('inside a code block leaves everything to CodeMirror', () => {
		expect(textToPaste(stateOf('```\n|\n```'), list, 'one\ntwo')).toBeNull();
	});
});
