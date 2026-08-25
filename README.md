# Athena

Athena 是一个面向区块链与预测市场的情报分析平台，用于采集链上数据、同步市场状态、分析代币项目风险，并通过 Web UI、API 和通知服务提供分析结果。

> [!WARNING]
> Athena 当前处于开发阶段，接口、数据结构和部署方式可能发生破坏性变更。

## 核心能力

- **Market Intelligence**：同步 Polymarket 热门市场、体育市场和 Optimistic Oracle 数据，并提供价格变化与事件告警。
- **Token Intelligence**：处理 EVM 链上代币、交易对、钱包和 Swap 数据，支持项目研究、风险分析与报告。
- **Blockchain Indexing**：索引 BSC 普通转账和 V2 Swap 事件，为交易查询与上层分析提供数据。
- **Platform Services**：提供 Web UI、HTTP/gRPC API、通知服务、数据迁移以及 PostgreSQL/Redis 运行支持。

## 快速开始

本地开发需要：

- Go
- Docker
- Yarn

在项目根目录启动本地服务：

```bash
make run
```

认证默认开启。首次启动前，需要在 `.env` 中配置本地 Google Web OAuth client、
精确回调地址 `http://localhost:4000/auth/google/callback`，以及五个成员账号和
`admin` 的唯一 Google `sub`；配置缺失或重复时 API Server 会拒绝启动。完整步骤
参见 [本地运行指南](docs/developer-guide/running-locally.md)。

服务启动后通过以下地址打开 UI 并使用已批准的 Google 账号登录：

```text
http://localhost:4000
```

停止本地服务：

```bash
make stop
```

完整的环境变量、代码生成、本地运行和生产部署说明参见 [Makefile 操作手册](docs/operator-manual/makefile-commands.md)。

## 项目结构

- `cmd/`：各服务与命令行程序入口。
- `internal/`：主要业务实现。
- `pkg/`：公共 API、合约 ABI 和客户端代码。
- `ui/`：Web 前端。
- `docs/design/`：当前实现的 Living Design Documentation。
- `deploy/`：独立服务的部署配置。
- `hack/`：开发、代码生成和部署脚本。

## 文档

- [文档首页](docs/index.md)
- [当前系统设计](docs/design/README.md)
- [开发者指南](docs/developer-guide/index.md)
- [Makefile 操作手册](docs/operator-manual/makefile-commands.md)
- [API 文档](docs/developer-guide/api-docs.md)

## 开发约定

修改组件职责、运行流程、数据模型、接口契约或其他设计级行为前，请先阅读 [Living Design Documentation](docs/design/README.md)，并在同一变更中同步相关设计文档。
