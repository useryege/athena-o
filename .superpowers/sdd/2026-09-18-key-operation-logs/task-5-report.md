# Task 5 报告：HTTP、认证、账户与 Worm 关键操作采集

## 实现

- 为目录批准的 61 个 HTTP/auth capture position 建立 catalog-driven route recorder；HTTP middleware 只观察状态和响应写入边界，业务事实仍由对应 handler 在可信下游结果后写入。gRPC-Web 请求跳过 native HTTP recorder，避免与 gRPC facade 重复记录。
- 完成账户头像、钱包头像、私钥 reveal、wallet selection replace、Worm connection/combination/execution plan/execution/Cash Out/Cash Out Batch 的资源、状态、版本、effect 和白名单详情采集。wallet selection 继续使用现有单 RPC 原子路径，不制造虚假 PARTIAL。
- 完成 Google、Phantom、development 身份登录、注册、登出、钱包 reveal 和 Worm 授权路径；Google callback 按可信 state 前缀分派，认证成功绑定下游返回的 account ID，注册使用 `UNAUTHENTICATED` credential kind，避免把未签发 session 写成 `LOGIN_SESSION`。
- 失败结果按业务证据映射为 `DENIED`、`FAILED` 或 `UNKNOWN`；已确认 effect 与后续失败保留 `PARTIAL` 语义。日志 sink、响应写入和日志状态不会阻断用户业务。

## 独立复审

`task-5-final-review.md` 记录了复审结论。复审覆盖 development 五个授权入口、Phantom challenge/verify/failure-only 路径、Google callback 状态选择、旧 cookie actor 归属、注册 child login、HTTP 状态映射和 gRPC-Web 去重；Critical/Important 均为 0。

## 验证证据

- `task-5-race.log`: `go test -race ./internal/server/... ./internal/authregistration/... ./internal/googleoidc/... ./internal/phantomauth/... ./internal/operationlog/...`
- `task-5-vet.log`: 受影响 Go 包 `go vet`
- `task-5-build.log`: `go build ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate`
- `task-5-integration.log`: 真实 PostgreSQL `athena-key-operation-logs-tests` (`127.0.0.1:56669`) 上 store/schema integration tests
- `task-6-build.log`: `yarn build`
- UI `yarn lint`：eslint-config、TypeScript、ESLint 均通过
- `git diff --check`：通过

上述专项验证均退出 0。Task 5 的产品改动在本任务分支提交，Task 6 前端改动另行提交；V01–V18、真实 ATHENA 运行环境验收和最终人工审查仍由 Task 8 负责。
