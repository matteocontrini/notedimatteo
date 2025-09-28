-- CreateTable
CREATE TABLE "public"."posts" (
    "id" SERIAL NOT NULL,
    "number" INTEGER NOT NULL,
    "category" TEXT NOT NULL,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    "publishedAt" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    "title" TEXT,
    "slug" TEXT,
    "hideTitle" BOOLEAN NOT NULL,
    "body" TEXT NOT NULL,

    CONSTRAINT "posts_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "public"."tags" (
    "name" TEXT NOT NULL,

    CONSTRAINT "tags_pkey" PRIMARY KEY ("name")
);

-- CreateTable
CREATE TABLE "public"."posts_tags" (
    "postId" INTEGER NOT NULL,
    "tagName" TEXT NOT NULL,

    CONSTRAINT "posts_tags_pkey" PRIMARY KEY ("postId","tagName")
);

-- CreateIndex
CREATE UNIQUE INDEX "posts_number_key" ON "public"."posts"("number");

-- AddForeignKey
ALTER TABLE "public"."posts_tags" ADD CONSTRAINT "posts_tags_postId_fkey" FOREIGN KEY ("postId") REFERENCES "public"."posts"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "public"."posts_tags" ADD CONSTRAINT "posts_tags_tagName_fkey" FOREIGN KEY ("tagName") REFERENCES "public"."tags"("name") ON DELETE CASCADE ON UPDATE CASCADE;
