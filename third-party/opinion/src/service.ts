import type { OpinionServiceServer } from './gen/opinion/opinion.js';
import { toGrpcError } from './grpc-error.js';
import { getOpinionClient, assertSdkSuccess } from './client.js';
import {
  parseGetCategoricalMarketRequest,
  parseGetMarketBySlugRequest,
  parseGetMarketRequest,
  parseGetMarketsRequest,
  parseGetQuoteTokensRequest,
  parseGetOrderbookRequest,
  parseGetLatestPriceRequest,
  parseGetPriceHistoryRequest,
  parseGetFeeRatesRequest,
  parsePlaceOrderRequest,
  parsePlaceOrdersBatchRequest,
  parseCancelOrderRequest,
  parseCancelOrdersBatchRequest,
  parseCancelAllOrdersRequest,
  parseGetMyOrdersRequest,
  parseGetOrderByIdRequest,
  parseGetMyPositionsRequest,
  parseGetMyTradesRequest,
  parseSplitRequest,
  parseMergeRequest,
  parseRedeemRequest,
  toGetMarketDetailResponse,
  toGetMarketsResponse,
  toGetQuoteTokensResponse,
  toGetOrderbookResponse,
  toGetLatestPriceResponse,
  toGetPriceHistoryResponse,
  toGetFeeRatesResponse,
  toPlaceOrderResponse,
  toPlaceOrdersBatchResponse,
  toCancelOrderApiResponse,
  toCancelOrdersBatchResponse,
  toCancelAllOrdersResponse,
  toGetMyOrdersResponse,
  toGetOrderByIdResponse,
  toGetMyBalancesResponse,
  toGetMyPositionsResponse,
  toGetMyTradesResponse,
  toGetUserAuthResponse,
  sdkTransactionResultToProto,
} from './common.js';

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

    async getOrderbook(call, callback) {
      try {
        const tokenId = parseGetOrderbookRequest(call.request);
        const response = await client.getOrderbook(tokenId);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetOrderbookResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getLatestPrice(call, callback) {
      try {
        const tokenId = parseGetLatestPriceRequest(call.request);
        const response = await client.getLatestPrice(tokenId);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetLatestPriceResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getPriceHistory(call, callback) {
      try {
        const { tokenId, options } = parseGetPriceHistoryRequest(call.request);
        const response = await client.getPriceHistory(tokenId, options);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetPriceHistoryResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getFeeRates(call, callback) {
      try {
        const tokenId = parseGetFeeRatesRequest(call.request);
        const settings = await client.getFeeRates(tokenId);
        callback(null, toGetFeeRatesResponse(settings));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async placeOrder(call, callback) {
      try {
        const { data, checkApproval } = parsePlaceOrderRequest(call.request);
        const response = await client.placeOrder(data, checkApproval);
        callback(null, toPlaceOrderResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async placeOrdersBatch(call, callback) {
      try {
        const { orders, checkApproval } = parsePlaceOrdersBatchRequest(call.request);
        const results = await client.placeOrdersBatch(orders, checkApproval);
        callback(null, toPlaceOrdersBatchResponse(results));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async cancelOrder(call, callback) {
      try {
        const orderId = parseCancelOrderRequest(call.request);
        const response = await client.cancelOrder(orderId);
        callback(null, toCancelOrderApiResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async cancelOrdersBatch(call, callback) {
      try {
        const orderIds = parseCancelOrdersBatchRequest(call.request);
        const results = await client.cancelOrdersBatch(orderIds);
        callback(null, toCancelOrdersBatchResponse(results));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async cancelAllOrders(call, callback) {
      try {
        const options = parseCancelAllOrdersRequest(call.request);
        const result = await client.cancelAllOrders(options);
        callback(null, toCancelAllOrdersResponse(result));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getMyOrders(call, callback) {
      try {
        const options = parseGetMyOrdersRequest(call.request);
        const response = await client.getMyOrders(options);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMyOrdersResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getOrderById(call, callback) {
      try {
        const orderId = parseGetOrderByIdRequest(call.request);
        const response = await client.getOrderById(orderId);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetOrderByIdResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getMyBalances(call, callback) {
      try {
        const response = await client.getMyBalances();
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMyBalancesResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getMyPositions(call, callback) {
      try {
        const options = parseGetMyPositionsRequest(call.request);
        const response = await client.getMyPositions(options);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMyPositionsResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getMyTrades(call, callback) {
      try {
        const options = parseGetMyTradesRequest(call.request);
        const response = await client.getMyTrades(options);
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetMyTradesResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async getUserAuth(call, callback) {
      try {
        const response = await client.getUserAuth();
        assertSdkSuccess(response.errno, response.errmsg);
        callback(null, toGetUserAuthResponse(response));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async enableTrading(call, callback) {
      try {
        void call;
        const result = await client.enableTrading();
        callback(null, sdkTransactionResultToProto(result));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async split(call, callback) {
      try {
        const { marketId, amount, checkApproval } = parseSplitRequest(call.request);
        const result = await client.split(marketId, amount, checkApproval);
        callback(null, sdkTransactionResultToProto(result));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async merge(call, callback) {
      try {
        const { marketId, amount, checkApproval } = parseMergeRequest(call.request);
        const result = await client.merge(marketId, amount, checkApproval);
        callback(null, sdkTransactionResultToProto(result));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },

    async redeem(call, callback) {
      try {
        const { marketId, checkApproval } = parseRedeemRequest(call.request);
        const result = await client.redeem(marketId, checkApproval);
        callback(null, sdkTransactionResultToProto(result));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },
  };
}
