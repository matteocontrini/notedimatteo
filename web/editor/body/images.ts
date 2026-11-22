// Each [img] markup shows its uploaded image below its line, as wide as the
// text, like in published posts.

import { type Range, StateField, type Text } from '@codemirror/state';
import { Decoration, type DecorationSet, EditorView, WidgetType } from '@codemirror/view';

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
	const decorations: Range<Decoration>[] = [];
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

const imageTheme = EditorView.theme({
	// The gap between paragraphs adds the space below
	'.cm-md-image': {
		padding: '0.25rem 0.75rem 0',
	},
	'.cm-md-image img': {
		display: 'block',
		maxWidth: '100%',
		height: 'auto',
	},
	// The parser reads [img] tags as link brackets: this also overrides the
	// link style inside them
	'.cm-md-image-tag, .cm-md-image-tag *': {
		color: 'var(--editor-mark)',
		textDecoration: 'none',
	},
});

// images takes the public address of uploaded images. Block widgets must come
// from a state field, not a view plugin.
export const images = (baseURL: string) => [
	StateField.define<DecorationSet>({
		create: (state) => imageDecorations(state.doc, baseURL),
		update: (decorations, transaction) =>
			transaction.docChanged ? imageDecorations(transaction.state.doc, baseURL) : decorations,
		provide: (field) => EditorView.decorations.from(field),
	}),
	imageTheme,
];
