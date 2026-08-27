# Athena Full-Account AI Access

Athena intentionally allows an ordinary account holder to give an account-level API Key to an AI system of their choice and let that system act as the user through Athena's HTTP API. The AI may be local, self-hosted, or hosted by a third party. Athena applies the same bearer behavior in every case.

This is direct credential handoff, not bounded authorization delegation.

## Authority of the Bearer

An API Key represents the owning account's complete current authorization for API-Key-eligible operations within Athena's HTTP API. It has no independent module scopes, read-only setting, operation allowlist, agent identity, or human approval gate.

A bearer can perform every API-Key-eligible read, write, sensitive-data, and account self-service operation that the account can currently perform. This includes API Key management when the account's current authorization permits it.

Some operations explicitly require an interactive browser login rather than a bearer credential. Wallet creation and import are unavailable to API Keys. Private-key reveal is also unavailable to API Keys and requires the owning user's browser session, Wallet `READ_WRITE`, and a fresh five-minute reauthentication lease. API Keys retain owner-scoped wallet metadata and avatar access at the applicable Wallet module level.

Full-account authority means all API-Key-eligible operations available to the owning account, not all operations in Athena. Server-side module checks, credential-class requirements, entitlements, memberships, ownership rules, business preconditions, and other authorization boundaries continue to apply when relevant to the requested operation.

The fixed administrator account does not support API Keys, so administrator authority is outside this AI-access model.

## Execution Model

Athena does not distinguish an AI caller from another bearer client and does not require the account holder to approve individual requests. Once authentication, authorization, and operation-specific validation succeed, the request executes normally.

Any confirmation workflow, tool-selection policy, prompt-injection defense, or secret-storage behavior supplied by an AI product belongs to that product. It is not an Athena authorization boundary.

## Connecting an AI

Use the browser workflow:

1. Open **Account Center > Security > Connect AI**.
2. Choose **Create connection**.
3. Choose **Copy connection instructions** from the one-time result.
4. Paste the complete instructions into the selected AI.

The instruction block contains absolute URLs for the Athena base, `/llms.txt`, the Swagger 2.0 contract at `/swagger.json`, and `/api/v1/session/userinfo`; the expected account ID; and the complete Bearer authorization value. It directs the AI to read the discovery document first, use Swagger as the source of truth for operations and schemas, verify its account identity, and then act with that ordinary account's complete current authority.

The selected AI must support HTTP requests or custom API tools. Pasting the instructions into a chat product without that capability does not give it network access to Athena. The AI sends the key as:

```http
Authorization: Bearer <athena-api-key>
```

While presenting the result, the Web UI checks the new credential against `/api/v1/session/userinfo` without sending ambient browser credentials. This check validates the bearer and expected account identity only; it does not prove that the external AI is connected. Verification failure does not hide the instructions, and the user can retry the check or copy the block for manual use.

The result is available only during creation. **Done** clears the bearer and assembled instructions from the page. Existing API Key rows contain metadata only and cannot recreate a connection block, so a lost block requires a new connection. The separate **Create API key** workflow remains available for non-AI clients.

## Credential Lifetime

Anyone possessing the bearer can act as the account while the key remains valid. Athena cannot control whether an external AI system stores, copies, or redistributes it.

A key expires at its configured time or can be revoked by the account holder. Disabling API Key access pauses all retained keys, and disabling account login blocks both browser sessions and API Keys. Because a bearer can create another API Key when account self-service permits it, revoking one key does not invalidate other keys belonging to the account.

See [Authentication](/docs/ai/authentication.md) for credential behavior, [Modules and Permissions](/docs/ai/modules.md) for the authorization model, and [Errors and Pagination](/docs/ai/errors-and-pagination.md) for client handling.
