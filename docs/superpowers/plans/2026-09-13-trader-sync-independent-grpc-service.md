# Trader Sync 独立 gRPC 服务实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Trader Sync 完整迁出 API 进程，使其能够独立构建、运行、验证及停止，并保持现有授权、同库事务和公开接口语义。

**Architecture:** API 通过内部 gRPC 调用 Trader Sync，Notification 独立运行；三个进程各自持有连接池，连接同一权威 account-state schema。Trader Sync runtime 独占采集 session，以持久 generation 保护全部自身写事务。局部运行器按开发实例准备最小基础设施，默认使用实例自己的持久数据库。

**Tech Stack:** 仓库当前 Go 工具链、gRPC/protobuf/gogo 生成链、pgx/PostgreSQL、sqlc、Docker、Make；已有 React/Playwright 消费者验收。

**Spec:** [已确认的服务边界](../specs/2026-09-13-trader-sync-service-boundaries-design.md)、[已确认的构建与运行](../specs/2026-09-13-trader-sync-local-runtime-design.md)、[已确认的字段契约](../specs/2026-09-13-trader-sync-grpc-contract-design.md)。执行者必须同时阅读三份文档；本计划没有修改前两份已确认决定。

**状态：**2026-09-13 用户已确认按现有方案开始实现；本计划与内部字段契约进入实施阶段。任务1–9及11完成并通过独立审阅；任务10与12的最终全栈验证/文档/审阅正在完成。任务勾选与验收报告记录实际进度，未完成项不视为已实现。

## 全局约束

- 遵守根 `AGENTS.md`、[服务开发规范 SDS-R1 至 SDS-R8](../../developer-guide/service-development-standards.md)、相关需求和长期设计。当前共享 checkout 有其他任务的未提交修改；实施时先核对并在 `.worktrees/` 隔离执行，准确带入本任务已确认文档及必要前置变更，不重置或混入其他任务。
- 业务决策固定为单活采集、重启恢复并展示中断、不历史补查；不实现多副本高可用或无中断滚动升级。
- API 不持有 Trader Sync runtime；Notification 不调用它来完成摘要/许可。账户撤权、活动和投递入队、通知许可分别保持现有 caller-owned transaction。
- `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` 是唯一同库定位配置；删除旧名称的读取和注入，无默认 localhost、兼容别名、旧同进程 fallback。
- Trader Sync 默认监听 `127.0.0.1:8122`；二进制默认 TLS，runner 显式本机明文，生产 TLS。默认 RPC 读/状态 5 秒、解析/写 15 秒；默认总停止预算 30 秒；初始 TS ready 60 秒、数据库等待 120 秒、迁移/验证各次调用总预算 120 秒。
- 公共 16 个 RPC、消息字段号、JSON 字段、精确数值、presence、游标及现有 UI 行为保持。内部 protobuf 强类型独立声明，字段契约是逐项实现依据。
- 配置按进程白名单注入；当前 HTTP/WSS、cursor key、`ATHENA_URL=http://localhost:4000` 值保留。内部 token 与 cursor key 独立，不记录到日志/state/命令行。
- SQL 稳定批次完成后运行一次 `make sqlc-local`；proto 源完成后运行 `make protogen` 并检查实际生成消费者。正常独立 build 不触发生成、UI 构建或聚合命令构建。
- 测试按 TDD 执行。每项先写能失败的行为测试，再实现、通过定向验证并审阅。本文测试代码为关键断言，文件内补齐正常 import；额外场景表同样属于验收要求。
- 每个任务结束只暂存该任务文件/已核对的 hunks，运行 `git diff --cached --check` 后做一个聚焦本地提交；不使用 `git add .`。计划不授权推送、发布或生产部署。
- 最终真实验收必须准备正确 checkout 的环境、保留会话和日志，失败要排查重验；受控 fixture 不替代真实入口验收。所有必需实现和验证完成后才按 AGENTS 发送一次完成邮件。

## 文件职责与执行顺序

| 任务 | 主要文件/目录 | 独立交付与依赖 |
| --- | --- | --- |
| 1 | `internal/tradersync/apiclient/trader_sync.proto` | 完整内部字段契约及生成物；无运行依赖 |
| 2 | `internal/tradersync/rpcconfig/`、`apiclient/client.go`、`transport/` | 内部鉴权、TLS、deadline 与领域映射；依赖 1 |
| 3 | account-state migration、TS `store/runtime_*.go` | 持久 runtime 所有权和事务 guard；独立于 1–2 |
| 4 | TS resolver/subscriptions/store/collector | 全部自身写入接入 guard，保留外部事务；依赖 3 |
| 5 | `internal/accountstate/schema/`、独立迁移命令 | 统一 DSN、显式 up 与只读 verify；依赖 3 的 schema |
| 6 | TS `runtime.go`、独立业务 mains | 独立资源生命周期和健康探测；依赖 2、4、5 |
| 7 | `internal/server/tradersync/`、API/Notification 组合 | API facade 完整切换与三池同库；依赖 2、4、6 |
| 8 | `internal/devruntime/`、本地运行命令 | 实例身份、资源记录、受控停止；可在 5–7 期间独立实现 |
| 9 | devruntime 服务注册/数据库/seed、Make | 可用的局部运行闭环；依赖 5–8 |
| 10 | 旧全栈 runner、Procfile | 全栈采用相同归属协议，消除误杀；依赖 8–9 |
| 11 | 独立 Dockerfile、Compose、部署脚本 | 独立镜像、TLS 与 schema 变更顺序；依赖 5–7 |
| 12 | acceptance、browser harness、长期文档 | 两跳、隔离故障、真实运行证据及交付；依赖全部前项 |

1–2、3–4、8 是可并行的开发分支；生成步骤、共享文件修改和最终集成串行。每项有自己的审阅点，最终作为一个完整边界改造交付，不以保留旧路径解决阶段性编译问题。

### 任务 1：落下完整内部 proto 和存在性测试

**文件：**新增 `internal/tradersync/apiclient/trader_sync.proto`、`wire_test.go`；生成同目录 `trader_sync.pb.go`。检查 `hack/update-codegen.sh` 对新目录的输出；公开 proto 与 `pkg/apis/application/v1alpha1/trader_sync_types.go` 作为只读对照。

**接口：**产出 package `tradersync.internal.v1` 的 `TraderSyncServiceClient`/`TraderSyncServiceServer`，方法和消息严格使用字段契约中的 16 行 RPC、34 个资源 DTO、215 个资源字段及基础包装消息。后续 Go 包统一别名 `trpc`。

- [x] 在 `wire_test.go` 写包装消息的真实二进制往返测试；把大整数、nil/false/空串、列表消息各做一例。关键测试：

```go
func TestWirePreservesFalseAndMaxRevision(t *testing.T) {
    in := &Subscription{Revision: math.MaxUint64,
        PausedAt: &StringValue{Value: ""}}
    raw, err := proto.Marshal(in)
    require.NoError(t, err)
    var out Subscription
    require.NoError(t, proto.Unmarshal(raw, &out))
    require.Equal(t, uint64(math.MaxUint64), out.Revision)
    require.NotNil(t, out.PausedAt)
    require.Empty(t, out.PausedAt.Value)
    require.Nil(t, out.CancelledAt)
    raw, err = proto.Marshal(&BoolField{Value: &BoolValue{Value: false}})
    require.NoError(t, err)
    var flag BoolField
    require.NoError(t, proto.Unmarshal(raw, &flag))
    require.NotNil(t, flag.Value)
    require.False(t, flag.Value.Value)
}
```

- [x] 运行 `go test ./internal/tradersync/apiclient -run TestWire -count=1`；预期因新消息未定义而失败。
- [x] 按字段契约创建完整 `.proto`。使用包装消息表达可选标量，固定 Actor 第 1 字段、请求业务字段从 2 起；不导入公开 DTO 或 HTTP annotations。例如：

```proto
message PauseSubscriptionRequest {
  Actor actor = 1;
  string subscription_id = 2;
  uint64 expected_revision = 3;
  string request_id = 4;
}
message PauseSubscriptionResponse { Subscription subscription = 1; }
```

- [x] 执行 `make protogen`，检查新消息/服务及所有生成差异；新内部服务不产生公开路由。若通用脚本尝试生成空 Swagger，按“是否有 HTTP annotations”跳过该输出，不将内部协议加入公开 Swagger。
- [x] 重跑 wire 测试；预期全部通过。检查 16 个方法、字段号/消息目录与设计逐项一致，禁止 JSON bytes、double 金额及 proto3 optional scalar。
- [x] 审阅生成物和测试后提交：`feat(trader-sync): define internal grpc contract`。

