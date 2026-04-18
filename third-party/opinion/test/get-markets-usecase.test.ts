import assert from 'node:assert/strict';
import test from 'node:test';

import { status as grpcStatus } from '@grpc/grpc-js';
import { TopicType } from '@opinion-labs/opinion-clob-sdk';

import {
  MarketSortBy,
  MarketStatusFilter,
  MarketTopicType,
  type GetMarketsRequest,
} from '../src/gen/opinion/opinion.js';
import { executeGetMarkets } from '../src/get-markets/usecase.js';

const defaultRequest: GetMarketsRequest = {
  topicType: MarketTopicType.MARKET_TOPIC_TYPE_ALL,
  page: 1,
  limit: 2,
  status: MarketStatusFilter.MARKET_STATUS_FILTER_UNSPECIFIED,
  sortBy: MarketSortBy.MARKET_SORT_BY_UNSPECIFIED,
};

test('executeGetMarkets uses injected client and maps successful sdk responses', async () => {
  const captured: unknown[] = [];
  const response = await executeGetMarkets(defaultRequest, {
    client: {
      async getMarkets(options) {
        captured.push(options);
        return {
          errno: 0,
          errmsg: '',
          result: {
            total: 1,
            list: [{ marketId: '100', marketTitle: 'Example market' }],
          },
        };
      },
    },
  });

  assert.deepEqual(captured, [
    {
      topicType: TopicType.ALL,
      page: 1,
      limit: 2,
    },
  ]);
  assert.deepEqual(response, {
    total: 1,
    markets: [
      {
        marketId: '100',
        marketTitle: 'Example market',
        slug: '',
        conditionId: '',
        chainId: '',
        quoteToken: '',
        status: 0,
        statusEnum: '',
        createdAt: '',
        cutoffAt: '',
        resolvedAt: '',
        yesTokenId: '',
        noTokenId: '',
        resultTokenId: '',
        yesLabel: '',
        noLabel: '',
        volume: '',
        isIncentivized: false,
        childMarkets: [],
      },
    ],
  });
});

test('executeGetMarkets translates sdk errno into grpc internal error', async () => {
  await assert.rejects(
    () =>
      executeGetMarkets(defaultRequest, {
        client: {
          async getMarkets() {
            return {
              errno: 7,
              errmsg: '',
              result: {
                total: 0,
                list: [],
              },
            };
          },
        },
      }),
    (error: unknown) =>
      error instanceof Error &&
      'code' in error &&
      error.code === grpcStatus.INTERNAL &&
      error.message === 'opinion sdk returned errno 7',
  );
});
