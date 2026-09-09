# Token Module Access Control

> 设计状态：已实现
>
> 相关目标需求（讨论中）：[Token 两板块目标设计](../../requirements/token/token.md)

## Scope

The Token module uses the repository-wide account authorization model to guard
public gRPC/HTTP APIs and control visibility of its disabled member menu entry.
`READ` permits every Token inspection endpoint. `READ_WRITE` additionally
permits policy blocklist mutations and chain checkpoint control. Background
Token workers use direct internal dependencies and are not account-authorized.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Module permissions | [internal/accountaccess](../../../internal/accountaccess) | Token module READ/READ_WRITE evaluation |
| gRPC authorization map | [internal/server/authz.go](../../../internal/server/authz.go) | Token method permissions |
| Public Token facade | [internal/server/tokenapi/tokenapi.go](../../../internal/server/tokenapi/tokenapi.go) | Catalog, collection, policy, and operations methods |
| Service registration | [internal/server/athena-server.go](../../../internal/server/athena-server.go) | Token gRPC and gateway registration |
| Member navigation | [ui/src/app/member/app.tsx](../../../ui/src/app/member/app.tsx) | `tokenNavItem`, `canAccessItem`, `toMenuItems` |
| Full permissions and display projection | [ui/src/app/shared/access-modules.ts](../../../ui/src/app/shared/access-modules.ts), [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts) | `accountDataModules`, `accountAccessDisplayModules`, complete-matrix updates and display summaries |

## Architecture

Authentication establishes an account credential and realm. Account access
resolves the current Token permission. Server interceptors apply the explicit
method-to-permission table before the public Token facade proxies to the
internal Token API. The member application uses the same resolved module access
to show a disabled Token item under Token & Risk. It has no Token business
routes or browser service. Server enforcement remains authoritative.

## Runtime Flow

1. A member session or API Key is authenticated and bound to its current
   account access record.
2. The member shell displays the disabled Token item only with Token `READ` or
   `READ_WRITE`. It has no path or children. `/token` and `/token/*` use the
   existing not-found page; there is no Token landing path or automatic jump.
3. Every public Token RPC/HTTP route is matched in `authz.go` and checked before
   its handler runs.
4. `READ` covers runtime configuration/status, chain attempt reads, projects,
   the unique profile, collection tasks/evidence, contract source, wallets,
   and pre-deployment transactions.
5. `READ_WRITE` covers contract-code and wallet blocklist creation/update/delete
   plus checkpoint start/stop controls. Corresponding read endpoints remain
   `READ`.
6. Account access refresh preserves the complete Token grant and updates the
   disabled item's visibility. Token has no browser requests, project return
   snapshots, or feature cache; shared request/session cleanup remains in place
   for the other modules.

## State / Data

Authorization state lives in the shared account-access domain, not the Token
database. Token project, evidence, profile, and operational rows are not
account-owned; module permission controls access to this shared intelligence.
Collection and profile endpoints are read-only and expose no retry, rebuild, or
reschedule mutation.

Administrator permission controls, Account Center cards, and display summaries
use `accountAccessDisplayModules`, which excludes Token. Authorization parsing,
draft cloning, replacement, equality, status derivation, and full-matrix
submission continue to use all nine `accountDataModules`. Editing another
module leaves Token's existing level intact. A Token-only grant still makes
an otherwise eligible ordinary account Active even though it has no Token page.

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
- The Token menu is disabled and permission-gated; no Token business page or
  browser API client is registered.
- Hiding Token in permission controls never deletes, drops, or zeroes its grant.
- Backend authorization remains authoritative for every public Token request.
- Worker database/provider access does not reuse browser credentials.

## Failure Recovery

Missing authentication, missing module permission, or insufficient permission
fails before Token business logic. A revoked grant takes effect on the next
authorized request and updates disabled-menu visibility when the account state
refreshes. Unknown Token methods are not implicitly permitted;
new RPCs must be added deliberately to the authorization map.

## Observability

Standard request tracing and authorization logs identify the RPC, account
context, required permission, and resulting status without changing Token
state. The administrator's shared Service Status still exposes Token API
health, and Etherscan Gateway administration remains available independently.

## Change Checklist

- [ ] Add every new Token RPC to the explicit authorization map.
- [ ] Keep read-only and mutation permission levels aligned with the handler.
- [ ] Recheck service and gateway registration after proto changes.
- [ ] Keep the disabled menu permission-gated and Token absent from business routes.
- [ ] Preserve Token in full permission updates while omitting it from permission UI.
- [ ] Keep the [design index](../README.md) current.
