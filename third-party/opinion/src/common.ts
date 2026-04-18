import {
  OrderSide,
  OrderType,
  TopicSortType,
  TopicStatusFilter,
  TopicType,
  type Client,
  type FeeRateSettings,
  type PlaceOrderDataInput,
} from '@opinion-labs/opinion-clob-sdk';
import { status as grpcStatus } from '@grpc/grpc-js';

import {
  CancelAllOrdersResponse,
  CancelOrderApiResponse,
  CancelOrderBatchItem,
  CancelOrdersBatchResponse,
  ChildMarket,
  GetFeeRatesResponse,
  GetLatestPriceResponse,
  GetMarketResponse,
  GetMarketsResponse,
  GetOrderbookResponse,
  GetPriceHistoryResponse,
  GetQuoteTokensResponse,
  Market,
  MarketStatusFilter,
  MarketSortBy,
  MarketTopicType,
  OrderApiData,
  OrderbookLevel,
  OrderTradeApiData,
  PlaceOrderBatchItem,
  PlaceOrderData,
  PlaceOrderResponse,
  PlaceOrdersBatchResponse,
  PriceHistoryInterval,
  PricePoint,
  QuoteToken,
  SdkOrderSide,
  SdkOrderType,
  type GetCategoricalMarketRequest,
  type GetFeeRatesRequest,
  type GetLatestPriceRequest,
  type GetMarketRequest,
  type GetMarketsRequest,
  type GetMarketBySlugRequest,
  type GetOrderbookRequest,
  type GetPriceHistoryRequest,
  type GetQuoteTokensRequest,
  type CancelAllOrdersRequest,
  type CancelOrderRequest,
  type CancelOrdersBatchRequest,
  type PlaceOrderRequest,
  type PlaceOrdersBatchRequest,
} from './gen/opinion/opinion.js';
import { Struct } from './gen/google/protobuf/struct.js';
import { createServiceError } from './grpc-error.js';

// ─── SDK-aligned result shapes (mirror opinion-clob-sdk Client) ─────────────

export interface SdkMarketListResult {
  total: number;
  list: Record<string, unknown>[];
}

export interface SdkMarketDetailResult {
  data?: Record<string, unknown>;
}

export interface SdkQuoteTokenListResult {
  total: number;
  list: Record<string, unknown>[];
}

export interface SdkApiResponse<T> {
  errno: number;
  errmsg: string;
  result: T;
}

// ─── GetMarkets: proto → SDK query ──────────────────────────────────────────

export type GetMarketsQuery = Parameters<Client['getMarkets']>[0];

export function parseGetMarketsRequest(request: GetMarketsRequest): GetMarketsQuery {
  const query: GetMarketsQuery = {};

  if (request.topicType !== undefined && request.topicType !== MarketTopicType.MARKET_TOPIC_TYPE_UNSPECIFIED) {
    query.topicType = toSdkTopicType(request.topicType);
  }

  if (request.page !== undefined) {
    const page = request.page;
    if (!Number.isInteger(page) || page < 1) {
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'page must be >= 1');
    }
    query.page = page;
  }

  if (request.limit !== undefined) {
    const limit = request.limit;
    if (!Number.isInteger(limit) || limit < 1 || limit > 20) {
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'limit must be between 1 and 20');
    }
    query.limit = limit;
  }

  const status = toSdkStatusFilterOptional(request.status);
  if (status !== undefined) {
    query.status = status;
  }

  if (request.sortBy !== undefined && request.sortBy !== MarketSortBy.MARKET_SORT_BY_UNSPECIFIED) {
    query.sortBy = toSdkSortBy(request.sortBy);
  }

  return query;
}

