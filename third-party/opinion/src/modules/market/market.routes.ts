import type { FastifyInstance } from 'fastify';
import { handleGetMarkets } from './market.controller.js';

export async function marketRoutes(app: FastifyInstance): Promise<void> {
  app.get('/v1/markets', handleGetMarkets);
}
