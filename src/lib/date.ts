const monthFormatter = new Intl.DateTimeFormat('it-IT', { month: 'long' });

export function formatMonthName(year: number, month: number, capitalize = false): string {
	const label = monthFormatter.format(new Date(Date.UTC(year, month - 1, 1)));
	return capitalize
		? label.replace(/^./, (char) => char.toLocaleUpperCase('it-IT'))
		: label;
}

export function formatMonthParam(month: number): string {
	return month.toString().padStart(2, '0');
}
