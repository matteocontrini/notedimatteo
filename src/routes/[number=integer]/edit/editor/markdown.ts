export type TextEdit = {
	value: string;
	selectionStart: number;
	selectionEnd: number;
};

const replaceSelection = (
	value: string,
	selectionStart: number,
	selectionEnd: number,
	replacement: string,
): string => value.substring(0, selectionStart) + replacement + value.substring(selectionEnd);

export const insertMarkdownLink = (
	value: string,
	selectionStart: number,
	selectionEnd: number,
): TextEdit => {
	const selectedText = value.substring(selectionStart, selectionEnd);

	if (selectedText.length === 0) {
		// Case 1: No selection - insert []() with cursor at link text position
		return {
			value: replaceSelection(value, selectionStart, selectionEnd, '[]()'),
			selectionStart: selectionStart + 1,
			selectionEnd: selectionStart + 1,
		};
	}

	if (selectedText.startsWith('http://') || selectedText.startsWith('https://')) {
		// Case 2: Selected text is URL - create [](url) with cursor at link text position
		return {
			value: replaceSelection(value, selectionStart, selectionEnd, `[](${selectedText})`),
			selectionStart: selectionStart + 1,
			selectionEnd: selectionStart + 1,
		};
	}

	// Case 3: Selected text is regular text - create [text]() with cursor at URL position
	const replacement = `[${selectedText}]()`;
	const cursorPosition = selectionStart + selectedText.length + 3;

	return {
		value: replaceSelection(value, selectionStart, selectionEnd, replacement),
		selectionStart: cursorPosition,
		selectionEnd: cursorPosition,
	};
};

const wrapMarkdown = (
	value: string,
	selectionStart: number,
	selectionEnd: number,
	marker: string,
): TextEdit => {
	const selectedText = value.substring(selectionStart, selectionEnd);
	// With no selection, insert paired markers with the cursor in the middle.
	// Otherwise, wrap the selected text in the marker.
	const replacement = `${marker}${selectedText}${marker}`;
	const nextSelectionStart = selectionStart + marker.length;

	return {
		value: replaceSelection(value, selectionStart, selectionEnd, replacement),
		selectionStart: nextSelectionStart,
		selectionEnd: selectedText.length === 0 ? nextSelectionStart : selectionEnd + marker.length,
	};
};

export const toggleMarkdownBold = (value: string, selectionStart: number, selectionEnd: number) =>
	wrapMarkdown(value, selectionStart, selectionEnd, '**');

export const toggleMarkdownItalic = (value: string, selectionStart: number, selectionEnd: number) =>
	wrapMarkdown(value, selectionStart, selectionEnd, '*');

export const toggleMarkdownQuote = (
	value: string,
	selectionStart: number,
	selectionEnd: number,
): TextEdit => {
	if (selectionStart === selectionEnd) {
		// No selection: insert "> " at cursor position
		const nextCursorPosition = selectionStart + 2;

		return {
			value: replaceSelection(value, selectionStart, selectionEnd, '> '),
			selectionStart: nextCursorPosition,
			selectionEnd: nextCursorPosition,
		};
	}

	// Has selection: quote each line in the selection
	// Find the line boundaries
	const lineStartPosition = value.lastIndexOf('\n', selectionStart - 1) + 1;
	const nextLineBreak = value.indexOf('\n', selectionEnd);
	const lineEndPosition = nextLineBreak === -1 ? value.length : nextLineBreak;
	const textToQuote = value.substring(lineStartPosition, lineEndPosition);
	const lines = textToQuote.split('\n');
	// Add "> " to each line, including blank lines to maintain quote block
	const quotedText = lines.map((line) => (line.length > 0 ? `> ${line}` : '>')).join('\n');
	// Calculate new cursor positions
	const totalAdded = lines.reduce((sum, line) => sum + (line.length > 0 ? 2 : 1), 0);
	const firstLineAdded = lines[0].length > 0 ? 2 : 1;

	return {
		value: replaceSelection(value, lineStartPosition, lineEndPosition, quotedText),
		selectionStart: selectionStart + firstLineAdded,
		selectionEnd: selectionEnd + totalAdded,
	};
};