### 任务 2：内部服务适配、配置、认证与调用预算

**文件：**新增 `internal/tradersync/rpcconfig/{config,secrets,tls}.go`、`config_test.go`；新增 `internal/tradersync/apiclient/client.go`、`client_test.go`；新增 `internal/tradersync/transport/{server,auth,mapping,errors}.go`、`auth_test.go`、`mapping_test.go`。将原 `internal/server/tradersync/tradersync.go` 中领域→展示的逻辑移入 transport，公开层切换在任务 7 完成。

**接口：**rpcconfig 只依赖标准库/gRPC，不导入 TS root、API 或其他服务 runtime。以下是跨任务固定接口；字段含义和环境变量严格对应运行规格第 4 节。

```go
// package rpcconfig
type Server struct {
    ListenAddress, Transport, Token, CertFile, KeyFile string
    MaxMessageBytes int
}
type Client struct {
    Address, Transport, Token, CAFile, ServerName string
    MaxMessageBytes int
}
func LoadServer(lookup func(string) (string, bool)) (Server, error)
func LoadClient(lookup func(string) (string, bool)) (Client, error)
func MethodBudget(fullMethod string) (time.Duration, error)
func ResolveSecret(lookup func(string) (string, bool), name string) (string, error)

// package apiclient；Close 只关闭本客户端连接，构造不等待远程在线。
func NewClient(cfg rpcconfig.Client) (TraderSyncServiceClient, io.Closer, error)
func NewUnavailableClient(reason string) TraderSyncServiceClient

// package transport；Actor 在内部认证后校验，领域服务仍复核数据库权限。
func NewServer(service *tradersync.Service) *Server
func NewUnaryInterceptor(token string, ready func() bool) grpc.UnaryServerInterceptor
func ValidateActor(actor *trpc.Actor) error
```

- [x] 写配置和 Actor 失败测试，使用 `t.TempDir()` 凭据文件覆盖 `_FILE`/直接值互斥、末尾单个换行、空白、短 token、非法地址；合法 UUID 使用项目现有规范化规则。关键断言：

```go
func TestActorRejectsUnknownRealm(t *testing.T) {
    actor := &trpc.Actor{AccountId: "b7b774b3-dcb5-4bf4-bc64-b0c523d263d7",
        Realm: trpc.ApplicationRealm(99)}
    err := ValidateActor(actor)
    require.Error(t, err)
    s := status.Convert(err)
    found := false
    for _, detail := range s.Details() {
        if info, ok := detail.(*errdetails.ErrorInfo); ok {
            found = info.Domain == "tradersync.internal.v1" && info.Reason == "ACTOR_INVALID"
        }
    }
    require.True(t, found)
}
func TestMethodBudgetsAreExplicit(t *testing.T) {
    read, err := rpcconfig.MethodBudget("/tradersync.internal.v1.TraderSyncService/GetActivity")
    require.NoError(t, err)
    write, err := rpcconfig.MethodBudget("/tradersync.internal.v1.TraderSyncService/CreateSubscription")
    require.NoError(t, err)
    require.Equal(t, 5*time.Second, read)
    require.Equal(t, 15*time.Second, write)
}
```

- [x] 运行 `go test ./internal/tradersync/rpcconfig ./internal/tradersync/apiclient ./internal/tradersync/transport -count=1`；预期新接口缺失或认证断言失败。
- [x] 实现 loader。默认最大消息 `200*1024*1024`，正整数 MB 溢出拒绝；默认 TLS，loopback-insecure 只接受确实为本机的目标/监听地址，通配符和非 loopback 拒绝。token 至少 32 字节且不含空白；服务组合处同时验证 token 不等于 cursor key。TLS 不跳过验证；client 不读取 HTTP/WSS/HMAC。
- [x] 实现认证、Actor 格式校验和 deadline。显式 allowlist 16 方法；无 token、错 token、坏 Actor 分别使用契约的 ErrorInfo。标准 health 不要求 Actor/token，但按相同 TLS 边界暴露；业务 RPC 在 runtime 未就绪时 Unavailable。常量时间比较 token，不能把原值写入错误。

```go
budget, err := rpcconfig.MethodBudget(info.FullMethod)
if err != nil { return nil, status.Error(codes.Unimplemented, "unknown Trader Sync method") }
bounded, cancel := context.WithTimeout(ctx, budget)
defer cancel()
return handler(bounded, req)
```

`WithTimeout` 会尊重较早父 deadline。client 同样设置上限，关闭 gRPC 配置自动重试，不添加写重试拦截器；context 取消不转换成“事务已回滚”。

- [x] 实现全部 server 方法和领域→内部映射。把 `publicResolved/publicSubscription/publicActivity` 的默认区间、invalid_bool/time evidence、观察状态、价格精度逻辑放到 service 一侧；沿用现有 parse/filter 校验和 Service 方法。transport 不导入 `internal/server` 或公开 v1alpha1。
- [x] 增加真实 bufconn/TLS 测试：父 50ms deadline 不变长；请求取消到达 handler；坏 CA/名字失败；TLS 默认不会退回明文；错 token 错 Actor 的 reason；合法但无权 Actor 的 PermissionDenied；配置失败的 unavailable client 对全部 16 方法返回 Unavailable。`NewUnavailableClient` 用统一返回错误的 `grpc.ClientConnInterface`（实现 `Invoke` 和 `NewStream`）包装生成客户端，避免手写 16 份错误实现。
- [x] 重跑本任务 Go 测试，预期通过；提交 `feat(trader-sync): add authenticated internal grpc transport`。

### 任务 3：持久 runtime 所有权与事务 guard

**文件：**新增 `internal/accountstate/store/migrations/000002_trader_sync_runtime_control.sql`、`internal/tradersync/store/queries/runtime.sql`、`internal/tradersync/store/runtime_session.go`、`internal/tradersync/store/runtime_gate.go`、`internal/tradersync/store/runtime_integration_test.go`；重构 `internal/tradersync/store/collector_session.go`，同步 `internal/tradersync/collector.go` 和现有 TS/store integration tests 的 session 类型/方法引用，保持定向测试可编译。检查 `sqlc.yaml` 的三个同库 target 及生成目录。保留现有 `internal/accountstate/txgate/gate.go` 的通用语义。

**接口：**现有 CollectorSession 演进为 RuntimeSession，移动并清理旧 Acquire/Close 路径，不保留兼容别名。相同 collector advisory lock 继续承担单活，不新增第二套 lease。

```go
type RuntimeToken struct { OwnerID uuid.UUID; Generation uint64 }
func (s *SQLStore) AcquireRuntimeSession(ctx context.Context) (*RuntimeSession, error)
func (s *RuntimeSession) RuntimeToken() RuntimeToken
func (s *RuntimeSession) CollectorToken() uint64
func (s *RuntimeSession) Check(ctx context.Context) error
func (s *RuntimeSession) CloseAfterWorkers(ctx context.Context) error
type RuntimeWriteGate struct { pool *pgxpool.Pool; token RuntimeToken }
func NewRuntimeWriteGate(pool *pgxpool.Pool, token RuntimeToken) (*RuntimeWriteGate, error)
func (g *RuntimeWriteGate) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
func (g *RuntimeWriteGate) CheckTx(ctx context.Context, tx pgx.Tx) error
var ErrRuntimeFenced = errors.New("Trader Sync runtime no longer owns the database")
```

`RuntimeSession` 是本任务新增的封装类型，私有字段持有专用连接、两个 token、互斥量及原 pending 恢复状态；调用者只使用上述方法，不能取得或释放专用连接。

- [x] 新增真实 PostgreSQL 测试：沿用 `pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)`。先验证带错 generation 的事务拒绝，且没有向业务表写入：

```go
func TestRuntimeRejectsStaleGeneration(t *testing.T) {
    db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
    ctx := context.Background()
    s := NewSQLStore(db.Pool)
    owner, err := s.AcquireRuntimeSession(ctx)
    require.NoError(t, err)
    t.Cleanup(func() { require.NoError(t, owner.CloseAfterWorkers(ctx)) })
    token := owner.RuntimeToken()
    token.Generation++
    gate, err := NewRuntimeWriteGate(db.Pool, token)
    require.NoError(t, err)
    tx, err := gate.BeginTx(ctx, pgx.TxOptions{})
    require.ErrorIs(t, err, ErrRuntimeFenced)
    require.Nil(t, tx)
}
```

- [x] 运行 `go test -tags=integration ./internal/tradersync/store -run TestRuntime -count=1`，预期缺少 runtime 实现而失败。集成测试必须有明确的 `ATHENA_TEST_PG_ADMIN_DSN`，沿用 pgtest 随机数据库隔离，不拿实例业务库做测试清理。
- [x] 增加 singleton 控制行及查询，现有初始化文件不重编号。schema 核心如下；接替 SQL 检查 bigint 上界，不允许回绕：

