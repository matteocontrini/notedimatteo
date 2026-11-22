import { EditorSelection, EditorState, type StateCommand } from '@codemirror/state';
import { describe, expect, it } from 'vitest';
import { insertLink, quoteLines, wrapBold, wrapItalic } from './formatting';

// run applies command to doc with the given selections, and returns the text
// and the selections after it.
const run = (command: StateCommand, doc: string, ...ranges: [number, number][]) => {
	let state = EditorState.create({
		doc,
		selection: EditorSelection.create(
			ranges.map(([anchor, head]) => EditorSelection.range(anchor, head)),
		),
		extensions: EditorState.allowMultipleSelections.of(true),
	});
	command({ state, dispatch: (transaction) => (state = transaction.state) });
	return {
		doc: state.doc.toString(),
		selections: state.selection.ranges.map((range) => [range.anchor, range.head]),
	};
};

describe('Markdown commands', () => {
	it('insert links for empty, URL, and text selections', () => {
		expect(run(insertLink, 'hello', [5, 5])).toEqual({ doc: 'hello[]()', selections: [[6, 6]] });
		expect(run(insertLink, 'https://example.com', [0, 19])).toEqual({
			doc: '[](https://example.com)',
			selections: [[1, 1]],
		});
		expect(run(insertLink, 'hello', [0, 5])).toEqual({ doc: '[hello]()', selections: [[8, 8]] });
	});

	it('wrap bold and italic selections', () => {
		expect(run(wrapBold, 'hello', [0, 5])).toEqual({ doc: '**hello**', selections: [[2, 7]] });
		expect(run(wrapItalic, '', [0, 0])).toEqual({ doc: '**', selections: [[1, 1]] });
	});

	it('wrap every selection', () => {
		expect(run(wrapBold, 'one two', [0, 3], [4, 7])).toEqual({
			doc: '**one** **two**',
			selections: [
				[2, 5],
				[10, 13],
			],
		});
	});

	it('quote every selected line, and blank lines too', () => {
		expect(run(quoteLines, 'one\ntwo', [0, 7])).toEqual({
			doc: '> one\n> two',
			selections: [[2, 11]],
		});
		expect(run(quoteLines, 'one\n\ntwo', [1, 8])).toEqual({
			doc: '> one\n>\n> two',
			selections: [[3, 13]],
		});
		expect(run(quoteLines, 'one', [1, 1])).toEqual({ doc: 'o> ne', selections: [[3, 3]] });
	});
});
