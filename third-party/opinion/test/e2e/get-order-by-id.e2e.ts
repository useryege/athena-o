/**
 * E2E: OpinionService.GetOrderById
 *
 * 前置：本机 gRPC + 与侧链一致的 `.env`（含 API key）。
 * 可选：`OPINION_E2E_ORDER_ID`；未设置时尝试 `getMyOrders(page:1,limit:1)` 取首条 `orderId`。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`）。
 *
 * 解析 orderId 时可能多一次上游请求；parity 再各走 SDK/gRPC，共最多三次订单相关 HTTP。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetOrderByIdRequest, toGetOrderByIdResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetOrderByIdRequest,
  type GetOrderByIdResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetOrderByIdResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  handleSdkUserEndpointResponse,
  resolveE2eOrderId,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getOrderById(client: OpinionGrpcClient, request: GetOrderByIdRequest): Promise<GetOrderByIdResponse> {
  return new Promise((resolve, reject) => {
    client.getOrderById(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetOrderById returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetOrderById — 成功返回 errno=0 且 orderData 结构合理', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const orderId = await resolveE2eOrderId(sdk, t);
  if (orderId === undefined) {
    client.close();
    return;
  }

  const request: GetOrderByIdRequest = { orderId };
  let sdkResp: Awaited<ReturnType<typeof sdk.getOrderById>>;
  try {
    sdkResp = await sdk.getOrderById(parseGetOrderByIdRequest(request));
  } catch (error) {
    client.close();
    throw new Error('官方 SDK getOrderById 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getOrderById')) {
    client.close();
    return;
  }

  try {
    let response: GetOrderByIdResponse;
    try {
      response = await getOrderById(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(response.errno, 0, `期望 errno=0，实际 ${response.errno} errmsg=${response.errmsg}`);
    assert.equal(typeof response.errmsg, 'string');
    assert.ok(response.orderData, 'errno=0 时期望返回 orderData');
  } finally {
    client.close();
  }
});

test('OpinionService.GetOrderById — 官方 SDK 与 gRPC 返回（经同一 toGetOrderByIdResponse）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const orderId = await resolveE2eOrderId(sdk, t);
  if (orderId === undefined) {
    grpcClient.close();
    return;
  }

  const request: GetOrderByIdRequest = { orderId };
  const id = parseGetOrderByIdRequest(request);

  let sdkResp: Awaited<ReturnType<typeof sdk.getOrderById>>;
  try {
    sdkResp = await sdk.getOrderById(id);
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getOrderById 抛错', { cause: error });
  }

  if (handleSdkUserEndpointResponse(sdkResp, t, 'getOrderById')) {
    grpcClient.close();
    return;
  }

  const expected = toGetOrderByIdResponse(sdkResp);

  let actual: GetOrderByIdResponse;
  try {
    actual = await getOrderById(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetOrderByIdResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「parseGetOrderByIdRequest + SDK + toGetOrderByIdResponse」一致；若偶发失败，可能是两次 getOrderById 之间上游数据变化。',
  );
});