```sql
CREATE TABLE trader_sync_runtime_control (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    owner_id uuid,
    generation bigint NOT NULL DEFAULT 0 CHECK (generation >= 0)
);
INSERT INTO trader_sync_runtime_control(singleton) VALUES(true);
-- name: CheckRuntimeWrite :one
SELECT owner_id, generation FROM trader_sync_runtime_control
WHERE singleton AND owner_id = $1 AND generation = $2 FOR SHARE;
-- name: LockRuntimeControl :one
SELECT owner_id, generation FROM trader_sync_runtime_control WHERE singleton FOR UPDATE;
```

- [x] 实现 `BeginTx` 装饰器：先底层 BeginTx，再 `CheckTx`，失败时有界 rollback 并返回 ErrRuntimeFenced；成功返回同一个 tx，让 runtime share 保持到提交/回滚。传给既有 `txgate.WithAccountTx(ctx, gate, owner, fn)` 即可保证 runtime 锁先于 account gate，不需要改写通用 gate。
- [x] 接替顺序固定：拿专用 session advisory lock → 同一 tx 锁 runtime FOR UPDATE → collector control → 结束旧 epoch、推进两个 token → commit。逐账户 pending/baseline 恢复在 commit 之后。关停仅匹配当前 owner+generation 才修改控制行，stale Close 不清除新 owner；无法确认 unlock 时关闭物理连接，不把锁连接放回池。
- [x] SQL 全部确定后执行 `make sqlc-local`，检查 accountstate、notification、tradersync 三处生成产物；增加竞争测试：A 的写 tx 已持有 share，通过 `pg_terminate_backend` 断开 **A 专用 owner 连接** 后 B 才开始接替，B 等 A 旧 tx 完成；B commit 后 A 新写失败。不能在 A 仍持有 advisory lock 时把 B 的 try-lock 失败误当作 share 等待。
- [x] 验证第二实例在工作前失败、旧 tx 可完成、stale Close 不影响 B、失败接替不留下半推进 token；测试通过后提交 `feat(trader-sync): fence runtime writes with durable ownership`。

### 任务 4：覆盖全部写入口，固定外部事务适配器

**文件：**修改 `internal/tradersync/target_resolver.go`、`subscriptions.go`、`collector.go`、`service.go`；修改 `internal/tradersync/store/{sql_store,intake,collector_session,baselines,finality,activities,metadata,revocation}.go`；新增 `internal/tradersync/store/runtime_write_paths_integration_test.go`、`internal/tradersync/store/access_adapter.go`；扩展现有 directory/baselines/activities/metadata 集成测试及 `internal/accountstate/store/access_integration_test.go`、`internal/notification/store/summary_permit_integration_test.go`。

**接口：**新增 `NewRuntimeSQLStore(pool *pgxpool.Pool, gate *RuntimeWriteGate) (*SQLStore,error)`；既有 `NewSQLStore(pool)` 保留给读取/Notification 窄适配器，调用 TS 自身写入会返回新增 `ErrRuntimeRequired`，不能自动无 guard 写入。TargetResolver 的事务输入由具体 pool 改为已有 `txgate.Beginner`；SubscriptionService 已接收 Beginner，生产组合直接改传 gate。普通读取仍用 pool。

新增 `NewAccessRevocationAdapter() *AccessRevocationAdapter`，方法为 `RevokeTx(ctx context.Context, tx pgx.Tx, ownerID, reason string) error` 和 `ApplyAccessChangeTx(ctx context.Context, tx pgx.Tx, ownerID string, previous, next accountaccess.Access) error`。从现有 revocation.go 原样迁出，仅使用调用方的 tx，不持有 pool、RuntimeSession 或后台任务。Collector 的自身事务在本任务加 guard；其 `Run(ctx context.Context, owner *store.RuntimeSession) error` 借用签名与 Acquire/Close 的最终迁移在任务 6 随新的组合根一起切换，避免新增临时生命周期实现。

- [x] 用真实 PG 加失败测试，逐一调用下表的入口；测试在 A token 被 B 替换后运行，断言返回 ErrRuntimeFenced 且相应表行/游标保持不变。测试先失败于现有无 guard 写入。

| 入口 | 必须使用的事务起点/顺序 |
| --- | --- |
| TargetResolver 令牌保存；Create/Change/Note | `txgate.WithAccountTx(ctx, gate, owner, fn)` |
| intake、finality、projection evidence | `gate.BeginTx` → 现有 wallet/source/collector 锁 |
| baseline 注册/持久化/恢复/结束 | TS 发起事务时用 gate；借用同一个已 guarded tx 的方法不重开事务 |
| collector epoch/checkpoint/结束 | 专用连接上的 tx 先 `gate.CheckTx`，再 collector control |
| activity + eligibility + Notification enqueue | gate → account → 原 source/membership 锁 → 同 tx 入队 |
| `CompleteProjectionMetadata` 调用 `SetProjectionMetadataComplete` | 将当前 `q.New(pool)` autocommit 改成 gate 事务 |
| `SaveMetadata` | gate → 原 metadata/source 锁 |
| directory admission TX1、fetch/settlement TX2 | 两次 Begin 都使用 gate，share 保持各自 tx 全程 |

- [x] 写直接写入的具体回归；以下复用现有 `activities_integration_test.go` 的 `activityFixture` 创建完整 source：

```go
func TestRuntimeWriteRejectsOldProjectionMetadata(t *testing.T) {
    db := pgtest.New(t, migrations.FS, migrations.Dir)
    ctx := context.Background()
    account, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx,
        accountcredentials.DevelopmentRoleMember)
    require.NoError(t, err)
    source := activityFixture(t, db.Pool, account.ID, 1)
    base := NewSQLStore(db.Pool)
    a, err := base.AcquireRuntimeSession(ctx)
    require.NoError(t, err)
    t.Cleanup(func() { require.NoError(t, a.CloseAfterWorkers(ctx)) })
    gate, err := NewRuntimeWriteGate(db.Pool, a.RuntimeToken())
    require.NoError(t, err)
    stale, err := NewRuntimeSQLStore(db.Pool, gate)
    require.NoError(t, err)
    require.NoError(t, a.CloseAfterWorkers(ctx))
    b, err := base.AcquireRuntimeSession(ctx)
    require.NoError(t, err)
    t.Cleanup(func() { require.NoError(t, b.CloseAfterWorkers(ctx)) })
    err = stale.CompleteProjectionMetadata(ctx, source.Candidate.SourceID, true)
    require.ErrorIs(t, err, ErrRuntimeFenced)
    var complete bool
    require.NoError(t, db.Pool.QueryRow(ctx,
        "SELECT metadata_complete FROM trader_sync_source_records WHERE id=$1",
        source.Candidate.SourceID).Scan(&complete))
    require.False(t, complete)
}
```

上表其他实际方法沿用各 integration 文件内完整 fixture，断言 token 替换前后原记录的游标、baseline、raw/activity 行数；不通过仅搜索 `BeginTx` 宣称所有写路径安全。新增 store 测试 fixture 统一创建 runtime session/gate，禁止为了让旧测试通过而恢复无 guard 生产写路径。

- [x] 运行 `go test -tags=integration ./internal/tradersync/store ./internal/tradersync -run 'TestRuntimeWrite|TestDirectory|TestMetadata|TestBaseline' -count=1`，确认新测试在对应入口没有 guard 时失败。
- [x] 将上表入口接入 gate，并用 `rg -n 'BeginTx|\.Begin\(|q.New\(s.pool\)|WithAccountTx' internal/tradersync` 审核剩余写点。保留只读查询和 caller-owned 外部事务，不给 Notification summary/permit、API revoke 增加 runtime 锁。

```go
func (s *SQLStore) CompleteProjectionMetadata(ctx context.Context, id int64, complete bool) error {
    if s.runtimeGate == nil { return ErrRuntimeRequired }
    tx, err := s.runtimeGate.BeginTx(ctx, pgx.TxOptions{})
    if err != nil { return err }
    defer func() {
        cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
        defer cancel()
        _ = tx.Rollback(cleanup)
    }()
    if err := q.New(tx).SetProjectionMetadataComplete(ctx,
        q.SetProjectionMetadataCompleteParams{ID: id, MetadataComplete: complete}); err != nil {
        return err
    }
    return tx.Commit(ctx)
}
```

总体强制退出预算由任务 6 保证，rollback/cleanup 使用有界上下文，不能让网络断开变成无限等待。

