import 'dotenv/config';
import { Client, CHAIN_ID_BNB_MAINNET, DEFAULT_API_HOST } from '@opinion-labs/opinion-clob-sdk';

const client = new Client({
  host: DEFAULT_API_HOST,
  apiKey: process.env.API_KEY!,
  chainId: CHAIN_ID_BNB_MAINNET, // 56
  rpcUrl: process.env.RPC_URL!,
  privateKey: process.env.PRIVATE_KEY! as `0x${string}`,
  multiSigAddress: process.env.MULTI_SIG_ADDRESS! as `0x${string}`,
});

client.getMarkets

console.log('Client initialized successfully!');