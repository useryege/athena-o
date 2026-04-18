import { TopicSortType, TopicStatusFilter, TopicType } from '@opinion-labs/opinion-clob-sdk';
import { status as grpcStatus } from '@grpc/grpc-js';

import type { GetMarketsRequest } from '../gen/opinion/opinion.js';
import { createServiceError } from '../grpc-error.js';

export interface GetMarketsOptions {
  topicType: TopicType;
  page?: number;
  limit?: number;
  status?: TopicStatusFilter;
  sortBy?: TopicSortType;
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

export function buildGetMarketsOptions(request: GetMarketsRequest): GetMarketsOptions {
  const page = normalizePositiveInteger('page', request.page);
  const limit = normalizePositiveInteger('limit', request.limit);
  const topicType = toSdkTopicType(request.topicType);
  const status = toSdkStatusFilter(request.status);
  const sortBy = toSdkSortBy(request.sortBy);

  return {
    topicType,
    ...(page ? { page } : {}),
    ...(limit ? { limit } : {}),
    ...(status ? { status } : {}),
    ...(sortBy ? { sortBy } : {}),
  };
}
