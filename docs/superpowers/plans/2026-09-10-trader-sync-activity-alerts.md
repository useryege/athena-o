# Trader Sync / Activity Alerts 前后端联合实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现已整体确认的 Trader Sync 前后端：目标确认与管理、共享实时采集、稳定活动阅读、Telegram 普通/摘要结果及管理员安全概要，桌面优先且手机完整可用。

**Architecture:** 保留 athena-server 与 athena-notification 两个进程，全部通知数据并入 athena PostgreSQL。原始日志先可靠保存，最终确认后按 owner 形成活动及投递资格；发送许可、结果与摘要成员均持久化。会员和管理员使用独立页面/service/DTO；可见页轮询更新固定活动页，新活动经提示后载入，采集与发送不依赖浏览器。

**Tech Stack:** 仓库当前 Go 1.25.5、pgx/v5、goose、sqlc、go-ethereum、gorilla/websocket、现有 Telegram 客户端、gogo/protobuf/grpc-gateway；React 18、TypeScript、Ant Design 6、Vite、Jest/react-test-renderer、Playwright。不升级依赖，不新增消息中间件或大型图表框架。

**Spec:** [已整体确认的后端 spec](../specs/2026-09-10-trader-sync-activity-alerts-design.md)、[已整体确认的 UI spec](../specs/2026-09-10-trader-sync-activity-alerts-ui-design.md)、[长期后端设计](../../design/trading/trader-sync-activity-alerts.md)、[长期 UI 设计](../../design/web-ui/trader-sync-activity-alerts.md)、[业务需求](../../requirements/polymarket-copy-trading/target-trade-monitoring-notifications.md)。两份 spec 已获用户整体确认，UI 读取补充已纳入本计划；执行者同时阅读，不按旧后端范围遗漏页面。

**状态：**21项计划的本地实现、运行文档、必要本地验收与最终审查已完成；完成通知已发送一次。执行工作树为 `.worktrees/trader-sync-activity-alerts/`，分支为 `codex/trader-sync-activity-alerts`。真实持续100活跃容量、公开时刻SLO和其他实网证据边界仍按验收记录保留未证明，不代表生产达标。

## Global Constraints

- 首期支持 10 名用户，每人最多 10 个未取消订阅，覆盖 100 个订阅关系及最多 100 个不同目标。
- desired_state 为 enabled/paused/permission_disabled/cancelled；observation_state 为 pending_baseline/healthy/interrupted。手动恢复递增 activation_generation，连接恢复只更新 collector_epoch。
- 单供应商 WSS；开发先用 Chainstack，dRPC 仅供手动切换。断线、重启、故障均不补遗漏；可靠保存的合格候选可以继续。
- effective_at=max(数据库当前时刻的下一整秒,H.timestamp+1秒)；H 不低于注册时观察高度，区块不落后数据库时钟超过 10 秒、不领先超过 2 秒。生效起点包含、终点排他。
- finality=2s、latest=10s、RPC timeout=5s；同一 WSS 每 15 秒关联 ping，5 秒 pong 截止；过滤组最多 100 钱包。
- 资料并发=4、最终确认后补资料预算=2s；目录每页最多100、每10分钟一轮且不重叠、最多每秒一页；重连1/2/4/8/16/30秒上限退避加抖动。
- 私有备注按 Unicode code point 最多 20 个；确认 token 有效期 5 分钟；分页默认 50、最多 100。token、游标和请求幂等均绑定 owner。
- trader_sync grant 只允许 NONE/READ_WRITE；READ 鉴权 requirement 合法。管理员只有安全概要，不读取私有备注、活动/消息正文或逐条投递。
- 活动、取消订阅、备注及投递审计无自动 TTL。活动原量/币种、备注快照、通知 payload 与成员归属不因重试改写。金额和 position/token ID 不经浮点。
- Telegram：总尝试最多5次，退避1/2/4/8秒，429至少等Retry-After；sent/failed/unknown/cancelled终态不可复活，unknown不自动重发。
- Trader Sync只发用户私聊；群组预算仅供现有系统通知。许可再次核验binding为该用户私聊，不接受任意chat_id或逐订阅渠道设置。
- 发送默认跨chat并发12、Bot20次/秒、私聊至少1秒间隔、群组最多20次/分钟、单次调用超时5秒；包括账户、系统通知及Bot回复。
- 滚动窗口(recorded_at−60s,recorded_at]内前10条逐条、第11条起摘要；相邻批次首条实际开始至少间隔60秒，最老待汇总活动到本批首条开始目标≤60秒。解析后单条最多4096字符，超长完整分条。
- 公开可查询→站内 P95≤15s/P99≤30s；平稳普通活动→成功回执P95≤5s。同用户集中普通提醒允许限速排队，单列并保留总体统计；未知公开时间不可用received_at替代。
- 暂停/取消保留旧队列；撤权/解绑/重绑终止未获许可旧任务，已许可单条可完成/未知；撤权永久禁止旧delivery后续attempt，重新授权也不能复活。
- 数据库只保留一个权威迁移集；不做双写、兼容读取或自动迁移旧开发数据。执行测试只操作本任务创建的隔离数据库，不重置现有环境。
- SQL输入形成稳定批次后执行一次 `make sqlc-local`，再适配消费者；application types/proto稳定后执行 `make protogen`。禁止手改生成输出，末尾不无条件重跑生成器。
- 落实已批准布局：六类会员页面、管理员独立概要、既有 Notifications/Service Status 衔接；不执行交易，不提供手动重发、持续收益刷新或新渠道设置。
- 可见会员页每5秒、Notifications保留3秒、Service Status保留10秒单飞轮询；隐藏停止。新活动提示后点击载入，刷新保留成员、焦点、选择和位置；撤权被前端获知后清空并忽略晚响应。
- 活动按owner形成顺序id DESC；bigint ID在account gate内由持久sequence分配，CACHE 1、正向、NO CYCLE，不预取/回拨。快照取该owner已提交max(id)，不取sequence.last_value；recorded_at保留真实时间。
- 会员与管理员DTO/service隔离；英文界面、UTC+8、深浅主题、1440/1280/900附近/390、键盘与触屏。身份/金额/费用/Outcome按证据，未知不造零，Combo NO为整体合取补集。

---

## 文件结构、依赖和执行约定

这是同一功能的依赖链，按可独立验证的交付拆成下表任务。原有模块已经有有效调用方，基础重构完成时也必须保持其可运行；不用临时双路径维持旧数据库行为。

| 任务 | 交付 | 前置 |
| --- | --- | --- |
| 1 | 单库迁移、连接归属与真实数据库测试基础 | 无 |
| 2 | 发送尝试状态、实际调用起点和结果分类 | 1 |
| 3 | Bot update、绑定与回复outbox同事务 | 1、2 |
| 4 | 公平调度、共享预算和sender恢复 | 2、3 |
| 5 | 身份确认、P/L与确认token契约 | 1 |
| 6 | 订阅/备注、十模块权限与原子撤权 | 1、2、5 |
| 7 | 来源解码、规范链与Exchange版本核验 | 无，可与1–6独立开发 |
| 8 | 普通市场、Combo腿与目录资料适配 | 5、7 |
| 9 | 基线、共享WSS、原始接收及无回补恢复 | 6、7 |
| 10 | 逐条活动、绑定快照与普通/摘要归属 | 2、6、8、9 |
| 11 | 完整分条摘要及首条短gate协议 | 4、10 |
| 12 | 会员/管理员API、服务组合和生成消费者 | 5–11 |
| 13 | 指标、故障矩阵及首期容量验证 | 12 |
| 14 | 会员类型化 service、可取消读取与精度基础 | 12 |
| 15 | 独立添加页、确认卡与 Notifications 返回草稿 | 14 |
| 16 | 完整订阅列表、详情、备注与观察历史 | 14、15 |
| 17 | 活动主页、导航与稳定刷新/游标会话 | 14、16 |
| 18 | 活动与摘要独立详情、完整通知证据 | 14、17 |
| 19 | 管理员概要、运行健康与安全计数 | 12、14 |
| 20 | 前后端浏览器验收、移动/键盘及隐私回归 | 13、15–19 |
| 21 | 长期文档、最终检查与完成通知 | 20 |

主要新增文件按责任分组：

| 路径 | 责任 |
| --- | --- |
| `internal/accountstate/store/migrations/embed.go` | 权威SQL的独立叶子包，供两个store和迁移CLI导入，避免业务包循环依赖。 |
| `internal/accountstate/txgate/gate.go` | 统一账户/钱包锁键、普通事务和摘要专用session gate。 |
| `internal/testutil/pgtest/postgres.go` | 显式测试DSN、随机隔离数据库、迁移和清理，不访问开发业务库。 |
| `internal/notification/delivery/types.go` | 不依赖store/Service的发送契约，供store、调度器及Trader Sync引用。 |
| `internal/notification/store/attempts.go`、`bot_updates.go`、`summary_heads.go` | 持久许可与结果、Bot事务、摘要首条领取/冻结。 |
| `internal/notification/dispatcher.go`、`budget.go`、`recovery.go` | 调度、公平预算、单sender恢复。 |
| `internal/tradersync/types/types.go`、`internal/tradersync/errors.go`、`config.go` | 领域类型放独立叶子包，service/store共同引用；错误与配置留业务包。 |
| `internal/tradersync/target_resolver.go`、`subscriptions.go` | 确认身份/资料及订阅业务。 |
| `internal/tradersync/exchange_decode.go`、`confirmation.go`、`source_version.go` | 原日志、规范链和来源执行版本。 |
| `internal/tradersync/baseline.go`、`collector.go`、`intake.go`、`rpc/session.go` | 秒级边界、连接代次、持久接收及单会话JSON-RPC。 |
| `internal/tradersync/market_metadata.go`、`combo_metadata.go`、`projector.go` | 市场资料、Combo映射、用户活动形成。 |
| `internal/tradersync/summary.go`、`render.go`、`metrics.go`、`service.go` | 摘要成员与显示、可观测性、后台生命周期。 |
| `internal/tradersync/store/`下各任务列出的SQL/适配文件 | 独立query包，schema引用accountstate权威目录。 |
| `util/polymarket/profile_identity.go`、`profile_stats.go`、`user_pnl.go`、`combo_markets.go` | 类型化公开资料适配和固定来源规则。 |
| `pkg/apis/application/v1alpha1/trader_sync_types.go`、`internal/server/tradersync/` | 手写公共类型/proto/handler；生成输出由protogen产生。 |
| `ui/src/app/member/trader-sync-service.ts`、`trader-sync-models.ts` | 会员HTTP契约与精确DTO；不接收客户端owner。 |
| `ui/src/app/member/pages/trader-sync/` | 六类页面与模块专属读取、草稿、分页、金额及事实组件；不在页面解码链协议。 |
| `ui/src/app/admin/trader-sync-service.ts`、`trader-sync-models.ts`、`pages/trader-sync/` | 管理员独立安全DTO、列表/详情；不导入会员正文模型。 |
| `ui/e2e/trader-sync.spec.ts`、`ui/e2e/trader-sync-live.spec.ts` | 分别验证受控页面场景和真实隔离API读写；证据不得混淆。 |

执行时先依据 using-git-worktrees 建立或识别隔离目录；项目内工作树放 `.worktrees/`，分支使用 `codex/`。计划编写没有创建工作树。各任务按“红灯→最小实现→绿灯→审阅→提交”进行；首次红灯须证明目标行为缺失，缺少工具/环境不是有效红灯。

每个Go测试片段放入指定包测试文件，补上片段实际使用的标准库及现有依赖import；使用标准库testing与仓库已有testify均可。集成测试加 `//go:build integration`，必须显式提供 `ATHENA_TEST_PG_ADMIN_DSN`，缺失时Fatal而非Skip。离线测试不请求真实RPC、Telegram或用户资料。下列命令都是执行阶段运行，本轮未执行。

## 任务1：通知统一数据库与真实事务测试基础

**Files**
- 新增：`internal/accountstate/store/migrations/embed.go`、`internal/accountstate/txgate/gate.go`、`internal/testutil/pgtest/postgres.go`。
- 修改：`internal/accountstate/store/migrations/000001_init.sql`、`internal/accountstate/store/sql_store.go`、`internal/notification/store/sql_store.go`、`internal/migration/modules.go`、`cmd/athena-migrate/commands/athena_migrate.go`、`sqlc.yaml`、`hack/local-runtime.sh`、`hack/postgres/init/00-databases.sql`、`docker-compose.prod.yml`。
- 移除：`internal/notification/store/migrations/000001_init.sql`；其有效Up/Down定义按依赖顺序纳入权威SQL，不直接拼接两个goose段。
- 测试：`internal/migration/modules_test.go`、`internal/accountstate/store/schema_integration_test.go`、`internal/accountstate/txgate/gate_integration_test.go`。
**Interfaces**
- 产出：`migrations.FS embed.FS`、`migrations.Dir = "."`；原 `accountstatestore.Migrations()` 返回此FS，消费者使用对应Dir。
- 产出：`txgate.Beginner`为只含`BeginTx(context.Context,pgx.TxOptions) (pgx.Tx,error)`的最小接口，生产`*pgxpool.Pool`直接满足；`txgate.WithAccountTx(ctx context.Context, pool txgate.Beginner, accountID string, fn func(pgx.Tx) error) error`；`txgate.LockWallet(ctx context.Context, tx pgx.Tx, wallet common.Address) error`。
- 产出：`pgtest.New(t *testing.T, schema fs.FS, dir string) *pgtest.DB`，DB含 `Pool *pgxpool.Pool`、`DSN string`。
- 产出：`(*accountstatestore.SQLStore).Pool() *pgxpool.Pool`，由现有store拥有并关闭；业务适配器只借用，不重复Close。

- [x] **步骤1：写失败测试，证明迁移CLI只有一个athena/notification schema归属。**

```go
func TestModulesHaveOneAthenaOwner(t *testing.T) {
    count := 0
    for _, m := range Modules() {
        if m.Name == "notification" || m.Database == "notification" {
            t.Fatalf("independent notification migration: %+v", m)
        }
        if m.Database == "athena" { count++ }
    }
    if count != 1 { t.Fatalf("athena migration owners=%d", count) }
}
```

运行 `go test ./internal/migration -run TestModulesHaveOneAthenaOwner -count=1`；预期因现有notification独立条目失败。

- [x] **步骤2：实现隔离数据库夹具及两连接锁测试。**夹具读取显式测试admin DSN，用 `pgx.Identifier{name}.Sanitize()` 创建 `athena_test_`+随机UUID数据库；从同DSN替换dbname，调用 `postgres.Migrate(ctx, dsn, schema, dir)`，再创建pool。Cleanup先关pool，再删除自己创建的数据库。禁止将输入DSN数据库直接清空；任何迁移失败Fatal并清理已创建资源。

```go
// migrations/embed.go
package migrations
import "embed"
//go:embed *.sql
var FS embed.FS
const Dir = "."
```

`schema_integration_test.go`用 `pgtest.New(t,migrations.FS,migrations.Dir)` 后查询 `to_regclass`，断言athena_account、account_access、telegram_bindings、account_notification_deliveries、system_notification_deliveries及telegram_polling_state均存在。对全新隔离库启动两个独立测试进程首次迁移，验证实际 advisory lock 竞争、建表及版本唯一性，避免进程内gooseMu使并发测试失去区分力。夹具同时验证current_database为新建随机库且管理库未被迁移；修改pgx配置后不能用仍返回原DSN的ConnString冒充重建连接字符串。gate测试用两个连接与channel屏障证明同owner串行、不同owner可并行。

- [x] **步骤3：合并SQL与连接归属，稳定输入后生成，再适配store/CLI。**

```sql
-- WithAccountTx取得锁后才读取权限/时间；UUID先由Go严格规范化。
SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text, 0));
-- 原始接收与基线注册共用钱包命名空间；不得反向再申请账户锁。
SELECT pg_advisory_xact_lock(hashtextextended('athena:wallet:' || $1::text, 0));
```

`WithAccountTx`使用ReadCommitted，Begin→锁→fn→Commit；defer Rollback只负责清理。迁移注册结构增加逐模块 `Dir` 字段，原模块用"migrations"，athena用migrations.Dir；调用点使用选中模块的Dir，不改其他模块schema。Notification source使用ATHENA_SERVER_POSTGRES_DSN、athena及同一叶子FS；更新本地/Compose/bootstrap消费者并移除旧通知数据库配置。`sqlc.yaml`通知schema改为权威目录后运行 `make sqlc-local`，检查生成类型没有丢表。

- [x] **步骤4：运行 `go test ./internal/migration ./internal/accountstate/... ./internal/notification/...` 和 `go test -tags=integration ./internal/accountstate/... -count=1`；确认两连接及同库建表断言通过。**测试DB实例可用专用命令创建：`docker run --rm -d --name athena-trader-sync-test-pg -e POSTGRES_PASSWORD=athena-test -p 127.0.0.1:55439:5432 postgres:16`；若名称/端口占用选择新的测试实例，不操作已有容器。测试DSN为 `postgres://postgres:athena-test@127.0.0.1:55439/postgres?sslmode=disable`，设置到ATHENA_TEST_PG_ADMIN_DSN；执行结束只停止本任务创建的容器。
- [x] **步骤5：审阅并提交 `refactor(storage): unify notification persistence in athena`。**只暂存本任务文件及该批真实生成输出。

## 任务2：持久发送许可、明确结果及实际调用起点

**Files**
- 新增：`internal/notification/delivery/types.go`、`internal/notification/store/attempts.go`、`internal/notification/store/queries/delivery_attempts.sql`、`util/telegram/send_transport.go`。
- 修改：权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/notification/store/queries/account_notifications.sql`、`internal/notification/store/queries/system_notifications.sql`、`internal/notification/store/account_notifications.go`、`internal/notification/store/system_notifications.go`、`internal/notification/sender.go`、`internal/notification/worker.go`、`util/telegram/telegram.go`、`internal/notification/notification.proto`、`internal/server/notification/notification.proto`、`internal/server/notification/notification.go`。
- 修改实际状态消费者：`internal/notification/service.go`、`pkg/apis/application/v1alpha1/notification_types.go`、`ui/src/app/admin/notification-service.ts`、`ui/src/app/admin/pages/service-status.tsx`、`ui/src/app/admin/pages/system-notifications.tsx`、`ui/src/app/admin/pages/system-notification-detail.tsx`。
- 测试：`internal/notification/store/attempts_integration_test.go`、`util/telegram/send_transport_test.go`、`internal/notification/sender_test.go`。同步当前worker真实Send调用，使许可和结果协议立即生效，任务4再替换调度。
- 执行基线补充：修复`ui/jest.config.js`的ts-jest/CommonJS解析配置、补测试专用Fetch API环境，并使`ui/src/app/app.test.tsx`登录界面断言和`ui/src/app/shared/services/user-service.test.ts`请求参数断言反映当前真实行为。保留产品bundler配置、权限/取消行为和有效断言，完整跑通既有UI回归；任务14仍负责扩大测试发现范围。
**Interfaces**
- 消费：任务1的账户gate、通知同库表。
- 产出：delivery叶子包中的 `WorkRef{Kind string; ID int64}`、`Permit{Work WorkRef; AttemptID uuid.UUID; OwnerID string; SenderIncarnation uuid.UUID; PayloadDigest []byte; AuthorizedAt time.Time}`、`Outcome{Kind string; MessageID string; RetryAfter time.Duration; Code string}`。Outcome.Kind为sent/retryable/failed/unknown。
- 产出：`(*SQLStore).Authorize(ctx context.Context, ref delivery.WorkRef, incarnation uuid.UUID) (delivery.Permit,error)`；`RecordStarted(ctx context.Context,p delivery.Permit,at time.Time) error`；`RecordOutcome(ctx context.Context,p delivery.Permit,o delivery.Outcome,at time.Time) error`。
- 修改Sender契约：`Send(ctx context.Context, request SendRequest, started func(time.Time)) delivery.Outcome`；保留CreateSystemTopic方法。store不导入父notification包，避免当前已有父包→store的循环。

- [x] **步骤1：写HTTP故障测试，先证明“对端已接收但无响应”不会重发。**使用httptest handler计数并读取请求体后Hijack关闭连接；调用一次Send，断言Outcome为unknown且handler计数为1。另一个handler返回Telegram `ok=false,error_code=429,parameters.retry_after=7`，断言retryable及7秒；本地空消息则failed且计数为0。

```go
func TestUnknownOutcomeIsTerminal(t *testing.T) {
    o := delivery.Outcome{Kind: "unknown", Code: "response_lost"}
    next, wait := delivery.NextState(o, 1, true)
    if next != "unknown" || wait != 0 { t.Fatalf("%s %s", next, wait) }
    o = delivery.Outcome{Kind: "retryable", RetryAfter: 7*time.Second}
    next, wait = delivery.NextState(o, 1, false)
    if next != "cancelled" || wait != 0 { t.Fatalf("revived revoked delivery: %s", next) }
}
```

定义并实现 `delivery.NextState(o Outcome, attempts int, eligible bool) (string,time.Duration)`；测试首先因缺少该规则失败。

- [x] **步骤2：实现attempt表、资格墓碑与CAS SQL，生成后适配消费者。**新增 `notification_delivery_attempts`：UUID主键、work_kind/work_id、incarnation、payload_digest、authorized_at、nullable started_at/result_at/message_id、outcome；账户投递增加eligibility_revoked_at/reason及current_attempt_id，系统投递增加current_attempt_id及sending/unknown状态。attempt与work关系按kind在许可事务核验，账户owner一致，不用可变payload复用许可。

```sql
-- 结果更新必须绑定当次许可；行数0表示迟到/已终结，不能重新发送。
UPDATE account_notification_deliveries
SET status = $3, provider_message_id = $4
WHERE id = $1 AND current_attempt_id = $2 AND status = 'sending';
```

许可与attempt同时提交后才调用Telegram。资格已永久撤销时，明确failed或retryable均将delivery终结为cancelled，attempt保留真实失败事实；sent/unknown照实保存。仅明确retryable且资格未永久撤销、attempts<5时回pending；等待=max(1/2/4/8秒对应值,RetryAfter)。success写库失败只重试结果CAS。提交结果不确定先按attempt UUID读取，无法确认则不发送。同步现有Notification公共/内部proto的状态表达，稳定后执行一次 `make protogen`，适配现有状态消费者，不让unknown被旧映射误报成功。

当前公共枚举cancelled已经存在（值4），新增sending/unknown使用未占用编号。同步deliveryStatusString、normalizeDeliveryStatusFilter、runtime计数和管理员系统列表/详情的筛选、颜色、时间字段；账户逐条投递仍不向管理员开放。若application类型新增attempt时间，同批 `make clientgen` 更新deepcopy。会员notification-service里的binding attempt状态不是消息状态，不改成sending/unknown。状态测试覆盖JSON字符串往返和unknown独立计数。

- [x] **步骤3：在实际HTTP RoundTrip入口记录started，禁用隐式重试。**新transport包装既有 `http.RoundTripper`，通过本次请求context的回调在进入base.RoundTrip前记录起点；不在goroutine启动/队列领取处调用。回调写入容量1的握手channel，不等待响应。预校验/编码失败不报告started；禁止跟随会重放POST的重定向，检查现有go-telegram/bot选项，发送路径只允许一个HTTP请求。保留原始结构化错误分类后再形成对外错误文本，不能把所有网络错误都归为“确定失败”。

- [x] **步骤4：运行 `go test ./util/telegram ./internal/notification/...`、`go test -tags=integration ./internal/notification/store -run 'TestAttempt|TestOutcome' -count=1`。集成矩阵覆盖许可后崩溃、成功后结果写库失败、结果CAS冲突、起点缺失但有成功回执，以及撤权→重授→旧attempt返回429仍cancelled。**
- [x] **步骤5：提交 `feat(notification): persist send permits and terminal outcomes`。**

## 任务3：Bot update、绑定与回复outbox原子消费

**Files**
- 新增：`internal/notification/store/bot_updates.go`、`internal/notification/store/queries/bot_updates.sql`。
- 修改：权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/notification/store/telegram_bindings.go`、`internal/notification/poller.go`。
- 核对并复用：`internal/notification/service.go`的同store装配、`internal/notification/store/queries/telegram_bindings.sql`的账户锁与单调offset；现有实现已满足，不要求无意义修改。
- 修改必要发送消费者：`internal/notification/store/attempts.go`、对应attempt SQL与`internal/notification/worker.go`，使reply复用已持久化的许可/结果路径；新调度器仍由任务4实现。
- 测试：`internal/notification/store/bot_updates_integration_test.go`、`internal/notification/poller_test.go`。
**Interfaces**
- 消费：任务1 txgate和任务2投递结果契约。
- 产出：`(*SQLStore).ApplyBotUpdate(ctx context.Context, update utiltelegram.Update) error`；内部binding的Tx方法只由该事务调用；`telegram_binding_replies`作为独立work_kind=reply参与发送。

