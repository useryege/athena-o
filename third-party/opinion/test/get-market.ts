import 'dotenv/config';

import * as grpc from '@grpc/grpc-js';

import { GetMarketRequest, OpinionServiceClient } from '../src/gen/opinion/opinion.js';

async function main() {
  const target = process.env.GRPC_TARGET ?? `127.0.0.1:${process.env.GRPC_PORT ?? '50051'}`;
  const client = new OpinionServiceClient(target, grpc.credentials.createInsecure());

  const marketId = process.env.MARKET_ID ?? '1';
  const request: GetMarketRequest = { marketId };

  const response = await new Promise((resolve, reject) => {
    client.getMarket(request, (error, result) => {
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
  console.error('Failed to fetch market:', error);
  process.exit(1);
});
