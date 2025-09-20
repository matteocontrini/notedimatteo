import type { ServerInit } from '@sveltejs/kit';
import { db } from '$lib/server/db';

export const init: ServerInit = async () => {
	await db.$connect();
};