- [x] Directory 保持现有两阶段协议：TX1 开始并提交 admission；TX2 从开始就 guard，保留目录行锁、原 5 秒来源预算和有界 detached settlement。不得提前释放 share、另开无 guard cleanup 或扩充一个新的第三阶段。扩展现有 `directory_admission_integration_test.go` 的未知提交结果、deadline、连接丢失场景；分别安排 TX1→TX2 间接替与 TX2 HTTP 期间 owner 连接丢失，断言 B 等待已开始 TX2，B commit 后 A 不能再次落盘。
- [x] 增加三个离线事务断言：无 TS runtime 时 API 撤权仍原子关闭 interval/baseline/subscription/membership/未许可 delivery；无 Notification runtime 时 TS 活动和 delivery 同成同败；无 TS runtime 时 Notification summary freeze/permit/result 可运行。再覆盖已许可请求实际结果可写、unknown 不重发、再授权不复活旧资格；登录/API Key 禁用不误当产品撤权。
- [x] 跑本项定向测试及 `go test -tags=integration ./internal/accountstate/store ./internal/notification/store -count=1`；审阅锁顺序后提交 `refactor(trader-sync): guard owned writes and retain atomic adapters`。

### 任务 5：统一同库配置与独立 schema 工具

**文件：**新增 `internal/accountstate/schema/{config,schema}.go`、`schema_test.go`、`schema_integration_test.go`、`contract.json`、`internal/accountstate/schema/catalog/catalog.go`、`catalog_test.go`；新增 `cmd/athena-account-state-migrate/main.go`、`commands/command.go`、`commands/command_test.go`；新增 `tools/account-state-schema-contract/main.go`。修改 `internal/accountstate/store/sql_store.go`、`internal/notification/store/sql_store.go`、`internal/migration/modules.go`、`cmd/athena-migrate/commands/athena_migrate.go`、`util/db/postgres/postgres.go` 及其锁测试、Makefile 和当前实际环境模板。

**接口：**schema 只导入权威 migration FS、通用 pgx/goose，不导入全模块注册表或 accountstate store，以免 store→schema→store 环。API、TS、Notification 调用同一 `ConnectVerified`，各次调用返回独立池。

```go
// package schema
const DSNEnv = "ATHENA_ACCOUNT_STATE_POSTGRES_DSN"
const MigrationLockName = postgres.MigrationLockName
func LoadDSN(lookup func(string) (string, bool)) (string, error)
func Up(ctx context.Context, dsn string) error
func Verify(ctx context.Context, pool *pgxpool.Pool) error
func ConnectVerified(ctx context.Context, dsn string) (*pgxpool.Pool, error)
// CLI：athena-account-state-migrate up|verify --timeout=120s
```

- [x] 写只给旧变量时仍失败的配置测试，以及 readonly verify 不创建版本表的 PG 测试：

```go
func TestLoadDSNRejectsLegacyOnly(t *testing.T) {
    lookup := func(name string) (string, bool) {
        if name == "ATHENA_SERVER_POSTGRES_DSN" { return "postgres://legacy/db", true }
        return "", false
    }
    _, err := LoadDSN(lookup)
    require.Error(t, err)
}
func TestVerifyDoesNotInitializeDatabase(t *testing.T) {
    db := pgtest.NewUnmigrated(t)
    ctx := context.Background()
    require.Error(t, Verify(ctx, db.Pool))
    var exists bool
    require.NoError(t, db.Pool.QueryRow(ctx,
        "SELECT to_regclass('public.goose_db_version') IS NOT NULL").Scan(&exists))
    require.False(t, exists)
}
```

- [x] 运行 `go test ./internal/accountstate/schema ./cmd/athena-account-state-migrate/commands -count=1` 和带 `-tags=integration` 的 schema 包测试；预期缺新 API 或现有自动建表行为导致失败。
- [x] 在通用 postgres 包增加 `const MigrationLockName = "athena:postgres:schema:v1"`，`Migrate/MigrationStatus` 的 session lock 统一使用它，删除从 DSN host/port 拼接锁名的逻辑。PostgreSQL advisory lock 按数据库隔离，因此不同模块的独立数据库仍互不阻塞，同库所有权威迁移入口、测试 helper 和目录快照工具天然使用同一锁。`schema.Up` 调用原 `postgres.Migrate` 的权威 FS/Dir，连接重试、锁等待、迁移共用传入总 deadline。account-state 聚合迁移入口委托 schema.Up；独立工具不导入聚合注册表。其他数据库迁移保持其自身 scope，不借新入口迁移全系统。
- [x] 实现 readonly verify：`BeginTx(ctx, pgx.TxOptions{AccessMode:pgx.ReadOnly})`，先查已有 goose 记录，再查 `pg_catalog`。比较全部有效 migration 版本，不接受未知更高版本；比较当前权威 schema 的表、列类型/nullability/default、约束、索引及业务触发器。缺失和不匹配返回稳定类别，不修表。

为避免手写一个漏字段的校验表，`contract.json` 保存由干净临时测试库生成的排序目录快照：排除 OID、owner、统计数据、序列当前值和版本表行；包含 `pg_get_constraintdef`、`pg_get_indexdef`、`pg_get_triggerdef` 及相关业务函数定义。共享目录查询放在无 embed 的子包 `catalog`，暴露 `Read(ctx context.Context, tx pgx.Tx) ([]byte,error)`，返回规范排序 JSON，schema.Verify 在只读 tx 内调用。工具 `go run ./tools/account-state-schema-contract` 只依赖 catalog、权威 FS 和通用 PostgreSQL 工具，只读取显式 `ATHENA_TEST_PG_ADMIN_DSN`，创建自己的随机库、应用权威 FS、输出快照并只删除自己的库；因此首次生成不依赖尚不存在的 embed 文件。生产 verify 仅读内嵌快照，绝不创建临时库。任务 3 的 migration 确定后生成一次；新增 migration 时重新生成并审阅差异。
- [x] 增加破坏性 schema fixture 测试（仅随机测试库）：少列、少约束、少索引/trigger、版本匹配但结构不完整、未知版本；全部 verify 失败。用两个等价 DSN 连接同库，A 持固定 advisory lock、B 调 Up 必须等待/超时，释放后成功；不能用两个不同库证明同库串行。
- [x] 三个业务 store 改为 connect+verify，删除 `ConnectAndMigrate` 的 account-state 调用；统一所有实际配置消费者。用 `rg -n 'ATHENA_SERVER_POSTGRES_DSN|account-state|ConnectAndMigrate' internal cmd common Makefile Procfile docker-compose.prod.yml hack docs/developer-guide docs/operator-manual` 列清范围；真实 `.env` 更名保留原 DSN 值，不打印凭据，模板存在才修改，历史 spec 的旧事实不批量替换。
- [x] 验证 `go list -deps ./cmd/athena-account-state-migrate` 不含 `internal/migration`、wallet/indexer/Notification runtime；运行 schema 和两个 store 集成测试，预期通过；提交 `feat(account-state): add explicit migration and read-only verification`。

### 任务 6：独立 runtime、薄入口和健康探测

**文件：**新增 `internal/tradersync/runtime.go`、`runtime_test.go`、`runtime_integration_test.go`；修改 `config.go`、`service.go`、`collector.go`；新增 `cmd/athena-trader-sync/main.go`、`commands/{command,health}.go`、`commands/command_test.go`；新增 `cmd/athena-server/main.go`、`cmd/athena-notification/main.go`。迁出 `internal/server/trader_sync_runtime.go` 的客户端/组件组装，旧文件在任务 7 删除。

**接口：**避免 `tradersync`→transport→`tradersync` 环。根 runtime 不导入 transport，通过命令组合根注入建 RPC server 的函数；runtime 拥有返回的 listener/server/health、pool、session、来源客户端和 workers。

```go
type RuntimeDependencies struct {
    NewRPCServer func(service *Service, ready func() bool) *grpc.Server
}
func NewRuntime(cfg Config, rpcCfg rpcconfig.Server, deps RuntimeDependencies) (*Runtime, error)
func (r *Runtime) Start(ctx context.Context) error
func (r *Runtime) Ready() bool
func (r *Runtime) Wait() error
func (r *Runtime) Shutdown(ctx context.Context) error
```

Config 增加 `AccountStateDSN string`、`ShutdownTimeout time.Duration`；Runtime 是本任务新增的资源 owner 类型，内部字段私有。命令构造 `NewRPCServer`：使用任务 2 的 TLS/限额/options 与 unary interceptor，注册 `transport.NewServer(service)`；标准 gRPC health 由 runtime 注册和更新。

