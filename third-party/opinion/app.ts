import { Client, CHAIN_ID_BNB_MAINNET, DEFAULT_API_HOST ,TopicStatusFilter } from '@opinion-labs/opinion-clob-sdk';
import { API_KEY, RPC_URL, PRIVATE_KEY, MULTI_SIG_ADDRESS } from './src/config/env';

const client = new Client({
  host: DEFAULT_API_HOST,
  apiKey: API_KEY,
  chainId: CHAIN_ID_BNB_MAINNET, // 56
  rpcUrl: RPC_URL,
  privateKey: PRIVATE_KEY as `0x${string}`,
  multiSigAddress: MULTI_SIG_ADDRESS as `0x${string}`,
});

console.log('Client initialized successfully!');

