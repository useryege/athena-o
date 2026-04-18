import { Client } from '@opinion-labs/opinion-clob-sdk';

import { type OpinionSdkConfig, loadRuntimeConfig } from './config.js';

let opinionClient: Client | undefined;

export function createOpinionClient(config: OpinionSdkConfig): Client {
  return new Client({
    host: config.host,
    apiKey: config.apiKey,
    chainId: config.chainId,
    rpcUrl: config.rpcUrl,
    privateKey: config.privateKey,
    multiSigAddress: config.multiSigAddress,
  });
}

export function getOpinionClient(): Client {
  if (opinionClient) {
    return opinionClient;
  }

  const { sdk } = loadRuntimeConfig();

  opinionClient = createOpinionClient(sdk);

  return opinionClient;
}
