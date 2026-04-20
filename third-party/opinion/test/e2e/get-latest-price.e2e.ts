/**
 * E2E: OpinionService.GetLatestPrice
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 *
 * 可选：`OPINION_E2E_TOKEN_ID`；未设置时先 `getMarkets(limit:1)` 取首条 `yesTokenId`。
 * 本用例会 **两次** 拉取最新价，盘中价格可能在两次请求之间变动导致字节级对照偶发失败。
 *
 * 与侧链共用同一 `.env`（`loadRuntimeConfig`）。Oracle：`parseGetLatestPriceRequest` → SDK `getLatestPrice` → `toGetLatestPriceResponse`。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetLatestPriceRequest, toGetLatestPriceResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetLatestPriceRequest,
  type GetLatestPriceResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetLatestPriceResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  isLikelySdkUpstreamRegionBlock,
  resolveE2eTokenId,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getLatestPrice(
  client: OpinionGrpcClient,
  request: GetLatestPriceRequest,
): Promise<GetLatestPriceResponse> {
  return new Promise((resolve, reject) => {
    client.getLatestPrice(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetLatestPrice returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetLatestPrice — 成功返回 errno=0', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  try {
    const tokenId = await resolveE2eTokenId(sdk, t);
    if (tokenId === undefined) {
      return;
    }

    const request: GetLatestPriceRequest = { tokenId };
    let response: GetLatestPriceResponse;

    try {
      response = await getLatestPrice(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(response.errno, 0, `期望 errno=0，实际 ${response.errno} errmsg=${response.errmsg}`);
    assert.equal(typeof response.errmsg, 'string');
  } finally {
    client.close();
  }
});

test('OpinionService.GetLatestPrice — 官方 SDK 与 gRPC 返回（经同一 toGetLatestPriceResponse）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const tokenId = await resolveE2eTokenId(sdk, t);
  if (tokenId === undefined) {
    grpcClient.close();
    return;
  }

  const request: GetLatestPriceRequest = { tokenId };
  const parsed = parseGetLatestPriceRequest(request);

  let sdkResp: Awaited<ReturnType<typeof sdk.getLatestPrice>>;
  try {
    sdkResp = await sdk.getLatestPrice(parsed);
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getLatestPrice 抛错', { cause: error });
  }

  if (sdkResp.errno !== 0) {
    grpcClient.close();
    if (isLikelySdkUpstreamRegionBlock(sdkResp)) {
      t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过与 gRPC 的字节级对照');
      return;
    }

    assert.fail(`官方 SDK getLatestPrice 失败：errno=${sdkResp.errno} errmsg=${sdkResp.errmsg}`);
  }

  const expected = toGetLatestPriceResponse(sdkResp);

  let actual: GetLatestPriceResponse;
  try {
    actual = await getLatestPrice(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetLatestPriceResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「parseGetLatestPriceRequest + SDK + toGetLatestPriceResponse」一致；若偶发失败，可能是两次请求之间成交价变化。',
  );
});
