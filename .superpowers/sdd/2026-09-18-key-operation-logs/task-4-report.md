# Task 4 报告：gRPC 业务事实与 API 生命周期接入

## 实现

- 在 `AthenaServer` 生命周期中接入独立 operation-log producer、内部查询 client、forwarding facade，并在初始化失败和关闭路径释放已创建资源。
- 在 unary/stream gRPC 拦截器中创建唯一 recorder，贯穿认证、授权、模块准入和 facade handler；认证失败不信任 stale credential，panic 保留原始 panic，日志 sink 失败不阻断业务。
- operation-log 查询五个公共方法已注册 gRPC/gateway，管理员查询要求交互式登录凭据；响应设置 `Cache-Control: no-store, private`。
- 37 个目录批准的 gRPC 变更入口已接入 direct typed hooks 或严格 path allowlist observer。写入字段使用白名单，API key 只记录展示 ID，权限快照支持真实 protobuf enum JSON，备注和 token secret 不进入日志。
- typed hooks 覆盖账户、模块准入、通知、Profit Sharing、gateway probe、token blocklist/checkpoint、Trader Sync、wallet；Trader Sync 的 enabled 状态按持久化操作语义记录，不读取被 observation state 替换的公共投影。

## 独立复审

`task-4-final-review.md` 记录了复审结论。复审未发现 Critical 或新的 Important；确认 37 个入口均有 direct hook 或明确的 typed observer 路径，认证成功 marker、交互管理员查询鉴权、资源 ID 严格校验和日志故障隔离有效。

## 验证证据

- `task-4-unit.log`: `go test ./internal/server/... ./internal/operationlog/... ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate`
- `task-4-race.log`: `go test -race ./internal/server/... ./internal/operationlog/...`
- `task-4-vet.log`: `go vet ./internal/server/... ./internal/operationlog/... ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate`
- `task-4-build.log`: `go build ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate`
- `task-4-integration.log`: 真实 PostgreSQL `athena-key-operation-logs-tests` (`127.0.0.1:56669`) 上的 store integration test
- `task-4-diff-check.log`: `git diff --check`

上述命令均退出 0。专项测试还覆盖日志 sink 失败仍执行业务、认证失败不绑定 stale actor、panic 重抛、CreateToken 敏感字段排除、AccountAccess enum JSON 和非 allowlist 嵌套字段不产生证据。

## 已知事实缺口

- `DeleteTelegramBindingAttempt` 当前内部/public response 只有 `Deleted bool`，没有被删除 attempt 的 ID/status；实现记录成功结果和 effect，但不伪造资源或 `attemptStatus`。
- contract blocklist create 的下游应用层根据 source chain/contract 计算 code hash，当前 facade response 为空；实现记录 commit，但没有伪造 code hash/resource。若要完整主资源，需在下游边界扩展内部返回契约。
- gateway probe 的 ACCEPTED 代表进程内 latest-run assignment 和 goroutine scheduling，当前不是跨重启可靠队列；持久队列属于后续 Task 7/8 范围。
- wallet batch 没有持久 batch ID；实现记录规范化 wallet type、请求/确认数量和钱包 ID 列表，不制造虚假的 batch resource。

Task 5–8、V01–V18 全量验证、真实 ATHENA 环境验收和最终人工审查材料尚未完成，不能据此宣称整项任务交付完成。
