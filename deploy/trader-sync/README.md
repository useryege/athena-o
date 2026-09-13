# Trader Sync 生产部署

Trader Sync 使用独立镜像 `TRADER_SYNC_IMAGE`，仅包含 `athena-trader-sync` 和
`athena-account-state-migrate` 两个应用程序。构建入口不触发 Node、UI、代码生成或
聚合主程序；其他服务继续使用 `PROD_IMAGE`。

```bash
make build-service-image SERVICE=trader-sync TRADER_SYNC_IMAGE=athena-trader-sync:release
```

镜像默认启动业务服务。迁移必须显式调用 Compose 的工具服务：

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod --profile tools \
  run --rm --no-deps athena-account-state-migrate verify
```

生产参数模板位于 [`.env.prod`](../../.env.prod)。API、Notification、Trader Sync
及 schema tool 必须配置完全相同的 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`，没有旧名称
或 localhost 回退。服务启动前必须已有正确 schema。初始化新的数据库同样需要显式维护；
不要用 reset 或删除数据卷处理不兼容的现有数据库。

## TLS 与配置

内部服务名是 `athena-trader-sync:8122`，没有宿主端口。证书 SAN 必须包含
`ATHENA_TRADER_SYNC_TLS_SERVER_NAME`（默认 `athena-trader-sync`）；API 校验 CA 和
服务名。容器内健康检查访问 `127.0.0.1:8122`，仍校验同一证书名及标准 gRPC 服务
`tradersync.internal.v1.TraderSyncService`，不读取数据库、token、cursor 或数据源配置。
业务就绪等待配置为 60 秒；停止宽限期 40 秒覆盖进程的 30 秒总停止预算。

环境模板中的五个 `ATHENA_TRADER_SYNC_*_FILE` 指向独立 token、cursor key、证书、
私钥和 CA 文件。内部 token 至少 32 字节且不含空白，必须与 cursor key 独立；保留已有
cursor key 的值可继续验证现有游标。HTTP、WSS、proxy、站点 URL 按部署实际值配置。
生产不接受 token/cursor 的明文环境变量。API 只挂载内部 token 和 CA；Notification
不获得这些 Trader Sync 文件。两个独立进程不通过 Compose 健康依赖互相阻塞启动。

文件必须可由 UID/GID `999:999` 读取。远程脚本将文件作为归档上传，统一在
`secrets/trader-sync-*` 下设为 `999:999`、`0600`，重写部署目录 `.env` 中的相对文件路径；
不会将凭据写入镜像或通过 SSH 参数传递。证书与 token 轮换需在调用方和服务端协调执行。
构建上下文排除 `.env*`、`secrets`、运行状态、验收证据和其他工作树。

## 替换与 schema 维护

只更新 Trader Sync 且 schema 兼容时，先构建镜像，再运行：

```bash
PROD_ENV_FILE=.env.prod TRADER_SYNC_IMAGE=athena-trader-sync:release \
  bash hack/prod-remote-deploy.sh trader-sync-deploy
```

该路径用新镜像的 schema tool `verify` 检查 migration 集及实际数据库 catalog。
兼容时只停止、确认退出并替换 Trader Sync，不停止 API/Notification，不执行 schema up。
它仍上传部署配置与镜像；只更新 TS 时应保持其他镜像标签及配置不变。

schema 不兼容时脚本拒绝继续，除非操作者已安排维护窗口、确认所有管理范围外的同库
消费者已经停止，并明确配置 `PROD_ACCOUNT_STATE_MAINTENANCE=true` 和
`PROD_ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED=true`。这两个值是操作者的事实声明，
脚本无法发现其他主机或其他 Compose 项目中全部同库连接者。

获授权的维护顺序固定为：停止 API、Notification、Trader Sync → 确认三个服务退出 →
独立 tool `up` → 独立 tool `verify` → 启动服务。任一迁移/验证失败都不会启动消费者。
TS 专项部署若确实改变 schema，也会恢复被停止的三个消费者。全栈部署的其他 schema
仍由原聚合 migrator 处理，明确排除 account-state，防止另一镜像越过该数据库的维护边界。
单活采集重启期间会中断，重启恢复并展示中断，不历史补查，不提供滚动零停机保证。

`make prod-hot-deploy-remote` 更新完整栈并保留数据卷；现有 `make prod-deploy-remote`
是重新创建生产数据卷的全量部署入口，不能用作保留数据的升级。本文实现及测试不执行
SSH 或生产发布。

## 验证与规范证据

```bash
bash hack/trader-sync-deploy_test.sh
bash hack/deploy-scripts_test.sh
TRADER_SYNC_IMAGE=athena-trader-sync:release bash hack/trader-sync-image_test.sh
```

前两项使用真实 Compose 解析及记录 argv 的本地命令替身，验证镜像 build/inspect/save/load、
环境传递、维护顺序、失败不重启和 TS 专项替换，不实际 SSH。最后一项使用真实镜像，
创建带唯一标签的内部网络和临时 PostgreSQL，执行 schema 初始化、正确 TLS 健康、错误
CA/服务名拒绝、空环境健康命令、非 root 最小文件系统及优雅退出。它不访问 Telegram
或真实采集端点，并且只清理自身标签匹配的资源；不能替代真实数据源和全栈 UI 验收。

本服务边界符合 [SDS-R1/R2/R3/R4/R6/R7/R8](../../docs/developer-guide/service-development-standards.md)：
独立业务进程与 gRPC、独立镜像/TLS、按进程配置、共享 schema 维护和部署证据。
测试资源归属及局部清理遵守 SDS-R5。详见
[已批准边界设计](../../docs/superpowers/specs/2026-09-13-trader-sync-service-boundaries-design.md)。
