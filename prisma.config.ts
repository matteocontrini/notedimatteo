import 'dotenv/config';
import path from 'node:path';
import type { PrismaConfig } from 'prisma';

export default {
	schema: path.join('src', 'prisma')
} satisfies PrismaConfig;
