// An upload shows a placeholder where its image will go. The placeholder
// moves with the text around it, so typing during an upload doesn't misplace
// the image.

import { StateEffect, StateField } from '@codemirror/state';
import { Decoration, type DecorationSet, EditorView, WidgetType } from '@codemirror/view';
import { imageInsertion } from '../uploads';

class UploadingWidget extends WidgetType {
	toDOM() {
		const placeholder = document.createElement('span');
		placeholder.className = 'cm-md-uploading';
		placeholder.textContent = 'Uploading image…';
		return placeholder;
	}
}

const addUpload = StateEffect.define<{ id: number; position: number }>({
	map: (value, changes) => ({ id: value.id, position: changes.mapPos(value.position) }),
});
const removeUpload = StateEffect.define<number>();

const placeholders = StateField.define<DecorationSet>({
	create: () => Decoration.none,
	update(decorations, transaction) {
		decorations = decorations.map(transaction.changes);
		for (const effect of transaction.effects) {
			if (effect.is(addUpload)) {
				const { id, position } = effect.value;
				const placeholder = Decoration.widget({ widget: new UploadingWidget(), id, side: 1 });
				decorations = decorations.update({ add: [placeholder.range(position)] });
			} else if (effect.is(removeUpload)) {
				decorations = decorations.update({
					filter: (_from, _to, decoration) => decoration.spec.id !== effect.value,
				});
			}
		}
		return decorations;
	},
	provide: (field) => EditorView.decorations.from(field),
});

export const uploadPlaceholders = [
	placeholders,
	EditorView.theme({ '.cm-md-uploading': { color: 'var(--editor-mark)' } }),
];

let lastUploadID = 0;

// startUpload shows a placeholder at the cursor and returns its ID.
export const startUpload = (view: EditorView) => {
	const id = ++lastUploadID;
	view.dispatch({ effects: addUpload.of({ id, position: view.state.selection.main.head }) });
	return id;
};

// finishUpload replaces the placeholder with the image markup, in its own
// paragraph. Without markup, as when the upload failed, it only removes the
// placeholder.
export const finishUpload = (view: EditorView, id: number, markup?: string) => {
	let position: number | undefined;
	view.state.field(placeholders).between(0, view.state.doc.length, (from, _to, decoration) => {
		if (decoration.spec.id === id) position = from;
	});
	if (position === undefined || markup === undefined) {
		view.dispatch({ effects: removeUpload.of(id) });
		return;
	}

	const { doc, selection } = view.state;
	const insert = imageInsertion(
		doc.sliceString(Math.max(0, position - 2), position),
		doc.sliceString(position, Math.min(doc.length, position + 2)),
		markup,
	);
	// The cursor follows the image if it hasn't moved since the upload started
	const cursorStayed = selection.main.empty && selection.main.head === position;
	view.dispatch({
		changes: { from: position, insert },
		effects: removeUpload.of(id),
		selection: cursorStayed ? { anchor: position + insert.length } : undefined,
		scrollIntoView: cursorStayed,
		userEvent: 'input',
	});
};
