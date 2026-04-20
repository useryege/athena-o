/**
 * 共享：gRPC 地址、上游地域错误、连接失败提示。
 */
import { status } from '@grpc/grpc-js';
import type { ServiceError } from '@grpc/grpc-js';
import type { TestContext } from 'node:test';

import assert from 'node:assert/strict';

import type { Client } from '@opinion-labs/opinion-clob-sdk';

import { parseGetMarketsRequest } from '../../src/common.js';
import {
  GetFeeRatesResponse,
  GetLatestPriceResponse,
  GetMarketResponse,
  GetMarketsResponse,
  GetOrderbookResponse,
  GetPriceHistoryResponse,
  GetQuoteTokensResponse,
} from '../../src/gen/opinion/opinion.js';

export function getE2EGrpcAddress(): string {
  return process.env.OPINION_E2E_GRPC_ADDR?.trim() || '127.0.0.1:50051';
}

const UPSTREAM_REGION_BLOCK_RE =
  /restricted jurisdiction|United States|located in China|not available to persons/i;

/** 与 gRPC INTERNAL 地域提示、官方 SDK `errmsg` 中可能出现的文案对齐。 */
export function isLikelyUpstreamRegionBlockMessage(msg: string): boolean {
  return UPSTREAM_REGION_BLOCK_RE.test(msg);
}

/** 官方 SDK 直连失败时，`errno !== 0` 且 `errmsg` 可能含地域限制文案。 */
export function isLikelySdkUpstreamRegionBlock(response: { errno: number; errmsg?: string }): boolean {
  if (response.errno === 0) {
    return false;
  }

  return isLikelyUpstreamRegionBlockMessage(`${response.errmsg ?? ''}`);
}

/** 上游 HTTP API 在部分国家/地区不可用，侧链会映射为 gRPC INTERNAL。 */
export function isLikelyUpstreamRegionBlock(error: unknown): boolean {
  if (!error || typeof error !== 'object') {
    return false;
  }

  const e = error as ServiceError;
  if (e.code !== status.INTERNAL) {
    return false;
  }

  const msg = `${e.details ?? ''}${e.message ?? ''}`;
  return isLikelyUpstreamRegionBlockMessage(msg);
}

/**
 * 使用 protobuf 二进制比较两条 GetMarketsResponse（与侧链 `toGetMarketsResponse` 输出对照时使用）。
 */
