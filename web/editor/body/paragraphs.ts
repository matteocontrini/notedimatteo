// Paragraphs. In Markdown, blank lines separate paragraphs and other blocks.
// The editor hides them and shows the space between blocks instead, as in
// published posts, and Enter inserts the blank line that starts a new
// paragraph. The text stays the same Markdown.

import { insertNewline } from '@codemirror/commands';
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
import { codeBlockNodes } from './blocks';

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

// insideCodeBlock reports whether pos is inside a code block, where Enter
// inserts a plain line break. After a closing fence, Enter starts a
// paragraph. A fence without its closing fence yet, as while typing it, runs
// to the end of the text.
const insideCodeBlock = (state: EditorState, pos: number) => {
	for (let node = syntaxTree(state).resolveInner(pos, -1); node.parent; node = node.parent) {
		if (node.name === 'FencedCode') {
			const fences = node.getChildren('CodeMark');
			const closing = fences.length > 1 ? fences[fences.length - 1] : undefined;
			return pos > node.from && (!closing || pos < closing.from);
		}
		if (node.name === 'CodeBlock') return node.from < pos && pos < node.to;
	}
	return false;
};

// A BlankRun is a run of blank lines to hide, by line number.
export type BlankRun = { first: number; last: number };

// hiddenBlankLines returns the runs of blank lines that the editor hides. It
// shows blank lines inside code blocks, and blank lines with a cursor, so the
// cursor never sits in hidden text.
export const hiddenBlankLines = (state: EditorState): BlankRun[] => {
	const shown = codeBlockLines(state);
	for (const range of state.selection.ranges) shown.add(state.doc.lineAt(range.head).number);

	const runs: BlankRun[] = [];
	let run: BlankRun | undefined;
	for (let number = 1; number <= state.doc.lines; number++) {
		if (!shown.has(number) && isBlank(state.doc.line(number).text)) {
			if (run) run.last = number;
			else runs.push((run = { first: number, last: number }));
		} else {
			run = undefined;
		}
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

// separateFromBlockAbove makes a blank line with the cursor, right below a
// block, the start of a new paragraph: it adds the blank line between them.
// Without it, text typed there would continue the block above, like a list's
// last item.
const separateFromBlockAbove = (state: EditorState) =>
	state.changeByRange((range) => {
		const line = state.doc.lineAt(range.head);
		const above = line.number > 1 ? state.doc.line(line.number - 1) : undefined;
		if (!range.empty || !isBlank(line.text) || !above || isBlank(above.text)) return { range };
		return {
			changes: { from: line.from, insert: '\n' },
			range: EditorSelection.cursor(range.head + 1),
		};
	});

// insertParagraph ends the paragraph at the cursor and starts a new one,
// inserting the blank line between them. On a blank line it only adds the
// blank line that separates it from the block above, if missing. In code
// blocks it does nothing, so Enter inserts a plain line break.
export const insertParagraph: StateCommand = ({ state, dispatch }) => {
	if (state.selection.ranges.some((range) => insideCodeBlock(state, range.head))) return false;

	const onBlankLine = state.selection.ranges.every(
		(range) => range.empty && isBlank(state.doc.lineAt(range.head).text),
	);
	const changes = onBlankLine
		? separateFromBlockAbove(state)
		: state.changeByRange((range) => ({
				changes: { from: range.from, to: range.to, insert: '\n\n' },
				range: EditorSelection.cursor(range.from + 2),
			}));
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

const paragraphKeymap = Prec.high(
	keymap.of([
		{ key: 'Enter', run: (view) => continueMarkup(view) || insertParagraph(view) },
		{ key: 'Shift-Enter', run: insertNewline },
	]),
);

const paragraphTheme = EditorView.theme({
	// The space between paragraphs in published posts
	'.cm-md-gap': { height: '10px' },
	'.cm-md-gap-edge': { height: '0' },
});

export const paragraphs = [paragraphLayout, paragraphKeymap, paragraphTheme];
