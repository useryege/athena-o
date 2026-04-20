/**
 * E2E: OpinionService.GetMyBalances
 *
 * 前置：本机 gRPC + 与侧链一致的 `.env`（含 API key）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`）。
 *
 * 一致性用例会对上游请求两次（SDK 与经 gRPC 各一次）。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { toGetMyBalancesResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetMyBalancesRequest,
  type GetMyBalancesResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetMyBalancesResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  handleSdkUserEndpointResponse,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getMyBalances(
  client: OpinionGrpcClient,
  request: GetMyBalancesRequest,
): Promise<GetMyBalancesResponse> {
  return new Promise((resolve, reject) => {
    client.getMyBalances(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetMyBalances returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetMyBalances — 成功返回 errno=0 且 balances 为数组', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetMyBalancesRequest = {};

  let sdkResp: Awaited<ReturnType<typeof sdk.getMyBalances>>;
  try {
    sdkResp = await sdk.getMyBalances();
  } catch (error) {
    client.close();
    throw new Error('官方 SDK getMyBalances 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getMyBalances')) {
    client.close();
    return;
  }

  try {
    let response: GetMyBalancesResponse;
    try {
      response = await getMyBalances(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(response.errno, 0, `期望 errno=0，实际 ${response.errno} errmsg=${response.errmsg}`);
    assert.equal(typeof response.errmsg, 'string');
    assert.ok(Array.isArray(response.balances), 'balances 应为数组');
  } finally {
    client.close();
  }
});

test('OpinionService.GetMyBalances — 官方 SDK 与 gRPC 返回（经同一 toGetMyBalancesResponse）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetMyBalancesRequest = {};

  let sdkResp: Awaited<ReturnType<typeof sdk.getMyBalances>>;
  try {
    sdkResp = await sdk.getMyBalances();
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getMyBalances 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getMyBalances')) {
    grpcClient.close();
    return;
  }

  const expected = toGetMyBalancesResponse(sdkResp);

  let actual: GetMyBalancesResponse;
  try {
    actual = await getMyBalances(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetMyBalancesResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「SDK getMyBalances + toGetMyBalancesResponse」一致。',
  );
});
