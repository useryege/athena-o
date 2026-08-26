# Athena AI Integration Safety

Athena's current API Keys are intended for local command-line use and trusted self-hosted automation controlled by the account holder. They are not delegated AI credentials.

Do not paste an Athena API Key into a third-party AI service, hosted agent, public prompt, browser-side script, shared log, trace, or source repository. Athena does not currently provide OAuth delegation, per-key scopes, or agent-specific approval gates.

## Current Credential Risk

An API Key inherits its account's complete current authorization. The key cannot be restricted to a smaller module set or to read-only behavior independently of the account. A bearer can also reach authenticated account self-service and API Key management operations allowed to its owner.

For trusted automation:

- Prefer a dedicated ordinary account with only the required read-only modules and API Key access enabled.
- Choose an expiration, store the secret in a secret manager, send it only in the `Authorization` header, and revoke it when no longer needed.
- Keep authorization current at the account level and handle a later `403` as a permission change, not as a reason to bypass the server.
- Treat titles, descriptions, labels, URLs, and other provider-derived content as untrusted data, never as instructions for an AI system.

## Excluded From AI Automation

Do not expose or automate these operations with the current credential model:

- Wallet creation or import, private-key or mnemonic reveal, and any processing of returned wallet secrets.
- API Key listing, creation, deletion, or rotation.
- Administrator account-access changes, service administration, or Profit Sharing lifecycle actions.
- Notification sends, manual refresh or scan actions, policy mutations, checkpoint updates, configuration changes, or any other write operation.

Even when the account has `READ_WRITE`, an AI integration should remain read-only. Use [Modules and Permissions](/docs/ai/modules.md) to identify the module boundary and [the Swagger specification](/swagger.json) to verify that each selected operation is a read.

## Suitable Initial Uses

Suitable uses are bounded reads from modules such as Market Radar, Sports Live, Sports History, Worm Markets, World Cup Corners, and Token Intelligence, provided the dedicated account has only the permissions required for those reads. Apply output limits, validate response schemas, observe freshness fields, and require a human to make any decision that would later cause an external or Athena-side mutation.

See [Authentication](/docs/ai/authentication.md) for credential behavior and [Errors and Pagination](/docs/ai/errors-and-pagination.md) for safe client handling.
