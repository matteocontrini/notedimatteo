import 'dotenv/config';
import path from 'node:path';
import type { PrismaConfig } from 'prisma';

export default {
	schema: path.join('src', 'prisma'),
	migrations: {
		path: path.join('src', 'prisma', 'migrations'),
	},
	datasource: {
		url: process.env.DATABASE_URL!,
	},
} satisfies PrismaConfig;