- [x] **步骤1：写重复update的失败集成测试。**夹具创建owner与pending绑定token，构造Update{ID:42,Message:私聊/start token}；调用ApplyBotUpdate两次。断言绑定revision只增一次、next_update_id=43、只有一条reply，且未调用Telegram SendMessage。异常分支包括错误token、群聊、bot用户、已被另一owner绑定和my_chat_member失联事件。

```sql
-- 必须与绑定变化、回复outbox及offset同事务；重复ID不再次执行副作用。
INSERT INTO telegram_consumed_updates (update_id, consumed_at)
VALUES ($1, clock_timestamp()) ON CONFLICT (update_id) DO NOTHING
RETURNING update_id;
```

- [x] **步骤2：写提交前后故障测试，运行 `go test -tags=integration ./internal/notification/store -run TestApplyBotUpdate -count=1` 确认红灯。**在提交前返回错误，断言四类记录全回滚；包装真实Commit使其提交后报告模拟断流，再重放update，断言不重绑/不重复回复。所用故障注入是测试事务装饰器，不添加业务feature flag。
- [x] **步骤3：增加consumed_updates及binding_replies表，形成SQL批次后 `make sqlc-local`，再将poller改为单次ApplyBotUpdate。**账户gate之后才取Telegram用户唯一锁；跨owner冲突不抢占已有绑定。解绑/重绑取消旧revision未授权任务，永久标记旧sending不得后续重试。删除poller直接sendBindingReply的网络调用，改写reply work。有效绑定修改、消费进度和回复必须使用同一个pgx.Tx。
- [x] **步骤4：运行本任务集成测试和 `go test ./internal/notification/...`；检查rollback后offset未前进、已提交重启从持久offset继续。**
- [x] **步骤5：提交 `feat(notification): consume bot updates transactionally`。**

## 任务4：共享限速、公平调度与单sender恢复

**Files**
- 新增：`internal/notification/dispatcher.go`、`internal/notification/budget.go`、`internal/notification/recovery.go`；对应`internal/notification/dispatcher_test.go`、`internal/notification/budget_test.go`、`internal/notification/recovery_integration_test.go`。
- 修改：`internal/notification/worker.go`、`internal/notification/service.go`、`internal/notification/server.go`、`internal/notification/store/queries/account_notifications.sql`、`internal/notification/store/queries/system_notifications.sql`、`internal/notification/store/queries/bot_updates.sql`。
- 扩展必要存储与组合入口：`internal/accountstate/store/migrations/000001_init.sql`及通知SQL/生成消费链、`internal/notification/store/`的sender实例/恢复持久化、`internal/notification/delivery/types.go`、`cmd/athena-notification/commands/athena_notification.go`。实例登记与明确恢复CLI、attempt实际路由持久化是本任务恢复/预算的组成部分。
- 动态限流入口适配：`internal/notification/sender.go`、`util/telegram/{send_transport.go,telegram.go,send_transport_test.go}`；实际HTTP前独立可取消准入，准入后计算五秒HTTP超时，测试许可已commit但HTTP未开始时沿用同attempt等待。
**Interfaces**
- 产出delivery类型：`Candidate{Ref WorkRef; OwnerID string; ChatID int64; Group bool; NotBefore time.Time; Deadline *time.Time}`。
- 产出：`Budget.Next(c delivery.Candidate, now time.Time) time.Time`、`Budget.Start(c delivery.Candidate, at time.Time)`；内部保存Bot、chat及group窗口。另提供`Reserve(c delivery.Candidate,now time.Time) (reservationID uint64,ok bool)`、`Release(reservationID uint64)`；Reserve计入尚未started的预留容量，Start按Candidate.Ref原子消耗预留并写实际起点，防并发Next检查超售。
- 产出：`WorkSource.Ready(ctx context.Context,now time.Time) ([]delivery.Candidate,error)`与`WorkSource.Dispatch(ctx context.Context,c delivery.Candidate,onStarted func(time.Time)) error`；账户/系统/reply源使用任务2许可流程，任务11增加摘要首条源。
- 产出：`NewDispatcher(clock Clock,sources []WorkSource,budget *Budget,concurrency int) *Dispatcher`、`Dispatcher.Run(ctx context.Context) error`。Clock契约为`Now() time.Time`、`After(time.Duration) <-chan time.Time`；生产包装time，测试手动推进。
- 产出：`(*notificationstore.SQLStore).RecoverSender(ctx context.Context,stoppedIncarnation uuid.UUID) error`只在外部已确认旧进程停止后调用。

- [x] **步骤1：写纯预算红灯测试。**

```go
func TestBudgetKeepsDifferentChatsIndependent(t *testing.T) {
    b := NewBudget(20, time.Second, 20, time.Minute)
    at := time.Unix(100,0)
    a := delivery.Candidate{ChatID:1}
    b.Start(a,at)
    if got:=b.Next(a,at); !got.Equal(at.Add(time.Second)) { t.Fatal(got) }
    other:=delivery.Candidate{ChatID:2}
    if got:=b.Next(other,at); !got.Equal(at) { t.Fatal(got) }
}
```

定义 `NewBudget(botPerSecond int, privateInterval time.Duration, groupCount int, groupWindow time.Duration) *Budget`；窗口用实际started事件推进，不用入队时间占用历史额度。Dispatcher在启动WorkSource前Reserve，onStarted回调通知Budget.Start；未调用HTTP的失败路径Release，不能吞掉预留或重复记入额度。

- [x] **步骤2：用可控clock和阻塞fake Sender写调度失败测试。**10个owner不同chat同时就绪，断言能同时启动且不超过12；同chat第二条等前一尝试结束及1秒预算；一个owner12条不阻塞其他owner。账户/系统/reply共用20/秒预算，group窗口20/分钟。使用channel+clock推进，不写真实time.Sleep。
- [x] **步骤3：替换全局1.1秒串行worker，接入WorkSource及许可流程。**按owner公平轮转并优先即将到期任务；看见尚未到NotBefore的未来Deadline时预留“HTTP最长5秒+chat间隔”的槽。取得预算前不拿账户gate，等锁后槽过期则重排；同chat只有一个执行中的许可。临时速率错误缩紧预算，unknown不回队。预算收紧同时使尚未许可的旧预留失效；已提交许可但未实际开始HTTP的调用沿用同一attempt等待，不新增许可或计次。独立可取消pre-start闸门与Tighten确定排序，等待不持账户gate，也不阻塞constant-time Started回调。任务11的摘要source须在动态等待时释放/重新进入短sessiongate并保存同一冻结批次及许可，持久未开始状态仍保护批次顺序。用单sender incarnation/session lease检测失锁，失锁后停止授权并退出；绝不因lease过期并行接管Telegram。

```sql
-- 只处理已确认停止的incarnation；其余kind使用相同CAS条件更新各自表。
UPDATE account_notification_deliveries d SET status = 'unknown'
FROM notification_delivery_attempts a
WHERE d.current_attempt_id = a.id AND d.status = 'sending'
  AND a.sender_incarnation = $1 AND a.outcome IS NULL;
```

有明确结果的旧attempt先补记该事实；没有结果才unknown，不能以重发探测成功与否。短期连接错误不能把旧permit当成未发送。 持久记录sender实例身份；启动持独占session锁，存在未确认停止的旧实例时拒绝启动，即使锁已失效。正常Stop等待poller及所有dispatch/sender与结果补记结束才登记stopped。异常失锁使健康失败、取消调度/poller并将错误传到CLI退出；操作员确认登记进程已退出后用`--recover-stopped-sender=<UUID>`明确恢复，不凭锁到期、端口或PID推断。

attempt许可冻结实际chat ID/group并与Sender请求一致。预算恢复读取全部近期attempt，缺起点且有结果以result_at作为保守上界；无起点无结果以停止确认时刻等待完整窗口，保留缺失started_at及真实结果。配置路由变化不能改写旧调用预算。 当前无跨进程UTC偏差证明，每次非首次启动另设60秒monotonic恢复屏障；只有实例与attempt历史均为空的可靠首次启动可豁免。持久保留未由monotonic计时确认届满的Retry-After，重启不按UTC过滤，保守等待完整未解除时长并在确实届满后标记解除。恢复等待是本地时延，单列且保留总体统计；测试UTC前跳、长Retry-After、已解除历史与停止取消屏障。

- [x] **步骤4：运行 `go test -race ./internal/notification/...` 和 `go test -tags=integration ./internal/notification -run TestRecoverSender -count=1`；断言恢复不增加HTTP调用，停止时无泄漏goroutine/连接。**
- [x] **步骤5：提交 `feat(notification): schedule deliveries fairly across chats`。**

## 任务5：精确身份、六区间资料与确认token

**Files**
- 新增：`internal/tradersync/types/types.go`、`internal/tradersync/target_resolver.go`、`internal/tradersync/profile.go`、`internal/tradersync/pnl.go`、`internal/tradersync/errors.go`、`internal/tradersync/store/sql_store.go`、`internal/tradersync/store/confirmations.go`、`internal/tradersync/store/queries/confirmations.sql`。
- 新增：`util/polymarket/profile_identity.go`、`util/polymarket/profile_stats.go`、`util/polymarket/user_pnl.go`。
- 修改：`util/polymarket/data.go`、`util/polymarket/gamma.go`、`sqlc.yaml`、权威`internal/accountstate/store/migrations/000001_init.sql`。
- 测试：`util/polymarket/profile_identity_test.go`、`util/polymarket/profile_stats_test.go`、`util/polymarket/user_pnl_test.go`、`internal/tradersync/target_resolver_test.go`、`internal/tradersync/pnl_test.go`、`internal/tradersync/store/confirmations_integration_test.go`。
- 依据：[Profile/P/L契约核验](../../requirements/polymarket-copy-trading/profile-pnl-contract-verification.md)。
**Interfaces**
- 以下任务统一将`internal/tradersync/types`导入为`tsmodel`，避免service→store→service循环。
- 产出类型：`Identity{Wallet common.Address; ResolutionInput,ProfileURL,DisplayName,AvatarURL string; Digest [32]byte}`；`Evidence{Availability,ReasonCode,Source string; QueriedAt time.Time}`；`Scalar{Evidence; Value *string}`；`PnLPoint{T int64; P string}`；`Curve{Evidence; Points []PnLPoint}`。数字以字符串保留原值，Value=nil代表缺失。
- Identity.Digest只摘要规范钱包、canonical身份映射及适配器版本；显示名、头像、收益变化不改变身份。ResolutionInput保存规范原始查询地址或受限输入URL，digest绑定其到规范钱包/canonical的映射；Revalidate重新解析该参照，不能只查询已解析钱包。原参照缺失、改指另一钱包或无法复核都要求重新解析确认，无旧身份格式兼容回退。完整确认卡资料另保存查询证据，不把辅助资料变化判成另一个目标。
- 产出类型：`ConfirmationCard{Identity Identity; Avatar,DisplayName,Verified,JoinedAt,PositionValue,LargestWin,Predictions Scalar; PnL map[string]PnLView; DefaultPeriod,UsageNotice string}`；`PnLView{Amount Scalar; Curve Curve; Interval,Fidelity string; Reference *time.Time; Timezone string}`；`ResolvedTarget{Card ConfirmationCard; Context ResolutionContext; Token string; ExpiresAt time.Time}`。
- 产出tsmodel：`TargetNote{Wallet common.Address; Note string; Revision uint64}`、`ExistingSubscription{ID,Status string; Revision uint64}`、`Quota{Used,Limit int32}`、`ResolutionContext{SavedNote *TargetNote; Existing *ExistingSubscription; Quota Quota}`。SavedNote=nil与Note=""不同；Current五态均占配额。
- 产出：`(*TargetResolver).Resolve(ctx context.Context,ownerID,input string) (tsmodel.ResolvedTarget,error)`；`Revalidate(ctx context.Context,identity tsmodel.Identity) error`。外部查询先完成再进入短事务保存token；当前grant在写入前重新核验。
- 产出：`PlanPNL(period string,allAge *time.Duration) (interval,fidelity string)`；`BuildPNL(period string,raw []tsmodel.PnLPoint,allAge *time.Duration,rules PNLRules) tsmodel.PnLView`；`PNLRules{Reference *time.Time; Zone *time.Location; Round func(*big.Rat) string}`。未知规则用nil表达，不自定官网算法。
- 产出store：`NewSQLStore(pool *pgxpool.Pool) *SQLStore`；`SaveConfirmationTx(ctx context.Context,tx pgx.Tx,ownerID string,identity tsmodel.Identity,tokenDigest []byte,expiresAt time.Time) error`；`ReadConfirmationTx(ctx context.Context,tx pgx.Tx,ownerID string,tokenDigest []byte) (tsmodel.Identity,error)`；`ConsumeConfirmationTx(ctx context.Context,tx pgx.Tx,ownerID string,tokenDigest []byte,requestID string,identityDigest []byte) (tsmodel.Identity,error)`。Read核验owner、未消费和expires_at但不消费。
- Resolver另注入必需`ResolveContextTx func(context.Context,pgx.Tx,string,common.Address) (tsmodel.ResolutionContext,error)`；本任务使用显式fake，任务6提供SQL实现，任务12才组合公开入口。外部资料完成后，在同account gate事务内RequireGrant→ResolveContextTx→SaveConfirmationTx，返回同一次确认的owner上下文；不在外部查询期间持锁。

- [x] **步骤1：写精确请求与缺失值红灯测试。**

```go
func TestMonthQueryUsesStrict31DayBoundary(t *testing.T) {
    short := 31*24*time.Hour-time.Nanosecond
    exact := 31*24*time.Hour
    interval,_ := PlanPNL("1M",&short)
    if interval!="all" { t.Fatal(interval) }
    interval,_ = PlanPNL("1M",&exact)
    if interval!="1m" { t.Fatal(interval) }
    interval,fidelity := PlanPNL("1Y",nil)
    if interval!="all" || fidelity!="1d" { t.Fatal(interval,fidelity) }
}
```

`profile_stats_test.go`用httptest返回`{"traded":9007199254740993}`与`{"traded":0}`，分别断言原整数不变和available真0；null/缺字段为unavailable。运行 `go test ./internal/tradersync ./util/polymarket -run 'TestMonthQuery|TestProfileStats' -count=1`，确认针对缺失行为失败。

- [x] **步骤2：实现受限身份读取与类型化数值。**URL只允许HTTPS、`polymarket.com`/`www.polymarket.com`和已核准`/@handle`路径；拒绝userinfo、端口、伪域、额外路径及改变身份的query/fragment；最大响应2MiB、重定向最多3次且每跳验证相同规则、超时5秒。这些是输入防护默认值，不改变合法身份。HTTP适配器注入RoundTripper供测试，生产allowlist不由用户输入改变。解析固定版本SSR身份和canonical，与public-profile明确钱包交叉核验；冲突、缺关键结构、多个候选钱包均FailedPrecondition，不调用PublicSearch。

```go
// 自定义数值UnmarshalJSON保留原始十进制token，不能先Decode进float64。
func decimalToken(raw []byte) (*string,error) {
    text:=strings.TrimSpace(string(raw))
    if text=="null" { return nil,nil }
    if !json.Valid(raw) || len(text)==0 || (text[0]!='-' && (text[0]<'0'||text[0]>'9')) {
        return nil,fmt.Errorf("invalid numeric token")
    }
    if _,ok:=new(big.Rat).SetString(text); !ok { return nil,fmt.Errorf("invalid decimal") }
    return &text,nil
}
```

替换现有GetTotalMarketsTraded的map及DataUserValue的float表示，适配实际调用方；不保留无消费者旧接口。加入时间只用joinDate，Predictions只用traded；辅助字段逐项记录source/queriedAt/reason。

- [x] **步骤3：实现六区间请求、裁切与独立availability。**fallback为1d/1h、1w/3h、1m/18h、all/1d；1Y/YTD共用ALL。已知ALL历史年龄时按证据表的H/N从[1,3,12,18,24]小时选最小误差，平局较小候选；1M严格<31天使用all。对实际t/p数组校验时间非递减（相邻同时间点按原顺序保留，不排序或去重）、原数组及裁切后至少2点，裁切边界包含，原p不平移。

```go
// 区间未舍入金额的核心；调用前已经验证两端存在及ALL年龄证据。
end,_:=new(big.Rat).SetString(points[len(points)-1].P)
begin,_:=new(big.Rat).SetString(points[0].P)
amount:=new(big.Rat).Set(end)
if period!="ALL" && !shortHistory { amount.Sub(end,begin) }
```

此片段位于BuildPNL内部：`points`是验证后裁切数组；`shortHistory`按报告各区间严格阈值计算，YTD使用报告的年初规则。Reference缺失使相关裁切不可验证；YTD缺时区只影响YTD；Round缺失使精确显示金额unavailable但保留可验证原始曲线。直接取得的官方显示原值可以有独立证据，不用排行榜/持仓求和代替。

- [x] **步骤4：增加token和幂等存储，稳定SQL后生成。**`trader_sync_targets`钱包唯一；`trader_sync_target_confirmations`保存token SHA-256 digest、owner、identity_json/digest、expires_at、consumed_request_id；`trader_sync_request_results`以(owner,operation,request_id)唯一并保存payload_digest/result_json。token使用32字节crypto/rand，DB时钟5分钟；成功Create事务中消费，过期或其他request重复消费拒绝。

```sql
UPDATE trader_sync_target_confirmations
SET consumed_request_id = $3
WHERE owner_id = $1 AND token_digest = $2
  AND expires_at > clock_timestamp() AND consumed_request_id IS NULL
  AND identity_digest = $4
RETURNING identity_json;
```

在sqlc.yaml新增独立tradersync输出包（`internal/tradersync/store/sqlc`），schema仍指向权威目录；`make sqlc-local`后实现适配器。Resolver要求注入`GrantCheck func(context.Context,pgx.Tx,string) error`与ResolveContextTx，缺任一依赖构造失败。Consume由已经核验grant的Create事务调用，owner条件仍由SQL强制。此任务用显式授权/拒绝fake测试调用及nil/空备注、六个period完整性，任务6提供真实`RequireGrantTx(ctx context.Context,tx pgx.Tx,ownerID string) error`及上下文查询，任务12才注册公共入口；不增加临时放行生产路径。

- [x] **步骤5：运行 `go test ./util/polymarket ./internal/tradersync/...` 及对应confirmation集成测试。**覆盖精确URL、canonical冲突、结构变化、跨域重定向、身份变化/超时、owner错用、到期、不同request重消费；6区间逐项缺失、负值、真0、31日边界、30日裁切、365日、跨年及舍入证据缺失。提交 `feat(trader-sync): resolve targets with evidence-backed profile data`。

## 任务6：独立订阅、备注、十模块授权与原子撤权

