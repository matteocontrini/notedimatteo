<script lang="ts">
	import { page } from '$app/state';

	const fallback = {
		title: 'Note di Matteo',
		description: 'Note di Matteo',
		ogType: 'website',
		// ogImage: ''
	};

	let seo = $derived(page.data.seo ?? {});

	let title = $derived(seo.title || fallback.title);
	let description = $derived(seo.description || fallback.description);
	let ogTitle = $derived(seo.ogTitle || seo.title || fallback.title);
	let ogType = $derived(seo.ogType || fallback.ogType);
	// let ogImage = seo.ogImage;

	// Built here rather than inline in the template: an inline JSON-LD script tag
	// inside an {@html `…`} mustache trips up the ESLint Svelte parser. The closing
	// tag is written as <\/script> so neither the Svelte compiler nor an HTML
	// parser ends the surrounding block early.
	let jsonLdScript = $derived(
		seo.jsonLd
			? // eslint-disable-next-line no-useless-escape
				`<script type="application/ld+json">${JSON.stringify(seo.jsonLd)}<\/script>`
			: null,
	);
</script>

<title>{title}</title>
<meta name="description" content={description} />
<meta property="og:site_name" content="Note di Matteo" />
<meta property="og:title" content={ogTitle} />
<meta property="og:type" content={ogType} />
<meta property="og:description" content={description} />
<!--<meta property="og:image" content={ogImage} />-->
<!--<meta name="twitter:card" content="summary_large_image" />-->
<!--<meta name="twitter:image" content={ogImage} />-->

{#if jsonLdScript}
	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html jsonLdScript}
{/if}

<link rel="canonical" href="https://notedimatteo.it{page.url.pathname}" />
