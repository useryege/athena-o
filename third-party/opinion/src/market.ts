import {
  TopicSortType,
  TopicStatusFilter,
  TopicType,
  type Client,
} from '@opinion-labs/opinion-clob-sdk';
import { status as grpcStatus } from '@grpc/grpc-js';

import {
  ChildMarket,
  GetMarketResponse,
  GetMarketsResponse,
  GetQuoteTokensResponse,
  Market,
  MarketStatusFilter,
  MarketSortBy,
  MarketTopicType,
  QuoteToken,
  type GetCategoricalMarketRequest,
  type GetMarketRequest,
  type GetMarketsRequest,
  type GetMarketBySlugRequest,
  type GetQuoteTokensRequest,
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
