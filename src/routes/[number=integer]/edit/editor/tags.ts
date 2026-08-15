type TagRange = {
	start: number;
	end: number;
};

export const getCurrentTagRange = (value: string, cursorPosition: number): TagRange => {
	const beforeCursor = value.substring(0, cursorPosition);
	const afterCursor = value.substring(cursorPosition);

	// Find where current tag starts (after last comma, or start)
	const lastComma = beforeCursor.lastIndexOf(',');
	// Find where current tag ends (before next comma, or end)
	const nextComma = afterCursor.indexOf(',');

	return {
		start: lastComma === -1 ? 0 : lastComma + 1,
		end: nextComma === -1 ? value.length : cursorPosition + nextComma,
	};
};

export const getCurrentTag = (value: string, cursorPosition: number) => {
	const range = getCurrentTagRange(value, cursorPosition);
	return value.substring(range.start, range.end).trim();
};

export const completeCurrentTag = (value: string, cursorPosition: number, suggestion: string) => {
	const range = getCurrentTagRange(value, cursorPosition);

	return {
		value: value.substring(0, range.start) + suggestion + value.substring(range.end),
		cursorPosition: range.start + suggestion.length,
	};
};
