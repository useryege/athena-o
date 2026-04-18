import type { Client } from '@opinion-labs/opinion-clob-sdk';
import { status as grpcStatus } from '@grpc/grpc-js';

import { getOpinionClient } from '../client.js';
import type { GetMarketsRequest, GetMarketsResponse } from '../gen/opinion/opinion.js';
import { createServiceError } from '../grpc-error.js';
import { mapGetMarketsResponse } from '../mappers.js';
import { buildGetMarketsOptions } from './request.js';

type OpinionGetMarketsOptions = Parameters<Client['getMarkets']>[0];
type OpinionGetMarketsResult = Awaited<ReturnType<Client['getMarkets']>>;

export interface OpinionMarketsClient {
  getMarkets(options: OpinionGetMarketsOptions): Promise<OpinionGetMarketsResult>;
}

export interface GetMarketsDependencies {
  client?: OpinionMarketsClient;
  getClient?: () => OpinionMarketsClient;
}

function resolveOpinionClient(dependencies: GetMarketsDependencies): OpinionMarketsClient {
  if (dependencies.client) {
    return dependencies.client;
  }

  if (dependencies.getClient) {
    return dependencies.getClient();
  }

  return getOpinionClient();
}

export async function executeGetMarkets(
  request: GetMarketsRequest,
  dependencies: GetMarketsDependencies = {},
): Promise<GetMarketsResponse> {
  const client = resolveOpinionClient(dependencies);
  const response = await client.getMarkets(buildGetMarketsOptions(request));

  if (response.errno !== 0) {
    throw createServiceError(
      grpcStatus.INTERNAL,
      response.errmsg || `opinion sdk returned errno ${response.errno}`,
    );
  }

  return mapGetMarketsResponse(response.result ?? {});
}
