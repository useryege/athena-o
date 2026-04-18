import { status as grpcStatus } from '@grpc/grpc-js';

import type { GetMarketRequest } from '../gen/opinion/opinion.js';
import { createServiceError } from '../grpc-error.js';

export function buildGetMarketOptions(request: GetMarketRequest): number {
  const marketId = Number(request.marketId);

  if (!Number.isInteger(marketId) || marketId < 1) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'market_id must be >= 1');
  }

  return marketId;
}
