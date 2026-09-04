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
| Development Runtime | Local process supervision, persistent infrastructure, provider and scoped step-up state, Worm credential database, realm-selected dual disabled-auth identities, dual frontend entry points, and full current-state reset lifecycle | [Local Runtime Orchestration](development-runtime/local-runtime-orchestration.md) |
| Developer Experience | Public LLM discovery documents, one-time Connect AI instructions, Swagger generation and embedding, unauthenticated documentation delivery, and root-path and safety boundaries | [AI Discovery Documentation](developer-experience/ai-discovery-documentation.md) |
| Developer Experience | Explicit task-completion email delivery through fixed Tencent Exmail transport and recipient boundaries, invocation content, configuration precedence, and bounded retries | [Task Completion Email](developer-experience/task-completion-email.md) |
| Identity and Access | UUID account identities, immutable public usernames, realm-scoped provider bindings, independent member/admin login cookies, persistent API Keys, Athena JWT v3, typed request credentials, and interactive-only sensitive boundaries | [Account Credentials](identity-access/account-credentials.md) |
| Identity and Access | Realm-bound browser Google Authorization Code flow, PKCE, one-time OAuth state, administrator admission, anonymous username registration, and independent member/admin cookie issuance | [Google OIDC Login](identity-access/google-oidc-login.md) |
| Identity and Access | Browser-injected Phantom Solana authentication, one-time SIWS challenges, Ed25519 verification, and wallet-first username registration | [Solana Wallet Authentication](identity-access/solana-wallet-authentication.md) |
| Identity and Access | Realm-to-persisted-role session binding, credential-specific Wallet and Worm-selection rules, login, API Key, Profit Sharing, nine-module access, Pending state, and transactional administrator control | [Account Access Control](identity-access/account-access-control.md) |
| Identity and Access | UUID-owned display profiles, immutable username presentation, display-only tiers, and cross-device theme preferences | [Account Profile and Preferences](identity-access/account-profile-and-preferences.md) |
| Identity and Access | Private account and Wallet avatar validation, S3-compatible object storage, distinct authorization, authenticated delivery, and orphan recovery | [Account and Wallet Avatar Storage](identity-access/account-avatar-storage.md) |
| Identity and Access | UUID-owned EVM and Solana custody, canonical key handling, owner-only safe metadata, persisted owner-resolved Worm Wallet selection, and separate purpose-bound Worm credential and live-execution signers | [Wallet Ownership and Custody](identity-access/wallet-ownership.md) |
| Identity and Access | Independent Wallet-reveal and Worm-credential leases plus selection-reconciliation, exact-Run, and exact-position-Cash-Out proof boundaries; rate-limited Google/Solana reauthentication; and durable intent-bound authorization | [Wallet Secret and Worm Credential Reauthentication](identity-access/wallet-secret-reauthentication.md) |
| Web UI | Shared bootstrap/session kernel, deployment-root and application-root separation, two HTML/React entry points, realm selection, and cross-realm cleanup | [Application Shell](web-ui/application-shell.md) |
| Web UI | Member-only Google/Phantom login, Pending access, module navigation, account Telegram binding, Account Center and API Keys, Profit Sharing participation, Wallet custody, and business feature request/cache lifecycle | [Member Application Shell](web-ui/member-application-shell.md) |
| Web UI | Google-only administrator login, role guard, account administration, Profit Sharing governance, system notification operations and runtime status, and administrator self-service | [Administrator Application Shell](web-ui/administrator-application-shell.md) |
| Notifications | Ordinary-account Telegram binding, one-time deep links, durable polling, binding-revision fencing, idempotent account delivery, and unreachable-recipient recovery | [Account Telegram Notifications](notifications/account-telegram-notifications.md) |
| Notifications | Authenticated operational producers, Telegram group Topics, durable system deliveries, fair shared dispatch, administrator inspection/testing, and shared runtime status | [System Notification Operations](notifications/system-notification-operations.md) |
| Governance | UUID membership, immutable participant-name snapshots, entitlement-gated profit-allocation rounds, proposals, voting, and runoff resolution | [Profit Sharing](governance/profit-sharing.md) |
| Market Intelligence | Read-only Market Radar module, Polymarket discovery, rolling price windows, mover ranking, and alerts | [Market Radar](market-intelligence/market-radar.md) |
| Market Intelligence | Current Polymarket sports synchronization, price history, and price/score alerts | [Sports Live](market-intelligence/sports-live.md) |
| Market Intelligence | Completed ATP/WTA event synchronization, price history, status, and manual refresh | [Sports History](market-intelligence/sports-history.md) |
| Market Intelligence | Managed Optimistic Oracle log ingestion, market enrichment, reads, scans, and alerts | [Managed OO](market-intelligence/managed-oo.md) |
| Market Intelligence | Worm sports-market synchronization, rules, live state, history, and alerts | [Worm Markets](market-intelligence/worm-markets.md) |
| Trading | Revisioned owner-scoped selection of up to 20 Solana Wallets, removal-first official-HMAC connection/activity flows, saved combinations and previews, Cash Out, and official Web JWT live-Run orchestration | [Worm Trading](trading/worm-trading.md) |
| Trading | Provider-backed Worm event catalogs and owner-scoped, revisioned market-combination CRUD with trusted display snapshots | [Worm Market Combinations](trading/worm-market-combinations.md) |
| Trading | Durable asynchronous read-only execution previews limited to the current selected Wallet revision, with frozen Wallet and market order, authoritative Worm exposure and estimates, cumulative USDC simulation, and expiring owner-scoped review | [Worm Execution Preview](trading/worm-execution-preview.md) |
| Trading | Current-selection-gated Run admission, Run-bound Worm Web JWT execution, purpose-bound custodial signing, browser-led Wallet-major coordination, durable mutation attempts, reconciliation, and permanent owner-scoped history | [Worm Order Execution](trading/worm-order-execution.md) |
| Trading | Current-selection exact-position, fresh-proof-authorized Worm HMAC Cash Out with whole-position market Close, at-most-once dispatch, read-only recovery, and durable historical detail | [Worm Position Cash Out](trading/worm-position-cash-out.md) |
| Trading | Up-to-20 selected-Wallet, Wallet-major serial Cash Out batches with complete position freezing, single-operation reuse, confirmed-USDC advancement gates, durable Wallet locks, and manual recovery controls | [Worm Position Cash Out Batches](trading/worm-position-cash-out-batches.md) |
| Token Intelligence | Synchronous EVM block discovery, token validation, project initialization, and per-attempt processing diagnostics | [Token Chain Processor](token-intelligence/chain-processor.md) |
| Token Intelligence | On-chain ERC-20, pair, wallet, and simulation-state aggregation | [ATHENA EVM Aggregator Contract](token-intelligence/athena-contract.md) |
| Token Intelligence | PostgreSQL-backed six-source one-time collection, fenced workers, terminal barrier, and immutable ProjectProfile construction | [Token Collection and Project Profile](token-intelligence/collection-profile.md) |
| Token Intelligence | Ave token market data and canonical pair collection | [Ave Market Data Collection](token-intelligence/ave-market-data.md) |
| Token Intelligence | Project list/detail, collection evidence, immutable profile, wallet, and contract-source reads | [Token Project Read Model](token-intelligence/project-read-model.md) |
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
