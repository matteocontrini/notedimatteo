import { describe, expect, it } from 'vitest';
import { continueMarkup, hiddenBlankLines, hiddenLinesOfRun, insertParagraph } from './paragraphs';
import { run, stateOf } from './testing';

// shown writes a run of blank lines as S for shown and h for hidden
const shown = (length: number, blockBefore: boolean, blockAfter: boolean) =>
	hiddenLinesOfRun(length, blockBefore, blockAfter)
		.map((hide) => (hide ? 'h' : 'S'))
		.join('');

describe('a run of blank lines', () => {
	it('between blocks is the space between them, plus an empty paragraph for each extra pair', () => {
		expect(shown(1, true, true)).toBe('h');
		expect(shown(2, true, true)).toBe('hh');
		expect(shown(3, true, true)).toBe('hSh');
		expect(shown(5, true, true)).toBe('hShSh');
	});

	it('after the last block ends with an empty paragraph when it has two lines', () => {
		expect(shown(1, true, false)).toBe('h');
		expect(shown(2, true, false)).toBe('hS');
		expect(shown(4, true, false)).toBe('hShS');
	});

	it('before the first block starts with an empty paragraph when it has two lines', () => {
		expect(shown(1, false, true)).toBe('h');
		expect(shown(2, false, true)).toBe('Sh');
	});

	it('without blocks is empty paragraphs', () => {
		expect(shown(1, false, false)).toBe('S');
		expect(shown(3, false, false)).toBe('ShS');
	});
});

describe('hidden blank lines', () => {
	it('are the blank lines between blocks', () => {
		expect(hiddenBlankLines(stateOf('A\n\nB\n\n\nC|'))).toEqual([
			{ first: 2, last: 2 },
			{ first: 4, last: 5 },
		]);
	});

	it("don't depend on the cursor", () => {
		expect(hiddenBlankLines(stateOf('A\n\n\n\nB|'))).toEqual([
			{ first: 2, last: 2 },
			{ first: 4, last: 4 },
		]);
		expect(hiddenBlankLines(stateOf('|A\n\n\n\nB'))).toEqual([
			{ first: 2, last: 2 },
			{ first: 4, last: 4 },
		]);
	});

	it('include lines with only spaces', () => {
		expect(hiddenBlankLines(stateOf('A\n  \nB|'))).toEqual([{ first: 2, last: 2 }]);
	});

	it("don't include a blank line with a cursor, which only odd texts allow", () => {
		expect(hiddenBlankLines(stateOf('A\n|'))).toEqual([]);
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

	it('at the start of a paragraph moves it down, with an empty paragraph above', () => {
		// Three blank lines between A and B: an empty paragraph shows between them
		expect(run(insertParagraph, 'A\n\n|B')).toBe('A\n\n\n\n|B');
		expect(run(insertParagraph, '|A')).toBe('\n\n|A');
	});

	it('at the start of a line inside a paragraph splits the paragraph there', () => {
		expect(run(insertParagraph, 'A\n|B')).toBe('A\n\n\n|B');
	});

	it('on an empty paragraph adds another one', () => {
		expect(run(insertParagraph, 'A\n\n|')).toBe('A\n\n\n\n|');
		expect(run(insertParagraph, '|')).toBe('\n\n|');
	});

	it('on a blank line right below a block only adds the blank line between them', () => {
		expect(run(insertParagraph, 'A\n|')).toBe('A\n\n|');
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
