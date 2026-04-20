/**
 * E2E: OpinionService.GetMyPositions
 *
 * 前置：本机 gRPC + 与侧链一致的 `.env`（含 API key）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`）。
 *
 * 一致性用例会对上游请求两次；持仓列表可能在两次请求间变化。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetMyPositionsRequest, toGetMyPositionsResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetMyPositionsRequest,
  type GetMyPositionsResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetMyPositionsResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  handleSdkUserEndpointResponse,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getMyPositions(
  client: OpinionGrpcClient,
  request: GetMyPositionsRequest,
): Promise<GetMyPositionsResponse> {
  return new Promise((resolve, reject) => {
    client.getMyPositions(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetMyPositions returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetMyPositions — 成功返回 errno=0 且 list 为数组', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetMyPositionsRequest = { page: 1, limit: 5 };

  let sdkResp: Awaited<ReturnType<typeof sdk.getMyPositions>>;
  try {
    sdkResp = await sdk.getMyPositions(parseGetMyPositionsRequest(request));
  } catch (error) {
    client.close();
    throw new Error('官方 SDK getMyPositions 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getMyPositions')) {
    client.close();
    return;
  }

  try {
    let response: GetMyPositionsResponse;
    try {
      response = await getMyPositions(client, request);
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

test('OpinionService.GetMyPositions — 官方 SDK 与 gRPC 返回（经同一 toGetMyPositionsResponse）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetMyPositionsRequest = { page: 1, limit: 5 };
  const query = parseGetMyPositionsRequest(request);

  let sdkResp: Awaited<ReturnType<typeof sdk.getMyPositions>>;
  try {
    sdkResp = await sdk.getMyPositions(query);
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getMyPositions 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getMyPositions')) {
    grpcClient.close();
    return;
  }

  const expected = toGetMyPositionsResponse(sdkResp);

  let actual: GetMyPositionsResponse;
  try {
    actual = await getMyPositions(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetMyPositionsResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「parseGetMyPositionsRequest + SDK + toGetMyPositionsResponse」一致；若偶发失败，可能是两次请求之间持仓变化。',
  );
});
