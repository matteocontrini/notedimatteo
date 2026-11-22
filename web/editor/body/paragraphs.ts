// Paragraphs. In Markdown, blank lines separate paragraphs and other blocks.
// The editor hides them and shows the space between blocks instead, as in
// published posts, and Enter inserts the blank line that starts a new
// paragraph. The text stays the same Markdown.
//
// Markdown has no empty paragraphs, but an editor needs them: Enter at the
// start of a paragraph adds one above it. Each pair of extra blank lines is
// an empty paragraph: one shows as an empty line, the other as the space
// after it. What shows depends only on the text, so moving the cursor changes
// nothing.

import { insertNewline, insertNewlineAndIndent } from '@codemirror/commands';
import { insertNewlineContinueMarkupCommand } from '@codemirror/lang-markdown';
import { syntaxTree } from '@codemirror/language';
import {
	EditorSelection,
	type EditorState,
	Prec,
	type Range,
	RangeSet,
	type StateCommand,
	StateField,
} from '@codemirror/state';
import { Decoration, type DecorationSet, EditorView, WidgetType, keymap } from '@codemirror/view';
import { closeFence, codeBlockNodes, insideCodeBlock } from './blocks';

const isBlank = (text: string) => !/\S/.test(text);

// codeBlockLines returns the numbers of the lines inside code blocks, where
// blank lines are part of the code.
const codeBlockLines = (state: EditorState) => {
	const lines = new Set<number>();
	syntaxTree(state).iterate({
		enter(node) {
			if (!codeBlockNodes.has(node.name)) return;
			const last = state.doc.lineAt(node.to).number;
			for (let number = state.doc.lineAt(node.from).number; number <= last; number++) {
				lines.add(number);
			}
			return false;
		},
	});
	return lines;
};

// A BlankRun is a run of blank lines to hide, by line number.
export type BlankRun = { first: number; last: number };

// hiddenLinesOfRun returns which lines of a run of blank lines to hide, by
// position in the run. The lines next to a block are the space between
// blocks; the lines in between alternate between an empty paragraph and the
// space after it. So one blank line, or two, between blocks is a plain
// paragraph break; three are an empty paragraph between two spaces.
export const hiddenLinesOfRun = (length: number, blockBefore: boolean, blockAfter: boolean) =>
	Array.from(
		{ length },
		(_, index) =>
			(blockBefore && index === 0) ||
			(blockAfter && index === length - 1) ||
			(index - (blockBefore ? 0 : 1)) % 2 === 0,
	);

// hiddenBlankLines returns the runs of blank lines that the editor hides.
// Blank lines inside code blocks are part of the code and always show. A
// blank line with a cursor shows too, so the cursor never sits in hidden text;
// Enter never puts it there, but text like "A\n" at the end of a body can.
export const hiddenBlankLines = (state: EditorState): BlankRun[] => {
	const { doc } = state;
	const code = codeBlockLines(state);
	const cursors = new Set(state.selection.ranges.map((range) => doc.lineAt(range.head).number));
	const blank = (number: number) => !code.has(number) && isBlank(doc.line(number).text);

	const hidden: number[] = [];
	for (let first = 1; first <= doc.lines; first++) {
		if (!blank(first)) continue;
		let last = first;
		while (last < doc.lines && blank(last + 1)) last++;
		hiddenLinesOfRun(last - first + 1, first > 1, last < doc.lines).forEach((hide, index) => {
			if (hide && !cursors.has(first + index)) hidden.push(first + index);
		});
		first = last;
	}

	const runs: BlankRun[] = [];
	for (const number of hidden) {
		const run = runs.at(-1);
		if (run && run.last === number - 1) run.last = number;
		else runs.push({ first: number, last: number });
	}
	return runs;
};

// A gap takes the place of hidden blank lines. Between two blocks it's the
// space between paragraphs; at the start or end of the text it has no height.
class GapWidget extends WidgetType {
	constructor(readonly between: boolean) {
		super();
	}

	eq(other: GapWidget) {
		return other.between === this.between;
	}

	get estimatedHeight() {
		return this.between ? 10 : 0;
	}

	toDOM() {
		const gap = document.createElement('div');
		gap.className = this.between ? 'cm-md-gap' : 'cm-md-gap cm-md-gap-edge';
		return gap;
	}
}

const gapBetween = new GapWidget(true);
const gapAtEdge = new GapWidget(false);
const atomic = Decoration.mark({});

type Layout = { gaps: DecorationSet; atomic: RangeSet<Decoration> };

