/**
 * 共享：gRPC 地址、上游地域错误、连接失败提示。
 */
import { status } from '@grpc/grpc-js';
import type { ServiceError } from '@grpc/grpc-js';
import type { TestContext } from 'node:test';

export function getE2EGrpcAddress(): string {
  return process.env.OPINION_E2E_GRPC_ADDR?.trim() || '127.0.0.1:50051';
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
  return /restricted jurisdiction|United States|located in China|not available to persons/i.test(msg);
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
