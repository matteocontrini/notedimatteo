CREATE TABLE "posts" (
	"id" serial PRIMARY KEY NOT NULL,
	"number" integer NOT NULL,
	"category" text NOT NULL,
	"createdAt" timestamp with time zone DEFAULT now() NOT NULL,
	"updatedAt" timestamp with time zone DEFAULT now(),
	"publishedAt" timestamp with time zone DEFAULT now(),
	"title" text,
	"slug" text,
	"hideTitle" boolean NOT NULL,
	"body" text NOT NULL,
	CONSTRAINT "posts_number_unique" UNIQUE("number")
);
--> statement-breakpoint
CREATE TABLE "posts_tags" (
	"postId" integer NOT NULL,
	"tagName" text NOT NULL,
	CONSTRAINT "posts_tags_postId_tagName_pk" PRIMARY KEY("postId","tagName")
);
--> statement-breakpoint
CREATE TABLE "tags" (
	"name" text PRIMARY KEY NOT NULL
);
--> statement-breakpoint
ALTER TABLE "posts_tags" ADD CONSTRAINT "posts_tags_postId_posts_id_fk" FOREIGN KEY ("postId") REFERENCES "public"."posts"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "posts_tags" ADD CONSTRAINT "posts_tags_tagName_tags_name_fk" FOREIGN KEY ("tagName") REFERENCES "public"."tags"("name") ON DELETE cascade ON UPDATE cascade;