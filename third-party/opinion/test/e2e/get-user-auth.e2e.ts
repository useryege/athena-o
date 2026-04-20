/**
 * E2E: OpinionService.GetUserAuth
 *
 * 前置：本机 gRPC + 与侧链一致的 `.env`。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`）。
 *
 * 一致性用例会对上游请求两次（SDK 与经 gRPC 各一次）。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { toGetUserAuthResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetUserAuthRequest,
  type GetUserAuthResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetUserAuthResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  handleSdkUserEndpointResponse,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getUserAuth(client: OpinionGrpcClient, request: GetUserAuthRequest): Promise<GetUserAuthResponse> {
  return new Promise((resolve, reject) => {
    client.getUserAuth(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetUserAuth returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetUserAuth — 成功返回 errno=0 且 walletUsers 为对象', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetUserAuthRequest = {};

  let sdkResp: Awaited<ReturnType<typeof sdk.getUserAuth>>;
  try {
    sdkResp = await sdk.getUserAuth();
  } catch (error) {
    client.close();
    throw new Error('官方 SDK getUserAuth 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getUserAuth')) {
    client.close();
    return;
  }

  try {
    let response: GetUserAuthResponse;
    try {
      response = await getUserAuth(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(response.errno, 0, `期望 errno=0，实际 ${response.errno} errmsg=${response.errmsg}`);
    assert.equal(typeof response.errmsg, 'string');
    assert.ok(response.walletUsers && typeof response.walletUsers === 'object', 'walletUsers 应为对象');
  } finally {
    client.close();
  }
});

test('OpinionService.GetUserAuth — 官方 SDK 与 gRPC 返回（经同一 toGetUserAuthResponse）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetUserAuthRequest = {};

  let sdkResp: Awaited<ReturnType<typeof sdk.getUserAuth>>;
  try {
    sdkResp = await sdk.getUserAuth();
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getUserAuth 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getUserAuth')) {
    grpcClient.close();
    return;
  }

  const expected = toGetUserAuthResponse(sdkResp);

  let actual: GetUserAuthResponse;
  try {
    actual = await getUserAuth(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetUserAuthResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「SDK getUserAuth + toGetUserAuthResponse」一致。',
  );
});
