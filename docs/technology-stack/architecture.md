# Overview

<!-- markdownlint-disable MD026 -->
## What Is Athena?
<!-- markdownlint-enable MD026 -->

Athena is an automated trading system designed for blockchain transaction scenarios. It continuously monitors on-chain data, maintains project data, and automatically determines buy and sell timing based on user-configured strategies. When strategy conditions are met, the system creates transactions through the Swap Server and, when necessary, accelerates confirmation through the Tx Speed Up Server. At the same time, the Protect Server continuously monitors user assets and order status, triggering protective sell actions when risks are detected to reduce potential losses.

![Athena Architecture](assets/athena-architecture.png)

## Token Intelligence

Token Intelligence is deployed as three coarse-grained services: `athena-token-discovery`, `athena-token-research`, and `athena-token-api`. Discovery owns chain scanning and project validation. Research owns scheduling, collection, immutable report construction, and project selection evaluation. The API process remains independent so a Research failure does not stop Discovery or management queries.

The worker binary accepts only `ATHENA_TOKEN_MODE=discovery` and `ATHENA_TOKEN_MODE=research`. The internal pipeline responsibilities remain separate components but are not deployed as individual microservices.

Project validation uses database-backed UUID leases and `FOR UPDATE SKIP LOCKED`, allowing multiple Discovery instances to claim different candidates and recover work after a crashed worker. Research collectors keep separate chain resources for every `(data type, chain ID)`, so a slow connection or reconnect in one source does not block the other collectors.

Research reports use normalized, schema-versioned observations. `ResearchReportV1.RiskSummary` is the source of truth for both immutable report JSON and database query projections. Selection strategies are registered by explicit key and version; the default `default/1` strategy always returns `deferred / strategy_not_configured`.

Discovery exposes health and Prometheus metrics on `127.0.0.1:8095` by default, while Research uses `127.0.0.1:8097`. Both provide `/healthz`, `/readyz`, and `/metrics`; `ATHENA_TOKEN_HEALTH_LISTEN_ADDRESS` overrides the listen address without restoring the removed Token gRPC status service.

See [Token Intelligence 与交易领域边界](../open-trade-time.md) for the future handoff from `ProjectSelection` to trade execution and exit decisions.
