// The post body editor: CodeMirror 6 with the Markdown parser. The text stays
// plain Markdown; the editor styles it like a published post, dims the syntax
// markers, and hides the blank lines between paragraphs. Each module adds one
// part of that.

import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { EditorSelection, EditorState } from '@codemirror/state';
import { EditorView, keymap } from '@codemirror/view';
import { blocks } from './blocks';
import { formattingShortcuts } from './formatting';
import { images } from './images';
import { markdownLanguage } from './language';
import { paragraphs } from './paragraphs';
import { theme } from './theme';
import { uploadPlaceholders } from './upload-placeholders';

export { finishUpload, startUpload } from './upload-placeholders';

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
				markdownLanguage,
				formattingShortcuts,
				paragraphs,
				history(),
				keymap.of([...defaultKeymap, ...historyKeymap]),
				blocks,
				images(options.imageBaseURL),
				uploadPlaceholders,
				theme,
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
			],
		}),
	});
};