export function assertGetMarketsResponsesEqual(
  expected: import('../../src/gen/opinion/opinion.js').GetMarketsResponse,
  actual: import('../../src/gen/opinion/opinion.js').GetMarketsResponse,
  message?: string,
): void {
  const a = GetMarketsResponse.encode(expected).finish();
  const b = GetMarketsResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

/**
 * 使用 protobuf 二进制比较两条 GetMarketResponse（与侧链 `toGetMarketDetailResponse` 输出对照时使用）。
 */
export function assertGetMarketResponsesEqual(
  expected: import('../../src/gen/opinion/opinion.js').GetMarketResponse,
  actual: import('../../src/gen/opinion/opinion.js').GetMarketResponse,
  message?: string,
): void {
  const a = GetMarketResponse.encode(expected).finish();
  const b = GetMarketResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

/**
 * 使用 protobuf 二进制比较两条 GetQuoteTokensResponse（与侧链 `toGetQuoteTokensResponse` 输出对照时使用）。
 */
export function assertGetQuoteTokensResponsesEqual(
  expected: import('../../src/gen/opinion/opinion.js').GetQuoteTokensResponse,
  actual: import('../../src/gen/opinion/opinion.js').GetQuoteTokensResponse,
  message?: string,
): void {
  const a = GetQuoteTokensResponse.encode(expected).finish();
  const b = GetQuoteTokensResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

/**
 * 使用 protobuf 二进制比较两条 GetOrderbookResponse。
 */
export function assertGetOrderbookResponsesEqual(
  expected: GetOrderbookResponse,
  actual: GetOrderbookResponse,
  message?: string,
): void {
  const a = GetOrderbookResponse.encode(expected).finish();
  const b = GetOrderbookResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

/**
 * 使用 protobuf 二进制比较两条 GetLatestPriceResponse。
 */
export function assertGetLatestPriceResponsesEqual(
  expected: GetLatestPriceResponse,
  actual: GetLatestPriceResponse,
  message?: string,
): void {
  const a = GetLatestPriceResponse.encode(expected).finish();
  const b = GetLatestPriceResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

/**
 * 使用 protobuf 二进制比较两条 GetPriceHistoryResponse。
 */
export function assertGetPriceHistoryResponsesEqual(
  expected: GetPriceHistoryResponse,
  actual: GetPriceHistoryResponse,
  message?: string,
): void {
  const a = GetPriceHistoryResponse.encode(expected).finish();
  const b = GetPriceHistoryResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

/**
 * 使用 protobuf 二进制比较两条 GetFeeRatesResponse（侧链 `toGetFeeRatesResponse` 输出）。
 */
export function assertGetFeeRatesResponsesEqual(
  expected: GetFeeRatesResponse,
  actual: GetFeeRatesResponse,
  message?: string,
): void {
  const a = GetFeeRatesResponse.encode(expected).finish();
  const b = GetFeeRatesResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

/**
 * 解析 E2E 用的 outcome tokenId：`OPINION_E2E_TOKEN_ID`，否则从官方 SDK `getMarkets(limit:1)` 首条的 `yesTokenId` 取值（与 `parseGetMarketsRequest` 一致）。
 */
export async function resolveE2eTokenId(sdk: Client, t: TestContext): Promise<string | undefined> {
  const fromEnv = process.env.OPINION_E2E_TOKEN_ID?.trim();
  if (fromEnv) {
    return fromEnv;
  }

  const listQuery = parseGetMarketsRequest({ limit: 1 });
  let listResp: Awaited<ReturnType<Client['getMarkets']>>;
  try {
    listResp = await sdk.getMarkets(listQuery);
  } catch (error) {
    throw new Error('官方 SDK getMarkets（解析 E2E tokenId）抛错', { cause: error });
  }

  if (listResp.errno !== 0) {
    if (isLikelySdkUpstreamRegionBlock(listResp)) {
      t.skip('上游 Opinion API 当前区域不可用（SDK errno），无法解析 E2E tokenId');
      return undefined;
    }

    assert.fail(`官方 SDK getMarkets 失败：errno=${listResp.errno} errmsg=${listResp.errmsg}`);
  }

  const first = listResp.result?.list?.[0];
  if (!first || typeof first !== 'object') {
    t.skip('市场列表为空，无法解析 E2E tokenId（可设置 OPINION_E2E_TOKEN_ID）');
    return undefined;
  }

  const raw = (first as Record<string, unknown>).yesTokenId;
  const tokenId = raw == null ? '' : String(raw).trim();
  if (!tokenId) {
    t.skip('首条市场缺少 yesTokenId（可设置 OPINION_E2E_TOKEN_ID）');
    return undefined;
  }

  return tokenId;
}

/**
 * 处理 unary 错误：UNAVAILABLE → 抛错；地域限制 → `t.skip` 并返回 true；其它 → 抛出。
 */
export function handleE2EUnaryError(error: unknown, address: string, t: TestContext): boolean {
  const se = error as ServiceError;
  if (se.code === status.UNAVAILABLE) {
    throw new Error(
      `无法连接 gRPC（${address}）。请先在本目录启动服务后再执行：npm run test:e2e（可用 OPINION_E2E_GRPC_ADDR 覆盖地址）`,
      { cause: error },
    );
  }

  if (isLikelyUpstreamRegionBlock(error)) {
    t.skip('上游 Opinion API 当前区域不可用，跳过本用例断言（RPC 与侧链仍已连通）');
    return true;
  }

  throw error;
}
