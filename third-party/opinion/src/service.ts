import type { OpinionServiceServer } from './gen/opinion/opinion.js';
import { toGrpcError } from './grpc-error.js';
import type { GetMarketsDependencies } from './get-markets/usecase.js';
import { executeGetMarkets } from './get-markets/usecase.js';

export function createOpinionService(
  dependencies: GetMarketsDependencies = {},
): OpinionServiceServer {
  return {
    async getMarkets(call, callback) {
      try {
        callback(null, await executeGetMarkets(call.request, dependencies));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },
  };
}
