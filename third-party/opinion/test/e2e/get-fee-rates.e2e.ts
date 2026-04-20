/**
 * E2E: OpinionService.GetFeeRates
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 *
 * 可选：`OPINION_E2E_TOKEN_ID`；未设置时先 `getMarkets(limit:1)` 取首条 `yesTokenId`。
 * SDK `getFeeRates` 走链上读，无 `ApiResponse.errno`；失败时抛错。侧链映射为 `toGetFeeRatesResponse(settings)`。
 *
 * 与侧链共用同一 `.env`（`loadRuntimeConfig`）。Oracle：`parseGetFeeRatesRequest` → SDK `getFeeRates` → `toGetFeeRatesResponse`。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetFeeRatesRequest, toGetFeeRatesResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetFeeRatesRequest,
  type GetFeeRatesResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetFeeRatesResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  isLikelyUpstreamRegionBlockMessage,
  resolveE2eTokenId,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getFeeRates(client: OpinionGrpcClient, request: GetFeeRatesRequest): Promise<GetFeeRatesResponse> {
  return new Promise((resolve, reject) => {
    client.getFeeRates(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetFeeRates returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetFeeRates — 返回费率字段类型合理', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  try {
    const tokenId = await resolveE2eTokenId(sdk, t);
    if (tokenId === undefined) {
      return;
    }

    const request: GetFeeRatesRequest = { tokenId };
    let response: GetFeeRatesResponse;

    try {
      response = await getFeeRates(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(typeof response.makerMaxFeeRate, 'number');
    assert.equal(typeof response.takerMaxFeeRate, 'number');
    assert.equal(typeof response.enabled, 'boolean');
  } finally {
    client.close();
  }
});

test('OpinionService.GetFeeRates — 官方 SDK 与 gRPC 返回（经同一 toGetFeeRatesResponse）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const tokenId = await resolveE2eTokenId(sdk, t);
  if (tokenId === undefined) {
    grpcClient.close();
    return;
  }

  const request: GetFeeRatesRequest = { tokenId };
  const parsed = parseGetFeeRatesRequest(request);

  let settings: Awaited<ReturnType<typeof sdk.getFeeRates>>;
  try {
    settings = await sdk.getFeeRates(parsed);
  } catch (error) {
    grpcClient.close();
    const msg = error instanceof Error ? error.message : String(error);
    if (isLikelyUpstreamRegionBlockMessage(msg)) {
      t.skip('上游/链上调用当前区域不可用（SDK 抛错文案），跳过与 gRPC 的字节级对照');
      return;
    }

    throw new Error('官方 SDK getFeeRates 抛错', { cause: error });
  }

  const expected = toGetFeeRatesResponse(settings);

  let actual: GetFeeRatesResponse;
  try {
    actual = await getFeeRates(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetFeeRatesResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「parseGetFeeRatesRequest + SDK getFeeRates + toGetFeeRatesResponse」一致。',
  );
});