function toSdkTopicType(value: MarketTopicType): TopicType {
  switch (value) {
    case MarketTopicType.MARKET_TOPIC_TYPE_BINARY:
      return TopicType.BINARY;
    case MarketTopicType.MARKET_TOPIC_TYPE_CATEGORICAL:
      return TopicType.CATEGORICAL;
    case MarketTopicType.MARKET_TOPIC_TYPE_ALL:
      return TopicType.ALL;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported topicType: ${value}`);
  }
}

function toSdkStatusFilterOptional(
  value: MarketStatusFilter | undefined,
): TopicStatusFilter | undefined {
  if (value === undefined || value === MarketStatusFilter.MARKET_STATUS_FILTER_UNSPECIFIED) {
    return undefined;
  }
  if (value === MarketStatusFilter.MARKET_STATUS_FILTER_ALL) {
    return undefined;
  }
  switch (value) {
    case MarketStatusFilter.MARKET_STATUS_FILTER_ACTIVATED:
      return TopicStatusFilter.ACTIVATED;
    case MarketStatusFilter.MARKET_STATUS_FILTER_RESOLVED:
      return TopicStatusFilter.RESOLVED;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported status filter: ${value}`);
  }
}

function toSdkSortBy(value: MarketSortBy): TopicSortType {
  switch (value) {
    case MarketSortBy.MARKET_SORT_BY_TIME_DESC:
      return TopicSortType.BY_TIME_DESC;
    case MarketSortBy.MARKET_SORT_BY_CUTOFF_TIME_ASC:
      return TopicSortType.BY_CUTOFF_TIME_ASC;
    case MarketSortBy.MARKET_SORT_BY_VOLUME_DESC:
      return TopicSortType.BY_VOLUME_DESC;
    case MarketSortBy.MARKET_SORT_BY_VOLUME_ASC:
      return TopicSortType.BY_VOLUME_ASC;
    case MarketSortBy.MARKET_SORT_BY_VOLUME_24H_DESC:
      return TopicSortType.BY_VOLUME_24H_DESC;
    case MarketSortBy.MARKET_SORT_BY_VOLUME_24H_ASC:
      return TopicSortType.BY_VOLUME_24H_ASC;
    case MarketSortBy.MARKET_SORT_BY_VOLUME_7D_DESC:
      return TopicSortType.BY_VOLUME_7D_DESC;
    case MarketSortBy.MARKET_SORT_BY_VOLUME_7D_ASC:
      return TopicSortType.BY_VOLUME_7D_ASC;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported sortBy: ${value}`);
  }
}

// ─── Market id parsing (string → SDK number, safe integer only) ────────────

export function parseMarketIdForSdk(request: GetMarketRequest | GetCategoricalMarketRequest): number {
  return parseDecimalStringToSafePositiveInt(request.marketId, 'marketId');
}

function parseDecimalStringToSafePositiveInt(raw: string, field: string): number {
  const trimmed = raw.trim();
  if (!/^\d+$/.test(trimmed)) {
    throw createServiceError(
      grpcStatus.INVALID_ARGUMENT,
      `${field} must be a non-empty decimal integer string`,
    );
  }
  const n = Number(trimmed);
  if (!Number.isSafeInteger(n) || n < 1) {
    throw createServiceError(
      grpcStatus.INVALID_ARGUMENT,
      `${field} must be a positive integer within JS safe integer range`,
    );
  }
  return n;
}

export function parseGetMarketRequest(request: GetMarketRequest): { marketId: number; useCache: boolean } {
  return {
    marketId: parseMarketIdForSdk(request),
    useCache: request.useCache ?? true,
  };
}

export function parseGetCategoricalMarketRequest(
  request: GetCategoricalMarketRequest,
): number {
  return parseMarketIdForSdk(request);
}

export function parseGetMarketBySlugRequest(request: GetMarketBySlugRequest): string {
  const slug = request.slug?.trim();
  if (!slug) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'slug is required');
  }
  return slug;
}

export function parseGetQuoteTokensRequest(request: GetQuoteTokensRequest): boolean {
  return request.useCache ?? true;
}

// ─── SDK result → proto responses ───────────────────────────────────────────

export function toGetMarketsResponse(response: SdkApiResponse<SdkMarketListResult>): GetMarketsResponse {
  const result = response.result ?? { total: 0, list: [] };
  const list = Array.isArray(result.list) ? result.list : [];
  return GetMarketsResponse.fromPartial({
    errno: 0,
    errmsg: '',
    total: typeof result.total === 'number' ? result.total : 0,
    list: list.filter(isRecord).map(mapSdkMarketRecordToProto),
  });
}

export function toGetMarketDetailResponse(response: SdkApiResponse<SdkMarketDetailResult>): GetMarketResponse {
  const data = response.result?.data;
  const marketData = data && isRecord(data) ? mapSdkMarketRecordToProto(data) : undefined;
  return GetMarketResponse.fromPartial({
    errno: 0,
    errmsg: '',
    data: marketData,
  });
}

export function toGetQuoteTokensResponse(
  response: SdkApiResponse<SdkQuoteTokenListResult>,
): GetQuoteTokensResponse {
  const result = response.result ?? { total: 0, list: [] };
  const list = Array.isArray(result.list) ? result.list : [];
  return GetQuoteTokensResponse.fromPartial({
    errno: 0,
    errmsg: '',
    total: typeof result.total === 'number' ? result.total : 0,
    list: list.filter(isRecord).map(mapSdkQuoteTokenRecordToProto),
  });
}

function mapSdkMarketRecordToProto(raw: Record<string, unknown>): Market {
  const childRaw = raw.childMarkets;
  const children: ChildMarket[] = Array.isArray(childRaw)
    ? childRaw.filter(isRecord).map(mapSdkChildMarketRecordToProto)
    : [];

  let collection: Market['collection'];
  const col = raw.collection;
  if (col !== undefined && col !== null && typeof col === 'object' && !Array.isArray(col)) {
    try {
      collection = Struct.wrap(col as { [key: string]: unknown });
    } catch {
      collection = undefined;
    }
  }

  return Market.fromPartial({
    marketId: scalarString(raw.marketId),
    marketTitle: scalarString(raw.marketTitle),
    slug: scalarString(raw.slug),
    conditionId: scalarString(raw.conditionId),
    chainId: scalarString(raw.chainId),
    quoteToken: scalarString(raw.quoteToken),
    status: scalarInt32(raw.status),
    statusEnum: scalarString(raw.statusEnum),
    createdAt: scalarString(raw.createdAt),
    cutoffAt: scalarString(raw.cutoffAt),
    resolvedAt: scalarString(raw.resolvedAt),
    yesTokenId: scalarString(raw.yesTokenId),
    noTokenId: scalarString(raw.noTokenId),
    resultTokenId: scalarString(raw.resultTokenId),
    yesLabel: scalarString(raw.yesLabel),
    noLabel: scalarString(raw.noLabel),
    volume: scalarString(raw.volume),
    isIncentivized: raw.isIncentivized === true,
    childMarkets: children,
    questionId: scalarString(raw.questionId),
    rules: typeof raw.rules === 'string' ? raw.rules : scalarString(raw.rules),
    collection,
  });
}

function mapSdkChildMarketRecordToProto(raw: Record<string, unknown>): ChildMarket {
  return ChildMarket.fromPartial({
    marketId: scalarString(raw.marketId),
    marketTitle: scalarString(raw.marketTitle),
    slug: scalarString(raw.slug),
    conditionId: scalarString(raw.conditionId),
    chainId: scalarString(raw.chainId),
    quoteToken: scalarString(raw.quoteToken),
    status: scalarInt32(raw.status),
    statusEnum: scalarString(raw.statusEnum),
    createdAt: scalarString(raw.createdAt),
    cutoffAt: scalarString(raw.cutoffAt),
    resolvedAt: scalarString(raw.resolvedAt),
    yesTokenId: scalarString(raw.yesTokenId),
    noTokenId: scalarString(raw.noTokenId),
    resultTokenId: scalarString(raw.resultTokenId),
    yesLabel: scalarString(raw.yesLabel),
    noLabel: scalarString(raw.noLabel),
    volume: scalarString(raw.volume),
    questionId: scalarString(raw.questionId),
    rules: typeof raw.rules === 'string' ? raw.rules : scalarString(raw.rules),
  });
}

function mapSdkQuoteTokenRecordToProto(raw: Record<string, unknown>): QuoteToken {
  return QuoteToken.fromPartial({
    chainId: scalarString(raw.chainId),
    createdAt: scalarString(raw.createdAt),
    ctfExchangeAddress: scalarString(raw.ctfExchangeAddress),
    decimal: scalarInt32(raw.decimal),
    id: scalarString(raw.id),
    quoteTokenAddress: scalarString(raw.quoteTokenAddress),
    quoteTokenName: scalarString(raw.quoteTokenName),
    symbol: scalarString(raw.symbol),
  });
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function scalarString(value: unknown): string {
  if (value === null || value === undefined) {
    return '';
  }
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }
  if (typeof value === 'boolean') {
    return value ? 'true' : 'false';
  }
  return String(value);
}

function scalarInt32(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return Math.trunc(value);
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      return Math.trunc(parsed);
    }
  }
  return 0;
}

// ─── Token / orderbook / trading: proto → SDK ───────────────────────────────

export function parseTokenIdString(raw: string | undefined, field: string): string {
  const tokenId = raw?.trim();
  if (!tokenId) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, `${field} is required`);
  }
  return tokenId;
}

export function parseGetOrderbookRequest(request: GetOrderbookRequest): string {
  return parseTokenIdString(request.tokenId, 'tokenId');
}

export function parseGetLatestPriceRequest(request: GetLatestPriceRequest): string {
  return parseTokenIdString(request.tokenId, 'tokenId');
}

export function parseGetFeeRatesRequest(request: GetFeeRatesRequest): string {
  return parseTokenIdString(request.tokenId, 'tokenId');
}

export type GetPriceHistorySdkOptions = {
  interval?: '1m' | '1h' | '1d' | '1w' | 'max';
  startAt?: number;
  endAt?: number;
};

export function parseGetPriceHistoryRequest(
  request: GetPriceHistoryRequest,
): { tokenId: string; options: GetPriceHistorySdkOptions } {
  const tokenId = parseTokenIdString(request.tokenId, 'tokenId');
  const options: GetPriceHistorySdkOptions = {};

  if (request.interval !== undefined && request.interval !== PriceHistoryInterval.PRICE_HISTORY_INTERVAL_UNSPECIFIED) {
    options.interval = priceHistoryIntervalToSdk(request.interval);
  }

  if (request.startAt !== undefined && request.startAt !== '') {
    const n = Number(request.startAt);
    if (!Number.isFinite(n)) {
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'startAt must be a finite number');
    }
    options.startAt = Math.trunc(n);
  }

  if (request.endAt !== undefined && request.endAt !== '') {
    const n = Number(request.endAt);
    if (!Number.isFinite(n)) {
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'endAt must be a finite number');
    }
    options.endAt = Math.trunc(n);
  }

  return { tokenId, options };
}

function priceHistoryIntervalToSdk(
  value: PriceHistoryInterval,
): '1m' | '1h' | '1d' | '1w' | 'max' {
  switch (value) {
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_1M:
      return '1m';
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_1H:
      return '1h';
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_1D:
      return '1d';
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_1W:
      return '1w';
    case PriceHistoryInterval.PRICE_HISTORY_INTERVAL_MAX:
      return 'max';
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported price history interval: ${value}`);
  }
}

