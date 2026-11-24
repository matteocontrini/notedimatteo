import { describe, expect, it } from 'vitest';
import { markdownLanguage } from './language';

// links returns the text of each link and image node in doc
const links = (doc: string) => {
	const found: string[] = [];
	markdownLanguage.parser.parse(doc).iterate({
		enter: (node) => {
			if (node.name === 'Link' || node.name === 'Image') found.push(doc.slice(node.from, node.to));
		},
	});
	return found;
};

describe('the language', () => {
	it('parses links and images with an address or a label', () => {
		expect(links('[a](https://example.com) [b]() [c][d] ![e](f.png)')).toEqual([
			'[a](https://example.com)',
			'[b]()',
			'[c][d]',
			'![e](f.png)',
		]);
	});

	it('keeps brackets without an address or a label as text', () => {
		expect(links('[hello] ![alt] [img]a.jpg[/img] []')).toEqual([]);
	});

	it('keeps the markup inside such brackets', () => {
		const names: string[] = [];
		markdownLanguage.parser
			.parse('[a *b* c]')
			.iterate({ enter: (node) => void names.push(node.name) });
		expect(names).toContain('Emphasis');
	});

	it('links the outer brackets around brackets without a link', () => {
		expect(links('[a [b] c](https://example.com)')).toEqual(['[a [b] c](https://example.com)']);
	});
});
