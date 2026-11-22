// Helpers for the editor's tests, which run commands on editor states without
// a browser.

import { EditorSelection, EditorState, type Extension, type StateCommand } from '@codemirror/state';
import { markdownLanguage } from './language';

// stateOf creates a Markdown editor state from text with a | at each cursor.
export const stateOf = (text: string, extensions: Extension = []) => {
	const parts = text.split('|');
	const cursors: number[] = [];
	let position = 0;
	for (const part of parts.slice(0, -1)) {
		position += part.length;
		cursors.push(position);
	}
	return EditorState.create({
		doc: parts.join(''),
		selection: EditorSelection.create(cursors.map((cursor) => EditorSelection.cursor(cursor))),
		extensions: [markdownLanguage, EditorState.allowMultipleSelections.of(true), extensions],
	});
};

// textOf returns the state's text with a | at each cursor.
export const textOf = (state: EditorState) => {
	let text = state.doc.toString();
	for (const range of [...state.selection.ranges].reverse()) {
		text = text.slice(0, range.head) + '|' + text.slice(range.head);
	}
	return text;
};

// run applies command to the text with cursors, and returns the text and
// cursors after it, or null if the command didn't apply.
export const run = (command: StateCommand, text: string) => {
	let state = stateOf(text);
	const applied = command({ state, dispatch: (transaction) => (state = transaction.state) });
	return applied ? textOf(state) : null;
};
