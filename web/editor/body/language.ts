import { commonmarkLanguage } from '@codemirror/lang-markdown';
import { Language } from '@codemirror/language';
import {
	InlineContext,
	type MarkdownConfig,
	type MarkdownParser,
	Strikethrough,
} from '@lezer/markdown';

// Parse [text] as text, not as a link, when no ( or [ follows it. The renderer
// links it only if the post defines [text]: url, and the parser can't check that.
const BracketsWithoutLink: MarkdownConfig = {
	parseInline: [
		{
			name: 'BracketsWithoutLink',
			before: 'LinkEnd',
			parse(cx, next, start) {
				if (next !== 93 /* ] */ || /[([]/.test(cx.slice(start + 1, start + 2))) return -1;
				// Find the nearest [ or ![, which the standard parser would close
				const index = Math.max(
					cx.findOpeningDelimiter(InlineContext.linkStart) ?? -1,
					cx.findOpeningDelimiter(InlineContext.imageStart) ?? -1,
				);
				if (index < 0) return -1;
				// Remove the opening bracket so it stays text, then put back the
				// content after it with its markup
				for (const element of cx.takeContent(index)) cx.addElement(element);
				return start + 1;
			},
		},
	],
};

// markdownLanguage is CommonMark with strikethrough, like the renderer in
// internal/markdown. It's built the way lang-markdown's markdown() builds its
// language, without the HTML support that markdown() adds, which brings the
// HTML, CSS, and JavaScript parsers.
export const markdownLanguage = new Language(
	commonmarkLanguage.data,
	(commonmarkLanguage.parser as MarkdownParser).configure([Strikethrough, BracketsWithoutLink]),
	[],
	'markdown',
);
