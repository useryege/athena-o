import 'dotenv/config';

import * as grpc from '@grpc/grpc-js';
import { status as grpcStatus } from '@grpc/grpc-js';
import { AssetType as SdkAssetType, Side as SdkSide } from '@polymarket/clob-client-v2';
import assert from 'node:assert/strict';
import { after, before, describe, test } from 'node:test';

import { createPolymarketClient } from '../../src/client.js';
import { parseGetOkRequest, toGetOkResponse, toProtoStructData } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import * as proto from '../../src/gen/polymarket/polymarket.js';
import { assertEncodedEqual, getE2EGrpcAddress, handleE2EUnaryError } from './helpers.js';

/**
 * SDK + gRPC 各发一次上游请求做 parity，对列表类接口可能存在上游快照漂移风险。
 */
type RpcCase = {
  method: keyof proto.PolymarketServiceClient;
  responseType: { encode(message: any): { finish(): Uint8Array } };
  makeRequest: (env: RequiredEnv) => Record<string, unknown> | undefined;
  sdkCall: (sdkClient: ReturnType<typeof createPolymarketClient>, request: Record<string, unknown>) => Promise<unknown> | unknown;
  toExpected: (sdkResult: unknown, request: Record<string, unknown>) => any;
};

type RequiredEnv = {
  conditionId: string | undefined;
  tokenId: string | undefined;
  orderId: string | undefined;
  builderCode: string | undefined;
  rewardsDate: string;
};

function readEnv(): RequiredEnv {
  const date = process.env.POLYMARKET_E2E_DATE?.trim();
  return {
    conditionId: process.env.POLYMARKET_E2E_CONDITION_ID?.trim() || undefined,
    tokenId: process.env.POLYMARKET_E2E_TOKEN_ID?.trim() || undefined,
    orderId: process.env.POLYMARKET_E2E_ORDER_ID?.trim() || undefined,
    builderCode: process.env.POLYMARKET_E2E_BUILDER_CODE?.trim() || undefined,
    rewardsDate: date || '2024-01-01',
  };
}

function toSdkSide(side: proto.Side): SdkSide {
  switch (side) {
    case proto.Side.SIDE_BUY:
      return SdkSide.BUY;
    case proto.Side.SIDE_SELL:
      return SdkSide.SELL;
    default:
      return SdkSide.BUY;
  }
}

function toProtoSide(side: unknown): proto.Side {
  if (typeof side !== 'string') {
    return proto.Side.SIDE_UNSPECIFIED;
  }

  if (side.toUpperCase() === 'BUY') {
    return proto.Side.SIDE_BUY;
  }

  if (side.toUpperCase() === 'SELL') {
    return proto.Side.SIDE_SELL;
  }

  return proto.Side.SIDE_UNSPECIFIED;
}

function snakeToCamelKey(value: string): string {
  return value.replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase());
}

function toCamelCaseObject<T = any>(value: unknown): T {
  if (value === null || value === undefined) {
    return value as T;
  }

  if (Array.isArray(value)) {
    return value.map((item) => toCamelCaseObject(item)) as T;
  }

  if (typeof value !== 'object') {
    return value as T;
  }

  const output: Record<string, unknown> = {};
  for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
    output[snakeToCamelKey(key)] = toCamelCaseObject(item);
  }

  return output as T;
}

function toProtoPaginationPayloadData(value: unknown): {
  limit: number;
  count: number;
  nextCursor: string;
  data: { [key: string]: any }[];
} {
  const payload = (value ?? {}) as Record<string, unknown>;
  const rawData = Array.isArray(payload.data) ? payload.data : [];

  return {
    limit: Number(payload.limit ?? 0),
    count: Number(payload.count ?? 0),
    nextCursor: String(payload.next_cursor ?? ''),
    data: rawData.map((item) => toProtoStructData(toCamelCaseObject(item)) ?? {}),
  };
}

