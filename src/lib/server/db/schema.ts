import {
	boolean,
	integer,
	pgTable,
	primaryKey,
	serial,
	text,
	timestamp
} from 'drizzle-orm/pg-core';
import { relations } from 'drizzle-orm';

export const posts = pgTable('posts', {
	id: serial('id').primaryKey(),
	number: integer().unique().notNull(),
	category: text().notNull(),
	createdAt: timestamp({ withTimezone: true }).defaultNow().notNull(),
	updatedAt: timestamp({ withTimezone: true }).defaultNow(),
	publishedAt: timestamp({ withTimezone: true }).defaultNow(),
	title: text(),
	slug: text(),
	hideTitle: boolean().notNull(),
	body: text().notNull()
});

export const tags = pgTable('tags', {
	name: text().primaryKey()
});

export const postsToTags = pgTable(
	'posts_tags',
	{
		postId: integer()
			.references(() => posts.id, { onDelete: 'cascade' })
			.notNull(),
		tagName: text()
			.references(() => tags.name, { onDelete: 'cascade', onUpdate: 'cascade' })
			.notNull()
	},
	(table) => [primaryKey({ columns: [table.postId, table.tagName] })]
);

export const postsToTagsRelations = relations(postsToTags, ({ one }) => ({
	post: one(posts, { fields: [postsToTags.postId], references: [posts.id] }),
	tag:  one(tags,  { fields: [postsToTags.tagName], references: [tags.name] }),
}));

export const postsRelations = relations(posts, ({ many }) => ({
	postsToTags: many(postsToTags)
}));

export const tagsRelations = relations(tags, ({ many }) => ({
	tagsToPosts: many(postsToTags)
}));