- [x] 写 subprocess 生命周期测试：通过 `go test` helper-process 模式启动包含阻塞 worker 的命令，保存 stdout 日志和退出时间；断言第二个同库 runtime 没有任何 worker 初始化、owner 丢失取消 Collector/Projector/Directory、WSS 断开时 RPC ready 但 collector degraded。关键预算验证使用真实子进程等待：

```go
started := time.Now()
err := child.Wait()
require.Error(t, err)
require.Less(t, time.Since(started), 5*time.Second)
require.Contains(t, logs.String(), "shutdown deadline exceeded")
```

此例 `child` 是在该测试内用 `exec.Command(os.Args[0], "-test.run=TestRuntimeHelperProcess")` 启动的测试自身，`logs` 为其 stdout/stderr 的 bytes.Buffer；helper 仅在测试环境标记下运行，使用 1 秒 ShutdownTimeout 和受 channel 阻塞的 worker。测试 helper 不能成为生产 skip-auth 或任意注入代码入口。
- [x] 运行 `go test ./internal/tradersync ./cmd/athena-trader-sync/commands -run 'TestRuntime|TestHealth' -count=1`，并运行对应 PG 集成测试；预期缺生命周期行为而失败。
- [x] 实现有序 Start：校验配置 → 自建 pool 并 readonly verify → listener/NOT_SERVING → AcquireRuntimeSession 提交 → 建 gate/注入同一个 Collector baseline registrar → 逐账户恢复 → 所有 worker 初始化 → SERVING。资源创建失败按已创建清单逆序收尾；配置错误不先拨来源连接；不能以所有外部来源探通作为 ready 条件。
- [x] 实现 Wait/Shutdown：收到 fatal/owner check 失败立即 not-ready，关闭新业务接入并取消整组；总预算中等待已接受请求和全部 workers，完成持久收尾后才 CloseAfterWorkers/pool/客户端。标准 health 的 service name 使用完整内部服务名。命令主 goroutine 设置总截止 watchdog，超时记录并 `os.Exit(1)`；不能仅返回超时后执行释放 owner 的 defer，而阻塞 worker 继续活着。
- [x] 生命周期日志包含 service、instance/run（直接启动时明确未由实例监管）、runtime generation、collector epoch 和阶段；未知/不可达状态显式呈现，不填零冒充健康。默认不开放 reflection；开发工具使用内部 proto/生成 descriptor。
- [x] 增加顺序断言：owner session 关闭事件必须晚于 worker join；卡死超过预算时由进程退出释放连接。恢复阶段的每次 TS 写入已取得 guard；新 subscription baseline 使用 runtime 的同一个 Collector。来源不可达情况下 List/Pause/Cancel 可达，Create/Resume 保持 pending baseline。
- [x] `health` 子命令单独解析目标/TLS CA/server name/deadline，仅做 gRPC health Check，不加载 DB、HTTP/WSS、cursor/token 或启动业务。容器内目标 `127.0.0.1:8122` 与证书名 `athena-trader-sync` 分别设置，不能用 loopback 名字错误验证证书。
- [x] 独立 main 只调用自己的 command 包；新 TS 不加入根 `cmd/main.go`。运行 `go build -o dist/athena-trader-sync ./cmd/athena-trader-sync`、对应 API/Notification direct main build、`go list -deps ./cmd/athena-trader-sync`。断言 TS 依赖中没有 `internal/server` 或其他业务 command/runtime，允许已确认的 Notification 窄 store。
- [x] 测试和 build 通过后提交 `feat(trader-sync): add standalone runtime and bounded lifecycle`。

### 任务 7：切换公开 facade，移除 API 内的 runtime

**文件：**修改 `internal/server/tradersync/{tradersync.go,tradersync_test.go,contract_test.go}`，新增 `mapping.go`、`errors.go`、`facade_test.go`；修改 `internal/server/athena-server.go`、`cmd/athena-server/commands/athena-server.go`、`internal/notification/service.go`、`cmd/athena-notification/commands/athena_notification.go` 及现有组合测试；删除 `internal/server/trader_sync_runtime.go`。

**接口：**公开包的 `New(client trpc.TraderSyncServiceClient, actors ActorResolver) *Server` 替代 `New(*ts.Service)`；`type ActorResolver func(context.Context) (*trpc.Actor,error)`。新增 `MapInternalError(err error) error`，只负责契约中的错误分类。API opts 接受 client/closer 或 client 配置错误分类，不接受 TS Config/Runtime。API 只关闭自己创建的 client。

- [x] 写 facade 单元和真实 gRPC 错误测试，证明内部 token 故障不会变成公开 401：

```go
func TestInternalAuthenticationFailureBecomesUnavailable(t *testing.T) {
    s, err := status.New(codes.Unauthenticated, "service authentication failed").WithDetails(
        &errdetails.ErrorInfo{Domain: "tradersync.internal.v1", Reason: "SERVICE_AUTH_INVALID"})
    require.NoError(t, err)
    require.Equal(t, codes.Unavailable, status.Code(MapInternalError(s.Err())))
    require.Equal(t, codes.PermissionDenied,
        status.Code(MapInternalError(status.Error(codes.PermissionDenied, "grant required"))))
}
```

用嵌入生成 `UnimplementedTraderSyncServiceServer` 的记录服务逐方法捕获内部请求，再经 bufconn 调用 facade；它记录 Actor、过滤器、page、note 与 request_id，响应填满必需对象。测试 16 方法，不能只测 Resolve/Create。
- [x] 运行 `go test ./internal/server/tradersync -count=1`，预期新构造/503/字段断言失败。
- [x] 实现 facade 请求和结果逐字段映射。使用任务 2 的客户端预算，不解析 HMAC cursor 或计算 evidence/观察状态。所有必需响应对象做契约校验，缺失映射 Unavailable；保留 nil wrapper 与 false/空值，最大 revision 不经 float。
- [x] API Actor resolver 使用 `session.AccountID(ctx)` 和 `credentialMgr.Get(id).ApplicationRealm()`。缺公开身份返回原 Unauthenticated；无 account 的管理员过滤器仍是普通过滤器；数据库中 realm 非法或无法形成可信 Actor 是局部依赖/契约故障。无需为 Bearer/API Key 添加新 header。
- [x] API 创建 nonblocking client；配置错误装 `NewUnavailableClient` 并记录安全原因，继续启动其他模块。删除 `TraderSyncConfig.Validate`、runtime 组装、process worker、`traderSyncDone` 的 Run/Serve select 及 Close 分支。安装 `NewAccessRevocationAdapter().ApplyAccessChangeTx` 到现有账户 store；不再为 hook 创建完整 TS runtime store。
- [x] Notification 保持自己的 pool 和已有 `ConfigureSummaries(pool, siteURL)` 借用校验；不持有 TS RPC client/token、provider 或 cursor 配置。统一 DSN 后用实际数据库身份核对三个不同 pool 指向同库，不能仅比较配置字符串。
- [x] 扩展 API/Notification 组合测试：删/污染 TS HTTP/WSS/HMAC 时 API 和 Notification 仍可启动；内部地址/证书/token 错只影响 TS facade；TS 停止/fatal 时 bootstrap、账户、其他已启动模块及 Notification 均存活。真正公开身份失效仍 401；合法会员/管理员、API Key、无 realm header Bearer、跨 owner、撤权均走真实权威检查。
- [x] 运行 `go test ./internal/server/... ./cmd/athena-server/commands ./cmd/athena-notification/commands -count=1`，并执行对应 auth/tx 集成测试。检查 `rg -n 'traderSyncRuntime|newTraderSyncRuntime|TraderSyncConfig|traderSyncDone' internal/server cmd/athena-server` 应无实现残留；提交 `refactor(api): delegate Trader Sync through internal grpc`。

### 任务 8：实现开发实例记录与资源所有权核心

**文件：**新增 `cmd/athena-local-runtime/main.go`、`internal/devruntime/{config,state,process_linux,docker,runner}.go`、`state_test.go`、`process_linux_test.go`、`runner_test.go`。采用小型 Go 编排器处理 dotenv、JSON 原子状态、PID 和 Docker argv；它不导入业务 command/runtime。Make 和旧 shell 作为薄命令适配，避免两套清理规则。

**接口：**本次目标平台是当前 Linux/WSL 本地环境；其他平台在分配资源前明确报不支持进程身份核验，不用较弱的 PID-only fallback。

