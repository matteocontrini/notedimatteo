import type { Plugin } from 'unified';
import type { Root, Element, Text } from 'hast';
import { visit } from 'unist-util-visit';

interface Options {
	storageBaseUrl: string;
}

/**
 * Rehype plugin to transform [img]path[/img] syntax into HTML img elements
 */
export const rehypeImgPlugin: Plugin<[Options?], Root> = (options) => {
	if (!options || !options.storageBaseUrl) {
		throw new Error('storageBaseUrl option is required for rehypeImgPlugin');
	}

	return (tree) => {
		visit(tree, 'text', (node: Text, index, parent) => {
			if (!parent || index === undefined) return;

			const value = node.value;

			const imgRegex =
				/\[img(?:\s+alt="([^"]*)")?(?:\s+width=(\d+))?(?:\s+height=(\d+))?(?:\s+link="([^"]*)")?](.+?)\[\/img]/g;

			// Check if this text node contains img tags
			if (!imgRegex.test(value)) return;

			// Reset regex for actual processing
			imgRegex.lastIndex = 0;

			const newNodes: (Element | Text)[] = [];
			let lastIndex = 0;
			let match;

			while ((match = imgRegex.exec(value)) !== null) {
				const [fullMatch, altText, width, height, linkUrl, imagePath] = match;
				const matchStart = match.index;
				const matchEnd = match.index + fullMatch.length;

				// Add text before the match (if any)
				if (matchStart > lastIndex) {
					const textBefore = value.slice(lastIndex, matchStart);
					if (textBefore) {
						newNodes.push({
							type: 'text',
							value: textBefore
						});
					}
				}

				// Create HTML img element with preview image and link wrapper
				const originalUrl = `${options!.storageBaseUrl}/${imagePath}`;

				// Generate preview URL by adding "_preview" before the file extension
				const previewUrl = originalUrl.replace(/(\.[^.]+)$/, '_preview$1');

				// Build img properties with dimensions if available
				const imgProperties: Record<string, string | number> = {
					src: previewUrl,
					alt: altText || '',
					loading: 'lazy'
				};

				// Add dimensions if provided (prevents layout shift)
				if (width) imgProperties.width = parseInt(width, 10);
				if (height) imgProperties.height = parseInt(height, 10);

				// Use custom link URL if provided, otherwise use original image URL
				const linkHref = linkUrl || originalUrl;

				// Create a figure wrapper with link and image
				newNodes.push({
					type: 'element',
					tagName: 'figure',
					properties: {},
					children: [
						{
							type: 'element',
							tagName: 'a',
							properties: {
								href: linkHref
							},
							children: [
								{
									type: 'element',
									tagName: 'img',
									properties: imgProperties,
									children: []
								}
							]
						}
					]
				});

				lastIndex = matchEnd;
			}

			// Add remaining text after the last match (if any)
			if (lastIndex < value.length) {
				const textAfter = value.slice(lastIndex);
				if (textAfter) {
					newNodes.push({
						type: 'text',
						value: textAfter
					});
				}
			}

			// Replace the current node with the new nodes
			if (newNodes.length > 0) {
				parent.children.splice(index, 1, ...newNodes);
			}
		});
	};
};
