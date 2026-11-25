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
//
// In a quote, a line with only the markers, like ">", separates paragraphs.
// The editor hides these lines like blank lines, and Enter in a quote inserts
// them.

import { insertNewline, insertNewlineAndIndent } from '@codemirror/commands';
import { insertNewlineContinueMarkupCommand } from '@codemirror/lang-markdown';
import { syntaxTree } from '@codemirror/language';
import {
	EditorSelection,
	type EditorState,
	Prec,
	type Range,
	RangeSet,
	type SelectionRange,
	type StateCommand,
	StateField,
} from '@codemirror/state';
import { Decoration, type DecorationSet, EditorView, WidgetType, keymap } from '@codemirror/view';
import {
	closeFence,
	codeBlockNodes,
	continueCodeLine,
	insideCodeBlock,
	markersBefore,
	quoteMarkers,
} from './blocks';

const isBlank = (text: string) => !/\S/.test(text);

// LineShape holds a line's quote markers, how many quotes the line is in, and
// whether the rest of the line is blank.
type LineShape = { markers: string; depth: number; blank: boolean };

const shapeOf = (text: string): LineShape => {
	const markers = quoteMarkers(text);
	return {
		markers,
		depth: markers.split('>').length - 1,
		blank: isBlank(text.slice(markers.length)),
	};
};

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
// In a quote, lines with only the markers count as blank. A run ends where the
// quote depth changes. A line next to a run counts as a block when it's in at
// least as many quotes as the run. Blank lines inside code blocks are part of
// the code and always show. A blank line with a cursor shows too, so the
// cursor never sits in hidden text; Enter never puts it there, but text like
// "A\n" at the end of a body can.
export const hiddenBlankLines = (state: EditorState): BlankRun[] => {
	const { doc } = state;
	const code = codeBlockLines(state);
	const cursors = new Set(state.selection.ranges.map((range) => doc.lineAt(range.head).number));
	const shapes = Array.from({ length: doc.lines }, (_, index) => shapeOf(doc.line(index + 1).text));
	const depth = (number: number) => shapes[number - 1].depth;
	const blank = (number: number) => !code.has(number) && shapes[number - 1].blank;

	const hidden: number[] = [];
	for (let first = 1; first <= doc.lines; first++) {
		if (!blank(first)) continue;
		let last = first;
		while (last < doc.lines && blank(last + 1) && depth(last + 1) === depth(first)) last++;
		const blockBefore = first > 1 && depth(first - 1) >= depth(first);
		const blockAfter = last < doc.lines && depth(last + 1) >= depth(first);
		hiddenLinesOfRun(last - first + 1, blockBefore, blockAfter).forEach((hide, index) => {
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
// In a quote, blocks.ts draws the quote's border on it.
class GapWidget extends WidgetType {
	constructor(
		readonly between: boolean,
		readonly quote: boolean,
	) {
		super();
	}

	eq(other: GapWidget) {
		return other.between === this.between && other.quote === this.quote;
	}

	get estimatedHeight() {
		return this.between ? 10 : 0;
	}

	toDOM() {
		const gap = document.createElement('div');
		gap.className = [
			'cm-md-gap',
			!this.between && 'cm-md-gap-edge',
			this.quote && 'cm-md-gap-quote',
		]
			.filter(Boolean)
			.join(' ');
		return gap;
	}
}

const gapBetween = new GapWidget(true, false);
const gapInQuote = new GapWidget(true, true);
const gapAtEdge = new GapWidget(false, false);
const atomic = Decoration.mark({});

type Layout = { gaps: DecorationSet; atomic: RangeSet<Decoration> };

// layout replaces each run of hidden blank lines with a gap. The cursor skips
// each run and the line breaks around it as one unit, so the arrows go from
// the end of a paragraph to the start of the next, and Backspace and Delete
// there join the two paragraphs. In a quote, the unit also covers the next
// paragraph's markers, so the cursor lands at the start of its text.
const layout = (state: EditorState): Layout => {
	const { doc } = state;
	const gaps: Range<Decoration>[] = [];
	const skipped: Range<Decoration>[] = [];
	for (const { first, last } of hiddenBlankLines(state)) {
		const between = first > 1 && last < doc.lines;
		const quote =
			between &&
			quoteMarkers(doc.line(first).text) !== '' &&
			quoteMarkers(doc.line(last).text) !== '';
		const widget = quote ? gapInQuote : between ? gapBetween : gapAtEdge;
		gaps.push(
			Decoration.replace({ widget, block: true }).range(doc.line(first).from, doc.line(last).to),
		);
		const next = last < doc.lines ? doc.line(last + 1) : undefined;
		skipped.push(
			atomic.range(
				first > 1 ? doc.line(first - 1).to : 0,
				next ? next.from + (quote ? quoteMarkers(next.text).length : 0) : doc.length,
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

// Add the missing blank line between a blank line with the cursor and the
// block right above it, so text typed there starts a new paragraph. In a
// quote, the added line gets the quote's markers. Return null when the line
// isn't right below a block.
const separation = (state: EditorState, range: SelectionRange) => {
	const line = state.doc.lineAt(range.head);
	const shape = shapeOf(line.text);
	if (!range.empty || !shape.blank || line.number === 1) return null;
	const above = shapeOf(state.doc.line(line.number - 1).text);
	if (above.blank && above.depth === shape.depth) return null;
	const insert = `${shape.markers.trimEnd()}\n`;
	return {
		changes: { from: line.from, insert },
		range: EditorSelection.cursor(range.head + insert.length),
	};
};

// Start a new paragraph on each blank line with a cursor right below a block.
const separateFromBlockAbove = (state: EditorState) =>
	state.changeByRange((range) => separation(state, range) ?? { range });

// End the paragraph at the cursor and start a new one below it, with a blank
// line between them. In a quote, both new lines get the quote's markers. At
// the start of a paragraph, move the paragraph down with the cursor, leaving an
// empty one above. On a blank line right below a block, only add the missing
// blank line. In code blocks, do nothing, so Enter inserts a plain line break.
export const insertParagraph: StateCommand = ({ state, dispatch }) => {
	if (state.selection.ranges.some((range) => insideCodeBlock(state, range.head))) return false;

	const changes = state.changeByRange((range) => {
		const separated = separation(state, range);
		if (separated) return separated;
		const markers = markersBefore(state, range.from);
		const insert = `\n${markers.trimEnd()}\n${markers}`;
		return {
			changes: { from: range.from, to: range.to, insert },
			range: EditorSelection.cursor(range.from + insert.length),
		};
	});
	dispatch(state.update(changes, { scrollIntoView: true, userEvent: 'input' }));
	return true;
};

// Report whether the innermost list item or quote around pos is a quote. Code
// blocks inside a quote don't count.
const inQuoteParagraph = (state: EditorState, pos: number) => {
	if (insideCodeBlock(state, pos)) return false;
	for (let node = syntaxTree(state).resolveInner(pos, -1); node.parent; node = node.parent) {
		if (node.name === 'ListItem') return false;
		if (node.name === 'Blockquote') return true;
	}
	return false;
};

// Report whether the cursor is on an empty paragraph at the end of a quote. That
// paragraph is a line with only markers, right below another such line.
const onEmptyQuoteParagraph = (state: EditorState) => {
	const { main } = state.selection;
	const line = state.doc.lineAt(main.head);
	const shape = shapeOf(line.text);
	if (!main.empty || !shape.blank || shape.depth === 0 || line.number === 1) return false;
	const above = shapeOf(state.doc.line(line.number - 1).text);
	return above.blank && above.depth === shape.depth;
};

// Enter on an empty list item leaves the list. By default, on the second
// item, lang-markdown's command would make the list non-tight instead.
const insertNewlineContinueMarkup = insertNewlineContinueMarkupCommand({ nonTightLists: false });

// Continue lists and quotes with lang-markdown's command. When Enter on an
// empty item leaves a list, also start a new paragraph, inside the quote when
// the list is in one.
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

// Start a paragraph in a quote. On an empty paragraph at the end of the quote,
// leave the quote instead, like Enter on an empty list item. Leave lists inside
// a quote to continueMarkup.
export const quoteParagraph: StateCommand = (target) => {
	const { state } = target;
	if (!state.selection.ranges.every((range) => inQuoteParagraph(state, range.head))) return false;
	return onEmptyQuoteParagraph(state) ? continueMarkup(target) : insertParagraph(target);
};

// Join two paragraphs on Backspace or Delete when the cursor sits inside the
// unit it skips between them. Home puts it there, before a quote's markers.
// Elsewhere, leave the keys to the usual commands.
export const joinParagraphs: StateCommand = ({ state, dispatch }) => {
	const { main } = state.selection;
	const skipped = state.field(paragraphLayout, false)?.atomic;
	if (!skipped || state.selection.ranges.length > 1 || !main.empty) return false;
	let unit: { from: number; to: number } | undefined;
	skipped.between(main.head, main.head, (from, to) => {
		if (from < main.head && main.head < to) unit = { from, to };
	});
	if (!unit) return false;
	dispatch(
		state.update({
			changes: unit,
			selection: EditorSelection.cursor(unit.from),
			scrollIntoView: true,
			userEvent: 'delete',
		}),
	);
	return true;
};

// Handle Enter: start paragraphs, in quotes too, and continue lists. After a
// code block's opening fence, add the closing fence. In code blocks, insert a
// line break that keeps the indentation of the line above, and the quote
// markers in a quote.
export const enter: StateCommand = (target) =>
	quoteParagraph(target) ||
	closeFence(target) ||
	continueCodeLine(target) ||
	continueMarkup(target) ||
	insertParagraph(target) ||
	insertNewlineAndIndent(target);

const paragraphKeymap = Prec.high(
	keymap.of([
		{ key: 'Enter', run: enter },
		{ key: 'Shift-Enter', run: insertNewline },
		{ key: 'Backspace', run: joinParagraphs },
		{ key: 'Delete', run: joinParagraphs },
	]),
);

const paragraphTheme = EditorView.theme({
	// The space between paragraphs in published posts
	'.cm-md-gap': { height: '10px' },
	'.cm-md-gap-edge': { height: '0' },
});

export const paragraphs = [paragraphLayout, paragraphKeymap, paragraphTheme];
