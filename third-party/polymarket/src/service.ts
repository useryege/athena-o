import { AssetType as SdkAssetType, PriceHistoryInterval as SdkPriceHistoryInterval, Side as SdkSide } from '@polymarket/clob-client-v2';
import type { handleUnaryCall, sendUnaryData } from '@grpc/grpc-js';
import { status as grpcStatus } from '@grpc/grpc-js';

import {
  parseCancelAllRequest,
  parseGetOkRequest,
  toProtoStructData,
  toCancelAllResponse,
  toGetOkResponse,
} from './common.js';
import { getPolymarketClient } from './client.js';
import {
  AssetType,
  PriceHistoryInterval,
  Side,
  type PolymarketServiceServer,
  PolymarketServiceService,
} from './gen/polymarket/polymarket.js';
import { createServiceError, toGrpcError } from './grpc-error.js';

function executeUnary<TResponse>(
  callback: sendUnaryData<TResponse>,
  operation: () => Promise<TResponse> | TResponse,
): void {
  Promise.resolve()
    .then(operation)
    .then((response) => callback(null, response))
    .catch((error) => callback(toGrpcError(error)));
}

function requireField(value: string, fieldName: string): string {
  if (!value) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, `${fieldName} is required`);
  }

  return value;
}

function toSdkSide(side: Side): SdkSide {
  switch (side) {
    case Side.SIDE_BUY:
      return SdkSide.BUY;
    case Side.SIDE_SELL:
      return SdkSide.SELL;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'side must be SIDE_BUY or SIDE_SELL');
  }
}

function toProtoSide(side: unknown): Side {
  if (typeof side !== 'string') {
    return Side.SIDE_UNSPECIFIED;
  }

  if (side.toUpperCase() === SdkSide.BUY) {
    return Side.SIDE_BUY;
  }

  if (side.toUpperCase() === SdkSide.SELL) {
    return Side.SIDE_SELL;
  }

  return Side.SIDE_UNSPECIFIED;
}

function toSdkAssetType(assetType: AssetType): SdkAssetType {
  switch (assetType) {
    case AssetType.ASSET_TYPE_COLLATERAL:
      return SdkAssetType.COLLATERAL;
    case AssetType.ASSET_TYPE_CONDITIONAL:
      return SdkAssetType.CONDITIONAL;
    default:
      throw createServiceError(
        grpcStatus.INVALID_ARGUMENT,
        'params.asset_type must be ASSET_TYPE_COLLATERAL or ASSET_TYPE_CONDITIONAL',
      );
  }
}

