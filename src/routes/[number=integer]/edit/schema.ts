import { z } from 'zod/v4';

const trimmedString = () => z.string().trim();

export const postFormSchema = z.object({
	number: z.coerce.number().int().min(1, 'Number must be at least 1'),
	category: trimmedString().min(1, 'Category is required'),
	title: trimmedString(),
	slug: trimmedString(),
	hideTitle: z.coerce.boolean(),
	body: z.string().min(1, 'Body is required'),
	tags: trimmedString()
});

export type PostFormSchema = typeof postFormSchema;
export type PostFormValues = z.infer<typeof postFormSchema>;
