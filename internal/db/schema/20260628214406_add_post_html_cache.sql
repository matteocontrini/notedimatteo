-- AlterTable
ALTER TABLE "public"."posts" ADD COLUMN     "bodyRevision" INTEGER NOT NULL DEFAULT 1;

-- CreateTable
CREATE TABLE "public"."post_html_cache" (
    "postId" INTEGER NOT NULL,
    "rendererVersion" INTEGER NOT NULL,
    "bodyRevision" INTEGER NOT NULL,
    "html" TEXT NOT NULL,
    "renderedAt" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "post_html_cache_pkey" PRIMARY KEY ("postId")
);

-- AddForeignKey
ALTER TABLE "public"."post_html_cache" ADD CONSTRAINT "post_html_cache_postId_fkey" FOREIGN KEY ("postId") REFERENCES "public"."posts"("id") ON DELETE CASCADE ON UPDATE CASCADE;
