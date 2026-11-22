import { commonmarkLanguage } from '@codemirror/lang-markdown';
import { Language } from '@codemirror/language';
import { type MarkdownParser, Strikethrough } from '@lezer/markdown';

// markdownLanguage is CommonMark with strikethrough, like the renderer in
// internal/markdown. It's built the way lang-markdown's markdown() builds its
// language, without the HTML support that markdown() adds, which brings the
// HTML, CSS, and JavaScript parsers.
export const markdownLanguage = new Language(
	commonmarkLanguage.data,
	(commonmarkLanguage.parser as MarkdownParser).configure([Strikethrough]),
	[],
	'markdown',
);
