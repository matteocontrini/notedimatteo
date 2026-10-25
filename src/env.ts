import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	ADMIN_USERNAME: {},
	ADMIN_PASSWORD: {},
	STORAGE_S3_REGION: {},
	STORAGE_S3_ENDPOINT: {},
	STORAGE_S3_ACCESS_KEY_ID: {},
	STORAGE_S3_SECRET_ACCESS_KEY: {},
	STORAGE_S3_BUCKET: {},
	STORAGE_PUBLIC_BASE_URL: {},
	DATABASE_URL: {},
});
