const monthFormatter = new Intl.DateTimeFormat('it-IT', { month: 'long' });

export function formatMonthName(year: number, month: number): string {
	return monthFormatter.format(new Date(Date.UTC(year, month - 1, 1)));
}

export function formatMonthParam(month: number): string {
	return month.toString().padStart(2, '0');
}
