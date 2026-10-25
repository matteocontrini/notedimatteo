import { PrismaPg } from '@prisma/adapter-pg';
import { PrismaClient } from './generated/client';
import { DATABASE_URL } from '$app/env/private';
import { building } from '$app/env';

if (!building && !DATABASE_URL) throw new Error('DATABASE_URL is not set');

const globalForPrisma = globalThis as unknown as {
	prisma: PrismaClient | undefined;
};

export const db =
	globalForPrisma.prisma ??
	new PrismaClient({ adapter: new PrismaPg({ connectionString: DATABASE_URL }) });

if (!globalForPrisma.prisma) {
	globalForPrisma.prisma = db;
}
