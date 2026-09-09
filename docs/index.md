# Overview

<!-- markdownlint-disable MD026 -->
## What Is Athena?
<!-- markdownlint-enable MD026 -->


Athena is a trading/project sync system for the blockchain.

![Athena Architecture](assets/athena-architecture.png)

## Requirements and Research

- [需求目标设计索引](requirements/README.md)
- [Polymarket 交易员跟随产品需求](requirements/polymarket-copy-trading/README.md) — 产品名称待确认；产品规划包含活动订阅与通知、未来 Copy Trading 两个板块，当前只讨论第一阶段。
- [Token 业务设计与研究资料](requirements/token/README.md)
- [Token 目标设计](requirements/token/token.md) — 中文需求草案，包含已确认子项和尚待讨论内容。
- [Token 流程图索引](requirements/token/README.md#流程图) — 每张流程图独立保存，附规则与待定边界说明。

## Backend Technical Design

- [后端技术设计索引](design/README.md) — 编码前形成、实现后持续维护，并以状态区分目标方案和当前实现。

## Development Workflow

- [设计主导的后端开发工作流](developer-guide/design-led-backend-development.md)
- [开发者指南](developer-guide/index.md)

## Operations

- [Makefile Commands and Deployment](operator-manual/makefile-commands.md)
- [BSC Transaction Indexer Server](bsc-transaction-indexer-server.md)
- [BSC Swap Indexer Server](bsc-swap-indexer-server.md)
- [Etherscan Gateway Servers](etherscan-gateway-servers.md)