**Files**
- 新增：`internal/tradersync/types/subscription.go`、`internal/tradersync/subscriptions.go`、`internal/tradersync/store/subscriptions.go`、`internal/tradersync/store/queries/subscriptions.sql`、`internal/tradersync/store/revocation.go`、`internal/accountstate/store/access_hooks.go`。
- 修改：`internal/accountaccess/access.go`、`internal/accountaccess/controller.go`、`internal/accountstate/store/sql_store.go`、`internal/accountstate/store/queries/account_access.sql`、权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/server/account/account.proto`、`internal/server/account/account.go`。
- 修改组合入口：`internal/server/athena-server.go`的`NewServer`，在账户store创建后、EnsureDevelopmentAccount与NewController之前注入真实AccessChangeHook；本步骤不依赖Collector。
- 修改：`ui/src/app/shared/access-modules.ts`、`ui/src/app/shared/account-access.ts`、`ui/src/app/shared/models.ts`、`ui/src/app/admin/accounts-service.ts`、`ui/src/app/admin/pages/admin-accounts.tsx`；核对`ui/src/app/shared/pages/account-center.tsx`及`ui/src/app/member/app.tsx`实际消费者。
- 测试：`internal/accountaccess/access_test.go`、`internal/accountstate/store/access_integration_test.go`、`internal/tradersync/subscriptions_test.go`、`internal/tradersync/store/subscriptions_integration_test.go`、`internal/server/account/account_test.go`、`ui/src/app/shared/access-modules.test.ts`、`ui/src/app/shared/account-access.test.ts`、`ui/src/app/shared/models.test.ts`、`ui/src/app/admin/pages/admin-accounts.test.tsx`。
**Interfaces**
- 消费：任务1 `txgate.WithAccountTx`、任务5 `ConsumeConfirmationTx`与`TargetResolver.Revalidate`；任务2旧delivery资格墓碑。
- 产出tsmodel：`Subscription{ID,OwnerID string; Wallet common.Address; DesiredState,ObservationState,Reason string; Revision,Generation,NoteRevision uint64; EffectiveAt,EndedAt *time.Time; CreatedAt,UpdatedAt time.Time; Note,BindingStatus,QueueNotice string}`；`CreateInput{Token,RequestID string; Note *string}`；`ChangeInput{SubscriptionID,RequestID string; ExpectedRevision uint64}`；`NoteInput{Wallet common.Address; RequestID,Note string; ExpectedRevision uint64}`，NoteInput的版本是note revision而非subscription revision。
- 产出：`(*SubscriptionService).Create(ctx context.Context,ownerID string,in tsmodel.CreateInput) (tsmodel.Subscription,error)`；`Change(ctx context.Context,ownerID,action string,in tsmodel.ChangeInput) (tsmodel.Subscription,error)`，action只允许pause/resume/cancel；`UpdateNote(ctx context.Context,ownerID string,in tsmodel.NoteInput) (tsmodel.TargetNote,error)`；`ValidateNote(note string) error`。
- 产出store：`ResolveContextTx(ctx context.Context,tx pgx.Tx,ownerID string,wallet common.Address) (tsmodel.ResolutionContext,error)`，返回自身当前备注、未取消订阅与配额；`ReadCreateResultTx(ctx context.Context,tx pgx.Tx,ownerID string,in tsmodel.CreateInput) (*tsmodel.Subscription,error)`，nil表示未提交成功，不同payload同request拒绝。必须在RequireGrantTx之后调用。
- 产出：`accountstatestore.AccessChangeHook func(ctx context.Context,tx pgx.Tx,accountID string,previous,next accountaccess.Access) error`，通过`SetAccessChangeHook(hook AccessChangeHook)`在服务启动前一次注入；`(*tradersyncstore.SQLStore).RevokeTx(ctx context.Context,tx pgx.Tx,ownerID,reason string) error`。accountstate不导入tradersync业务包。
- 基线注册接口由本任务定义、任务9实现：`BaselineRegistrar.RegisterTx(ctx context.Context,tx pgx.Tx,sub tsmodel.Subscription) error`。Create/resume事务在account gate后调用，未连接也持久pending attempt；禁止提交后才异步补登记归属。

- [x] **步骤1：写授权与Unicode备注红灯测试。**

```go
func TestTraderSyncGrantDiffersFromReadRequirement(t *testing.T) {
    access:=Access{Modules:NoModuleAccess()}
    access.Modules[ModuleTraderSync]=AccessLevelRead
    if access.Validate()==nil { t.Fatal("READ grant accepted") }
    if err:=RequireModule(ModuleTraderSync,AccessLevelRead).validate(); err!=nil { t.Fatal(err) }
    if !accessLevelSatisfies(AccessLevelReadWrite,AccessLevelRead) { t.Fatal("RW cannot read") }
    if len(AllModules())!=10 { t.Fatal(len(AllModules())) }
}
```

`ValidateNote`测试用`strings.Repeat("🙂",20)`成功、21失败；不要用字节长度。运行 `go test ./internal/accountaccess ./internal/tradersync -run 'TestTraderSyncGrant|TestNote' -count=1`。

- [x] **步骤2：补齐schema约束和完整矩阵写入。**订阅表以(owner,wallet)非cancelled部分唯一，revision/generation>0；备注独立(owner,wallet)唯一且保留取消后数据。baseline_attempts/monitor_intervals在本批建立基础列：subscription/generation/epoch/filter_revision/expected_revision、registered_high、candidate_effective_at、state、effective_at/ended_at及原因，attempt状态pending/succeeded/failed单向结束。owner引用和订阅引用使用复合FK保持一致；request结果事务失败不落库。

```sql
CREATE UNIQUE INDEX trader_sync_one_live_target
ON trader_sync_subscriptions(owner_id,wallet)
WHERE desired_state <> 'cancelled';
-- 在账户gate内读配额，拒绝第11个；不限制整个产品第11名用户。
SELECT count(*) FROM trader_sync_subscriptions
WHERE owner_id=$1 AND desired_state<>'cancelled';
```

account_module_access的合法模块与grant检查、ReplaceAccountModuleAccess及Go读写参数均扩为十项；Trader Sync READ用CHECK和Go校验双重拒绝。`make sqlc-local`后适配stores。

- [x] **步骤3：实现创建/状态/备注事务及并发测试。**Create分两次短事务：首次account gate内核验grant→ReadCreateResultTx，有成功则立即返回，不访问token/外部身份；未命中才ReadConfirmationTx。锁外Revalidate身份；再次account gate核验grant→再次ReadCreateResultTx→重新核验token/identity digest/配额/唯一性→备注（nil沿用，指针空清空）→subscription→RegisterTx→消费token→成功结果。第二次检查处理并发同request已完成；外部核验不能跨锁。暂停/取消用事务实际时间关闭区间、revision+1；允许ended_at早于effective_at，表示尚未生效即终止的空覆盖区间，不钳制真实终点；resume生成新generation且登记新pending attempt；自动网络恢复不经过resume。取消不可恢复，重建新ID沿用备注。UpdateNote返回保存后的值/revision，响应丢失亦重取相同成功结果。

```go
func ValidateNote(note string) error {
    if !utf8.ValidString(note) || utf8.RuneCountInString(note)>20 {
        return status.Error(codes.InvalidArgument,"note exceeds 20 Unicode code points")
    }
    return nil
}
```

真实数据库两连接屏障测试同时创建第10/11项只一成功、同wallet仅一未取消、失败token不占配额、相同幂等返回同ID而不同payload拒绝。首次成功后将token置过期并令身份adapter调用即失败，重试仍返回原ID；撤权后同请求必须拒绝。ResolveContextTx验证nil备注、保留空串、取消后保留、五态占位与另owner不可见。取消/重建及改备注均不重写历史、generation或基线。

- [x] **步骤4：将撤权hook嵌入权限事务并验证不可复活。**controller全局更新mutex改为按账户串行，缓存发布仍在Commit后；DB revision CAS保留。UpdateAccountAccess在同account gate/tx内执行hook，RW→NONE时关闭区间、permission_disabled、取消无许可工作、给包括sending在内的旧Trader Sync delivery写永久墓碑。登录/API Key开关变化不触发产品撤权；重授不清墓碑、不自动resume。

```sql
UPDATE account_notification_deliveries
SET eligibility_revoked_at=COALESCE(eligibility_revoked_at,clock_timestamp()),
    eligibility_revoked_reason=COALESCE(eligibility_revoked_reason,$2),
    status=CASE WHEN status='pending' THEN 'cancelled' ELSE status END
WHERE account_id=$1 AND source='trader_sync';
```

未冻结摘要资格由任务10与其表一起接入RevokeTx；当前任务只操作已经存在的订阅、区间及delivery，不能引用未来尚未创建的表。任务10上线前尚无活动或摘要业务入口。sending原attempt结果照实记录，明确失败因墓碑cancelled。hook注入是必需启动依赖，缺失时server启动错误，不能静默漏撤权；使用fake registrar验证当前任务，不提前实现采集。

- [x] **步骤5：生成权限proto并适配实际UI矩阵。**AccountDataModule使用空闲enum值12（6、10已reserved），不是因十模块而占用10；拆开grant校验和requirement校验。稳定后 `make protogen`，适配Go映射及UI共享allowedAccessLevels。非法READ从解析端fail-closed为NONE，绝不提升RW；选择器只NONE/RW，十项序列化/克隆/比较/概览一致。业务页面与导航由任务15–19交付，本任务不挂空路由。

```ts
// access-modules.ts共享允许值；其他模块沿用已有maxAccess逻辑。
if (module.id === 'trader_sync') {
  return [AccountDataAccess.None, AccountDataAccess.ReadWrite]
}
```

共享函数命名为`allowedAccessLevels(module: AccountDataModuleDefinition): AccountDataAccess[]`，使用现有AccountDataAccess枚举。测试不仅检查选项，还检查parse→edit→serialize十项往返与RW→NONE缓存清理。

- [x] **步骤6：运行 `go test ./internal/accountaccess ./internal/accountstate/... ./internal/tradersync/... ./internal/server/account`、上述真实数据库集成测试和 `yarn --cwd ui test --runInBand --watch=false`。**提交 `feat(trader-sync): manage isolated subscriptions and revocation`。

## 任务7：三Exchange成交解码、规范链确认与版本证据

**Files**
- 新增：`internal/tradersync/exchange_decode.go`、`internal/tradersync/confirmation.go`、`internal/tradersync/source_version.go`、`internal/tradersync/abi/embed.go`、`internal/tradersync/abi/core_exchange.json`、`internal/tradersync/abi/combos_exchange.json`、`internal/tradersync/abi/proxy.json`、`internal/tradersync/testdata/source_records.json`。
- 新增测试：`internal/tradersync/exchange_decode_test.go`、`internal/tradersync/confirmation_test.go`、`internal/tradersync/source_version_test.go`；新增类型`internal/tradersync/types/trade.go`。
- 新增薄适配器：`internal/tradersync/source_rpc.go`、`source_rpc_test.go`；借用调用方持有的 ethclient，端点替换时重建实例，关闭由调用方负责。
- 依据：[已核准来源契约与部署值](../../requirements/polymarket-copy-trading/source-contract-verification.md)。ABI资源记录来源仓库commit及hash，不新增或修改`pkg/abi/**/*.sol`。
**Interfaces**
- 产出tsmodel：`Trade{Wallet common.Address; Side,PositionID,CollateralRaw,SharesRaw,FeeRaw,CollateralSymbol string; CollateralDecimals,SharesDecimals uint8; Exchange common.Address; SourceVersion,PriceNumerator,PriceDenominator string}`；`CanonicalEvidence{Status string; BlockHash common.Hash; SettledAt time.Time; CheckedAt time.Time; Reason string}`；Status=unverified/invalid/confirmed。
- 产出：`DecodeOwnTrade(log types.Log,version string) (tsmodel.Trade,error)`；`ConfirmReceived(ctx context.Context,node CanonicalRPC,raw types.Log) (tsmodel.CanonicalEvidence,error)`；`(*VersionVerifier).Verify(ctx context.Context,raw types.Log) (version string,err error)`。
- CanonicalRPC精确契约：`FinalizedHeader(context.Context) (*types.Header,error)`、`TransactionReceipt(context.Context,common.Hash) (*types.Receipt,error)`、`HeaderByHash(context.Context,common.Hash) (*types.Header,error)`、`HeaderByNumber(context.Context,*big.Int) (*types.Header,error)`；fake按调用记录，不请求真实链。
- `FinalizedHeader`先保证当前实例连接 chain 137，再取 finalized；正面链证明仅在同不可变端点实例内缓存，失败不永久缓存。`SourceRPC`单一拥有成功头的2秒缓存与可取消并发合并，过期懒刷新且失败不使用旧头；不启动独立 poller。任务10 ActivityProjector的2秒工作调度消费此入口，不维护第二份 finalized 缓存。每次真实 RPC 截止5秒，回环测试覆盖错链、并发、取消、过期失败和真实超时。
- `ConfirmReceived`在读取失败或错链时返回 unverified 证据、稳定 Reason 和底层 error；调用方保留原始候选再报告错误，取消只结束本轮。removed、完整回执的明确失败、明确规范定位或字节不一致才返回 invalid；不能根据另一链的头否定 Polygon 候选。

- [x] **步骤1：固定已核准原始日志fixtures，写BUY/SELL及自身归属红灯。**fixtures逐项存address/topics/data/block/tx/logIndex及预期wallet/方向/原量/fee，至少三Exchange×BUY/SELL×Maker/Taker；同tx两个自身日志必须两个Trade；只topics[3]匹配的目标不得被归属该Trade（纯解码器只取topics[2]，目标过滤由任务9验证），OrdersMatched及SPLIT/MERGE不得产出Trade。

```go
func TestUnknownExecutionVersionIsNotDecoded(t *testing.T) {
    _,err:=DecodeOwnTrade(types.Log{},"unrecognized-implementation")
    if err==nil { t.Fatal("decoded unknown execution version") }
}
```

运行 `go test ./internal/tradersync -run 'TestDecode|TestUnknownExecution' -count=1`确认红灯。fixture构造必须按来源ABI编码，不能把decoder输出当预期再反序列化。现存真实记录缺Combo SELL maker，该格允许明确标注按固定ABI构造的synthetic，其余11格保留原始日志；不宣称十二格均完成真实来源核验。任务13在有界实时验收中继续补证，未观察到时明确保留缺口。

- [x] **步骤2：实现白名单版本解码及精确单位。**先核对chain/address/topic0/version，再按真实ABI解码自身资金钱包topics[2]、side、position及filled/fee原整数。BUY的抵押币/份额分别为maker/taker，SELL反转；6位pUSD与6位份额作为已核准版本属性保存，费用独立。无大于0的自定义门槛；零值合法性按合约事件语义处理，不能新增最小金额过滤。

```go
collateral,shares:=makerAmount,takerAmount
if side=="SELL" { collateral,shares=takerAmount,makerAmount }
trade.CollateralRaw=collateral.String()
trade.SharesRaw=shares.String()
trade.FeeRaw=feeAmount.String()
```

片段变量来自ABI解码后的`*big.Int`，trade为`tsmodel.Trade`；symbol/decimals从本次version注册项取得，不能从API字段名usdcSize推断。价格保存未含fee的collateral/shares精确比值（两个字符串），分母为0则价格不可用，不造0价格，不阻止保存可识别成交；当前两者decimals均6，未来版本按精度换算。

- [x] **步骤3：实现确认后回执和已知哈希验证。**共享fresh finalized可覆盖候选后重新取known tx receipt；status=成功、规范高度头hash、receipt定位和原日志address/topics/data/index均一致，才取HeaderByHash时间。null/403/timeout或回执必需字段缺失返回unverified；适配器区分missing/null与合法零值，不能让JSON默认值变成明确负证据。removed、完整回执明确失败或明确不同规范定位返回invalid。receipt其他日志仅核对，不建新候选。fake构造receipt有两条日志时只返回被请求原日志的确认事实，并断言先finalized后receipt。

```go
if raw.Removed { return tsmodel.CanonicalEvidence{Status:"invalid",Reason:"removed"},nil }
// 在finalized覆盖且receipt成功之后执行逐字节核验。
same:=received.Address==raw.Address && received.Index==raw.Index &&
    reflect.DeepEqual(received.Topics,raw.Topics) && bytes.Equal(received.Data,raw.Data)
if !same { return tsmodel.CanonicalEvidence{Status:"invalid",Reason:"canonical_log_changed"},nil }
```

- [x] **步骤4：加入按blockHash的版本证据缓存。**HTTP adapter用StorageAtHash/CodeAtHash读取已知候选块及父块代理/实现，核对已核准升级记录语义；仅允许候选known blockHash的升级事件查询，不用latest槽证明旧执行。不认识实现或无法排除同块升级保持unverified，不能用块末实现盲解。该能力通过窄接口`VersionRPC`定义`ChainID(context.Context) (*big.Int,error)`、`HeaderByHash(context.Context,common.Hash) (*types.Header,error)`、StorageAtHash、CodeAtHash及`UpgradeLogs(ctx context.Context,blockHash common.Hash,proxy common.Address) ([]types.Log,error)`；先核实节点chain 137，再从已知候选头取得ParentHash，不以配置常量或按高度猜父块代替；不提供任意from/to扫描接口。

```go
if parentImplementation!=blockImplementation || len(upgrades)>0 || !upgradeEvidenceComplete {
    return "",status.Error(codes.FailedPrecondition,"execution version unverified")
}
```

缓存键至少chain/exchange/blockHash；无法确认结果不当作永久无升级。CombinatorialModule资料版本失败归任务8metadata缺失，不阻塞已确认成交。

- [x] **步骤5：运行 `go test ./internal/tradersync -run 'TestDecode|TestConfirm|TestVersion|TestUnknownExecution' -count=1`。**覆盖旧hash/新规范块、403/null、removed、已确认后深度重组隔离所需证据、同块升级/未知实现；提交 `feat(trader-sync): verify and decode canonical source trades`。

## 任务8：普通市场、Combo逻辑及持久目录资料

**Files**
- 新增：`internal/tradersync/market_metadata.go`、`internal/tradersync/combo_metadata.go`、`internal/tradersync/abi/combinatorial_module.json`、`internal/tradersync/abi/binary_module.json`、`internal/tradersync/store/metadata.go`、`internal/tradersync/store/queries/metadata.sql`、`util/polymarket/combo_markets.go`。
- 新增类型：`internal/tradersync/types/metadata.go`。
- 修改：`util/polymarket/gamma.go`、权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/tradersync/abi/embed.go`、`internal/tradersync/source_rpc.go`及其测试。
- 测试：`internal/tradersync/market_metadata_test.go`、`internal/tradersync/combo_metadata_test.go`、`internal/tradersync/store/metadata_integration_test.go`、`util/polymarket/combo_markets_test.go`。
**Interfaces**
- 产出tsmodel：`MarketRef{Evidence; ID,Title,URL,ConditionID,PositionID,Outcome string}`；`ComboLeg{PositionID string; Market MarketRef}`；`TradeMetadata{Market MarketRef; LegsEvidence Evidence; Legs []ComboLeg; Relationship string}`。腿数仅LegsEvidence.available时由len(Legs)取得。
- 产出：`(*MetadataResolver).Resolve(ctx context.Context,trade tsmodel.Trade,blockHash common.Hash) tsmodel.TradeMetadata`；`(*DirectoryRefresher).Run(ctx context.Context) error`。
- 资料 RPC 复用任务7的 SourceRPC，增加窄方法 `CallContractAtHash(context.Context,ethereum.CallMsg,common.Hash) ([]byte,error)`，只调用已知 hash 并沿用5秒截止；模块版本核验复用同包 deployment helper，先核实 chain/known 头/真实父 hash。模块 ABI 纳入既有 embed，不加入 Exchange 解码白名单。
- 产出公开适配：`ComboMarketPage{Markets []ComboMarket; NextCursor string}`、`ComboMarket{ID string; PositionIDs []string; ConditionID string}`；`(*GammaClient).ListComboMarkets(ctx context.Context,cursor string,limit int) (ComboMarketPage,error)`；在util/polymarket定义这些类型。

- [x] **步骤1：写组合逻辑和缺失整腿红灯。**

```go
func TestComboNoMeansComplementOfConjunction(t *testing.T) {
    got:=ComboRelationship("NO")
    if got!="NOT(AND(legs))" { t.Fatal(got) }
    if ComboRelationship("YES")!="AND(legs)" { t.Fatal("wrong YES relationship") }
}
```

定义`ComboRelationship(outcome string) string`，只对已核验YES/NO返回该表达式，其余未知为空并附reason。用HTTP/RPC fake返回getLegs失败，断言LegsEvidence=unavailable而不是available空数组/0；运行 `go test ./internal/tradersync ./util/polymarket -run 'TestCombo|TestMarket' -count=1`。

- [x] **步骤2：实现精确市场与两类腿映射。**普通token查询考虑open/closed，并按精确token下标取Outcome。迁移Binary腿执行legacyConditionId/getLegacyPositionId→旧CTF token→Gamma；其他已核准module腿用目录PositionId→market ID，再GetMarketByID核对positionIds、condition、Outcome，不以名字或当前持仓猜测。PositionID保留十进制字符串，Gamma类型增加有presence的positionIds解析。

```go
func ComboRelationship(outcome string) string {
    switch outcome { case "YES": return "AND(legs)"; case "NO": return "NOT(AND(legs))" }
    return ""
}
```

getLegs在已知块hash调用并保留module版本证据；metadata错误不改变TRADE方向/金额。已知N腿部分缺失时仍保留N条位置及各自状态，关系针对组合自身Outcome；不是每腿分别BUY/SELL。

module1腿的legacy无值/读取失败时，可用已持久目录加Gamma作为独立精确资料路线，Evidence.Source标明实际来源，不推断native或宣称链上版本/映射成功。legacy链上调用本身先核验Binary版本；Combinatorial版本/getLegs失败不能由目录猜出整腿，已取得的明确冲突不能用另一条路线掩盖。

- [x] **步骤3：实现有界分页目录和资料缓存。**新增`trader_sync_market_metadata`、`trader_sync_combo_leg_index`、`trader_sync_directory_refresh`；索引以position_id/market_id核验关系持久保存，不能刷新时删除已关闭记录。每页最多100，当前cursor/round_started_at/next_page_at在同事务推进；失败保留cursor，最多1page/s，10分钟轮次不重叠。

目录索引保存外部声明的待核验关系，不直接授予metadata available；缓存缺失时按精确market ID查询Gamma并核对positionIds/condition/Outcome，成功的资料及实际Source另存market_metadata，冲突明确unavailable。

```sql
UPDATE trader_sync_directory_refresh
SET cursor=$2,next_page_at=clock_timestamp()+interval '1 second'
WHERE name='combo_markets' AND cursor=$1;
```

该CAS与本页映射upsert同事务；最后一页另保存round_completed_at和下轮时刻。专属refresh行锁可覆盖一个最多5秒的HTTP页请求，失败保留cursor且提交至少1秒后的next_page_at，不能持account/wallet gate等网络；请求已发出后的父上下文取消也需有界收尾保存节流，再释放行锁。真实两连接验证不会并发请求页面及取消后抢跑，事务中途写失败则整页映射与cursor回滚。稳定SQL后 `make sqlc-local`。并发资料默认4；热路径缓存miss返回可见缺失，不能遍历目录阻塞活动。Resolve按调用方ctx取消、单次HTTP/RPC最多5秒，不硬设2秒总截止；任务10负责最终确认后额外最多2秒的活动等待，后台补资料使用自身有界上下文。

- [x] **步骤4：运行本任务单元/集成测试。**覆盖两迁移Binary腿、module1/2、已关闭市场、空持仓、目录分页重启/失败/重复页、超大ID及unknown Outcome；验证晚补资料只变显示数据。提交 `feat(trader-sync): resolve market and combo metadata precisely`。

## 任务9：注册先于过滤的基线、共享WSS与持久接收