function toProtoTrade(value: unknown): Record<string, unknown> {
  const trade = (value ?? {}) as Record<string, unknown>;
  const makerOrders = Array.isArray(trade.maker_orders) ? trade.maker_orders : [];

  return {
    id: String(trade.id ?? ''),
    takerOrderId: String(trade.taker_order_id ?? ''),
    market: String(trade.market ?? ''),
    assetId: String(trade.asset_id ?? ''),
    side: toProtoSide(trade.side),
    size: String(trade.size ?? ''),
    feeRateBps: String(trade.fee_rate_bps ?? ''),
    price: String(trade.price ?? ''),
    status: String(trade.status ?? ''),
    matchTime: String(trade.match_time ?? ''),
    lastUpdate: String(trade.last_update ?? ''),
    outcome: String(trade.outcome ?? ''),
    bucketIndex: Number(trade.bucket_index ?? 0),
    owner: String(trade.owner ?? ''),
    makerAddress: String(trade.maker_address ?? ''),
    makerOrders: makerOrders.map((item) => {
      const order = (item ?? {}) as Record<string, unknown>;
      return {
        orderId: String(order.order_id ?? ''),
        owner: String(order.owner ?? ''),
        makerAddress: String(order.maker_address ?? ''),
        matchedAmount: String(order.matched_amount ?? ''),
        price: String(order.price ?? ''),
        feeRateBps: String(order.fee_rate_bps ?? ''),
        assetId: String(order.asset_id ?? ''),
        outcome: String(order.outcome ?? ''),
        side: toProtoSide(order.side),
      };
    }),
    transactionHash: String(trade.transaction_hash ?? ''),
    traderSide: String(trade.trader_side ?? ''),
  };
}

function toProtoMarketTradeEvent(value: unknown): Record<string, unknown> {
  const item = (value ?? {}) as Record<string, unknown>;
  const rawMarket = (item.market ?? {}) as Record<string, unknown>;
  const rawUser = (item.user ?? {}) as Record<string, unknown>;

  return {
    eventType: String(item.event_type ?? ''),
    market: {
      conditionId: String(rawMarket.condition_id ?? ''),
      assetId: String(rawMarket.asset_id ?? ''),
      question: String(rawMarket.question ?? ''),
      icon: String(rawMarket.icon ?? ''),
      slug: String(rawMarket.slug ?? ''),
    },
    user: {
      address: String(rawUser.address ?? ''),
      username: String(rawUser.username ?? ''),
      profilePicture: String(rawUser.profile_picture ?? ''),
      optimizedProfilePicture: String(rawUser.optimized_profile_picture ?? ''),
      pseudonym: String(rawUser.pseudonym ?? ''),
    },
    side: toProtoSide(item.side),
    size: String(item.size ?? ''),
    feeRateBps: String(item.fee_rate_bps ?? ''),
    price: String(item.price ?? ''),
    outcome: String(item.outcome ?? ''),
    outcomeIndex: Number(item.outcome_index ?? 0),
    transactionHash: String(item.transaction_hash ?? ''),
    timestamp: String(item.timestamp ?? ''),
  };
}

function toSdkBookParams(request: Record<string, unknown>): Array<{ token_id: string; side: SdkSide }> {
  const params = Array.isArray(request.params) ? request.params : [];
  return params.map((item) => {
    const p = item as Record<string, unknown>;
    return {
      token_id: String(p.tokenId ?? ''),
      side: toSdkSide((p.side as proto.Side | undefined) ?? proto.Side.SIDE_BUY),
    };
  });
}

function toSdkTradeParams(params: unknown): Record<string, string> | undefined {
  if (!params || typeof params !== 'object') {
    return undefined;
  }

  const p = params as Record<string, unknown>;
  const mapped: Record<string, string> = {};
  if (p.id) mapped.id = String(p.id);
  if (p.makerAddress) mapped.maker_address = String(p.makerAddress);
  if (p.market) mapped.market = String(p.market);
  if (p.assetId) mapped.asset_id = String(p.assetId);
  if (p.before) mapped.before = String(p.before);
  if (p.after) mapped.after = String(p.after);

  return Object.keys(mapped).length > 0 ? mapped : undefined;
}

