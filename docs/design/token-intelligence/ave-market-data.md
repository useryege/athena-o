# Ave Market Data Collection

> 设计状态：已实现
>
> 相关目标需求（讨论中）：[Token 两板块目标设计](../../requirements/token/token.md)

## Scope

The `ave` collector performs one Ave Token Detail request for a validated
project and persists normalized market evidence. It validates the provider's
chain and Token contract, retains only pair data that exactly matches the
project's canonical wrapped-native or USDT pair, and keeps provider risk claims
under the `aveRisk` namespace. It does not schedule refreshes or make Ave data
authoritative for on-chain facts.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Provider adapter | [internal/token/adapters/ave/provider.go](../../../internal/token/adapters/ave/provider.go) | `Provider.GetMarketData`, `normalizeResult` |
| Result contract | [internal/token/collection/payload.go](../../../internal/token/collection/payload.go) | `AveResultV1`, `AveTokenV1`, `AveRiskV1`, `AvePairV1` |
| Collector processor | [internal/token/collection/application/processors.go](../../../internal/token/collection/application/processors.go) | `AveProcessor` |
| Process composition | [cmd/athena-token-collector/commands/athena-token-collector.go](../../../cmd/athena-token-collector/commands/athena-token-collector.go) | Ave branch in `NewCommand` |
| Profile projection | [internal/token/profile/builder.go](../../../internal/token/profile/builder.go) | Ave evidence mapping in `Build` |

## Architecture

The Ave collector claims only `ave` tasks from PostgreSQL. It calls the shared
Ave client once, validates the response, converts optional decimal strings to
lossless decimal values, and returns a V1 payload to the generic fenced
collector commit. ProjectProfile construction later combines this provider
evidence with the independently collected on-chain evidence.

## Runtime Flow

1. The process claims one eligible `ave` task and receives the project's chain,
   Token contract, and two canonical pair addresses.
2. The adapter requests Ave Token Detail with the chain and contract.
3. The response Token address must equal the requested contract and its chain
   name must map to the requested chain ID.
4. Market, supply, holder, timestamp, and risk values are normalized. Invalid
   present decimals, booleans, addresses, or pair chains fail the task attempt.
5. Provider pairs are scanned in response order. At most one exact match for
   each canonical pair is retained; unrelated or malformed-address pair entries
   are ignored.
6. The generic collector hashes and commits the normalized payload. The result
   records no EVM block because it is a provider snapshot.

## State / Data

The V1 payload contains `chainId`, Token identity and market values, zero to two
canonical pair market records, and `aveRisk`. Optional provider values remain
absent rather than becoming zero. Provider timestamps are converted from
positive Unix seconds to UTC.

Risk fields are explicitly provider-owned: audit flag, level, score, text,
optional mintable flag, mint-method claim, LP-unlocked claim, ownership,
audit/source, blocklist, and honeypot claims. The ProjectProfile may present
these values but does not reinterpret them as chain-derived proof.

## Configuration

`ATHENA_TOKEN_AVE_API_KEY` must be nonempty. `ATHENA_TOKEN_AVE_API_BASE_URL`
defaults to the shared Ave client base URL. HTTP timeout is 60 seconds. The
collector polls each second and retries real Ave failures after five minutes;
its queue lease and heartbeat are 90 and 30 seconds.

## Invariants

- The normalized Token address and chain match the requested project.
- Only exact project canonical-pair addresses enter evidence.
- Ave risk claims remain namespaced and never replace on-chain signals.
- A project has one `ave` task and at most one immutable Ave result.
- Ave collection occurs once; ProjectProfile Builder never calls Ave.

## Failure Recovery

Transport, provider, response-shape, identity, or normalization failure consumes
one real collection attempt. The first two attempts return to pending after the
five-minute interval; the third becomes failed and allows an incomplete profile.
No partial Ave result is saved. Lease loss discards the response without
consuming a provider failure.

## Observability

Collector health, readiness, loop metrics, queue counts/age, task failure count,
last error, and timestamps identify Ave work. The task-detail API exposes the
complete normalized evidence and content hash.

## Change Checklist

- [ ] Recheck chain and contract validation against Ave mappings.
- [ ] Recheck lossless optional decimal and boolean parsing.
- [ ] Recheck exact canonical-pair matching and deterministic order.
- [ ] Keep provider risk fields inside `aveRisk`.
- [ ] Keep the [design index](../README.md) current.
