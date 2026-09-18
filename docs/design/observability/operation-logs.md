# 关键操作日志技术设计入口

> 设计状态：已确认待实现。2026-09-18 用户整体审阅通过，并明确要求暂不进入实施；尚未编写实施计划，代码未实现。
>
> 关联需求：[关键操作日志](../../requirements/observability/key-operation-logs.md)。完整技术契约见[详细规格](../../superpowers/specs/2026-09-18-key-operation-logs-design.md)。

## 当前差距

当前 API 有运行日志、账户身份与权限及局部最后修改人记录，没有统一的关键操作历史查询服务。原生 HTTP 和 gRPC 都存在关键操作，仅在一种传输层增加拦截器会遗漏登录、敏感授权和部分业务。

## 目标职责

独立 operation-log 服务拥有原始事件、收件状态、查询投影和管理员查询。API 中的窄 producer adapter 根据可信身份和业务提交事实产生事件，以有界写入进入 PostgreSQL 的 operation_log schema；API 不启动日志消费 runtime。管理员通过 API facade 和内部 gRPC 查询，服务本身复核内部身份及持久管理员权限。

采集采用显式同库持久消息协议，查询采用 gRPC。共享账户 PostgreSQL 允许独立 schema 和服务进程，但保持同一数据库故障域；事务和连接由各自 owner 管理，不跨 RPC 传递。对应 SDS-R1／R2／R4／R6 的方案与例外理由已写入规格。

## 契约位置

| 内容 | 权威文档 |
| --- | --- |
| 事件及真实采集位置 | [事件目录](../../superpowers/specs/2026-09-18-key-operation-logs/event-catalog.md) |
| 身份、字段、提交点和结果 | [采集与结果](../../superpowers/specs/2026-09-18-key-operation-logs/collection-and-results.md) |
| schema、迁移、队列、版本和快照 | [存储与查询](../../superpowers/specs/2026-09-18-key-operation-logs/storage-and-query.md) |
| 管理员 API、错误与前端流程 | [接口和页面](../../superpowers/specs/2026-09-18-key-operation-logs/api-and-admin-ui.md) |
| 服务入口、配置、生成、故障和验收 | [运行与验证](../../superpowers/specs/2026-09-18-key-operation-logs/runtime-and-verification.md) |

目标路径为 `internal/operationlog`、`cmd/athena-operation-log`、`cmd/athena-operation-log-migrate`、`internal/server/operationlog` 与管理员 Operation Logs 页面，均尚未实现。构建、局部启动和停止入口须在实施时与现有本地 registry、schema owner 及生产 Compose 一同落实；不执行未授权生产部署。

## 不变量与维护

日志异常时业务继续；补记只作用于日志。成功／已受理／失败／拒绝／未知等结果依赖提交事实，不从 HTTP 200 推断。原始事件只追加，查询版本用于固定跨页快照；管理员角色变化、登录失效或账户切换后重新核验并清理客户端旧数据。

实施时按源到消费者更新 SQL／sqlc、proto、生成接口、API 和 UI，不手改生成物。V01–V18 是所需验证，不是已通过结果。完成代码后应把本入口更新为实际源码、差距和验收证据；当前文档静态检查不证明服务能运行。
