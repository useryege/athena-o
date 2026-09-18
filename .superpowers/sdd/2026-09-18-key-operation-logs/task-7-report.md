# Task 7 实施报告

## 实现

- 在 `internal/devruntime` 注册独立 `operation-log` 服务，默认监听 `127.0.0.1:8124`，仅依赖 PostgreSQL；完整栈和 API schema 选择会准备 account 与 operation-log schema，但不会隐式启动日志服务。
- 为日志服务和 API producer 配置独立环境白名单、内部 token、cursor HMAC key 的 0600 持久化和互不复用校验；API 环境不包含 cursor key。API 与日志服务在同一实例中复用 account DSN。
- 增加 operation-log 专用 schema owner 和迁移 binary 路由；`athena-migrate --module all` 对 operation-log 使用专用 schema owner，不经过公共迁移器。
- 增加独立 operation-log Dockerfile、migration 工具镜像、生产 Compose 内部 TLS 服务、健康探针、迁移 profile、secret 隔离和 account-state 维护归属；日志服务不发布宿主机端口。
- devruntime readiness/status 现在按配置支持 operation-log plaintext 和 TLS 探测；生产 Compose 的 operation-log 与 account schema 强制使用同一 DSN。

## 验证

- 红灯基线：`task-7-red.log`。
- 通过：`task-7-unit-final2.log`、`task-7-race-final2.log`、`task-7-vet-final2.log`、`task-7-compose-final.log`、`task-7-shellcheck-final.log`。
- 通过：`go build` 独立 operation-log、专用 migration 和聚合 `./cmd` binary。
- 通过：`task-7-migration-up.log` 与 `task-7-migration-verify.log`，真实任务 PostgreSQL `127.0.0.1:56669` 上 operation-log schema up/verify；account schema 由 `task-7-account-migration-up.log` 准备。
- 通过：`task-7-service.log`，独立 binary 在真实任务数据库上监听并被探测；`TestOperationLogReadinessUsesConfiguredTLS` 覆盖 TLS readiness。
- 通过：`task-7-devruntime-integration.log`，真实 Docker PostgreSQL managed schema preparation 覆盖 operation-log owner、共享 account verify 和零消费者前置条件。
- `git diff --check`、`bash -n hack/lib/account-state-deploy.sh hack/prod-start-local.sh` 和生产 Compose 契约检查通过。

## 独立审阅

复审文件为 `task-7-final-review.md`。Critical=0、Important=0。审阅期间修复了 Compose operation DSN 与 account DSN 可分叉、account-state 维护停止归属、TLS CA 健康探测以及 devruntime TLS readiness 四项问题。剩余 Minor：API 与 operation-log 分开建立不同 devruntime instance 时，若未显式配置同一 internal token，日志调用按 fail-open 规则不可用；完整栈和生产 Compose 已共享凭据。

## 边界

未执行生产部署或推送。真实 ATHENA 全栈会员／管理员验收、V01–V18 汇总和人工审查材料属于 Task 8。
