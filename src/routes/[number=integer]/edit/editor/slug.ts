export const normalizeSlug = (value: string) =>
	value
		.toLowerCase()
		.normalize('NFD') // Decompose combined characters
		.replace(/[\u0300-\u036f]/g, '') // Remove diacritics (accents)
		.replace(/\s+/g, '-') // Replace spaces with hyphens
		.replace(/[^\w-]+/g, '') // Remove non-word chars except hyphens
		.replace(/--+/g, '-'); // Replace multiple hyphens with single hyphen
