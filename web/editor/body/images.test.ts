import { Text } from '@codemirror/state';
import { describe, expect, it } from 'vitest';
import { imageDecorations, previewURL } from './images';

describe('images', () => {
	it('show the preview of each image markup, with its size, and dim its tags', () => {
		const doc = Text.of([
			'text',
			'[img width=1200 height=675]2026/11/abc.jpg[/img] [img]b[/img]',
			'[img width=wide]c.png[/img]',
		]);
		const found: { at: number; src: string; width?: string; height?: string }[] = [];
		const dimmed: string[] = [];
		imageDecorations(doc, 'https://storage.example.com').between(
			0,
			doc.length,
			(from, to, deco) => {
				if (!deco.spec.widget) {
					dimmed.push(doc.sliceString(from, to));
					return;
				}
				const { src, width, height } = deco.spec.widget;
				found.push({ at: from, src, width, height });
			},
		);

		const lineEnd = (number: number) => doc.line(number).to;
		expect(found).toEqual([
			{
				at: lineEnd(2),
				src: 'https://storage.example.com/2026/11/abc_preview.jpg',
				width: '1200',
				height: '675',
			},
			{ at: lineEnd(2), src: 'https://storage.example.com/b_preview' },
			{ at: lineEnd(3), src: 'https://storage.example.com/c_preview.png' },
		]);
		expect(dimmed).toEqual([
			'[img width=1200 height=675]',
			'[/img]',
			'[img]',
			'[/img]',
			'[img width=wide]',
			'[/img]',
		]);
	});

	it('put _preview before the extension of the file name', () => {
		expect(previewURL('https://s', 'a.b/c')).toBe('https://s/a.b/c_preview');
		expect(previewURL('https://s', 'x/y.jpeg')).toBe('https://s/x/y_preview.jpeg');
	});
});
