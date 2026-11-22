// Quotes and code blocks look like in published posts: line classes from the
// parse tree carry the styles.

import { syntaxTree } from '@codemirror/language';
import type { EditorState, Range } from '@codemirror/state';
import {
	Decoration,
	type DecorationSet,
	EditorView,
	ViewPlugin,
	type ViewUpdate,
	WidgetType,
} from '@codemirror/view';

// codeBlockNodes are the parse tree nodes of code blocks, fenced or indented
export const codeBlockNodes = new Set(['FencedCode', 'CodeBlock']);

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
	// Citations show quotation marks instead of the border. The category comes
	// from data-category on the editor's container, as on the site's <article>.
	"[data-category='cit'] & .cm-line.cm-md-quote": { borderLeft: 'none' },
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

export const blocks = [blockLines, blockTheme];
