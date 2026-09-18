# Task 2 交付报告：持久收件箱与原子投影

## 范围与 ruling

本任务在 `codex/key-operation-logs` 的 `/home/yege/work/athena/.worktrees/key-operation-logs` 完成，基线为 `9c90cc12`。保留 Task 1 的 `event.Event`、`event.Decode`、`event.Canonical`、`ingest.Sink` 与 `ingest.Status` 接口，没有导入 API runtime。

实现包含：

- `internal/operationlog/schema`：`Up(ctx, dsn string) error`、只读 `Verify(ctx, pool *pgxpool.Pool) error`，Goose provider 使用 `operation_log.goose_db_version`，与现有公共 account-state migration lock 共享有界 advisory lock；verify 不创建 schema 或版本表。
- `internal/operationlog/store`：`Open(ctx, dsn) (*Store, error)`、`OpenProducer(ctx, dsn) (*Store, error)`、`Close()`、`Append(ctx, event.Event) error`、`PublishStatus(ctx, ingest.Status) error`、`Project(ctx) (Projection, error)` 与短只读 `Read(ctx, func(pgx.Tx) error) error`。
- `cmd/athena-operation-log-migrate`：显式复用 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`，提供 `up` 与 `verify`，不回退到 legacy DSN。

按 storage-and-query.md 的“FINISH 自包含、跨阶段身份未知值可补全”规则，投影按 `received_at, ingest_id` 折叠已处理事实与当前 pending 事实；冲突只隔离晚到/冲突来源，不让其污染此前可见版本。`source_event_ids` 与 first/last received facts 只包含该版本实际采用的源事件。PostgreSQL 的 `timestamptz` 精度由数据库承担，排序仍使用 receipt timestamp 与 identity ingest id。

## TDD 证据

已有红灯在实现前保留：

- 编译红灯：`.superpowers/sdd/2026-09-18-key-operation-logs/task-2-compile.log`。记录了 sqlc 生成返回 `int64`、`QuarantineParams` 调用不匹配等实际缺口。
- 存储红灯：`.superpowers/sdd/2026-09-18-key-operation-logs/task-2-store-red.log`。记录了幂等、START/FINISH 两种到达顺序、晚提交、双 projector、回滚重启、隔离和状态快照行为失败。
- schema／sqlc 既有证据：`task-2-schema-red.log`、`task-2-schema-green.log`、`task-2-sqlc.log`。

最小修复后，针对同批折叠、历史补全和冲突来源又执行了 focused green 测试，完整输出保存在：

- `.superpowers/sdd/2026-09-18-key-operation-logs/task-2-store-green.log`
- `.superpowers/sdd/2026-09-18-key-operation-logs/task-2-race-green.log`
- `.superpowers/sdd/2026-09-18-key-operation-logs/task-2-unit-green.log`
- `.superpowers/sdd/2026-09-18-key-operation-logs/task-2-sqlc-rerun.log`
- `.superpowers/sdd/2026-09-18-key-operation-logs/task-2-cli-build.log`

## 验证结果

使用真实 PostgreSQL：

```text
ATHENA_TEST_PG_ADMIN_DSN=postgres://postgres:athena-operation-log-test@127.0.0.1:56669/postgres?sslmode=disable
```

通过：

```text
go test -tags=integration ./internal/operationlog/store ./internal/operationlog/schema ./cmd/athena-operation-log-migrate -v
go test -race -tags=integration ./internal/operationlog/store ./internal/operationlog/schema
go test ./internal/operationlog/... ./cmd/athena-operation-log-migrate
go build ./cmd/athena-operation-log-migrate
make sqlc-local
```

集成覆盖了空库只读 verify、公共 catalog/version 不变、共享 migration lock、event+delivery 原子入箱、eventId 与 operation+phase hash 幂等/冲突、START-first 与 FINISH-first、晚提交低 ingest_id、双 projector publication lock、同批单次折叠、历史可见区间、冲突/非法事件隔离、未来 pending 不提前折叠、发布触发器回滚后重启积压、单调 producer snapshot、只读查询事务及 producer 四连接池恢复。

另执行过 `go test ./...`。该命令在无相关失败输出的情况下长时间停留于仓库中其他长时测试包，按任务边界中止；未把 full suite 记为通过，亦未修改 unrelated package。

## 生成与工作区边界

`make sqlc-local` 生成 `internal/operationlog/store/sqlc`，没有手改生成物；SQL 输入稳定后生成差异仅为本任务 operation-log 查询代码。Task 1、父代理持有的 docs 变更及其他 worktree 保持不改；提交只包含 Task 2 的 schema/store/migration/CLI/sqlc/report 文件。

## Remaining concerns

1. Task 3 尚未实现稳定查询、cursor/signature、内部 gRPC/API transport；本任务的 `Read` 仅提供短只读事务，不是管理员查询协议。
2. Projector loop 的周期、退避和服务生命周期由 Task 3 runtime 消费；`Project` 单次调用已限制为 100 条与 2 秒事务。
3. 共享 PostgreSQL 仍是 account-state 与 operation-log 的共同故障域；本任务不声称数据库隔离，也不停止已有测试容器。
4. 仓库级 `go test ./...` 仍需由父任务按其完整验证计划处理；本报告只将 focused/unit/race/真实 PostgreSQL 证据记为通过。