**Files**
- 新增：`internal/tradersync/baseline.go`、`internal/tradersync/collector.go`、`internal/tradersync/intake.go`、`internal/tradersync/config.go`、`internal/tradersync/rpc/session.go`、`internal/tradersync/rpc/http.go`、`internal/tradersync/store/intake.go`、`internal/tradersync/store/baselines.go`、`internal/tradersync/store/collector_session.go`、`internal/tradersync/store/queries/intake.sql`、`internal/tradersync/store/queries/baselines.sql`。
- 新增类型：`internal/tradersync/types/candidate.go`。
- 修改：权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/tradersync/source_rpc.go`及`source_rpc_test.go`；以`util/ethws/dial.go`的真实实现作核对入口，不复用其自动重连语义。
- 测试：`internal/tradersync/baseline_test.go`、`internal/tradersync/collector_test.go`、`internal/tradersync/collector_integration_test.go`、`internal/tradersync/rpc/session_test.go`、`internal/tradersync/store/intake_integration_test.go`、`internal/tradersync/store/baselines_integration_test.go`。
**Interfaces**
- 消费：任务6 `BaselineRegistrar.RegisterTx`，任务7SourceRPC的latest `HeaderByNumber`窄能力；原始接收不依赖metadata。最终确认的2秒工作调度、版本核验及候选证据更新由任务10的Projector统一负责。
- `SourceRPC.HeaderByNumber`先复用已有ChainID正面缓存/并发合并核验Polygon 137再读取；失败不永久缓存。WSS与HTTP端点分别核验，不能用前者证明后者，真实HTTP回环覆盖错误链不读头及恢复。Session在标准Log解码前保留必需定位字段的presence，缺失/null不能变成合法零位置；完整零值仍合法。
- 产出：`ComputeBaseline(now time.Time,registeredHigh uint64,h *types.Header) (time.Time,error)`；`(*Collector).Run(ctx context.Context) error`；`(*Intake).Persist(ctx context.Context,epoch uint64,received tsmodel.ReceivedLog) error`。
- 产出rpc：`DialSession(ctx context.Context,endpoint,proxyURL string) (*Session,error)`；`(*Session).Subscribe(ctx context.Context,query ethereum.FilterQuery) (subscriptionID string,error)`；`Unsubscribe(ctx context.Context,id string) error`；`Logs() <-chan tsmodel.ReceivedLog`；钱包观察快照方法；`Done() <-chan struct{}`；`Err() error`；`Close() error`。`ReceivedLog{Raw types.Log; ReceivedAt time.Time; Sequence uint64}`在read loop内记录实际接收时刻及会话序号，再进入有界队列；钱包快照在同一临界区取得ObservedHigh与ReadSequence。每个Session仅一物理WSS，绝不内部重连。
- 产出tsmodel：`Candidate{SourceID int64; SubscriptionID,OwnerID string; Generation uint64; AttemptID string; ReceivedAt time.Time}`；`Eligibility{Generation uint64; BaselineSucceeded bool; SettledAt,EffectiveAt time.Time; EndedAt *time.Time}`。`Eligible(c tsmodel.Eligibility,s tsmodel.Subscription) bool`只判原区间+当前意图/代次，不判当前epoch健康。

- [x] **步骤1：写秒级边界红灯测试。**

```go
func TestBaselineIsNextSecondAndNotBelowObservedHead(t *testing.T) {
    now:=time.Unix(100,800_000_000)
    h:=&types.Header{Number:big.NewInt(10),Time:100}
    at,err:=ComputeBaseline(now,10,h)
    if err!=nil || !at.Equal(time.Unix(101,0)) { t.Fatal(at,err) }
    if _,err=ComputeBaseline(now,11,h); err==nil { t.Fatal("accepted older head") }
}
```

运行 `go test ./internal/tradersync -run TestBaseline -count=1`。补充滞后10秒/超前2秒精确端点、同秒结算、等待前保存已过期重算和最新revision竞争。

- [x] **步骤2：实现基线纯函数和持久状态协议。**

```go
func ComputeBaseline(now time.Time,registeredHigh uint64,h *types.Header) (time.Time,error) {
    if h==nil || h.Number==nil || !h.Number.IsUint64() || h.Number.Uint64()<registeredHigh {
        return time.Time{},fmt.Errorf("head below registered observation")
    }
    settled:=time.Unix(int64(h.Time),0)
    if settled.Before(now.Add(-10*time.Second)) || settled.After(now.Add(2*time.Second)) {
        return time.Time{},fmt.Errorf("head clock outside bounds")
    }
    at:=now.Truncate(time.Second).Add(time.Second)
    if candidate:=settled.Add(time.Second); candidate.After(at) { at=candidate }
    return at,nil
}
```

RegisterTx在account→wallet gate内记录attempt与当前已观察最高高度/会话注册序号，安装过滤前完成；最高高度包含本会话read loop已解出而尚在持久化队列中的该钱包日志，不能只看已写DB高度。候选只匹配原epoch中注册截点之后的接收序号，不给后来注册的订阅追领旧排队记录；不得在消费者出队时补造接收时刻/序号。Registrar只在传入事务登记，Collector读到已提交pending后才安装过滤，不新增after-commit回调或提前发布内存成员。未连接attempt保存pending，不能沿用失败attempt。全部相关过滤ACK后新查latest、保存候选未来边界，等到边界再同account gate复核grant/revision/generation/epoch并CAS succeeded+interval。提交结果未知先读真实状态，不能把可能已成功attempt覆盖为failed。已确定未成功的重启attempt终结并新建。

取得collector所有权时，在control锁下快照已提交且epoch为空的pending attempt ID；换代提交后逐account→wallet→fence仅终结这些原ID并重新建立实时attempt。取得所有权之后的新登记不受影响；暂停、撤权或新generation仍优先。此截点也适用于首次启动前既有未绑定登记，原因不声称此前已健康观察或确有遗漏。

- [x] **步骤3：实现显式WSS会话与共享过滤。**用现有gorilla/websocket写一条read loop及串行JSON-RPC写队列，request ID路由ACK，notifications推送有界队列；15秒发送随机关联ping，5秒内只有同nonce pong才成功。队列满/DB无法持久接收关闭epoch并显示可能遗漏，不静默丢日志。HTTP latest成功不得覆盖WSS pong失败，无匹配日志保持健康。Session Done独立取消阻塞的账户/ACK等待，关闭epoch不等待逐账户收尾完成。已发无ACK占满64个pending后，后续控制请求明确关闭Session交外层恢复；不能因永久占槽而原地无限失败，也不能复用旧请求ID。

```go
// 普通CTF/NegRisk共享一组，Combos独立一组；walletTopics每组<=100。
filter:=ethereum.FilterQuery{
    Addresses:coreAddresses,
    Topics:[][]common.Hash{{orderFilledTopic},nil,walletTopics},
}
```

地址/topic来自任务7版本注册；只topics[2]筛目标。目标集合全局去重，新增过滤ACK后删旧过滤；新增目标失败不拆已有有效过滤、不重建其他用户边界。外层Collector按配置退避重连、建立新epoch/实时边界；专属PG session advisory lock负责单实例所有权，新增collector control行保存单调fencing token，每次写入在同事务以行锁/CAS与换代串行，过时代次拒绝写入。锁序为account→wallet（需要时）→fence，intake只wallet→fence；换代/关闭epoch事务先独立提交，再逐account收尾，不持fence反向等account。失锁取消会话，不套用Telegram sender的外部停止确认CLI。StartEpoch COMMIT未知时有界读回同token/active epoch/未结束状态，不能确认则fatal退出并保留清理错误，不能在同owner无限重连。

- [x] **步骤4：实现raw第一次落库与候选归属事务。**新增`trader_sync_collector_epochs`、`trader_sync_interruptions`、`trader_sync_source_records`、`trader_sync_source_candidates`；source定位五元唯一。钱包gate内先验证epoch fencing，再插raw+首次候选；重复raw不能添加后来注册的订阅，removed=true必须更新证据，包括最后enabled订阅已停用但既有source更新已收到的情况；该更新不改原raw/时刻/序号或补建候选，首次插入仍受当前目标约束。baseline失败的候选保留原attempt，不能归入新attempt。

```sql
INSERT INTO trader_sync_source_records
  (chain_id,exchange_address,block_hash,transaction_hash,log_index,raw_json,received_at,removed)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (chain_id,exchange_address,block_hash,transaction_hash,log_index)
DO UPDATE SET removed=trader_sync_source_records.removed OR EXCLUDED.removed
RETURNING id;
```

是否首次必须由独立INSERT DO NOTHING RETURNING或显式事务查询判定，不能凭upsert返回ID为每次补候选。source永久原事实与可变确认/removed证据分列；失败回滚意味着未可靠接收，记录中断。稳定SQL后 `make sqlc-local`。

- [x] **步骤5：运行WSS和两连接集成故障矩阵。**loopback WSS在ACK前后推日志，断言注册无空窗；错误pong/静默连接/队列满/DB断流触发可见中断。本任务用spy证明Collector不调用finality、receipt或版本读取；任务10/13组合验证单次finality失败仅积压、WSS不关。恢复后只收新推送，旧成功attempt持久候选仍可处理，failed attempt不可复活。Collector拒绝任何历史范围扫描和从receipt其他日志造候选；Projector需要的known-block升级查询由任务10验收。

```go
func Eligible(c tsmodel.Eligibility,s tsmodel.Subscription) bool {
    return s.DesiredState=="enabled" && s.Generation==c.Generation && c.BaselineSucceeded &&
        !c.SettledAt.Before(c.EffectiveAt) && (c.EndedAt==nil || c.SettledAt.Before(*c.EndedAt))
}
```

执行 `go test -race ./internal/tradersync/...` 及 `go test -tags=integration ./internal/tradersync/store -run 'TestIntake|TestBaseline' -count=1`；提交 `feat(trader-sync): collect live trades with durable observation boundaries`。

## 任务10：按owner形成活动、快照和普通/摘要资格

**Files**
- 新增：`internal/tradersync/projector.go`、`internal/tradersync/store/activities.go`、`internal/tradersync/store/queries/activities.sql`、`internal/tradersync/types/activity.go`、`internal/tradersync/render.go`、`internal/tradersync/render_test.go`、`internal/notification/store/transactional_enqueue.go`。
- 修改：权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/tradersync/store/revocation.go`、`internal/notification/store/queries/account_notifications.sql`、`internal/notification/store/queries/delivery_attempts.sql`、`internal/notification/store/attempts.go`。
- 测试：`internal/tradersync/projector_test.go`、`internal/tradersync/store/activities_integration_test.go`、`internal/notification/store/transactional_enqueue_integration_test.go`。
- 最早纯文本消费者适配：共享通知载荷/必要schema-query、Sender/SendRequest及util Telegram实际调用明确保存并消费消息格式；Trader Sync普通纯文本、既有HTML消费者显式保留HTML。格式与正文冻结，摘要任务11复用。
- 共享纯规则：新增`internal/tradersync/activity/`叶子包（资格/分类/纯文本渲染及测试），同步`baseline.go`与上述公开入口为薄调用。store和tradersync共用唯一算法，types保持数据结构；该包无IO、不import store或notification业务根包；摘要可复用notification/delivery的无IO冻结codec。
- 目标展示接续：同步订阅创建、confirmation/订阅SQL及共享types，在成功Create同事务持久保存小型公开展示快照，新增真实集成测试；不从过期确认记录临时推导。
**Interfaces**
- 消费：任务7确认/版本/Trade、任务8TradeMetadata、任务9Candidate/Eligible、任务1account gate。
- 产出tsmodel：`Activity{ID int64; OwnerID,SubscriptionID string; SourceID int64; Trade Trade; Metadata TradeMetadata; TargetDisplaySnapshot TargetDisplay; NoteSnapshot,NotificationMode,NotificationReason string; SettledAt,ReceivedAt,RecordedAt time.Time; Generation uint64}`；`Projection{Candidate Candidate; Trade Trade; Confirmation CanonicalEvidence; Metadata TradeMetadata}`。NotificationMode在形成事务中固定in_app_only/ordinary/summary；未绑定原因unbound_at_formation不因未来绑定改变。
- 产出：`(*Projector).Run(ctx context.Context) error`；`(*tradersyncstore.SQLStore).Project(ctx context.Context,input tsmodel.Projection) (activityID int64,created bool,err error)`；`ClassifyActivity(windowCount int64) string`返回ordinary/summary。Projector窄构造依赖store、CanonicalRPC、VersionVerifier、共享MetadataResolver及siteURL；Run拒绝重复启动，取消并join全部有界工作；MaxInFlightSources默认100且独立可配置，包含晚补job，超量source持久等待并公平选择到期重试，不因最老失败记录而饿死后续正常source。SQLStore的Project必需配置缺失明确失败，不影响其独立读取能力。
- 产出`TargetDisplay{DisplayName,Avatar,ProfileURL Scalar}`，Subscription增加TargetDisplay；成功Create精确读取同owner/token卡片及身份digest后保存少量确认时展示值和查询证据，Activity形成时复制快照。DisplayName采用card.Scalar；非空备注优先，其次available名称、最后钱包识别，名称缺失仍unavailable。ProfileURL仅核验canonical非空可用，地址输入缺值不猜链接。暂停恢复保留确认时资料，重新订阅可得新快照；不复制PNL、无周期刷新或全局缓存，钱包身份及历史消息不变。
- 产出通知事务入口：`(*notificationstore.SQLStore).EnqueueAccountTx(ctx context.Context,tx pgx.Tx,in delivery.AccountEnqueue) (int64,error)`。`AccountEnqueue{OwnerID,Source string; ActivityID int64; BindingRevision uint64; ChatID int64; Payload []byte; RecordedAt time.Time}`放delivery叶子包；调用方已在同事务核验并冻结绑定，不二次读取新绑定。source='trader_sync'，其他有效来源不加Trader Sync grant。
- 产出schema固定：`trader_sync_alert_memberships(activity_id bigint PK,owner_id,binding_revision,chat_id,form,state,created_at,eligibility_revoked_at,reason,batch_id nullable)`。form仅ordinary/summary；unbound不插membership。ordinary同事务建delivery；summary只建资格，任务11冻结。summary phase由state/batch映射waiting/frozen/cancelled_before_freeze，冻结前终止保留reason且不创建空batch；已有batch部分终止仍为frozen。owner与activity复合FK，批次FK在任务11添加。

- [x] **步骤1：写第10/11条与窗口端点红灯测试。**

```go
func TestClassificationRetainsFirstTenOrdinary(t *testing.T) {
    for i:=int64(1); i<=12; i++ {
        want:="ordinary"; if i>10 { want="summary" }
        if got:=ClassifyActivity(i); got!=want { t.Fatal(i,got) }
    }
}
```

真实DB测试用同owner记录时间t-60秒、t-60秒+1微秒和当前t，断言左端排除、右端包含；同秒用ID稳定顺序。运行 `go test ./internal/tradersync -run TestClassification -count=1` 和新集成测试确认红灯。

- [x] **步骤2：实现活动/外发资格同事务和持久唯一约束。**新增`trader_sync_activities`，(subscription_id,source_record_id)唯一，owner/订阅/原interval引用一致，trade_json与note_snapshot不可变，metadata展示引用独立。取得account gate后重查数据库grant、当前enabled、generation及成功原区间；原epoch关闭不失去资格。普通消息渲染入口定义`RenderActivity(activity tsmodel.Activity,siteURL string) (string,error)`，在`internal/tradersync/render.go`实现，输出纯文本：目标备注/名称和完整wallet、TRADE及BUY/SELL、市场/Outcome/精确价格比值、份额/金额/币种/独立fee、明确“结算时间”、市场链接及站内详情。缺市场保留PositionID与缺失原因；Combo标明组合自身Outcome、整体关系及腿资料状态，完整腿在站内详情保留。长标题允许显式省略显示字符但保留身份与完整链接，不静默截断交易事实。本任务同步移除Sender对所有消息强制HTML的假设：显式格式随实际规范文本持久冻结并进入不可变发送证据，纯文本请求不设parse_mode。真实回环测试覆盖特殊字符/emoji/完整链接与旧HTML消费者；不在重试时重新渲染或改变冻结文本。

```sql
CREATE SEQUENCE trader_sync_activity_id_seq AS bigint INCREMENT BY 1 CACHE 1 NO CYCLE;
-- 建表id为bigint PRIMARY KEY DEFAULT nextval('trader_sync_activity_id_seq')，再设置归属。
ALTER SEQUENCE trader_sync_activity_id_seq OWNED BY trader_sync_activities.id;
-- 在gate内先确认非重复，读取clock_timestamp()作为唯一recorded_at=t。
-- 对既有窗口count+1，再一次INSERT最终mode；等价于包含本次活动的计数。
SELECT count(*) FROM trader_sync_activities
WHERE owner_id=$1 AND recorded_at>$2::timestamptz-interval '60 seconds'
  AND recorded_at<=$2;
```

窗口计数包含本次活动、未绑定活动和所有目标；实现可在同gate/同一DB时刻先统计既有窗口并加当前1，再INSERT最终mode，避免修改冻结事实。重复插入不重复计数/规划资格；无当前binding则只活动，有binding则固定revision/chat与备注。暂停/取消和外发分开：旧活动ordinary/summary继续，尚未投影候选终止。插入到Commit耗时计入延迟，成功Commit才通知后续worker。

activity ID只在持有owner gate的INSERT内分配，不先nextval预取，不回拨或回填；用索引(owner_id,id DESC)支持读取。测试查询pg_sequences确认cache_size=1、increment_by=1、cycle=false；两连接同owner串行形成、不同owner交错、回滚空号与recorded_at回退时，ID及snapshot仍正确。时间回退测试在隔离DB直接构造既有行的时间反序来验证读取，不改主机/生产数据库时钟。保留recorded_at用于滚动统计和时效，不用ID差值推算活动数量。

- [x] **步骤3：实现并行资料预算、重组隔离及晚资料更新。**metadata与finality并行；确认完成后最多额外2秒，超时保留字段unavailable并形成活动。未知Exchange执行版本仍raw/unverified；不是metadata超时理由。后台补资料仅更新可变metadata，不修改备注、原量、payload、不追加通知。

本步骤同时接续已有market_metadata/combo_metadata解析器、types/metadata、store/metadata及对应查询/测试：增加唯一进度消费入口，getLegs核验后先初始化全部N个PositionID与逐腿unavailable，再发布深拷贝快照。两秒到点直接读取已知部分，无需等待RPC退出；原job继续晚补，默认总截止30秒（可配置资源参数），Run取消并等待退出。前后台共用同一Resolver的4并发，不复制解析算法或重新计活动两秒预算。

同证据键的缓存/显示资料按非冲突证据合并，暂时失败/取消及并发晚到的较差结果不清空已知N腿和已核验部分；明确condition/position/来源冲突仍unavailable，不能按available数量或旧成功掩盖。必要合并在短DB事务内完成，不跨网络拿锁；不跨SourceVersion/PositionID/Combo knownHash合并。相关metadata SQL如需调整，纳入本任务稳定生成批次。

```go
func ClassifyActivity(windowCount int64) string {
    if windowCount<=10 { return "ordinary" }
    return "summary"
}
```

确认后深度重组以chain+tx证据建立`trader_sync_finality_anomalies`；在投影新分叉前检查曾发布该tx的冲突证据，隔离该tx后续新分叉候选，原activity保留异常；仅removed而替代块未知时conflicting_block_hash允许NULL，不能填零或原hash冒充新分叉。不能仅以txHash/logIndex跨分叉去重，也不能把同一规范tx多个自身日志合并。后续真实接收并确认同交易的不同区块时，可在同交易证据锁下仅补首次未知的冲突哈希；保留首次原因和检测时间，已有冲突哈希不覆盖，不恢复被隔离活动或生成新通知。

- [x] **步骤4：稳定活动/通知SQL后 `make sqlc-local`，适配EnqueueAccountTx和RevokeTx。**事务内永久撤销所有旧summary membership；解绑/重绑亦终结旧revision。绑定不可达的发送结果及Bot更新沿同一旧revision终止资格，重新connected不复活；暂停/取消不撤销已形成队列。最终Authorize在既有owner gate内复核Trader Sync grant、冻结binding及membership；无资格返回eligible=false而不让结果查询丢行，保证先前已许可的sent/unknown仍能照实落库。不得从notification反向调用会停订阅的RevokeTx；各模块SQL在同一传入事务内写资格，避免循环依赖和重入account gate。测试两个owner共享同一source得到两份独立活动/备注/资格，而owner内部同source只有一份。
- [x] **步骤5：运行 `go test -race ./internal/tradersync/... ./internal/notification/...` 和对应真实DB测试。**覆盖投影/暂停/撤权/重绑屏障竞争、activity插入后事务回滚、无binding后再绑定不补发、取消旧队列继续、newgeneration丢旧候选、100关系及全不同目标；提交 `feat(trader-sync): persist activities and alert eligibility atomically`。

## 任务11：不可变完整摘要、分条结果与首条gate

**Files**
- 新增：`internal/tradersync/summary.go`、`internal/tradersync/types/summary.go`、`internal/tradersync/store/summaries.go`、`internal/tradersync/store/queries/summaries.sql`、`internal/notification/store/summary_heads.go`、`internal/notification/summary_source.go`。
- 修改：`internal/accountstate/txgate/gate.go`、`internal/notification/store/attempts.go`、`internal/notification/source.go`、`internal/notification/worker.go`、`internal/notification/budget.go`、权威`internal/accountstate/store/migrations/000001_init.sql`。
- 测试：`internal/tradersync/summary_test.go`、`internal/tradersync/store/summaries_integration_test.go`、`internal/notification/summary_source_integration_test.go`、`internal/notification/summary_dispatcher_integration_test.go`、`internal/accountstate/txgate/session_integration_test.go`。
- 真实消费者接续：`internal/tradersync/activity/`纯摘要渲染、`internal/notification/delivery/payload.go`及摘要part窄事务入队、`notification/store/queries/delivery_attempts.sql`的summary关联资格、account ready查询排除首条抢领；同步同次permit发送核心、budget非阻塞准入与service/recovery的摘要head接续及对应测试/生成物。notification/store不得反向import tradersync/store或根包。
**Interfaces**
- 消费：任务4 `WorkSource`、任务2 `Permit`/Sender实际started回调、任务10memberships。
- 产出：`SummaryWindow(oldest time.Time,previousStart *time.Time) (notBefore,deadline time.Time)`；`RenderSummary(items []tsmodel.Activity,batchID string,siteURL string) ([]tsmodel.RenderedPart,error)`；`RenderedPart{Index,Total int; Text string; ActivityIDs []int64; PayloadDigest []byte}`。
- 产出：`txgate.AcquireAccountSession(ctx context.Context,pool *pgxpool.Pool,ownerID string) (*AccountSession,error)`；`AccountSession{Conn *pgxpool.Conn}`；`(*AccountSession).Release(ctx context.Context) error`。同一账户键与普通事务锁相互排斥。
- 产出：`(*notificationstore.SQLStore).AuthorizeTx(ctx context.Context,tx pgx.Tx,candidate delivery.Candidate,incarnation uuid.UUID,guard func() error) (delivery.Permit,error)`；任务2普通Authorize包装account事务并复用它，摘要在session固定连接内BeginTx调用，不能持session锁后向另一个连接请求account锁。Candidate的实际chat/group路由核验与行锁后的guard保留；预算finish覆盖外层真实Commit/Rollback，不在AuthorizeTx返回时提前解锁或假定提交成功。
- 产出：`(*SummarySource).Ready(ctx context.Context,now time.Time) ([]delivery.Candidate,error)`；`Dispatch(ctx context.Context,c delivery.Candidate,onStarted func(time.Time)) error`，实现WorkSource；摘要首条ready给出未来notBefore/deadline，后续部分普通许可。
- 摘要part入队采用独立`EnqueueSummaryPartTx`窄事务入口及稳定batch/part幂等身份，保持activity_id=NULL、source=trader_sync、WorkKind=account；在最终许可SQL以实际part/batch/冻结成员核验资格，不拿首个ActivityID冒充整部分。RenderedPart.PayloadDigest使用最终分页plain载荷经现有delivery.EncodePayload后的原bytes摘要；实际发送只解码已提交Permit.Payload，不另传正文。

