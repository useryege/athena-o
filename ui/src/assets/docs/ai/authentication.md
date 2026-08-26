# Athena API Authentication

Athena accepts its own browser login session or an Athena API Key. Google credentials, Phantom signatures, Solana addresses, and wallet private keys are never business-API bearer credentials.

## Browser Sessions

The Web UI authenticates through Google or the injected Phantom desktop provider. After provider verification and, for a new identity, username registration, Athena issues its own login session in an HttpOnly cookie. Browser clients should let the browser send that cookie; scripts should not attempt to extract it.

## API Keys

API Keys are account-level bearer credentials for command-line clients, automation, and AI systems selected by the account holder. An ordinary user may give the one-time-displayed secret to a local, self-hosted, or third-party AI system and allow that system to act as the user through Athena's HTTP API.

This is direct use of the user's account credential. It is not OAuth delegation, an agent-specific credential, or a separately permissioned integration.

An administrator must first enable an ordinary account's independent API Key entitlement. The account holder can then open **Account Center > Security**, choose **Create API key**, provide a display ID and expiration, and copy the secret from the one-time display. The Web UI offers 30 days, 90 days, one year, or no expiration; the API accepts a non-negative `expiresIn` duration in seconds, where zero means no expiration. Athena's fixed administrator account does not support API Keys.

Send the secret in the HTTP `Authorization` header:

```http
Authorization: Bearer <athena-api-key>
```

For example:

```bash
curl "${ATHENA_SERVER}/api/v1/market-radar/status" \
  -H "Authorization: Bearer ${ATHENA_TOKEN}"
```

An API Key has no key-specific scope, read-only mode, operation allowlist, or per-request approval gate. Within Athena's HTTP API, possession of the bearer lets the caller perform every operation that the owning account is currently authorized to perform. This can include reads, writes, account self-service, API Key management, wallet-secret operations, notification sends, and Profit Sharing member operations when the account's current authorization permits them.

Full-account access does not bypass Athena authorization. Every API Key request rechecks the owning account's current `LoginEnabled` and `APIKeyEnabled` state. Module levels, Profit Sharing entitlement and membership, resource ownership, and operation-specific rules are then evaluated when applicable to the requested operation. The key is not a permission snapshot and immediately gains or loses effective access when the account's authorization changes. Administrator-only operations remain unavailable because the fixed administrator account cannot issue an API Key. Review [Modules and Permissions](/docs/ai/modules.md) and [Full-Account AI Access](/docs/ai/safety.md) for the complete authority model.

## Expiration and Revocation

- An expiring key stops working after its configured expiration time. A no-expiration key remains valid until another invalidation condition occurs.
- Revoking a key from **Account Center > Security** removes it immediately.
- Disabling the account's API Key entitlement pauses every retained key. Re-enabling it restores keys that have not been revoked or expired.
- Disabling account login blocks both browser sessions and API Keys.
- Rotating Athena's JWT signing secret invalidates existing browser sessions and API Keys.

A bearer that can manage API Keys may create another key. Revoking one key does not revoke other keys retained by the account; disable the account's API Key entitlement when every retained key must be paused.

The API lists only key metadata after creation. Athena does not display the bearer secret again.

## Authentication and Authorization Errors

- HTTP `401` means the request has no acceptable Athena credential, or the bearer is invalid, expired, revoked, or currently unable to authenticate because API Key access is disabled.
- HTTP `403` means Athena accepted the credential but the current account lacks the role, entitlement, module level, resource ownership, or other authorization required by the operation. Account self-service for API Key management can also return `403` when API Key access is disabled.
- An account in maintenance because login is disabled can be rejected with HTTP `503`.

The response body follows the gateway error envelope described in [Errors and Pagination](/docs/ai/errors-and-pagination.md). Do not treat a valid bearer as proof that a particular module or write operation is authorized.
