// The post body editor: CodeMirror 6 with the Markdown parser. The text stays
// plain Markdown; the editor only styles it, and dims the syntax markers.

import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { commonmarkLanguage } from '@codemirror/lang-markdown';
import { HighlightStyle, Language, syntaxHighlighting } from '@codemirror/language';
import {
	EditorSelection,
	EditorState,
	Prec,
	StateEffect,
	StateField,
	type StateCommand,
	type Text,
} from '@codemirror/state';
import { Decoration, type DecorationSet, EditorView, WidgetType, keymap } from '@codemirror/view';
import { tags } from '@lezer/highlight';
import { type MarkdownParser, Strikethrough } from '@lezer/markdown';
import { insertLink, quoteLines, wrapBold, wrapItalic } from './markdown';
import { imageInsertion } from './uploads';

export type BodyEditorOptions = {
	parent: HTMLElement;
	doc: string;
	// imageBaseURL is the public address of uploaded images
	imageBaseURL: string;
	onChange: (doc: string) => void;
	onPasteImage: (file: File) => void;
};

export const createBodyEditor = (options: BodyEditorOptions) => {
	// CodeMirror turns \r\n into \n, so the length must come from the converted text
	const doc = options.doc.replace(/\r\n?/g, '\n');
	return new EditorView({
		parent: options.parent,
		state: EditorState.create({
			doc,
			// The cursor starts at the end, where new text usually goes
			selection: EditorSelection.cursor(doc.length),
			extensions: [
				shortcuts,
				history(),
				keymap.of([...defaultKeymap, ...historyKeymap]),
				markdownLanguage,
				syntaxHighlighting(markdownStyle),
				images(options.imageBaseURL),
				uploads,
				EditorView.lineWrapping,
				EditorView.contentAttributes.of({
					'aria-label': 'Body',
					spellcheck: 'true',
					autocorrect: 'on',
					autocapitalize: 'sentences',
					lang: 'it',
				}),
				EditorView.updateListener.of((update) => {
					if (update.docChanged) options.onChange(update.state.doc.toString());
				}),
				EditorView.domEventHandlers({
					paste(event) {
						const file = Array.from(event.clipboardData?.items ?? [])
							.find((item) => item.type.startsWith('image/'))
							?.getAsFile();
						if (!file) return false;
						event.preventDefault();
						options.onPasteImage(file);
						return true;
					},
				}),
				theme,
			],
		}),
	});
};

// markdownLanguage is CommonMark with strikethrough, like the renderer. It's
// built the way lang-markdown's markdown() builds it, without the HTML support
// that markdown() adds, which brings the HTML, CSS, and JavaScript parsers.
const markdownLanguage = new Language(
	commonmarkLanguage.data,
	(commonmarkLanguage.parser as MarkdownParser).configure([Strikethrough]),
	[],
	'markdown',
);

// The shortcuts compare event.key like the textarea's did, so every keyboard
// layout keeps the same keys. They run before the default keymap, which binds
// Mod-i to something else.
const shortcutCommands: [(event: KeyboardEvent) => boolean, StateCommand][] = [
	[(event) => event.key.toLowerCase() === 'b', wrapBold],
	[(event) => event.key.toLowerCase() === 'i', wrapItalic],
	[(event) => event.key.toLowerCase() === 'k', insertLink],
	[(event) => event.shiftKey && event.key === "'", quoteLines],
];

const shortcuts = Prec.highest(
	EditorView.domEventHandlers({
		keydown(event, view) {
			if (!(event.metaKey || event.ctrlKey)) return false;
			const command = shortcutCommands.find(([matches]) => matches(event))?.[1];
			if (!command) return false;
			event.preventDefault();
			return command(view);
		},
	}),
);

const markdownStyle = HighlightStyle.define([
	{ tag: tags.strong, fontWeight: '600' },
	{ tag: tags.emphasis, fontStyle: 'italic' },
	{ tag: tags.strikethrough, textDecoration: 'line-through' },
	{ tag: tags.heading, fontWeight: '600' },
	{ tag: tags.link, color: 'var(--editor-link)' },
	{ tag: tags.url, color: 'var(--editor-mark)' },
	{ tag: tags.quote, color: 'var(--editor-quote)' },
	{ tag: tags.monospace, fontFamily: 'var(--font-mono)', fontSize: '0.875em' },
	// Last, so markers inside links and quotes are dimmed too
	{ tag: tags.processingInstruction, color: 'var(--editor-mark)' },
]);

