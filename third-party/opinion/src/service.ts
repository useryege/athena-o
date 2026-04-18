import type { OpinionServiceServer } from './gen/opinion/opinion.js';
import { toGrpcError } from './grpc-error.js';
import { getOpinionClient, assertSdkSuccess } from './client.js';
import {
  parseGetMarketRequest,
  parseGetMarketsRequest,
  toGetMarketResponse,
  toGetMarketsResponse,
} from './market.js';
import type { MarketDetailResult, MarketListResult } from './market.js';

export function createOpinionService(): OpinionServiceServer {
  const client = getOpinionClient();

  return {
    async getMarkets(call, callback) {
      try {
        const query = parseGetMarketsRequest(call.request);
        const response = await client.getMarkets(query);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMarketsResponse((response.result ?? {}) as MarketListResult));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getMarket(call, callback) {
      try {
        const marketId = parseGetMarketRequest(call.request);
        const response = await client.getMarket(marketId);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMarketResponse((response.result ?? {}) as MarketDetailResult));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },
  };
}
