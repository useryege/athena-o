# Test Gates And Commands

## Gate naming

Use deterministic gate naming:
- Main gate: `<VENDOR>_<MODULE>_INTEGRATION=1`
- Optional subgroup: `<VENDOR>_<MODULE>_INTEGRATION_<GROUP>=1`
- Full payload logs: `<VENDOR>_<MODULE>_INTEGRATION_LOG_RESPONSE=1`

Example:
- `POLYMARKET_GAMMA_INTEGRATION=1`
- `POLYMARKET_GAMMA_INTEGRATION_MARKETS=1`
- `POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1`

## Command style

Use no-cache mode for live tests:
- `go test -count=1 -v ./util/<vendor> -run '^TestIntegrationXxx$'`

Keep module isolation first, then run all.

## Test design checklist

- Map every public client method to a dedicated `t.Run` subtest.
- Keep integration tests read-only unless explicitly scoped otherwise.
- Discover sample IDs once and reuse across subtests.
- Allow empty-data scenarios only when API semantics allow empty datasets.
- Fail on request errors; skip only for documented temporary constraints.

## Logging checklist

When log gate enabled:
- Marshal full response as indented JSON.
- Include endpoint/subtest name in log prefix.
- Fall back to `%+v` if JSON marshal fails.

Keep logging off by default.