function toSdkPriceHistoryInterval(interval: PriceHistoryInterval | undefined): SdkPriceHistoryInterval | undefined {
  switch (interval) {
    case undefined:
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_UNSPECIFIED:
      return undefined;
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_MAX:
      return SdkPriceHistoryInterval.MAX;
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_ONE_WEEK:
      return SdkPriceHistoryInterval.ONE_WEEK;
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_ONE_DAY:
      return SdkPriceHistoryInterval.ONE_DAY;
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_SIX_HOURS:
      return SdkPriceHistoryInterval.SIX_HOURS;
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_ONE_HOUR:
      return SdkPriceHistoryInterval.ONE_HOUR;
    default:
      return undefined;
  }
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

function optionalCursor(value: string | undefined): string | undefined {
  return value ? value : undefined;
}

function toSdkBookParams(params: Array<{ tokenId: string; side: Side }>): Array<{ token_id: string; side: SdkSide }> {
  return params.map((param, index) => ({
    token_id: requireField(param.tokenId, `params[${index}].token_id`),
    side: toSdkSide(param.side),
  }));
}

function toSdkTradeParams(params: {
  id?: string | undefined;
  makerAddress?: string | undefined;
  market?: string | undefined;
  assetId?: string | undefined;
  before?: string | undefined;
  after?: string | undefined;
} | undefined): Record<string, string> | undefined {
  if (!params) {
    return undefined;
  }

  const mapped: Record<string, string> = {};
  if (params.id) mapped.id = params.id;
  if (params.makerAddress) mapped.maker_address = params.makerAddress;
  if (params.market) mapped.market = params.market;
  if (params.assetId) mapped.asset_id = params.assetId;
  if (params.before) mapped.before = params.before;
  if (params.after) mapped.after = params.after;

  return Object.keys(mapped).length > 0 ? mapped : undefined;
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

function toProtoTrade(value: unknown): {
  id: string;
  takerOrderId: string;
  market: string;
  assetId: string;
  side: Side;
  size: string;
  feeRateBps: string;
  price: string;
  status: string;
  matchTime: string;
  lastUpdate: string;
  outcome: string;
  bucketIndex: number;
  owner: string;
  makerAddress: string;
  makerOrders: Array<{
    orderId: string;
    owner: string;
    makerAddress: string;
    matchedAmount: string;
    price: string;
    feeRateBps: string;
    assetId: string;
    outcome: string;
    side: Side;
  }>;
  transactionHash: string;
  traderSide: string;
} {
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

function toProtoMarketTradeEvent(value: unknown): {
  eventType: string;
  market: {
    conditionId: string;
    assetId: string;
    question: string;
    icon: string;
    slug: string;
  } | undefined;
  user: {
    address: string;
    username: string;
    profilePicture: string;
    optimizedProfilePicture: string;
    pseudonym: string;
  } | undefined;
  side: Side;
  size: string;
  feeRateBps: string;
  price: string;
  outcome: string;
  outcomeIndex: number;
  transactionHash: string;
  timestamp: string;
} {
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

function createUnimplementedHandlers(): Record<string, handleUnaryCall<unknown, unknown>> {
  const handlers: Record<string, handleUnaryCall<unknown, unknown>> = {};
  for (const methodName of Object.keys(PolymarketServiceService)) {
    handlers[methodName] = (_call, callback) => {
      callback(createServiceError(grpcStatus.UNIMPLEMENTED, `RPC ${methodName} is not implemented`));
    };
  }

  return handlers;
}

export function createPolymarketService(): PolymarketServiceServer {
  const client = getPolymarketClient();
  const handlers = createUnimplementedHandlers();

  return {
    ...(handlers as unknown as PolymarketServiceServer),
    health(call, callback) {
      executeUnary(callback, () => {
        void call;
        return { status: 'ok' };
      });
    },

    getOk(call, callback) {
      executeUnary(callback, async () => {
        parseGetOkRequest(call.request);
        const sdkResponse = await client.getOk();
        return toGetOkResponse(sdkResponse);
      });
    },

    getVersion(call, callback) {
      executeUnary(callback, async () => {
        void call;
        return { value: await client.getVersion() };
      });
    },

    getServerTime(call, callback) {
      executeUnary(callback, async () => {
        void call;
        return { value: await client.getServerTime() };
      });
    },

    getSamplingSimplifiedMarkets(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoPaginationPayloadData(
          await client.getSamplingSimplifiedMarkets(optionalCursor(call.request.nextCursor)),
        ),
      }));
    },

    getSamplingMarkets(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoPaginationPayloadData(await client.getSamplingMarkets(optionalCursor(call.request.nextCursor))),
      }));
    },

    getSimplifiedMarkets(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoPaginationPayloadData(
          await client.getSimplifiedMarkets(optionalCursor(call.request.nextCursor)),
        ),
      }));
    },

    getMarkets(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoPaginationPayloadData(await client.getMarkets(optionalCursor(call.request.nextCursor))),
      }));
    },

    getMarket(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoStructData(await client.getMarket(requireField(call.request.conditionId, 'condition_id'))),
      }));
    },

    getClobMarketInfo(call, callback) {
      executeUnary(callback, async () => ({
        data: toCamelCaseObject(await client.getClobMarketInfo(requireField(call.request.conditionId, 'condition_id'))),
      }));
    },

    getOrderBook(call, callback) {
      executeUnary(callback, async () => ({
        data: toCamelCaseObject(await client.getOrderBook(requireField(call.request.tokenId, 'token_id'))),
      }));
    },

    getOrderBooks(call, callback) {
      executeUnary(callback, async () => ({
        items: (await client.getOrderBooks(toSdkBookParams(call.request.params))).map((item) =>
          toCamelCaseObject(item),
        ),
      }));
    },

    getTickSize(call, callback) {
      executeUnary(callback, async () => ({
        minimumTickSize: await client.getTickSize(requireField(call.request.tokenId, 'token_id')),
      }));
    },

    getNegRisk(call, callback) {
      executeUnary(callback, async () => ({
        value: await client.getNegRisk(requireField(call.request.tokenId, 'token_id')),
      }));
    },

    getFeeRateBps(call, callback) {
      executeUnary(callback, async () => ({
        value: await client.getFeeRateBps(requireField(call.request.tokenId, 'token_id')),
      }));
    },

    getFeeExponent(call, callback) {
      executeUnary(callback, async () => ({
        value: await client.getFeeExponent(requireField(call.request.tokenId, 'token_id')),
      }));
    },

    getOrderBookHash(call, callback) {
      executeUnary(callback, async () => {
        const orderbook = call.request.orderbook;
        if (!orderbook) {
          throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'orderbook is required');
        }

        return {
          value: client.getOrderBookHash({
            market: orderbook.market,
            asset_id: orderbook.assetId,
            timestamp: orderbook.timestamp,
            bids: orderbook.bids,
            asks: orderbook.asks,
            min_order_size: orderbook.minOrderSize,
            tick_size: orderbook.tickSize,
            neg_risk: orderbook.negRisk,
            hash: orderbook.hash,
          }),
        };
      });
    },

    getMidpoint(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoStructData(await client.getMidpoint(requireField(call.request.tokenId, 'token_id'))),
      }));
    },

    getMidpoints(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoStructData(await client.getMidpoints(toSdkBookParams(call.request.params))),
      }));
    },

    getPrice(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoStructData(
          await client.getPrice(
            requireField(call.request.tokenId, 'token_id'),
            requireField(call.request.side, 'side'),
          ),
        ),
      }));
    },

    getPrices(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoStructData(await client.getPrices(toSdkBookParams(call.request.params))),
      }));
    },

    getSpread(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoStructData(await client.getSpread(requireField(call.request.tokenId, 'token_id'))),
      }));
    },

    getSpreads(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoStructData(await client.getSpreads(toSdkBookParams(call.request.params))),
      }));
    },

    getLastTradePrice(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoStructData(await client.getLastTradePrice(requireField(call.request.tokenId, 'token_id'))),
      }));
    },

    getLastTradesPrices(call, callback) {
      executeUnary(callback, async () => ({
        data: toProtoStructData(await client.getLastTradesPrices(toSdkBookParams(call.request.params))),
      }));
    },

    getPricesHistory(call, callback) {
      executeUnary(callback, async () => {
        const params = call.request.params;
        if (!params) {
          throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'params is required');
        }

        const sdkParams: {
          market?: string;
          startTs?: number;
          endTs?: number;
          fidelity?: number;
          interval?: SdkPriceHistoryInterval;
        } = {
          ...(params.market ? { market: params.market } : {}),
          ...(params.startTs !== undefined ? { startTs: params.startTs } : {}),
          ...(params.endTs !== undefined ? { endTs: params.endTs } : {}),
          ...(params.fidelity !== undefined ? { fidelity: params.fidelity } : {}),
        };
        const mappedInterval = toSdkPriceHistoryInterval(params.interval);
        if (mappedInterval !== undefined) {
          sdkParams.interval = mappedInterval;
        }

        return {
          items: await client.getPricesHistory(sdkParams),
        };
      });
    },

    getApiKeys(call, callback) {
      executeUnary(callback, async () => {
        void call;
        const response = (await client.getApiKeys()) as unknown as Record<string, unknown>;
        const apiKeys = Array.isArray(response.apiKeys)
          ? response.apiKeys
          : Array.isArray(response.api_keys)
            ? response.api_keys
            : [];

        return { apiKeys: apiKeys.map((item) => toCamelCaseObject(item)) };
      });
    },

    getClosedOnlyMode(call, callback) {
      executeUnary(callback, async () => {
        void call;
        const response = (await client.getClosedOnlyMode()) as unknown as Record<string, unknown>;
        return { closedOnly: Boolean(response.closed_only) };
      });
    },

    getReadonlyApiKeys(call, callback) {
      executeUnary(callback, async () => {
        void call;
        return { items: await client.getReadonlyApiKeys() };
      });
    },

    getOrder(call, callback) {
      executeUnary(callback, async () => ({
        data: toCamelCaseObject(await client.getOrder(requireField(call.request.orderId, 'order_id'))),
      }));
    },

    getTrades(call, callback) {
      executeUnary(callback, async () => ({
        items: (await client.getTrades(
          toSdkTradeParams(call.request.params),
          call.request.onlyFirstPage ?? false,
          optionalCursor(call.request.nextCursor),
        )).map((item) => toProtoTrade(item)),
      }));
    },

    getTradesPaginated(call, callback) {
      executeUnary(callback, async () => {
        const response = await client.getTradesPaginated(
          toSdkTradeParams(call.request.params),
          optionalCursor(call.request.nextCursor),
        );

        return {
          data: {
            trades: (Array.isArray(response.trades) ? response.trades : []).map((item) => toProtoTrade(item)),
            nextCursor: String(
              (response as unknown as Record<string, unknown>).next_cursor ?? response.next_cursor ?? '',
            ),
            limit: Number(response.limit ?? 0),
            count: Number(response.count ?? 0),
          },
        };
      });
    },

    getBuilderTrades(call, callback) {
      executeUnary(callback, async () => {
        const params = call.request.params;
        if (!params?.builderCode) {
          throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'params.builder_code is required');
        }

        const baseParams = toSdkTradeParams(params) ?? {};
        const response = await client.getBuilderTrades(
          {
            ...baseParams,
            builder_code: params.builderCode,
          },
          optionalCursor(call.request.nextCursor),
        );

        return {
          data: {
            trades: (Array.isArray(response.trades) ? response.trades : []).map((item) => toCamelCaseObject(item)),
            nextCursor: String(
              (response as unknown as Record<string, unknown>).next_cursor ?? response.next_cursor ?? '',
            ),
            limit: Number(response.limit ?? 0),
            count: Number(response.count ?? 0),
          },
        };
      });
    },

    getNotifications(call, callback) {
      executeUnary(callback, async () => {
        void call;
        const response = await client.getNotifications();
        return {
          items: response.map((item) => ({
            type: Number(item.type ?? 0),
            owner: String(item.owner ?? ''),
            payload: toProtoStructData(toCamelCaseObject(item.payload)),
          })),
        };
      });
    },

    getBalanceAllowance(call, callback) {
      executeUnary(callback, async () => {
        const params = call.request.params;
        if (!params) {
          throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'params is required');
        }

        return {
          data: toCamelCaseObject(
            await client.getBalanceAllowance({
              asset_type: toSdkAssetType(params.assetType),
              ...(params.tokenId ? { token_id: params.tokenId } : {}),
            }),
          ),
        };
      });
    },

    getOpenOrders(call, callback) {
      executeUnary(callback, async () => ({
        items: (
          await client.getOpenOrders(
            (() => {
              const params = call.request.params;
              if (!params) return undefined;
              const mapped: Record<string, string> = {};
              if (params.id) mapped.id = params.id;
              if (params.market) mapped.market = params.market;
              if (params.assetId) mapped.asset_id = params.assetId;
              return Object.keys(mapped).length > 0 ? mapped : undefined;
            })(),
            call.request.onlyFirstPage ?? false,
            optionalCursor(call.request.nextCursor),
          )
        ).map((item) => toCamelCaseObject(item)),
      }));
    },

    getPreMigrationOrders(call, callback) {
      executeUnary(callback, async () => ({
        items: (
          await client.getPreMigrationOrders(
            call.request.onlyFirstPage ?? false,
            optionalCursor(call.request.nextCursor),
          )
        ).map((item) => toCamelCaseObject(item)),
      }));
    },

    getEarningsForUserForDay(call, callback) {
      executeUnary(callback, async () => ({
        items: (await client.getEarningsForUserForDay(requireField(call.request.date, 'date'))).map((item) =>
          toCamelCaseObject(item),
        ),
      }));
    },

    getTotalEarningsForUserForDay(call, callback) {
      executeUnary(callback, async () => ({
        items: (await client.getTotalEarningsForUserForDay(requireField(call.request.date, 'date'))).map((item) =>
          toCamelCaseObject(item),
        ),
      }));
    },

    getUserEarningsAndMarketsConfig(call, callback) {
      executeUnary(callback, async () => ({
        items: (
          await client.getUserEarningsAndMarketsConfig(
            requireField(call.request.date, 'date'),
            call.request.orderBy ?? '',
            call.request.position ?? '',
            call.request.noCompetition ?? false,
          )
        ).map((item) => toCamelCaseObject(item)),
      }));
    },

    getRewardPercentages(call, callback) {
      executeUnary(callback, async () => {
        void call;
        return { data: await client.getRewardPercentages() };
      });
    },

    getCurrentRewards(call, callback) {
      executeUnary(callback, async () => {
        void call;
        return { items: (await client.getCurrentRewards()).map((item) => toCamelCaseObject(item)) };
      });
    },

    getRawRewardsForMarket(call, callback) {
      executeUnary(callback, async () => ({
        items: (await client.getRawRewardsForMarket(requireField(call.request.conditionId, 'condition_id'))).map(
          (item) => toCamelCaseObject(item),
        ),
      }));
    },

    getBuilderApiKeys(call, callback) {
      executeUnary(callback, async () => {
        void call;
        return { items: (await client.getBuilderApiKeys()).map((item) => toCamelCaseObject(item)) };
      });
    },

    getMarketTradesEvents(call, callback) {
      executeUnary(callback, async () => ({
        items: (await client.getMarketTradesEvents(requireField(call.request.conditionId, 'condition_id'))).map(
          (item) => toProtoMarketTradeEvent(item),
        ),
      }));
    },

    async cancelAll(call, callback) {
      try {
        parseCancelAllRequest(call.request);
        const sdkResponse = await client.cancelAll();
        callback(null, toCancelAllResponse(sdkResponse));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },
  };
}
