import type { Client } from '@opinion-labs/opinion-clob-sdk';

export {
  TopicType,
  TopicStatusFilter,
  TopicSortType,
} from '@opinion-labs/opinion-clob-sdk';

type ClientInstance = InstanceType<typeof Client>;

/** Wraps all SDK API responses: { errno, errmsg, result } */
export type ApiResponse<T> = { errno: number; errmsg: string; result: T };

/** Derived from Client#getMarkets return type */
export type GetMarketsReturnType = Awaited<ReturnType<ClientInstance['getMarkets']>>;
export type MarketListResult = GetMarketsReturnType['result'];
export type MarketData = MarketListResult['list'][number];

/** Derived from Client#getMarket return type */
export type GetMarketReturnType = Awaited<ReturnType<ClientInstance['getMarket']>>;
export type MarketDetailResult = GetMarketReturnType['result'];

/** Derived from Client#getQuoteTokens return type */
export type GetQuoteTokensReturnType = Awaited<ReturnType<ClientInstance['getQuoteTokens']>>;
export type QuoteTokenListResult = GetQuoteTokensReturnType['result'];
