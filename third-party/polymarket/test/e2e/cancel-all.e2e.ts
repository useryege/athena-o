import 'dotenv/config';

import * as grpc from '@grpc/grpc-js';
import assert from 'node:assert/strict';
import { after, before, describe, test } from 'node:test';

import { createPolymarketClient } from '../../src/client.js';
import { parseCancelAllRequest, toCancelAllResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  CancelAllResponse,
  type PolymarketServiceClient,
  PolymarketServiceClient as PolymarketServiceGrpcClient,
} from '../../src/gen/polymarket/polymarket.js';
import { assertEncodedEqual, getE2EGrpcAddress, handleE2EUnaryError } from './helpers.js';

async function grpcCancelAll(client: PolymarketServiceClient): Promise<CancelAllResponse> {
  return await new Promise<CancelAllResponse>((resolve, reject) => {
    client.cancelAll({}, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      resolve(response);
    });
  });
}

describe('Polymarket gRPC CancelAll parity (E2E)', () => {
  const address = getE2EGrpcAddress();
  let grpcClient: PolymarketServiceClient;

  before(() => {
    grpcClient = new PolymarketServiceGrpcClient(address, grpc.credentials.createInsecure());
  });

  after(() => {
    grpcClient.close();
  });

  test('CancelAll mirrors sidecar mapping for sdk result', async (t) => {
    if (process.env.POLYMARKET_E2E_ALLOW_CANCEL_ALL?.trim() !== 'true') {
      t.skip('Set POLYMARKET_E2E_ALLOW_CANCEL_ALL=true to run destructive CancelAll parity E2E');
    }

    const request = {};
    parseCancelAllRequest(request);

    const sdkClient = createPolymarketClient(loadRuntimeConfig().sdk);

    let sdkResult: unknown;
    try {
      sdkResult = await sdkClient.cancelAll();
      console.log('sdkResult:', sdkResult);
    } catch (sdkError) {
      try {
        await grpcCancelAll(grpcClient);
        assert.fail('gRPC CancelAll succeeded but SDK CancelAll failed');
      } catch (grpcError) {
        const serviceError = grpcError as grpc.ServiceError;
        if (serviceError.code === grpc.status.UNAVAILABLE) {
          handleE2EUnaryError(grpcError, address);
        }

        assert.equal(serviceError.code, grpc.status.INTERNAL);
        if (sdkError instanceof Error) {
          assert.match(
            serviceError.details || serviceError.message,
            new RegExp(sdkError.message.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')),
          );
        }
      }
      return;
    }

    const expected = toCancelAllResponse(sdkResult);
    const actual = await grpcCancelAll(grpcClient).catch((error) => handleE2EUnaryError(error, address));
    assertEncodedEqual(
      CancelAllResponse,
      actual,
      expected,
      'CancelAll parity mismatch (SDK request + sidecar request are separate upstream calls)',
    );
  });
});
