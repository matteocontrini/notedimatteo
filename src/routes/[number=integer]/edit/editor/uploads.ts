type UploadedImage = {
	main: string;
	preview: string;
	width: number;
	height: number;
};

export const uploadImage = async (endpoint: string, file: File): Promise<UploadedImage> => {
	const formData = new FormData();
	formData.append('file', file);

	const response = await fetch(endpoint, {
		method: 'POST',
		body: formData,
	});

	if (!response.ok) {
		throw new Error('Upload failed');
	}

	return (await response.json()) as UploadedImage;
};

export const getImageMarkup = (image: UploadedImage) =>
	// Add image markdown to the body with dimensions
	`[img width=${image.width} height=${image.height}]${image.main}[/img]`;

export const insertImageMarkup = (body: string, markup: string, cursorPosition?: number) => {
	if (cursorPosition === undefined) {
		// Fallback: append to end (for file input upload)
		return { body: `${body}\n\n${markup}`, cursorPosition: undefined };
	}

	// If cursor position is provided, insert at cursor with smart spacing
	const before = body.substring(0, cursorPosition);
	const after = body.substring(cursorPosition);
	// Smart spacing: only add newlines if needed before and after the image
	const spacingBefore =
		before.length === 0 || before.endsWith('\n\n') ? '' : before.endsWith('\n') ? '\n' : '\n\n';
	const spacingAfter =
		after.length === 0 || after.startsWith('\n\n') ? '' : after.startsWith('\n') ? '\n' : '\n\n';

	return {
		body: before + spacingBefore + markup + spacingAfter + after,
		// Calculate new cursor position (after the inserted image)
		cursorPosition: before.length + spacingBefore.length + markup.length + spacingAfter.length,
	};
};
