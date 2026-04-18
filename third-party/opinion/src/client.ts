import { Client } from '@opinion-labs/opinion-clob-sdk';

import { loadRuntimeConfig } from './config.js';

let opinionClient: Client | undefined;

export function getOpinionClient(): Client {
  if (opinionClient) {
    return opinionClient;
  }

  const { sdk } = loadRuntimeConfig();

  opinionClient = new Client({
    host: sdk.host,
    apiKey: sdk.apiKey,
    chainId: sdk.chainId,
    rpcUrl: sdk.rpcUrl,
    privateKey: sdk.privateKey,
    multiSigAddress: sdk.multiSigAddress,
  });

  return opinionClient;
}
