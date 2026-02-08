import { gfmStrikethrough, type Options } from 'micromark-extension-gfm-strikethrough';
import { gfmStrikethroughFromMarkdown } from 'mdast-util-gfm-strikethrough';
import type { Processor } from 'unified';
import type { Root } from 'hast';

export default function remarkStrikethroughOnly(options: Options = {}) {
	// @ts-expect-error: TS is wrong about `this`.
	const self = this as Processor<Root>;
	const data = self.data();

	const micromarkExtensions = data.micromarkExtensions || (data.micromarkExtensions = []);
	const fromMarkdownExtensions = data.fromMarkdownExtensions || (data.fromMarkdownExtensions = []);

	// Parse ~~strike~~ (and optionally ~strike~ depending on options)
	micromarkExtensions.push(gfmStrikethrough(options));

	// Map from mdast "delete" nodes
	fromMarkdownExtensions.push(gfmStrikethroughFromMarkdown());
}
