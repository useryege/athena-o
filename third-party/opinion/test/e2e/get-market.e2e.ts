/**
 * E2E: OpinionService.GetMarket
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 *
 * 可选：`OPINION_E2E_MARKET_ID` 指定十进制 marketId；未设置时会先调用官方 SDK `getMarkets(limit:1)` 取首条 id，
 * 因此本文件在部分场景下会对上游发起 **多于两次** HTTP 请求（解析 id + SDK GetMarket + gRPC GetMarket），
 * 详情字段在两次 GetMarket 之间若发生变化可能导致字节级对照偶发失败。
 *
 * 与侧链共用同一 `.env`（`loadRuntimeConfig`），以便官方 SDK 直连与 gRPC 使用相同配置。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test, { type TestContext } from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import {
  parseGetMarketRequest,
  parseGetMarketsRequest,
  toGetMarketDetailResponse,
} from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetMarketRequest,
  type GetMarketResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetMarketResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  isLikelySdkUpstreamRegionBlock,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getMarket(client: OpinionGrpcClient, request: GetMarketRequest): Promise<GetMarketResponse> {
  return new Promise((resolve, reject) => {
    client.getMarket(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetMarket returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

async function resolveE2eMarketId(
  sdk: ReturnType<typeof createOpinionClient>,
  t: TestContext,
): Promise<string | undefined> {
  const fromEnv = process.env.OPINION_E2E_MARKET_ID?.trim();
  if (fromEnv) {
    return fromEnv;
  }

  const listQuery = parseGetMarketsRequest({ limit: 1 });
  let listResp: Awaited<ReturnType<typeof sdk.getMarkets>>;
  try {
    listResp = await sdk.getMarkets(listQuery);
  } catch (error) {
    throw new Error('官方 SDK getMarkets（解析 E2E marketId）抛错', { cause: error });
  }

  if (listResp.errno !== 0) {
    if (isLikelySdkUpstreamRegionBlock(listResp)) {
      t.skip('上游 Opinion API 当前区域不可用（SDK errno），无法解析 E2E marketId');
      return undefined;
    }

    assert.fail(`官方 SDK getMarkets 失败：errno=${listResp.errno} errmsg=${listResp.errmsg}`);
  }

  const first = listResp.result?.list?.[0];
  if (!first || typeof first !== 'object') {
    t.skip('市场列表为空，无法解析 E2E marketId（可设置 OPINION_E2E_MARKET_ID）');
    return undefined;
  }

  const rawId = (first as Record<string, unknown>).marketId;
  const id = rawId == null ? '' : String(rawId).trim();
  if (!id || !/^\d+$/.test(id)) {
    t.skip('首条市场缺少合法十进制 marketId');
    return undefined;
  }

  return id;
}

test('OpinionService.GetMarket — 成功返回 errno=0 且 data 结构合理', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  try {
    const marketId = await resolveE2eMarketId(sdk, t);
    if (marketId === undefined) {
      return;
    }

    const request: GetMarketRequest = { marketId, useCache: true };
    let response: GetMarketResponse;

    try {
      response = await getMarket(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(response.errno, 0, `期望 errno=0，实际 ${response.errno} errmsg=${response.errmsg}`);
    assert.equal(typeof response.errmsg, 'string');
    assert.ok(response.data, 'errno=0 时期望返回 data');
    assert.equal(typeof response.data?.marketId, 'string');
  } finally {
    client.close();
  }
});

test('OpinionService.GetMarket — 官方 SDK 与 gRPC 返回（经同一 toGetMarketDetailResponse 语义）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const marketId = await resolveE2eMarketId(sdk, t);
  if (marketId === undefined) {
    grpcClient.close();
    return;
  }

  const request: GetMarketRequest = { marketId, useCache: true };
  const { marketId: idNum, useCache } = parseGetMarketRequest(request);

  let sdkResp: Awaited<ReturnType<typeof sdk.getMarket>>;
  try {
    sdkResp = await sdk.getMarket(idNum, useCache);
  } catch (error) {
    grpcClient.close();
    throw new Error('官方 SDK getMarket 抛错', { cause: error });
  }

  if (sdkResp.errno !== 0) {
    grpcClient.close();
    if (isLikelySdkUpstreamRegionBlock(sdkResp)) {
      t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过与 gRPC 的字节级对照');
      return;
    }

    assert.fail(`官方 SDK getMarket 失败：errno=${sdkResp.errno} errmsg=${sdkResp.errmsg}`);
  }

  const expected = toGetMarketDetailResponse(sdkResp);

  let actual: GetMarketResponse;
  try {
    actual = await getMarket(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetMarketResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「parseGetMarketRequest + SDK getMarket + toGetMarketDetailResponse」一致；若偶发失败，可能是两次 GetMarket 之间上游数据变化。',
  );
});
