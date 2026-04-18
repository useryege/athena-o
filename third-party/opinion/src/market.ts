import { TopicSortType, TopicStatusFilter, TopicType } from '@opinion-labs/opinion-clob-sdk';
import { status as grpcStatus } from '@grpc/grpc-js';

import type {
  GetMarketRequest,
  GetMarketsRequest,
  GetMarketResponse,
  GetMarketsResponse,
  Market,
} from './gen/opinion/opinion.js';
import { createServiceError } from './grpc-error.js';

// ─── Internal query types ──────────────────────────────────────────────────

export interface GetMarketsQuery {
  topicType: TopicType;
  page?: number;
  limit?: number;
  status?: TopicStatusFilter;
  sortBy?: TopicSortType;
}

// ─── Raw result shapes returned by the client layer ───────────────────────

export type MarketDetailResult = {
  data?: unknown;
};

export type MarketListResult = {
  total?: number;
  list?: unknown;
};

// ─── Request parsing ───────────────────────────────────────────────────────

export function parseGetMarketRequest(request: GetMarketRequest): number {
  const marketId = Number(request.marketId);
  if (!Number.isInteger(marketId) || marketId < 1) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, 'market_id must be >= 1');
  }
  return marketId;
}

export function parseGetMarketsRequest(request: GetMarketsRequest): GetMarketsQuery {
  const page = normalizePositiveInt('page', request.page);
  const limit = normalizePositiveInt('limit', request.limit);
  const topicType = toSdkTopicType(request.topicType);
  const status = toSdkStatusFilter(request.status);
  const sortBy = toSdkSortBy(request.sortBy);

  return {
    topicType,
    ...(page ? { page } : {}),
    ...(limit ? { limit } : {}),
    ...(status ? { status } : {}),
    ...(sortBy ? { sortBy } : {}),
  };
}

// ─── Response mapping ──────────────────────────────────────────────────────

export function toGetMarketResponse(result: MarketDetailResult): GetMarketResponse {
  const raw = result.data;
  return {
    market:
      raw !== null && typeof raw === 'object' ? mapMarket(raw as RawRecord) : undefined,
  };
}

export function toGetMarketsResponse(result: MarketListResult): GetMarketsResponse {
  return {
    total: typeof result.total === 'number' ? result.total : 0,
    markets: asRecordList(result.list).map(mapMarket),
  };
}

// ─── Private helpers ───────────────────────────────────────────────────────

type RawRecord = Record<string, unknown>;

function mapMarket(raw: RawRecord): Market {
  return {
    marketId: asString(raw.marketId),
    marketTitle: asString(raw.marketTitle),
    slug: asString(raw.slug),
    conditionId: asString(raw.conditionId),
    chainId: asString(raw.chainId),
    quoteToken: asString(raw.quoteToken),
    status: asNumber(raw.status),
    statusEnum: asString(raw.statusEnum),
    createdAt: asString(raw.createdAt),
    cutoffAt: asString(raw.cutoffAt),
    resolvedAt: asString(raw.resolvedAt),
    yesTokenId: asString(raw.yesTokenId),
    noTokenId: asString(raw.noTokenId),
    resultTokenId: asString(raw.resultTokenId),
    yesLabel: asString(raw.yesLabel),
    noLabel: asString(raw.noLabel),
    volume: asString(raw.volume),
    isIncentivized: asBoolean(raw.isIncentivized),
    childMarkets: asRecordList(raw.childMarkets).map(mapMarket),
  };
}

function normalizePositiveInt(name: string, value: number | undefined): number | undefined {
  if (!value) return undefined;
  if (!Number.isInteger(value) || value < 1) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, `${name} must be >= 1`);
  }
  return value;
}

function toSdkTopicType(value: number | undefined): TopicType {
  switch (value) {
    case 1:
      return TopicType.BINARY;
    case 2:
      return TopicType.CATEGORICAL;
    case 3:
    case 0:
    case undefined:
      return TopicType.ALL;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported topic_type: ${value}`);
  }
}

function toSdkStatusFilter(value: number | undefined): TopicStatusFilter | undefined {
  switch (value) {
    case 0:
    case 1:
    case undefined:
      return undefined;
    case 2:
      return TopicStatusFilter.ACTIVATED;
    case 3:
      return TopicStatusFilter.RESOLVED;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported status filter: ${value}`);
  }
}

function toSdkSortBy(value: number | undefined): TopicSortType | undefined {
  switch (value) {
    case 0:
    case undefined:
      return undefined;
    case 1:
      return TopicSortType.BY_TIME_DESC;
    case 2:
      return TopicSortType.BY_CUTOFF_TIME_ASC;
    case 3:
      return TopicSortType.BY_VOLUME_DESC;
    case 4:
      return TopicSortType.BY_VOLUME_ASC;
    case 5:
      return TopicSortType.BY_VOLUME_24H_DESC;
    case 6:
      return TopicSortType.BY_VOLUME_24H_ASC;
    case 7:
      return TopicSortType.BY_VOLUME_7D_DESC;
    case 8:
      return TopicSortType.BY_VOLUME_7D_ASC;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported sort_by: ${value}`);
  }
}

function asString(value: unknown): string {
  if (value === null || value === undefined) return '';
  return String(value);
}

function asNumber(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return 0;
}

function asBoolean(value: unknown): boolean {
  return value === true;
}

function asRecordList(value: unknown): RawRecord[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is RawRecord => typeof item === 'object' && item !== null);
}
