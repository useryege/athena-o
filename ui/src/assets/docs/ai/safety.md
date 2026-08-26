# Athena Full-Account AI Access

Athena intentionally allows an ordinary account holder to give an account-level API Key to an AI system of their choice and let that system act as the user through Athena's HTTP API. The AI may be local, self-hosted, or hosted by a third party. Athena applies the same bearer behavior in every case.

This is direct credential handoff, not bounded authorization delegation.

## Authority of the Bearer

An API Key represents the owning account's complete current authorization within Athena's HTTP API. It has no independent module scopes, read-only setting, operation allowlist, agent identity, or human approval gate.

A bearer can perform every read, write, sensitive-data, and account self-service operation that the account can currently perform. This includes API Key management and wallet-secret operations when the account has the required module level and resource ownership.

Full-account authority means all operations available to the owning account, not all operations in Athena. Server-side module checks, entitlements, memberships, ownership rules, business preconditions, and other authorization boundaries continue to apply when relevant to the requested operation.

The fixed administrator account does not support API Keys, so administrator authority is outside this AI-access model.

## Execution Model

Athena does not distinguish an AI caller from another bearer client and does not require the account holder to approve individual requests. Once authentication, authorization, and operation-specific validation succeed, the request executes normally.

Any confirmation workflow, tool-selection policy, prompt-injection defense, or secret-storage behavior supplied by an AI product belongs to that product. It is not an Athena authorization boundary.

## Connecting an AI

Give the selected AI:

1. The Athena origin that serves the Web UI.
2. The API Key copied from its one-time creation display.
3. The discovery document at `/llms.txt`.

The AI can use `/llms.txt` to locate the API documentation and `/swagger.json` to discover the authoritative operations and schemas. It sends the key as:

```http
Authorization: Bearer <athena-api-key>
```

## Credential Lifetime

Anyone possessing the bearer can act as the account while the key remains valid. Athena cannot control whether an external AI system stores, copies, or redistributes it.

A key expires at its configured time or can be revoked by the account holder. Disabling API Key access pauses all retained keys, and disabling account login blocks both browser sessions and API Keys. Because a bearer can create another API Key when account self-service permits it, revoking one key does not invalidate other keys belonging to the account.

See [Authentication](/docs/ai/authentication.md) for credential behavior, [Modules and Permissions](/docs/ai/modules.md) for the authorization model, and [Errors and Pagination](/docs/ai/errors-and-pagination.md) for client handling.
