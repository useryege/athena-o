import { status as grpcStatus } from '@grpc/grpc-js';
import type { OpinionServiceServer } from './gen/opinion/opinion.js';
import { createServiceError, toGrpcError } from './grpc-error.js';
import { getOpinionClient, assertSdkSuccess } from './client.js';
import {
  parseGetCategoricalMarketRequest,
  parseGetMarketBySlugRequest,
  parseGetMarketRequest,
  parseGetMarketsRequest,
  parseGetQuoteTokensRequest,
  toGetMarketDetailResponse,
  toGetMarketsResponse,
  toGetQuoteTokensResponse,
} from './market.js';

export function createOpinionService(): OpinionServiceServer {
  const client = getOpinionClient();

  return {
    async getMarkets(call, callback) {
      try {
        const query = parseGetMarketsRequest(call.request);
        const response = await client.getMarkets(query);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMarketsResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getMarket(call, callback) {
      try {
        const { marketId, useCache } = parseGetMarketRequest(call.request);
        const response = await client.getMarket(marketId, useCache);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMarketDetailResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getCategoricalMarket(call, callback) {
      try {
        const marketId = parseGetCategoricalMarketRequest(call.request);
        const response = await client.getCategoricalMarket(marketId);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMarketDetailResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getMarketBySlug(call, callback) {
      try {
        const slug = parseGetMarketBySlugRequest(call.request);
        const response = await client.getMarketBySlug(slug);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMarketDetailResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getQuoteTokens(call, callback) {
      try {
        const useCache = parseGetQuoteTokensRequest(call.request);
        const response = await client.getQuoteTokens(useCache);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetQuoteTokensResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getOrderbook(_call, callback) {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, 'getOrderbook is not implemented'));
    },

    async getLatestPrice(_call, callback) {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, 'getLatestPrice is not implemented'));
    },

    async getPriceHistory(_call, callback) {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, 'getPriceHistory is not implemented'));
    },

    async getFeeRates(_call, callback) {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, 'getFeeRates is not implemented'));
    },

    async placeOrder(_call, callback) {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, 'placeOrder is not implemented'));
    },

    async placeOrdersBatch(_call, callback) {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, 'placeOrdersBatch is not implemented'));
    },

    async cancelOrder(_call, callback) {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, 'cancelOrder is not implemented'));
    },

    async cancelOrdersBatch(_call, callback) {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, 'cancelOrdersBatch is not implemented'));
    },

    async cancelAllOrders(_call, callback) {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, 'cancelAllOrders is not implemented'));
    },
  };
}
