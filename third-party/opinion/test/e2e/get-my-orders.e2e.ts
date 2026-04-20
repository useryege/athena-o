/**
 * E2E: OpinionService.GetMyOrders
 *
 * 前置：本机已启动 opinion gRPC；`.env` 与侧链一致且含有效 API key（用户订单接口）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`）。
 *
 * 一致性用例会先后请求上游两次（SDK 与经 gRPC 各一次），列表可能在两次请求间变化。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetMyOrdersRequest, toGetMyOrdersResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetMyOrdersRequest,
  type GetMyOrdersResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetMyOrdersResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  handleSdkUserEndpointResponse,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getMyOrders(client: OpinionGrpcClient, request: GetMyOrdersRequest): Promise<GetMyOrdersResponse> {
  return new Promise((resolve, reject) => {
    client.getMyOrders(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetMyOrders returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetMyOrders — 成功返回 errno=0 且 list/total 结构合理', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetMyOrdersRequest = { page: 1, limit: 5 };
  let sdkResp: Awaited<ReturnType<typeof sdk.getMyOrders>>;
  try {
    sdkResp = await sdk.getMyOrders(parseGetMyOrdersRequest(request));
  } catch (error) {
    client.close();
    throw new Error('官方 SDK getMyOrders 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getMyOrders')) {
    client.close();
    return;
  }

  try {
    let response: GetMyOrdersResponse;
    try {
      response = await getMyOrders(client, request);
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

test('OpinionService.GetMyOrders — 官方 SDK 与 gRPC 返回（经同一 toGetMyOrdersResponse）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetMyOrdersRequest = { page: 1, limit: 5 };
  const query = parseGetMyOrdersRequest(request);

  let sdkResp: Awaited<ReturnType<typeof sdk.getMyOrders>>;
  try {
    sdkResp = await sdk.getMyOrders(query);
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getMyOrders 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getMyOrders')) {
    grpcClient.close();
    return;
  }

  const expected = toGetMyOrdersResponse(sdkResp);

  let actual: GetMyOrdersResponse;
  try {
    actual = await getMyOrders(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetMyOrdersResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「parseGetMyOrdersRequest + SDK + toGetMyOrdersResponse」一致；若偶发失败，可能是两次请求之间订单列表变化。',
  );
});
