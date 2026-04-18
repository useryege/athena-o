import type { OpinionServiceServer } from './gen/opinion/opinion.js';
import { toGrpcError } from './grpc-error.js';
import type { GetMarketDependencies } from './get-market/usecase.js';
import { executeGetMarket } from './get-market/usecase.js';
import type { GetMarketsDependencies } from './get-markets/usecase.js';
import { executeGetMarkets } from './get-markets/usecase.js';

export interface OpinionServiceDependencies {
  getMarkets?: GetMarketsDependencies;
  getMarket?: GetMarketDependencies;
}

export function createOpinionService(
  dependencies: OpinionServiceDependencies = {},
): OpinionServiceServer {
  return {
    async getMarkets(call, callback) {
      try {
        callback(null, await executeGetMarkets(call.request, dependencies.getMarkets ?? {}));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },
    async getMarket(call, callback) {
      try {
        callback(null, await executeGetMarket(call.request, dependencies.getMarket ?? {}));
      } catch (error) {
        callback(toGrpcError(error));
      }
    },
  };
}
