import { describe, expect, it } from 'vitest';
import {
	insertMarkdownLink,
	toggleMarkdownBold,
	toggleMarkdownItalic,
	toggleMarkdownQuote,
} from './markdown';
import { normalizeSlug } from './slug';
import { completeCurrentTag, getCurrentTag } from './tags';
import { insertImageMarkup } from './uploads';

describe('post editor utilities', () => {
	it('inserts Markdown links for empty, URL, and text selections', () => {
		expect(insertMarkdownLink('hello', 5, 5)).toEqual({
			value: 'hello[]()',
			selectionStart: 6,
			selectionEnd: 6,
		});
		expect(insertMarkdownLink('https://example.com', 0, 19).value).toBe('[](https://example.com)');
		expect(insertMarkdownLink('hello', 0, 5).value).toBe('[hello]()');
	});

	it('wraps Markdown bold and italic selections', () => {
		expect(toggleMarkdownBold('hello', 0, 5)).toEqual({
			value: '**hello**',
			selectionStart: 2,
			selectionEnd: 7,
		});
		expect(toggleMarkdownItalic('', 0, 0)).toEqual({
			value: '**',
			selectionStart: 1,
			selectionEnd: 1,
		});
	});

	it('quotes every selected line', () => {
		expect(toggleMarkdownQuote('one\ntwo', 0, 7)).toEqual({
			value: '> one\n> two',
			selectionStart: 2,
			selectionEnd: 11,
		});
	});

	it('normalizes slugs', () => {
		expect(normalizeSlug('È già  Un Titolo!')).toBe('e-gia-un-titolo');
	});

	it('finds and completes the tag at the cursor', () => {
		expect(getCurrentTag('one, tw, three', 6)).toBe('tw');
		expect(completeCurrentTag('one, tw, three', 6, 'two')).toEqual({
			value: 'one,two, three',
			cursorPosition: 7,
		});
	});

	it('inserts image markup with paragraph spacing', () => {
		expect(insertImageMarkup('beforeafter', '[img]url[/img]', 6)).toEqual({
			body: 'before\n\n[img]url[/img]\n\nafter',
			cursorPosition: 24,
		});
	});
});