function protoSdkOrderSideToSdk(value: SdkOrderSide): OrderSide {
  switch (value) {
    case SdkOrderSide.SDK_ORDER_SIDE_BUY:
      return OrderSide.BUY;
    case SdkOrderSide.SDK_ORDER_SIDE_SELL:
      return OrderSide.SELL;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported order side: ${value}`);
  }
}

function protoSdkOrderTypeToSdk(value: SdkOrderType): OrderType {
  switch (value) {
    case SdkOrderType.SDK_ORDER_TYPE_MARKET_ORDER:
      return OrderType.MARKET_ORDER;
    case SdkOrderType.SDK_ORDER_TYPE_LIMIT_ORDER:
      return OrderType.LIMIT_ORDER;
    default:
      throw createServiceError(
        grpcStatus.INVALID_ARGUMENT,
        'orderType must be MARKET_ORDER or LIMIT_ORDER',
      );
  }
}

export function parsePlaceOrderData(data: PlaceOrderData | undefined): PlaceOrderDataInput {
  if (!data) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'data is required');
  }
  if (!Number.isInteger(data.marketId) || data.marketId < 1) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'marketId must be a positive integer');
  }
  const tokenId = data.tokenId?.trim();
  if (!tokenId) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'tokenId is required');
  }
  const price = typeof data.price === 'string' ? data.price.trim() : '';
  if (price === '') {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'price is required');
  }

  const input: PlaceOrderDataInput = {
    marketId: data.marketId,
    tokenId,
    side: protoSdkOrderSideToSdk(data.side),
    orderType: protoSdkOrderTypeToSdk(data.orderType),
    price,
  };

  if (data.makerAmountInQuoteToken !== undefined && data.makerAmountInQuoteToken !== '') {
    input.makerAmountInQuoteToken = data.makerAmountInQuoteToken;
  }
  if (data.makerAmountInBaseToken !== undefined && data.makerAmountInBaseToken !== '') {
    input.makerAmountInBaseToken = data.makerAmountInBaseToken;
  }

  return input;
}

export function parsePlaceOrderRequest(request: PlaceOrderRequest): {
  data: PlaceOrderDataInput;
  checkApproval: boolean;
} {
  return {
    data: parsePlaceOrderData(request.data),
    checkApproval: request.checkApproval ?? false,
  };
}

export function parsePlaceOrdersBatchRequest(request: PlaceOrdersBatchRequest): {
  orders: PlaceOrderDataInput[];
  checkApproval: boolean;
} {
  const orders = request.orders ?? [];
  if (orders.length === 0) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'orders must not be empty');
  }
  return {
    orders: orders.map((o) => parsePlaceOrderData(o)),
    checkApproval: request.checkApproval ?? false,
  };
}

export function parseCancelOrderRequest(request: CancelOrderRequest): string {
  const orderId = request.orderId?.trim();
  if (!orderId) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'orderId is required');
  }
  return orderId;
}

export function parseCancelOrdersBatchRequest(request: CancelOrdersBatchRequest): string[] {
  const ids = request.orderIds ?? [];
  if (ids.length === 0) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'orderIds must not be empty');
  }
  return ids.map((id, i) => {
    const trimmed = id?.trim();
    if (!trimmed) {
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `orderIds[${i}] is empty`);
    }
    return trimmed;
  });
}

export function parseCancelAllOrdersRequest(request: CancelAllOrdersRequest): {
  marketId?: number;
  side?: OrderSide;
} {
  const options: { marketId?: number; side?: OrderSide } = {};
  if (request.marketId !== undefined) {
    const mid = request.marketId;
    if (!Number.isInteger(mid) || mid < 1) {
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'marketId must be a positive integer when set');
    }
    options.marketId = mid;
  }
  if (request.side !== undefined) {
    options.side = protoSdkOrderSideToSdk(request.side);
  }
  return options;
}

// ─── SDK → proto (orderbook, prices, orders) ───────────────────────────────

export function toGetOrderbookResponse(
  response: SdkApiResponse<Record<string, unknown>>,
): GetOrderbookResponse {
  const result = response.result ?? {};
  const asks = Array.isArray(result.asks) ? result.asks : [];
  const bids = Array.isArray(result.bids) ? result.bids : [];
  return GetOrderbookResponse.fromPartial({
    errno: 0,
    errmsg: '',
    asks: asks.filter(isRecord).map(mapOrderbookLevelRecord),
    bids: bids.filter(isRecord).map(mapOrderbookLevelRecord),
    market: optionalString(result.market),
    timestamp: int64LikeToProtoString(result.timestamp),
    tokenId: optionalString(result.tokenId),
  });
}

export function toGetLatestPriceResponse(
  response: SdkApiResponse<Record<string, unknown>>,
): GetLatestPriceResponse {
  const result = response.result ?? {};
  return GetLatestPriceResponse.fromPartial({
    errno: 0,
    errmsg: '',
    price: optionalString(result.price),
    side: optionalString(result.side),
    size: optionalString(result.size),
    timestamp: int64LikeToProtoString(result.timestamp),
    tokenId: optionalString(result.tokenId),
  });
}

export function toGetPriceHistoryResponse(
  response: SdkApiResponse<{ history?: Record<string, unknown>[] }>,
): GetPriceHistoryResponse {
  const result = response.result ?? {};
  const history = Array.isArray(result.history) ? result.history : [];
  return GetPriceHistoryResponse.fromPartial({
    errno: 0,
    errmsg: '',
    history: history.filter(isRecord).map(mapPricePointRecord),
  });
}

export function toGetFeeRatesResponse(settings: FeeRateSettings): GetFeeRatesResponse {
  return GetFeeRatesResponse.fromPartial({
    makerMaxFeeRate: settings.makerMaxFeeRate,
    takerMaxFeeRate: settings.takerMaxFeeRate,
    enabled: settings.enabled,
  });
}

export function toPlaceOrderResponse(
  response: SdkApiResponse<{ orderData?: Record<string, unknown> }>,
): PlaceOrderResponse {
  const od = response.result?.orderData;
  return PlaceOrderResponse.fromPartial({
    errno: response.errno,
    errmsg: response.errmsg ?? '',
    orderData: od && isRecord(od) ? mapOrderRecordToOrderApiData(od) : undefined,
  });
}

export function toCancelOrderApiResponse(
  response: SdkApiResponse<{ result?: boolean }>,
): CancelOrderApiResponse {
  return CancelOrderApiResponse.fromPartial({
    errno: response.errno,
    errmsg: response.errmsg ?? '',
    result: response.result?.result,
  });
}

export function toPlaceOrdersBatchResponse(
  items: Array<{ index: number; success: boolean; result?: unknown; error?: string }>,
): PlaceOrdersBatchResponse {
  return PlaceOrdersBatchResponse.fromPartial({
    items: items.map(toPlaceOrderBatchItem),
  });
}

function toPlaceOrderBatchItem(item: {
  index: number;
  success: boolean;
  result?: unknown;
  error?: string;
}): PlaceOrderBatchItem {
  if (item.success && item.result) {
    return PlaceOrderBatchItem.fromPartial({
      index: item.index,
      success: true,
      result: toPlaceOrderResponse(item.result as SdkApiResponse<{ orderData?: Record<string, unknown> }>),
    });
  }
  return PlaceOrderBatchItem.fromPartial({
    index: item.index,
    success: false,
    error: item.error,
  });
}

export function toCancelOrdersBatchResponse(
  items: Array<{ index: number; success: boolean; result?: unknown; error?: string }>,
): CancelOrdersBatchResponse {
  return CancelOrdersBatchResponse.fromPartial({
    items: items.map(toCancelOrderBatchItem),
  });
}

function toCancelOrderBatchItem(item: {
  index: number;
  success: boolean;
  result?: unknown;
  error?: string;
}): CancelOrderBatchItem {
  if (item.success && item.result) {
    return CancelOrderBatchItem.fromPartial({
      index: item.index,
      success: true,
      result: toCancelOrderApiResponse(item.result as SdkApiResponse<{ result?: boolean }>),
    });
  }
  return CancelOrderBatchItem.fromPartial({
    index: item.index,
    success: false,
    error: item.error,
  });
}

export function toCancelAllOrdersResponse(result: {
  totalOrders: number;
  cancelled: number;
  failed: number;
  results: Array<{ index: number; success: boolean; result?: unknown; error?: string }>;
}): CancelAllOrdersResponse {
  return CancelAllOrdersResponse.fromPartial({
    totalOrders: result.totalOrders,
    cancelled: result.cancelled,
    failed: result.failed,
    results: result.results.map(toCancelOrderBatchItem),
  });
}

function mapOrderbookLevelRecord(raw: Record<string, unknown>): OrderbookLevel {
  return OrderbookLevel.fromPartial({
    price: optionalString(raw.price),
    size: optionalString(raw.size),
  });
}

function mapPricePointRecord(raw: Record<string, unknown>): PricePoint {
  return PricePoint.fromPartial({
    p: optionalString(raw.p),
    t: int64LikeToProtoString(raw.t),
  });
}

function optionalString(value: unknown): string | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }
  return String(value);
}

function int64LikeToProtoString(value: unknown): string | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }
  if (typeof value === 'bigint') {
    return value.toString();
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(Math.trunc(value));
  }
  if (typeof value === 'string' && value.trim() !== '') {
    return value.trim();
  }
  return undefined;
}

function mapOrderRecordToOrderApiData(raw: Record<string, unknown>): OrderApiData {
  const tradesRaw = raw.trades;
  const trades: OrderTradeApiData[] = Array.isArray(tradesRaw)
    ? tradesRaw.filter(isRecord).map(mapOrderTradeRecordToProto)
    : [];

  return OrderApiData.fromPartial({
    createdAt: int64LikeToProtoString(raw.createdAt),
    expiresAt: int64LikeToProtoString(raw.expiresAt),
    filledAmount: optionalString(raw.filledAmount),
    filledShares: optionalString(raw.filledShares),
    marketId: optionalInt32(raw.marketId),
    marketTitle: optionalString(raw.marketTitle),
    orderAmount: optionalString(raw.orderAmount),
    orderId: optionalString(raw.orderId),
    orderShares: optionalString(raw.orderShares),
    outcome: optionalString(raw.outcome),
    outcomeSide: optionalInt32(raw.outcomeSide),
    outcomeSideEnum: optionalString(raw.outcomeSideEnum),
    price: optionalString(raw.price),
    profit: optionalString(raw.profit),
    quoteToken: optionalString(raw.quoteToken),
    rootMarketId: optionalInt32(raw.rootMarketId),
    rootMarketTitle: optionalString(raw.rootMarketTitle),
    side: optionalInt32(raw.side),
    sideEnum: optionalString(raw.sideEnum),
    status: optionalInt32(raw.status),
    statusEnum: optionalString(raw.statusEnum),
    trades,
    tradingMethod: optionalInt32(raw.tradingMethod),
    tradingMethodEnum: optionalString(raw.tradingMethodEnum),
    transNo: optionalString(raw.transNo),
  });
}

function optionalInt32(value: unknown): number | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return Math.trunc(value);
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const n = Number(value);
    if (Number.isFinite(n)) {
      return Math.trunc(n);
    }
  }
  return undefined;
}

function mapOrderTradeRecordToProto(raw: Record<string, unknown>): OrderTradeApiData {
  return OrderTradeApiData.fromPartial({
    amount: optionalString(raw.amount),
    chainId: optionalString(raw.chainId),
    createdAt: int64LikeToProtoString(raw.createdAt),
    fee: typeof raw.fee === 'number' && Number.isFinite(raw.fee) ? raw.fee : undefined,
    feeFormatted: optionalString(raw.feeFormatted),
    marketId: optionalInt32(raw.marketId),
    marketTitle: optionalString(raw.marketTitle),
    orderNo: optionalString(raw.orderNo),
    outcome: optionalString(raw.outcome),
    outcomeSide: optionalInt32(raw.outcomeSide),
    outcomeSideEnum: optionalString(raw.outcomeSideEnum),
    price: optionalString(raw.price),
    profit: optionalString(raw.profit),
    quoteToken: optionalString(raw.quoteToken),
    quoteTokenUsdPrice: optionalString(raw.quoteTokenUsdPrice),
    rootMarketId: optionalInt32(raw.rootMarketId),
    rootMarketTitle: optionalString(raw.rootMarketTitle),
    shares: optionalString(raw.shares),
    side: optionalString(raw.side),
    status: optionalInt32(raw.status),
    statusEnum: optionalString(raw.statusEnum),
    tradeNo: optionalString(raw.tradeNo),
    txHash: optionalString(raw.txHash),
    usdAmount: optionalString(raw.usdAmount),
  });
}
