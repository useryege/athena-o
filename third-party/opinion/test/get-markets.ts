import 'dotenv/config';

import * as grpc from '@grpc/grpc-js';

import { OpinionServiceClient, GetMarketsRequest } from '../src/gen/opinion/opinion.js';
import { MarketTopicType } from '../src/gen/opinion/opinion.js';

async function main() {
  const target = process.env.GRPC_TARGET ?? `127.0.0.1:${process.env.GRPC_PORT ?? '50051'}`;
  const client = new OpinionServiceClient(target, grpc.credentials.createInsecure());

  const request: GetMarketsRequest = {
    topicType: MarketTopicType.MARKET_TOPIC_TYPE_ALL,
    page: 1,
    limit: Number.parseInt(process.env.MARKETS_LIMIT ?? '1', 10),
    status: 0,
    sortBy: 0,
  };

  const response = await new Promise((resolve, reject) => {
    client.getMarkets(request, (error, result) => {
      if (error) {
        reject(error);
        return;
      }

      resolve(result);
    });
  });

  console.log(JSON.stringify(response, null, 2));
  client.close();
}

void main().catch((error) => {
  console.error('Failed to fetch markets:', error);
  process.exit(1);
});
