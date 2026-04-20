/**
 * E2E: OpinionService.GetMyTrades
 *
 * 前置：本机 gRPC + 与侧链一致的 `.env`（含 API key）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`）。
 *
 * 一致性用例会对上游请求两次；成交列表可能在两次请求间变化。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetMyTradesRequest, toGetMyTradesResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetMyTradesRequest,
  type GetMyTradesResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetMyTradesResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  handleSdkUserEndpointResponse,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getMyTrades(client: OpinionGrpcClient, request: GetMyTradesRequest): Promise<GetMyTradesResponse> {
  return new Promise((resolve, reject) => {
    client.getMyTrades(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetMyTrades returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetMyTrades — 成功返回 errno=0 且 list 为数组', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetMyTradesRequest = { page: 1, limit: 5 };

  let sdkResp: Awaited<ReturnType<typeof sdk.getMyTrades>>;
  try {
    sdkResp = await sdk.getMyTrades(parseGetMyTradesRequest(request));
  } catch (error) {
    client.close();
    throw new Error('官方 SDK getMyTrades 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getMyTrades')) {
    client.close();
    return;
  }

  try {
    let response: GetMyTradesResponse;
    try {
      response = await getMyTrades(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(response.errno, 0, `期望 errno=0，实际 ${response.errno} errmsg=${response.errmsg}`);
    assert.equal(typeof response.errmsg, 'string');
    assert.ok(Array.isArray(response.list), 'list 应为数组');
    if (response.total !== undefined) {
      assert.equal(typeof response.total, 'number');
    }
  } finally {
    client.close();
  }
});

test('OpinionService.GetMyTrades — 官方 SDK 与 gRPC 返回（经同一 toGetMyTradesResponse）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetMyTradesRequest = { page: 1, limit: 5 };
  const query = parseGetMyTradesRequest(request);

  let sdkResp: Awaited<ReturnType<typeof sdk.getMyTrades>>;
  try {
    sdkResp = await sdk.getMyTrades(query);
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getMyTrades 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getMyTrades')) {
    grpcClient.close();
    return;
  }

  const expected = toGetMyTradesResponse(sdkResp);

  let actual: GetMyTradesResponse;
  try {
    actual = await getMyTrades(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetMyTradesResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「parseGetMyTradesRequest + SDK + toGetMyTradesResponse」一致；若偶发失败，可能是两次请求之间成交列表变化。',
  );
});
