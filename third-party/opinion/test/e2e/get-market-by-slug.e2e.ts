/**
 * E2E: OpinionService.GetMarketBySlug
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 *
 * 与侧链共用同一 `.env`（`loadRuntimeConfig`），以便官方 SDK 直连与 gRPC 使用相同配置。
 *
 * Oracle：`parseGetMarketBySlugRequest` → SDK `getMarketBySlug` → `toGetMarketDetailResponse`（与 `src/service.ts` 一致）。
 * 先 `getMarkets` 取一条带 `slug` 的样本，再各请求上游一次（SDK 与经 gRPC）；详情快照在极端情况下可能略有差异。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetMarketBySlugRequest, toGetMarketDetailResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetMarketBySlugRequest,
  type GetMarketResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetMarketResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  isLikelySdkUpstreamRegionBlock,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getMarketBySlug(client: OpinionGrpcClient, request: GetMarketBySlugRequest): Promise<GetMarketResponse> {
  return new Promise((resolve, reject) => {
    client.getMarketBySlug(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }
      if (!response) {
        reject(new Error('GetMarketBySlug returned empty response'));
        return;
      }
      resolve(response);
    });
  });
}

test('OpinionService.GetMarketBySlug — 官方 SDK 与 gRPC 返回（经同一 toGetMarketDetailResponse 语义）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());

  try {
    const sdk = createOpinionClient(loadRuntimeConfig().sdk);

    let listResp: Awaited<ReturnType<typeof sdk.getMarkets>>;
    try {
      listResp = await sdk.getMarkets({ page: 1, limit: 10 });
    } catch (error) {
      throw new Error('官方 SDK getMarkets 抛错', { cause: error });
    }

    if (listResp.errno !== 0) {
      if (isLikelySdkUpstreamRegionBlock(listResp)) {
        t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过');
        return;
      }
      assert.fail(`官方 SDK getMarkets 失败：errno=${listResp.errno} errmsg=${listResp.errmsg}`);
    }

    const withSlug = listResp.result?.list?.find((m) => typeof m.slug === 'string' && m.slug.trim().length > 0);
    if (!withSlug?.slug) {
      t.skip('列表中无带 slug 的市场样本，跳过');
      return;
    }

    const request: GetMarketBySlugRequest = { slug: withSlug.slug.trim() };

    let sdkResp: Awaited<ReturnType<typeof sdk.getMarketBySlug>>;
    try {
      sdkResp = await sdk.getMarketBySlug(parseGetMarketBySlugRequest(request));
    } catch (error) {
      throw new Error('官方 SDK getMarketBySlug 抛错', { cause: error });
    }

    if (sdkResp.errno !== 0) {
      if (isLikelySdkUpstreamRegionBlock(sdkResp)) {
        t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过');
        return;
      }
      assert.fail(`官方 SDK getMarketBySlug 失败：errno=${sdkResp.errno} errmsg=${sdkResp.errmsg}`);
    }

    const expected = toGetMarketDetailResponse(sdkResp);

    let actual: GetMarketResponse;
    try {
      actual = await getMarketBySlug(grpcClient, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }
      throw error;
    }

    assertGetMarketResponsesEqual(
      expected,
      actual,
      'gRPC 应与 parseGetMarketBySlugRequest + SDK + toGetMarketDetailResponse 一致',
    );
  } finally {
    grpcClient.close();
  }
});
