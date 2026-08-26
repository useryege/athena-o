# Solana Wallet Authentication

## Scope

Solana Wallet Authentication lets an anonymous browser prove control of one
Solana public key through the injected Phantom extension and then enter Athena's
UUID account lifecycle. It owns the server-generated Sign-In With Solana
(SIWS) message, one-time Redis challenge, Ed25519 verification, and the handoff
to shared username registration or Athena session issuance.

This capability never handles a private key, mnemonic, transaction, balance,
or RPC request. The custodial [Wallet Ownership](wallet-ownership.md) subsystem
remains separate. Google OIDC is an independent login provider, and identities
from the two providers create separate Athena accounts.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| HTTP flow and SIWS verification | [internal/phantomauth/handler.go](../../../internal/phantomauth/handler.go) | `Handler`, `NewHandler`, `Challenge`, `Verify`, `siwsMessage`, `validatePublicOrigin` |
| One-time challenge and rate state | [internal/phantomauth/store.go](../../../internal/phantomauth/store.go) | `challengeStore`, `Create`, `Consume`, `challengeTTL`, `challengeGlobalRateLimit`, `challengeClientRateLimit` |
| Shared username registration | [internal/authregistration/handler.go](../../../internal/authregistration/handler.go), [internal/authregistration/store.go](../../../internal/authregistration/store.go), [internal/authregistration/types.go](../../../internal/authregistration/types.go) | `Handler.Begin`, `Registration`, `UsernameAvailability`, `Store`, `Identity` |
| Durable identity and session | [internal/accountcredentials/manager.go](../../../internal/accountcredentials/manager.go), [util/session/sessionmanager.go](../../../util/session/sessionmanager.go), [internal/server/external_auth_backend.go](../../../internal/server/external_auth_backend.go) | `GetByIdentity`, `RegisterExternalAccount`, `IssueLoginSession`, `CreateExternalLogin`, `externalAuthBackend` |
| Public route wiring | [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `newHTTPServer`, `/auth/phantom/*`, `/auth/registration*` |
| Browser integration | [ui/src/app/pages/login.tsx](../../../ui/src/app/pages/login.tsx), [ui/src/app/pages/register.tsx](../../../ui/src/app/pages/register.tsx) | `LoginPage`, `PhantomProvider`, `RegisterPage` |

## Architecture

```mermaid
flowchart LR
    B["Anonymous browser"] --> P["Injected Phantom provider"]
    B --> H["Athena Phantom HTTP handler"]
    H --> R["Redis one-time challenge"]
    P -->|"SIWS message signature"| B
    H --> C["Credential and session managers"]
    C --> D["Account-state PostgreSQL"]
```

The browser connection reveals a public key but does not authenticate it.
Athena creates the exact SIWS message server-side and accepts the wallet only
after verifying its Ed25519 signature. The server records the provider as
`solana_wallet`: a signature proves control of a Solana key, not the brand of
software that produced it. The current UI deliberately exposes only Phantom's
injected desktop provider.

Every verified address is an independent external identity. It cannot be
merged with a Google account, bound as a second credential, or used to recover
another account. Athena account UUID remains the only key for authorization,
API Keys, profiles, Wallet ownership, and Profit Sharing relationships.

## Runtime Flow

1. The login page detects `window.phantom.solana.isPhantom`, calls `connect()`,
   and obtains a candidate base58 public key. A missing extension or rejected
   request remains a browser-local login error.
2. `POST /auth/phantom/challenge` accepts JSON `{address, returnTo}` and validates
   the exact request origin, canonical public key, content type, body size, and
   safe return target. It generates a 16-byte random lowercase-hex nonce and a
   five-minute SIWS message whose domain and URI come from the configured Google
   redirect origin, never request host headers. The message includes the address,
   `Version: 1`, `Chain ID: solana:mainnet`, nonce, issue time, and expiration.
   Its signed statement explicitly says that the proof is only for Athena login
   and triggers no blockchain transaction or network fee.
   The challenge-store Lua script atomically limits creation to 20 per hashed
   client identity and 120 deployment-wide in each one-minute window.
3. Redis stores the address, exact message bytes, nonce, safe return path, issue
   time, and expiry behind an opaque identifier. The identifier is written only
   to a Secure-in-production, HttpOnly, SameSite=Strict challenge cookie.
4. Phantom signs the UTF-8 message without creating a blockchain transaction.
   The browser aborts if the connected account changes before verification and
   submits only the raw-base64url signature.
5. `POST /auth/phantom/verify` first requires the exact Origin and a valid
   challenge cookie, then atomically consumes the challenge. After that point,
   invalid JSON, freshness, message binding, or signature cannot be retried. It
   reconstructs the exact message from the saved address, nonce, origin, and
   times and compares it with the stored text. It then requires canonical raw-
   base64url for exactly 64 signature bytes, decodes exactly 32 public-key bytes,
   and verifies the saved UTF-8 message with Ed25519.
6. A known identity passes current login-access checks and receives an Athena
   JWT cookie plus the saved return path. An unknown address receives only a
   shared 15-minute registration ticket and is sent to `/register`.
7. Username submission creates the complete Pending account aggregate. Only
   after PostgreSQL commit and runtime publication does Athena consume the
   registration ticket and issue a login cookie.

## State / Data

The five-minute challenge is transient Redis state. It contains the canonical
address, exact message, validated return path, nonce, issue time, and expiry.
The opaque identifier is hashed into the Redis key and sent only as the
`athena.phantom.challenge` HttpOnly cookie scoped to `/auth/phantom`. It is
single-use even when verification fails; the response contains only message and
expiry.

Challenge rate counters share Redis but are separate from challenge state. The
client key is a SHA-256 digest of the canonical client network address; neither
the raw client address nor wallet identity becomes a Redis rate-key suffix or
metric label.

The shared 15-minute registration ticket contains provider, identity subject,
verified email, administrator-candidate flag, return path, CSRF secret, and
creation time. Solana tickets always have an empty email and a false
administrator candidate. The registration handler derives `solanaAddress` from
the server-held subject for the anonymous setup response. PostgreSQL stores the
canonical address as the `solana_wallet` identity subject; Account and Session
APIs expose it through the provider-specific `solanaAddress` projection only to
the account owner or administrators.

## Configuration

Phantom desktop-extension login adds no App ID, client secret, callback, RPC,
or per-wallet environment variable. Authentication-enabled startup already
requires an exact `ATHENA_GOOGLE_OIDC_REDIRECT_URI`; its scheme and authority
provide the trusted SIWS URI and domain. Local injection works at the existing
`http://localhost:4000` origin, while production uses HTTPS and Secure cookies.

## Invariants

- A Phantom connection alone is never authentication; a fresh verified
  signature is required.
- Every challenge and registration ticket is browser-bound, short-lived, and
  one-time. After Origin and Cookie binding succeeds, every verification attempt
  consumes its challenge. The public address is not accepted again during
  verification.
- Google and Solana identities never resolve to or create the same account.
- A Solana identity can never create or become the administrator.
- The address is permanent and cannot be replaced, recovered, or transferred.
- Wallet signatures and opaque challenge identifiers never enter an Athena JWT
  or public response. The SIWS message and expiry are the only public challenge
  projection.
- Solana login never changes Wallet records or grants any business entitlement.

## Failure Recovery

Missing or rejected Phantom prompts leave server state unchanged except for a
challenge that expires naturally. Redis failure or challenge-rate exhaustion
prevents challenge creation and returns `phantom_unavailable`; Redis failure
also prevents consumption. Invalid origin, cookie, time, address, replay, or
signature fails closed and never issues a cookie. Database failure after a
valid signature can create neither a partial registration nor a session.

If account creation commits but the registration ticket or cookie step fails,
the address is already durable; repeating wallet authentication follows the
known-identity path. Disconnecting Phantom after login does not revoke the
Athena session. A lost wallet has no recovery path; an administrator may only
disable the old Athena account.

## Observability

Existing login counters record success or failure. Structured logs include the
provider and bounded failure stage, plus account UUID only after resolution.
They exclude signatures, SIWS messages, challenge identifiers, identity
subjects, Athena JWTs, and high-cardinality metric labels. Phantom and Solana
RPC are absent from health checks because the server calls neither service.

## Change Checklist

- [ ] SIWS origin, exact-message verification, and one-time consumption remain current.
- [ ] Solana address canonicalization and Ed25519 lengths remain enforced.
- [ ] Unknown identities still use shared registration without partial accounts.
- [ ] Google, administrator, Wallet, and business-entitlement boundaries remain isolated.
- [ ] Browser support and configuration accurately describe desktop injected Phantom only.
- [ ] The [design index](../README.md) contains the current entry.
