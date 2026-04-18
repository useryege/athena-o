import { InvalidParamError, OpenApiError } from '@opinion-labs/opinion-clob-sdk';
import type { ServiceError } from '@grpc/grpc-js';
import { status as grpcStatus } from '@grpc/grpc-js';

export function createServiceError(code: number, message: string): ServiceError {
  const error = new Error(message) as ServiceError;
  error.code = code;
  error.details = message;
  return error;
}

export function toGrpcError(error: unknown): ServiceError {
  if (error && typeof error === 'object' && 'code' in error && typeof error.code === 'number') {
    return error as ServiceError;
  }

  if (error instanceof InvalidParamError) {
    return createServiceError(grpcStatus.INVALID_ARGUMENT, error.message);
  }

  if (error instanceof OpenApiError) {
    return createServiceError(grpcStatus.INTERNAL, error.message);
  }

  if (error instanceof Error) {
    return createServiceError(grpcStatus.INTERNAL, error.message);
  }

  return createServiceError(grpcStatus.INTERNAL, 'unknown opinion service error');
}
