/**
 * E2E: OpinionService.GetMarkets
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 *
 * 与侧链共用同一 `.env`（`loadRuntimeConfig`），以便官方 SDK 直连与 gRPC 使用相同配置。
 *
 * 一致性用例会先后请求上游两次（SDK 一次、gRPC 经侧链再请求一次），极端情况下列表快照可能略有差异。
 */
import 'dotenv/config';

import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import { createOpinionClient } from '../../src/client.js';
import { parseGetMarketsRequest, toGetMarketsResponse } from '../../src/common.js';
import { loadRuntimeConfig } from '../../src/config.js';
import {
  OpinionServiceClient,
  type GetMarketsRequest,
  type GetMarketsResponse,
} from '../../src/gen/opinion/opinion.js';
import {
  assertGetMarketsResponsesEqual,
  getE2EGrpcAddress,
  handleE2EUnaryError,
  isLikelySdkUpstreamRegionBlock,
} from './helpers.js';

type OpinionGrpcClient = InstanceType<typeof OpinionServiceClient>;

function getMarkets(client: OpinionGrpcClient, request: GetMarketsRequest): Promise<GetMarketsResponse> {
  return new Promise((resolve, reject) => {
    client.getMarkets(request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }

      if (!response) {
        reject(new Error('GetMarkets returned empty response'));
        return;
      }

      resolve(response);
    });
  });
}

test('OpinionService.GetMarkets — 成功返回 errno=0 且 list/total 结构正确', async (t) => {
  const address = getE2EGrpcAddress();
  const client = new OpinionServiceClient(address, credentials.createInsecure());

  try {
    const request: GetMarketsRequest = { limit: 5 };
    let response: GetMarketsResponse;

    try {
      response = await getMarkets(client, request);
    } catch (error) {
      if (handleE2EUnaryError(error, address, t)) {
        return;
      }

      throw error;
    }

    assert.equal(response.errno, 0, `期望 errno=0，实际 ${response.errno} errmsg=${response.errmsg}`);
    assert.equal(typeof response.errmsg, 'string');
    assert.equal(typeof response.total, 'number');
    assert.ok(Array.isArray(response.list), 'list 应为数组');

    for (const m of response.list) {
      assert.ok(m && typeof m === 'object', '每条 Market 应为对象');
    }
  } finally {
    client.close();
  }
});

test('OpinionService.GetMarkets — 官方 SDK 与 gRPC 返回（经同一 toGetMarketsResponse 语义）一致', async (t) => {
  const address = getE2EGrpcAddress();
  const grpcClient = new OpinionServiceClient(address, credentials.createInsecure());
  const sdk = createOpinionClient(loadRuntimeConfig().sdk);

  const request: GetMarketsRequest = { limit: 5 };
  const query = parseGetMarketsRequest(request);

  let sdkResp: Awaited<ReturnType<typeof sdk.getMarkets>>;
  try {
    sdkResp = await sdk.getMarkets(query);
  } catch (error) {
    throw new Error('官方 SDK getMarkets 抛错', { cause: error });
  }

  if (sdkResp.errno !== 0) {
    if (isLikelySdkUpstreamRegionBlock(sdkResp)) {
      t.skip('上游 Opinion API 当前区域不可用（SDK errno），跳过与 gRPC 的字节级对照');
      return;
    }

    assert.fail(`官方 SDK getMarkets 失败：errno=${sdkResp.errno} errmsg=${sdkResp.errmsg}`);
  }

  const expected = toGetMarketsResponse(sdkResp);

  let actual: GetMarketsResponse;
  try {
    actual = await getMarkets(grpcClient, request);
  } catch (error) {
    if (handleE2EUnaryError(error, address, t)) {
      return;
    }

    throw error;
  } finally {
    grpcClient.close();
  }

  assertGetMarketsResponsesEqual(
    expected,
    actual,
    'gRPC 响应应与「parseGetMarketsRequest + SDK + toGetMarketsResponse」一致；若偶发失败，可能是两次上游请求之间列表变化。',
  );
});
