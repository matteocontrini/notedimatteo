// Quotes and code blocks look like in published posts: line classes from the
// parse tree carry the styles. In code blocks, Tab and Shift+Tab indent.

import { indentLess, indentMore } from '@codemirror/commands';
import { indentUnit, syntaxTree } from '@codemirror/language';
import {
	EditorSelection,
	type EditorState,
	type Range,
	type StateCommand,
} from '@codemirror/state';
import {
	Decoration,
	type DecorationSet,
	EditorView,
	ViewPlugin,
	type ViewUpdate,
	WidgetType,
	keymap,
} from '@codemirror/view';

// codeBlockNodes are the parse tree nodes of code blocks, fenced or indented
export const codeBlockNodes = new Set(['FencedCode', 'CodeBlock']);

// Return the quote markers at the start of a line, like "> " or "> > ", or ""
// outside a quote.
export const quoteMarkers = (text: string) => /^(?: {0,3}> ?)+/.exec(text)?.[0] ?? '';

// Return the quote markers of pos's line, or "" when pos comes before them.
export const markersBefore = (state: EditorState, pos: number) => {
	const line = state.doc.lineAt(pos);
	const markers = quoteMarkers(line.text);
	return pos - line.from >= markers.length ? markers : '';
};

// Return the start of the code block whose code holds pos, and whether the
// block is fenced. After a closing fence, pos isn't in the code. A fence
// without its closing fence yet, as while typing it, runs to the end of the
// text.
const codeBlockAt = (state: EditorState, pos: number) => {
	for (let node = syntaxTree(state).resolveInner(pos, -1); node.parent; node = node.parent) {
		if (node.name === 'FencedCode') {
			const fences = node.getChildren('CodeMark');
			const closing = fences.length > 1 ? fences[fences.length - 1] : undefined;
			const inside = pos > node.from && (!closing || pos < closing.from);
			return inside ? { from: node.from, fenced: true } : undefined;
		}
		if (node.name === 'CodeBlock') {
			return node.from < pos && pos < node.to ? { from: node.from, fenced: false } : undefined;
		}
	}
	return undefined;
};

// insideCodeBlock reports whether pos is inside a code block's code.
export const insideCodeBlock = (state: EditorState, pos: number) =>
	codeBlockAt(state, pos) !== undefined;

// Return the text that each line of the code block at from needs before its
// code, like "> " in a quote. Turn list markers into spaces, because only the
// block's first line has them.
const codeLinePrefix = (state: EditorState, from: number, fenced: boolean) => {
	const first = state.doc.lineAt(from);
	const before = fenced ? first.text.slice(0, from - first.from) : quoteMarkers(first.text);
	return before.replace(/[^>\s]/g, ' ');
};

// closeFence, at the end of an opening fence without its closing fence,
// adds the closing fence and puts the cursor on the empty line between them.
// In a quote or a list, both lines get the prefix that the fence line has.
// When text follows, a blank line keeps it out of the code block.
export const closeFence: StateCommand = ({ state, dispatch }) => {
	const { doc, selection } = state;
	if (selection.ranges.length > 1 || !selection.main.empty) return false;
	const pos = selection.main.head;
	const line = doc.lineAt(pos);
	if (pos !== line.to) return false;

	const fence = syntaxTree(state).resolveInner(pos, -1);
	const block = fence.name === 'FencedCode' ? fence : fence.parent;
	if (block?.name !== 'FencedCode' || doc.lineAt(block.from).number !== line.number) return false;
	const fences = block.getChildren('CodeMark');
	if (fences.length !== 1) return false;

	const marker = doc.sliceString(fences[0].from, fences[0].to);
	const prefix = codeLinePrefix(state, block.from, true);
	const below = line.number < doc.lines ? doc.line(line.number + 1).text : '';
	const textBelow = /\S/.test(below.slice(quoteMarkers(below).length));
	const insert = `\n${prefix}\n${prefix}${marker}${textBelow ? `\n${prefix.trimEnd()}` : ''}`;
	dispatch(
		state.update({
			changes: { from: pos, insert },
			selection: EditorSelection.cursor(pos + 1 + prefix.length),
			scrollIntoView: true,
			userEvent: 'input',
		}),
	);
	return true;
};

// Start a new line in a code block inside a quote or a list. The new line gets
// the prefix that each line of the block needs, like "> ", and the indentation
// of the line above. Leave other code blocks to insertNewlineAndIndent.
export const continueCodeLine: StateCommand = ({ state, dispatch }) => {
	let applies = true;
	const changes = state.changeByRange((range) => {
		const block = codeBlockAt(state, range.from);
		const prefix = block ? codeLinePrefix(state, block.from, block.fenced) : '';
		const line = state.doc.lineAt(range.from);
		const column = range.from - line.from;
		if (!prefix) {
			applies = false;
			return { range };
		}
		// Before the prefix, as after Home, add an empty code line above
		if (range.empty && column < Math.min(prefix.length, line.text.trimEnd().length)) {
			const above = `${prefix.trimEnd()}\n`;
			return {
				changes: { from: line.from, insert: above },
				range: EditorSelection.cursor(range.head + above.length),
			};
		}
		const indent = /^\s*/.exec(line.text.slice(prefix.length, column))![0];
		const insert = `\n${prefix}${indent}`;
		return {
			changes: { from: range.from, to: range.to, insert },
			range: EditorSelection.cursor(range.from + insert.length),
		};
	});
	if (!applies) return false;
	dispatch(state.update(changes, { scrollIntoView: true, userEvent: 'input' }));
	return true;
};

