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

// imageInsertion returns what to insert between `before` and `after` so that
// the image markup sits in its own paragraph, adding only the line breaks
// that are missing. At the end of the text it adds a paragraph after the
// image, where the cursor goes. `before` and `after` need only their last and
// first two characters.
export const imageInsertion = (before: string, after: string, markup: string) => {
	const spacingBefore =
		before === '' || before.endsWith('\n\n') ? '' : before.endsWith('\n') ? '\n' : '\n\n';
	const spacingAfter = after.startsWith('\n\n') ? '' : after.startsWith('\n') ? '\n' : '\n\n';
	return spacingBefore + markup + spacingAfter;
};
