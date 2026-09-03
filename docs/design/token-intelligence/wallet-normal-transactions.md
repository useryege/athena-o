# Project Wallet Pre-Deployment Normal Transactions

## Scope

The `wallet_normal_transactions` collector obtains one bounded historical
sample for every distinct project-related wallet. It reads blocks from zero
through the block immediately before project deployment, requests only the
first descending page of at most 300 transactions per wallet, and commits all
wallet associations only when every wallet succeeds. It is not a wallet-history
synchronizer and never collects post-deployment transactions.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Provider adapter | [internal/token/adapters/normaltransactions/provider.go](../../../internal/token/adapters/normaltransactions/provider.go) | `Provider.ListNormalTransactions` |
| Processor and summary | [internal/token/collection/application/processors.go](../../../internal/token/collection/application/processors.go), [internal/token/collection/application/normal_transactions.go](../../../internal/token/collection/application/normal_transactions.go) | `WalletNormalTransactionsProcessor`, `summarizeNormalTransactions` |
| Durable detail rows | [internal/token/adapters/postgres/wallet_normal_transaction_store.go](../../../internal/token/adapters/postgres/wallet_normal_transaction_store.go) | project-wallet transaction persistence and reads |
| Schema and SQL | [internal/token/adapters/postgres/migrations/000001_init.sql](../../../internal/token/adapters/postgres/migrations/000001_init.sql), [internal/token/adapters/postgres/queries/project_wallet_normal_transaction.sql](../../../internal/token/adapters/postgres/queries/project_wallet_normal_transaction.sql) | normal transaction tables/queries |
| API read | [internal/tokenapi/project_detail_service.go](../../../internal/tokenapi/project_detail_service.go) | `ListProjectWalletNormalTransactions` |

## Architecture

The collector obtains data through Etherscan Manager, which owns explorer
gateway selection and response normalization. Collection buffers all wallet
results in memory. The generic collection repository writes the V1 summary,
transaction facts, project-wallet associations, and succeeded task in one
fenced PostgreSQL transaction.

## Runtime Flow

1. Related wallets are deduplicated in stable project order and zero addresses
   are discarded.
2. For a positive deployment block, each wallet is requested sequentially for
   block range `0..deploymentBlock-1`, page 1, page size 300, descending order.
3. Every response is bounded to 300 and each row must lie within the requested
   range and involve the requested wallet. Hashes, addresses, unsigned numeric
   values, and receipt status are validated while mapping.
4. Any wallet failure aborts processing and none of the buffered rows are saved.
5. Before persistence, identical repeated rows for the same wallet and
   transaction hash are collapsed in provider order; conflicting duplicate rows
   reject the batch. The summary then counts wallet associations and distinct
   transaction hashes,
   classifies distinct-hash success/failure, computes related-wallet-set inflow
   and outflow, and ranks up to ten methods and counterparties deterministically.
6. A wallet returning exactly 300 rows is listed as capped. An empty result for
   every wallet is a successful, explicit zero summary.
7. Commit retains associations across different wallets while the profile
   summary deduplicates transactions by hash.

## State / Data

The result payload contains wallet count, association count, unique hash count,
success/failure counts, native-value inflow/outflow, top methods, top external
counterparties, and capped-wallet addresses. Detailed rows preserve block/time,
transaction and log position, nonce, from/to, value, gas, input, decoded method,
receipt status, error flag, and collection time.

One transaction may be associated with multiple related wallets. The database
uniqueness key preserves each required project-wallet association without
double-counting that transaction in the profile.

## Configuration

The collector uses `ATHENA_TOKEN_ETHERSCAN_MANAGER_SERVER_ADDRESS`, the Token
PostgreSQL DSN, and shared health settings. Page size 300, descending page one,
top-list size ten, and the pre-deployment range are fixed. Real failures retry
after ten minutes; polling, lease, and heartbeat defaults are one, 90, and 30
seconds.

## Invariants

- The upper bound is strictly before deployment; block zero deployment yields
  a valid empty sample without a provider call.
- Every distinct related wallet must succeed before any result is committed.
- No wallet contributes more than 300 associations.
- Detailed associations are unique per wallet and hash but remain separate
  across wallets, while profile metrics use distinct transaction hashes.
- Zero transactions is success, not missing evidence.
- The one-time collector never refreshes this sample.

## Failure Recovery

A provider or validation failure discards the complete in-memory batch and
consumes a real attempt. Database transaction failure rolls back detail rows,
summary result, and task status together but does not consume another provider
failure. Lease loss cancels remaining calls and rejects the batch. After three
real failures the source becomes terminal and the profile can be incomplete.

## Observability

Collector health and queue metrics expose availability and age. Task detail
shows failure count/error and summary evidence; the project transaction API
provides paged detailed rows filterable by wallet, receipt status, and method.
Capped wallets distinguish a bounded sample from a complete history.

## Change Checklist

- [ ] Recheck range, order, page size, and wallet-involvement validation.
- [ ] Recheck all-or-nothing multi-wallet commit and task fencing.
- [ ] Recheck association uniqueness and hash-deduplicated summaries.
- [ ] Recheck zero-result and capped-wallet semantics.
- [ ] Keep the [design index](../README.md) current.