- [x] **步骤1：写首条时间窗口及完整分条红灯。**

```go
func TestSummaryWindowHasBothBounds(t *testing.T) {
    previous:=time.Unix(100,0)
    oldest:=time.Unix(110,0)
    earliest,latest:=SummaryWindow(oldest,&previous)
    if !earliest.Equal(time.Unix(160,0)) || !latest.Equal(time.Unix(170,0)) {
        t.Fatal(earliest,latest)
    }
}
```

render测试构造20市场、多备注快照、跨条Combo、emoji和HTML特殊字符；解析后每部分≤4096字符且所有输入activity/market/Outcome/方向/链接可追溯，不以截断丢成员。运行 `go test ./internal/tradersync -run 'TestSummaryWindow|TestRender' -count=1`。

- [x] **步骤2：实现窗口和冻结schema。**新增`trader_sync_summary_batches`、`trader_sync_summary_parts`、`trader_sync_summary_part_items`；batch保存owner/binding_revision/oldest/first_started_at/恢复基点，part保存不可变text/digest/index/total及delivery_id，member固定batch唯一。group key含wallet/note_snapshot/market/Outcome/side。默认纯文本Telegram正文（不设parse_mode），同时以Unicode code point和UTF-16 code unit保守计数≤4096来分完整展示行，链接以完整URL显示；超长单行拆成有明确延续标识的多行，不能只取前4096字符。

```go
func SummaryWindow(oldest time.Time,previousStart *time.Time) (time.Time,time.Time) {
    earliest:=oldest
    if previousStart!=nil && previousStart.Add(time.Minute).After(earliest) {
        earliest=previousStart.Add(time.Minute)
    }
    return earliest,oldest.Add(time.Minute)
}
```

先确定所有parts再写最终n/m，若页码增加使边界变化重新切分至稳定；部分与activity多对多，不把跨部分activity只关联首条。冻结同事务令summary membership由waiting进入frozen；撤权先于冻结则cancelled_before_freeze、不建空批次。活动/摘要站内链接分别固定为member `/trader-sync/activities/{id}`、`/trader-sync/summaries/{id}`，保留部署base，不能用旧占位路径。稳定SQL后 `make sqlc-local`。

- [x] **步骤3：实现冻结至实际started的短session gate。**worker/chat/Bot槽在锁前准备；取专用连接的session advisory lock，BeginTx核验资格、冻结此刻全部同revision待汇总成员、写parts与首条许可、Commit，紧接进入一次Sender。RoundTrip入口started握手后补记起点再释放gate，结果等HTTP完成另开短CAS。RecordStarted只更新attempt/start事实，不再次取得account gate；否则会与持有者自锁。获取槽后锁等待超出槽有效性时释放并重新排队，未取得有效许可不能发。

```sql
-- AcquireAccountSession使用固定连接，键与WithAccountTx完全相同。
SELECT pg_advisory_lock(hashtextextended('athena:account:' || $1::text,0));
-- 只有该连接释放成功才能归还pool；失败则Hijack并Close。
SELECT pg_advisory_unlock(hashtextextended('athena:account:' || $1::text,0));
```

若本地校验未进HTTP，发送结果返回且无started，释放gate并按明确失败处理；若started写库失败，短deadline后关闭/释放连接并记录缺失，在进程内用已观测monotonic起点维持间隔。不能持锁重试至HTTP回执。重启必须确认旧sender停止；起点无证据时采用恢复时刻为下一批保守基点并标异常，不伪造first_started_at。

- [x] **步骤4：写冻结竞态与混合部分结果集成测试。**用真实两连接：首条冻结完成后阻塞HTTP入口，同时发起新activity投影；断言其无法获得recorded_at，直至started已记录且gate释放；HTTP响应仍阻塞时允许新活动/撤权完成。注入started持久失败仍明确ACK成功，断言sent且started缺失；成功/unknown部分不重发，临时失败只重试该部分≤5次，整批只有all sent成功。
- [x] **步骤5：验证调度在旧批未完时保留下批首条预算。**可控clock运行第10/11条、紧贴60秒成员、多个owner、Bot回复/系统消息竞争及同owner12条集中；检查起点间隔、最老成员deadline、公平性、所有部分完整度和失约计数。对本地渲染/锁/调度造成miss保留失败样本，不改recorded_at或cohort制造通过。执行本任务单元、真实DB与`-race`测试，提交 `feat(trader-sync): deliver complete summaries with durable timing boundaries`。

本项验证记录：真实组合同时保留monotonic与UTC起点。当前环境曾出现约3.8秒墙钟前移，首发后1秒形成的最紧样本发生失约并保留；独立首发后5秒实际形成样本在相同60秒阈值下达标，本地暂停负样本验证真实失约计数。该结果不证明任意时钟跳跃下所有紧窗达标，不替代任务13全路径时效与容量验收。

## 任务12：公共API、隐私查询、配置与进程组合

**Files**
- 新增：`pkg/apis/application/v1alpha1/trader_sync_types.go`、`pkg/apis/application/v1alpha1/trader_sync_protomessage.go`、`internal/server/tradersync/tradersync.proto`、`internal/server/tradersync/tradersync.go`、`internal/tradersync/service.go`、`internal/tradersync/pagination.go`、`internal/tradersync/types/reads.go`、`internal/tradersync/store/reads.go`、`internal/tradersync/store/queries/reads.sql`。
- 新增组合：`internal/server/trader_sync_runtime.go`及定向`trader_sync_runtime_test.go`；在server同包集中构造唯一后台及自有来源client/transport，复用已创建pool/traderStore；AthenaServer保留进程cancel/join/致命出口。
- 修改：`internal/server/athena-server.go`、`internal/server/authz.go`、`cmd/athena-server/commands/athena-server.go`、`cmd/athena-notification/commands/athena_notification.go`、`internal/notification/server.go`、`internal/tradersync/config.go`、`hack/local-runtime.sh`、`Procfile`及必要的本地启动helper、`docker-compose.prod.yml`；只有实际生成契约发现问题才调整`hack/generate-proto.sh`的精确schema处理。
- 生成：`pkg/apis/application/v1alpha1/generated.proto`、`pkg/apis/application/v1alpha1/generated.pb.go`、`pkg/apis/application/v1alpha1/generated.protomessage.pb.go`、`pkg/apis/application/v1alpha1/zz_generated.deepcopy.go`、`pkg/apiclient/tradersync/tradersync.pb.go`、`pkg/apiclient/tradersync/tradersync.pb.gw.go`、`assets/swagger.json`。
- 测试：`internal/server/tradersync/tradersync_test.go`、`internal/server/tradersync/contract_test.go`、`internal/server/authz_test.go`、`internal/tradersync/pagination_test.go`、`internal/tradersync/service_test.go`、`internal/tradersync/store/reads_integration_test.go`、`internal/server/trader_sync_activity_pagination_integration_test.go`。
- 订阅错误码真实消费者：`internal/tradersync/subscriptions.go`及对应测试；坏确认token按本项既定FailedPrecondition契约映射，保留当前grant/幂等优先顺序。
- 观察健康真实消费者接续：`internal/tradersync/collector.go`、`internal/tradersync/rpc/session.go`及对应Session/Collector/真实PG回归，monitor_interval权威schema及相关store查询/生成消费者；只复用原健康检查，不新增轮询。
**Interfaces**
- 消费：任务5Resolve、任务6SubscriptionService、任务9Collector.Run、任务10Projector.Run、任务8DirectoryRefresher.Run与任务11SummarySource。
- 产出业务组合：`NewService(cfg Config,deps Dependencies) (*Service,error)`、`(*Service).Run(ctx context.Context) error`、`Close() error`。Dependencies明确含`Pool *pgxpool.Pool`、`Resolver *TargetResolver`、`Subscriptions *SubscriptionService`、`Collector *Collector`、`Projector *Projector`、`Directory *DirectoryRefresher`；各组件由同一composition root组装，不在业务方法内创建新pool。
- 产出handler：`New(service *tradersync.Service) *Server`；全部RPC标准`(context.Context,*Request)(*Response,error)`，精确消息字段如下表。内部Service转发会员方法时显式传owner；管理员读取另用仅概要的store方法，不复用Activity。
- 游标：`Cursor{Version int; Kind,PrincipalID,FilterDigest,Direction string; PageSize int32; SnapshotID,UpperID,LowerID,AfterID,Time,ID string; Empty bool}`；Kind为activity_snapshot/activity_next/activity_refresh/history/subscription/part/admin_subscription，互不混用。`EncodeCursor(c Cursor,key []byte) (string,error)`、`DecodeCursor(token string,key []byte,principalID,filterDigest,kind string,pageSize int32) (Cursor,error)`。HMAC-SHA256保护规范JSON，base64url编码payload和签名，constant-time核验；activity数值键内部解析int64，不能按字符串词典序排序。history/subscription使用Time/ID，part使用稳定序号键。

所有API资源ID/金额/链上position用string，revision用uint64并验证实际gateway表示；时间用UTC RFC3339Nano字符串。生产gateway使用encoding/json，revision及其他uint64字段需局部JSON string tag并与实际Swagger一致；不全局更换marshaler。可缺value/时间/对象统一省略缺失key，evidence仍保留，非nil空串/false/真0必须输出；输入note:null与缺note同为nil。member请求不增加account_id。列表`PageInput{int32 page_size; string cursor}`，`PageInfo{string next_cursor}`；activity另用`ActivityPageInfo{nextCursor,refreshCursor,snapshot,asOf string; hasNewer bool}`。snapshot为签名不透明标记，refreshCursor已含原水位，客户端不另传latest_marker。具体DTO使用明确字段及独立availability wrapper，不能把map[string]any暴露为契约。HTTP GET分页必须用嵌套query `page.page_size`与`page.cursor`，普通过滤为`subscription_id`等；真实gateway测试固定此形状，不照抄其他模块的顶层page_size。服务器沿用runtime的protobuf JSONName别名接受面，不新增严格query parser；测试须断言handler实际收到嵌套值，不能只检查HTTP200。

application DTO契约如下；每个struct按表中顺序分配连续字段号，从1开始，嵌套对象使用独立struct。可选值用指针/nullable消息，重复字段保留nil与availability，JSON tag使用camelCase。

| DTO | 字段契约 |
| --- | --- |
| TraderSyncFieldEvidence | availability、reasonCode、source、queriedAt；均string。 |
| TraderSyncStringField / DecimalField / BoolField / TimeField | evidence及可选value；分别string/string/bool/UTC时间字符串。DecimalField不转浮点。 |
| TraderSyncCurve | evidence、points（每点t为Unix秒字符串、p为精确金额字符串）。 |
| TraderSyncPnLView | period、amount（DecimalField）、curve、interval、fidelity、referenceTime（TimeField）、timezone（StringField）。 |
| TraderSyncResolvedTarget | wallet、canonicalProfileURL、avatar/displayName（StringField）、verified（BoolField）、joinedAt（TimeField）、positionValue/largestWin/predictions（DecimalField）、pnl（恰好六项PnLView）、defaultPeriod=1Y、confirmationToken、expiresAt、usageNotice、savedNote（可缺TargetNote）、existingSubscription（可缺id/status/revision）、quota（used/limit）。 |
| TraderSyncTargetNote | wallet、note、revision；取消后保留。 |
| TraderSyncQuota / ExistingSubscription | Quota含used/limit（int32）；ExistingSubscription含id/status字符串及revision（uint64）。 |
| TraderSyncTargetDisplay | displayName、avatar、profileURL；各为StringField，保留确认时查询证据。 |
| TraderSyncSubscription | id、wallet、status、revision、generation、note、noteRevision、createdAt/updatedAt/pausedAt/cancelledAt/permissionDisabledAt、currentInterval（可缺）、observation、bindingStatus、queueNotice、queueCounts（六态）、targetDisplay。status是六种投影；bindingStatus是当前账户事实，不能冒充每个活动的notificationMode。 |
| TraderSyncInterval / Observation / HistoryEntry | Interval含effectiveAt、endedAt及generation/epoch；Observation含state、reason、lastReliableAt、latestInterruption（可缺）、interruptionCount。HistoryEntry含id/kind/sortAt及interval或interruption之一；kind=interval/interruption。中断含start/end/recoveredAt（可缺）、reason、uncertainty、possibleMissing=true，不推测遗漏数量。 |
| TraderSyncActivity | id、subscriptionId、sourceRecordId、wallet、side、positionId、collateralRaw/sharesRaw/feeRaw、collateralSymbol及两种decimals、priceNumerator/priceDenominator和priceEvidence、sourceVersion、settledAt/receivedAt/recordedAt、publicTimeEvidence、metadata、noteSnapshot、notificationMode、notificationReason、delivery（普通可缺）、summaryProgress（摘要可缺）、targetDisplaySnapshot、finalityAnomaly（可缺）、sourceLocation；priceEvidence/publicTimeEvidence均为FieldEvidence，首次公开时刻当前不可观测。 |
| TraderSyncSourceLocation | chainId、exchangeAddress、transactionHash、blockHash、blockNumber、logIndex；均string，数字十进制，hash/地址完整0x，读取原source独立列。 |
| TraderSyncFinalityAnomaly | reason、detectedAt、publishedBlockHash、conflictingBlockHash（可缺string）；仅真实异常存在时返回对象。 |
| TraderSyncTradeMetadata | market（MarketRef）、legsEvidence、legs（positionId及MarketRef）、relationship；MarketRef含evidence/id/title/url/conditionId/positionId/outcome。 |
| TraderSyncDelivery / Attempt | Delivery含id、status、reason、authorizedAt/startedAt/resultAt/messageId（均按实际证据可缺）、attemptCount、latestAttempt（可缺）；Attempt含index、已知authorizedAt、可缺startedAt/resultAt、status/reason，仅最新一次。完整attempts留存但不内嵌。 |
| TraderSyncStatusCounts / SummaryProgress | Counts为total/pending/sending/sent/failed/unknown/cancelled字符串计数；SummaryProgress含phase、reason、batchId（可缺）、relatedPartCounts、batchPartCounts、oldestAt、firstStartedAt（可缺），不内嵌parts。 |
| TraderSyncSummaryBatch / SummaryPart | Batch含id、oldestAt、settledFrom/settledTo、recordedFrom/recordedTo、firstStartedAt（可缺）、activityCount、targetCounts（wallet/count）、partCounts、asOf；Part含id、index、total、delivery、associatedActivityCount，不返回消息正文。 |
| TraderSyncSubscriptionSummary | subscriptionId、accountId/username/email、wallet、status、生命周期时间、安全observation、activityCount、associatedDeliveryCounts（六态distinct逻辑delivery）、asOf；无note、消息或活动正文。 |
| TraderSyncRuntimeMetric / RuntimeStatus | Metric含name、value字符串、unit、kind=gauge/window/epoch、可缺windowStart/windowEnd/serviceEpoch；RuntimeStatus含collector连接/epoch/filter概要、metrics、asOf。raw/确认/投影积压、metadata缺失、队列/结果/时钟异常分别标明单位；不含逐条事件或payload。 |

订阅pausedAt/cancelledAt/permissionDisabledAt为可缺时间，仅当前对应desired_state且真实ended_at存在时输出，表示当前记录保存的对应停止操作时刻；恢复后省略，不从updated_at或其他interval拼出旧生命周期事件。会员与管理员概要采用同一事实边界。

Resolve的usageNotice固定说明“优先选择低频交易者；高频监控不纳入性能保障”；活动notificationMode表达形成时资格，Subscription.bindingStatus只表达当前绑定。暂停/取消queueNotice明确“已排队通知仍会继续发送，可能稍后收到”。任务15–19提供英文等义展示；后端通过不代替页面验收。

| RPC / 权限 | 请求消息字段 | 响应消息字段 / HTTP |
| --- | --- | --- |
| ResolveTarget / read | ResolveTargetRequest{input:string} | ResolveTargetResponse{target:TraderSyncResolvedTarget}；POST `/api/v1/trader-sync/targets:resolve` |
| CreateSubscription / write | CreateSubscriptionRequest{confirmation_token:string,request_id:string,note:TraderSyncNoteInput} | CreateSubscriptionResponse{subscription:TraderSyncSubscription}；POST `/api/v1/trader-sync/subscriptions` |
| ListSubscriptions / read | ListSubscriptionsRequest{page:PageInput,view:string,state:string}；view=current/cancelled，默认current | ListSubscriptionsResponse{subscriptions:repeated TraderSyncSubscription,page:PageInfo,quota:TraderSyncQuota,as_of:string}；GET `/api/v1/trader-sync/subscriptions` |
| GetSubscription / read | GetSubscriptionRequest{subscription_id:string} | GetSubscriptionResponse{subscription:TraderSyncSubscription}；GET `/api/v1/trader-sync/subscriptions/{subscription_id}` |
| PauseSubscription / write | PauseSubscriptionRequest{subscription_id:string,expected_revision:uint64,request_id:string} | PauseSubscriptionResponse{subscription:TraderSyncSubscription}；POST `/api/v1/trader-sync/subscriptions/{subscription_id}:pause` |
| ResumeSubscription / write | ResumeSubscriptionRequest{subscription_id:string,expected_revision:uint64,request_id:string} | ResumeSubscriptionResponse{subscription:TraderSyncSubscription}；POST `/api/v1/trader-sync/subscriptions/{subscription_id}:resume` |
| CancelSubscription / write | CancelSubscriptionRequest{subscription_id:string,expected_revision:uint64,request_id:string} | CancelSubscriptionResponse{subscription:TraderSyncSubscription}；POST `/api/v1/trader-sync/subscriptions/{subscription_id}:cancel` |
| UpdateTargetNote / write | UpdateTargetNoteRequest{wallet:string,note:string,expected_revision:uint64,request_id:string} | UpdateTargetNoteResponse{note:TraderSyncTargetNote}；PATCH `/api/v1/trader-sync/targets/{wallet}/note` |
| ListActivities / read | ListActivitiesRequest{page:PageInput,subscription_id:string,from:string,to:string,summary_batch_id:string,refresh_cursor:string} | ListActivitiesResponse{activities:repeated TraderSyncActivity,page:ActivityPageInfo}；GET `/api/v1/trader-sync/activities` |
| GetActivity / read | GetActivityRequest{activity_id:string} | GetActivityResponse{activity:TraderSyncActivity}；GET `/api/v1/trader-sync/activities/{activity_id}` |
| ListSubscriptionHistory / read | ListSubscriptionHistoryRequest{subscription_id:string,page:PageInput} | ListSubscriptionHistoryResponse{entries:repeated TraderSyncHistoryEntry,page:PageInfo,as_of:string}；GET `/api/v1/trader-sync/subscriptions/{subscription_id}/history` |
| GetSummaryBatch / read | GetSummaryBatchRequest{batch_id:string} | GetSummaryBatchResponse{batch:TraderSyncSummaryBatch}；GET `/api/v1/trader-sync/summaries/{batch_id}` |
| ListSummaryParts / read | ListSummaryPartsRequest{batch_id:string,activity_id:string,page:PageInput} | ListSummaryPartsResponse{parts:repeated TraderSyncSummaryPart,page:PageInfo,as_of:string}；GET `/api/v1/trader-sync/summaries/{batch_id}/parts` |
| ListSubscriptionSummaries / administrator | ListSubscriptionSummariesRequest{page:PageInput,account_id:string,state:string,wallet:string,include_cancelled:bool} | ListSubscriptionSummariesResponse{summaries:repeated TraderSyncSubscriptionSummary,page:PageInfo,as_of:string}；GET `/api/v1/admin/trader-sync/subscriptions` |
| GetSubscriptionSummary / administrator | GetSubscriptionSummaryRequest{subscription_id:string} | GetSubscriptionSummaryResponse{summary:TraderSyncSubscriptionSummary}；GET `/api/v1/admin/trader-sync/subscriptions/{subscription_id}` |
| GetTraderSyncRuntimeStatus / administrator | GetTraderSyncRuntimeStatusRequest{} | GetTraderSyncRuntimeStatusResponse{status:TraderSyncRuntimeStatus}；GET `/api/v1/admin/trader-sync/status` |

管理员列表的account_id是已批准概要过滤，仍绑定当前管理员身份的游标；“无account_id”约束针对会员请求。API facade内只用`session.AccountID(ctx)`，不能使用任意请求字段作为会员owner。

- [x] **步骤1：写真实gateway表示与授权登记红灯测试。**Create请求分别`{"confirmationToken":"x","requestId":"a"}`、`{"confirmationToken":"x","requestId":"b","note":{"value":""}}`，断言传给业务的Note分别nil/指针空串。positionId=`90071992547409931234567890`经过实际gateway JSON往返保持字符串，available真0与unavailable nil不同。所有16个RPC都登记对应鉴权，未知方法拒绝。savedNote缺失/空串、latestAttempt可缺、sent+startedAt缺失、notification三模式和summary三阶段须经过真实gateway往返。

```proto
message TraderSyncNoteInput { string value = 1; }
message CreateSubscriptionRequest {
  string confirmation_token = 1;
  string request_id = 2;
  TraderSyncNoteInput note = 3;
}
```

note wrapper保留存在性，不能用GetNote().GetValue()丢失nil；现有gogofast链未经验证不改用proto3 optional。运行新handler/contract测试确认方法尚未实现导致红灯。

- [x] **步骤2：实现会员和管理员分开的SQL读取与游标。**会员读在account gate事务内验证当前grant，再查owner数据；跨owner和不存在均NotFound。管理员只选择账户身份、wallet、状态/生命周期、健康/中断及活动/结果数量，不选note/活动正文/payload/逐条delivery。分页0→50、1–100合法、负值或>100拒绝，cursor绑定身份、筛选及方向。

```sql
-- member查详情必须有owner条件；管理员另写聚合投影。
SELECT * FROM trader_sync_activities WHERE owner_id=$1 AND id=$2;
```

所有幂等结果读取之前也重查当前grant；冲突Aborted、配额ResourceExhausted、重复AlreadyExists、坏输入InvalidArgument、坏token FailedPrecondition。授权前完成的读取可以晚返回，不持锁等客户端收包。稳定SQL后 `make sqlc-local`。

活动首次列表在同owner gate取已提交max(id)，之后限定id<=snapshot，按id DESC取page_size+1判断next。refresh使用签名页的上下界与Empty标志，只重新查询原成员；空页返回空，仍查询新活动exists。所有过滤先规范化后摘要：subscription_id、summary_batch_id、UTC结算时间[from,to)，变化即拒绝旧cursor；page.cursor与refresh_cursor不能同时传。固定batch先验证owner，activity过滤必须属于该batch，否则NotFound。

