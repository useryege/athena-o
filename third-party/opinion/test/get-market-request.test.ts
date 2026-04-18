import assert from 'node:assert/strict';
import test from 'node:test';

import { status as grpcStatus } from '@grpc/grpc-js';

import type { GetMarketRequest } from '../src/gen/opinion/opinion.js';
import { buildGetMarketOptions } from '../src/get-market/request.js';

test('buildGetMarketOptions parses a valid market_id string to a number', () => {
  const request: GetMarketRequest = { marketId: '42' };
  assert.strictEqual(buildGetMarketOptions(request), 42);
});

test('buildGetMarketOptions rejects market_id of 0', () => {
  assert.throws(
    () => buildGetMarketOptions({ marketId: '0' }),
    (error: unknown) =>
      error instanceof Error &&
      'code' in error &&
      error.code === grpcStatus.INVALID_ARGUMENT &&
      error.message === 'market_id must be >= 1',
  );
});

test('buildGetMarketOptions rejects negative market_id', () => {
  assert.throws(
    () => buildGetMarketOptions({ marketId: '-5' }),
    (error: unknown) =>
      error instanceof Error &&
      'code' in error &&
      error.code === grpcStatus.INVALID_ARGUMENT &&
      error.message === 'market_id must be >= 1',
  );
});

test('buildGetMarketOptions rejects non-integer market_id', () => {
  assert.throws(
    () => buildGetMarketOptions({ marketId: '1.5' }),
    (error: unknown) =>
      error instanceof Error &&
      'code' in error &&
      error.code === grpcStatus.INVALID_ARGUMENT &&
      error.message === 'market_id must be >= 1',
  );
});

test('buildGetMarketOptions rejects empty market_id', () => {
  assert.throws(
    () => buildGetMarketOptions({ marketId: '' }),
    (error: unknown) =>
      error instanceof Error &&
      'code' in error &&
      error.code === grpcStatus.INVALID_ARGUMENT &&
      error.message === 'market_id must be >= 1',
  );
});
