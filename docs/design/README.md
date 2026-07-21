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
| Polymarket | Managed Optimistic Oracle log ingestion, market enrichment, and alerts | [Managed OO Alerts](polymarket/managed-oo-alerts.md) |
| Token Intelligence | EVM chain scanning and project candidate ingestion | [Token Scanner](token-intelligence/scanner.md) |
| Token Intelligence | Research lifecycle and collection scheduling | [Token Research Lifecycle](token-intelligence/research-lifecycle.md) |
| Token Intelligence | One-time related-wallet BSC funding-source history | [Wallet Funding Source History](token-intelligence/wallet-funding-source-history.md) |
| Blockchain Data | Finalized inbound BSC transaction indexing and lookup | [BSC Inbound Normal Transactions](blockchain-data/bsc-inbound-normal-transactions.md) |

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
