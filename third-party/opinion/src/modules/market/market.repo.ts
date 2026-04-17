import { getClient } from '../../infra/opinion/client.js';
import {
  TopicType,
  TopicStatusFilter,
  TopicSortType,
  type GetMarketsReturnType,
} from '../../infra/opinion/types.js';

export interface GetMarketsOptions {
  topicType?: TopicType;
  page?: number;
  limit?: number;
  status?: TopicStatusFilter;
  sortBy?: TopicSortType;
}

export async function fetchMarkets(options: GetMarketsOptions): Promise<GetMarketsReturnType> {
  const client = getClient();
  return client.getMarkets(options);
}