```go
type InstanceKey struct { Checkout, Name, Namespace string }
type ProcessIdentity struct {
    PID, PGID int
    StartTicks uint64
    BootID, Exe, RunID string
}
type ResourceRef struct {
    Kind, ID, Name, Namespace, RunID string
    Owned bool
}
type State struct {
    Version int
    Key InstanceKey
    RunID, Phase, DBMode string
    Supervisor ProcessIdentity
    Processes map[string]ProcessIdentity
    Resources []ResourceRef
    Services []string
    Logs map[string]string
    Endpoints map[string]string
    ExitCodes map[string]int
}
func NewInstanceKey(checkout, name string) (InstanceKey,error)
func SameProcess(recorded, current ProcessIdentity) bool
func SaveState(path string, state State) error
func LoadState(path string) (State,error)
func Stop(ctx context.Context, key InstanceKey) error
func Reset(ctx context.Context, key InstanceKey) error
```

`State` 后续增加非秘密 build fingerprint、配置 fingerprint、创建意图和初始化模式；实际密码/token/完整 DSN 单独保存在 `0600` 文件。Namespace 从规范 checkout 路径和名称计算，run UUID 每次变；资源的持久 namespace owner 与当前 run 创建者分别保存，不能因重启换 run ID 就把原 volume 判成外部或自动删除。

- [x] 写真实文件原子化/路径限制、纯身份判断测试；使用注入的 process reader/signal 和 Docker command executor 测试不误杀，生产实现仍读 `/proc`。关键 PID 重用断言：

```go
func TestSameProcessRejectsPIDReuse(t *testing.T) {
    recorded := ProcessIdentity{PID: 42, PGID: 42, StartTicks: 123,
        BootID: "boot-a", Exe: "/repo/dist/athena-trader-sync", RunID: "run-a"}
    current := recorded
    current.StartTicks++
    require.False(t, SameProcess(recorded, current))
    current = recorded
    current.RunID = "run-b"
    require.False(t, SameProcess(recorded, current))
}
```

- [x] 运行 `go test ./internal/devruntime -count=1`，预期新身份/状态逻辑未实现而失败。
- [x] 实现 `.run/instances/<instance>/` 记录，名称只允许 `[A-Za-z0-9][A-Za-z0-9_-]{0,62}`。所有变更取短期 flock；先记录 supervisor 启动意图和自身完整身份，再释放锁做慢操作；stop 能取得锁并看到 starting/stopping，不会被整个前台运行挡住。重复 supervisor 启动核验身份后拒绝。
- [x] 资源创建先持久 intent，Docker create 使用精确 namespace/instance/run/kind labels，创建后 fsync+rename 更新 state。崩溃恢复只用对应 intent+精确 labels 查询，不按名称、端口或泛化 `owner=athena` 接管。命令用 `exec.CommandContext(name,args...)`，dotenv 不 source/eval。
- [x] 进程 spawn 有独立 PGID 和 run 标记，立即保存 PID/start ticks/boot ID/exe；stop 发信号前重复核对完整身份。信号发送与 PID 重用间的竞争使用 Linux pidfd 等可验证的进程句柄保护；进程组成员也必须属于已记录的 supervisor/child 层级，不向已复用组号盲发 KILL。child 退出由 supervisor reap。
- [x] 停止协议只覆盖本实例拥有进程和容器，业务总预算沿用服务声明；失败记录未清理资源，不扩成全局 kill。短期锁协调 stop 请求后释放，允许 supervisor 写退出结果；重复 stop 幂等。reset 要求已停止、没有仍可核实活跃成员、数据库 managed 且精确归属正确。
- [x] 增加故障表测试：相同 checkout 不同实例、两个 worktree、PID/PGID 重用、资源创建后崩溃、意图写入前失败、state 版本未知、signal 权限失败、Docker cleanup 失败。每例断言实际 signal/docker argv 清单和最终 state，而非只断言打印字符串。
- [x] 测试通过后提交 `feat(devruntime): track instance-owned resources and safe shutdown`。

### 任务 9：接上最小依赖、持久数据库与开发命令

**文件：**新增 `internal/devruntime/{registry,environment,postgres,seed}.go`、相应 `_test.go`、`runner_integration_test.go`；新增 `tools/trader-sync-dev/main.go`、`internal/devruntime/seed_integration_test.go`；修改 Makefile；迁移 `hack/trader-sync-local.sh` 的 WSL 默认代理逻辑及 `hack/trader-sync-local_test.sh` 测试，删除 API helper 的原用途。

**接口：**任务 8 的状态/归属基础上产出：

```go
type ServiceSpec struct {
    Name, BuildPackage, Binary string
    Infrastructure, EnvironmentKeys []string
    Args []string
    StartupTimeout, ShutdownTimeout time.Duration
}
type RunOptions struct { Key InstanceKey; Services []string; DBMode, EnvFile string }
func ResolveServices(names []string) ([]ServiceSpec,error)
func Build(ctx context.Context, key InstanceKey, names []string) error
func Run(ctx context.Context, options RunOptions) error
func Status(ctx context.Context, key InstanceKey) (State,error)
func Seed(ctx context.Context, key InstanceKey, service string) error
```

注册表中的 infrastructure 是 PostgreSQL/Redis/MinIO 等基础设施名，不将远程业务服务写成隐式依赖。允许 `trader-sync`、`api-server`、`notification`、`ui` 单独或显式组合；ui 的使用者自己选择 API。其他旧服务只在任务 10 的 full-stack 清单使用，未具备 direct main 者拒绝独立 SERVICE 构建。

- [x] 写选择测试，明确 Notification 和 API 单独运行不自动加入 TS：

```go
func TestResolveNotificationDoesNotStartTraderSync(t *testing.T) {
    specs, err := ResolveServices([]string{"notification"})
    require.NoError(t, err)
    require.Len(t, specs, 1)
    require.Equal(t, "notification", specs[0].Name)
    require.NotContains(t, specs[0].EnvironmentKeys, "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY")
    require.NotContains(t, specs[0].EnvironmentKeys, "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN")
}
```

同时覆盖 unknown/重复名/循环基础设施依赖在资源创建前失败；未定义 proxy、显式空、shell 覆盖 dotenv、每个进程白名单。
- [x] 运行 `go test ./internal/devruntime -count=1`，预期注册表/配置选择断言失败。
- [x] 实现 managed PostgreSQL：只建 `athena` 和必要角色，动态 loopback 端口、持久 volume、正常 fsync 配置；不调用全栈固定数据库初始化脚本。受管 DSN 由资源实际地址生成，覆盖本实例注入值，不能因根 `.env` 有共享 DSN 而选错库。等 PG ready（120 秒）后运行独立 `up/verify`（120 秒）再启动选定业务。
- [x] 实现 external：必须 `DB_MODE=external` 且显式共享 DSN/token；只 verify，不创建/停止/删除 DB、不 DDL、不 seed。已有模式和数据归属不可静默切换：本轮直接拒绝模式变更并提示新实例，不另加热重配置命令。原库 owner 停库影响借用方时 status 如实报告，不建立全局借用 registry。
- [x] 配置处理按 approved spec：ENV_FILE 默认 `.env`，shell wins；本机 TS proxy 未定义时才用现有 WSL gateway 策略，显式空直连；binary/prod 未定义直连。managed 首次无显式 token 才用 crypto/rand 生成 32 字节以上凭据、0600 持久化并复用；内部 token 与 cursor key 相等拒绝。TLS/明文只向相应进程注入。
- [x] Run 构建已选 binary、计算基础设施并集、记录启动意图。TS 若被选先等待其 RPC ready，随后启其他选定进程；若未选则直接启动其余进程。各初始 ready 全通过才标 running；之前失败只回滚本批次新启动的自有进程/容器，保留 volume/log/state。成功后 TS fatal 留 API/Notification 运行并记录退出，不默认重启。
- [x] 接上命令，所有参数按 argv 传入编排器，不用 shell 拼接 SERVICES 的任意命令。Make 的公开界面固定为：

```bash
make build-service SERVICE=trader-sync
make run-service SERVICE=trader-sync INSTANCE=trader-sync
make run-services SERVICES="api-server trader-sync notification" INSTANCE=ts-integration
make runtime-status INSTANCE=trader-sync
make stop-instance INSTANCE=trader-sync
make reset-instance INSTANCE=trader-sync
make seed-service SERVICE=trader-sync INSTANCE=trader-sync
make account-state-migrate
```

`run-service` 未给 INSTANCE 时默认 SERVICE；`run-services` 要求显式 INSTANCE，防止不同组合落入不明默认实例。`account-state-migrate` 调独立工具 up 后 verify，读取显式共享 DSN；external run 不调用该 target。
- [x] seed 工具调用 `accountstate.SQLStore.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)` 和 `DevelopmentRoleAdministrator` 创建完整固定 aggregate；首次 fixture 的 member TS grant 使用现有账户权限事务设置，把 fixture 标识/账户 ID 持久存实例配置。重复 seed 先查已存在账户，不更新其 grant/暂停/撤权结果，不绑定 Telegram 或制造活动。初始化与标识提交失败不能导致下一次重置已有授权：以数据库中固定账户是否存在为准，已有账户只返回 ID。运行前复核 target DB 与 namespace，外部模式和生产镜像无此工具。
- [x] 执行 `go test -tags=integration ./internal/devruntime -count=1`：真实 Docker 临时实例写入→stop→run仍存在；两个实例互不影响；端口冲突不发信号；external verify 失败无 child；借用 stop/reset无 Docker DB操作；启动失败保留日志/volume。测试创建的专属临时实例可按归属清理，不操作用户开发实例。
- [x] 定向测试通过后提交 `feat(devruntime): run selected services with persistent instance databases`。

