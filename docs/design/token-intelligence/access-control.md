# Token Module Access Control

## Scope

The Token module uses the repository-wide account authorization model to guard
member navigation, browser requests, public gRPC/HTTP APIs, and client caches.
`READ` permits every Token inspection endpoint. `READ_WRITE` additionally
permits policy blocklist mutations and chain checkpoint control. Background
Token workers use direct internal dependencies and are not account-authorized.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Module permissions | [internal/server/accountaccess](../../../internal/server/accountaccess) | Token module READ/READ_WRITE evaluation |
| gRPC authorization map | [internal/server/authz.go](../../../internal/server/authz.go) | Token method permissions |
| Public Token facade | [internal/server/tokenapi/tokenapi.go](../../../internal/server/tokenapi/tokenapi.go) | Catalog, collection, policy, and operations methods |
| Service registration | [internal/server/athena-server.go](../../../internal/server/athena-server.go) | Token gRPC and gateway registration |
| Member route guard | [ui/src/app/member/routes.tsx](../../../ui/src/app/member/routes.tsx) | Token-authorized route subtree |
| Browser client | [ui/src/app/shared/services/token-service.ts](../../../ui/src/app/shared/services/token-service.ts) | authenticated Token requests and cache keys |

## Architecture

Authentication establishes an account credential and realm. Account access
resolves the current Token permission. Server interceptors apply the explicit
method-to-permission table before the public Token facade proxies to the
internal Token API. The member application uses the same resolved module access
to construct navigation and guard routes; server enforcement remains
authoritative.

## Runtime Flow

1. A member session or API Key is authenticated and bound to its current
   account access record.
2. The member shell displays Token navigation only with Token `READ` or
   `READ_WRITE`; direct navigation without permission is rejected.
3. Every public Token RPC/HTTP route is matched in `authz.go` and checked before
   its handler runs.
4. `READ` covers runtime configuration/status, chain attempt reads, projects,
   the unique profile, collection tasks/evidence, contract source, wallets,
   and pre-deployment transactions.
5. `READ_WRITE` covers contract-code and wallet blocklist creation/update/delete
   plus checkpoint start/stop controls. Corresponding read endpoints remain
   `READ`.
6. A permission or identity revision invalidates module-aware frontend caches,
   preventing data retained under an older grant from being reused.

## State / Data

Authorization state lives in the shared account-access domain, not the Token
database. Token project, evidence, profile, and operational rows are not
account-owned; module permission controls access to this shared intelligence.
Collection and profile endpoints are read-only and expose no retry, rebuild, or
reschedule mutation.

## Configuration

There is no Token-specific authorization secret or role list. The public server
uses its normal authentication/session/API-Key configuration and the persisted
module access model. Disabled-auth development mode still produces a typed
identity and applies its configured module permissions.

## Invariants

- Every public Token method has an explicit authorization-map entry.
- Token `READ` is sufficient for all catalog, collection, profile,
  contract-source, and operational read endpoints.
- Only `READ_WRITE` may mutate blocklists or chain checkpoint status.
- UI visibility is advisory; backend authorization is authoritative.
- Token responses and caches never cross account or access-revision boundaries.
- Worker database/provider access does not reuse browser credentials.

## Failure Recovery

Missing authentication, missing module permission, or insufficient permission
fails before Token business logic. A revoked grant takes effect on the next
authorized request and triggers frontend route/cache reconciliation when the
account state refreshes. Unknown Token methods are not implicitly permitted;
new RPCs must be added deliberately to the authorization map.

## Observability

Standard request tracing and authorization logs identify the RPC, account
context, required permission, and resulting status without changing Token
state. Frontend access failures are surfaced through the shared authenticated
request and route-boundary behavior.

## Change Checklist

- [ ] Add every new Token RPC to the explicit authorization map.
- [ ] Keep read-only and mutation permission levels aligned with the handler.
- [ ] Recheck service and gateway registration after proto changes.
- [ ] Recheck member route visibility and module-aware cache invalidation.
- [ ] Keep the [design index](../README.md) current.
