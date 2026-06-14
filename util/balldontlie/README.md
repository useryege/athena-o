# BALLDONTLIE Docs

`util/balldontlie` stores local snapshots of BALLDONTLIE documentation and a typed Go client for BALLDONTLIE APIs used by future ATHENA integration work.

This package does not add ATHENA service code, proto definitions, SQL, runtime wiring, or UI.

## Official Sources

- OpenAPI: `https://www.balldontlie.io/openapi/atp.yml`
- Human docs: `https://atp.balldontlie.io/`
- API base server: `https://api.balldontlie.io`
- Authentication: `Authorization` request header

The ATP OpenAPI document has been confirmed to use OpenAPI 3.1.0 and currently contains 12 paths and 12 schemas.

## Sync Docs

```bash
make -C util/balldontlie sync-docs
```

The sync command rebuilds `balldontlie-docs/` on every run and writes:

- `balldontlie-docs/openapi/atp.yml`
- `balldontlie-docs/html/atp-api.html`
- `balldontlie-docs/metadata/atp-openapi.headers.txt`
- `balldontlie-docs/metadata/atp-html.headers.txt`

## Current ATP Coverage

The OpenAPI spec is the machine-readable source of truth for these ATP endpoints:

- Players
- Tournaments
- Rankings
- Matches
- ATP Race
- Match statistics
- Player career statistics
- Head-to-head records
- Betting odds

Free and paid tier differences are documented by BALLDONTLIE, but this directory does not implement entitlement behavior.

## Go Client

The Go wrapper entrypoint is `NewClient(Config{})`. The default upstream API base URL is `DefaultBaseURL`.

```go
client, err := balldontlie.NewClient(balldontlie.Config{
	APIKey: "YOUR_API_KEY",
})
if err != nil {
	return err
}

players, err := client.ListPlayers(ctx, balldontlie.ListPlayersOptions{
	Search: "Alcaraz",
})
if err != nil {
	return err
}
_ = players
```

Head-to-head lookup:

```go
h2h, err := client.GetHeadToHead(ctx, balldontlie.HeadToHeadOptions{
	Player1ID: 1,
	Player2ID: 2,
})
if err != nil {
	return err
}
_ = h2h
```

The client includes all 12 ATP GET endpoints from the local OpenAPI document:

- `ListPlayers`
- `GetPlayer`
- `ListTournaments`
- `GetTournament`
- `ListRankings`
- `ListMatches`
- `GetMatch`
- `ListATPRace`
- `ListMatchStats`
- `ListPlayerCareerStats`
- `GetHeadToHead`
- `ListOdds`

Every request sends the configured API key in the `Authorization` header. Endpoints that require a paid BALLDONTLIE tier are still exposed by the client; insufficient account access is returned as `*APIError` with status code `401`.

By default, the client creates a conservative `5 req/min` limiter for free/trial usage. Paid-tier callers can pass a custom `RateLimiter` in `Config`.