### 任务 10：让原全栈启停遵守同一归属协议

**文件：**修改 `hack/local-runtime.sh`、`Procfile`、Makefile、`internal/devruntime/registry.go`，新增 `internal/devruntime/fullstack.go`、`fullstack_test.go`；修改 `hack/shell-local_test.sh` 和现有 `hack/trader-sync-local_test.sh`，按实际保留脚本调整测试。

**接口：**新增 `FullStackServices() []string`，显式列出现有 Procfile 的全图；`make run/stop/run-reset` 默认使用当前 checkout 的 `INSTANCE=full-stack`，进入任务 8–9 同一个引擎。旧脚本只做命令参数适配。full-stack 可保留无关服务的聚合 build 差距，但实际受管业务进程必须运行已构建 binary，不能监管 `go run` 包装器。

- [ ] 写两个状态目录/资源集合的测试，调用 full-stack stop/reset 后断言局部 TS 的 PGID、container、volume 没有出现在操作清单。用新 engine 的同一命令执行器收集真实调用参数：

```go
require.NotContains(t, signalledPGIDs, traderSyncPGID)
require.NotContains(t, removedContainerIDs, traderSyncContainerID)
require.NotContains(t, removedVolumeNames, traderSyncVolumeName)
```

这些变量均由测试内创建的两个 State 和注入执行器捕获，不查询或终止机器上的现有未知进程。
- [ ] 运行 `go test ./internal/devruntime -run TestFullStack -count=1` 与现有 shell runtime 测试；预期旧全局清理路径导致隔离断言失败。
- [ ] 将全栈数据库/Redis/MinIO 生命周期接到资源记录。删除按端口、cwd、ATHENA_BINARY_NAME、泛化标签清杀和固定 volume reset 的路径；历史无归属资源出现时提示冲突，不能自动认领。保留用户现有数据，不把第一次新命令执行当作清理旧环境的授权。
- [ ] 明确全栈 account-state up/verify 先于其 API/TS/Notification；其余数据库迁移按原全栈列表显式执行。独立 TS 作为独立进程登记，API 不再经过 trader-sync-local helper。去掉 Goreman 全量 dotenv 给所有进程的行为，逐个服务使用白名单；Procfile 同步为实际清单说明/直接 binary 入口，不保留第二条可绕开归属记录的默认运行路径。
- [ ] 保持全栈原来声明的 API/UI/Redis/avatar 行为；没有把其他业务架构全改成独立服务。测试 full-stack 启动前 schema 失败不会启动消费者；运行中 TS fatal 不结束全图；两个 checkout 的 stop/reset 互不触碰。
- [ ] 运行 `go test ./internal/devruntime -count=1`、`bash hack/shell-local_test.sh`、适用 shellcheck；审阅当前其他任务对 shell 的修改并合并，提交 `refactor(devruntime): unify full-stack resource ownership`。

### 任务 11：独立镜像、TLS 注入与部署消费者

**文件：**新增 `deploy/trader-sync/Dockerfile`、`hack/trader-sync-deploy_test.sh`；修改 Makefile、`docker-compose.prod.yml`、`hack/prod-remote-deploy.sh`、`hack/deploy-scripts_test.sh`、现有生产环境模板及部署操作文档。

**接口：**新增 `make build-service-image SERVICE=trader-sync`，镜像名称由 `TRADER_SYNC_IMAGE` 指定；Compose 服务固定 `athena-trader-sync`，内部地址 `athena-trader-sync:8122`。独立镜像同时装入 `athena-trader-sync` 和独立 account-state migrator；镜像入口默认只执行业务命令，迁移必须显式选择。`athena-account-state-migrate` Compose 工具服务使用该镜像，`ATHENA_IMAGE` 的其他服务不变。

- [x] 写 fake Docker/deploy 命令记录测试：独立镜像被 build/inspect/save/load，所有 Compose 阶段都有 TRADER_SYNC_IMAGE；schema 变更路径出现严格顺序“停止三使用者→确认退出→up→verify→启动”，迁移失败不启动。只替换 TS 且 schema 相同的路径不停止 API/Notification。测试不能实际 SSH 到生产。

```bash
bash hack/trader-sync-deploy_test.sh
bash hack/deploy-scripts_test.sh
```

预期首次因缺独立镜像/顺序仍为先 migrate 后 stop 而失败。测试使用项目已有 fake 命令目录和操作记录文件，断言顺序与 argv，不用在源码中搜索几个词作为部署行为证据。
- [x] Dockerfile 以仓库当前 Go 基础镜像构建 direct main 和 schema tool，运行层仅包含二进制、CA 等所需运行依赖，不使用根 Dockerfile、Node/UI、`make athena-all`。核心 build 指令：

```dockerfile
RUN go build -o /out/athena-trader-sync ./cmd/athena-trader-sync
RUN go build -o /out/athena-account-state-migrate ./cmd/athena-account-state-migrate
```

正式文件保留仓库已有可重现构建参数和非 root 运行用户，不复制 `.env`、seed 工具、Wallet signer 或 Telegram secret。
- [x] Compose 注入共同 DSN；TS listen `0.0.0.0:8122`、API target `athena-trader-sync:8122`、默认 TLS、专用 token 的 `_FILE`、TS cert/key、API/health CA/server name。无 host ports，无 API→TS健康启动依赖、无 TS→API/Notification 依赖；`stop_grace_period: 40s` 留出进程 30 秒总预算。health 单独命令通过 loopback 检查服务名和证书名，不装配业务 runtime。
- [x] 更新远程脚本的镜像生命周期及每处 `docker compose` 配置注入。schema 有变化时必须停止全部同库消费者；由新 schema tool `verify` 判兼容，不能仅比较最新版本号后忽略结构差异。对部署管理范围外的同库消费者，要求操作者在维护前明确协调并提供已停止事实；脚本不伪称能发现全网所有借用者。开发测试若旧 schema 不兼容，只能在明确维护/新实例中处理，不能偷偷 reset。
- [x] 执行本机镜像 build、`docker compose config`（检查结果不输出解析后的 secrets）及 fake 部署测试；启动专属临时容器网络，用临时 CA+正确 SAN 验证独立 TS TLS/health、错误 CA/名字拒绝、无业务凭据仍能运行 health、镜像文件没有 UI/seed。真实环境与临时受控网络的证据分别记录。
- [x] tests/build 通过后提交 `feat(deploy): package standalone Trader Sync with TLS`。本任务不执行生产发布。

### 任务 12：两跳契约、故障隔离与真实本地验收

**文件：**扩展 `internal/server/tradersync/contract_test.go`、`internal/tradersync/acceptance/{acceptance_integration_test.go,ui_harness_integration_test.go}`；新增 `internal/tradersync/acceptance/independent_runtime_integration_test.go`、`hack/trader-sync-independent-acceptance.sh`；新增 `docs/testing/trader-sync-independent-service-acceptance.md`；同步 `docs/developer-guide/running-locally.md`、`docs/operator-manual/makefile-commands.md`、`docs/design/development-runtime/local-runtime-orchestration.md`、`docs/design/trading/trader-sync-activity-alerts.md`、`docs/design/README.md`。UI 只在真实消费契约变化必须修改时调整对应文件，不借此重新设计页面。

**接口：**新增 `make trader-sync-acceptance INSTANCE=ts-acceptance` 调受控运行验收脚本；脚本不启动未知环境或调用全局 reset，调用任务 9 的实例命令并核验所属 checkout/state。UI harness 的公共 facade 改为内部真实 gRPC 客户端，形成“领域/fixture→内部 gRPC→facade→公开 gRPC→生产 gateway JSON”的两次编解码。

- [ ] 先让当前 contractGateway 在新增内部 hop 下运行；断言实际经过内部 server（计数/捕获请求），防止测试误绕回内存 service。以下是必须保留的最终 JSON 断言形状，使用现有测试的 JSON 解码 map：

