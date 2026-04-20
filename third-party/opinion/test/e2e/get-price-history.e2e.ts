/**
 * E2E: OpinionService.GetPriceHistory
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 *
 * 可选：`OPINION_E2E_TOKEN_ID`；未设置时先 `getMarkets(limit:1)` 取首条 `yesTokenId`。
 * 请求仅含 `tokenId`，`interval` 未指定时与侧链一致地交给 SDK 默认（1h）。仍会 **两次** 请求 K 线，极端情况下尾部蜡烛可能追加导致对照失败。
 *
 * 与侧链共用同一 `.env`（`loadRuntimeConfig`）。Oracle：`parseGetPriceHistoryRequest` → SDK `getPriceHistory` → `toGetPriceHistoryResponse`。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetPriceHistoryRequest, toGetPriceHistoryResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetPriceHistoryRequest,
  type GetPriceHistoryResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetPriceHistoryResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  isLikelySdkUpstreamRegionBlock,
  resolveE2eTokenId,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getPriceHistory(
  client: OpinionGrpcClient,
  request: GetPriceHistoryRequest,
): Promise<GetPriceHistoryResponse> {
  return new Promise((resolve, reject) => {
    client.getPriceHistory(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetPriceHistory returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetPriceHistory — 成功返回 errno=0 且 history 为数组', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  try {
    const tokenId = await resolveE2eTokenId(sdk, t);
    if (tokenId === undefined) {
      return;
    }

    const request: GetPriceHistoryRequest = { tokenId };
    let response: GetPriceHistoryResponse;

    try {
      response = await getPriceHistory(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(response.errno, 0, `期望 errno=0，实际 ${response.errno} errmsg=${response.errmsg}`);
    assert.equal(typeof response.errmsg, 'string');
    assert.ok(Array.isArray(response.history), 'history 应为数组');
  } finally {
    client.close();
  }
});

test('OpinionService.GetPriceHistory — 官方 SDK 与 gRPC 返回（经同一 toGetPriceHistoryResponse）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const tokenId = await resolveE2eTokenId(sdk, t);
  if (tokenId === undefined) {
    grpcClient.close();
    return;
  }

  const request: GetPriceHistoryRequest = { tokenId };
  const { tokenId: id, options } = parseGetPriceHistoryRequest(request);

  let sdkResp: Awaited<ReturnType<typeof sdk.getPriceHistory>>;
  try {
    sdkResp = await sdk.getPriceHistory(id, options);
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getPriceHistory 抛错', { cause: error });
  }

  if (sdkResp.errno !== 0) {
    grpcClient.close();
    if (isLikelySdkUpstreamRegionBlock(sdkResp)) {
      t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过与 gRPC 的字节级对照');
      return;
    }

    assert.fail(`官方 SDK getPriceHistory 失败：errno=${sdkResp.errno} errmsg=${sdkResp.errmsg}`);
  }

  const expected = toGetPriceHistoryResponse(sdkResp);

  let actual: GetPriceHistoryResponse;
  try {
    actual = await getPriceHistory(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetPriceHistoryResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「parseGetPriceHistoryRequest + SDK + toGetPriceHistoryResponse」一致；若偶发失败，可能是两次请求之间历史数据追加。',
  );
});
