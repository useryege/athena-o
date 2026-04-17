import Fastify, { type FastifyInstance } from 'fastify';
import sensible from '@fastify/sensible';
import { registerRoutes } from './router.js';
import { registerErrorHandler } from '../infra/http/error-middleware.js';
import { getClientStatus } from '../infra/opinion/client.js';
import { ok } from '../infra/http/response.js';

export function buildServer(): FastifyInstance {
  const isDev = process.env['NODE_ENV'] !== 'production';

  const app = Fastify({
    logger: isDev
      ? { level: 'info' }
      : { level: 'info' },
  });

  void app.register(sensible);

  // liveness — process is up
  app.get('/healthz', async (_req, reply) => {
    await reply.send(ok({ status: 'ok' }));
  });

  // readiness — SDK client is initialized and ready to serve
  app.get('/readyz', async (_req, reply) => {
    const status = getClientStatus();
    if (!status.initialized) {
      await reply.status(503).send({
        success: false,
        initialized: false,
        error: status.error ?? 'Client not yet initialized',
      });
      return;
    }
    await reply.send(ok({ initialized: true }));
  });

  registerErrorHandler(app);
  void app.register(registerRoutes);

  return app;
}
