export const OpinionErrorCode = {
  INIT_FAILED: 'OPINION_INIT_FAILED',
  NOT_INITIALIZED: 'OPINION_NOT_INITIALIZED',
  API_ERROR: 'OPINION_API_ERROR',
  INVALID_PARAMS: 'OPINION_INVALID_PARAMS',
  UNKNOWN: 'OPINION_UNKNOWN',
} as const;

export type OpinionErrorCode = (typeof OpinionErrorCode)[keyof typeof OpinionErrorCode];

export class OpinionError extends Error {
  readonly code: OpinionErrorCode;
  readonly upstream?: unknown;

  constructor(code: OpinionErrorCode, message: string, upstream?: unknown) {
    super(message);
    this.name = 'OpinionError';
    this.code = code;
    this.upstream = upstream;
  }
}

export function mapSdkError(err: unknown): OpinionError {
  if (err instanceof OpinionError) return err;

  const message = err instanceof Error ? err.message : String(err);

  if (message.toLowerCase().includes('not initialized')) {
    return new OpinionError(OpinionErrorCode.NOT_INITIALIZED, message, err);
  }

  return new OpinionError(OpinionErrorCode.API_ERROR, message, err);
}
