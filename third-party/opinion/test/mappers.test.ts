import assert from 'node:assert/strict';
import test from 'node:test';

import { mapGetMarketsResponse } from '../src/mappers.js';

test('mapGetMarketsResponse converts raw sdk payloads into grpc response shape', () => {
  const response = mapGetMarketsResponse({
    total: 2,
    list: [
      {
        marketId: 123,
        marketTitle: 'Will it rain?',
        slug: 'will-it-rain',
        conditionId: 'cond-1',
        chainId: '56',
        quoteToken: 'USDT',
        status: '4',
        statusEnum: 'ACTIVATED',
        createdAt: '2026-01-01T00:00:00Z',
        cutoffAt: '2026-01-02T00:00:00Z',
        resolvedAt: null,
        yesTokenId: 'yes-token',
        noTokenId: 'no-token',
        resultTokenId: 456,
        yesLabel: 'Yes',
        noLabel: 'No',
        volume: 789.12,
        isIncentivized: true,
        childMarkets: [
          {
            marketId: '789',
            marketTitle: 'Child market',
            status: 1,
            isIncentivized: false,
          },
        ],
      },
      'ignored',
    ],
  });

  assert.deepEqual(response, {
    total: 2,
    markets: [
      {
        marketId: '123',
        marketTitle: 'Will it rain?',
        slug: 'will-it-rain',
        conditionId: 'cond-1',
        chainId: '56',
        quoteToken: 'USDT',
        status: 4,
        statusEnum: 'ACTIVATED',
        createdAt: '2026-01-01T00:00:00Z',
        cutoffAt: '2026-01-02T00:00:00Z',
        resolvedAt: '',
        yesTokenId: 'yes-token',
        noTokenId: 'no-token',
        resultTokenId: '456',
        yesLabel: 'Yes',
        noLabel: 'No',
        volume: '789.12',
        isIncentivized: true,
        childMarkets: [
          {
            marketId: '789',
            marketTitle: 'Child market',
            slug: '',
            conditionId: '',
            chainId: '',
            quoteToken: '',
            status: 1,
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
      },
    ],
  });
});

test('mapGetMarketsResponse falls back for missing or invalid values', () => {
  assert.deepEqual(mapGetMarketsResponse({ list: null }), {
    total: 0,
    markets: [],
  });
});