```go
require.Equal(t, "9007199254740993", activity["id"])
require.Equal(t, "18446744073709551615", subscription["revision"])
require.Equal(t, "0", activity["priceNumerator"])
require.Equal(t, false, verified["value"])
require.Equal(t, "1725148800", curvePoint["t"])
require.NotContains(t, delivery, "authorizedAt")
require.Equal(t, originalCursor, page["nextCursor"])
```

变量从该测试完整 fixture 取得；每个被断言字段在 fixture 中显式赋值。最终 JSON 的空 repeated 编码沿用现有 gateway 实测行为，不把 Go nil slice 当成协议 presence。新增 16 RPC 请求/响应表驱动覆盖及字段目录全覆盖比较。
- [ ] 运行 `go test ./internal/server/tradersync ./internal/tradersync/apiclient ./internal/tradersync/transport -count=1`，确认两跳新增场景在错误映射版本上失败，然后修复最小映射问题并通过。测试包含 note 未传/null/显式空、无效 Bool/Time、六 PnL 区间、同时间重复点、legs 未知/空/部分值、长整数金额、未知时间/消息 ID、空列表和必需对象缺失。
- [ ] 扩展真实 PG 授权/事务链：会员/API Key/Bearer/admin、跨 owner/realm、grant 撤销、旧 cursor owner/filter/page_size 不匹配、写请求提交后丢响应。最后一例在内部 handler 已 commit 后阻断响应，facade 得到超时/取消，再以原 request_id 和相同载荷调用，断言只生成一次订阅/版本/备注更新；不同载荷复用 request_id 仍拒绝。
- [ ] 跑集中后端回归，保存命令、退出码、日志；先准备只供 pgtest 创建随机测试库的 admin DSN：

```bash
go test -tags=integration ./internal/tradersync/acceptance -count=1
go test -race -p 2 -tags=integration ./internal/accountstate/... ./internal/notification/... ./internal/tradersync/... ./internal/server/... ./util/telegram/... ./cmd/athena-trader-sync/commands ./cmd/athena-notification/commands -count=1
go test -race ./internal/devruntime/... -count=1
```

覆盖既有 finality/removed/version retry/summary/unknown 规则；100 distinct 与 10 shared 目标的既有受控容量场景保持。失败按 systematic-debugging 处理；不因较小单测通过略过必要数据库并发验证。
- [ ] 用正确 checkout 的持久会话启动 `make run-service SERVICE=trader-sync INSTANCE=ts-acceptance`，只应出现 TS+该实例 PostgreSQL。先 `seed-service` 获得真实开发 Actor，直接通过内部客户端调用授权查询；记录 DB身份、进程、日志、ready/collector状态。停止并重启该专属验收实例，验证数据保留、中断可见、不发起历史补查。验收脚本通过自身临时 provider/fixture 验证基线、活动和故障，真实配置来源的连接与恢复结果另列，不把 fixture 证明外推为实网 SLO。
- [ ] 运行故障场景表并保存逐项证据：

| 场景 | 必须观察到的结果 |
| --- | --- |
| 两个 runtime 连接同库 | 第二个在任一 worker 写入前拒绝；不释放第一实例资源 |
| WSS 暂时断开 | RPC ready，collector degraded；新基线 pending，重连按新 epoch 恢复并展示中断 |
| owner 连接丢失/卡死 worker | 全 TS workers 取消；旧 generation 不新写；总停止预算内进程实际退出 |
| TS 停止或配置错误 | API bootstrap/账户及 Notification 继续；仅 TS facade 503，用户不被登出 |
| TS 离线时撤权 | 同库事务完整墓碑；重启后旧资格不复活 |
| Notification 离线再恢复 | 活动与普通投递同 tx；恢复消费冻结结果，unknown 不重发 |
| 局部实例与 full-stack 并存 | 双向 stop/reset 只操作自己的测试资源；端口冲突不杀占用者 |
| external 及 schema 错误 | readonly verify；无隐式 DDL/seed/reset，借用 stop 不操作数据库 |
| 创建中失败/超时 | 只收回本次新启动资源，数据和日志保留，能通过精确归属恢复状态 |

测试发送路径使用受控 Telegram fixture。需要向真实 Telegram 账户发送消息时必须有该次明确授权；本计划不额外授权对外发信。
- [ ] 运行镜像/TLS和独立构建证据：TS build 不调用 Node/UI/其他 service build，迁移工具不导入聚合 registry，容器 health 使用正确证书名。检查 `/proc` 和 Docker 实际资源，不能仅依据 runner 声称最小依赖。
- [ ] 按浏览器验收 skill 执行 `make ui-acceptance` 的 `/`、`/athena` 两个隔离入口。真实 smoke 前检查目标地址、仓库/worktree、进程归属；健康现有环境复用，否则从目标仓库按 Node 版本要求启动 `make run` 并保存持久会话/log，再执行：

```bash
make ui-acceptance UI_ACCEPTANCE_MODE=smoke
```

确认前端入口和 member/admin bootstrap 后检查 smoke 报告。smoke 证明真实 shell/bootstrap；Trader Sync 页面/操作链用显式选择的 API+TS+必要 UI 集成环境和两跳 fixture 验收共同记录，不能称一次 shell smoke 覆盖全部业务。保留本次开发服务运行，交付说明停止命令；故障演练只启停自己专属实例。
- [ ] 更新长期设计为真实独立进程/配置/事务/命令图，替换“当前内嵌 API”的旧实现叙述；保留历史任务 spec 原有事实。验收报告逐项写 SDS-R1–R8 映射、命令/退出结果/日志、运行地址和剩余外部限制；未通过的必需项不可记为完成。
- [ ] 执行 verification-before-completion、requesting-code-review，处理审阅发现并只重跑受影响检查；检查生成物与消费者同步、本次 diff whitespace/链接。提交 `test(trader-sync): verify independent service boundaries`。
- [ ] 确认全部代码、配置、文档与必需验收完成后，从仓库根目录发送唯一完成通知，并等待命令结束：

```bash
make notify-task-complete \
  TASK_NOTIFICATION_SUBJECT='任务完成：Trader Sync 独立 gRPC 服务' \
  TASK_NOTIFICATION_BODY='已完成：独立服务、内部契约和实例化本地运行；验证：后端事务、故障隔离、镜像与本地验收通过。'
```

通知失败不改实现结果，最终如实说明；存在必需验收未通过时不发送。最终交付列明实际运行地址、checkout/worktree、会话/进程、日志、报告及停止方式：全栈从同仓库 `make stop`，局部实例 `make stop-instance INSTANCE=<实际实例名>`。

## 计划自审与规格覆盖

以下是计划撰写阶段的自审映射，不是实现测试结果。

| 规格条目 | 实施任务 | 核心验收 |
| --- | --- | --- |
| 服务职责、独立边界，SDS-R1/R3 | 6、7、9、11 | direct main/build 依赖、单服务进程图、镜像 |
| 16 RPC/Actor/服务身份/业务权威，SDS-R2/R4 | 1、2、7、12 | 两跳编解码、realm/owner/grant、内部故障503 |
| generation 接替、所有写事务、目录两阶段 | 3、4、6、12 | 真实 PG 竞争、直接 projection 写入、接替和故障 |
| 同库撤权/活动入队/许可，SDS-R6 | 4、5、7、12 | 进程离线仍原子执行、再授权与 unknown |
| 配置/白名单/WSL/TLS/预算 | 2、5、6、9、11 | 环境优先级、TLS错误、总停止、health独立 |
| 持久开发实例/最小依赖/外部复用 | 8、9 | 数据保留、external readonly、seed幂等 |
| 资源归属/失败回收/旧全栈，SDS-R5 | 8、9、10、12 | 双实例/双checkout、PID重用、精确labels、拒绝误杀 |
| schema准备/固定迁移锁/只读结构验证 | 3、5、9、10、11 | 同库别名并发、空/错误/较新schema、先停消费者 |
| 现有业务和 UI 消费者，SDS-R7/R8 | 7、12 | 精度/presence/游标/重放、既有集成与浏览器 |
| 交付事实、文档及完成通知 | 12 | 真实环境报告、运行会话、只在全部完成后通知 |

自审修订重点：内部包避免 import cycle；利用现有 Beginner 装饰器保证锁顺序；补齐 autocommit projection 写点；PnL 曲线 Unix 秒保留；API 内部认证故障不触发公开 401；外部事务不被 runtime guard 阻断；full-stack 与局部实例使用同一归属规则。执行时采用分任务实现和审阅，已确认架构不再重复确认。

本轮文档静态检查覆盖 5 份新增/更新文档、174 个本地链接与锚点；字段目录与当前公开协议核对为 16 RPC、34 个资源 DTO、215 个字段，计划包含 12 个任务。代码、生成和运行测试均留在实施阶段，本记录不表示那些验证已经通过。
