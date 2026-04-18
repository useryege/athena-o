import type { GetMarketsResponse, MarketMessage } from './types.js';

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

export function mapMarket(raw: Record<string, unknown>): MarketMessage {
  return {
    market_id: asString(raw.marketId),
    market_title: asString(raw.marketTitle),
    slug: asString(raw.slug),
    condition_id: asString(raw.conditionId),
    chain_id: asString(raw.chainId),
    quote_token: asString(raw.quoteToken),
    status: asNumber(raw.status),
    status_enum: asString(raw.statusEnum),
    created_at: asString(raw.createdAt),
    cutoff_at: asString(raw.cutoffAt),
    resolved_at: asString(raw.resolvedAt),
    yes_token_id: asString(raw.yesTokenId),
    no_token_id: asString(raw.noTokenId),
    result_token_id: asString(raw.resultTokenId),
    yes_label: asString(raw.yesLabel),
    no_label: asString(raw.noLabel),
    volume: asString(raw.volume),
    is_incentivized: asBoolean(raw.isIncentivized),
    child_markets: asMarketList(raw.childMarkets).map(mapMarket),
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
