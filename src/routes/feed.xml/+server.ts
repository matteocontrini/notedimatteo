import { db } from '$lib/server/db';
import { renderMarkdown, renderMarkdownWithPreview } from '$lib/server/markdown';
import { XMLBuilder } from 'fast-xml-parser';
import type { RequestHandler } from './$types';

const SITE_URL = 'https://notedimatteo.it';

const xmlBuilder = new XMLBuilder({
	ignoreAttributes: false,
	format: true,
	cdataPropName: '__cdata'
});

export const GET = (async () => {
	const posts = await db.post.findMany({
		where: { publishedAt: { not: null } },
		orderBy: { publishedAt: 'desc' },
		take: 50,
		include: {
			postsToTags: {
				select: {
					tag: { select: { name: true } }
				}
			}
		}
	});

	const items = await Promise.all(
		posts.map(async (post) => {
			const publishedAt = post.publishedAt!;
			const urlPath = `/${post.number}${post.slug ? `/${post.slug}` : ''}`;
			const permalink = `${SITE_URL}${urlPath}`;
			const title = post.title ? `#${post.number} / ${post.title}` : `#${post.number}`;
			const { html, preview } = await renderMarkdownWithPreview(post.body);
			const categories = [post.category, ...post.postsToTags.map((entry) => entry.tag.name)];

			return {
				title,
				link: permalink,
				guid: {
					'@_isPermaLink': 'true',
					'#text': permalink
				},
				pubDate: publishedAt.toUTCString(),
				category: categories,
				description: preview,
				'content:encoded': { __cdata: html }
			};
		})
	);

	const feed = {
		'?xml': {
			'@_version': '1.0',
			'@_encoding': 'UTF-8'
		},
		rss: {
			'@_version': '2.0',
			'@_xmlns:atom': 'http://www.w3.org/2005/Atom',
			'@_xmlns:content': 'http://purl.org/rss/1.0/modules/content/',
			channel: {
				title: 'Note di Matteo',
				link: SITE_URL,
				description: 'Note di Matteo',
				language: 'it-IT',
				'atom:link': {
					'@_href': SITE_URL + '/feed.xml',
					'@_rel': 'self',
					'@_type': 'application/rss+xml'
				},
				item: items
			}
		}
	};

	const rss = xmlBuilder.build(feed);

	return new Response(rss, {
		headers: {
			'Content-Type': 'application/xml; charset=utf-8',
			'Cache-Control': 'public, max-age=60'
		}
	});
}) satisfies RequestHandler;
