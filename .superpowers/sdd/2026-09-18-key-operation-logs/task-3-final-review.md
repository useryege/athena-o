# Task 3 第三轮最终独立审阅

审阅范围为 `ba5be08a..5929a540`，以及当前 worktree 中尚未提交的 proto、生成物、facade、gateway marshaler、transport 和 runtime 修复。依据 task brief、`api-and-admin-ui.md`、`runtime-and-verification.md`、设计/计划、Task 3 report 和前两轮 review 审阅；未修改产品代码。

## 验证

- `go test ./internal/operationlog/... ./internal/server/operationlog ./cmd/athena-operation-log`：通过。
- `go test -race ./internal/operationlog/store ./internal/operationlog/transport ./internal/server/operationlog`：通过。
- `go vet ./internal/operationlog/... ./internal/server/operationlog ./cmd/athena-operation-log`：通过。
- `go build ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate`：通过。
- `git diff --check`：通过。
- `ATHENA_TEST_PG_ADMIN_DSN=postgres://postgres:athena-operation-log-test@127.0.0.1:56669/postgres?sslmode=disable go test -tags integration ./internal/operationlog/query -run 'Test(AdapterReadsPublishedVersionsWithStableSnapshotAndKeyset|RuntimeStatusReadsPublicationAndPendingFacts)' -count=1`：通过（真实 PostgreSQL）。
- 真实 bufconn listener 测试覆盖 bearer、请求/响应 64 KiB 拦截和稳定 reason；临时 CA 的真实 TLS listener 握手测试通过。
- `TestProjectorReadinessRecoversAfterVerification` 直接验证 `queryReady=false` 经有界 Verify 后恢复为 true；新增 `refreshQueryReadiness` 已关闭此前“恢复后永久 NOT_SERVING”的状态漏洞。

## 前轮项目复核

- Nullable wrapper 的二进制 presence、public gateway scalar flatten、Detail 旧字段清理、`effect` 重复字段、`FILTER_INVALID`/`CURSOR_INVALID`/`SNAPSHOT_INVALID` 映射、snapshot 解码分类、投影错误与 query readiness 分离、成功投影后的 readiness refresh、Authenticator 的 Unauthenticated 语义均已修复到当前代码。
- `OperationLogDetail` 已删除旧文本字段并将 8–12 保留；两套 proto 与对应 pb 生成物能够编译，详情 `effect` 保持数组。
- 上一版发现的 gateway whitelist 遗漏已修复：`referenceVerified`、`observedAt`、`inFlightEvents` 已加入，资源和 CaptureStatus 的 scalar JSON 回归测试已补齐。

## Critical

无新的 Critical 代码问题。

## Important

### I1：仍没有独立服务 health/readiness 恢复的端到端证据

- **位置**：`cmd/athena-operation-log/main.go:87-103`；测试目前为 `internal/operationlog/store/runtime_test.go:25-39`、`internal/operationlog/transport/transport_test.go:72-117`、`internal/operationlog/rpcconfig/tls_test.go:17-76`。
- **证据**：readiness 单测只直接调用 `refreshQueryReadiness`，bufconn 测试未运行命令的 ticker/health server，TLS 测试只注册独立 health server；没有测试 `ATHENA` 命令启动后 `RuntimeStatus`/schema Verify 失败→`NOT_SERVING`→数据库恢复→`SERVING` 的完整路径，也没有把真实 account/operation-log schema 依赖接到该 listener。因而无法发现 `verifySchemas` 注入、ticker wiring 或 health 状态更新的集成回归。
- **建议**：在可控真实 PostgreSQL（或注入的 bounded Verify/RuntimeStatus）上启动 command 的 gRPC listener，使用 health client 断言故障、恢复和投影写失败但旧快照可读的状态；保留当前 bufconn/TLS 单测作为边界测试。Task brief 的 V17 服务侧证据完成前，不建议把 Task 3 标为完整通过。

## Minor

### M1：gateway 类型识别依赖生成包路径字符串

- **位置**：`internal/server/module_access.go:213-220`。
- `isOperationLogMessage` 通过 `reflect.Type.PkgPath()` 是否包含 `/pkg/apiclient/operationlog` 决定是否展平。当前生成包路径固定，功能测试通过；但重命名包或从另一 gateway 使用同一 DTO 时会静默退回 wrapper JSON。建议将 marshaler 作用域绑定到显式类型接口/独立 MIME 配置，并保留完整 golden 测试。

### M2：TLS 只有成功握手测试

- **位置**：`internal/operationlog/rpcconfig/tls_test.go:17-76`。
- 现有测试验证 CA、ServerName 和 listener 成功握手；没有错误 CA 或错误 ServerName 的拒绝断言。实现当前符合显式 TLS 配置要求，属于验证覆盖不足。

## Questions

1. Task 4 是否会把生成的 public gateway 注册到 API Server？当前 Task 3 的独立命令只注册内部服务，gateway marshaler 的实际公共路由仍由后续 API Server 注册负责；应在注册任务中复用本轮 scalar flatten golden。
2. V17 是否要求真实命令进程和真实数据库故障注入作为阻断验收，还是允许注入式 readiness 测试？当前实现路径已修复，但证据边界需要明确。

## 结论

当前 readiness recovery、错误 reason、nullable protocol/detail 映射、重复字段清理、gateway scalar 展平、transport/TLS/bufconn 主要代码问题已修复，现无 Critical。剩余 I1 是 brief V17 要求的独立服务健康恢复端到端验证缺口：代码路径已有有界 refresh 和单测，但尚未以命令 listener、schema/account 依赖和 health client 证明故障→恢复。若 V17 是本任务的必需验收证据，补齐该测试后再标记 Task 3 完整通过；若另行安排 V17 验收，当前代码审阅无阻塞实现缺陷。
