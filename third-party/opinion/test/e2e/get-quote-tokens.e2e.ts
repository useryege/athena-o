/**
 * E2E: OpinionService.GetQuoteTokens
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 *
 * 与侧链共用同一 `.env`（`loadRuntimeConfig`），以便官方 SDK 直连与 gRPC 使用相同配置。
 *
 * Oracle：`parseGetQuoteTokensRequest` → SDK `getQuoteTokens` → `toGetQuoteTokensResponse`（与 `src/service.ts` 一致）。
 * 一致性用例会先后请求上游两次（SDK 一次、gRPC 经侧链再请求一次），极端情况下列表快照可能略有差异。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetQuoteTokensRequest, toGetQuoteTokensResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetQuoteTokensRequest,
  type GetQuoteTokensResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetQuoteTokensResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  isLikelySdkUpstreamRegionBlock,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getQuoteTokens(client: OpinionGrpcClient, request: GetQuoteTokensRequest): Promise<GetQuoteTokensResponse> {
  return new Promise((resolve, reject) => {
    client.getQuoteTokens(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }
      if (!response) {
        reject(new Error('GetQuoteTokens returned empty response'));
        return;
      }
      resolve(response);
    });
  });
}

test('OpinionService.GetQuoteTokens — 官方 SDK 与 gRPC 返回（经同一 toGetQuoteTokensResponse 语义）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());

  try {
    const sdk = createOpinionClient(loadRuntimeConfig().sdk);

    const request: GetQuoteTokensRequest = { useCache: true };
    const useCache = parseGetQuoteTokensRequest(request);

    let sdkResp: Awaited<ReturnType<typeof sdk.getQuoteTokens>>;
    try {
      sdkResp = await sdk.getQuoteTokens(useCache);
    } catch (error) {
      throw new Error('官方 SDK getQuoteTokens 抛错', { cause: error });
    }

    if (sdkResp.errno !== 0) {
      if (isLikelySdkUpstreamRegionBlock(sdkResp)) {
        t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过');
        return;
      }
      assert.fail(`官方 SDK getQuoteTokens 失败：errno=${sdkResp.errno} errmsg=${sdkResp.errmsg}`);
    }

    const expected = toGetQuoteTokensResponse(sdkResp);

    let actual: GetQuoteTokensResponse;
    try {
      actual = await getQuoteTokens(grpcClient, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }
      throw error;
    }

    assertGetQuoteTokensResponsesEqual(
      expected,
      actual,
      'gRPC 应与 parseGetQuoteTokensRequest + SDK + toGetQuoteTokensResponse 一致；若偶发失败，可能是两次请求之间上游列表变化。',
    );
  } finally {
    grpcClient.close();
  }
});
