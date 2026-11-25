import { describe, expect, it } from 'vitest';
import {
	continueMarkup,
	enter,
	hiddenBlankLines,
	hiddenLinesOfRun,
	insertParagraph,
	joinParagraphs,
	paragraphs,
} from './paragraphs';
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

	it('include the lines with only markers between paragraphs of a quote', () => {
		expect(hiddenBlankLines(stateOf('> A\n>\n> B|'))).toEqual([{ first: 2, last: 2 }]);
		// Three lines with only markers show an empty paragraph in the quote
		expect(hiddenBlankLines(stateOf('> A\n>\n>\n>\n> B|'))).toEqual([
			{ first: 2, last: 2 },
			{ first: 4, last: 4 },
		]);
		expect(hiddenBlankLines(stateOf('> A\n> > B\n>\n> C|'))).toEqual([{ first: 3, last: 3 }]);
	});

	it('around a quote are the space between it and the blocks around it', () => {
		expect(hiddenBlankLines(stateOf('A\n\n> B\n\nC|'))).toEqual([
			{ first: 2, last: 2 },
			{ first: 4, last: 4 },
		]);
		// The quote's last line and the blank line after it make one gap
		expect(hiddenBlankLines(stateOf('> A\n>\n\nB|'))).toEqual([{ first: 2, last: 3 }]);
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

	it('in a quote starts a paragraph in the quote', () => {
		expect(run(enter, '> A|')).toBe('> A\n>\n> |');
		expect(run(enter, '> Hello wor|ld')).toBe('> Hello wor\n>\n> |ld');
		expect(run(enter, '> > A|')).toBe('> > A\n> >\n> > |');
		expect(run(enter, '> > A\n>\n> B|')).toBe('> > A\n>\n> B\n>\n> |');
	});

	it('at the start of a paragraph in a quote adds an empty paragraph above it', () => {
		expect(run(enter, '> A\n>\n> |B')).toBe('> A\n>\n> \n>\n> |B');
	});

	it('before the quote markers moves the quote down', () => {
		expect(run(enter, 'A\n\n|> B')).toBe('A\n\n\n\n|> B');
	});

	it('on a line with only markers right below a paragraph adds the line between them', () => {
		expect(run(enter, '> A\n> |')).toBe('> A\n>\n> |');
	});

	it('on an empty paragraph at the end of a quote leaves the quote', () => {
		expect(run(enter, '> A\n>\n> |')).toBe('> A\n\n|');
		expect(run(enter, '> A\n>\n>|')).toBe('> A\n\n|');
	});

	it('continues lists inside a quote, and leaves them for a paragraph in the quote', () => {
		expect(run(enter, '> - a|')).toBe('> - a\n> - |');
		expect(run(enter, '> - a\n> - |')).toBe('> - a\n>\n> |');
	});

	it('leaves code blocks to the plain line break, except after the closing fence', () => {
		expect(run(insertParagraph, '```\nco|de\n```')).toBeNull();
		expect(run(insertParagraph, '```\ncode\n```|')).toBe('```\ncode\n```\n\n|');
		// While typing a code block, before its closing fence
		expect(run(insertParagraph, 'A\n\n```|')).toBeNull();
		expect(run(insertParagraph, '```\ncode()|')).toBeNull();
	});

	it('after an opening fence adds the closing fence, with the cursor between them', () => {
		expect(run(enter, '```|')).toBe('```\n|\n```');
		expect(run(enter, 'A\n\n```go|')).toBe('A\n\n```go\n|\n```');
		expect(run(enter, '~~~~|')).toBe('~~~~\n|\n~~~~');
		// Text below stays out of the code block
		expect(run(enter, '```|\nB')).toBe('```\n|\n```\n\nB');
		// An opening fence that has its closing fence only gets a new line
		expect(run(enter, '```|\ncode\n```')).toBe('```\n|\ncode\n```');
	});

	it('in a code block inside a quote keeps the quote markers', () => {
		expect(run(enter, '> ```\n> co|de\n> ```')).toBe('> ```\n> co\n> |de\n> ```');
		expect(run(enter, '> ```\n>     a|\n> ```')).toBe('> ```\n>     a\n>     |\n> ```');
		expect(run(enter, '> > ```\n> > a|\n> > ```')).toBe('> > ```\n> > a\n> > |\n> > ```');
		// On an empty code line with only ">"
		expect(run(enter, '> ```\n> a\n>|\n> ```')).toBe('> ```\n> a\n>\n> |\n> ```');
		// Indented code
		expect(run(enter, '>     co|de')).toBe('>     co\n>     |de');
		// In a list inside the quote, the list marker becomes spaces
		expect(run(enter, '> - ```\n>   a|\n>   ```')).toBe('> - ```\n>   a\n>   |\n>   ```');
	});

	it('before the quote markers in a code block adds an empty code line above', () => {
		expect(run(enter, '> ```\n|> code\n> ```')).toBe('> ```\n>\n|> code\n> ```');
	});

	it('after an opening fence in a quote or a list adds the closing fence with the same markers', () => {
		expect(run(enter, '> ```|')).toBe('> ```\n> |\n> ```');
		expect(run(enter, '> ```|\n> text')).toBe('> ```\n> |\n> ```\n>\n> text');
		expect(run(enter, '> - ```|')).toBe('> - ```\n>   |\n>   ```');
		expect(run(enter, '- ```|')).toBe('- ```\n  |\n  ```');
	});

	it('keeps the indentation of the line above in code blocks', () => {
		expect(run(enter, '```\n    a|')).toBe('```\n    a\n    |');
	});

	it('continues lists, and starts paragraphs in quotes', () => {
		expect(run(continueMarkup, '- a|')).toBe('- a\n- |');
		expect(run(continueMarkup, '1. a|')).toBe('1. a\n2. |');
		expect(run(continueMarkup, '> a|')).toBe('> a\n>\n> |');
		expect(run(continueMarkup, 'text|')).toBeNull();
	});

	it('on an empty list item leaves the list and starts a paragraph', () => {
		expect(run(continueMarkup, '- a\n- |')).toBe('- a\n\n|');
		expect(run(continueMarkup, '- a\n- b\n- |')).toBe('- a\n- b\n\n|');
	});
});

describe('Backspace and Delete between two paragraphs', () => {
	it("before the next paragraph's quote markers join the paragraphs", () => {
		expect(run(joinParagraphs, '> A\n>\n|> B', paragraphs)).toBe('> A|B');
		expect(run(joinParagraphs, '> A\n>\n>| B', paragraphs)).toBe('> A|B');
	});

	it('elsewhere are left to the usual commands', () => {
		expect(run(joinParagraphs, '> A\n>\n> |B', paragraphs)).toBeNull();
		expect(run(joinParagraphs, '> A|\n>\n> B', paragraphs)).toBeNull();
		expect(run(joinParagraphs, 'A\n\n|B', paragraphs)).toBeNull();
	});
});
