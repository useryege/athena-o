/**
 * 共享：gRPC 地址、上游地域错误、连接失败提示。
 */
import { status } from '@grpc/grpc-js';
import type { ServiceError } from '@grpc/grpc-js';
import type { TestContext } from 'node:test';

import assert from 'node:assert/strict';

import type { Client } from '@opinion-labs/opinion-clob-sdk';

import { parseGetMarketsRequest, parseGetMyOrdersRequest } from '../../src/common.js';
import {
  GetFeeRatesResponse,
  GetLatestPriceResponse,
  GetMarketResponse,
  GetMarketsResponse,
  GetMyBalancesResponse,
  GetMyOrdersResponse,
  GetMyPositionsResponse,
  GetMyTradesResponse,
  GetOrderByIdResponse,
  GetOrderbookResponse,
  GetPriceHistoryResponse,
  GetQuoteTokensResponse,
  GetUserAuthResponse,
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

const SDK_AUTH_OR_CONFIG_RE =
  /unauthorized|invalid.*key|apikey|api key|401|403|forbidden|signature|not.*authenticated|authentication required|wallet|credential/i;

/** 用户类接口在无 API key 或配置不全时，`errno !== 0` 且 `errmsg` 常含认证相关提示（启发式，用于 E2E skip）。 */
export function isLikelySdkAuthOrConfigFailure(response: { errno: number; errmsg?: string }): boolean {
  if (response.errno === 0) {
    return false;
  }

  return SDK_AUTH_OR_CONFIG_RE.test(`${response.errmsg ?? ''}`);
}

/**
 * 用户类 RPC 的 SDK 路径：非 0 时区分地域限制、认证/配置问题与其它错误（其它错误会 `assert.fail` 抛出）。
 * @returns errno=0 时为 false；已 `t.skip` 时为 true。
 */
export function handleSdkUserEndpointResponse(
  sdkResp: { errno: number; errmsg?: string },
  t: TestContext,
  label: string,
): boolean {
  if (sdkResp.errno === 0) {
    return false;
  }

  if (isLikelySdkUpstreamRegionBlock(sdkResp)) {
    t.skip(`${label}：上游 Opinion API 当前区域不可用（SDK errno）`);
    return true;
  }

  if (isLikelySdkAuthOrConfigFailure(sdkResp)) {
    t.skip(
      `${label}：SDK 返回认证或配置相关错误，跳过（请检查 .env 中 API key、链 ID 等；无订单时可设置 OPINION_E2E_ORDER_ID）`,
    );
    return true;
  }

  assert.fail(`官方 SDK ${label} 失败：errno=${sdkResp.errno} errmsg=${sdkResp.errmsg}`);
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

export function assertGetMyOrdersResponsesEqual(
  expected: GetMyOrdersResponse,
  actual: GetMyOrdersResponse,
  message?: string,
): void {
  const a = GetMyOrdersResponse.encode(expected).finish();
  const b = GetMyOrdersResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

export function assertGetOrderByIdResponsesEqual(
  expected: GetOrderByIdResponse,
  actual: GetOrderByIdResponse,
  message?: string,
): void {
  const a = GetOrderByIdResponse.encode(expected).finish();
  const b = GetOrderByIdResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

export function assertGetMyBalancesResponsesEqual(
  expected: GetMyBalancesResponse,
  actual: GetMyBalancesResponse,
  message?: string,
): void {
  const a = GetMyBalancesResponse.encode(expected).finish();
  const b = GetMyBalancesResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

export function assertGetMyPositionsResponsesEqual(
  expected: GetMyPositionsResponse,
  actual: GetMyPositionsResponse,
  message?: string,
): void {
  const a = GetMyPositionsResponse.encode(expected).finish();
  const b = GetMyPositionsResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

export function assertGetMyTradesResponsesEqual(
  expected: GetMyTradesResponse,
  actual: GetMyTradesResponse,
  message?: string,
): void {
  const a = GetMyTradesResponse.encode(expected).finish();
  const b = GetMyTradesResponse.encode(actual).finish();
  assert.deepStrictEqual(a, b, message);
}

export function assertGetUserAuthResponsesEqual(
  expected: GetUserAuthResponse,
  actual: GetUserAuthResponse,
  message?: string,
): void {
  const a = GetUserAuthResponse.encode(expected).finish();
  const b = GetUserAuthResponse.encode(actual).finish();
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
 * 解析 E2E 用的订单 ID：`OPINION_E2E_ORDER_ID`，否则在 SDK 已可用时 `getMyOrders(page:1,limit:1)` 取首条 `orderId`（与 `parseGetMyOrdersRequest` 一致）。
 */
export async function resolveE2eOrderId(sdk: Client, t: TestContext): Promise<string | undefined> {
  const fromEnv = process.env.OPINION_E2E_ORDER_ID?.trim();
  if (fromEnv) {
    return fromEnv;
  }

  const listQuery = parseGetMyOrdersRequest({ page: 1, limit: 1 });
  let listResp: Awaited<ReturnType<Client['getMyOrders']>>;
  try {
    listResp = await sdk.getMyOrders(listQuery);
  } catch (error) {
    throw new Error('官方 SDK getMyOrders（解析 E2E orderId）抛错', { cause: error });
  }

  if (listResp.errno !== 0) {
    if (isLikelySdkUpstreamRegionBlock(listResp)) {
      t.skip('上游 Opinion API 当前区域不可用（SDK errno），无法解析 E2E orderId');
      return undefined;
    }
    if (isLikelySdkAuthOrConfigFailure(listResp)) {
      t.skip('无法解析 E2E orderId：SDK 认证/配置错误（可设置 OPINION_E2E_ORDER_ID）');
      return undefined;
    }

    assert.fail(`官方 SDK getMyOrders 失败：errno=${listResp.errno} errmsg=${listResp.errmsg}`);
  }

  const first = listResp.result?.list?.[0];
  if (!first || typeof first !== 'object') {
    t.skip('订单列表为空，无法解析 E2E orderId（可设置 OPINION_E2E_ORDER_ID）');
    return undefined;
  }

  const raw = (first as Record<string, unknown>).orderId;
  const orderId = raw == null ? '' : String(raw).trim();
  if (!orderId) {
    t.skip('首条订单缺少 orderId（可设置 OPINION_E2E_ORDER_ID）');
    return undefined;
  }

  return orderId;
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
