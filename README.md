# Athena

Athena 是一个面向区块链与预测市场的情报分析平台，用于采集链上数据、同步市场状态、分析代币项目风险，并通过 Web UI、API 和通知服务提供分析结果。

> [!WARNING]
> Athena 当前处于开发阶段，接口、数据结构和部署方式可能发生破坏性变更。

## 核心能力

- **Market Intelligence**：同步 Polymarket 热门市场、体育市场和 Optimistic Oracle 数据，并提供价格变化与事件告警。
- **Token Intelligence**：处理 EVM 链上代币、交易对和钱包数据，支持项目研究、风险分析与报告。
- **Blockchain Indexing**：索引 BSC 普通转账和 V2 Swap 事件，为交易查询与上层分析提供数据。
- **Trader Sync Activity Alerts**：监控人工选择的低频 Polymarket 交易者，保留站内活动并发送 Telegram 普通或摘要提醒；不执行交易。
- **Platform Services**：提供 Web UI、HTTP/gRPC API、通知服务、数据迁移以及 PostgreSQL/Redis 运行支持。

## 快速开始

本地开发需要：

- Go
- Docker
- Yarn

首次初始化当前数据模型时清空本地运行状态，然后启动服务：

```bash
make run-reset
make run
```

认证默认开启。首次启动前，需要在 `.env` 中配置本地 Google Web OAuth client、
精确回调地址 `http://localhost:4000/auth/google/callback` 和
`ATHENA_ADMIN_GOOGLE_EMAIL`。普通 Google 用户不需要预先登记 `sub`：未知身份通过
OIDC 校验后会进入 `/register` 选择永久 username；提交成功时系统才创建以 UUID
`account_id` 为内部身份、默认无业务权限的账号，由管理员在账号管理页授权。匹配管理员
邮箱的注册者只是管理员候选，最终角色由账号记录的 `administrator` 字段确定。完整步骤参见
[本地运行指南](docs/developer-guide/running-locally.md)。

服务启动后通过以下地址打开 UI，并使用任意已验证邮箱的 Google 账号登录：

```text
http://localhost:4000
```

停止本地服务：

```bash
make stop
```

完整的环境变量、代码生成、本地运行和生产部署说明参见 [Makefile 操作手册](docs/operator-manual/makefile-commands.md)。
Trader Sync 独立运行，通过 `make run-service SERVICE=trader-sync` 只启动它和本实例 PostgreSQL。API、Trader Sync 与 Notification 各自持有 pool，共用 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` 指向的权威数据库；本地单实例生命周期、Polygon HTTP/WSS、Profile 来源、代理值和重置边界见[本地运行时编排](docs/design/development-runtime/local-runtime-orchestration.md)。

## 项目结构

- `cmd/`：各服务与命令行程序入口。
- `internal/`：主要业务实现。
- `pkg/`：公共 API、合约 ABI 和客户端代码。
- `ui/`：Web 前端。
- `docs/requirements/`：长期维护的业务需求、目标行为与已知决定。
- `docs/design/`：长期维护的系统设计、实现事实与源码关系。
- `docs/superpowers/specs/`：Superpowers 架构任务的设计规格。
- `docs/superpowers/plans/`：Superpowers 多步骤任务的实现计划。
- `deploy/`：独立服务的部署配置。
- `hack/`：开发、代码生成和部署脚本。

## 文档

- [文档首页](docs/index.md)
- [需求目标设计](docs/requirements/README.md)
- [后端技术设计](docs/design/README.md)
- [Superpowers 开发工作流](docs/developer-guide/superpowers-development.md)
- [开发者指南](docs/developer-guide/index.md)
- [Makefile 操作手册](docs/operator-manual/makefile-commands.md)
- [API 文档](docs/developer-guide/api-docs.md)
- [Trader Sync 需求](docs/requirements/polymarket-copy-trading/target-trade-monitoring-notifications.md)、[实现设计](docs/design/trading/trader-sync-activity-alerts.md)与[验收记录](docs/testing/trader-sync-activity-alerts-acceptance.md)

## 开发约定

开发任务采用原版 [Superpowers 工作流](docs/developer-guide/superpowers-development.md)，按上游规则开展需求探索、设计、计划、TDD、执行与审查。开始前读取相关[业务需求](docs/requirements/README.md)、[系统设计](docs/design/README.md)和实际源码；任务规格与计划使用 `docs/superpowers/specs/`、`docs/superpowers/plans/`，完成后同步项目长期知识。