const theme = EditorView.theme({
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
	// About 16 lines, like the textarea it replaces
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
	'.cm-md-image': {
		padding: '0.25rem 0.75rem 0.5rem',
	},
	'.cm-md-image img': {
		maxWidth: 'min(100%, 24rem)',
		height: 'auto',
	},
	// The parser reads [img] tags as link brackets: this also overrides the link color inside them
	'.cm-md-image-tag, .cm-md-image-tag *': {
		color: 'var(--editor-mark)',
	},
	'.cm-md-uploading': {
		color: 'var(--editor-mark)',
	},
});

// Images

// The same pattern as the renderer's in internal/markdown/image.go
const imagePattern = /\[img([^\]]*)\]([^[\]\n]+?)\[\/img\]/g;
const attributePattern = /(\w+)=(?:"([^"]*)"|(\S+))/g;

// previewURL returns the address of an uploaded image's preview, which has
// _preview before the extension, like the renderer's.
export const previewURL = (baseURL: string, path: string) => {
	const dot = path.lastIndexOf('.');
	const extension = dot > path.lastIndexOf('/') ? path.slice(dot) : '';
	return `${baseURL}/${path.slice(0, path.length - extension.length)}_preview${extension}`;
};

class ImageWidget extends WidgetType {
	constructor(
		readonly src: string,
		readonly width: string | undefined,
		readonly height: string | undefined,
	) {
		super();
	}

	eq(other: ImageWidget) {
		return other.src === this.src && other.width === this.width && other.height === this.height;
	}

	toDOM() {
		const wrapper = document.createElement('div');
		wrapper.className = 'cm-md-image';
		const image = document.createElement('img');
		image.src = this.src;
		image.alt = '';
		// The size lets the browser keep the space before the image loads
		if (this.width) image.width = Number(this.width);
		if (this.height) image.height = Number(this.height);
		wrapper.append(image);
		return wrapper;
	}
}

const imageTag = Decoration.mark({ class: 'cm-md-image-tag' });

// imageDecorations shows each [img] markup's preview below its line, and dims
// its [img] and [/img] tags like the other markers.
export const imageDecorations = (doc: Text, baseURL: string) => {
	const decorations = [];
	for (let number = 1; number <= doc.lines; number++) {
		const line = doc.line(number);
		for (const match of line.text.matchAll(imagePattern)) {
			const [markup, attributeString, path] = match;
			const start = line.from + match.index;
			const end = start + markup.length;
			decorations.push(
				imageTag.range(start, start + markup.length - path.length - '[/img]'.length),
				imageTag.range(end - '[/img]'.length, end),
			);

			const attributes = new Map(
				Array.from(attributeString.matchAll(attributePattern), ([, name, quoted, plain]) => [
					name,
					quoted || plain,
				]),
			);
			const widget = new ImageWidget(
				previewURL(baseURL, path),
				integer(attributes.get('width')),
				integer(attributes.get('height')),
			);
			decorations.push(Decoration.widget({ widget, block: true, side: 1 }).range(line.to));
		}
	}
	return Decoration.set(decorations, true);
};

const integer = (value: string | undefined) =>
	value !== undefined && /^\d+$/.test(value) ? value : undefined;

// Block widgets must come from a state field, not a view plugin
const images = (baseURL: string) =>
	StateField.define<DecorationSet>({
		create: (state) => imageDecorations(state.doc, baseURL),
		update: (decorations, transaction) =>
			transaction.docChanged ? imageDecorations(transaction.state.doc, baseURL) : decorations,
		provide: (field) => EditorView.decorations.from(field),
	});

// Uploads

// An upload shows a placeholder where its image will go. The placeholder
// moves with the text around it, so typing during an upload doesn't misplace
// the image.

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

const uploads = StateField.define<DecorationSet>({
	create: () => Decoration.none,
	update(placeholders, transaction) {
		placeholders = placeholders.map(transaction.changes);
		for (const effect of transaction.effects) {
			if (effect.is(addUpload)) {
				const { id, position } = effect.value;
				const placeholder = Decoration.widget({ widget: new UploadingWidget(), id, side: 1 });
				placeholders = placeholders.update({ add: [placeholder.range(position)] });
			} else if (effect.is(removeUpload)) {
				placeholders = placeholders.update({
					filter: (_from, _to, decoration) => decoration.spec.id !== effect.value,
				});
			}
		}
		return placeholders;
	},
	provide: (field) => EditorView.decorations.from(field),
});

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
	view.state.field(uploads).between(0, view.state.doc.length, (from, _to, decoration) => {
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
