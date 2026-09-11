# API Docs

You can find the Swagger docs by setting the path to `/swagger-ui` in your Athena UI. E.g. [http://localhost:8080/swagger-ui](http://localhost:8080/swagger-ui) or [http://localhost:4000/swagger-ui](http://localhost:4000/swagger-ui).

On a deployed Athena server, machine-oriented discovery starts at
[`/llms.txt`](/llms.txt). The curated API overview is available at
[`/docs/ai/overview.md`](/docs/ai/overview.md), and the complete machine-readable
Swagger 2.0 contract is available at [`/swagger.json`](/swagger.json).

## Public Version Endpoint

The server version endpoint is public and does not require a browser session or
an API Key:

```bash
curl "$ATHENA_SERVER/api/version"
```

```json
{"Version":"...","BuildDate":"...","GitCommit":"...","GitTreeState":"...","GoVersion":"...","Compiler":"...","Platform":"..."}
```

## Authorization

Browser users authenticate either through Google at `/auth/google/login` or a desktop
Phantom Solana provider through `/auth/phantom/challenge` and
`/auth/phantom/verify`. Athena verifies OIDC or the server-generated SIWS message, then
stores its own session token in an HttpOnly cookie. Google tokens and wallet signatures
are never used for business API requests.

For an unknown Google subject or Solana address, provider verification creates a short-lived
shared registration ticket and directs the browser to `/register`; it does not create an
account or issue the session cookie. The registration API is
`GET|POST|DELETE /auth/registration` plus
`GET /auth/registration/username-availability`. The user must first choose a permanent
username. Successful registration creates the UUID `account_id` used by JWT subjects and
authorization, while API and UI display the username separately. Google and Phantom
identities always create separate accounts.

Phantom login accepts only the injected desktop extension in the Web UI. Challenge creation
accepts `{address, returnTo}` and returns `{message, expiresAt}`; verification accepts only a
raw-base64url `{signature}` and returns `{redirectTo}`. The server performs Ed25519
verification without Solana RPC and never asks for a transaction, private key, or fee.

CLI, automation, and AI clients use an ordinary account's Athena API Key from
**Account Center → Security** after an administrator enables the account's
independent API Key access. The fixed administrator account cannot issue API
Keys. Copy the key when it is issued, then export it locally or give it directly
to the AI selected by the account holder:

```bash
export ATHENA_TOKEN='<newly-issued-athena-api-key>'
```

Pass it using the HTTP `Authorization` header, prefixing with `Bearer `:

```bash
curl "$ATHENA_SERVER/api/v1/market-radar/status" \
  -H "Authorization: Bearer $ATHENA_TOKEN"
```

```json
{"started":true,"status":"running"}
```

This request also requires `READ` access to the Market Radar module. An API Key
has no separate scope, read-only mode, operation allowlist, or approval gate. It
can perform every HTTP API operation currently authorized for the ordinary
account, but it does not bypass Athena's account, module, entitlement,
membership, ownership, or operation-specific authorization.

Only Athena token version 3 is accepted. Rotating `ATHENA_JWT_SECRET` or deleting the API
Key invalidates it. Disabling account login blocks it while login remains disabled.
Turning off API Key access pauses existing keys without deleting them; re-enabling access
restores undeleted, unexpired keys.

## Trader Sync

The generated Trader Sync contract is in
[`tradersync.proto`](../../internal/server/tradersync/tradersync.proto); its 13
member methods use `/api/v1/trader-sync`, and its three administrator summary
methods use `/api/v1/admin/trader-sync`. Member login and enabled API Key callers
need current Trader Sync access. Administrator summaries never expose member
notes, activity metadata, frozen payloads, or individual deliveries.

IDs, raw amounts, position IDs, and JSON revisions remain strings. Use nested
GET pagination keys `page.page_size` and `page.cursor`; page size defaults to 50
and is limited to 1–100. Opaque signed cursors are scoped to the principal,
filters, list kind, and page size. Activity `refreshCursor` refreshes the existing
page; a new initial request obtains the newest snapshot. Missing optional times
and evidence values remain absent, and a sent delivery may legitimately have no
recorded `startedAt`. `CreateSubscription.note` is an optional wrapper: omit it
to retain an existing note; `{ "value": "" }` explicitly clears it.
