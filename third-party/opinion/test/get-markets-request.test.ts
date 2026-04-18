import assert from 'node:assert/strict';
import test from 'node:test';

import { status as grpcStatus } from '@grpc/grpc-js';
import { TopicSortType, TopicStatusFilter, TopicType } from '@opinion-labs/opinion-clob-sdk';

import {
  MarketSortBy,
  MarketStatusFilter,
  MarketTopicType,
  type GetMarketsRequest,
} from '../src/gen/opinion/opinion.js';
import { buildGetMarketsOptions } from '../src/get-markets/request.js';

test('buildGetMarketsOptions maps proto fields to sdk options', () => {
  const request: GetMarketsRequest = {
    topicType: MarketTopicType.MARKET_TOPIC_TYPE_BINARY,
    page: 2,
    limit: 5,
    status: MarketStatusFilter.MARKET_STATUS_FILTER_ACTIVATED,
    sortBy: MarketSortBy.MARKET_SORT_BY_VOLUME_DESC,
  };

  assert.deepEqual(buildGetMarketsOptions(request), {
    topicType: TopicType.BINARY,
    page: 2,
    limit: 5,
    status: TopicStatusFilter.ACTIVATED,
    sortBy: TopicSortType.BY_VOLUME_DESC,
  });
});

test('buildGetMarketsOptions keeps zero values omitted and defaults topicType to ALL', () => {
  const request: GetMarketsRequest = {
    topicType: MarketTopicType.MARKET_TOPIC_TYPE_UNSPECIFIED,
    page: 0,
    limit: 0,
    status: MarketStatusFilter.MARKET_STATUS_FILTER_ALL,
    sortBy: MarketSortBy.MARKET_SORT_BY_UNSPECIFIED,
  };

  assert.deepEqual(buildGetMarketsOptions(request), {
    topicType: TopicType.ALL,
  });
});

test('buildGetMarketsOptions rejects invalid integers and unsupported enums', () => {
  assert.throws(
    () =>
      buildGetMarketsOptions({
        topicType: MarketTopicType.MARKET_TOPIC_TYPE_ALL,
        page: -1,
        limit: 0,
        status: MarketStatusFilter.MARKET_STATUS_FILTER_UNSPECIFIED,
        sortBy: MarketSortBy.MARKET_SORT_BY_UNSPECIFIED,
      }),
    (error: unknown) =>
      error instanceof Error &&
      'code' in error &&
      error.code === grpcStatus.INVALID_ARGUMENT &&
      error.message === 'page must be >= 1',
  );

  assert.throws(
    () =>
      buildGetMarketsOptions({
        topicType: 99 as MarketTopicType,
        page: 0,
        limit: 0,
        status: MarketStatusFilter.MARKET_STATUS_FILTER_UNSPECIFIED,
        sortBy: MarketSortBy.MARKET_SORT_BY_UNSPECIFIED,
      }),
    /unsupported topic_type: 99/,
  );
});
