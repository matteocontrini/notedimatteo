// The editor's look. The colors are CSS variables that web/assets/main.css sets
// for light and dark: CodeMirror adds its styles outside CSS cascade layers,
// so they beat Tailwind's rules.

import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { EditorView } from '@codemirror/view';
import { tags } from '@lezer/highlight';

// The text styles follow published posts, except headings, which posts show
// as plain text
const markdownStyle = HighlightStyle.define([
	{ tag: tags.strong, fontWeight: '700' },
	{ tag: tags.emphasis, fontStyle: 'italic' },
	{ tag: tags.strikethrough, textDecoration: 'line-through' },
	{ tag: tags.heading, fontWeight: '600' },
	{ tag: tags.link, color: 'var(--editor-link)', textDecoration: 'underline' },
	{ tag: tags.url, color: 'var(--editor-mark)', textDecoration: 'none' },
	{ tag: tags.monospace, fontFamily: 'var(--font-mono)' },
	// Last, so markers inside links are dimmed and not underlined too
	{ tag: tags.processingInstruction, color: 'var(--editor-mark)', textDecoration: 'none' },
]);

const editorTheme = EditorView.theme({
	'&': {
		border: '1px solid var(--editor-border)',
		backgroundColor: 'var(--editor-background)',
		color: 'var(--editor-text)',
	},
	'&.cm-focused': {
		outline: 'none',
		borderColor: 'var(--editor-border-focused)',
	},
	'.cm-scroller': {
		fontFamily: 'inherit',
		lineHeight: '1.5rem',
	},
	// About 16 lines, like the textarea it replaced
	'.cm-content': {
		minHeight: '24rem',
		padding: '0.5rem 0',
		caretColor: 'var(--editor-text)',
	},
	'.cm-line': {
		padding: '0 0.75rem',
	},
	'.cm-cursor, .cm-dropCursor': {
		borderLeftColor: 'var(--editor-text)',
	},
	'&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection':
		{
			backgroundColor: 'var(--editor-selection)',
		},
});

export const theme = [syntaxHighlighting(markdownStyle), editorTheme];
