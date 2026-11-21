-- CreateTable
CREATE TABLE "public"."sessions" (
    "id" SERIAL NOT NULL,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastActivityAt" TIMESTAMPTZ NOT NULL,
    "token" TEXT NOT NULL,

    CONSTRAINT "sessions_pkey" PRIMARY KEY ("id")
);
