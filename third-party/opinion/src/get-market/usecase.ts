import type { Client } from '@opinion-labs/opinion-clob-sdk';
import { status as grpcStatus } from '@grpc/grpc-js';

import { getOpinionClient } from '../client.js';
import type { GetMarketRequest, GetMarketResponse } from '../gen/opinion/opinion.js';
import { createServiceError } from '../grpc-error.js';
import { mapGetMarketResponse } from '../mappers.js';
import { buildGetMarketOptions } from './request.js';

type OpinionGetMarketResult = Awaited<ReturnType<Client['getMarket']>>;

export interface OpinionMarketClient {
  getMarket(marketId: number): Promise<OpinionGetMarketResult>;
}

export interface GetMarketDependencies {
  client?: OpinionMarketClient;
  getClient?: () => OpinionMarketClient;
}

function resolveOpinionClient(dependencies: GetMarketDependencies): OpinionMarketClient {
  if (dependencies.client) {
    return dependencies.client;
  }

  if (dependencies.getClient) {
    return dependencies.getClient();
  }

  return getOpinionClient();
}

export async function executeGetMarket(
  request: GetMarketRequest,
  dependencies: GetMarketDependencies = {},
): Promise<GetMarketResponse> {
  const marketId = buildGetMarketOptions(request);
  const client = resolveOpinionClient(dependencies);
  const response = await client.getMarket(marketId);

  if (response.errno !== 0) {
    throw createServiceError(
      grpcStatus.INTERNAL,
      response.errmsg || `opinion sdk returned errno ${response.errno}`,
    );
  }

  return mapGetMarketResponse(response.result ?? {});
}
