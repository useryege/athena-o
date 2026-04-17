import type { FastifyInstance } from 'fastify';
import { marketRoutes } from '../modules/market/market.routes.js';

export async function registerRoutes(app: FastifyInstance): Promise<void> {
  await app.register(marketRoutes);
}
