import type {
  CancelAllRequest,
  CancelAllResponse,
  GetOkRequest,
  GetOkResponse,
} from './gen/polymarket/polymarket.js';

function sanitizeJsonValue(
  value: unknown,
  visited: WeakSet<object> = new WeakSet<object>(),
): any | undefined {
  if (value === null) {
    return null;
  }
  if (value === undefined) {
    return undefined;
  }

  switch (typeof value) {
    case 'string':
    case 'number':
    case 'boolean':
      return value;
    case 'bigint':
      return value.toString();
    case 'function':
    case 'symbol':
      return undefined;
    case 'object':
      break;
    default:
      return undefined;
  }

  if (value instanceof Date) {
    return value.toISOString();
  }

  if (Array.isArray(value)) {
    return value.map((item) => sanitizeJsonValue(item, visited) ?? null);
  }

  const objectValue = value as Record<string, unknown>;
  if (visited.has(objectValue)) {
    return '[circular]';
  }
  visited.add(objectValue);

  const result: Record<string, any> = {};
  for (const [key, item] of Object.entries(objectValue)) {
    const sanitized = sanitizeJsonValue(item, visited);
    if (sanitized !== undefined) {
      result[key] = sanitized;
    }
  }

  visited.delete(objectValue);
  return result;
}

export function toProtoStructData(value: unknown): { [key: string]: any } | undefined {
  const sanitized = sanitizeJsonValue(value);
  if (sanitized === undefined) {
    return undefined;
  }
  if (sanitized && typeof sanitized === 'object' && !Array.isArray(sanitized)) {
    return sanitized as { [key: string]: any };
  }

  return { value: sanitized };
}

export function parseGetOkRequest(request: GetOkRequest): void {
  void request;
}

export function toGetOkResponse(data: unknown): GetOkResponse {
  return {
    data: toProtoStructData(data),
  };
}

export function parseCancelAllRequest(request: CancelAllRequest): void {
  void request;
}

export function toCancelAllResponse(data: unknown): CancelAllResponse {
  return {
    data: toProtoStructData(data),
  };
}
