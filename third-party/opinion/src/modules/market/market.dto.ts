import type { MarketData } from '../../infra/opinion/types.js';

/**
 * Query params accepted by GET /v1/markets.
 * All values are plain strings so Go callers never touch SDK enums.
 */
export interface GetMarketsQuery {
  /** 0=BINARY, 1=CATEGORICAL, 2=ALL (default: 2) */
  topicType?: string;
  page?: string;
  limit?: string;
  /** ""=ALL, "activated", "resolved" (default: all) */
  status?: string;
  /** 1=BY_TIME_DESC…8=BY_VOLUME_7D_ASC (default: 1) */
  sortBy?: string;
}

export interface GetMarketsResponse {
  total: number;
  list: MarketData[];
}