function toSdkPriceHistoryInterval(
  interval: proto.PriceHistoryInterval | undefined,
): 'max' | '1w' | '1d' | '6h' | '1h' | undefined {
  switch (interval) {
    case proto.PriceHistoryInterval.PRICE_HISTORY_INTERVAL_MAX:
      return 'max';
    case proto.PriceHistoryInterval.PRICE_HISTORY_INTERVAL_ONE_WEEK:
      return '1w';
    case proto.PriceHistoryInterval.PRICE_HISTORY_INTERVAL_ONE_DAY:
      return '1d';
    case proto.PriceHistoryInterval.PRICE_HISTORY_INTERVAL_SIX_HOURS:
      return '6h';
    case proto.PriceHistoryInterval.PRICE_HISTORY_INTERVAL_ONE_HOUR:
      return '1h';
    default:
      return undefined;
  }
}

function getCases(): RpcCase[] {
  return [
    {
      method: 'getOk',
      responseType: proto.GetOkResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getOk(),
      toExpected: (sdkResult, request) => {
        parseGetOkRequest(request as proto.GetOkRequest);
        return toGetOkResponse(sdkResult);
      },
    },
    {
      method: 'getVersion',
      responseType: proto.GetVersionResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getVersion(),
      toExpected: (sdkResult) => ({ value: sdkResult }),
    },
    {
      method: 'getServerTime',
      responseType: proto.GetServerTimeResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getServerTime(),
      toExpected: (sdkResult) => ({ value: sdkResult }),
    },
    {
      method: 'getSamplingSimplifiedMarkets',
      responseType: proto.GetSamplingSimplifiedMarketsResponse,
      makeRequest: () => ({ nextCursor: 'MA==' }),
      sdkCall: (sdkClient, request) => sdkClient.getSamplingSimplifiedMarkets((request.nextCursor as string | undefined) ?? undefined),
      toExpected: (sdkResult) => ({ data: toProtoPaginationPayloadData(sdkResult) }),
    },
    {
      method: 'getSamplingMarkets',
      responseType: proto.GetSamplingMarketsResponse,
      makeRequest: () => ({ nextCursor: 'MA==' }),
      sdkCall: (sdkClient, request) => sdkClient.getSamplingMarkets((request.nextCursor as string | undefined) ?? undefined),
      toExpected: (sdkResult) => ({ data: toProtoPaginationPayloadData(sdkResult) }),
    },
    {
      method: 'getSimplifiedMarkets',
      responseType: proto.GetSimplifiedMarketsResponse,
      makeRequest: () => ({ nextCursor: 'MA==' }),
      sdkCall: (sdkClient, request) => sdkClient.getSimplifiedMarkets((request.nextCursor as string | undefined) ?? undefined),
      toExpected: (sdkResult) => ({ data: toProtoPaginationPayloadData(sdkResult) }),
    },
    {
      method: 'getMarkets',
      responseType: proto.GetMarketsResponse,
      makeRequest: () => ({ nextCursor: 'MA==' }),
      sdkCall: (sdkClient, request) => sdkClient.getMarkets((request.nextCursor as string | undefined) ?? undefined),
      toExpected: (sdkResult) => ({ data: toProtoPaginationPayloadData(sdkResult) }),
    },
    {
      method: 'getMarket',
      responseType: proto.GetMarketResponse,
      makeRequest: (env) => (env.conditionId ? { conditionId: env.conditionId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getMarket(String(request.conditionId)),
      toExpected: (sdkResult) => ({ data: toProtoStructData(sdkResult) }),
    },
    {
      method: 'getClobMarketInfo',
      responseType: proto.GetClobMarketInfoResponse,
      makeRequest: (env) => (env.conditionId ? { conditionId: env.conditionId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getClobMarketInfo(String(request.conditionId)),
      toExpected: (sdkResult) => ({ data: toCamelCaseObject(sdkResult) }),
    },
    {
      method: 'getOrderBook',
      responseType: proto.GetOrderBookResponse,
      makeRequest: (env) => (env.tokenId ? { tokenId: env.tokenId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getOrderBook(String(request.tokenId)),
      toExpected: (sdkResult) => ({ data: toCamelCaseObject(sdkResult) }),
    },
    {
      method: 'getOrderBooks',
      responseType: proto.GetOrderBooksResponse,
      makeRequest: (env) =>
        env.tokenId
          ? { params: [{ tokenId: env.tokenId, side: proto.Side.SIDE_BUY }] }
          : undefined,
      sdkCall: (sdkClient, request) => sdkClient.getOrderBooks(toSdkBookParams(request)),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toCamelCaseObject(item)),
      }),
    },
    {
      method: 'getTickSize',
      responseType: proto.GetTickSizeResponse,
      makeRequest: (env) => (env.tokenId ? { tokenId: env.tokenId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getTickSize(String(request.tokenId)),
      toExpected: (sdkResult) => ({ minimumTickSize: sdkResult }),
    },
    {
      method: 'getNegRisk',
      responseType: proto.GetNegRiskResponse,
      makeRequest: (env) => (env.tokenId ? { tokenId: env.tokenId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getNegRisk(String(request.tokenId)),
      toExpected: (sdkResult) => ({ value: sdkResult }),
    },
    {
      method: 'getFeeRateBps',
      responseType: proto.GetFeeRateBpsResponse,
      makeRequest: (env) => (env.tokenId ? { tokenId: env.tokenId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getFeeRateBps(String(request.tokenId)),
      toExpected: (sdkResult) => ({ value: sdkResult }),
    },
    {
      method: 'getFeeExponent',
      responseType: proto.GetFeeExponentResponse,
      makeRequest: (env) => (env.tokenId ? { tokenId: env.tokenId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getFeeExponent(String(request.tokenId)),
      toExpected: (sdkResult) => ({ value: sdkResult }),
    },
    {
      method: 'getOrderBookHash',
      responseType: proto.GetOrderBookHashResponse,
      makeRequest: () => ({
        orderbook: {
          market: 'test-market',
          assetId: 'test-asset',
          timestamp: '0',
          bids: [{ price: '0.45', size: '10' }],
          asks: [{ price: '0.55', size: '10' }],
          minOrderSize: '1',
          tickSize: '0.01',
          negRisk: false,
          hash: '',
        },
      }),
      sdkCall: (sdkClient, request) => {
        const orderbook = request.orderbook as Record<string, unknown>;
        return sdkClient.getOrderBookHash({
          market: String(orderbook.market),
          asset_id: String(orderbook.assetId),
          timestamp: String(orderbook.timestamp),
          bids: orderbook.bids as Array<{ price: string; size: string }>,
          asks: orderbook.asks as Array<{ price: string; size: string }>,
          min_order_size: String(orderbook.minOrderSize),
          tick_size: String(orderbook.tickSize),
          neg_risk: Boolean(orderbook.negRisk),
          hash: String(orderbook.hash ?? ''),
        });
      },
      toExpected: (sdkResult) => ({ value: sdkResult }),
    },
    {
      method: 'getMidpoint',
      responseType: proto.GetMidpointResponse,
      makeRequest: (env) => (env.tokenId ? { tokenId: env.tokenId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getMidpoint(String(request.tokenId)),
      toExpected: (sdkResult) => ({ data: toProtoStructData(sdkResult) }),
    },
    {
      method: 'getMidpoints',
      responseType: proto.GetMidpointsResponse,
      makeRequest: (env) =>
        env.tokenId
          ? { params: [{ tokenId: env.tokenId, side: proto.Side.SIDE_BUY }] }
          : undefined,
      sdkCall: (sdkClient, request) => sdkClient.getMidpoints(toSdkBookParams(request)),
      toExpected: (sdkResult) => ({ data: toProtoStructData(sdkResult) }),
    },
    {
      method: 'getPrice',
      responseType: proto.GetPriceResponse,
      makeRequest: (env) => (env.tokenId ? { tokenId: env.tokenId, side: 'BUY' } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getPrice(String(request.tokenId), String(request.side)),
      toExpected: (sdkResult) => ({ data: toProtoStructData(sdkResult) }),
    },
    {
      method: 'getPrices',
      responseType: proto.GetPricesResponse,
      makeRequest: (env) =>
        env.tokenId
          ? { params: [{ tokenId: env.tokenId, side: proto.Side.SIDE_BUY }] }
          : undefined,
      sdkCall: (sdkClient, request) => sdkClient.getPrices(toSdkBookParams(request)),
      toExpected: (sdkResult) => ({ data: toProtoStructData(sdkResult) }),
    },
    {
      method: 'getSpread',
      responseType: proto.GetSpreadResponse,
      makeRequest: (env) => (env.tokenId ? { tokenId: env.tokenId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getSpread(String(request.tokenId)),
      toExpected: (sdkResult) => ({ data: toProtoStructData(sdkResult) }),
    },
    {
      method: 'getSpreads',
      responseType: proto.GetSpreadsResponse,
      makeRequest: (env) =>
        env.tokenId
          ? { params: [{ tokenId: env.tokenId, side: proto.Side.SIDE_BUY }] }
          : undefined,
      sdkCall: (sdkClient, request) => sdkClient.getSpreads(toSdkBookParams(request)),
      toExpected: (sdkResult) => ({ data: toProtoStructData(sdkResult) }),
    },
    {
      method: 'getLastTradePrice',
      responseType: proto.GetLastTradePriceResponse,
      makeRequest: (env) => (env.tokenId ? { tokenId: env.tokenId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getLastTradePrice(String(request.tokenId)),
      toExpected: (sdkResult) => ({ data: toProtoStructData(sdkResult) }),
    },
    {
      method: 'getLastTradesPrices',
      responseType: proto.GetLastTradesPricesResponse,
      makeRequest: (env) =>
        env.tokenId
          ? { params: [{ tokenId: env.tokenId, side: proto.Side.SIDE_BUY }] }
          : undefined,
      sdkCall: (sdkClient, request) => sdkClient.getLastTradesPrices(toSdkBookParams(request)),
      toExpected: (sdkResult) => ({ data: toProtoStructData(sdkResult) }),
    },
    {
      method: 'getPricesHistory',
      responseType: proto.GetPricesHistoryResponse,
      makeRequest: () => ({ params: { interval: proto.PriceHistoryInterval.PRICE_HISTORY_INTERVAL_ONE_DAY } }),
      sdkCall: (sdkClient, request) => {
        const params = (request.params ?? {}) as Record<string, unknown>;
        const sdkParams: Record<string, unknown> = {};
        if (params.market) sdkParams.market = params.market;
        if (params.startTs !== undefined) sdkParams.startTs = params.startTs;
        if (params.endTs !== undefined) sdkParams.endTs = params.endTs;
        if (params.fidelity !== undefined) sdkParams.fidelity = params.fidelity;
        const interval = toSdkPriceHistoryInterval(params.interval as proto.PriceHistoryInterval | undefined);
        if (interval) sdkParams.interval = interval;
        return sdkClient.getPricesHistory(sdkParams);
      },
      toExpected: (sdkResult) => ({ items: sdkResult }),
    },
    {
      method: 'getApiKeys',
      responseType: proto.GetApiKeysResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getApiKeys(),
      toExpected: (sdkResult) => {
        const response = sdkResult as Record<string, unknown>;
        const apiKeys = Array.isArray(response.apiKeys)
          ? response.apiKeys
          : Array.isArray(response.api_keys)
            ? response.api_keys
            : [];
        return { apiKeys: apiKeys.map((item) => toCamelCaseObject(item)) };
      },
    },
    {
      method: 'getClosedOnlyMode',
      responseType: proto.GetClosedOnlyModeResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getClosedOnlyMode(),
      toExpected: (sdkResult) => {
        const response = sdkResult as Record<string, unknown>;
        return { closedOnly: Boolean(response.closed_only) };
      },
    },
    {
      method: 'getReadonlyApiKeys',
      responseType: proto.GetReadonlyApiKeysResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getReadonlyApiKeys(),
      toExpected: (sdkResult) => ({ items: sdkResult }),
    },
    {
      method: 'getOrder',
      responseType: proto.GetOrderResponse,
      makeRequest: (env) => (env.orderId ? { orderId: env.orderId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getOrder(String(request.orderId)),
      toExpected: (sdkResult) => ({ data: toCamelCaseObject(sdkResult) }),
    },
    {
      method: 'getTrades',
      responseType: proto.GetTradesResponse,
      makeRequest: () => ({ params: {}, onlyFirstPage: true, nextCursor: 'MA==' }),
      sdkCall: (sdkClient, request) =>
        sdkClient.getTrades(
          toSdkTradeParams(request.params),
          Boolean(request.onlyFirstPage),
          request.nextCursor ? String(request.nextCursor) : undefined,
        ),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toProtoTrade(item)),
      }),
    },
    {
      method: 'getTradesPaginated',
      responseType: proto.GetTradesPaginatedResponse,
      makeRequest: () => ({ params: {}, nextCursor: 'MA==' }),
      sdkCall: (sdkClient, request) =>
        sdkClient.getTradesPaginated(
          toSdkTradeParams(request.params),
          request.nextCursor ? String(request.nextCursor) : undefined,
        ),
      toExpected: (sdkResult) => {
        const response = sdkResult as Record<string, unknown>;
        const trades = Array.isArray(response.trades) ? response.trades : [];
        return {
          data: {
            trades: trades.map((item) => toProtoTrade(item)),
            nextCursor: String(response.next_cursor ?? response.nextCursor ?? ''),
            limit: Number(response.limit ?? 0),
            count: Number(response.count ?? 0),
          },
        };
      },
    },
    {
      method: 'getBuilderTrades',
      responseType: proto.GetBuilderTradesResponse,
      makeRequest: (env) => (env.builderCode ? { params: { builderCode: env.builderCode }, nextCursor: 'MA==' } : undefined),
      sdkCall: (sdkClient, request) => {
        const params = (request.params ?? {}) as Record<string, unknown>;
        return sdkClient.getBuilderTrades(
          {
            ...(toSdkTradeParams(params) ?? {}),
            builder_code: String(params.builderCode ?? ''),
          },
          request.nextCursor ? String(request.nextCursor) : undefined,
        );
      },
      toExpected: (sdkResult) => {
        const response = sdkResult as Record<string, unknown>;
        const trades = Array.isArray(response.trades) ? response.trades : [];
        return {
          data: {
            trades: trades.map((item) => toCamelCaseObject(item)),
            nextCursor: String(response.next_cursor ?? response.nextCursor ?? ''),
            limit: Number(response.limit ?? 0),
            count: Number(response.count ?? 0),
          },
        };
      },
    },
    {
      method: 'getNotifications',
      responseType: proto.GetNotificationsResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getNotifications(),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => {
          const value = item as Record<string, unknown>;
          return {
            type: Number(value.type ?? 0),
            owner: String(value.owner ?? ''),
            payload: toProtoStructData(toCamelCaseObject(value.payload)),
          };
        }),
      }),
    },
    {
      method: 'getBalanceAllowance',
      responseType: proto.GetBalanceAllowanceResponse,
      makeRequest: () => ({ params: { assetType: proto.AssetType.ASSET_TYPE_COLLATERAL } }),
      sdkCall: (sdkClient, request) => {
        const params = (request.params ?? {}) as Record<string, unknown>;
        return sdkClient.getBalanceAllowance({
          asset_type:
            params.assetType === proto.AssetType.ASSET_TYPE_CONDITIONAL
              ? SdkAssetType.CONDITIONAL
              : SdkAssetType.COLLATERAL,
          ...(params.tokenId ? { token_id: String(params.tokenId) } : {}),
        });
      },
      toExpected: (sdkResult) => ({ data: toCamelCaseObject(sdkResult) }),
    },
    {
      method: 'getOpenOrders',
      responseType: proto.GetOpenOrdersResponse,
      makeRequest: () => ({ params: {}, onlyFirstPage: true, nextCursor: 'MA==' }),
      sdkCall: (sdkClient, request) => {
        const params = (request.params ?? {}) as Record<string, unknown>;
        const sdkParams: Record<string, string> = {};
        if (params.id) sdkParams.id = String(params.id);
        if (params.market) sdkParams.market = String(params.market);
        if (params.assetId) sdkParams.asset_id = String(params.assetId);

        return sdkClient.getOpenOrders(
          Object.keys(sdkParams).length > 0 ? sdkParams : undefined,
          Boolean(request.onlyFirstPage),
          request.nextCursor ? String(request.nextCursor) : undefined,
        );
      },
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toCamelCaseObject(item)),
      }),
    },
    {
      method: 'getPreMigrationOrders',
      responseType: proto.GetPreMigrationOrdersResponse,
      makeRequest: () => ({ onlyFirstPage: true, nextCursor: 'MA==' }),
      sdkCall: (sdkClient, request) =>
        sdkClient.getPreMigrationOrders(
          Boolean(request.onlyFirstPage),
          request.nextCursor ? String(request.nextCursor) : undefined,
        ),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toCamelCaseObject(item)),
      }),
    },
    {
      method: 'getEarningsForUserForDay',
      responseType: proto.GetEarningsForUserForDayResponse,
      makeRequest: (env) => ({ date: env.rewardsDate }),
      sdkCall: (sdkClient, request) => sdkClient.getEarningsForUserForDay(String(request.date)),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toCamelCaseObject(item)),
      }),
    },
    {
      method: 'getTotalEarningsForUserForDay',
      responseType: proto.GetTotalEarningsForUserForDayResponse,
      makeRequest: (env) => ({ date: env.rewardsDate }),
      sdkCall: (sdkClient, request) => sdkClient.getTotalEarningsForUserForDay(String(request.date)),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toCamelCaseObject(item)),
      }),
    },
    {
      method: 'getUserEarningsAndMarketsConfig',
      responseType: proto.GetUserEarningsAndMarketsConfigResponse,
      makeRequest: (env) => ({ date: env.rewardsDate }),
      sdkCall: (sdkClient, request) =>
        sdkClient.getUserEarningsAndMarketsConfig(String(request.date), '', '', false),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toCamelCaseObject(item)),
      }),
    },
    {
      method: 'getRewardPercentages',
      responseType: proto.GetRewardPercentagesResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getRewardPercentages(),
      toExpected: (sdkResult) => ({ data: sdkResult }),
    },
    {
      method: 'getCurrentRewards',
      responseType: proto.GetCurrentRewardsResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getCurrentRewards(),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toCamelCaseObject(item)),
      }),
    },
    {
      method: 'getRawRewardsForMarket',
      responseType: proto.GetRawRewardsForMarketResponse,
      makeRequest: (env) => (env.conditionId ? { conditionId: env.conditionId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getRawRewardsForMarket(String(request.conditionId)),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toCamelCaseObject(item)),
      }),
    },
    {
      method: 'getBuilderApiKeys',
      responseType: proto.GetBuilderApiKeysResponse,
      makeRequest: () => ({}),
      sdkCall: (sdkClient) => sdkClient.getBuilderApiKeys(),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toCamelCaseObject(item)),
      }),
    },
    {
      method: 'getMarketTradesEvents',
      responseType: proto.GetMarketTradesEventsResponse,
      makeRequest: (env) => (env.conditionId ? { conditionId: env.conditionId } : undefined),
      sdkCall: (sdkClient, request) => sdkClient.getMarketTradesEvents(String(request.conditionId)),
      toExpected: (sdkResult) => ({
        items: (Array.isArray(sdkResult) ? sdkResult : []).map((item) => toProtoMarketTradeEvent(item)),
      }),
    },
  ];
}

