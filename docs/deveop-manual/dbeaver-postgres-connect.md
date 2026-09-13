# 使用 DBeaver 连接本地 PostgreSQL

本地运行器默认给每个 checkout/实例分配独立持久库和动态 loopback 端口。连接前先确认目标实例，不能假定所有环境都使用 `localhost:5432`。

## 查询实例连接信息

从目标 checkout 启动所需服务，并在另一个终端查询状态：

```bash
make run-service SERVICE=trader-sync INSTANCE=ts-dev
make runtime-status INSTANCE=ts-dev
```

完整栈使用 `make run`，默认实例名 `full-stack`。状态给出本实例 PostgreSQL 容器与实际宿主端口；0600的 `.run/instances/<instance>/environment.json` 保存已解析的 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`，从中取数据库、用户与密码。managed 密码由实例生成并持久保存，不是根 `.env` 中 `POSTGRES_PASSWORD` 的值。每次重启可能改变宿主端口，应重新查询。

## 新建连接

1. 在 DBeaver 新建 PostgreSQL 连接。
2. 按目标实例填写以下信息。
3. 点击“测试连接”，成功后保存。

| 字段 | 值 |
| --- | --- |
| Host | 实例DSN的loopback地址，通常 `127.0.0.1` |
| Port | `runtime-status` / 当前DSN中的动态宿主端口 |
| Database | `athena`；全栈还包含 `wallet`、`profit_sharing` 等模块库 |
| Authentication | `Database Native` |
| Username / Password | 该实例DSN中的用户与密码 |

![DBeaver PostgreSQL 连接配置示意](../assets/pg-dbeaver-connect.png)

上图仅展示配置界面，端口和数据库名以当前实例为准。只开发TS时只有其所需的 `athena` 业务库；不能把旧示例 `application` / `worm` 当作当前模块名。

## 停止、重启与外部库

`make stop-instance INSTANCE=ts-dev` 停止本实例拥有的进程和容器，保留数据库volume。重启复用原库；`make reset-instance INSTANCE=ts-dev` 只在已停止时删除本实例数据。没有为了连接DBeaver而执行reset的步骤。

external 模式使用操作者明确提供的DSN，只读验证，不管理借用数据库的生命周期。如果原 owner 停库，DBeaver和借用服务都会断开；恢复后重新核对连接地址。

`hack/start-postgres-with-password.sh` 保留为**手动基础设施工具**：使用旧固定 `athena-postgres` 容器、`athena-local-postgres-data` volume及自身 `POSTGRES_*` 配置，不属于新实例运行器，也不会被 `make stop` 管理。仅在明确管理该手动实例时使用，不能与固定名已有容器并行。新开发优先使用上面的实例命令。

连接失败时先查 `runtime-status` 和对应容器日志，核对工作区、实例、动态端口和持久凭据。数据库不兼容或凭据不匹配时保留证据，不自动reset或删除其他实例数据。更多边界见[本地运行编排](../design/development-runtime/local-runtime-orchestration.md)。
