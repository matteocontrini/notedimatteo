const monthFormatter = new Intl.DateTimeFormat('it-IT', { month: 'long' });

export function formatMonthName(year: number, month: number): string {
	return monthFormatter.format(new Date(Date.UTC(year, month - 1, 1)));
}