async function grpcUnary(
  client: proto.PolymarketServiceClient,
  method: keyof proto.PolymarketServiceClient,
  request: Record<string, unknown>,
): Promise<unknown> {
  return await new Promise((resolve, reject) => {
    const unary = (client[method] as unknown as (req: Record<string, unknown>, cb: (err: grpc.ServiceError | null, res: unknown) => void) => void);
    unary.call(client, request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      resolve(response);
    });
  });
}

function assertGrpcErrorMatchesSdkError(grpcError: unknown, sdkError: unknown, address: string): void {
  const serviceError = grpcError as grpc.ServiceError;
  if (serviceError.code === grpcStatus.UNAVAILABLE) {
    handleE2EUnaryError(grpcError, address);
  }

  assert.ok(
    serviceError.code === grpcStatus.INTERNAL || serviceError.code === grpcStatus.INVALID_ARGUMENT,
    `Unexpected gRPC error code: ${serviceError.code}`,
  );

  if (sdkError instanceof Error) {
    assert.match(
      serviceError.details || serviceError.message,
      new RegExp(sdkError.message.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')),
    );
  }
}

const CASES = getCases();

export type GetRpcMethodName = (typeof CASES)[number]['method'];

export const GET_RPC_METHODS: GetRpcMethodName[] = CASES.map((item) => item.method);

export function registerGetRpcParityCase(method: GetRpcMethodName): void {
  const rpcCase = CASES.find((item) => item.method === method);
  if (!rpcCase) {
    throw new Error(`Unknown Get RPC method: ${method}`);
  }

  describe(`Polymarket gRPC ${method} parity (E2E)`, () => {
    const address = getE2EGrpcAddress();
    const env = readEnv();
    let grpcClient: proto.PolymarketServiceClient;

    before(() => {
      grpcClient = new proto.PolymarketServiceClient(address, grpc.credentials.createInsecure());
    });

    after(() => {
      grpcClient.close();
    });

    test(`${method} parity`, async (t) => {
      const request = rpcCase.makeRequest(env);
      if (!request) {
        t.skip(`Missing required E2E env vars for ${method}`);
      }

      const sdkClient = createPolymarketClient(loadRuntimeConfig().sdk);
      let sdkResult: unknown;
      try {
        sdkResult = await rpcCase.sdkCall(sdkClient, request!);
      } catch (sdkError) {
        try {
          await grpcUnary(grpcClient, method, request!);
          assert.fail(`${String(method)} succeeded on gRPC but failed on SDK`);
        } catch (grpcError) {
          assertGrpcErrorMatchesSdkError(grpcError, sdkError, address);
        }
        return;
      }

      const expected = rpcCase.toExpected(sdkResult, request!);
      const actual = await grpcUnary(grpcClient, method, request!).catch((error) =>
        handleE2EUnaryError(error, address),
      );

      assertEncodedEqual(
        rpcCase.responseType,
        actual,
        expected,
        `${String(method)} parity mismatch (SDK + gRPC are separate upstream calls)`,
      );
    });
  });
}
