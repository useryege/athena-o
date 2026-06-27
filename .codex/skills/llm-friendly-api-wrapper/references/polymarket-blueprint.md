# Polymarket Blueprint

Use `util/polymarket` as the canonical reference implementation.

## 1. Docs synchronization

Pattern:
- `util/polymarket/sync-docs.sh`
- Source: `https://docs.polymarket.com/llms.txt`
- Target: `util/polymarket/polymarket-docs/`

Key behavior:
- Rebuild docs directory on each run.
- Parse markdown links from llms index.
- Fail when zero links are found.

## 2. Public-read snapshot command

Pattern:
- `util/polymarket/cmd-polymarket-public-read-snapshot/main.go`
- Output: `util/polymarket/request-response/latest/`

Key behavior:
- Parse local `api-reference/**/*.md`.
- Filter to public read endpoints.
- Discover sample IDs automatically.
- Execute all endpoints, continue on failures.
- Emit per-endpoint logs + summary/index.

## 3. Wrapper modules

Pattern modules:
- `GammaClient`
- `DataClient`
- `CLOBClient`
- WS read clients (market/sports)

Shared design:
- Config + interface + private impl + centralized `do()`
- typed options and models
- guarded integration tests

## 4. Test orchestration

Pattern:
- Unit tests always run.
- Integration tests gated by env flags.
- Each public method has dedicated `t.Run`.
- Optional full payload logs with `..._LOG_RESPONSE=1`.

## 5. Drift operations

Pattern:
1. sync docs
2. run snapshot
3. compare latest snapshot
4. adjust models/tests
5. rerun integration

Use this exact loop for every new vendor integration.
