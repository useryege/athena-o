import type * as grpc from '@grpc/grpc-js';

export interface GetMarketsRequest {
  topic_type?: number;
  page?: number;
  limit?: number;
  status?: number;
  sort_by?: number;
}

export interface MarketMessage {
  market_id: string;
  market_title: string;
  slug: string;
  condition_id: string;
  chain_id: string;
  quote_token: string;
  status: number;
  status_enum: string;
  created_at: string;
  cutoff_at: string;
  resolved_at: string;
  yes_token_id: string;
  no_token_id: string;
  result_token_id: string;
  yes_label: string;
  no_label: string;
  volume: string;
  is_incentivized: boolean;
  child_markets: MarketMessage[];
}

export interface GetMarketsResponse {
  total: number;
  markets: MarketMessage[];
}

export interface OpinionServiceHandlers extends grpc.UntypedServiceImplementation {
  getMarkets: grpc.handleUnaryCall<GetMarketsRequest, GetMarketsResponse>;
}

export interface OpinionServiceClient extends grpc.Client {
  getMarkets(
    request: GetMarketsRequest,
    callback: (error: grpc.ServiceError | null, response: GetMarketsResponse) => void,
  ): void;
}

export interface OpinionServiceClientConstructor extends grpc.ServiceClientConstructor {
  service: grpc.ServiceDefinition<grpc.UntypedServiceImplementation>;
  new (
    address: string,
    credentials: grpc.ChannelCredentials,
    options?: Partial<grpc.ChannelOptions>,
  ): OpinionServiceClient;
}
