import type { GetMarketsResponse, Market } from './gen/opinion/opinion.js';

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

function asMarketList(value: unknown): Record<string, unknown>[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter((item): item is Record<string, unknown> => typeof item === 'object' && item !== null);
}

export function mapMarket(raw: Record<string, unknown>): Market {
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
    childMarkets: asMarketList(raw.childMarkets).map(mapMarket),
  };
}

export function mapGetMarketsResponse(result: {
  total?: number;
  list?: unknown;
}): GetMarketsResponse {
  return {
    total: typeof result.total === 'number' ? result.total : 0,
    markets: asMarketList(result.list).map(mapMarket),
  };
}
