# Reference: Opinion sidecar parity E2E

This repository implements the parity pattern under `third-party/opinion/`.

## Key files

| Role | Path |
|------|------|
| gRPC handler | `third-party/opinion/src/service.ts` — calls `parse*Request`, SDK, `assertSdkSuccess`, `to*Response` |
| Request parsing and response mapping | `third-party/opinion/src/common.ts` — e.g. `parseGetMarketsRequest`, `toGetMarketsResponse` |
| SDK factory | `third-party/opinion/src/client.ts` — `createOpinionClient`, `loadRuntimeConfig` from `config.ts` |
| E2E tests | `third-party/opinion/test/e2e/get-markets.e2e.ts` |
| Shared helpers | `third-party/opinion/test/e2e/helpers.ts` — `getE2EGrpcAddress`, `handleE2EUnaryError`, `isLikelySdkUpstreamRegionBlock`, `assertGetMarketsResponsesEqual` |

## Environment

- Server entry: `third-party/opinion/cmd/server/main.ts` uses `import 'dotenv/config'`.
- E2E tests that call `loadRuntimeConfig()` should also load dotenv first.
- Tests connect with `OPINION_E2E_GRPC_ADDR` (default aligns with local gRPC bind port).

## Parity assertion snippet (conceptual)

1. `expected = toGetMarketsResponse(await sdk.getMarkets(parseGetMarketsRequest(request)))`
2. `actual = await unaryPromise(client.getMarkets(request))`
3. `GetMarketsResponse.encode(expected).finish()` byte-compared to `encode(actual).finish()`

## Flakiness note

`GetMarkets` triggers two upstream list fetches when both SDK and sidecar run; markets may change between calls. Keep `limit` small and accept occasional drift, or document retries outside the default test runner.
