import { Client, CHAIN_ID_BNB_MAINNET, DEFAULT_API_HOST ,TopicStatusFilter } from '@opinion-labs/opinion-clob-sdk';
import { API_KEY, RPC_URL, PRIVATE_KEY, MULTI_SIG_ADDRESS } from './config.js';

const client = new Client({
  host: DEFAULT_API_HOST,
  apiKey: API_KEY,
  chainId: CHAIN_ID_BNB_MAINNET, // 56
  rpcUrl: RPC_URL,
  privateKey: PRIVATE_KEY as `0x${string}`,
  multiSigAddress: MULTI_SIG_ADDRESS as `0x${string}`,
});

console.log('Client initialized successfully!');


// Get all active markets
const marketsResponse = await client.getMarkets({
  status: TopicStatusFilter.ACTIVATED,
  page: 1,
  limit: 10,
});

// Parse the response
if (marketsResponse.errno === 0) {
  const markets = marketsResponse.result.list;
  console.log(`\nFound ${markets.length} active markets:`);

  for (const market of markets.slice(0, 3)) {
    console.log(`  - Market #${market.marketId}: ${market.marketTitle}`);
    console.log(`    Status: ${market.status}`);
  }
} else {
  console.error(`Error: ${marketsResponse.errmsg}`);
}