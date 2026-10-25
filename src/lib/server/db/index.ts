import { PrismaClient } from '@prisma/client';
import { DATABASE_URL } from '$app/env/private';
import { building } from '$app/env';

if (!building && !DATABASE_URL) throw new Error('DATABASE_URL is not set');

const globalForPrisma = globalThis as unknown as {
	prisma: PrismaClient | undefined;
};

export const db = globalForPrisma.prisma ?? new PrismaClient();

if (!globalForPrisma.prisma) {
	globalForPrisma.prisma = db;
}
