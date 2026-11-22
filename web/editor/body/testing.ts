// Helpers for the editor's tests, which run commands on editor states without
// a browser. Texts mark each cursor with | and each selection with « and ».

import {
	EditorSelection,
	EditorState,
	type Extension,
	type SelectionRange,
	type StateCommand,
} from '@codemirror/state';
import { markdownLanguage } from './language';

// stateOf creates a Markdown editor state from text with cursors and
// selections marked.
export const stateOf = (text: string, extensions: Extension = []) => {
	let doc = '';
	let selectionStart: number | undefined;
	const ranges: SelectionRange[] = [];
	for (const char of text) {
		if (char === '|') ranges.push(EditorSelection.cursor(doc.length));
		else if (char === '«') selectionStart = doc.length;
		else if (char === '»') ranges.push(EditorSelection.range(selectionStart!, doc.length));
		else doc += char;
	}
	return EditorState.create({
		doc,
		selection: EditorSelection.create(ranges),
		extensions: [markdownLanguage, EditorState.allowMultipleSelections.of(true), extensions],
	});
};

// textOf returns the state's text with cursors and selections marked.
export const textOf = (state: EditorState) => {
	let text = state.doc.toString();
	for (const range of [...state.selection.ranges].reverse()) {
		text = range.empty
			? text.slice(0, range.head) + '|' + text.slice(range.head)
			: text.slice(0, range.from) +
				'«' +
				text.slice(range.from, range.to) +
				'»' +
				text.slice(range.to);
	}
	return text;
};

// run applies command to the marked text, and returns the text and its marks
// after it, or null if the command didn't apply.
export const run = (command: StateCommand, text: string, extensions: Extension = []) => {
	let state = stateOf(text, extensions);
	const applied = command({ state, dispatch: (transaction) => (state = transaction.state) });
	return applied ? textOf(state) : null;
};
