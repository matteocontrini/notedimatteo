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

			const imgRegex = /\[img([^\]]*)](.+?)\[\/img]/g;

			// Check if this text node contains img tags
			if (!imgRegex.test(value)) return;

			// Reset regex for actual processing
			imgRegex.lastIndex = 0;

			const newNodes: (Element | Text)[] = [];
			let lastIndex = 0;
			let match;

			while ((match = imgRegex.exec(value)) !== null) {
				const [fullMatch, attributesString, imagePath] = match;
				const matchStart = match.index;
				const matchEnd = match.index + fullMatch.length;

				// Parse attributes from the captured string
				const attributes = parseAttributes(attributesString);

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
					alt: attributes.alt || '',
					loading: 'lazy'
				};

				// Add dimensions if provided (prevents layout shift)
				if (attributes.width) imgProperties.width = parseInt(attributes.width, 10);
				if (attributes.height) imgProperties.height = parseInt(attributes.height, 10);

				// Use custom link URL if provided, otherwise use original image URL
				const linkHref = attributes.link || originalUrl;

				// Create the img element
				newNodes.push({
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

/**
 * Parse attributes from the attributes string, allowing any order
 */
function parseAttributes(attributesString: string): Record<string, string> {
	const attributes: Record<string, string> = {};

	// Remove leading/trailing whitespace
	const trimmed = attributesString.trim();

	if (!trimmed) {
		return attributes;
	}

	// Match attribute patterns: name="value" or name=value
	const attrRegex = /(\w+)=(?:"([^"]*)"|(\S+))/g;
	let attrMatch;

	while ((attrMatch = attrRegex.exec(trimmed)) !== null) {
		const [, name, quotedValue, unquotedValue] = attrMatch;
		attributes[name] = quotedValue || unquotedValue || '';
	}

	return attributes;
}
