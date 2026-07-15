# Overview

<!-- markdownlint-disable MD026 -->
## What Is Athena?
<!-- markdownlint-enable MD026 -->

Athena is an automated trading system designed for blockchain transaction scenarios. It continuously monitors on-chain data, maintains project data, and automatically determines buy and sell timing based on user-configured strategies. When strategy conditions are met, the system creates transactions through the Swap Server and, when necessary, accelerates confirmation through the Tx Speed Up Server. At the same time, the Protect Server continuously monitors user assets and order status, triggering protective sell actions when risks are detected to reduce potential losses.

![Athena Architecture](assets/athena-architecture.png)

## Token Intelligence

Token Intelligence is a modular monolith backed by one PostgreSQL database and a persistent database task queue. Its code is divided into discovery, catalog, research, reporting, selection, and policy domains. PostgreSQL, EVM, Ave, and source-code integrations are adapters behind application-owned ports.

Production deploys ten isolated workers: scanner, validator, scheduler, five data-type collectors, report builder, and selector. Each worker uses its own binary name and health port while collaborating only through PostgreSQL. `athena-token-api` exposes catalog, research, policy, and operations gRPC services on the shared `8096` endpoint.

Project validation uses database-backed UUID leases and `FOR UPDATE SKIP LOCKED`, allowing multiple Discovery instances to claim different candidates and recover work after a crashed worker. Research collectors keep separate chain resources for every `(data type, chain ID)`, so a slow connection or reconnect in one source does not block the other collectors.

Research reports use normalized, schema-versioned observations. `ResearchReportV1.RiskSummary` is the source of truth for both immutable report JSON and database query projections. Selection strategies are registered by explicit key and version; the default `default/1` strategy always returns `deferred / strategy_not_configured`.

Workers expose `/healthz`, `/readyz`, and `/metrics` on ports `8110` through `8119`. Chain metadata and EVM endpoints come from the shared `ATHENA_TOKEN_CHAINS_JSON` registry; adding a chain does not require a worker code change.

See [Token Intelligence 与交易领域边界](../open-trade-time.md) for the future handoff from `ProjectSelection` to trade execution and exit decisions.
