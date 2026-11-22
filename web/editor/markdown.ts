import { EditorSelection, type StateCommand, type Text } from '@codemirror/state';

// An Edit replaces the text from `from` to `to` with `insert`, then selects
// from `anchor` to `head`, in positions of the edited text.
type Edit = { from: number; to: number; insert: string; anchor: number; head: number };

// editSelections makes a command that applies edit to each selected range.
const editSelections =
	(edit: (doc: Text, from: number, to: number) => Edit): StateCommand =>
	({ state, dispatch }) => {
		const changes = state.changeByRange((range) => {
			const { from, to, insert, anchor, head } = edit(state.doc, range.from, range.to);
			return { changes: { from, to, insert }, range: EditorSelection.range(anchor, head) };
		});
		dispatch(state.update(changes, { scrollIntoView: true, userEvent: 'input' }));
		return true;
	};

// insertLink inserts []() with the cursor between the brackets. A selected
// URL becomes the link's address; other selected text becomes its text, with
// the cursor where the address goes.
export const insertLink = editSelections((doc, from, to) => {
	const text = doc.sliceString(from, to);
	if (text === '' || text.startsWith('http://') || text.startsWith('https://')) {
		return { from, to, insert: `[](${text})`, anchor: from + 1, head: from + 1 };
	}
	const cursor = from + text.length + 3;
	return { from, to, insert: `[${text}]()`, anchor: cursor, head: cursor };
});

// wrap puts marker on both sides of the selection and keeps the text
// selected. Without a selection, the cursor goes between the two markers.
const wrap = (marker: string) =>
	editSelections((doc, from, to) => {
		const text = doc.sliceString(from, to);
		const start = from + marker.length;
		return { from, to, insert: marker + text + marker, anchor: start, head: start + text.length };
	});

export const wrapBold = wrap('**');
export const wrapItalic = wrap('*');

// quoteLines inserts "> " at the cursor, or quotes every line the selection
// touches. Blank lines get ">" so the quote stays one block.
export const quoteLines = editSelections((doc, from, to) => {
	if (from === to) return { from, to, insert: '> ', anchor: from + 2, head: from + 2 };

	const start = doc.lineAt(from).from;
	const end = doc.lineAt(to).to;
	const lines = doc.sliceString(start, end).split('\n');
	const quoted = lines.map((line) => (line === '' ? '>' : `> ${line}`));
	const added = (line: string) => (line === '' ? 1 : 2);
	const totalAdded = lines.reduce((sum, line) => sum + added(line), 0);
	return {
		from: start,
		to: end,
		insert: quoted.join('\n'),
		anchor: from + added(lines[0]),
		head: to + totalAdded,
	};
});
