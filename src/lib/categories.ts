const CATEGORY_LABELS: Record<string, string> = {
	note: 'Note',
	til: 'TIL',
	cit: 'Cit.',
	segnalibri: 'Segnalibri'
};

export function getCategoryLabel(slug: string): string {
	if (!slug) return '';
	return (
		CATEGORY_LABELS[slug] ?? slug.replace(/-/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase())
	);
}

export const categorySlugs = Object.keys(CATEGORY_LABELS);
