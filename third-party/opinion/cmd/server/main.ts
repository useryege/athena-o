import { buildServer } from '../../src/app/server.js';
import { bootstrap } from '../../src/app/container.js';
import { env } from '../../src/config/env.js';
import { logger } from '../../src/infra/logger/index.js';

async function main(): Promise<void> {
  bootstrap();

  const app = buildServer();

  try {
    await app.listen({ port: env.PORT, host: '0.0.0.0' });
    logger.info(`Opinion Sidecar listening on 0.0.0.0:${env.PORT}`);
  } catch (err) {
    logger.error('Failed to start server', err);
    process.exit(1);
  }
}

void main();