```sql
-- 三类活动查询都在已核验owner/grant的事务中；实际SQL加相同的可选过滤。
SELECT COALESCE(max(id),0) FROM trader_sync_activities WHERE owner_id=$1;
SELECT * FROM trader_sync_activities
WHERE owner_id=$1 AND id<=$2 AND ($3::bigint IS NULL OR id<$3)
ORDER BY id DESC LIMIT $4;
SELECT * FROM trader_sync_activities
WHERE owner_id=$1 AND id BETWEEN $2 AND $3 ORDER BY id DESC;
SELECT EXISTS(SELECT 1 FROM trader_sync_activities WHERE owner_id=$1 AND id>$2);
```

下列测试在`pagination_test.go`，不需要数据库；结构和签名由本任务Interfaces定义。

```go
func TestActivityCursorBindsOwnerAndPageSize(t *testing.T) {
    key:=[]byte("test-only-cursor-key")
    c:=Cursor{Version:1,Kind:"activity_refresh",PrincipalID:"owner-a",
        FilterDigest:"all",Direction:"desc",PageSize:50,SnapshotID:"9007199254740993",
        UpperID:"9007199254740993",LowerID:"9007199254740900"}
    token,err:=EncodeCursor(c,key); if err!=nil { t.Fatal(err) }
    got,err:=DecodeCursor(token,key,"owner-a","all","activity_refresh",50)
    if err!=nil || got.SnapshotID!=c.SnapshotID { t.Fatal(got,err) }
    for _,p:=range []struct{owner string; size int32}{{"owner-b",50},{"owner-a",100}} {
        if _,err:=DecodeCursor(token,key,p.owner,"all","activity_refresh",p.size); err==nil {
            t.Fatal("cursor accepted outside its context")
        }
    }
}
```

`reads_integration_test.go`使用任务1隔离DB和两连接屏障：先取第一页/下一页/空页，新增更大ID但更早recorded_at/settled_at，再refresh，断言原成员不变、has_newer符合原过滤、点击最新可见；回滚插入不制造新活动。覆写page_size、kind、batch、owner、过滤和签名均拒绝。准备51条观察记录和101个parts验证完整分页，单活动关联第1/101部分不得丢第二处；unbound后绑定历史仍in_app_only，sent缺started仍sent。

管理员SQL先以subscription_id+delivery_id去重再聚合状态，普通由activity关联，摘要由part_items关联；重试不增数量，同part跨两个订阅各计一次，全局只计一次。缺started_at不从sent排除。history/parts按页读、attempts只用LATERAL读取最新一条及count，不对每行单发SQL；所有数组/计数使用专门查询。运行metric明确unit/kind，window必须含起止、epoch必须含serviceEpoch；以handler JSON断言管理员响应不存在noteSnapshot、metadata、payload或逐条delivery。 raw阶段由实际Session队列raw_queue_depth及Persist调用中的raw_persist_in_flight两个gauge（raw_logs）表达；只有活session/epoch与DB当前epoch一致才返回值，其他情况省略，并以raw_observation_available布尔gauge表明可观测性，不把confirmation积压改名或缺值填0。复杂观察/中断归属与同代恢复提取为权威schema的安全普通view，独立member/admin/history外层投影共用，不物化缓存。

观察读取补充：monitor_interval新增nullable last_reliable_at，实际Collector复用原匹配pong/latest/已持久ACK过滤覆盖记录当前代合格健康检查点，保存较早实际观察的原UTC，网络不持账户锁、owner→fence复核；每原健康周期持久化pass最多5秒且轮转不饿死，不新轮询。普通checkpoint写失败/提交未知保留旧点并报告，不单独关闭健康epoch；真实失权/网络或raw失败边界保持。中断只归属实际受epoch关闭影响的对象，start物理起始未知则缺失，end/recoveredAt只取同代因果后继首个成功interval.effective_at；跨代手动恢复另列，不从epoch开始/结束制造恢复。无真实epoch中断的准备失败仅reason，不扩history kind或计数。覆盖quiet健康、晚加入、暂停/代次、旧session晚回调、提交未知和钟差；与读取/目录SQL形成稳定批次生成。

- [x] **步骤3：编写DTO/proto源并生成。**确认卡用具体String/Decimal/Bool/Time/Curve wrapper，统一availability/reasonCode/source/queriedAt；列表/详情包含有界观察/投递概要，history/parts走分页；授权/提交/结果时间各自保留缺失。新增非生成ProtoMessage声明遵循现有notification_protomessage.go build tag。输入稳定后 `make protogen`、`make clientgen`；检查go-to-protobuf、gogofast、gateway、Swagger和deepcopy真实diff，再写handler映射。以response fixtures固定camelCase和uint64为JSON字符串；当前generator若不能保真，先修源/生成配置，不让前端Number补救。

```go
note:=(*string)(nil)
if req.Note!=nil { value:=req.Note.Value; note=&value }
input:=tsmodel.CreateInput{Token:req.ConfirmationToken,RequestID:req.RequestId,Note:note}
```

protogen使用GOPATH的仓库路径：执行前确认脚本实际解析到本次隔离工作区，必要时为命令设置本任务专用GOPATH并建立对应symlink；不得让生成器写到另一个用户checkout。Swagger的全局int64处理不应改变普通string金额/ID；用真实JSON测试确认casing。

- [x] **步骤4：组合现有两进程并注册完整生命周期。**AthenaServiceSet字段、newAthenaServiceSet构造、gRPC registration、gateway registration及Run/Close全部接入；只启动一个Collector/Projector/目录任务；每个Run由Service.Run使用errgroup与同一取消context管理，Close取消并等待退出，不能重复Run开启多个实例。账户store持有pool，TS和notification adapter借用；server关闭时先停后台再关pool，notification进程另有自己的pool。复用任务6已经在NewServer中完成的权限hook注入，不延迟到newAthenaServiceSet，也不覆盖已安装hook。通知进程注册account/system/reply/summary源，共用唯一dispatcher/poller，启动前确认旧sender停止，不能自动从过期lease判定安全接管。TS后台只随API进程启动/退出，不因HTTP/gRPC graceful restart重建；后台fatal通过Run错误出口结束CLI重启循环，先cancel/join再关pool/client。目录刷新故障独立报告阶段、HTTP是否已发、提交是否已确认及收尾错误；可恢复的DB/提交未知通过重新锁定并读取持久状态裁定，不按目录错误类别终止健康Collector。未拿到目录状态锁或读回失败不发页，错误时不按未确认提交的next等待整轮；取消合并真实错误仍保留原因。须补足发页后事务回滚丢失节流的跨实例接续边界，并用真实双实例故障测试证明；本地退避或退出进程不能替代共享发页约束。允许同步目录Run/store协议及实际初始化/CLI消费者和相关生命周期测试。

目录协议接续：现有refresh行新增nullable admission_id UUID。第一短TX仅在到期时写token及DB now+6秒并明确提交，第二TX重锁核验同token/cursor并跨一次HTTP保存映射/游标；HTTP原5秒截止从准入UPDATE前单调时点起算，覆盖慢提交/重锁等待，超期或准入提交未知绝不fetch。正常结算清token并恢复完成后+1秒/末页10分钟原公式，后段回滚保留第一TX预约；旧事务不得换连接补写旧响应。错误后重读确认状态，调度按DB now差值转本地单调等待。以真实双实例+回环覆盖post-send upsert回滚、provider失败后结算失败、两阶段COMMIT未知、第二TX锁等待超期、断连接/旧响应、取消收尾和DB时差；目录错误不单独关闭健康Collector。此项修正Task8单TX请求后回滚会丢节流的已知缺口；正常调度/DB时钟及ctx取消边界下保证共享节流，不宣称任意暂停/时钟跳跃下的物理发包证明。所需schema/query与本项读取SQL形成稳定输入批次后统一生成，不手改sqlc产物。

配置显式`ATHENA_TRADER_SYNC_HTTP_URL`、`ATHENA_TRADER_SYNC_WSS_URL`、`ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY`及spec默认节奏，来源地址/version registry由核验值初始化；可选`ATHENA_TRADER_SYNC_PROXY_URL`统一TS外部HTTP/WSS、Gamma/Profile/目录的显式代理，空串直连，本地WSL未设置时沿用gateway:10809默认策略，非WSL/部署未配置直连，不隐式借Token字段或系统代理。本地默认在Goreman合并.env与继承环境之后判定，已设置空串必须保留，不能在dotenv前export默认值而遮盖.env；实际transport显式设置代理，不靠脚本unset全局代理。专用TS键不接管Telegram，不改变Token现有加载顺序。HMAC key由部署配置提供，不每次重启随机生成导致全部cursor失效。端点缺失明确报告不可监控，不用未配置的dRPC自动兜底，不新增独立库/服务端口；沿用5秒发送专属timeout，Bot30秒poll不被全局5秒timeout截断。

```go
// NewServer中accountStateStore创建后立即安装；它不启动Collector。
traderStore:=tradersyncstore.NewSQLStore(accountStateStore.Pool())
accountStateStore.SetAccessChangeHook(func(ctx context.Context,tx pgx.Tx,id string,before,after accountaccess.Access) error {
    if before.Modules[accountaccess.ModuleTraderSync]==accountaccess.AccessLevelReadWrite &&
       after.Modules[accountaccess.ModuleTraderSync]!=accountaccess.AccessLevelReadWrite {
        return traderStore.RevokeTx(ctx,tx,id,"permission_revoked")
    }
    return nil
})
```

此hook是任务6交付的真实实现示例，任务12复用，不在两个构造点各装一份。Source URL/config验证与Run生命周期的fake启动测试必须证明缺依赖报错、重复启动拒绝、关闭网页不停止后台、关闭进程才停止后台。

- [x] **步骤5：运行 `go test ./internal/server/... ./internal/tradersync/... ./pkg/apis/application/v1alpha1 ./pkg/apiclient/tradersync ./pkg/apiclient/account` 与新reads集成测试。**检查普通登录/API Key、禁用凭据、NONE/管理员不能读member、三个admin方法只概要、别人的cursor/token/id均拒绝。生成后编译两个cmd包；提交 `feat(trader-sync): expose authorized APIs and wire service lifecycle`。

本项实现与独立复审已完成（64d1ec61、823658f0）。实际签名活动游标已通过真实gateway/Service/隔离PG组合；本轮覆盖首/空页refresh及refresh后next、to范围之外排除，未单独验证第二页refresh与恰好等于to边界，不能外推报告覆盖。真实来源全链/容量、checkpoint服务端提交成功但ACK丢失继续由任务13承接，UI由后续页面任务验收。

## 任务13：指标、故障矩阵与首期容量验收

**Files**
- 新增：`internal/tradersync/metrics.go`、`internal/tradersync/metrics_test.go`、`internal/tradersync/acceptance/acceptance_integration_test.go`、`internal/tradersync/acceptance/fixtures_test.go`、`internal/tradersync/acceptance/capacity_test.go`。
- 修改：`internal/tradersync/projector.go`、`internal/tradersync/collector.go`、`internal/notification/dispatcher.go`、`internal/notification/store/attempts.go`，在现有日志/指标机制上增加观测，不新建监控基础设施。
- 指标真实消费者接续：按需要追加`internal/tradersync/store/activities.go`及其queries/types、权威accountstate初始schema/sqlc，Session/ReceivedLog/intake的同Session mono字段，notification worker/attempts结果SQL与transport原始Started采样，txgate窄timing、summary实际等待接点；不以无意义Dispatcher hunk代替真实消费者。
- 恢复状态接续：`internal/notification/recovery.go`、`service.go`、`notification.proto`，`internal/server/notification/notification.proto`及facade/全部实际生成消费者；Task19接现有admin notification-service/Service Status。
- 新增验收报告：`docs/testing/trader-sync-activity-alerts-acceptance.md`；原始临时执行产物放`.superpowers/trader-sync-acceptance/`，长期报告只引用保留的可复现数据摘要。
**Interfaces**
- 产出：`TimingSample{PublicEarliest,PublicLatest *time.Time; ReceivedAt,RecordedAt time.Time; AuthorizedAt,StartedAt,AckAt *time.Time; Cohort,Outcome,ClockSource string; ClockUncertainty time.Duration}`；`EvaluateTiming(sample TimingSample) TimingResult`；`TimingResult{PublicAssessable bool; Lower,Upper,ReceivedToRecorded time.Duration; Reason string}`。
- 验收fixture定义在`fixtures_test.go`：`newHarness(t *testing.T,owners,targets int) *harness`、`(*harness).Close()`、`Push(log types.Log)`、`Advance(d time.Duration)`、`Stats() snapshot`。该目录三个测试文件全部加integration build tag，普通go test不会连接数据库。harness持有任务1真实隔离DB、loopback HTTP/WSS/Telegram、实际Service与Dispatcher、可控clock；snapshot含活动/HTTP调用/终态/预算/扫描调用计数。test helper只在测试包，业务不加入故障开关。

指标接续以backend spec第10节的“形成时统计与接收时钟”“分段与结果观测”“恢复状态与运行接口”为准：活动INSERT原子冻结typed evidence，真实Session采样mono offset；严格burst要求最近原接收前件、同私聊绑定、实测间隔<既定1秒且前件首次普通任务仍竞争。证据不足保留ordinary_unclassified，和default均保留普通非burst及总体，不按结果改组。单statement snapshot记录形成前竞争，活动/part/attempt分母独立。worker捕获Sender返回即刻时间、原Started mono及本地处理差，attempt随原CAS保存可缺结果证据，result_at语义保留；gate单独度量实际Begin/锁等待。recovery由通知进程唯一等待owner发布，原runtime追加可缺对象并转发，未恢复完成不报running，毫秒string/未知/原因均保真。必要实际文件范围随这些消费者接续，不将早期四个观测文件名当实现上限，也不新建监控基础设施。

- [x] **步骤1：写时间证据红灯测试。**

```go
func TestUnknownPublicTimeIsNotReceivedTime(t *testing.T) {
    result:=EvaluateTiming(TimingSample{
        ReceivedAt:time.Unix(100,0),RecordedAt:time.Unix(102,0),
    })
    if result.PublicAssessable { t.Fatal("fabricated public timestamp") }
    if result.ReceivedToRecorded!=2*time.Second { t.Fatal(result) }
}
```

运行 `go test ./internal/tradersync -run TestUnknownPublicTime -count=1`。在活动形成时冻结cohort依据：到达间隔、当时同owner排队竞争；不以最终超时倒推突发。

- [x] **步骤2：实现可测量段与不可判定证据。**分别记录received→recorded、finality等待、metadata额外等待、gate等待、recorded→authorized/started/ACK、summary oldest→start、相邻批次start间隔。结果没有ACK也必须在pending/failed/unknown总数和年龄中出现，不能从报表消失。独立公开观测区间[p_min,p_max]且时钟可信时总延迟在[recorded-p_max,recorded-p_min]，不确定度扩展区间。

```go
result.ReceivedToRecorded=sample.RecordedAt.Sub(sample.ReceivedAt)
if sample.PublicEarliest==nil || sample.PublicLatest==nil || sample.ClockSource=="untrusted" {
    result.PublicAssessable=false
    result.Reason="public_time_or_clock_unverified"
    return result
}
```

不要标称区块时间就是公开查询时刻。SLO区间不能唯一判断阈值时报告不可判定；普通集中队列与外部故障单列并仍保留总体，其他owner的正常慢样本不豁免。只用有限维度标签，不将wallet/owner/note/payload作指标标签；详证据在受控日志/DB。

- [x] **步骤3：实现100关系两种矩阵并执行全路径故障注入。**场景A为10owner×各10互异wallet，场景B为10owner共享10wallet；断言100关系、100/10唯一目标及每个owner独立活动/备注/消息。每个fixture驱动真实过滤→raw→确认→活动→sender路径，不直接插activity冒充采集验收。

```go
func TestCapacityDistinctTargets(t *testing.T) {
    h:=newHarness(t,10,100)
    defer h.Close()
    s:=h.Stats()
    if s.Relationships!=100 || s.UniqueTargets!=100 { t.Fatal(s) }
}
```

snapshot在本步骤定义字段`Relationships,UniqueTargets,Activities,HTTPCalls,HistoricalRangeCalls int`及每owner结果map；newHarness创建成功基线并校验，无HTTP发送前直接通过测试不算容量验证。继续推送三来源fixture，等待/推进clock直到预期活动和确定终态，断言所有owner结果及调用次数。

- [x] **步骤4：执行完整故障矩阵并保存证据。**按本计划覆盖表逐项记录测试名/命令/结果：注册ACK竞争、同秒、raw失败、baseline提交未知、403/null、removed/重组/未知版本、暂停/撤权/重绑、permit提交未知、超时/ACK落库失败/缺起点、429/5次上限、summary冻结窗口/超长/混合结果/跨owner公平、sender旧实例停止确认和时钟跳变。真实PG事务测试用channel屏障，不用随机sleep碰运气。

RPC spy同时拒绝eth_sendTransaction/eth_sendRawTransaction，交易/签名接口使用调用即失败的fake；断言所有场景没有交易、钱包或仓位副作用。低频提示及暂停旧队列提示在本任务验证API字段和通知文案，任务15–20落实已批准页面并验收，不用后端测试冒充UI通过。
- [x] **步骤5：做限定真实只读来源联调与Telegram联调。**来源只查询配置的开发RPC和公开Profile/PNL，重新核对实现版本/ABI/币种、实际100OR订阅与少量已知活跃目标；按HTTP方法、WSS帧、版本/metadata/重连分别累计成本。Telegram只发送到用户明确提供或当前任务已明确授权的测试接收者；若无此信息，先完成loopback全部验收，再请求该缺失信息，不能使用仓库发现的任意chat自动外发。全六区间不可用照实报告字段原因，不把所有unavailable算资料适配通过。
- [x] **步骤6：运行 `go test -tags=integration ./internal/tradersync/acceptance -count=1`、相关race套件，并保存容量与时效报告。**明确区分可控100目标矩阵、真实小样本、真实100活跃目标压力和长期稳定性。真实公开时刻不明时公开→站内P95/P99标不可判定，不声明该项验收通过；没有真实100活跃样本时容量生产结论仍待验证。不得用预算算例代替计费全量或吞吐证据。提交 `test(trader-sync): verify failure boundaries and capacity scenarios`。

本项实现与独立修复复审已完成（27b2caa0、05ffee81）。真实可控矩阵与故障证据、首次确认分段、真实来源及一次获准Telegram发送见[验收报告](../../testing/trader-sync-activity-alerts-acceptance.md)。公开时刻SLO、真实100活跃目标与长期稳定性仍不可据此判定通过；资料六区间金额缺口如实保留。

## 任务14：会员 service、可取消读取与精确数值

**Files**
- 新增：`ui/src/app/member/trader-sync-models.ts`、`trader-sync-service.ts`、`trader-sync-service.test.ts`（均在member目录）；`ui/src/app/shared/use-visible-query.ts`、`use-visible-query.test.tsx`；`ui/src/app/member/pages/trader-sync/precision.ts`、`precision.test.ts`、`state.ts`、`state.test.ts`。
- 修改：`ui/src/app/member/services.ts`、`ui/jest.config.js`；新增/更新`ui/src/app/shared/services/requests.test.ts`验证模块请求scope。
**Interfaces**
- 消费：任务12真实gateway响应；现有`requests.get/post/patch(path,scope)`及superagent `.query/.send`，路径省略已有API base。
- models逐项对应任务12 DTO：`FieldEvidence`、`StringField`、`DecimalField`、`Curve`、`PnLView`、`TargetNote`、`ResolvedTarget`、`Quota`、`Subscription`、`HistoryEntry`、`Activity`、`Delivery`、`StatusCounts`、`SummaryProgress`、`SummaryBatch`、`SummaryPart`；时间/ID/金额/revision/计数保持string（quota/part序号等有界int32为number），可缺消息为`undefined`，不造默认数值。
- 产出：`AbortablePromise<T> = Promise<T> & {abort?:()=>void}`；`MemberTraderSyncService`注册为`memberServices.traderSync`。方法均返回AbortablePromise：`resolveTarget(input:string):ResolvedTarget`、`createSubscription(input:CreateRequest):Subscription`、`listSubscriptions(input:SubscriptionQuery):SubscriptionPage`、`getSubscription(id:string):Subscription`、`listSubscriptionHistory(id:string,page:PageInput):HistoryPage`、`pauseSubscription/resumeSubscription/cancelSubscription(id:string,input:ChangeRequest):Subscription`、`updateTargetNote(wallet:string,input:NoteRequest):TargetNote`、`listActivities(input:ActivityQuery):ActivityPage`、`getActivity(id:string):Activity`、`getSummaryBatch(id:string):SummaryBatch`、`listSummaryParts(id:string,input:PartQuery):PartPage`。
- 输入：`PageInput{pageSize?:number;cursor?:string}`、`SubscriptionQuery extends PageInput {view?:'current'|'cancelled';state?:string}`、`ActivityQuery extends PageInput {subscriptionId?:string;from?:string;to?:string;summaryBatchId?:string;refreshCursor?:string}`、`PartQuery extends PageInput {activityId?:string}`、`CreateRequest{confirmationToken:string;requestId:string;note?:{value:string}}`、`ChangeRequest{expectedRevision:string;requestId:string}`、`NoteRequest extends ChangeRequest {note:string}`。输出Page封装使用对应items字段`subscriptions/entries/activities/parts`和任务12的page/asOf/quota。
- 产出shared：`ReadScope{key:string;isCurrent:()=>boolean;subscribeInvalidation?:(listener:()=>void)=>()=>void}`、`useVisibleQuery<T>(load:()=>AbortablePromise<T>,scope:ReadScope,intervalMs:number):{data?:T;error?:Error;loading:boolean;stale:boolean;reload:()=>void}`。AbortablePromise的唯一类型定义放shared use-visible-query.ts，会员/admin仅type import，shared不导入member。轮询load保持最新ref，scope.key变化清数据/abort；每页scope.key由捕获的owner/generation再追加资源ID及规范化筛选/页游标，切换目标立即清理旧请求，不等下次5秒。依赖不使用每次render新函数触发循环。
- 产出state：`captureTraderSyncScope(ownerId:string):ReadScope`、`clearTraderSyncState():void`；后者递增generation并清理所有本模块内存草稿/分页缓存。后续任务在此文件扩展同一清理入口，App不为每种缓存另写一套撤权hook。
- 精度：`formatRaw(raw:string,decimals:number):string`、`formatFillPrice(numerator:string,denominator:string,places?:number):{text:string;approximate:boolean}|undefined`；默认6位小数，微小非零不能变成0，必要时改用精确比值文本。

