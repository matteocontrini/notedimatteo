import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { existsSync } from 'node:fs';
import { PostgreSqlContainer } from '@testcontainers/postgresql';
import { GenericContainer, Wait } from 'testcontainers';
import { E2E_PORT } from '../../playwright.config.ts';
import { ADMIN_PASSWORD, ADMIN_USERNAME, BUCKET, seedSQL } from './data.ts';

/**
 * Runs the server built by `npm run test:e2e` against a throwaway Postgres
 * and an in-memory S3 server. The server applies the migrations when it
 * starts; the seed data from data.ts goes in after that.
 */
export default async function globalSetup() {
	if (!existsSync('.e2e/server')) {
		throw new Error('.e2e/server is missing: run the tests with npm run test:e2e');
	}

	const [postgres, s3] = await Promise.all([startPostgres(), startS3()]);
	const databaseURL = postgres.getConnectionUri();

	// rclone serves the bucket over plain HTTP too, as the public address of the
	// uploaded images
	const storageURL = `http://localhost:${s3.getMappedPort(8080)}`;
	// Tests read the stored files from there
	process.env.E2E_STORAGE_URL = storageURL;

	const server = spawn('.e2e/server', {
		env: {
			PATH: process.env.PATH,
			PORT: String(E2E_PORT),
			DATABASE_URL: databaseURL,
			// A site address that isn't the test server's, so the page turns GoatCounter off
			SITE_URL: 'https://notedimatteo.it',
			ADMIN_USERNAME,
			ADMIN_PASSWORD,
			STORAGE_PUBLIC_BASE_URL: storageURL,
			STORAGE_S3_ENDPOINT: `http://localhost:${s3.getMappedPort(9000)}`,
			STORAGE_S3_REGION: 'us-east-1',
			STORAGE_S3_ACCESS_KEY_ID: 'test',
			STORAGE_S3_SECRET_ACCESS_KEY: 'test',
			STORAGE_S3_BUCKET: BUCKET,
		},
		stdio: ['ignore', 'pipe', 'pipe'],
	});

	let output = '';
	server.stdout.on('data', (chunk) => (output += chunk));
	server.stderr.on('data', (chunk) => (output += chunk));

	const exited = once(server, 'exit').then(([code]) => {
		throw new Error(`The server exited early with code ${code}:\n${output}`);
	});
	await Promise.race([waitForServer(`http://localhost:${E2E_PORT}/healthz`), exited]);

	const seed = await postgres.exec([
		'psql',
		'-v',
		'ON_ERROR_STOP=1',
		'-U',
		postgres.getUsername(),
		'-d',
		postgres.getDatabase(),
		'-c',
		seedSQL(),
	]);
	const stop = async () => {
		server.removeAllListeners('exit');
		exited.catch(() => {});
		server.kill();
		await Promise.all([postgres.stop(), s3.stop()]);

		if (process.env.E2E_SERVER_LOGS) console.log(output);
	};

	if (seed.exitCode !== 0) {
		await stop();
		throw new Error(`The seed failed:\n${seed.output}`);
	}
	return stop;
}

function startPostgres() {
	return new PostgreSqlContainer('postgres:18')
		.withDatabase('notedimatteo')
		.withCommand(['postgres', '-c', 'fsync=off'])
		.start();
}

// startS3 runs rclone with two servers over one directory: S3 on port 9000,
// where the bucket is a subdirectory, and plain HTTP on 8080 for that bucket.
function startS3() {
	return new GenericContainer('rclone/rclone:1.74.4')
		.withEntrypoint(['sh', '-c'])
		.withCommand([
			`mkdir -p /data/${BUCKET} && ` +
				`(rclone serve http /data/${BUCKET} --addr :8080 &) && ` +
				'exec rclone serve s3 /data --addr :9000 --auth-key test,test --force-path-style=false',
		])
		.withExposedPorts(9000, 8080)
		.withWaitStrategy(Wait.forListeningPorts())
		.start();
}

async function waitForServer(url: string, timeoutMs = 30_000) {
	const deadline = Date.now() + timeoutMs;

	while (Date.now() < deadline) {
		try {
			const response = await fetch(url);
			if (response.ok) return;
		} catch {
			// Not listening yet
		}
		await new Promise((resolve) => setTimeout(resolve, 250));
	}

	throw new Error(`The server did not start within ${timeoutMs}ms`);
}
