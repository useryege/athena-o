import axios from 'axios';
import { ClobClient } from '@polymarket/clob-client-v2';
import { HttpsProxyAgent } from 'https-proxy-agent';
import { createWalletClient, custom } from 'viem';
import { privateKeyToAccount } from 'viem/accounts';

import { type PolymarketSdkConfig, loadRuntimeConfig } from './config.js';

export function createPolymarketClient(config: PolymarketSdkConfig): ClobClient {
  if (config.proxyUrl) {
    axios.defaults.httpsAgent = new HttpsProxyAgent(config.proxyUrl);
  }

  const signer = config.privateKey
    ? createWalletClient({
        account: privateKeyToAccount(config.privateKey),
        transport: custom({ request: () => Promise.reject(new Error('no rpc')) }),
      })
    : undefined;

  return new ClobClient({
    host: config.host,
    chain: config.chain,
    ...(signer ? { signer } : {}),
    ...(config.creds ? { creds: config.creds } : {}),
  });
}

let polymarketClient: ClobClient | undefined;

export function getPolymarketClient(): ClobClient {
  if (polymarketClient) {
    return polymarketClient;
  }

  polymarketClient = createPolymarketClient(loadRuntimeConfig().sdk);
  return polymarketClient;
}
