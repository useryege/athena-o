import { CHAIN_ID_BNB_MAINNET, DEFAULT_API_HOST } from '@opinion-labs/opinion-clob-sdk';
import { type ClientConfig } from '@opinion-labs/opinion-clob-sdk';
import { env } from './env.js';

export function buildOpinionConfig(): ClientConfig {
  return {
    host: env.OPINION_HOST || DEFAULT_API_HOST,
    apiKey: env.API_KEY,
    chainId: CHAIN_ID_BNB_MAINNET,
    rpcUrl: env.RPC_URL,
    privateKey: env.PRIVATE_KEY as `0x${string}`,
    multiSigAddress: env.MULTI_SIG_ADDRESS as `0x${string}`,
    marketCacheTtl: env.MARKET_CACHE_TTL,
    quoteTokensCacheTtl: env.QUOTE_TOKENS_CACHE_TTL,
    enableTradingCheckInterval: env.ENABLE_TRADING_CHECK_INTERVAL,
  };
}
