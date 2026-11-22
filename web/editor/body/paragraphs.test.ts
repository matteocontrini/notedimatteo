import { describe, expect, it } from 'vitest';
import { continueMarkup, hiddenBlankLines, insertParagraph } from './paragraphs';
import { run, stateOf } from './testing';

describe('hidden blank lines', () => {
	it('are the blank lines between blocks', () => {
		expect(hiddenBlankLines(stateOf('A\n\nB\n\n\nC|'))).toEqual([
			{ first: 2, last: 2 },
			{ first: 4, last: 5 },
		]);
	});

	it('include lines with only spaces, and the ones at the start and end', () => {
		expect(hiddenBlankLines(stateOf('\n|A\n  \nB\n'))).toEqual([
			{ first: 1, last: 1 },
			{ first: 3, last: 3 },
			{ first: 5, last: 5 },
		]);
	});

	it("don't include a blank line with a cursor", () => {
		expect(hiddenBlankLines(stateOf('A\n\n|'))).toEqual([{ first: 2, last: 2 }]);
		expect(hiddenBlankLines(stateOf('A\n\n|\n\nB'))).toEqual([
			{ first: 2, last: 2 },
			{ first: 4, last: 4 },
		]);
	});

	it("don't include blank lines inside code blocks", () => {
		expect(hiddenBlankLines(stateOf('A\n\n```\nx\n\ny\n```\n\nB|'))).toEqual([
			{ first: 2, last: 2 },
			{ first: 8, last: 8 },
		]);
	});
});

describe('Enter', () => {
	it('starts a new paragraph', () => {
		expect(run(insertParagraph, 'A|')).toBe('A\n\n|');
		expect(run(insertParagraph, 'Hello wor|ld')).toBe('Hello wor\n\n|ld');
		expect(run(insertParagraph, '# Title|')).toBe('# Title\n\n|');
	});

	it('on a blank line only separates it from the block above', () => {
		expect(run(insertParagraph, 'A\n|')).toBe('A\n\n|');
		expect(run(insertParagraph, 'A\n\n|')).toBe('A\n\n|');
		expect(run(insertParagraph, '|')).toBe('|');
	});

	it('leaves code blocks to the plain line break, except after the closing fence', () => {
		expect(run(insertParagraph, '```\nco|de\n```')).toBeNull();
		expect(run(insertParagraph, '```\ncode\n```|')).toBe('```\ncode\n```\n\n|');
		// While typing a code block, before its closing fence
		expect(run(insertParagraph, 'A\n\n```|')).toBeNull();
		expect(run(insertParagraph, '```\ncode()|')).toBeNull();
	});

	it('continues lists and quotes', () => {
		expect(run(continueMarkup, '- a|')).toBe('- a\n- |');
		expect(run(continueMarkup, '1. a|')).toBe('1. a\n2. |');
		expect(run(continueMarkup, '> a|')).toBe('> a\n> |');
		expect(run(continueMarkup, 'text|')).toBeNull();
	});

	it('on an empty list item leaves the list and starts a paragraph', () => {
		expect(run(continueMarkup, '- a\n- |')).toBe('- a\n\n|');
		expect(run(continueMarkup, '- a\n- b\n- |')).toBe('- a\n- b\n\n|');
	});
});
