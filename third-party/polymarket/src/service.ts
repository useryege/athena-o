import type { PolymarketServiceServer } from './gen/polymarket/polymarket.js';
import { toGrpcError } from './grpc-error.js';

export function createPolymarketService(): PolymarketServiceServer {
  return {
    async health(call, callback) {
      try {
        callback(null, { status: 'ok' });
      } catch (error) {
        callback(toGrpcError(error));
      }
    },
  };
}
