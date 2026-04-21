import 'dotenv/config';

import * as grpc from '@grpc/grpc-js';
import assert from 'node:assert/strict';
import { after, before, describe, test } from 'node:test';

import {
  PolymarketServiceClient,
  HealthResponse,
} from '../../src/gen/polymarket/polymarket.js';
import { getE2EGrpcAddress, handleE2EUnaryError } from './helpers.js';

describe('Polymarket gRPC Health (E2E placeholder)', () => {
  const address = getE2EGrpcAddress();
  let client: PolymarketServiceClient;

  before(() => {
    client = new PolymarketServiceClient(address, grpc.credentials.createInsecure());
  });

  after(() => {
    client.close();
  });

  test('Health returns ok', async () => {
    await new Promise<void>((resolve, reject) => {
      client.health({}, (error, response) => {
        try {
          if (error) {
            handleE2EUnaryError(error, address);
          }

          assert.ok(response);
          const encoded = HealthResponse.encode(HealthResponse.fromJSON({ status: 'ok' })).finish();
          assert.deepStrictEqual(HealthResponse.encode(response).finish(), encoded);
          resolve();
        } catch (e) {
          reject(e);
        }
      });
    });
  });
});
