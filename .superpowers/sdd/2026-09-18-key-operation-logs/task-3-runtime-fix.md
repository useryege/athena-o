# Task 3 runtime/transport 修复记录

## 变更范围

- `internal/operationlog/store`：把查询 readiness 与 projector processing error 分离；投影错误保留 RuntimeStatus 的 `ERROR` 和错误码，但不让仍可读的旧快照变为不可读；暂时错误退避固定为 1、2、4、8、16、30 秒并封顶；schema/account schema reverify 使用独立 2 秒上限，避免停止阶段继承 120 秒 schema 验证预算。
- `cmd/athena-operation-log`：启动和 projector 恢复校验同时核对 operation-log 与 account-state schema；健康检查只依据查询依赖/readiness 和 RuntimeStatus 读取错误，不把 projector `ERROR` 单独当作 NOT_SERVING。
- `internal/operationlog/transport`：Authenticator 对错误 bearer 统一返回 `Unauthenticated`；新增真实 bufconn listener 测试，覆盖 bearer、请求超容量、响应超容量及稳定 reason。
- `internal/operationlog/rpcconfig`：plaintext client/server 均限制 loopback；TLS 仍要求显式 server name 与服务端证书/密钥，secret resolver 的互斥规则保持不变，并用临时 CA 做成功握手测试。

## 验证证据

- `go test ./internal/operationlog/transport ./internal/operationlog/rpcconfig ./internal/operationlog/store ./cmd/athena-operation-log`（在主代理的 proto/server 生成物尚未同步前，`cmd` 编译会被 `server.go` 的 `ProducerID` string→`*NullableString` 不匹配阻塞；不属于本轮文件）
- `go test ./internal/operationlog/access ./internal/operationlog/apiclient ./internal/operationlog/event ./internal/operationlog/ingest ./internal/operationlog/query ./internal/operationlog/record ./internal/operationlog/schema ./internal/operationlog/store ./internal/operationlog/transport ./internal/operationlog/rpcconfig`（本代理范围及其 operationlog 依赖通过）
- `go test -race ./internal/operationlog/transport ./internal/operationlog/rpcconfig ./internal/operationlog/store`
- `go vet ./internal/operationlog/transport ./internal/operationlog/rpcconfig ./internal/operationlog/store ./cmd/athena-operation-log`
- `ATHENA_TEST_PG_ADMIN_DSN=postgres://postgres:athena-operation-log-test@127.0.0.1:56669/postgres?sslmode=disable go test -tags integration ./internal/operationlog/store -run 'Test(ProjectorBackoffSequence|FoldBothArrivalOrdersAndStableHistory|LateCommitAndDualProjectors|LockedPublicationReturnsBusy)' -count=1`
- `git diff --check`

## 未完成项与边界

- 本轮没有修改 proto/query/server/facade；公共 DTO presence、snapshot reason、详情错误映射等复审项仍由主代理负责。
- 没有在本轮启动独立服务进程执行数据库断开→health SERVING/NOT_SERVING→恢复的端到端 smoke；bufconn 已覆盖真实 gRPC listener 的 bearer 与双向消息容量拦截，rpcconfig 测试覆盖临时 CA TLS 握手。
- 没有停止用户或其他任务持有的 PostgreSQL 容器；本轮未启动需收尾的持久服务。