// layout replaces each run of hidden blank lines with a gap. The cursor skips
// each run and the line breaks around it as one unit, so the arrows go from
// the end of a paragraph to the start of the next, and Backspace and Delete
// there join the two paragraphs.
const layout = (state: EditorState): Layout => {
	const { doc } = state;
	const gaps: Range<Decoration>[] = [];
	const skipped: Range<Decoration>[] = [];
	for (const { first, last } of hiddenBlankLines(state)) {
		const between = first > 1 && last < doc.lines;
		const widget = between ? gapBetween : gapAtEdge;
		gaps.push(
			Decoration.replace({ widget, block: true }).range(doc.line(first).from, doc.line(last).to),
		);
		skipped.push(
			atomic.range(
				first > 1 ? doc.line(first - 1).to : 0,
				last < doc.lines ? doc.line(last + 1).from : doc.length,
			),
		);
	}
	return { gaps: Decoration.set(gaps), atomic: RangeSet.of(skipped) };
};

// Hidden lines must come from a state field, not a view plugin
const paragraphLayout = StateField.define<Layout>({
	create: layout,
	update(value, transaction) {
		const treeChanged = syntaxTree(transaction.state) !== syntaxTree(transaction.startState);
		return transaction.docChanged || transaction.selection || treeChanged
			? layout(transaction.state)
			: value;
	},
	provide: (field) => [
		EditorView.decorations.from(field, (value) => value.gaps),
		EditorView.atomicRanges.of((view) => view.state.field(field).atomic),
	],
});

// belowBlock reports whether the line is right below a block, with no blank
// line between them.
const belowBlock = (state: EditorState, number: number) =>
	number > 1 && !isBlank(state.doc.line(number - 1).text);

// separateFromBlockAbove makes a blank line with the cursor, right below a
// block, the start of a new paragraph: it adds the blank line between them.
// Without it, text typed there would continue the block above, like a list's
// last item.
const separateFromBlockAbove = (state: EditorState) =>
	state.changeByRange((range) => {
		const line = state.doc.lineAt(range.head);
		if (!range.empty || !isBlank(line.text) || !belowBlock(state, line.number)) return { range };
		return {
			changes: { from: line.from, insert: '\n' },
			range: EditorSelection.cursor(range.head + 1),
		};
	});

// insertParagraph ends the paragraph at the cursor and starts a new one,
// inserting the blank line between them, and moves the cursor to the new one.
// At the start of a paragraph, the paragraph moves down with the cursor and
// an empty one appears above. On a blank line right below a block, it only
// adds the missing blank line between them. In code blocks it does nothing, so
// Enter inserts a plain line break.
export const insertParagraph: StateCommand = ({ state, dispatch }) => {
	if (state.selection.ranges.some((range) => insideCodeBlock(state, range.head))) return false;

	const changes = state.changeByRange((range) => {
		const line = state.doc.lineAt(range.head);
		if (range.empty && isBlank(line.text) && belowBlock(state, line.number)) {
			return {
				changes: { from: line.from, insert: '\n' },
				range: EditorSelection.cursor(range.head + 1),
			};
		}
		return {
			changes: { from: range.from, to: range.to, insert: '\n\n' },
			range: EditorSelection.cursor(range.from + 2),
		};
	});
	dispatch(state.update(changes, { scrollIntoView: true, userEvent: 'input' }));
	return true;
};

// Enter on an empty list item leaves the list. By default, on the second
// item, lang-markdown's command would make the list non-tight instead.
const insertNewlineContinueMarkup = insertNewlineContinueMarkupCommand({ nonTightLists: false });

// continueMarkup continues lists and quotes, with lang-markdown's command.
// When Enter on an empty item leaves a list, it also starts a new paragraph.
export const continueMarkup: StateCommand = ({ state, dispatch }) => {
	let continued: EditorState | undefined;
	const handled = insertNewlineContinueMarkup({
		state,
		dispatch: (transaction) => {
			dispatch(transaction);
			continued = transaction.state;
		},
	});
	const separated = continued && separateFromBlockAbove(continued);
	if (continued && separated && !separated.changes.empty) {
		dispatch(continued.update(separated, { userEvent: 'input' }));
	}
	return handled;
};

// enter continues lists and quotes, starts paragraphs, closes a code block
// after its opening fence, and in code blocks inserts a line break that keeps
// the indentation of the line above.
export const enter: StateCommand = (target) =>
	continueMarkup(target) ||
	insertParagraph(target) ||
	closeFence(target) ||
	insertNewlineAndIndent(target);

const paragraphKeymap = Prec.high(
	keymap.of([
		{ key: 'Enter', run: enter },
		{ key: 'Shift-Enter', run: insertNewline },
	]),
);

const paragraphTheme = EditorView.theme({
	// The space between paragraphs in published posts
	'.cm-md-gap': { height: '10px' },
	'.cm-md-gap-edge': { height: '0' },
});

export const paragraphs = [paragraphLayout, paragraphKeymap, paragraphTheme];
