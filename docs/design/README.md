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
| Development Runtime | Local process supervision, capability process composition, and persistent infrastructure lifecycle | [Local Runtime Orchestration](development-runtime/local-runtime-orchestration.md) |
| Market Intelligence | Polymarket hot-market discovery, rolling price windows, mover ranking, and alerts | [Market Radar](market-intelligence/market-radar.md) |
| Market Intelligence | Current Polymarket sports synchronization, price history, and price/score alerts | [Sports Live](market-intelligence/sports-live.md) |
| Market Intelligence | Completed ATP/WTA event synchronization, price history, status, and manual refresh | [Sports History](market-intelligence/sports-history.md) |
| Market Intelligence | Managed Optimistic Oracle log ingestion, market enrichment, reads, scans, and alerts | [Managed OO](market-intelligence/managed-oo.md) |
| Market Intelligence | Worm sports-market synchronization, rules, live state, history, and alerts | [Worm Markets](market-intelligence/worm-markets.md) |
| Market Intelligence | Configured Worm/Polymarket FIFA composition, balances, and requester holdings | [FIFA Market Dashboard](market-intelligence/fifa-market-dashboard.md) |
| Token Intelligence | Synchronous EVM block discovery, token validation, project initialization, and per-attempt processing diagnostics | [Token Chain Processor](token-intelligence/chain-processor.md) |
| Token Intelligence | Per-project WETH and USDT Pair Swap-block collection | [Token Swap Processor](token-intelligence/swap-processor.md) |
| Token Intelligence | On-chain ERC-20, pair, wallet, and simulation-state aggregation | [ATHENA EVM Aggregator Contract](token-intelligence/athena-contract.md) |
| Token Intelligence | Research lifecycle and collection scheduling | [Token Research Lifecycle](token-intelligence/research-lifecycle.md) |
| Token Intelligence | Ave token market data and canonical pair collection | [Ave Market Data Collection](token-intelligence/ave-market-data.md) |
| Token Intelligence | Unified project pages, current snapshots, Report risk, trends, and history reads | [Token Project Read Model](token-intelligence/project-read-model.md) |
| Token Intelligence | One-time pre-deployment normal transactions for related wallets | [Project Wallet Pre-Deployment Normal Transactions](token-intelligence/wallet-normal-transactions.md) |
| Token Intelligence | Administrator-only UI and API access control | [Token UI and API Access Control](token-intelligence/access-control.md) |
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
