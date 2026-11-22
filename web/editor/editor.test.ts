import { describe, expect, it } from 'vitest';
import { normalizeSlug } from './slug';
import { completeCurrentTag, getCurrentTag } from './tags';
import { imageInsertion } from './uploads';

describe('image uploads', () => {
	it('insert markup in its own paragraph', () => {
		expect(imageInsertion('before', 'after', '[img]a.jpg[/img]')).toBe('\n\n[img]a.jpg[/img]\n\n');
		expect(imageInsertion('e\n', '\n\na', 'X')).toBe('\nX');
		expect(imageInsertion('', '', 'X')).toBe('X\n\n');
	});
});

describe('post fields', () => {
	it('normalize slugs', () => {
		expect(normalizeSlug('È già  Un Titolo!')).toBe('e-gia-un-titolo');
	});

	it('find and complete the tag at the cursor', () => {
		expect(getCurrentTag('one, tw, three', 6)).toBe('tw');
		expect(completeCurrentTag('one, tw, three', 6, 'two')).toEqual({
			value: 'one,two, three',
			cursorPosition: 7,
		});
	});
});
