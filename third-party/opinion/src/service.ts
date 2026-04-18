import { InvalidParamError, TopicSortType, TopicStatusFilter, TopicType } from '@opinion-labs/opinion-clob-sdk';
import type { ServiceError } from '@grpc/grpc-js';
import { status as grpcStatus } from '@grpc/grpc-js';

import { getOpinionClient } from './client.js';
import { mapGetMarketsResponse } from './mappers.js';
import type { GetMarketsRequest, OpinionServiceHandlers } from './types.js';

function createServiceError(code: number, message: string): ServiceError {
  const error = new Error(message) as ServiceError;
  error.code = code;
  error.details = message;
  return error;
}

function normalizePositiveInteger(name: string, value: number | undefined): number | undefined {
  if (!value) {
    return undefined;
  }

  if (!Number.isInteger(value) || value < 1) {
    throw createServiceError(grpcStatus.INVALID_ARGUMENT, `${name} must be >= 1`);
  }

  return value;
}

function toSdkTopicType(value: number | undefined): TopicType {
  switch (value) {
    case 1:
      return TopicType.BINARY;
    case 2:
      return TopicType.CATEGORICAL;
    case 3:
    case 0:
    case undefined:
      return TopicType.ALL;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported topic_type: ${value}`);
  }
}

function toSdkStatusFilter(value: number | undefined): TopicStatusFilter | undefined {
  switch (value) {
    case 0:
    case 1:
    case undefined:
      return undefined;
    case 2:
      return TopicStatusFilter.ACTIVATED;
    case 3:
      return TopicStatusFilter.RESOLVED;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported status filter: ${value}`);
  }
}

function toSdkSortBy(value: number | undefined): TopicSortType | undefined {
  switch (value) {
    case 0:
    case undefined:
      return undefined;
    case 1:
      return TopicSortType.BY_TIME_DESC;
    case 2:
      return TopicSortType.BY_CUTOFF_TIME_ASC;
    case 3:
      return TopicSortType.BY_VOLUME_DESC;
    case 4:
      return TopicSortType.BY_VOLUME_ASC;
    case 5:
      return TopicSortType.BY_VOLUME_24H_DESC;
    case 6:
      return TopicSortType.BY_VOLUME_24H_ASC;
    case 7:
      return TopicSortType.BY_VOLUME_7D_DESC;
    case 8:
      return TopicSortType.BY_VOLUME_7D_ASC;
    default:
      throw createServiceError(grpcStatus.INVALID_ARGUMENT, `unsupported sort_by: ${value}`);
  }
}

function buildGetMarketsOptions(request: GetMarketsRequest) {
  const page = normalizePositiveInteger('page', request.page);
  const limit = normalizePositiveInteger('limit', request.limit);
  const topicType = toSdkTopicType(request.topic_type);
  const status = toSdkStatusFilter(request.status);
  const sortBy = toSdkSortBy(request.sort_by);

  return {
    topicType,
    ...(page ? { page } : {}),
    ...(limit ? { limit } : {}),
    ...(status ? { status } : {}),
    ...(sortBy ? { sortBy } : {}),
  };
}

function toGrpcError(error: unknown): ServiceError {
  if (error && typeof error === 'object' && 'code' in error && typeof error.code === 'number') {
    return error as ServiceError;
  }

  if (error instanceof InvalidParamError) {
    return createServiceError(grpcStatus.INVALID_ARGUMENT, error.message);
  }

  if (error instanceof Error) {
    return createServiceError(grpcStatus.INTERNAL, error.message);
  }

  return createServiceError(grpcStatus.INTERNAL, 'unknown opinion service error');
}

export function createOpinionService(): OpinionServiceHandlers {
  return {
    async getMarkets(call, callback) {
      try {
        const client = getOpinionClient();
        const response = await client.getMarkets(buildGetMarketsOptions(call.request));

        if (response.errno !== 0) {
          throw createServiceError(
            grpcStatus.INTERNAL,
            response.errmsg || `opinion sdk returned errno ${response.errno}`,
          );
        }

        callback(null, mapGetMarketsResponse(response.result ?? {}));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },
  };
}