const inCode = (state: EditorState) =>
	state.selection.ranges.every((range) => insideCodeBlock(state, range.head));

// indentCode inserts an indentation step at the cursor, or indents the
// selected lines. Outside code blocks it does nothing, so Tab moves the focus
// to the next field as usual.
export const indentCode: StateCommand = ({ state, dispatch }) => {
	if (!inCode(state)) return false;
	if (state.selection.ranges.some((range) => !range.empty)) return indentMore({ state, dispatch });
	const step = state.replaceSelection(state.facet(indentUnit));
	dispatch(state.update(step, { scrollIntoView: true, userEvent: 'input.indent' }));
	return true;
};

// dedentCode removes an indentation step from the lines with a cursor or a
// selection.
export const dedentCode: StateCommand = (target) => inCode(target.state) && indentLess(target);

const codeIndentation = [
	indentUnit.of('    '),
	keymap.of([
		{ key: 'Tab', run: indentCode },
		{ key: 'Shift-Tab', run: dedentCode },
	]),
];

// The closing quotation mark of a citation, which CSS hides elsewhere
class QuoteCloseWidget extends WidgetType {
	toDOM() {
		const mark = document.createElement('span');
		mark.className = 'cm-md-quote-close';
		return mark;
	}
}

const quoteClose = Decoration.widget({ widget: new QuoteCloseWidget(), side: 1 });

// blockDecorations gives each line of a quote or code block a class, plus one
// for the block's first and last lines. Blocks inside a quote keep the
// quote's style.
export const blockDecorations = (state: EditorState) => {
	const decorations: Range<Decoration>[] = [];
	syntaxTree(state).iterate({
		enter(node) {
			const block =
				node.name === 'Blockquote' ? 'quote' : codeBlockNodes.has(node.name) ? 'code' : null;
			if (!block) return;

			const first = state.doc.lineAt(node.from);
			const last = state.doc.lineAt(node.to);
			for (let number = first.number; number <= last.number; number++) {
				const classes = [`cm-md-${block}`];
				if (number === first.number) classes.push(`cm-md-${block}-first`);
				if (number === last.number) classes.push(`cm-md-${block}-last`);
				const line = state.doc.line(number);
				decorations.push(Decoration.line({ class: classes.join(' ') }).range(line.from));
			}
			if (block === 'quote') decorations.push(quoteClose.range(last.to));
			return false;
		},
	});
	return Decoration.set(decorations, true);
};

const blockLines = ViewPlugin.fromClass(
	class {
		decorations: DecorationSet;

		constructor(view: EditorView) {
			this.decorations = blockDecorations(view.state);
		}

		update(update: ViewUpdate) {
			// The parser can finish after the change that started it
			if (update.docChanged || syntaxTree(update.state) !== syntaxTree(update.startState)) {
				this.decorations = blockDecorations(update.state);
			}
		}
	},
	{ decorations: (plugin) => plugin.decorations },
);

// Lines take horizontal margins only: CodeMirror measures line heights, which
// vertical margins would break.
const blockTheme = EditorView.theme({
	'.cm-line.cm-md-quote': {
		marginLeft: '0.75rem',
		paddingLeft: '1rem',
		borderLeft: '6px solid var(--editor-quote-border)',
	},
	'.cm-md-quote-first': { paddingTop: '0.25rem' },
	'.cm-md-quote-last': { paddingBottom: '0.25rem' },
	'.cm-md-quote-close': { display: 'none' },
	// Continue the quote's border across the gap between its paragraphs.
	// paragraphs.ts adds the gap.
	'.cm-md-gap-quote': {
		marginLeft: '0.75rem',
		borderLeft: '6px solid var(--editor-quote-border)',
	},
	// Citations show quotation marks instead of the border. The category comes
	// from data-category on the editor's container, as on the site's <article>.
	"[data-category='cit'] & .cm-line.cm-md-quote, [data-category='cit'] & .cm-md-gap-quote": {
		borderLeft: 'none',
	},
	"[data-category='cit'] & .cm-md-quote-first": {
		paddingTop: '0.5rem',
		textIndent: '28px',
		backgroundImage: 'var(--quote-open-mark)',
		backgroundSize: '23px 23px',
		backgroundRepeat: 'no-repeat',
		backgroundPosition: '14px 4px',
	},
	"[data-category='cit'] & .cm-md-quote-last": { paddingBottom: '0.5rem' },
	"[data-category='cit'] & .cm-md-quote-close": {
		display: 'inline-block',
		width: '25px',
		height: '1.5rem',
		verticalAlign: 'top',
		backgroundImage: 'var(--quote-close-mark)',
		backgroundSize: '23px 23px',
		backgroundRepeat: 'no-repeat',
		backgroundPosition: 'bottom 2px right',
	},
	'.cm-line.cm-md-code': {
		margin: '0 0.75rem',
		backgroundColor: 'var(--editor-code)',
		// Tailwind's text-sm, like posts' code blocks
		fontSize: '0.875rem',
		lineHeight: '1.25rem',
	},
	'.cm-md-code-first': {
		paddingTop: '0.5rem',
		borderTopLeftRadius: '0.25rem',
		borderTopRightRadius: '0.25rem',
	},
	'.cm-md-code-last': {
		paddingBottom: '0.5rem',
		borderBottomLeftRadius: '0.25rem',
		borderBottomRightRadius: '0.25rem',
	},
});

export const blocks = [blockLines, blockTheme, codeIndentation];
