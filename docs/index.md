# Overview

<!-- markdownlint-disable MD026 -->
## What Is Athena?
<!-- markdownlint-enable MD026 -->


Athena is a trading/project sync system for the blockchain.

![Athena Architecture](assets/athena-architecture.png)

## Requirements and Research

- [需求目标设计索引](requirements/README.md)
- [Polygon PoS 自建节点配置与 Hetzner 费用调研](polygon-pos-node-hetzner-research.md) — 默认 RPC 端口、主网硬件要求、服务器月租与磁盘容量限制。
- [Trader Sync 产品需求](requirements/polymarket-copy-trading/README.md) — 首期按 10 人设计；后端及[UI 完整书面均已确认](superpowers/specs/2026-09-10-trader-sync-activity-alerts-ui-design.md)，尚未实施；未来 Copy Trading 具体需求另行讨论。
- [Token 业务设计与研究资料](requirements/token/README.md)
- [Token 目标设计](requirements/token/token.md) — 中文需求草案，包含已确认子项和尚待讨论内容。
- [Token 流程图索引](requirements/token/README.md#流程图) — 每张流程图独立保存，附规则与待定边界说明。

## Backend Technical Design

- [后端技术设计索引](design/README.md) — 长期维护系统设计与源码关系，区分目标方案和当前实现。

## Development Workflow

- [Superpowers 开发工作流](developer-guide/superpowers-development.md) — 任务规格与计划分别保存在 `docs/superpowers/specs/`、`docs/superpowers/plans/`；需求与系统设计目录保存项目长期知识。
- [开发者指南](developer-guide/index.md)

## Operations

- [Makefile Commands and Deployment](operator-manual/makefile-commands.md)
- [BSC Transaction Indexer Server](bsc-transaction-indexer-server.md)
- [BSC Swap Indexer Server](bsc-swap-indexer-server.md)
- [Etherscan Gateway Servers](etherscan-gateway-servers.md)
