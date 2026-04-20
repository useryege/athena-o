/**
 * E2E: OpinionService.GetCategoricalMarket
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 *
 * 与侧链共用同一 `.env`（`loadRuntimeConfig`），以便官方 SDK 直连与 gRPC 使用相同配置。
 *
 * Oracle：`parseGetCategoricalMarketRequest` → SDK `getCategoricalMarket` → `toGetMarketDetailResponse`（与 `src/service.ts` 一致）。
 * 先 `getMarkets(topicType: CATEGORICAL)` 取样本 `marketId`，再各请求上游一次（SDK 与经 gRPC）；详情快照在极端情况下可能略有差异。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { TopicType } from '@opinion-labs/opinion-clob-sdk';
import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetCategoricalMarketRequest, toGetMarketDetailResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetCategoricalMarketRequest,
  type GetMarketResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetMarketResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  isLikelySdkUpstreamRegionBlock,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getCategoricalMarket(
  client: OpinionGrpcClient,
  request: GetCategoricalMarketRequest,
): Promise<GetMarketResponse> {
  return new Promise((resolve, reject) => {
    client.getCategoricalMarket(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }
      if (!response) {
        reject(new Error('GetCategoricalMarket returned empty response'));
        return;
      }
      resolve(response);
    });
  });
}

test('OpinionService.GetCategoricalMarket — 官方 SDK 与 gRPC 返回（经同一 toGetMarketDetailResponse 语义）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());

  try {
    const sdk = createOpinionClient(loadRuntimeConfig().sdk);

    let listResp: Awaited<ReturnType<typeof sdk.getMarkets>>;
    try {
      listResp = await sdk.getMarkets({ topicType: TopicType.CATEGORICAL, page: 1, limit: 5 });
    } catch (error) {
      throw new Error('官方 SDK getMarkets(CATEGORICAL) 抛错', { cause: error });
    }

    if (listResp.errno !== 0) {
      if (isLikelySdkUpstreamRegionBlock(listResp)) {
        t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过');
        return;
      }
      assert.fail(`官方 SDK getMarkets 失败：errno=${listResp.errno} errmsg=${listResp.errmsg}`);
    }

    const first = listResp.result?.list?.[0];
    if (!first || typeof first.marketId !== 'number') {
      t.skip('当前环境未返回 categorical 市场样本，跳过');
      return;
    }

    const request: GetCategoricalMarketRequest = { marketId: String(first.marketId) };

    let sdkResp: Awaited<ReturnType<typeof sdk.getCategoricalMarket>>;
    try {
      sdkResp = await sdk.getCategoricalMarket(parseGetCategoricalMarketRequest(request));
    } catch (error) {
      throw new Error('官方 SDK getCategoricalMarket 抛错', { cause: error });
    }

    if (sdkResp.errno !== 0) {
      if (isLikelySdkUpstreamRegionBlock(sdkResp)) {
        t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过');
        return;
      }
      assert.fail(`官方 SDK getCategoricalMarket 失败：errno=${sdkResp.errno} errmsg=${sdkResp.errmsg}`);
    }

    const expected = toGetMarketDetailResponse(sdkResp);

    let actual: GetMarketResponse;
    try {
      actual = await getCategoricalMarket(grpcClient, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }
      throw error;
    }

    assertGetMarketResponsesEqual(
      expected,
      actual,
      'gRPC 应与 parseGetCategoricalMarketRequest + SDK + toGetMarketDetailResponse 一致',
    );
  } finally {
    grpcClient.close();
  }
});
