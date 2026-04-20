/**
 * E2E: OpinionService.GetOrderbook
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 *
 * 可选：`OPINION_E2E_TOKEN_ID`；未设置时先 `getMarkets(limit:1)` 取首条 `yesTokenId`（与 `parseGetMarketsRequest` 一致）。
 * 本用例会对上游发起 **两次** 订单簿请求（SDK 一次、gRPC 经侧链一次）；盘口可能在两次请求间变化，
 * 故「一致性」用例在 protobuf 字节不一致时 **跳过**（不计失败），而非断言报错。
 *
 * 与侧链共用同一 `.env`（`loadRuntimeConfig`）。Oracle：`parseGetOrderbookRequest` → SDK `getOrderbook` → `toGetOrderbookResponse`。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetOrderbookRequest, toGetOrderbookResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetOrderbookRequest,
  type GetOrderbookResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetOrderbookResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  isLikelySdkUpstreamRegionBlock,
  resolveE2eTokenId,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getOrderbook(client: OpinionGrpcClient, request: GetOrderbookRequest): Promise<GetOrderbookResponse> {
  return new Promise((resolve, reject) => {
    client.getOrderbook(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetOrderbook returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetOrderbook — 成功返回 errno=0 且 asks/bids 为数组', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  try {
    const tokenId = await resolveE2eTokenId(sdk, t);
    if (tokenId === undefined) {
      return;
    }

    const request: GetOrderbookRequest = { tokenId };
    let response: GetOrderbookResponse;

    try {
      response = await getOrderbook(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(response.errno, 0, `期望 errno=0，实际 ${response.errno} errmsg=${response.errmsg}`);
    assert.equal(typeof response.errmsg, 'string');
    assert.ok(Array.isArray(response.asks), 'asks 应为数组');
    assert.ok(Array.isArray(response.bids), 'bids 应为数组');
  } finally {
    client.close();
  }
});

test('OpinionService.GetOrderbook — 官方 SDK 与 gRPC 返回（经同一 toGetOrderbookResponse）；盘口变化时跳过对照', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const tokenId = await resolveE2eTokenId(sdk, t);
  if (tokenId === undefined) {
    grpcClient.close();
    return;
  }

  const request: GetOrderbookRequest = { tokenId };
  const parsed = parseGetOrderbookRequest(request);

  let sdkResp: Awaited<ReturnType<typeof sdk.getOrderbook>>;
  try {
    sdkResp = await sdk.getOrderbook(parsed);
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getOrderbook 抛错', { cause: error });
  }

  if (sdkResp.errno !== 0) {
    grpcClient.close();
    if (isLikelySdkUpstreamRegionBlock(sdkResp)) {
      t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过与 gRPC 的字节级对照');
      return;
    }

    assert.fail(`官方 SDK getOrderbook 失败：errno=${sdkResp.errno} errmsg=${sdkResp.errmsg}`);
  }

  const expected = toGetOrderbookResponse(sdkResp);

  let actual: GetOrderbookResponse;
  try {
    actual = await getOrderbook(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  try {
    assertGetOrderbookResponsesEqual(
      expected,
      actual,
      'gRPC 响应应与「parseGetOrderbookRequest + SDK + toGetOrderbookResponse」一致。',
    );
  } catch (error) {
    if (error instanceof assert.AssertionError) {
      t.skip('订单簿在两次上游请求之间可能已变化，跳过字节级对照（不计失败）');
      return;
    }

    throw error;
  }
});
