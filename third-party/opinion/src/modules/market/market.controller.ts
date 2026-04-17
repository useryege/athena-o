import type { FastifyRequest, FastifyReply } from 'fastify';
import { getMarkets } from './market.service.js';
import type { GetMarketsQuery } from './market.dto.js';
import { ok } from '../../infra/http/response.js';

export async function handleGetMarkets(
  req: FastifyRequest<{ Querystring: GetMarketsQuery }>,
  reply: FastifyReply,
): Promise<void> {
  const result = await getMarkets(req.query);
  await reply.send(ok(result));
}
