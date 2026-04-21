/**
 * Shared E2E helpers (gRPC address, unary error handling).
 */
import { status } from '@grpc/grpc-js';
import type { ServiceError } from '@grpc/grpc-js';

export function getE2EGrpcAddress(): string {
  return process.env.POLYMARKET_E2E_GRPC_ADDR?.trim() || '127.0.0.1:50051';
}

export function handleE2EUnaryError(error: unknown, address: string): never {
  const se = error as ServiceError;
  if (se.code === status.UNAVAILABLE) {
    throw new Error(
      `无法连接 gRPC（${address}）。请先在本目录启动服务后再执行：npm run test:e2e（可用 POLYMARKET_E2E_GRPC_ADDR 覆盖地址）`,
      { cause: error },
    );
  }

  throw error;
}
