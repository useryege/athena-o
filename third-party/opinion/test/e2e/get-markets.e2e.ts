/**
 * E2E: OpinionService.GetMarkets
 *
 * 前置：已在本机启动 opinion gRPC 服务（例如 `npm run dev` 或 `npm start`）。
 * 地址：`OPINION_E2E_GRPC_ADDR`（默认 `127.0.0.1:50051`，需与 `GRPC_PORT` 一致）。
 */
import assert from 'node:assert/strict';
import test from 'node:test';

import { credentials } from '@grpc/grpc-js';

import {
  OpinionServiceClient,
  type GetMarketsRequest,
  type GetMarketsResponse,
} from '../../src/gen/opinion/opinion.js';
import { getE2EGrpcAddress, handleE2EUnaryError } from './helpers.js';

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
