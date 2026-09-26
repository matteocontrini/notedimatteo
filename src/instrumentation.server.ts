import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { PrismaInstrumentation } from '@prisma/instrumentation';
import { createAddHookMessageChannel } from 'import-in-the-middle';
import { register } from 'node:module';

if (process.env.OTEL_EXPORTER_OTLP_ENDPOINT) {
	process.env.OTEL_METRICS_EXPORTER ??= 'none';
	process.env.OTEL_LOGS_EXPORTER ??= 'none';

	const { registerOptions } = createAddHookMessageChannel();
	register('import-in-the-middle/hook.mjs', import.meta.url, registerOptions);

	const sdk = new NodeSDK({
		instrumentations: [getNodeAutoInstrumentations(), new PrismaInstrumentation()],
	});
	sdk.start();
	process.once('SIGTERM', () => {
		void sdk
			.shutdown()
			.catch((error: unknown) => console.error('Failed to shut down tracing', error));
	});
}
