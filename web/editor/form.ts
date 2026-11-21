// This file defines the editor's data and sends its requests to the Go server.
// The JSON types match editorData and saveResponse in internal/server/edit.go.

export type EditorURLs = {
	post: string;
	edit: string;
	preview: string;
	upload: string;
};

export type EditorPost = {
	id: number;
	number: number;
	category: string;
	title: string;
	slug: string;
	hideTitle: boolean;
	body: string;
	tags: string[];
	createdAt: string;
	updatedAt: string | null;
	publishedAt: string | null;
	urls: EditorURLs;
};

export type Category = {
	slug: string;
	label: string;
};

export type EditorData = {
	post: EditorPost;
	// tags are the names the tags field suggests
	tags: string[];
	categories: Category[];
};

// PostForm holds the fields as the inputs edit them. A number input that the
// user empties holds null.
export type PostForm = {
	number: number | null;
	category: string;
	title: string;
	slug: string;
	hideTitle: boolean;
	body: string;
	tags: string;
};

export type Intent = 'save' | 'publish' | 'unpublish';

export type FieldErrors = Partial<Record<keyof PostForm, string>>;

export type SaveResult = { post: EditorPost; errors?: undefined } | { errors: FieldErrors };

export const formFromPost = (post: EditorPost): PostForm => ({
	number: post.number,
	category: post.category,
	title: post.title,
	slug: post.slug,
	hideTitle: post.hideTitle,
	body: post.body,
	tags: post.tags.join(', '),
});

export const sameForm = (a: PostForm, b: PostForm) =>
	(Object.keys(a) as (keyof PostForm)[]).every((field) => a[field] === b[field]);

export const toFormData = (form: PostForm, intent: Intent) => {
	const data = new FormData();
	data.set('intent', intent);
	data.set('number', form.number === null ? '' : String(form.number));
	data.set('category', form.category);
	data.set('title', form.title);
	data.set('slug', form.slug);
	data.set('hideTitle', String(form.hideTitle));
	data.set('body', form.body);
	data.set('tags', form.tags);
	return data;
};

// savePost sends the form to the save endpoint. It returns the saved post, or
// the errors of the fields that the server rejected. It throws for every other
// failure, like an expired session.
export const savePost = async (
	url: string,
	form: PostForm,
	intent: Intent,
): Promise<SaveResult> => {
	const response = await fetch(url, { method: 'POST', body: toFormData(form, intent) });
	const result = await response.json().catch(() => ({}));

	if (response.ok && result.post) return { post: result.post };
	if (response.status === 400 && result.errors) return { errors: result.errors };
	throw new Error(result.error ?? `${response.status} ${response.statusText}`);
};