- [x] **步骤1：让Jest发现新测试并写数值红灯。**将testMatch收敛为`['<rootDir>/src/app/**/*.test.ts','<rootDir>/src/app/**/*.test.tsx']`，覆盖原有范围及新文件；不移动既有测试。

```ts
test('raw amounts remain exact above JS integer precision', () => {
    expect(formatRaw('9007199254740993', 6)).toBe('9007199254.740993');
    expect(formatRaw('1', 6)).toBe('0.000001');
    expect(formatFillPrice('1', '0')).toBeUndefined();
    expect(formatFillPrice('1', '3')?.approximate).toBe(true);
});
```

- [x] **步骤2：运行红灯与发现检查。**`yarn --cwd ui test --listTests --runInBand`确认新test.ts被发现，再运行`yarn --cwd ui test --runInBand --watch=false --runTestsByPath src/app/member/pages/trader-sync/precision.test.ts`，失败必须是缺少目标实现。
- [x] **步骤3：实现精度与严格DTO映射。**formatRaw用BigInt与十进制字符串位置，不浮点除法；ratio以BigInt商/余数判断是否精确，分母<=0返回undefined。字段缺失保持缺失；必要ID/枚举非法返回协议错误，不能静默补空资源。Public DTO统一camelCase，不添加无消费者历史snake_case响应兼容。服务请求query仍使用proto snake_case。

```ts
const readScope = {module: AccountDataModule.TraderSync, mode: 'read'} as const;
const writeScope = {module: AccountDataModule.TraderSync, mode: 'write'} as const;
// MemberTraderSyncService.getActivity；normalizeActivity在models中按任务12字段逐项验证。
const req = requests.get(`/trader-sync/activities/${encodeURIComponent(id)}`, readScope);
const result = req.then(response => normalizeActivity(response.body.activity)) as AbortablePromise<Activity>;
result.abort = () => req.abort();
return result;
```

`normalizeActivity(value:unknown):Activity`及各顶层响应`normalizeResolvedTarget/Subscription/SubscriptionPage/HistoryPage/ActivityPage/SummaryBatch/PartPage/TargetNote`在models定义；嵌套字段用同一Evidence与string校验器。六period按1D/1W/1M/1Y/YTD/ALL固定输出，不依赖Go map遍历顺序。service测试使用任务12真实JSON fixture验证长ID、revision、零/缺失、note省略/空串及正确scope，捕获request.abort是否调用。所有分页把client pageSize/cursor转换为`{'page.page_size':input.pageSize,'page.cursor':input.cursor}`，刷新另用顶层refresh_cursor，不发送undefined query值。
- [x] **步骤4：写读取竞争红灯。**react-test-renderer挂载调用useVisibleQuery的最小Probe组件；fake timers+deferred Promise验证：5秒期间首请求未完不重入，hidden停止，visible立即刷新，失败保留旧data且stale，clearTraderSyncState后旧Promise完成不能发布。deferred在测试内用`new Promise<T>(resolve=>...)`保存resolve，不用真实sleep。
- [x] **步骤5：实现单飞读取与generation清理。**每轮只保留一个request；finally在generation匹配时释放slot。首次loading与后续刷新分开，不因刷新卸载已成功子树。visibilitychange/focus使用同一reload去重；cleanup abort并移除监听。发布前必须检查组件mounted、当前请求generation与scope.isCurrent，晚错误也不能覆盖新状态。member scope始终提供subscribeInvalidation；clearTraderSyncState同步清空Map、递增generation并通知仍挂载的hook中止请求/清除data/error，订阅时scope已失效也立即通知。仅有isCurrent不能替代主动清屏；真实App的三个清理边界由任务15步骤5接入。
- [x] **步骤6：运行新service/precision/state/hook测试、`yarn --cwd ui lint`；提交 `feat(trader-sync-ui): add typed reads and precise display primitives`。**检查改Jest匹配后发现数量增加而原测试仍被发现。此任务不挂空页面或新导航。

本项已实现并通过独立审阅（7c937df2）：124项定向测试通过，末次仅测试回调修正后14项hook测试通过；生产lint与额外测试类型检查通过，测试发现11→15且旧11项文件保留。真实Shell清理和页面接入由后续任务完成，不将基础hook测试视作浏览器验收。

## 任务15：独立添加页与 Telegram 返回草稿

**Files**
- 新增：`ui/src/app/member/pages/trader-sync/add.tsx`、`add.test.tsx`、`confirmation-card.tsx`、`pnl-chart.tsx`、`pnl-chart.test.tsx`。
- 修改：同目录`state.ts/state.test.ts`；`ui/src/app/member/routes.tsx`、`app.tsx`、`pages/notifications.tsx`；新增`ui/src/app/member/pages/notifications.test.tsx`。修改真实Shell回归文件`ui/src/app/app.test.tsx`（不是member/app.test.tsx）。局部样式修改`ui/src/app/styles/member-features.css`，只涉及添加/确认/PnL区域。
**Interfaces**
- 消费：任务14 service/models/precision/state；现有memberNotifications与5分钟expiresAt。
- 产出：`TraderSyncAddPage({ownerId}:{ownerId:string})`；`ConfirmationCard({target,note,onNoteChange}:{target:ResolvedTarget;note:string;onNoteChange:(value:string)=>void})`；`PnLChart({view}:{view:PnLView})`。
- state新增`AddDraft{ownerId,input,note:string;noteEdited:boolean;wallet?:string;target?:ResolvedTarget;createRequest?:CreateRequest;returnPath:string;scrollY:number}`；`readAddDraft(ownerId:string):AddDraft|undefined`、`saveAddDraft(draft:AddDraft):void`。内存仅一owner，clearTraderSyncState统一删除；不使用notification-storage、localStorage/sessionStorage或URL存草稿/token。
- AppRoutes的六类页面均将`props.access.user.accountId`作为ownerId；新增局部`traderSyncRoute(element)`仅RW放行，未授权Navigate `/account/access`。本任务先挂实际完成的`/trader-sync/add`，其余路径随对应任务挂载。

- [x] **步骤1：写添加与草稿红灯。**fake service返回恰好六区间，1Y默认、YTD amount unavailable但curve可用；断言切换不再次Resolve。相同owner返回保留输入/备注，新owner不可读取，20 emoji成功、21禁用；输入变化废弃旧target，另一钱包不沿用草稿。

```ts
test('draft clearing fences a pending read', () => {
    const scope = captureTraderSyncScope('owner-a');
    saveAddDraft({ownerId:'owner-a',input:'0xabc',note:'desk',noteEdited:true,
        returnPath:'/trader-sync',scrollY:120});
    clearTraderSyncState();
    expect(readAddDraft('owner-a')).toBeUndefined();
    expect(scope.isCurrent()).toBe(false);
});
```

- [x] **步骤2：运行添加/state测试确认目标失败，然后实现页面状态。**使用`input/resolving/review/creating/error`判别联合；输入改变立即取消旧Resolve并废弃token。真实身份失败无确认按钮，辅助unavailable给原因；当前重复给GetSubscription入口，配额满给管理入口。noteEdited=false时Create不传note，否则传`{value:note}`。

```ts
const request: CreateRequest = {
    confirmationToken: target.confirmationToken,
    requestId: crypto.randomUUID(),
    ...(noteEdited ? {note: {value: note}} : {})
};
// 写入AddDraft.createRequest后再发送；网络未知结果复用这个对象，不生成第二个requestId。
saveAddDraft({...draft, target, createRequest: request});
```

Create成功记录新subscription ID供主页侧栏定位；目标在原筛选外时保留筛选。token到期禁止新确认，已发送但结果未知使用原request恢复；重新Resolve得到新token才新建requestId。基线成功前不显示Monitoring。
- [x] **步骤3：实现六区间P/L与精确确认卡。**数字/曲线各自evidence；缺图显示原因而非零线；SVG只用经过有限归一化的图形坐标，标题和tooltip取原字符串。提供当前区间/起止/单位及文字摘要，按钮用aria-pressed；不只靠hover。钱包完整换行与复制，低频提醒放确认前，无额外复选框。
- [x] **步骤4：衔接Notifications返回并测试过期。**Add按钮在内存保存draft后导航`/notifications`，location.state仅放固定`{returnTo:'/trader-sync/add'}`，不带token/备注；Notifications校验精确允许路径且readAddDraft当前owner存在才显示Return to Trader Sync。返回不改expiresAt。解绑/重绑已有交互文案加入许可边界和不补历史，不增加第二次确认，不复制3秒状态机。
- [x] **步骤5：挂载共同清理。**member app的endSession、身份变化分支以及Trader Sync从RW失权分支调用clearTraderSyncState；请求继续使用模块scope abort。非法READ按任务6解析为NONE；六个深链都须RW路由保护，导航稍后任务17加入。
- [x] **步骤6：运行添加/图表/Notifications/state及`app.test.tsx`相关测试、lint；提交 `feat(trader-sync-ui): add target review and binding return flow`。**测试覆盖失败保留草稿、token过期、同payload结果重取、身份换钱包、Confirm重复点击阻止。

## 任务16：订阅列表、详情与生命周期

**Files**
- 新增：`ui/src/app/member/pages/trader-sync/subscriptions.tsx`、`subscription-detail.tsx`、`subscription-state.tsx`、`observation-history.tsx`、`subscriptions.test.tsx`、`subscription-detail.test.tsx`。
- 修改：`ui/src/app/member/routes.tsx`、`app.tsx`；同目录`pages/trader-sync/state.ts/state.test.ts`及`ui/src/app/styles/member-features.css`的订阅局部样式；`ui/src/app/app.test.tsx`验证实际RW深链和标题。
**Interfaces**
- 公开订阅状态沿真实wire：healthy显示Monitoring，interrupted显示Monitoring interrupted；其余四态不变。页面动作/过滤使用wire值，不能把显示语义monitoring/error当作协议枚举。
- 消费：任务14 list/get/change/note/history service、任务15 RW路由与state清理。
- 产出：`TraderSyncSubscriptionsPage({ownerId})`、`TraderSyncSubscriptionPage({ownerId})`，props均`{ownerId:string}`；详情用useParams取subscriptionId。`allowedSubscriptionActions(status:Subscription['status']):Array<'pause'|'resume'|'cancel'>`放subscription-state.tsx供列表/详情共用。
- `ObservationHistory({ownerId,subscriptionId}:{ownerId:string;subscriptionId:string})`独立分页；只展示时间线事实，不触发历史采集。

- [x] **步骤1：写六态动作矩阵与取消红灯。**

```ts
test.each([
    ['pending_baseline',['cancel']], ['healthy',['pause','cancel']],
    ['interrupted',['pause','cancel']], ['paused',['resume','cancel']],
    ['permission_disabled',['resume','cancel']], ['cancelled',[]]
])('actions for %s', (status, expected) => {
    expect(allowedSubscriptionActions(status as Subscription['status'])).toEqual(expected);
});
```

renderer测试当前/取消列表、完整钱包、取消一次确认、paused恢复响应仍Preparing、取消后无resume且有重新订阅入口；运行新tests确认失败。
- [x] **步骤2：实现列表/详情与页级读取。**Current默认，Cancelled独立cursor；ResourceTable传items/columns/compactRender，不传虚构total。详情5秒useVisibleQuery只更新状态概要；历史由ListSubscriptionHistory独立分页，不随概要刷新重新取全史。
- [x] **步骤3：实现直接暂停/恢复与一次取消确认。**同资源一个mutation slot，每次意图生成requestId并保存直到确定结果；按钮依据矩阵，服务端仍终判。取消弹窗含完整身份、不可恢复、释放名额、旧队列继续与历史保留；确认成功采用服务端事实，完整响应与当前已知状态按生命周期revision和独立noteRevision协调；较早响应不回退较新版本，同revision仍允许正常观察概要刷新。

```ts
const change: ChangeRequest = {expectedRevision: subscription.revision, requestId: crypto.randomUUID()};
const next = await services.traderSync.pauseSubscription(subscription.id, change);
// next为服务端事实；不自行将状态改成healthy/paused。
```

未知响应保留同change供重试；Aborted丢弃旧动作、读最新并提示重新选择，不能新revision自动重放。interrupted自动恢复不调用resume。
- [x] **步骤4：实现备注保存/冲突。**expectedRevision取noteRevision。空串可保存；冲突保留本地值与服务器最新备注，用户再次保存才用新noteRevision。取消目标仍改owner-wallet保留备注，说明当前/未来同钱包会沿用，旧活动快照不变。重新订阅跳Add并预填钱包，仍走完整确认。
- [x] **步骤5：实现中断时间线并测试51条。**每页显示interval或interruption、已知起止/恢复、reason及possibleMissing；未知显示原因，不能生成遗漏数量。上一页恢复缓存游标，更多记录可访问。添加两条记录同sortAt稳定ID排序测试，自动恢复后原中断仍存在。
- [x] **步骤6：运行订阅页面/state及service测试、lint；提交 `feat(trader-sync-ui): manage subscriptions and observation history`。**注册两个真实路由和lazy exports，验证无grant深链不调用service，不导入管理员页面。

## 任务17：活动主页、目标侧栏与稳定刷新

**Files**
- 新增：`ui/src/app/member/pages/trader-sync/home.tsx`、`home.test.tsx`、`activity-feed.tsx`、`target-sidebar.tsx`、`cursor-navigation.tsx`、`activity-session.ts`、`activity-session.test.ts`、`trader-sync.css`。
- 修改：同目录`state.ts`；`ui/src/app/member/app.tsx`、`routes.tsx`及`ui/src/app/app.test.tsx`。
**Interfaces**
- 消费：任务14 ActivityPage/ActivityQuery及useVisibleQuery，任务16状态展示和管理路由。
- 产出：`TraderSyncHomePage({ownerId}:{ownerId:string})`；`CursorNavigation({canPrevious,nextCursor,onPrevious,onNext}:{canPrevious:boolean;nextCursor?:string;onPrevious:()=>void;onNext:()=>void})`。
- 产出：`ActivitySession{query:ActivityQuery;current?:ActivityPage;previous:Array<{query:ActivityQuery;page:ActivityPage;scrollY:number}>;scrollY:number}`；`applyActivityRefresh(current:ActivityPage,incoming:ActivityPage):ActivityPage`只接纳相同成员及snapshot，非法成员改变抛协议错误、保留旧页；`readActivitySession(ownerId:string):ActivitySession|undefined`、`saveActivitySession(ownerId:string,value:ActivitySession,scope?:ReadScope):void`，异步/卸载写入传原捕获scope，state统一清理。

- [x] **步骤1：写刷新与分页红灯。**用服务层fixture构造current=[12,11]、snapshot12，refresh同成员但通知结果变化且hasNewer=true，断言成员和顺序不变；服务误返13导致协议错误；点击新活动发不带cursor/refreshCursor请求；空页refresh仍空但显示提示。日期筛选转换UTC+8为UTC `[from,to)`，历史页继续使用原snapshot。

```ts
// home.test.tsx中的page服务spy：下一次刷新必须引用当前页refreshCursor。
expect(listActivities).toHaveBeenLastCalledWith(expect.objectContaining({refreshCursor: 'signed-current-page'}));
expect(listActivities.mock.calls[listActivities.mock.calls.length - 1][0].cursor).toBeUndefined();
// 点击New activity available后断言两种cursor均未携带，而subscriptionId/from/to保留。
```

- [x] **步骤2：运行home/session红灯，再实现ActivitySession。**首次按query读取；Next压栈并用nextCursor、Previous恢复上页及其refreshCursor，在恢复页DOM提交后还原滚动位置，等待期间不回写旧短页位置；切filter/pageSize重建session；详情返回用session恢复位置。新活动按钮删除两个cursor，成功后清页栈并将焦点移到列表标题，失败仍保留当前页。

```ts
const refreshQuery: ActivityQuery = {...session.query, cursor: undefined, refreshCursor: session.current?.page.refreshCursor};
const latestQuery: ActivityQuery = {...session.query, cursor: undefined, refreshCursor: undefined};
// 原页没有current时用latestQuery；禁止同时传两个cursor。
```

- [x] **步骤3：实现同屏布局和三类读取。**活动一个页级ListActivities、侧栏一个Current ListSubscriptions（最多10）、当前绑定一个getTelegramSettings，5秒各自单飞。侧栏仅筛选/状态，管理入口进入列表/详情；All活动包含取消历史，点目标按subscriptionId。主区包含日期、结算时间标签、金额/份额、资料状态和通知概要；初始、无订阅、无活动、筛选空、失败保留分开。
- [x] **步骤4：实现稳定行和响应式。**列表key=activity.id、未变化字段不重挂载；状态更新只更新相应区域。复制/选择期间若该行metadata变化，暂存该行新展示资料，在selectionchange折叠后应用，其他行和通知徽标继续更新。目标栏在900附近转折叠，收起仍显示所选/配额/异常；390无页面级横溢出。CSS仅本模块class，复用theme变量，不影响其他页。
- [x] **步骤5：注册导航与入口并写权限回归。**member Markets添加Trader Sync，沿现有breadcrumbItems/routeTitle/moduleLandingPaths接入实际主页及深链标题；canAccessItem对该模块只接受RW，六个深链沿用traderSyncRoute。更新app.test.tsx校验NONE/非法READ不显示入口，带RW刷新/详情返回/失权跳转不泄露旧数据。添加成功聚焦新目标，但原filter不同不自动改变。
- [x] **步骤6：运行home/session/app/hook及precision测试、lint、build；提交 `feat(trader-sync-ui): add stable cross-target activity feed`。**轮询隐藏/恢复与选中文本由任务20浏览器再次验证真实DOM；不把renderer当浏览器证据。

## 任务18：独立成交与摘要详情

**Files**
- 新增：`ui/src/app/member/pages/trader-sync/activity-detail.tsx`、`summary-detail.tsx`、`trade-facts.tsx`、`combo-conditions.tsx`、`notification-result.tsx`、`activity-detail.test.tsx`、`summary-detail.test.tsx`。
- 修改：`ui/src/app/member/app.tsx`、`routes.tsx`及`ui/src/app/app.test.tsx`；同目录`trader-sync.css`、`state.ts/state.test.ts`承接独立详情返回会话与统一清理；`cursor-navigation.tsx`增加可选ariaLabel区分两套分页，默认名称不变；`activity-feed.tsx`增加可选summaryBatchId，将合法摘要ID写入本次活动链接state。
**Interfaces**
- 消费：任务14 getActivity/getSummaryBatch/listSummaryParts/listActivities，任务17CursorNavigation及返回会话。
- 产出：`TraderSyncActivityPage({ownerId})`、`TraderSyncSummaryPage({ownerId})`，props均`{ownerId:string}`；`TradeFacts({activity}:{activity:Activity})`、`ComboConditions({metadata}:{metadata:Activity['metadata']})`、`deliveryLabel(delivery:Delivery):string`。
- 页面分别从useParams取activityId/batchId；站内URL不接收owner覆盖，市场外链只能使用已核验metadata.url。
- 详情资源会话由state按owner+resource保留独立parts/activities页栈和scroll，并统一清理；返回来源取当前历史条目的受限traderSyncSummaryBatchId，不存入资源缓存。CursorNavigation增加可选ariaLabel，默认名称不变。

- [x] **步骤1：写结果证据与Combo红灯。**

```ts
test('sent remains sent when the submit timestamp was not recorded', () => {
    const delivery: Delivery = {id:'d1',status:'sent',reason:'',authorizedAt:'2026-09-10T10:00:00Z',
        resultAt:'2026-09-10T10:00:02Z',attemptCount:'1'};
    expect(deliveryLabel(delivery)).toBe('Telegram accepted');
});
```

renderer同时断言“Submit time not recorded”，没有已读/未调用推断；Combo NO显示“Not all conditions met”，不是各腿NOT；未知腿数不显示0。用1个活动关联2个parts（sent/unknown）证明不是整体成功。
- [x] **步骤2：运行详情测试红灯，再实现事实与证据组件。**事实优先、完整钱包、Settled at UTC+8、成交额/份额/费用分开；本条价格用precision helper，分母不可用/舍入说明完整。原始量、Position ID、tx/log、来源与时间在可展开区域。缺metadata可读ID/原因，无可靠市场URL不给伪链接；绑定当前状态不覆盖活动mode。
- [x] **步骤3：实现通知模式/阶段和相关部分。**in_app_only显示形成时未绑定；waiting无batch链接，cancelled_before_freeze给原因；frozen按batch+activityId调用ListSummaryParts分页。六态分别显示，latestAttempt证据及attemptCount可读，无重发/全部attempt入口。5秒页级刷新当前parts，不逐part发请求。
- [x] **步骤4：实现批次页两个独立游标。**GetSummaryBatch给总量和六态计数；Parts按序号，Activities带summaryBatchId按固定成员分页，各自保存页栈。只有sent==total且其他五态为0才显示整批成功，不用“部分成功”覆盖细分结果。

```ts
const complete = BigInt(batch.partCounts.total) > 0n &&
    batch.partCounts.sent === batch.partCounts.total &&
    ['pending','sending','failed','unknown','cancelled'].every(
        key => BigInt(batch.partCounts[key as keyof StatusCounts]) === 0n
    );
```

计数字符串在models规范为无前导零十进制；活动跨part可多次关联但总activityCount独立，不能把part数量叫成交数量。
- [x] **步骤5：挂实际路由并验深链接。**三个入口（列表、普通Telegram、摘要Telegram）落同一详情；登录returnTo复用既有helper与base路径。无上下文返回主页，有上下文恢复filter/cursor/scroll；摘要来源只随本次导航/历史条目保留，从主页再次进入同一活动不会继承旧摘要。NotFound与跨owner相同，不闪出缓存；外链点击停止行导航且可键盘聚焦。
- [x] **步骤6：运行详情/摘要/precision/app相关测试、lint；提交 `feat(trader-sync-ui): show trade and summary delivery evidence`。**含多part第二页、缺started但sent、未知公开时间不可判定、午夜UTC+8和超长Combo。

## 任务19：管理员概要与可见运行健康

