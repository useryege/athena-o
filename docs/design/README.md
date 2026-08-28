# Living Design Documentation

This directory is the repository-internal guide to ATHENA's currently implemented design. It is written for developers and AI agents that need to understand component responsibilities, runtime behavior, state, interfaces, and operational constraints before changing the code.

The executable code remains the source of truth. These documents provide the maintained explanation of that code. If a document and the implementation disagree, inspect the implementation, correct the document in the same task, and avoid preserving the mismatch as historical commentary.

## Reading Order

1. Read this index to understand the documentation rules and locate the relevant subsystem.
2. Read every design document that covers the component or capability being changed.
3. Follow the source links and inspect the named symbols before planning or implementation.
4. Revisit the affected design documents before completing the change.

## Document Map

| Subsystem | Capability | Document |
| --- | --- | --- |
| Development Runtime | Local process supervision, persistent infrastructure, provider and scoped step-up state, Worm credential database, isolated disabled-auth identity, and full current-state reset lifecycle | [Local Runtime Orchestration](development-runtime/local-runtime-orchestration.md) |
| Developer Experience | Public LLM discovery documents, one-time Connect AI instructions, Swagger generation and embedding, unauthenticated documentation delivery, and root-path and safety boundaries | [AI Discovery Documentation](developer-experience/ai-discovery-documentation.md) |
| Identity and Access | UUID account identities, immutable public usernames, permanent single-provider bindings, persistent API Keys, Athena JWT v3, typed request credentials, and interactive-only sensitive boundaries | [Account Credentials](identity-access/account-credentials.md) |
| Identity and Access | Browser Google Authorization Code flow, PKCE, one-time OAuth state, shared anonymous username registration, and Athena cookie issuance | [Google OIDC Login](identity-access/google-oidc-login.md) |
| Identity and Access | Browser-injected Phantom Solana authentication, one-time SIWS challenges, Ed25519 verification, and wallet-first username registration | [Solana Wallet Authentication](identity-access/solana-wallet-authentication.md) |
| Identity and Access | Database-role authorization, credential-specific Wallet rules, login, API Key, Profit Sharing, ten-module access, Pending state, and transactional administrator control | [Account Access Control](identity-access/account-access-control.md) |
| Identity and Access | UUID-owned display profiles, immutable username presentation, display-only tiers, and cross-device theme preferences | [Account Profile and Preferences](identity-access/account-profile-and-preferences.md) |
| Identity and Access | Private account and Wallet avatar validation, S3-compatible object storage, distinct authorization, authenticated delivery, and orphan recovery | [Account and Wallet Avatar Storage](identity-access/account-avatar-storage.md) |
| Identity and Access | UUID-owned EVM and Solana custody, canonical key import/generation, required remarks, private avatars, owner-only safe metadata, and the purpose-bound Worm credential signer | [Wallet Ownership and Custody](identity-access/wallet-ownership.md) |
| Identity and Access | Independent login-only Wallet-reveal and Worm-credential proofs, rate-limited Google/Solana reauthentication, and fixed five-minute scope-bound Redis leases | [Wallet Secret and Worm Credential Reauthentication](identity-access/wallet-secret-reauthentication.md) |
| Web UI | Google and Phantom login, responsive shell, Pending access, custodial Wallet management, Worm balances/connections/activity, Connect AI, UUID-scoped caches, and administrator directory | [Application Shell](web-ui/application-shell.md) |
| Governance | UUID membership, immutable participant-name snapshots, entitlement-gated profit-allocation rounds, proposals, voting, and runoff resolution | [Profit Sharing](governance/profit-sharing.md) |
| Market Intelligence | Read-only Market Radar module, Polymarket discovery, rolling price windows, mover ranking, and alerts | [Market Radar](market-intelligence/market-radar.md) |
| Market Intelligence | Current Polymarket sports synchronization, price history, and price/score alerts | [Sports Live](market-intelligence/sports-live.md) |
| Market Intelligence | Completed ATP/WTA event synchronization, price history, status, and manual refresh | [Sports History](market-intelligence/sports-history.md) |
| Market Intelligence | Managed Optimistic Oracle log ingestion, market enrichment, reads, scans, and alerts | [Managed OO](market-intelligence/managed-oo.md) |
| Market Intelligence | Worm sports-market synchronization, rules, live state, history, and alerts | [Worm Markets](market-intelligence/worm-markets.md) |
| Trading | Owner-scoped Solana balances, official HMAC wallet connections, encrypted Worm credentials, open positions and in-flight requests, and partial-failure semantics | [Worm Trading](trading/worm-trading.md) |
| Token Intelligence | Synchronous EVM block discovery, token validation, project initialization, and per-attempt processing diagnostics | [Token Chain Processor](token-intelligence/chain-processor.md) |
| Token Intelligence | Per-project WETH and USDT Pair Swap-block collection | [Token Swap Processor](token-intelligence/swap-processor.md) |
| Token Intelligence | On-chain ERC-20, pair, wallet, and simulation-state aggregation | [ATHENA EVM Aggregator Contract](token-intelligence/athena-contract.md) |
| Token Intelligence | Research lifecycle and collection scheduling | [Token Research Lifecycle](token-intelligence/research-lifecycle.md) |
| Token Intelligence | Ave token market data and canonical pair collection | [Ave Market Data Collection](token-intelligence/ave-market-data.md) |
| Token Intelligence | Unified project pages, current snapshots, Report risk, trends, and history reads | [Token Project Read Model](token-intelligence/project-read-model.md) |
| Token Intelligence | One-time pre-deployment normal transactions for related wallets | [Project Wallet Pre-Deployment Normal Transactions](token-intelligence/wallet-normal-transactions.md) |
| Token Intelligence | Independent Token-module READ and READ_WRITE authorization for UI, requests, caches, and APIs | [Token Module Access Control](token-intelligence/access-control.md) |
| Blockchain Data | Finalized inbound BSC transaction indexing and lookup | [BSC Inbound Normal Transactions](blockchain-data/bsc-inbound-normal-transactions.md) |
| Blockchain Data | Finalized BSC V2 Swap-topic transaction indexing and lookup | [BSC V2 Swap Transactions](blockchain-data/bsc-v2-swap-transactions.md) |
| Blockchain Data | Etherscan API-key and Gateway request scheduling | [Etherscan Manager](blockchain-data/etherscan-manager.md) |

Use [the design document template](template.md) when adding a subsystem or an independently understandable capability.

## Maintenance Principles

- Organize documents by subsystem and capability, not by date, issue, requirement, pull request, or release.
- Describe only the current implementation. Replace obsolete design text instead of retaining change logs, migration narratives, future plans, or deprecated behavior.
- Link to the real source files and name the important symbols. Do not copy large implementation fragments into documentation.
- Keep responsibility boundaries, runtime flow, state and data, interface contracts, configuration defaults, dependencies, recovery behavior, health checks, and observability aligned with the code.
- Update the relevant document in the same task when a design-level behavior changes.
- Purely local refactors, formatting changes, copy edits, and generated-file refreshes that do not change design semantics do not require documentation changes.

## Adding a Document

1. Choose a stable subsystem directory and capability-oriented filename.
2. Copy the section structure from [the template](template.md).
3. Inspect the implementation and link every primary entry point, boundary, state store, and operational endpoint.
4. Add the document to the map above.
5. Check every relative link and remove placeholders before completing the task.
