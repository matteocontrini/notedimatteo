export interface Post {
	id: number;
	number: number;
	category: string;
	createdAt: Date;
	updatedAt: Date | null;
	publishedAt: Date | null;
	title: string | null;
	slug: string | null;
	hideTitle: boolean;
	body: string;
	tags: string[];
}