**Files**
- 新增：`ui/src/app/admin/trader-sync-models.ts`、`trader-sync-service.ts`、`trader-sync-service.test.ts`；`ui/src/app/admin/pages/trader-sync/subscriptions.tsx`、`subscription-detail.tsx`、`subscriptions.test.tsx`；`ui/src/app/admin/pages/service-status.test.tsx`。
- 新增：`ui/src/app/admin/read-scope.ts`、`read-scope.test.tsx`、`app.test.tsx`，承接独立管理员身份失效与真实Shell回归。
- 修改：`ui/src/app/admin/services.ts`、`routes.tsx`、`app.tsx`、`pages/service-status.tsx`；`ui/src/app/shared/services/requests.ts`及其test。
- 恢复字段消费者：`ui/src/app/admin/notification-service.ts`及相关测试；接Task13既有runtime的可缺recovery对象，精确string/缺值和Recovering状态，不新建API。
**Interfaces**
- 消费：任务12三个admin RPC；任务14中立shared useVisibleQuery，不导入任何member models/service/页面。
- admin models按任务12独立定义`SubscriptionSummary`、`RuntimeMetric`、`RuntimeStatus`及Counts/Page；无note、activity、payload或delivery明细字段。
- 产出：`AdminTraderSyncService.listSubscriptionSummaries(input:{pageSize?:number;cursor?:string;accountId?:string;state?:string;wallet?:string;includeCancelled?:boolean}):AbortablePromise<{summaries:SubscriptionSummary[];page:{nextCursor?:string};asOf:string}>`、`getSubscriptionSummary(id:string):AbortablePromise<SubscriptionSummary>`、`getRuntimeStatus():AbortablePromise<RuntimeStatus>`；注册为adminServices.adminTraderSync。
- 产出页面：`TraderSyncAdminSubscriptionsPage()`、`TraderSyncAdminSubscriptionPage()`，管理员身份由既有AdminShell保护；内部读取`useAuthorization()`的accountId、iss与isAdmin，使用独立admin generation及失效订阅；普通页面卸载取消自身读取，Shell退出/失权/身份变化/卸载结束会话并清屏。权限复查失败在Shell显示明确错误与受控重试，仅最新成功的管理员身份恢复合法session。

- [x] **步骤1：写安全DTO/请求scope红灯。**将`'admin-trader-sync'`加入AuthorizationRequestFeature；service读统一该feature+read。用带额外私有字段的输入fixture验证规范化结果只选安全字段（接口本身也由任务12证明不返回）；新增GET路径、wallet/include_cancelled/page_size query和abort测试。

```ts
const scope = {feature: 'admin-trader-sync', mode: 'read'} as const;
const req = requests.get('/admin/trader-sync/subscriptions', scope).query({
    wallet: input.wallet, include_cancelled: input.includeCancelled, 'page.page_size': input.pageSize,
    'page.cursor': input.cursor, account_id: input.accountId, state: input.state
});
// 以独立normalizeSummaryPage提取允许字段并转AbortablePromise，禁止复用member normalizeActivity。
```

`normalizeSummaryPage(value:unknown)`、`normalizeSubscriptionSummary(value:unknown)`、`normalizeRuntimeStatus(value:unknown)`由admin models定义，返回上述安全类型。
- [x] **步骤2：实现只读列表/详情和导航。**System→Trader Sync，Current默认，可包含Cancelled，账户/规范钱包/状态过滤后重建cursor。完整钱包、用户身份、生命周期、健康/中断计数与Associated deliveries；说明跨行不可相加，无查看活动/消息和订阅写按钮。注册两条实际lazy路由，详情链接Service Status。
- [x] **步骤3：写Service Status可见/单飞红灯。**现有页面setInterval无visibility判断，useAsyncData不保证单飞。使用fake timers令一次list未完成，10秒不能并发；hidden不能启动新轮，恢复visible立即一次；一个runtime失败保留另一来源旧数据并分别标更新时间，不重置计数为0。
- [x] **步骤4：接入三个读取与运行区域。**替换该页无条件定时器，以shared useVisibleQuery为serviceStatus.list、adminNotifications.getRuntimeStatus、adminTraderSync.getRuntimeStatus分别维护10秒单飞和错误；手动Refresh调用同一reload去重。既有服务/Notification内容保留，新增Trader Sync Section并互链订阅概要。asOf/单位/windowStart/windowEnd/serviceEpoch按kind显示，未知资料显示不可用，不能把不同单位加总。
公开kind只取gauge/window/epoch；epoch必须有serviceEpoch。Projector累计项的serviceEpoch为计数实例的稳定opaque string，同实例跨Collector重连不变，新实例/计数重置才改变；不能按Collector连接序号解析或跨不同指标生产者合并累计值。
发送进程正在初始化或等待恢复预算时显示 Recovering，单列原因、该进程报告的剩余/已用等待及更新时间；fatal/stopped优先，恢复未完成不显示Running。recovery对象可缺，remainingMillis/elapsedMillis为十进制string，未知剩余保持Unavailable，合法0保留；浏览器不重新推算或解除恢复预算，也不把恢复等待归为Telegram网络耗时。raw队列与持久化中数量只在raw_observation_available可观测时展示，不可观测时不填0；按原10秒单飞读取更新，不另加计时或逐条请求。

- [x] **步骤5：运行admin service/页面/Service Status/requests及shared hook测试、lint、build；提交 `feat(trader-sync-ui): add private-safe admin monitoring views`。**源码静态检查admin新目录没有member imports；验证后台请求不使用member scope或任意accountId伪装owner。

本项已实现并通过独立修复复审（e2f1db6a、c14d73f6）。原范围62项测试、lint与构建通过；最后权限复查失败修复覆盖30项相关测试及lint，旧构建不冒充修复后版本。管理员安全概要、三来源独立可见读取、恢复状态与身份清理均已接入；真实浏览器与部署前缀验收由任务20执行。路由先行RED的历史偏差、预期jsdom导航输出和大chunk提示保留在审查记录，未改写成无警告或完整TDD证明。

## 任务20：前后端浏览器验收与证据

**Files**
- 新增：`ui/playwright.config.ts`、`ui/e2e/trader-sync.spec.ts`、`ui/e2e/trader-sync-live.spec.ts`、`ui/e2e/trader-sync-fixtures.ts`、`internal/tradersync/acceptance/ui_harness_integration_test.go`。
- 验收发现的必要修复：`ui/src/app/member/pages/trader-sync/subscription-state.tsx` 保留 Modal 的关闭生命周期，使 Escape/取消关闭后焦点返回触发按钮；对应既有订阅测试按实际需要适配，真实浏览器验证焦点及确认取消行为。
- 文字对比度修复：会员 `pages/trader-sync/{add.tsx,subscriptions.tsx,subscription-state.tsx,trader-sync.css}` 与管理员 `pages/trader-sync/{subscriptions.tsx,subscription-detail.tsx,trader-sync.css}`（管理员 CSS 新增），以及 `ui/src/app/admin/pages/service-status.tsx` 本次新增 runtime 状态标签；仅在 Trader Sync 与该新增状态区域内复用现有主题 token 修正辅助文字、placeholder、选中/危险控件与状态标签，保留全局 StatusTag；正常可操作状态按 AA 严格阈值验证，禁用/隐藏按标准单列。
- 测试接线：`internal/server/trader_sync_ui_harness.go` 及其 `trader_sync_ui_harness_test.go` 仅以 `integration && uiharness` 编译，窄适配真实鉴权、gateway 和静态 handler；StartUI、HTTP装配及直接消费者放在同 tag 的 `ui_harness_integration_test.go`。普通 `fixtures_test.go` 不引用这些入口。签名、cookie、logout 复用已有公开入口，不增加生产控制或绕过认证入口。
- 修改：`internal/tradersync/acceptance/fixtures_test.go`补本地 Telegram Bot update 队列及确定性故障响应（不引用双tag adapter）；`docs/testing/trader-sync-activity-alerts-acceptance.md`补UI结果，原始trace/截图放`.superpowers/trader-sync-acceptance/`；每次调用用`--output`指定独立运行/前缀/project/轮次子目录，避免Playwright清理输出时覆盖先前证据。
**Interfaces**
- 消费：任务13真实隔离DB/Service/Dispatcher/loopback来源，任务15–19实际页面；当前仓库有Playwright依赖但没有配置/spec，必须本任务新增。
- harness新增测试方法`(*harness).StartUI(t *testing.T,distDir string) UIHarnessInfo`，`UIHarnessInfo{BaseURL,PathPrefix,MemberAState,MemberBState,AdminState string}`。测试使用真实网关、权限store、handler及数据库，以测试签名器颁发仅隔离环境可用的会话，导出Playwright storageState文件；不得增加生产绕过认证路由。测试控制入口仅test进程localhost，提供push source/断流/修改grant/丢一次响应的确定性屏障，不能出现在正式server构建。
- `ui_harness_integration_test.go`使用`//go:build integration && uiharness`，单独启动需外部stop信号的交互harness，不纳入普通integration套件。真实浏览器验收仍必须显式启动并执行。
- `TestUIHarness`以env `ATHENA_UI_E2E_DIR`为输出目录、`ATHENA_UI_DIST`为已构建静态目录，启动后写`harness.json`，等待目录下`stop`信号再关闭自己创建的服务/DB；缺必需env/测试DSN直接Fatal。manifest仅测试会话路径及loopback base，不含真实凭据。
- 先构建 UI，再编译 Go harness；`ATHENA_UI_DIST` 指向新的 `ui/dist/app`。两份 member/admin HTML 必须与本次 Go 编译嵌入的对应原文件逐字相同，否则启动失败。保留生产 `getIndexData` 的 base/meta 替换和原静态 handler，不预填缓存或复制 SPA fallback；保存对应 hash，运行中不替换资产。

- [x] **步骤1：建立测试配置与受控fixture。**所有测试目标必须通过`ATHENA_UI_E2E_BASE_URL`显式设置；没有目标就失败，不能默认访问开发/生产。两个project按testMatch区分route-fixture和live，fixture项目允许intercept，live不得intercept业务读取/写入来伪造通过。使用已有系统Chrome，不为验收引入新库。

```ts
// ui/playwright.config.ts
import {defineConfig} from '@playwright/test';
const baseURL = process.env.ATHENA_UI_E2E_BASE_URL;
if (!baseURL) throw new Error('ATHENA_UI_E2E_BASE_URL is required');
export default defineConfig({
    testDir: './e2e', workers: 1, fullyParallel: false,
    outputDir: '../.superpowers/trader-sync-acceptance/playwright',
    use: {baseURL, trace: 'retain-on-failure', screenshot: 'only-on-failure',
        launchOptions: {executablePath: process.env.ATHENA_CHROME_PATH || '/usr/bin/google-chrome'}},
    projects: [
        {name:'ui-fixtures',testMatch:'trader-sync.spec.ts'},
        {name:'live',testMatch:'trader-sync-live.spec.ts'}
    ]
});
```

- [x] **步骤2：写关键验收断言，再运行以暴露未覆盖行为。**fixture构造UI spec九种通知场景、超大ID/微小金额、51条history/101 parts、长Combo。桌面/手机分别覆盖，不把交互线框HTML当产品页面。

```ts
import {test, expect} from '@playwright/test';
import {installTraderSyncRoutes, memberPath} from './trader-sync-fixtures';
test.beforeEach(async ({page}) => installTraderSyncRoutes(page, 'monitoring'));
test('desktop and mobile keep the document within the viewport', async ({page}) => {
    for (const width of [1440,1280,900,390]) {
        await page.setViewportSize({width,height:900});
        await page.goto(memberPath('/trader-sync'));
        await expect(page.getByRole('heading',{name:'Trader Sync',exact:true})).toBeVisible();
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    }
});
```

在`trader-sync-fixtures.ts`导出`installTraderSyncRoutes(page:Page,scenario:string):Promise<void>`，通过已知bootstrap、权限和任务12JSON回复；测试beforeEach显式安装，不能遗漏导致上例跳登录后误判通过。另导出`memberPath(path:string):string`，实现为`(process.env.ATHENA_UI_E2E_PATH_PREFIX || '').replace(/\/$/, '') + path`；从manifest设置该前缀，分别以空前缀和`/athena`运行，不能用以斜杠开头的goto绕过部署前缀。admin路径在此前缀下加`/admin`。其余fixture需等待请求/响应事件，不以长sleep掩盖竞争。
- [x] **步骤3：启动隔离全链harness并运行live。**`yarn --cwd ui build`后，以任务1测试DSN及ATHENA_UI_DIST/ATHENA_UI_E2E_DIR运行`go test -tags=integration,uiharness ./internal/tradersync/acceptance -run '^TestUIHarness$' -count=1 -timeout=30m`。StartUI复用任务12真实API注册及测试身份；UI可作为同源静态资源或本地代理提供，固定部署子路径测试也由此入口配置。读取manifest设置BASE_URL及三个storageState路径，再运行`yarn --cwd ui test:e2e --project=live`。
- [x] **步骤4：走通三身份真实读写。**A添加/备注/暂停/恢复/取消并由真实基线/source推送形成activity；B同钱包独立note/activity，A读取B的资源ID/批次返回NotFound（HTTP404）；跨owner或其他上下文不匹配的游标统一InvalidArgument（HTTP400），错误正文不泄漏私有内容；admin仅概要且无member读取。Create第一次成功响应被测试传输层丢弃，token过期后同request重取原ID；不能拦截响应伪造第二个成功。撤权屏障后旧请求晚到不显示正文，重授不自动恢复；初始空列表和历史页的新活动提示、点击载入与原位刷新用真实DB确认。
- [x] **步骤5：验证设备、键盘、绑定与结果矩阵。**实际DOM检查1440/1280/900附近/390、深浅主题、tab焦点/弹窗返回、20 emoji、复制精度、选择文本后5秒刷新不丢选区；市场外链不误触行。Telegram仅loopback生成实际普通/摘要结果，验证未知/失败/缺started但sent/跨101 parts及全批计数。绑定流程使用loopback Bot update，真实绑定状态与草稿往返/过期不延长；真实Telegram联调仍按任务13明确接收者边界。登录returnTo和部署`/athena/`分别测试。管理员gauge/window/epoch和不可相加计数以只读SQL交叉核验。
- [x] **步骤6：处理发现并保存证据。**失败按systematic-debugging定位；只修导致批准行为不满足的问题，重跑受影响场景。执行`yarn --cwd ui test:e2e --project=ui-fixtures`及live，保留运行命令、提交、场景数量、截图/trace和实际失败；无障碍按AA计算新增文本对比度并人工/自动键盘核验，未测不能写通过。写stop并等待harness退出，只清理本任务创建资源；提交 `test(trader-sync): verify complete member and admin browser flows`。

本项已实现并完成独立任务审查及一次限定修复复审（e781e505、599028cf）。根路径与 `/athena` 的完整验收分别为33项受控界面场景和10项真实接口流程；备注持久化/owner隔离与单次颜色采样修正后，两种路径各重跑10项真实流程及5项受影响页面矩阵。另有169项相关UI测试、类型检查、资产等价和只读SQL证据。浏览器撤权旧读为实际中止，成功不可取消晚回调由组件测试单列证明；旧告警、范围外标签AA及公网时效/真实持续100活跃目标限制均保留。仅本地提交，未推送、合并或部署。

## 任务21：运行文档、相关回归和执行交付

执行分工：本项子代理交付文档、相关回归与本地提交；控制器完成限定任务审查后组织唯一全分支审查，再执行一次完成通知并记录整计划完成。步骤3和5属于最终收尾，不要求在全分支审查开始前已经完成，也不授权子代理提前外发。

**Files**
- 同步：`docs/requirements/polymarket-copy-trading/target-trade-monitoring-notifications.md`、`docs/design/trading/trader-sync-activity-alerts.md`、`docs/design/web-ui/trader-sync-activity-alerts.md`、`docs/design/web-ui/member-application-shell.md`、`docs/design/web-ui/administrator-application-shell.md`、`docs/design/identity-access/account-access-control.md`、`docs/design/notifications/account-telegram-notifications.md`、`docs/design/notifications/system-notification-operations.md`、`docs/design/development-runtime/local-runtime-orchestration.md`、`docs/testing/trader-sync-activity-alerts-acceptance.md`、`PRODUCT.md`。
- 按实际导航同步：`docs/design/README.md`、`docs/requirements/README.md`、`README.md`；仅调整受影响条目。实现规格保留批准记录，偏离必须写清原因和对应修订，不把技术限制悄悄改成需求放宽。
**Interfaces**
- 消费：任务1–20实现、生成记录和验收证据。
- 产出：可复现运行/恢复说明与真实完成状态；不新增公共接口。

- [x] **步骤1：更新当前实现说明和运行命令。**去掉Notification独立DSN/数据库归属；说明同库两个pool、单实例启动/停止顺序、确认旧sender已停止后恢复、Chainstack/dRPC手动切换、新实时边界和未补遗漏。清理旧1.1秒全局串行发送、旧三态/泛化自动重试描述，给出unknown、永久资格失效、摘要首条缺失和资料unavailable的排查入口。同步六类会员页、管理员概要/Service Status、Notifications草稿往返、UTC+8、游标与可见刷新、失权清理的实际文件/入口。保留Telegram绑定权限与系统通知原有功能。
- [x] **步骤2：完成受影响链的最终回归。**复用各任务已完成且源码未再变化的生成记录，不无条件重跑生成器。当前HEAD执行以下相关套件；若某项在最后改动后已执行且证据完整可引用，不重复为计数运行。

```bash
go test ./internal/accountaccess ./internal/accountstate/... ./internal/migration ./internal/notification/... ./internal/tradersync/... ./internal/server/... ./util/telegram ./util/polymarket
go test -tags=integration ./internal/accountstate/... ./internal/notification/... ./internal/tradersync/... -count=1
go test -race ./internal/notification/... ./internal/tradersync/...
go test ./pkg/apis/application/v1alpha1 ./pkg/apiclient/account ./pkg/apiclient/notification ./pkg/apiclient/tradersync
go test ./cmd/athena-server/... ./cmd/athena-notification/... ./cmd/athena-migrate/...
yarn --cwd ui test --runInBand --watch=false
yarn --cwd ui lint
yarn --cwd ui build
git diff --check
```

不要盲跑`go test ./...`：仓库存在真实网络/交易相关e2e，超出此功能验证范围。新增测试不能默认连接真实Bot或交易端点。遇到失败按systematic-debugging定位，不能用Skip、放宽断言或改业务SLO消除失败。

- [x] **步骤3：按Superpowers进行实现代码审阅并修复实际问题。**与批准spec和任务覆盖表核对；检查权限绕过、锁序、未知结果重发、摘要成员丢失、metadata阻塞成交和生成消费者错配。修复之后只重跑受影响验证；报告遗留的真实环境/性能证据缺口，未执行的必要验收不能写成通过或把整份计划标完成。
- [x] **步骤4：提交最终文档和验证修正，给出分支/提交与验收证据。**局部提交不默认授权推送、合并或部署；按finishing-a-development-branch进行交付。完整实施与必要验收尚未完成时停在真实状态，不发完成邮件。
- [x] **步骤5：仅在用户确认的整份计划已完整实施且必需验证完成后，从仓库根执行一次完成通知。**由控制器在实现、必要验证与最终审查全部完成后执行并记录实际结果。

```bash
make notify-task-complete \
  TASK_NOTIFICATION_SUBJECT='任务完成：Trader Sync 前后端' \
  TASK_NOTIFICATION_BODY='已完成：目标订阅、实时活动、Telegram通知、会员页面与管理概要；验证：单元、事务、接口和浏览器验收完成。'
```

等待命令完成；内置重试后仍失败则报告安全的错误概要，不声称邮件已发送。主题/正文按实际已完成范围修正，不能掩盖缺失验收。

本项代码、长期文档、相关回归、限定任务审查和唯一全分支审查已完成；最终统一修复提交为 `83f54c57`，限定复审确认 I1/I2/M1/M2 全部解决且没有新 Critical/Important/Minor。M3–M8 为非阻塞的测试完善与既有告警，明细及实网未证明边界见验收记录。完成通知已按默认 `.env` 执行一次，SMTP在第1次尝试接受，命令exit0；本地分支和工作树保留。

## 规格覆盖与计划自检

| 批准spec / 长期需求 | 实现任务 | 主要验证 |
| --- | --- | --- |
| §1–3进程/同库/规模；规则1–6、14、30–33 | 1、6、12、13 | 单权威迁移、pool归属、十模块、owner/配额、100关系两矩阵 |
| 后端§4接口/隐私/幂等；验收1–9、17、27–28 | 5、6、12、14、19 | 16 RPC、登录/API Key权限、管理员概要、CAS、snapshot/cursor/token owner绑定 |
| §5身份/确认卡/P/L/备注；规则40–42、验收9、33–34 | 5、6、8、12 | 精确URL、六区间独立证据、20/21 code point、nil/空、历史快照 |
| §6来源/基线/恢复；规则7–9、12–17、24–29、44；验收3、10–13、16、24–26、36–37 | 7、8、9、10 | 自身OrderFilled、三来源、最终确认、注册/ACK边界、无回补、重组及版本 |
| §7事务/持久实体；规则10–11、18–23、27 | 1–3、6、9–11 | 同库gate、候选固定归属、活动与外发原子、去重、无自动历史TTL |
| §8许可/结果；规则34–39、43；验收14–15、18–23、29、35 | 2–4、6、10、11 | 许可前后竞争、永久墓碑、unknown不重发、ACK落库恢复、Bot offset原子 |
| §9摘要/限速；验收20–23、29–31、35 | 4、10、11、13 | 滚动端点、前10逐条、冻结到started、完整分条、公平及双60秒边界 |
| 后端§10–11配置/证据/容量；规则45、验收38–39 | 9、12、13、20、21 | 心跳/头/时钟、计费全量、真实与模拟分开、不确定不当通过 |
| 后端§12及UI§13文档/非目标；规则17、验收32 | 12、20、21 | 无交易/签名/重发入口，实际消费者、页面及运行文档同步 |
| UI§1–4、§8添加/绑定；UI-1、UI-4 | 5、6、12、14、15、20 | 六区间、身份/缺失、备注、token和响应丢失、Notifications返回及清理 |
| UI§5订阅与历史；UI-1 | 6、12、16、20 | Current/Cancelled、六态、一次取消、note revision、观察历史分页 |
| UI§2–3、§9–10主页/刷新；UI-3、UI-4 | 10、12、14、17、20 | ID水位/空页/历史页、5秒单飞、隐藏停止、新活动提示、阅读位置、失权晚响应 |
| UI§6–7事实/通知；UI-2 | 10–12、14、18、20 | 精度/Combo/缺失证据、模式阶段、多个parts、sent缺起点、Telegram共用深链 |
| UI§8管理员；UI-5 | 12、14、19、20 | 独立DTO/SQL、非可加数量、Service Status可见单飞/单位/窗口 |
| UI§11–12架构/可访问性；UI-6 | 14–20 | 小组件、真实service、主题/1440至390、键盘/触屏/焦点、精确复制、无横溢出 |

计划作者自检（编写阶段，不代表实现测试）：

- [x] 逐节核对两份批准spec及长期设计覆盖表；45条业务规则、39条原验收及UI-1至UI-6均有实现与验证落点。
- [x] 扫描占位语、未定义跨任务接口及文件路径；修正引用和生成顺序。
- [x] 检查任务消费/产出类型、account/session锁序、owner引用、授权与恢复语义一致。
- [x] 核对实际UI入口、Jest发现规则、Playwright缺少配置、Service Status现有定时器和gateway嵌套query；在对应任务明确补齐。
- [x] 校验本次文档链接与diff；改动仅文档，未执行实现命令、业务测试或外发通知。
