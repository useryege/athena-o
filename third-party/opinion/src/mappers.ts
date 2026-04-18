import type { GetMarketsResponse, Market } from './gen/opinion/opinion.js';

type RawRecord = Record<string, unknown>;
type RawGetMarketsResult = {
  total?: number;
  list?: unknown;
};

function asString(value: unknown): string {
  if (value === null || value === undefined) {
    return '';
  }

  return String(value);
}

function asNumber(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }

  return 0;
}

function asBoolean(value: unknown): boolean {
  return value === true;
}

function asRecordList(value: unknown): RawRecord[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter((item): item is RawRecord => typeof item === 'object' && item !== null);
}

export function mapMarket(raw: RawRecord): Market {
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

export function mapGetMarketsResponse(result: RawGetMarketsResult): GetMarketsResponse {
  return {
    total: typeof result.total === 'number' ? result.total : 0,
    markets: asRecordList(result.list).map(mapMarket),
  };
}
