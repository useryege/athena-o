import {
  TopicType,
  TopicStatusFilter,
  TopicSortType,
} from '../../infra/opinion/types.js';
import { OpinionError, OpinionErrorCode, mapSdkError } from '../../infra/opinion/errors.js';
import { fetchMarkets, type GetMarketsOptions } from './market.repo.js';
import type { GetMarketsQuery, GetMarketsResponse } from './market.dto.js';


const MAX_LIMIT = 20;

function parseTopicType(raw?: string): TopicType {
  if (raw === undefined) return TopicType.ALL;
  const n = parseInt(raw, 10);
  if (n === TopicType.BINARY) return TopicType.BINARY;
  if (n === TopicType.CATEGORICAL) return TopicType.CATEGORICAL;
  if (n === TopicType.ALL) return TopicType.ALL;
  throw new OpinionError(
    OpinionErrorCode.INVALID_PARAMS,
    `Invalid topicType: ${raw}. Expected 0 (BINARY), 1 (CATEGORICAL), or 2 (ALL).`,
  );
}

function parseStatus(raw?: string): TopicStatusFilter {
  if (raw === undefined || raw === '') return TopicStatusFilter.ALL;
  if (raw === TopicStatusFilter.ACTIVATED) return TopicStatusFilter.ACTIVATED;
  if (raw === TopicStatusFilter.RESOLVED) return TopicStatusFilter.RESOLVED;
  if (raw === TopicStatusFilter.ALL) return TopicStatusFilter.ALL;
  throw new OpinionError(
    OpinionErrorCode.INVALID_PARAMS,
    `Invalid status: "${raw}". Expected "", "activated", or "resolved".`,
  );
}

function parseSortBy(raw?: string): TopicSortType {
  if (raw === undefined) return TopicSortType.BY_TIME_DESC;
  const n = parseInt(raw, 10);
  if (Object.values(TopicSortType).includes(n as TopicSortType)) {
    return n as TopicSortType;
  }
  throw new OpinionError(
    OpinionErrorCode.INVALID_PARAMS,
    `Invalid sortBy: ${raw}. Expected a number 1–8.`,
  );
}

function parsePage(raw?: string): number {
  if (raw === undefined) return 1;
  const n = parseInt(raw, 10);
  if (isNaN(n) || n < 1) {
    throw new OpinionError(OpinionErrorCode.INVALID_PARAMS, `Invalid page: ${raw}`);
  }
  return n;
}

function parseLimit(raw?: string): number {
  if (raw === undefined) return MAX_LIMIT;
  const n = parseInt(raw, 10);
  if (isNaN(n) || n < 1 || n > MAX_LIMIT) {
    throw new OpinionError(
      OpinionErrorCode.INVALID_PARAMS,
      `Invalid limit: ${raw}. Must be 1–${MAX_LIMIT}.`,
    );
  }
  return n;
}

export async function getMarkets(query: GetMarketsQuery): Promise<GetMarketsResponse> {
  const options: GetMarketsOptions = {
    topicType: parseTopicType(query.topicType),
    page: parsePage(query.page),
    limit: parseLimit(query.limit),
    status: parseStatus(query.status),
    sortBy: parseSortBy(query.sortBy),
  };

  let response;
  try {
    response = await fetchMarkets(options);
  } catch (e) {
    throw mapSdkError(e);
  }

  if (response.errno !== 0) {
    throw new OpinionError(
      OpinionErrorCode.API_ERROR,
      `Opinion API error ${response.errno}: ${response.errmsg}`,
    );
  }

  return {
    total: response.result.total,
    list: response.result.list,
  };
}
