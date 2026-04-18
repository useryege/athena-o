import assert from 'node:assert/strict';
import test from 'node:test';

import { status as grpcStatus } from '@grpc/grpc-js';

import type { GetMarketRequest } from '../src/gen/opinion/opinion.js';
import { executeGetMarket } from '../src/get-market/usecase.js';

const defaultRequest: GetMarketRequest = { marketId: '100' };

const emptyMarket = {
  marketId: '',
  marketTitle: '',
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
};

test('executeGetMarket passes parsed marketId to sdk and maps successful response', async () => {
  const captured: unknown[] = [];
  const response = await executeGetMarket(defaultRequest, {
    client: {
      async getMarket(marketId) {
        captured.push(marketId);
        return {
          errno: 0,
          errmsg: '',
          result: {
            data: { marketId: 100, marketTitle: 'Test Market', chainId: '56', quoteToken: 'USDT', conditionId: 'cond1', status: 1 },
          },
        };
      },
    },
  });

  assert.deepEqual(captured, [100]);
  assert.deepEqual(response, {
    market: {
      ...emptyMarket,
      marketId: '100',
      marketTitle: 'Test Market',
      chainId: '56',
      quoteToken: 'USDT',
      conditionId: 'cond1',
      status: 1,
    },
  });
});

test('executeGetMarket translates sdk errno into grpc internal error', async () => {
  await assert.rejects(
    () =>
      executeGetMarket(defaultRequest, {
        client: {
          async getMarket() {
            return {
              errno: 5,
              errmsg: 'market not found',
              result: { data: undefined },
            };
          },
        },
      }),
    (error: unknown) =>
      error instanceof Error &&
      'code' in error &&
      error.code === grpcStatus.INTERNAL &&
      error.message === 'market not found',
  );
});

test('executeGetMarket translates sdk errno with empty errmsg into generic message', async () => {
  await assert.rejects(
    () =>
      executeGetMarket(defaultRequest, {
        client: {
          async getMarket() {
            return {
              errno: 3,
              errmsg: '',
              result: { data: undefined },
            };
          },
        },
      }),
    (error: unknown) =>
      error instanceof Error &&
      'code' in error &&
      error.code === grpcStatus.INTERNAL &&
      error.message === 'opinion sdk returned errno 3',
  );
});

test('executeGetMarket returns undefined market when sdk result has no data', async () => {
  const response = await executeGetMarket(defaultRequest, {
    client: {
      async getMarket() {
        return {
          errno: 0,
          errmsg: '',
          result: { data: undefined },
        };
      },
    },
  });

  assert.deepEqual(response, { market: undefined });
});

test('executeGetMarket rejects invalid market_id', async () => {
  await assert.rejects(
    () => executeGetMarket({ marketId: '0' }, {}),
    (error: unknown) =>
      error instanceof Error &&
      'code' in error &&
      error.code === grpcStatus.INVALID_ARGUMENT &&
      error.message === 'market_id must be >= 1',
  );
});
