import { Client } from '@opinion-labs/opinion-clob-sdk';
import { status as grpcStatus } from '@grpc/grpc-js';

import { type OpinionSdkConfig, loadRuntimeConfig } from './config.js';
import { createServiceError } from './grpc-error.js';

export function createOpinionClient(config: OpinionSdkConfig): Client {
  return new Client({
    host: config.host,
    apiKey: config.apiKey,
    chainId: config.chainId,
    rpcUrl: config.rpcUrl,
    privateKey: config.privateKey,
    multiSigAddress: config.multiSigAddress,
    ...(config.proxyUrl ? { proxyUrl: config.proxyUrl } : {}),
  });
}

// ─── Singleton ────────────────────────────────────────────────────────────

let opinionClient: Client | undefined;

export function getOpinionClient(): Client {
  if (opinionClient) return opinionClient;
  const { sdk } = loadRuntimeConfig();
  opinionClient = createOpinionClient(sdk);
  return opinionClient;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

export function assertSdkSuccess(errno: number, errmsg: string): void {
  if (errno !== 0) {
    throw createServiceError(
      grpcStatus.INTERNAL,
      errmsg || `opinion sdk returned errno ${errno}`,
    );
  }
}
