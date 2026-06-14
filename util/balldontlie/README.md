# BALLDONTLIE Docs

`util/balldontlie` stores local snapshots of BALLDONTLIE documentation used as source material for future ATHENA integration work.

This stage only syncs documentation. It does not add a Go client, ATHENA service code, proto definitions, SQL, runtime wiring, or UI.

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
