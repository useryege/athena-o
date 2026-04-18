import 'dotenv/config';

import * as grpc from '@grpc/grpc-js';

import { loadOpinionProto } from '../src/proto.js';
import type { GetMarketsRequest } from '../src/types.js';

async function main() {
  const opinionProto = loadOpinionProto();
  const target = process.env.GRPC_TARGET ?? `127.0.0.1:${process.env.GRPC_PORT ?? '50051'}`;
  const client = new opinionProto.OpinionService(target, grpc.credentials.createInsecure());

  const request: GetMarketsRequest = {
    page: 1,
    limit: Number.parseInt(process.env.MARKETS_LIMIT ?? '1', 10),
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
