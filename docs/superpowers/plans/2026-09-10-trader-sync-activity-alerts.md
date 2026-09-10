# Trader Sync / Activity Alerts 实现计划

> 当前执行状态：尚未执行。本文件是原后端设计对应的 14 项计划。用户随后要求先补 UI；[完整 UI spec](../specs/2026-09-10-trader-sync-activity-alerts-ui-design.md)各节已确认、书面待整体审阅。确认后按 writing-plans 更新为前后端联合计划，当前正文不能直接作为完整交付范围。
>
> 联合修订必须覆盖：Resolve 备注/重复/配额快照及 Create 成功结果优先恢复；活动 ID 顺序与 snapshot/refresh_cursor；ListSubscriptionHistory、GetSummaryBatch、ListSummaryParts；有界结果 DTO 和管理员计数；六类会员页面、管理员概要/Service Status、Notifications 返回及权限清理。任务 6 的“不增加业务页面”、任务 12 的旧游标/13 RPC/无界内嵌 DTO，以及仅后端验收范围均须修订。本轮仅标明差距，未开始执行或提前编写未经整体审阅的联合步骤。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现已确认的 Trader Sync 后端：目标确认、独立订阅、共享实时采集、持久活动、Telegram 普通/摘要投递及可核验运行状态。

**Architecture:** 保留 athena-server 与 athena-notification 两个进程，全部通知数据并入 athena PostgreSQL。原始日志先可靠保存，最终确认后按 owner 形成活动及投递资格；发送许可、结果与摘要成员均持久化。采集和发送各由单实例负责，所有异步恢复遵守原代次与不可重发终态。

**Tech Stack:** 仓库当前 Go 1.25.5、pgx/v5、goose、sqlc、go-ethereum、gorilla/websocket、现有 Telegram 客户端、gogo/protobuf/grpc-gateway、React/TypeScript 权限消费者；不升级依赖版本，不新增消息中间件。

**Spec:** [已整体确认的设计规格](../specs/2026-09-10-trader-sync-activity-alerts-design.md)、[长期设计](../../design/trading/trader-sync-activity-alerts.md)、[业务需求](../../requirements/polymarket-copy-trading/target-trade-monitoring-notifications.md)。执行者同时阅读三者。本计划细化实现，不改变已批准业务决定。

**状态：**计划已编写，尚未执行。代码块是实现指导与测试起点，不代表相应源码已经存在。本轮只交付计划；进入执行时选择工作方式和隔离工作区。

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
- 不做新产品页面布局、不执行交易。现有授权编辑器及类型消费者必须适配第十模块；新导航/页面仍需独立界面设计。

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
| 14 | 运行文档、最终检查与完成通知 | 13 |

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
- 产出：`txgate.WithAccountTx(ctx context.Context, pool *pgxpool.Pool, accountID string, fn func(pgx.Tx) error) error`；`txgate.LockWallet(ctx context.Context, tx pgx.Tx, wallet common.Address) error`。
- 产出：`pgtest.New(t *testing.T, schema fs.FS, dir string) *pgtest.DB`，DB含 `Pool *pgxpool.Pool`、`DSN string`。
- 产出：`(*accountstatestore.SQLStore).Pool() *pgxpool.Pool`，由现有store拥有并关闭；业务适配器只借用，不重复Close。

- [ ] **步骤1：写失败测试，证明迁移CLI只有一个athena/notification schema归属。**

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

- [ ] **步骤2：实现隔离数据库夹具及两连接锁测试。**夹具读取显式测试admin DSN，用 `pgx.Identifier{name}.Sanitize()` 创建 `athena_test_`+随机UUID数据库；从同DSN替换dbname，调用 `postgres.Migrate(ctx, dsn, schema, dir)`，再创建pool。Cleanup先关pool，再删除自己创建的数据库。禁止将输入DSN数据库直接清空；任何迁移失败Fatal并清理已创建资源。

```go
// migrations/embed.go
package migrations
import "embed"
//go:embed *.sql
var FS embed.FS
const Dir = "."
```

`schema_integration_test.go`用 `pgtest.New(t,migrations.FS,migrations.Dir)` 后查询 `to_regclass`，断言athena_account、account_access、telegram_bindings、account_notification_deliveries、system_notification_deliveries及telegram_polling_state均存在。另开两个pool同时迁移同一DB，断言成功且只有一组goose版本。gate测试用两个连接与channel屏障证明同owner串行、不同owner可并行。

- [ ] **步骤3：合并SQL与连接归属，稳定输入后生成，再适配store/CLI。**

```sql
-- WithAccountTx取得锁后才读取权限/时间；UUID先由Go严格规范化。
SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text, 0));
-- 原始接收与基线注册共用钱包命名空间；不得反向再申请账户锁。
SELECT pg_advisory_xact_lock(hashtextextended('athena:wallet:' || $1::text, 0));
```

`WithAccountTx`使用ReadCommitted，Begin→锁→fn→Commit；defer Rollback只负责清理。迁移注册结构增加逐模块 `Dir` 字段，原模块用"migrations"，athena用migrations.Dir；调用点使用选中模块的Dir，不改其他模块schema。Notification source使用ATHENA_SERVER_POSTGRES_DSN、athena及同一叶子FS；更新本地/Compose/bootstrap消费者并移除旧通知数据库配置。`sqlc.yaml`通知schema改为权威目录后运行 `make sqlc-local`，检查生成类型没有丢表。

- [ ] **步骤4：运行 `go test ./internal/migration ./internal/accountstate/... ./internal/notification/...` 和 `go test -tags=integration ./internal/accountstate/... -count=1`；确认两连接及同库建表断言通过。**测试DB实例可用专用命令创建：`docker run --rm -d --name athena-trader-sync-test-pg -e POSTGRES_PASSWORD=athena-test -p 127.0.0.1:55439:5432 postgres:16`；若名称/端口占用选择新的测试实例，不操作已有容器。测试DSN为 `postgres://postgres:athena-test@127.0.0.1:55439/postgres?sslmode=disable`，设置到ATHENA_TEST_PG_ADMIN_DSN；执行结束只停止本任务创建的容器。
- [ ] **步骤5：审阅并提交 `refactor(storage): unify notification persistence in athena`。**只暂存本任务文件及该批真实生成输出。

## 任务2：持久发送许可、明确结果及实际调用起点

**Files**
- 新增：`internal/notification/delivery/types.go`、`internal/notification/store/attempts.go`、`internal/notification/store/queries/delivery_attempts.sql`、`util/telegram/send_transport.go`。
- 修改：权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/notification/store/queries/account_notifications.sql`、`internal/notification/store/queries/system_notifications.sql`、`internal/notification/store/account_notifications.go`、`internal/notification/store/system_notifications.go`、`internal/notification/sender.go`、`util/telegram/telegram.go`、`internal/notification/notification.proto`、`internal/server/notification/notification.proto`、`internal/server/notification/notification.go`。
- 修改实际状态消费者：`internal/notification/service.go`、`pkg/apis/application/v1alpha1/notification_types.go`、`ui/src/app/admin/notification-service.ts`、`ui/src/app/admin/pages/service-status.tsx`、`ui/src/app/admin/pages/system-notifications.tsx`、`ui/src/app/admin/pages/system-notification-detail.tsx`。
- 测试：`internal/notification/store/attempts_integration_test.go`、`util/telegram/send_transport_test.go`、`internal/notification/sender_test.go`。
**Interfaces**
- 消费：任务1的账户gate、通知同库表。
- 产出：delivery叶子包中的 `WorkRef{Kind string; ID int64}`、`Permit{Work WorkRef; AttemptID uuid.UUID; OwnerID string; SenderIncarnation uuid.UUID; PayloadDigest []byte; AuthorizedAt time.Time}`、`Outcome{Kind string; MessageID string; RetryAfter time.Duration; Code string}`。Outcome.Kind为sent/retryable/failed/unknown。
- 产出：`(*SQLStore).Authorize(ctx context.Context, ref delivery.WorkRef, incarnation uuid.UUID) (delivery.Permit,error)`；`RecordStarted(ctx context.Context,p delivery.Permit,at time.Time) error`；`RecordOutcome(ctx context.Context,p delivery.Permit,o delivery.Outcome,at time.Time) error`。
- 修改Sender契约：`Send(ctx context.Context, request SendRequest, started func(time.Time)) delivery.Outcome`；保留CreateSystemTopic方法。store不导入父notification包，避免当前已有父包→store的循环。

- [ ] **步骤1：写HTTP故障测试，先证明“对端已接收但无响应”不会重发。**使用httptest handler计数并读取请求体后Hijack关闭连接；调用一次Send，断言Outcome为unknown且handler计数为1。另一个handler返回Telegram `ok=false,error_code=429,parameters.retry_after=7`，断言retryable及7秒；本地空消息则failed且计数为0。

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

- [ ] **步骤2：实现attempt表、资格墓碑与CAS SQL，生成后适配消费者。**新增 `notification_delivery_attempts`：UUID主键、work_kind/work_id、incarnation、payload_digest、authorized_at、nullable started_at/result_at/message_id、outcome；账户投递增加eligibility_revoked_at/reason及current_attempt_id，系统投递增加current_attempt_id及sending/unknown状态。attempt与work关系按kind在许可事务核验，账户owner一致，不用可变payload复用许可。

```sql
-- 结果更新必须绑定当次许可；行数0表示迟到/已终结，不能重新发送。
UPDATE account_notification_deliveries
SET status = $3, provider_message_id = $4
WHERE id = $1 AND current_attempt_id = $2 AND status = 'sending';
```

许可与attempt同时提交后才调用Telegram。仅明确retryable且资格未永久撤销、attempts<5时回pending；等待=max(1/2/4/8秒对应值,RetryAfter)。success写库失败只重试结果CAS。提交结果不确定先按attempt UUID读取，无法确认则不发送。同步现有Notification公共/内部proto的状态表达，稳定后执行一次 `make protogen`，适配现有状态消费者，不让unknown被旧映射误报成功。

当前公共枚举cancelled已经存在（值4），新增sending/unknown使用未占用编号。同步deliveryStatusString、normalizeDeliveryStatusFilter、runtime计数和管理员系统列表/详情的筛选、颜色、时间字段；账户逐条投递仍不向管理员开放。若application类型新增attempt时间，同批 `make clientgen` 更新deepcopy。会员notification-service里的binding attempt状态不是消息状态，不改成sending/unknown。状态测试覆盖JSON字符串往返和unknown独立计数。

- [ ] **步骤3：在实际HTTP RoundTrip入口记录started，禁用隐式重试。**新transport包装既有 `http.RoundTripper`，通过本次请求context的回调在进入base.RoundTrip前记录起点；不在goroutine启动/队列领取处调用。回调写入容量1的握手channel，不等待响应。预校验/编码失败不报告started；禁止跟随会重放POST的重定向，检查现有go-telegram/bot选项，发送路径只允许一个HTTP请求。保留原始结构化错误分类后再形成对外错误文本，不能把所有网络错误都归为“确定失败”。

- [ ] **步骤4：运行 `go test ./util/telegram ./internal/notification/...`、`go test -tags=integration ./internal/notification/store -run 'TestAttempt|TestOutcome' -count=1`。集成矩阵覆盖许可后崩溃、成功后结果写库失败、结果CAS冲突、起点缺失但有成功回执，以及撤权→重授→旧attempt返回429仍cancelled。**
- [ ] **步骤5：提交 `feat(notification): persist send permits and terminal outcomes`。**

## 任务3：Bot update、绑定与回复outbox原子消费

**Files**
- 新增：`internal/notification/store/bot_updates.go`、`internal/notification/store/queries/bot_updates.sql`。
- 修改：权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/notification/store/telegram_bindings.go`、`internal/notification/store/queries/telegram_bindings.sql`、`internal/notification/poller.go`、`internal/notification/service.go`。
- 测试：`internal/notification/store/bot_updates_integration_test.go`、`internal/notification/poller_test.go`。
**Interfaces**
- 消费：任务1 txgate和任务2投递结果契约。
- 产出：`(*SQLStore).ApplyBotUpdate(ctx context.Context, update utiltelegram.Update) error`；内部binding的Tx方法只由该事务调用；`telegram_binding_replies`作为独立work_kind=reply参与发送。

- [ ] **步骤1：写重复update的失败集成测试。**夹具创建owner与pending绑定token，构造Update{ID:42,Message:私聊/start token}；调用ApplyBotUpdate两次。断言绑定revision只增一次、next_update_id=43、只有一条reply，且未调用Telegram SendMessage。异常分支包括错误token、群聊、bot用户、已被另一owner绑定和my_chat_member失联事件。

```sql
-- 必须与绑定变化、回复outbox及offset同事务；重复ID不再次执行副作用。
INSERT INTO telegram_consumed_updates (update_id, consumed_at)
VALUES ($1, clock_timestamp()) ON CONFLICT (update_id) DO NOTHING
RETURNING update_id;
```

- [ ] **步骤2：写提交前后故障测试，运行 `go test -tags=integration ./internal/notification/store -run TestApplyBotUpdate -count=1` 确认红灯。**在提交前返回错误，断言四类记录全回滚；包装真实Commit使其提交后报告模拟断流，再重放update，断言不重绑/不重复回复。所用故障注入是测试事务装饰器，不添加业务feature flag。
- [ ] **步骤3：增加consumed_updates及binding_replies表，形成SQL批次后 `make sqlc-local`，再将poller改为单次ApplyBotUpdate。**账户gate之后才取Telegram用户唯一锁；跨owner冲突不抢占已有绑定。解绑/重绑取消旧revision未授权任务，永久标记旧sending不得后续重试。删除poller直接sendBindingReply的网络调用，改写reply work。有效绑定修改、消费进度和回复必须使用同一个pgx.Tx。
- [ ] **步骤4：运行本任务集成测试和 `go test ./internal/notification/...`；检查rollback后offset未前进、已提交重启从持久offset继续。**
- [ ] **步骤5：提交 `feat(notification): consume bot updates transactionally`。**

## 任务4：共享限速、公平调度与单sender恢复

**Files**
- 新增：`internal/notification/dispatcher.go`、`internal/notification/budget.go`、`internal/notification/recovery.go`；对应`internal/notification/dispatcher_test.go`、`internal/notification/budget_test.go`、`internal/notification/recovery_integration_test.go`。
- 修改：`internal/notification/worker.go`、`internal/notification/service.go`、`internal/notification/server.go`、`internal/notification/store/queries/account_notifications.sql`、`internal/notification/store/queries/system_notifications.sql`、`internal/notification/store/queries/bot_updates.sql`。
**Interfaces**
- 产出delivery类型：`Candidate{Ref WorkRef; OwnerID string; ChatID int64; Group bool; NotBefore time.Time; Deadline *time.Time}`。
- 产出：`Budget.Next(c delivery.Candidate, now time.Time) time.Time`、`Budget.Start(c delivery.Candidate, at time.Time)`；内部保存Bot、chat及group窗口。另提供`Reserve(c delivery.Candidate,now time.Time) (reservationID uint64,ok bool)`、`Release(reservationID uint64)`；Reserve计入尚未started的预留容量，Start按Candidate.Ref原子消耗预留并写实际起点，防并发Next检查超售。
- 产出：`WorkSource.Ready(ctx context.Context,now time.Time) ([]delivery.Candidate,error)`与`WorkSource.Dispatch(ctx context.Context,c delivery.Candidate,onStarted func(time.Time)) error`；账户/系统/reply源使用任务2许可流程，任务11增加摘要首条源。
- 产出：`NewDispatcher(clock Clock,sources []WorkSource,budget *Budget,concurrency int) *Dispatcher`、`Dispatcher.Run(ctx context.Context) error`。Clock契约为`Now() time.Time`、`After(time.Duration) <-chan time.Time`；生产包装time，测试手动推进。
- 产出：`(*notificationstore.SQLStore).RecoverSender(ctx context.Context,stoppedIncarnation uuid.UUID) error`只在外部已确认旧进程停止后调用。

- [ ] **步骤1：写纯预算红灯测试。**

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

- [ ] **步骤2：用可控clock和阻塞fake Sender写调度失败测试。**10个owner不同chat同时就绪，断言能同时启动且不超过12；同chat第二条等前一尝试结束及1秒预算；一个owner12条不阻塞其他owner。账户/系统/reply共用20/秒预算，group窗口20/分钟。使用channel+clock推进，不写真实time.Sleep。
- [ ] **步骤3：替换全局1.1秒串行worker，接入WorkSource及许可流程。**按owner公平轮转并优先即将到期任务；看见尚未到NotBefore的未来Deadline时预留“HTTP最长5秒+chat间隔”的槽。取得预算前不拿账户gate，等锁后槽过期则重排；同chat只有一个执行中的许可。临时速率错误缩紧预算，unknown不回队。用单sender incarnation/session lease检测失锁，失锁后停止授权并退出；绝不因lease过期并行接管Telegram。

```sql
-- 只处理已确认停止的incarnation；其余kind使用相同CAS条件更新各自表。
UPDATE account_notification_deliveries d SET status = 'unknown'
FROM notification_delivery_attempts a
WHERE d.current_attempt_id = a.id AND d.status = 'sending'
  AND a.sender_incarnation = $1 AND a.outcome IS NULL;
```

有明确结果的旧attempt先补记该事实；没有结果才unknown，不能以重发探测成功与否。短期连接错误不能把旧permit当成未发送。

- [ ] **步骤4：运行 `go test -race ./internal/notification/...` 和 `go test -tags=integration ./internal/notification -run TestRecoverSender -count=1`；断言恢复不增加HTTP调用，停止时无泄漏goroutine/连接。**
- [ ] **步骤5：提交 `feat(notification): schedule deliveries fairly across chats`。**

## 任务5：精确身份、六区间资料与确认token

**Files**
- 新增：`internal/tradersync/types/types.go`、`internal/tradersync/target_resolver.go`、`internal/tradersync/profile.go`、`internal/tradersync/pnl.go`、`internal/tradersync/errors.go`、`internal/tradersync/store/sql_store.go`、`internal/tradersync/store/confirmations.go`、`internal/tradersync/store/queries/confirmations.sql`。
- 新增：`util/polymarket/profile_identity.go`、`util/polymarket/profile_stats.go`、`util/polymarket/user_pnl.go`。
- 修改：`util/polymarket/data.go`、`util/polymarket/gamma.go`、`sqlc.yaml`、权威`internal/accountstate/store/migrations/000001_init.sql`。
- 测试：`util/polymarket/profile_identity_test.go`、`util/polymarket/profile_stats_test.go`、`util/polymarket/user_pnl_test.go`、`internal/tradersync/target_resolver_test.go`、`internal/tradersync/pnl_test.go`、`internal/tradersync/store/confirmations_integration_test.go`。
- 依据：[Profile/P/L契约核验](../../requirements/polymarket-copy-trading/profile-pnl-contract-verification.md)。
**Interfaces**
- 以下任务统一将`internal/tradersync/types`导入为`tsmodel`，避免service→store→service循环。
- 产出类型：`Identity{Wallet common.Address; ProfileURL,DisplayName,AvatarURL string; Digest [32]byte}`；`Evidence{Availability,ReasonCode,Source string; QueriedAt time.Time}`；`Scalar{Evidence; Value *string}`；`PnLPoint{T int64; P string}`；`Curve{Evidence; Points []PnLPoint}`。数字以字符串保留原值，Value=nil代表缺失。
- Identity.Digest只摘要规范钱包、canonical身份映射及适配器版本；显示名、头像、收益变化不改变身份。完整确认卡资料另保存查询证据，不把辅助资料变化判成另一个目标。
- 产出类型：`ConfirmationCard{Identity Identity; Avatar,DisplayName,Verified,JoinedAt,PositionValue,LargestWin,Predictions Scalar; PnL map[string]PnLView; DefaultPeriod,UsageNotice string}`；`PnLView{Amount Scalar; Curve Curve; Interval,Fidelity string; Reference *time.Time; Timezone string}`；`ResolvedTarget{Card ConfirmationCard; Token string; ExpiresAt time.Time}`。
- 产出：`(*TargetResolver).Resolve(ctx context.Context,ownerID,input string) (tsmodel.ResolvedTarget,error)`；`Revalidate(ctx context.Context,identity tsmodel.Identity) error`。外部查询先完成再进入短事务保存token；当前grant在写入前重新核验。
- 产出：`PlanPNL(period string,allAge *time.Duration) (interval,fidelity string)`；`BuildPNL(period string,raw []tsmodel.PnLPoint,allAge *time.Duration,rules PNLRules) tsmodel.PnLView`；`PNLRules{Reference *time.Time; Zone *time.Location; Round func(*big.Rat) string}`。未知规则用nil表达，不自定官网算法。
- 产出store：`NewSQLStore(pool *pgxpool.Pool) *SQLStore`；`SaveConfirmation(ctx context.Context,ownerID string,identity tsmodel.Identity,tokenDigest []byte,expiresAt time.Time) error`；`ConsumeConfirmationTx(ctx context.Context,tx pgx.Tx,ownerID string,tokenDigest []byte,requestID string,identityDigest []byte) (tsmodel.Identity,error)`。

- [ ] **步骤1：写精确请求与缺失值红灯测试。**

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

- [ ] **步骤2：实现受限身份读取与类型化数值。**URL只允许HTTPS、`polymarket.com`/`www.polymarket.com`和已核准`/@handle`路径；拒绝userinfo、端口、伪域、额外路径及改变身份的query/fragment；最大响应2MiB、重定向最多3次且每跳验证相同规则、超时5秒。这些是输入防护默认值，不改变合法身份。HTTP适配器注入RoundTripper供测试，生产allowlist不由用户输入改变。解析固定版本SSR身份和canonical，与public-profile明确钱包交叉核验；冲突、缺关键结构、多个候选钱包均FailedPrecondition，不调用PublicSearch。

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

- [ ] **步骤3：实现六区间请求、裁切与独立availability。**fallback为1d/1h、1w/3h、1m/18h、all/1d；1Y/YTD共用ALL。已知ALL历史年龄时按证据表的H/N从[1,3,12,18,24]小时选最小误差，平局较小候选；1M严格<31天使用all。对实际t/p数组校验升序、原数组及裁切后至少2点，裁切边界包含，原p不平移。

```go
// 区间未舍入金额的核心；调用前已经验证两端存在及ALL年龄证据。
end,_:=new(big.Rat).SetString(points[len(points)-1].P)
begin,_:=new(big.Rat).SetString(points[0].P)
amount:=new(big.Rat).Set(end)
if period!="ALL" && !shortHistory { amount.Sub(end,begin) }
```

此片段位于BuildPNL内部：`points`是验证后裁切数组；`shortHistory`按报告各区间严格阈值计算，YTD使用报告的年初规则。Reference缺失使相关裁切不可验证；YTD缺时区只影响YTD；Round缺失使精确显示金额unavailable但保留可验证原始曲线。直接取得的官方显示原值可以有独立证据，不用排行榜/持仓求和代替。

- [ ] **步骤4：增加token和幂等存储，稳定SQL后生成。**`trader_sync_targets`钱包唯一；`trader_sync_target_confirmations`保存token SHA-256 digest、owner、identity_json/digest、expires_at、consumed_request_id；`trader_sync_request_results`以(owner,operation,request_id)唯一并保存payload_digest/result_json。token使用32字节crypto/rand，DB时钟5分钟；成功Create事务中消费，过期或其他request重复消费拒绝。

```sql
UPDATE trader_sync_target_confirmations
SET consumed_request_id = $3
WHERE owner_id = $1 AND token_digest = $2
  AND expires_at > clock_timestamp() AND consumed_request_id IS NULL
  AND identity_digest = $4
RETURNING identity_json;
```

在sqlc.yaml新增独立tradersync输出包（`internal/tradersync/store/sqlc`），schema仍指向权威目录；`make sqlc-local`后实现适配器。Resolver要求注入`GrantCheck func(context.Context,pgx.Tx,string) error`，在account gate内写token前调用；缺少该依赖构造失败。Consume由已经核验grant的Create事务调用，owner条件仍由SQL强制。此任务用显式授权/拒绝fake测试该调用，任务6提供真实`RequireGrantTx(ctx context.Context,tx pgx.Tx,ownerID string) error`，任务12才注册公共入口；不增加临时放行生产路径。

- [ ] **步骤5：运行 `go test ./util/polymarket ./internal/tradersync/...` 及对应confirmation集成测试。**覆盖精确URL、canonical冲突、结构变化、跨域重定向、身份变化/超时、owner错用、到期、不同request重消费；6区间逐项缺失、负值、真0、31日边界、30日裁切、365日、跨年及舍入证据缺失。提交 `feat(trader-sync): resolve targets with evidence-backed profile data`。

## 任务6：独立订阅、备注、十模块授权与原子撤权

**Files**
- 新增：`internal/tradersync/types/subscription.go`、`internal/tradersync/subscriptions.go`、`internal/tradersync/store/subscriptions.go`、`internal/tradersync/store/queries/subscriptions.sql`、`internal/tradersync/store/revocation.go`、`internal/accountstate/store/access_hooks.go`。
- 修改：`internal/accountaccess/access.go`、`internal/accountaccess/controller.go`、`internal/accountstate/store/sql_store.go`、`internal/accountstate/store/queries/account_access.sql`、权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/server/account/account.proto`、`internal/server/account/account.go`。
- 修改组合入口：`internal/server/athena-server.go`的`NewServer`，在账户store创建后、EnsureDevelopmentAccount与NewController之前注入真实AccessChangeHook；本步骤不依赖Collector。
- 修改：`ui/src/app/shared/access-modules.ts`、`ui/src/app/shared/account-access.ts`、`ui/src/app/shared/models.ts`、`ui/src/app/admin/accounts-service.ts`、`ui/src/app/admin/pages/admin-accounts.tsx`；核对`ui/src/app/shared/pages/account-center.tsx`及`ui/src/app/member/app.tsx`实际消费者。
- 测试：`internal/accountaccess/access_test.go`、`internal/accountstate/store/access_integration_test.go`、`internal/tradersync/subscriptions_test.go`、`internal/tradersync/store/subscriptions_integration_test.go`、`internal/server/account/account_test.go`、`ui/src/app/shared/access-modules.test.ts`、`ui/src/app/shared/account-access.test.ts`、`ui/src/app/shared/models.test.ts`、`ui/src/app/admin/pages/admin-accounts.test.tsx`。
**Interfaces**
- 消费：任务1 `txgate.WithAccountTx`、任务5 `ConsumeConfirmationTx`与`TargetResolver.Revalidate`；任务2旧delivery资格墓碑。
- 产出tsmodel：`Subscription{ID,OwnerID string; Wallet common.Address; DesiredState,ObservationState,Reason string; Revision,Generation uint64; EffectiveAt,EndedAt *time.Time; CreatedAt,UpdatedAt time.Time; Note,NotificationMode,QueueNotice string}`；`CreateInput{Token,RequestID string; Note *string}`；`ChangeInput{SubscriptionID,RequestID string; ExpectedRevision uint64}`；`NoteInput{Wallet common.Address; RequestID,Note string; ExpectedRevision uint64}`。
- 产出：`(*SubscriptionService).Create(ctx context.Context,ownerID string,in tsmodel.CreateInput) (tsmodel.Subscription,error)`；`Change(ctx context.Context,ownerID,action string,in tsmodel.ChangeInput) (tsmodel.Subscription,error)`，action只允许pause/resume/cancel；`UpdateNote(ctx context.Context,ownerID string,in tsmodel.NoteInput) error`；`ValidateNote(note string) error`。
- 产出：`accountstatestore.AccessChangeHook func(ctx context.Context,tx pgx.Tx,accountID string,previous,next accountaccess.Access) error`，通过`SetAccessChangeHook(hook AccessChangeHook)`在服务启动前一次注入；`(*tradersyncstore.SQLStore).RevokeTx(ctx context.Context,tx pgx.Tx,ownerID,reason string) error`。accountstate不导入tradersync业务包。
- 基线注册接口由本任务定义、任务9实现：`BaselineRegistrar.RegisterTx(ctx context.Context,tx pgx.Tx,sub tsmodel.Subscription) error`。Create/resume事务在account gate后调用，未连接也持久pending attempt；禁止提交后才异步补登记归属。

- [ ] **步骤1：写授权与Unicode备注红灯测试。**

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

- [ ] **步骤2：补齐schema约束和完整矩阵写入。**订阅表以(owner,wallet)非cancelled部分唯一，revision/generation>0；备注独立(owner,wallet)唯一且保留取消后数据。baseline_attempts/monitor_intervals在本批建立基础列：subscription/generation/epoch/filter_revision/expected_revision、registered_high、candidate_effective_at、state、effective_at/ended_at及原因，attempt状态pending/succeeded/failed单向结束。owner引用和订阅引用使用复合FK保持一致；request结果事务失败不落库。

```sql
CREATE UNIQUE INDEX trader_sync_one_live_target
ON trader_sync_subscriptions(owner_id,wallet)
WHERE desired_state <> 'cancelled';
-- 在账户gate内读配额，拒绝第11个；不限制整个产品第11名用户。
SELECT count(*) FROM trader_sync_subscriptions
WHERE owner_id=$1 AND desired_state<>'cancelled';
```

account_module_access的合法模块与grant检查、ReplaceAccountModuleAccess及Go读写参数均扩为十项；Trader Sync READ用CHECK和Go校验双重拒绝。`make sqlc-local`后适配stores。

- [ ] **步骤3：实现创建/状态/备注事务及并发测试。**Create先读取token身份并在锁外重新验证；进入account gate后核验当前grant→幂等→token/配额/唯一性→备注（nil沿用，指针空清空）→subscription→RegisterTx→消费token→成功结果。外部核验不能跨数据库锁。暂停/取消用事务实际时间关闭区间、revision+1；resume生成新generation且登记新pending attempt；自动网络恢复不经过resume。取消不可恢复，重建新ID沿用备注。

```go
func ValidateNote(note string) error {
    if !utf8.ValidString(note) || utf8.RuneCountInString(note)>20 {
        return status.Error(codes.InvalidArgument,"note exceeds 20 Unicode code points")
    }
    return nil
}
```

真实数据库两连接屏障测试同时创建第10/11项只一成功、同wallet仅一未取消、失败token不占配额、相同幂等返回同ID而不同payload拒绝。取消/重建及改备注均不重写历史、generation或基线。

- [ ] **步骤4：将撤权hook嵌入权限事务并验证不可复活。**controller全局更新mutex改为按账户串行，缓存发布仍在Commit后；DB revision CAS保留。UpdateAccountAccess在同account gate/tx内执行hook，RW→NONE时关闭区间、permission_disabled、取消无许可工作、给包括sending在内的旧Trader Sync delivery写永久墓碑。登录/API Key开关变化不触发产品撤权；重授不清墓碑、不自动resume。

```sql
UPDATE account_notification_deliveries
SET eligibility_revoked_at=COALESCE(eligibility_revoked_at,clock_timestamp()),
    eligibility_revoked_reason=COALESCE(eligibility_revoked_reason,$2),
    status=CASE WHEN status='pending' THEN 'cancelled' ELSE status END
WHERE account_id=$1 AND source='trader_sync';
```

未冻结摘要资格由任务10与其表一起接入RevokeTx；当前任务只操作已经存在的订阅、区间及delivery，不能引用未来尚未创建的表。任务10上线前尚无活动或摘要业务入口。sending原attempt结果照实记录，明确失败因墓碑cancelled。hook注入是必需启动依赖，缺失时server启动错误，不能静默漏撤权；使用fake registrar验证当前任务，不提前实现采集。

- [ ] **步骤5：生成权限proto并适配实际UI矩阵。**AccountDataModule使用空闲enum值12（6、10已reserved），不是因十模块而占用10；拆开grant校验和requirement校验。稳定后 `make protogen`，适配Go映射及UI共享allowedAccessLevels。非法READ从解析端fail-closed为NONE，绝不提升RW；选择器只NONE/RW，十项序列化/克隆/比较/概览一致。不增加Trader Sync业务导航或页面。

```ts
// access-modules.ts共享允许值；其他模块沿用已有maxAccess逻辑。
if (module.id === 'trader_sync') {
  return [AccountDataAccess.None, AccountDataAccess.ReadWrite]
}
```

共享函数命名为`allowedAccessLevels(module: AccountDataModuleDefinition): AccountDataAccess[]`，使用现有AccountDataAccess枚举。测试不仅检查选项，还检查parse→edit→serialize十项往返与RW→NONE缓存清理。

- [ ] **步骤6：运行 `go test ./internal/accountaccess ./internal/accountstate/... ./internal/tradersync/... ./internal/server/account`、上述真实数据库集成测试和 `yarn --cwd ui test --runInBand --watch=false`。**提交 `feat(trader-sync): manage isolated subscriptions and revocation`。

## 任务7：三Exchange成交解码、规范链确认与版本证据

**Files**
- 新增：`internal/tradersync/exchange_decode.go`、`internal/tradersync/confirmation.go`、`internal/tradersync/source_version.go`、`internal/tradersync/abi/embed.go`、`internal/tradersync/abi/core_exchange.json`、`internal/tradersync/abi/combos_exchange.json`、`internal/tradersync/abi/proxy.json`、`internal/tradersync/testdata/source_records.json`。
- 新增测试：`internal/tradersync/exchange_decode_test.go`、`internal/tradersync/confirmation_test.go`、`internal/tradersync/source_version_test.go`；新增类型`internal/tradersync/types/trade.go`。
- 依据：[已核准来源契约与部署值](../../requirements/polymarket-copy-trading/source-contract-verification.md)。ABI资源记录来源仓库commit及hash，不新增或修改`pkg/abi/**/*.sol`。
**Interfaces**
- 产出tsmodel：`Trade{Wallet common.Address; Side,PositionID,CollateralRaw,SharesRaw,FeeRaw,CollateralSymbol string; CollateralDecimals,SharesDecimals uint8; Exchange common.Address; SourceVersion,PriceNumerator,PriceDenominator string}`；`CanonicalEvidence{Status string; BlockHash common.Hash; SettledAt time.Time; CheckedAt time.Time; Reason string}`；Status=unverified/invalid/confirmed。
- 产出：`DecodeOwnTrade(log types.Log,version string) (tsmodel.Trade,error)`；`ConfirmReceived(ctx context.Context,node CanonicalRPC,raw types.Log) (tsmodel.CanonicalEvidence,error)`；`(*VersionVerifier).Verify(ctx context.Context,raw types.Log) (version string,err error)`。
- CanonicalRPC精确契约：`FinalizedHeader(context.Context) (*types.Header,error)`、`TransactionReceipt(context.Context,common.Hash) (*types.Receipt,error)`、`HeaderByHash(context.Context,common.Hash) (*types.Header,error)`、`HeaderByNumber(context.Context,*big.Int) (*types.Header,error)`；fake按调用记录，不请求真实链。

- [ ] **步骤1：固定已核准原始日志fixtures，写BUY/SELL及自身归属红灯。**fixtures逐项存address/topics/data/block/tx/logIndex及预期wallet/方向/原量/fee，至少三Exchange×BUY/SELL×Maker/Taker；同tx两个自身日志必须两个Trade；只topics[3]匹配、OrdersMatched及SPLIT/MERGE不得产出Trade。

```go
func TestUnknownExecutionVersionIsNotDecoded(t *testing.T) {
    _,err:=DecodeOwnTrade(types.Log{},"unrecognized-implementation")
    if err==nil { t.Fatal("decoded unknown execution version") }
}
```

运行 `go test ./internal/tradersync -run 'TestDecode|TestUnknownExecution' -count=1`确认红灯。fixture构造必须按来源ABI编码，不能把decoder输出当预期再反序列化。

- [ ] **步骤2：实现白名单版本解码及精确单位。**先核对chain/address/topic0/version，再按真实ABI解码自身资金钱包topics[2]、side、position及filled/fee原整数。BUY的抵押币/份额分别为maker/taker，SELL反转；6位pUSD与6位份额作为已核准版本属性保存，费用独立。无大于0的自定义门槛；零值合法性按合约事件语义处理，不能新增最小金额过滤。

```go
collateral,shares:=makerAmount,takerAmount
if side=="SELL" { collateral,shares=takerAmount,makerAmount }
trade.CollateralRaw=collateral.String()
trade.SharesRaw=shares.String()
trade.FeeRaw=feeAmount.String()
```

片段变量来自ABI解码后的`*big.Int`，trade为`tsmodel.Trade`；symbol/decimals从本次version注册项取得，不能从API字段名usdcSize推断。价格保存未含fee的collateral/shares精确比值（两个字符串），分母为0则价格不可用，不造0价格，不阻止保存可识别成交；当前两者decimals均6，未来版本按精度换算。

- [ ] **步骤3：实现确认后回执和已知哈希验证。**共享fresh finalized可覆盖候选后重新取known tx receipt；status=成功、规范高度头hash、receipt定位和原日志address/topics/data/index均一致，才取HeaderByHash时间。null/403/timeout返回unverified；removed或明确不同规范定位返回invalid。receipt其他日志仅核对，不建新候选。fake构造receipt有两条日志时只返回被请求原日志的确认事实，并断言先finalized后receipt。

```go
if raw.Removed { return tsmodel.CanonicalEvidence{Status:"invalid",Reason:"removed"},nil }
// 在finalized覆盖且receipt成功之后执行逐字节核验。
same:=received.Address==raw.Address && received.Index==raw.Index &&
    reflect.DeepEqual(received.Topics,raw.Topics) && bytes.Equal(received.Data,raw.Data)
if !same { return tsmodel.CanonicalEvidence{Status:"invalid",Reason:"canonical_log_changed"},nil }
```

- [ ] **步骤4：加入按blockHash的版本证据缓存。**HTTP adapter用StorageAtHash/CodeAtHash读取已知候选块及父块代理/实现，核对已核准升级记录语义；仅允许候选known blockHash的升级事件查询，不用latest槽证明旧执行。不认识实现或无法排除同块升级保持unverified，不能用块末实现盲解。该能力通过窄接口`VersionRPC`定义StorageAtHash、CodeAtHash及`UpgradeLogs(ctx context.Context,blockHash common.Hash,proxy common.Address) ([]types.Log,error)`；不提供任意from/to扫描接口。

```go
if parentImplementation!=blockImplementation || len(upgrades)>0 || !upgradeEvidenceComplete {
    return "",status.Error(codes.FailedPrecondition,"execution version unverified")
}
```

缓存键至少chain/exchange/blockHash；无法确认结果不当作永久无升级。CombinatorialModule资料版本失败归任务8metadata缺失，不阻塞已确认成交。

- [ ] **步骤5：运行 `go test ./internal/tradersync -run 'TestDecode|TestConfirm|TestVersion|TestUnknownExecution' -count=1`。**覆盖旧hash/新规范块、403/null、removed、已确认后深度重组隔离所需证据、同块升级/未知实现；提交 `feat(trader-sync): verify and decode canonical source trades`。

## 任务8：普通市场、Combo逻辑及持久目录资料

**Files**
- 新增：`internal/tradersync/market_metadata.go`、`internal/tradersync/combo_metadata.go`、`internal/tradersync/abi/combinatorial_module.json`、`internal/tradersync/abi/binary_module.json`、`internal/tradersync/store/metadata.go`、`internal/tradersync/store/queries/metadata.sql`、`util/polymarket/combo_markets.go`。
- 新增类型：`internal/tradersync/types/metadata.go`。
- 修改：`util/polymarket/gamma.go`、权威`internal/accountstate/store/migrations/000001_init.sql`。
- 测试：`internal/tradersync/market_metadata_test.go`、`internal/tradersync/combo_metadata_test.go`、`internal/tradersync/store/metadata_integration_test.go`、`util/polymarket/combo_markets_test.go`。
**Interfaces**
- 产出tsmodel：`MarketRef{Evidence; ID,Title,URL,ConditionID,PositionID,Outcome string}`；`ComboLeg{PositionID string; Market MarketRef}`；`TradeMetadata{Market MarketRef; LegsEvidence Evidence; Legs []ComboLeg; Relationship string}`。腿数仅LegsEvidence.available时由len(Legs)取得。
- 产出：`(*MetadataResolver).Resolve(ctx context.Context,trade tsmodel.Trade,blockHash common.Hash) tsmodel.TradeMetadata`；`(*DirectoryRefresher).Run(ctx context.Context) error`。
- 产出公开适配：`ComboMarketPage{Markets []ComboMarket; NextCursor string}`、`ComboMarket{ID string; PositionIDs []string; ConditionID string}`；`(*GammaClient).ListComboMarkets(ctx context.Context,cursor string,limit int) (ComboMarketPage,error)`；在util/polymarket定义这些类型。

- [ ] **步骤1：写组合逻辑和缺失整腿红灯。**

```go
func TestComboNoMeansComplementOfConjunction(t *testing.T) {
    got:=ComboRelationship("NO")
    if got!="NOT(AND(legs))" { t.Fatal(got) }
    if ComboRelationship("YES")!="AND(legs)" { t.Fatal("wrong YES relationship") }
}
```

定义`ComboRelationship(outcome string) string`，只对已核验YES/NO返回该表达式，其余未知为空并附reason。用HTTP/RPC fake返回getLegs失败，断言LegsEvidence=unavailable而不是available空数组/0；运行 `go test ./internal/tradersync ./util/polymarket -run 'TestCombo|TestMarket' -count=1`。

- [ ] **步骤2：实现精确市场与两类腿映射。**普通token查询考虑open/closed，并按精确token下标取Outcome。迁移Binary腿执行legacyConditionId/getLegacyPositionId→旧CTF token→Gamma；其他已核准module腿用目录PositionId→market ID，再GetMarketByID核对positionIds、condition、Outcome，不以名字或当前持仓猜测。PositionID保留十进制字符串，Gamma类型增加有presence的positionIds解析。

```go
func ComboRelationship(outcome string) string {
    switch outcome { case "YES": return "AND(legs)"; case "NO": return "NOT(AND(legs))" }
    return ""
}
```

getLegs在已知块hash调用并保留module版本证据；metadata错误不改变TRADE方向/金额。已知N腿部分缺失时仍保留N条位置及各自状态，关系针对组合自身Outcome；不是每腿分别BUY/SELL。

- [ ] **步骤3：实现有界分页目录和资料缓存。**新增`trader_sync_market_metadata`、`trader_sync_combo_leg_index`、`trader_sync_directory_refresh`；索引以position_id/market_id核验关系持久保存，不能刷新时删除已关闭记录。每页最多100，当前cursor/round_started_at/next_page_at在同事务推进；失败保留cursor，最多1page/s，10分钟轮次不重叠。

```sql
UPDATE trader_sync_directory_refresh
SET cursor=$2,next_page_at=clock_timestamp()+interval '1 second'
WHERE name='combo_markets' AND cursor=$1;
```

该CAS与本页映射upsert同事务；最后一页另保存round_completed_at和下轮时刻。稳定SQL后 `make sqlc-local`。并发资料默认4；热路径缓存miss返回可见缺失，不能遍历目录阻塞活动。

- [ ] **步骤4：运行本任务单元/集成测试。**覆盖两迁移Binary腿、module1/2、已关闭市场、空持仓、目录分页重启/失败/重复页、超大ID及unknown Outcome；验证晚补资料只变显示数据。提交 `feat(trader-sync): resolve market and combo metadata precisely`。

## 任务9：注册先于过滤的基线、共享WSS与持久接收

**Files**
- 新增：`internal/tradersync/baseline.go`、`internal/tradersync/collector.go`、`internal/tradersync/intake.go`、`internal/tradersync/config.go`、`internal/tradersync/rpc/session.go`、`internal/tradersync/rpc/http.go`、`internal/tradersync/store/intake.go`、`internal/tradersync/store/baselines.go`、`internal/tradersync/store/queries/intake.sql`、`internal/tradersync/store/queries/baselines.sql`。
- 新增类型：`internal/tradersync/types/candidate.go`。
- 修改：权威`internal/accountstate/store/migrations/000001_init.sql`；以`util/ethws/dial.go`的真实实现作核对入口，不复用其自动重连语义。
- 测试：`internal/tradersync/baseline_test.go`、`internal/tradersync/collector_test.go`、`internal/tradersync/rpc/session_test.go`、`internal/tradersync/store/intake_integration_test.go`、`internal/tradersync/store/baselines_integration_test.go`。
**Interfaces**
- 消费：任务6 `BaselineRegistrar.RegisterTx`，任务7CanonicalRPC；原始接收不依赖metadata。
- 产出：`ComputeBaseline(now time.Time,registeredHigh uint64,h *types.Header) (time.Time,error)`；`(*Collector).Run(ctx context.Context) error`；`(*Intake).Persist(ctx context.Context,epoch uint64,raw types.Log,receivedAt time.Time) error`。
- 产出rpc：`DialSession(ctx context.Context,endpoint,proxyURL string) (*Session,error)`；`(*Session).Subscribe(ctx context.Context,query ethereum.FilterQuery) (subscriptionID string,error)`；`Unsubscribe(ctx context.Context,id string) error`；`Logs() <-chan types.Log`；`Done() <-chan struct{}`；`Err() error`；`Close() error`。每个Session仅一物理WSS，绝不内部重连。
- 产出tsmodel：`Candidate{SourceID int64; SubscriptionID,OwnerID string; Generation uint64; AttemptID int64; ReceivedAt time.Time}`；`Eligibility{Generation uint64; BaselineSucceeded bool; SettledAt,EffectiveAt time.Time; EndedAt *time.Time}`。`Eligible(c tsmodel.Eligibility,s tsmodel.Subscription) bool`只判原区间+当前意图/代次，不判当前epoch健康。

- [ ] **步骤1：写秒级边界红灯测试。**

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

- [ ] **步骤2：实现基线纯函数和持久状态协议。**

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

RegisterTx在account→wallet gate内记录attempt与当前已观察最高高度，安装过滤前完成；最高高度包含本会话read loop已解出而尚在持久化队列中的该钱包日志，不能只看已写DB高度。未连接attempt保存pending，不能沿用失败attempt。全部相关过滤ACK后新查latest、保存候选未来边界，等到边界再同account gate复核grant/revision/generation/epoch并CAS succeeded+interval。提交结果未知先读真实状态，不能把可能已成功attempt覆盖为failed。已确定未成功的重启attempt终结并新建。

- [ ] **步骤3：实现显式WSS会话与共享过滤。**用现有gorilla/websocket写一条read loop及串行JSON-RPC写队列，request ID路由ACK，notifications推送有界队列；15秒发送随机关联ping，5秒内只有同nonce pong才成功。队列满/DB无法持久接收关闭epoch并显示可能遗漏，不静默丢日志。HTTP latest成功不得覆盖WSS pong失败，无匹配日志保持健康。

```go
// 普通CTF/NegRisk共享一组，Combos独立一组；walletTopics每组<=100。
filter:=ethereum.FilterQuery{
    Addresses:coreAddresses,
    Topics:[][]common.Hash{{orderFilledTopic},nil,walletTopics},
}
```

地址/topic来自任务7版本注册；只topics[2]筛目标。目标集合全局去重，新增过滤ACK后删旧过滤；新增目标失败不拆已有有效过滤、不重建其他用户边界。外层Collector按配置退避重连、建立新epoch/实时边界；单实例持久fencing每次写入CAS，过时代次拒绝写入。

- [ ] **步骤4：实现raw第一次落库与候选归属事务。**新增`trader_sync_collector_epochs`、`trader_sync_interruptions`、`trader_sync_source_records`、`trader_sync_source_candidates`；source定位五元唯一。钱包gate内先验证epoch fencing，再插raw+首次候选；重复raw不能添加后来注册的订阅，removed=true必须更新证据。baseline失败的候选保留原attempt，不能归入新attempt。

```sql
INSERT INTO trader_sync_source_records
  (chain_id,exchange_address,block_hash,transaction_hash,log_index,raw_json,received_at,removed)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (chain_id,exchange_address,block_hash,transaction_hash,log_index)
DO UPDATE SET removed=trader_sync_source_records.removed OR EXCLUDED.removed
RETURNING id;
```

是否首次必须由独立INSERT DO NOTHING RETURNING或显式事务查询判定，不能凭upsert返回ID为每次补候选。source永久原事实与可变确认/removed证据分列；失败回滚意味着未可靠接收，记录中断。稳定SQL后 `make sqlc-local`。

- [ ] **步骤5：运行WSS和两连接集成故障矩阵。**loopback WSS在ACK前后推日志，断言注册无空窗；错误pong/静默连接/队列满/DB断流触发可见中断。单次finality失败仅积压，WSS不关；恢复后只收新推送，旧成功attempt持久候选仍可处理，failed attempt不可复活。spy允许known-block升级查询，拒绝任何历史范围扫描和从receipt其他日志造候选。

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
- 修改：权威`internal/accountstate/store/migrations/000001_init.sql`、`internal/tradersync/store/revocation.go`、`internal/notification/store/queries/account_notifications.sql`、`internal/notification/store/attempts.go`。
- 测试：`internal/tradersync/projector_test.go`、`internal/tradersync/store/activities_integration_test.go`、`internal/notification/store/transactional_enqueue_integration_test.go`。
**Interfaces**
- 消费：任务7确认/版本/Trade、任务8TradeMetadata、任务9Candidate/Eligible、任务1account gate。
- 产出tsmodel：`Activity{ID int64; OwnerID,SubscriptionID string; SourceID int64; Trade Trade; Metadata TradeMetadata; NoteSnapshot string; SettledAt,ReceivedAt,RecordedAt time.Time; Generation uint64}`；`Projection{Candidate Candidate; Trade Trade; Confirmation CanonicalEvidence; Metadata TradeMetadata}`。
- 产出：`(*Projector).Run(ctx context.Context) error`；`(*tradersyncstore.SQLStore).Project(ctx context.Context,input tsmodel.Projection) (activityID int64,created bool,err error)`；`ClassifyActivity(windowCount int64) string`返回ordinary/summary。
- 产出通知事务入口：`(*notificationstore.SQLStore).EnqueueAccountTx(ctx context.Context,tx pgx.Tx,in delivery.AccountEnqueue) (int64,error)`。`AccountEnqueue{OwnerID,Source string; ActivityID int64; BindingRevision uint64; ChatID int64; Payload []byte; RecordedAt time.Time}`放delivery叶子包；调用方已在同事务核验并冻结绑定，不二次读取新绑定。source='trader_sync'，其他有效来源不加Trader Sync grant。
- 产出schema固定：`trader_sync_alert_memberships(activity_id bigint PK,owner_id,binding_revision,chat_id,form,state,created_at,eligibility_revoked_at,reason,batch_id nullable)`。form仅ordinary/summary；unbound不插membership。ordinary同事务建delivery；summary只建资格，任务11冻结。owner与activity复合FK，批次FK在任务11添加。

- [ ] **步骤1：写第10/11条与窗口端点红灯测试。**

```go
func TestClassificationRetainsFirstTenOrdinary(t *testing.T) {
    for i:=int64(1); i<=12; i++ {
        want:="ordinary"; if i>10 { want="summary" }
        if got:=ClassifyActivity(i); got!=want { t.Fatal(i,got) }
    }
}
```

真实DB测试用同owner记录时间t-60秒、t-60秒+1微秒和当前t，断言左端排除、右端包含；同秒用ID稳定顺序。运行 `go test ./internal/tradersync -run TestClassification -count=1` 和新集成测试确认红灯。

- [ ] **步骤2：实现活动/外发资格同事务和持久唯一约束。**新增`trader_sync_activities`，(subscription_id,source_record_id)唯一，owner/订阅/原interval引用一致，trade_json与note_snapshot不可变，metadata展示引用独立。取得account gate后重查数据库grant、当前enabled、generation及成功原区间；原epoch关闭不失去资格。普通消息渲染入口定义`RenderActivity(activity tsmodel.Activity,siteURL string) (string,error)`，在`internal/tradersync/render.go`实现，输出纯文本：目标备注/名称和完整wallet、TRADE及BUY/SELL、市场/Outcome/精确价格比值、份额/金额/币种/独立fee、明确“结算时间”、市场链接及站内详情。缺市场保留PositionID与缺失原因；Combo标明组合自身Outcome、整体关系及腿资料状态，完整腿在站内详情保留。长标题允许显式省略显示字符但保留身份与完整链接，不静默截断交易事实。

```sql
-- 先在gate内读取clock_timestamp()作为本次recorded_at，再插活动。
SELECT count(*) FROM trader_sync_activities
WHERE owner_id=$1 AND recorded_at>$2::timestamptz-interval '60 seconds'
  AND recorded_at<=$2;
```

查询包含刚插入活动、未绑定活动和所有目标。重复插入不重复计数/规划资格；无当前binding则只活动，有binding则固定revision/chat与备注。暂停/取消和外发分开：旧活动ordinary/summary继续，尚未投影候选终止。插入到Commit耗时计入延迟，成功Commit才通知后续worker。

- [ ] **步骤3：实现并行资料预算、重组隔离及晚资料更新。**metadata与finality并行；确认完成后最多额外2秒，超时保留字段unavailable并形成活动。未知Exchange执行版本仍raw/unverified；不是metadata超时理由。后台补资料仅更新可变metadata，不修改备注、原量、payload、不追加通知。

```go
func ClassifyActivity(windowCount int64) string {
    if windowCount<=10 { return "ordinary" }
    return "summary"
}
```

确认后深度重组以chain+tx证据建立`trader_sync_finality_anomalies`；在投影新分叉前检查曾发布该tx的冲突证据，隔离该tx后续新分叉候选，原activity保留异常。不能仅以txHash/logIndex跨分叉去重，也不能把同一规范tx多个自身日志合并。

- [ ] **步骤4：稳定活动/通知SQL后 `make sqlc-local`，适配EnqueueAccountTx和RevokeTx。**事务内永久撤销所有旧summary membership；解绑/重绑亦终结旧revision。测试两个owner共享同一source得到两份独立活动/备注/资格，而owner内部同source只有一份。
- [ ] **步骤5：运行 `go test -race ./internal/tradersync/... ./internal/notification/...` 和对应真实DB测试。**覆盖投影/暂停/撤权/重绑屏障竞争、activity插入后事务回滚、无binding后再绑定不补发、取消旧队列继续、newgeneration丢旧候选、100关系及全不同目标；提交 `feat(trader-sync): persist activities and alert eligibility atomically`。

## 任务11：不可变完整摘要、分条结果与首条gate

**Files**
- 新增：`internal/tradersync/summary.go`、`internal/tradersync/types/summary.go`、`internal/tradersync/store/summaries.go`、`internal/tradersync/store/queries/summaries.sql`、`internal/notification/store/summary_heads.go`、`internal/notification/summary_source.go`。
- 修改：`internal/tradersync/render.go`、`internal/accountstate/txgate/gate.go`、`internal/notification/store/attempts.go`、`internal/notification/dispatcher.go`、权威`internal/accountstate/store/migrations/000001_init.sql`。
- 测试：`internal/tradersync/render_test.go`、`internal/tradersync/summary_test.go`、`internal/tradersync/store/summaries_integration_test.go`、`internal/notification/summary_source_integration_test.go`、`internal/accountstate/txgate/session_integration_test.go`。
**Interfaces**
- 消费：任务4 `WorkSource`、任务2 `Permit`/Sender实际started回调、任务10memberships。
- 产出：`SummaryWindow(oldest time.Time,previousStart *time.Time) (notBefore,deadline time.Time)`；`RenderSummary(items []tsmodel.Activity,batchID string,siteURL string) ([]tsmodel.RenderedPart,error)`；`RenderedPart{Index,Total int; Text string; ActivityIDs []int64; PayloadDigest []byte}`。
- 产出：`txgate.AcquireAccountSession(ctx context.Context,pool *pgxpool.Pool,ownerID string) (*AccountSession,error)`；`AccountSession{Conn *pgxpool.Conn}`；`(*AccountSession).Release(ctx context.Context) error`。同一账户键与普通事务锁相互排斥。
- 产出：`(*notificationstore.SQLStore).AuthorizeTx(ctx context.Context,tx pgx.Tx,ref delivery.WorkRef,incarnation uuid.UUID) (delivery.Permit,error)`；任务2普通Authorize包装account事务并复用它，摘要在session固定连接内BeginTx调用，不能持session锁后向另一个连接请求account锁。
- 产出：`(*SummarySource).Ready(ctx context.Context,now time.Time) ([]delivery.Candidate,error)`；`Dispatch(ctx context.Context,c delivery.Candidate,onStarted func(time.Time)) error`，实现WorkSource；摘要首条ready给出未来notBefore/deadline，后续部分普通许可。

- [ ] **步骤1：写首条时间窗口及完整分条红灯。**

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

- [ ] **步骤2：实现窗口和冻结schema。**新增`trader_sync_summary_batches`、`trader_sync_summary_parts`、`trader_sync_summary_part_items`；batch保存owner/binding_revision/oldest/first_started_at/恢复基点，part保存不可变text/digest/index/total及delivery_id，member固定batch唯一。group key含wallet/note_snapshot/market/Outcome/side。默认纯文本Telegram正文（不设parse_mode），同时以Unicode code point和UTF-16 code unit保守计数≤4096来分完整展示行，链接以完整URL显示；超长单行拆成有明确延续标识的多行，不能只取前4096字符。

```go
func SummaryWindow(oldest time.Time,previousStart *time.Time) (time.Time,time.Time) {
    earliest:=oldest
    if previousStart!=nil && previousStart.Add(time.Minute).After(earliest) {
        earliest=previousStart.Add(time.Minute)
    }
    return earliest,oldest.Add(time.Minute)
}
```

先确定所有parts再写最终n/m，若页码增加使边界变化重新切分至稳定；部分与activity多对多，不把跨部分activity只关联首条。稳定SQL后 `make sqlc-local`。

- [ ] **步骤3：实现冻结至实际started的短session gate。**worker/chat/Bot槽在锁前准备；取专用连接的session advisory lock，BeginTx核验资格、冻结此刻全部同revision待汇总成员、写parts与首条许可、Commit，紧接进入一次Sender。RoundTrip入口started握手后补记起点再释放gate，结果等HTTP完成另开短CAS。RecordStarted只更新attempt/start事实，不再次取得account gate；否则会与持有者自锁。获取槽后锁等待超出槽有效性时释放并重新排队，未取得有效许可不能发。

```sql
-- AcquireAccountSession使用固定连接，键与WithAccountTx完全相同。
SELECT pg_advisory_lock(hashtextextended('athena:account:' || $1::text,0));
-- 只有该连接释放成功才能归还pool；失败则Hijack并Close。
SELECT pg_advisory_unlock(hashtextextended('athena:account:' || $1::text,0));
```

若本地校验未进HTTP，发送结果返回且无started，释放gate并按明确失败处理；若started写库失败，短deadline后关闭/释放连接并记录缺失，在进程内用已观测monotonic起点维持间隔。不能持锁重试至HTTP回执。重启必须确认旧sender停止；起点无证据时采用恢复时刻为下一批保守基点并标异常，不伪造first_started_at。

- [ ] **步骤4：写冻结竞态与混合部分结果集成测试。**用真实两连接：首条冻结完成后阻塞HTTP入口，同时发起新activity投影；断言其无法获得recorded_at，直至started已记录且gate释放；HTTP响应仍阻塞时允许新活动/撤权完成。注入started持久失败仍明确ACK成功，断言sent且started缺失；成功/unknown部分不重发，临时失败只重试该部分≤5次，整批只有all sent成功。
- [ ] **步骤5：验证调度在旧批未完时保留下批首条预算。**可控clock运行第10/11条、紧贴60秒成员、多个owner、Bot回复/系统消息竞争及同owner12条集中；检查起点间隔、最老成员deadline、公平性、所有部分完整度和失约计数。对本地渲染/锁/调度造成miss保留失败样本，不改recorded_at或cohort制造通过。执行本任务单元、真实DB与`-race`测试，提交 `feat(trader-sync): deliver complete summaries with durable timing boundaries`。

## 任务12：公共API、隐私查询、配置与进程组合

**Files**
- 新增：`pkg/apis/application/v1alpha1/trader_sync_types.go`、`pkg/apis/application/v1alpha1/trader_sync_protomessage.go`、`internal/server/tradersync/tradersync.proto`、`internal/server/tradersync/tradersync.go`、`internal/tradersync/service.go`、`internal/tradersync/pagination.go`、`internal/tradersync/store/reads.go`、`internal/tradersync/store/queries/reads.sql`。
- 修改：`internal/server/athena-server.go`、`internal/server/authz.go`、`cmd/athena-server/commands/athena-server.go`、`cmd/athena-notification/commands/athena_notification.go`、`internal/notification/server.go`、`internal/tradersync/config.go`、`hack/local-runtime.sh`、`docker-compose.prod.yml`；只有实际生成契约发现问题才调整`hack/generate-proto.sh`的精确schema处理。
- 生成：`pkg/apis/application/v1alpha1/generated.proto`、`pkg/apis/application/v1alpha1/generated.pb.go`、`pkg/apis/application/v1alpha1/generated.protomessage.pb.go`、`pkg/apis/application/v1alpha1/zz_generated.deepcopy.go`、`pkg/apiclient/tradersync/tradersync.pb.go`、`pkg/apiclient/tradersync/tradersync.pb.gw.go`、`assets/swagger.json`。
- 测试：`internal/server/tradersync/tradersync_test.go`、`internal/server/tradersync/contract_test.go`、`internal/server/authz_test.go`、`internal/tradersync/pagination_test.go`、`internal/tradersync/service_test.go`、`internal/tradersync/store/reads_integration_test.go`。
**Interfaces**
- 消费：任务5Resolve、任务6SubscriptionService、任务9Collector.Run、任务10Projector.Run、任务8DirectoryRefresher.Run与任务11SummarySource。
- 产出业务组合：`NewService(cfg Config,deps Dependencies) (*Service,error)`、`(*Service).Run(ctx context.Context) error`、`Close() error`。Dependencies明确含`Pool *pgxpool.Pool`、`Resolver *TargetResolver`、`Subscriptions *SubscriptionService`、`Collector *Collector`、`Projector *Projector`、`Directory *DirectoryRefresher`；各组件由同一composition root组装，不在业务方法内创建新pool。
- 产出handler：`New(service *tradersync.Service) *Server`；全部RPC标准`(context.Context,*Request)(*Response,error)`，精确消息字段如下表。内部Service转发会员方法时显式传owner；管理员读取另用仅概要的store方法，不复用Activity。
- 游标：`Cursor{OwnerID,FilterDigest,Direction,Time,ID string}`、`EncodeCursor(c Cursor,key []byte) (string,error)`、`DecodeCursor(token string,key []byte,ownerID,filterDigest,direction string) (Cursor,error)`；HMAC-SHA256保护规范JSON，base64url编码payload和签名，constant-time核验。

所有API资源ID/金额/链上position用string，revision用uint64并验证实际gateway表示；时间用UTC RFC3339Nano字符串。请求/响应采用以下消息，不增加account_id。列表`PageInput{int32 page_size; string cursor}`，`PageInfo{string next_cursor}`；具体DTO使用前面领域结构的明确protobuf字段号及独立availability wrapper，不能把map[string]any暴露为契约。

application DTO契约如下；每个struct按表中顺序分配连续字段号，从1开始，嵌套对象使用独立struct。可选值用指针/nullable消息，重复字段保留nil与availability，JSON tag使用camelCase。

| DTO | 字段契约 |
| --- | --- |
| TraderSyncFieldEvidence | availability、reasonCode、source、queriedAt；均string。 |
| TraderSyncStringField / DecimalField / BoolField / TimeField | evidence及可选value；分别string/string/bool/UTC时间字符串。DecimalField不转浮点。 |
| TraderSyncCurve | evidence、points（每点t为Unix秒字符串、p为精确金额字符串）。 |
| TraderSyncPnLView | period、amount（DecimalField）、curve、interval、fidelity、referenceTime（TimeField）、timezone（StringField）。 |
| TraderSyncResolvedTarget | wallet、canonicalProfileURL、avatar/displayName（StringField）、verified（BoolField）、joinedAt（TimeField）、positionValue/largestWin/predictions（DecimalField）、pnl（六项PnLView）、defaultPeriod=1Y、confirmationToken、expiresAt、usageNotice。 |
| TraderSyncTargetNote | wallet、note、revision；取消后保留。 |
| TraderSyncSubscription | id、wallet、status、revision、generation、note、noteRevision、createdAt/updatedAt/pausedAt/cancelledAt/permissionDisabledAt、intervals、observation、notificationMode、queueNotice、queueCounts。status是六种用户状态的投影。 |
| TraderSyncInterval / Observation | Interval含effectiveAt、endedAt及generation/epoch；Observation含state、reason、lastReliableAt、interruptions。中断项含start/end/recoveredAt（均可缺）、reason、uncertainty、possibleMissing=true；不提供推测遗漏数量。 |
| TraderSyncActivity | id、subscriptionId、sourceRecordId、wallet、side、positionId、collateralRaw/sharesRaw/feeRaw、collateralSymbol及两种decimals、priceNumerator/priceDenominator和priceEvidence、sourceVersion、settledAt/receivedAt/recordedAt、publicTimeEvidence、metadata、noteSnapshot、deliveries、summaryProgress。 |
| TraderSyncTradeMetadata | market（MarketRef）、legsEvidence、legs（positionId及MarketRef）、relationship；MarketRef含evidence/id/title/url/conditionId/positionId/outcome。 |
| TraderSyncDelivery / SummaryProgress | Delivery含id、status、reason、authorizedAt、startedAt（可缺）、resultAt、messageId、partIndex/partTotal、attempts；SummaryProgress含batchId、total/sent/failed/unknown/cancelled/pending/sending数量、oldestAt、firstStartedAt（可缺）、各部分结果。 |
| TraderSyncSubscriptionSummary | subscriptionId、accountId/username/email、wallet、status、生命周期时间、observation、activityCount、sentCount/failedCount/unknownCount/cancelledCount及pendingCount；无note、消息或活动正文。 |
| TraderSyncRuntimeStatus | collector连接/epoch/filter概要、关系数/目标数、raw/确认/投影积压、metadata缺失、队列/结果/时钟异常数量；不含钱包逐条事件或payload。 |

Resolve的usageNotice固定说明“优先选择低频交易者；高频监控不纳入性能保障”，Subscription.notificationMode区分站内与Telegram，暂停/取消的queueNotice明确“已排队通知仍会继续发送，可能稍后收到”。本任务提供后端契约，后续页面任务负责展示，不能将页面验收提前记为完成。

| RPC / 权限 | 请求消息字段 | 响应消息字段 / HTTP |
| --- | --- | --- |
| ResolveTarget / read | ResolveTargetRequest{input:string} | ResolveTargetResponse{target:TraderSyncResolvedTarget}；POST `/api/v1/trader-sync/targets:resolve` |
| CreateSubscription / write | CreateSubscriptionRequest{confirmation_token:string,request_id:string,note:TraderSyncNoteInput} | CreateSubscriptionResponse{subscription:TraderSyncSubscription}；POST `/api/v1/trader-sync/subscriptions` |
| ListSubscriptions / read | ListSubscriptionsRequest{page:PageInput,state:string} | ListSubscriptionsResponse{subscriptions:repeated TraderSyncSubscription,page:PageInfo}；GET `/api/v1/trader-sync/subscriptions` |
| GetSubscription / read | GetSubscriptionRequest{subscription_id:string} | GetSubscriptionResponse{subscription:TraderSyncSubscription}；GET `/api/v1/trader-sync/subscriptions/{subscription_id}` |
| PauseSubscription / write | PauseSubscriptionRequest{subscription_id:string,expected_revision:uint64,request_id:string} | PauseSubscriptionResponse{subscription:TraderSyncSubscription}；POST `/api/v1/trader-sync/subscriptions/{subscription_id}:pause` |
| ResumeSubscription / write | ResumeSubscriptionRequest{subscription_id:string,expected_revision:uint64,request_id:string} | ResumeSubscriptionResponse{subscription:TraderSyncSubscription}；POST `/api/v1/trader-sync/subscriptions/{subscription_id}:resume` |
| CancelSubscription / write | CancelSubscriptionRequest{subscription_id:string,expected_revision:uint64,request_id:string} | CancelSubscriptionResponse{subscription:TraderSyncSubscription}；POST `/api/v1/trader-sync/subscriptions/{subscription_id}:cancel` |
| UpdateTargetNote / write | UpdateTargetNoteRequest{wallet:string,note:string,expected_revision:uint64,request_id:string} | UpdateTargetNoteResponse{note:TraderSyncTargetNote}；PATCH `/api/v1/trader-sync/targets/{wallet}/note` |
| ListActivities / read | ListActivitiesRequest{page:PageInput,subscription_id:string,from:string,to:string} | ListActivitiesResponse{activities:repeated TraderSyncActivity,page:PageInfo}；GET `/api/v1/trader-sync/activities` |
| GetActivity / read | GetActivityRequest{activity_id:string} | GetActivityResponse{activity:TraderSyncActivity}；GET `/api/v1/trader-sync/activities/{activity_id}` |
| ListSubscriptionSummaries / administrator | ListSubscriptionSummariesRequest{page:PageInput,account_id:string,state:string} | ListSubscriptionSummariesResponse{summaries:repeated TraderSyncSubscriptionSummary,page:PageInfo}；GET `/api/v1/admin/trader-sync/subscriptions` |
| GetSubscriptionSummary / administrator | GetSubscriptionSummaryRequest{subscription_id:string} | GetSubscriptionSummaryResponse{summary:TraderSyncSubscriptionSummary}；GET `/api/v1/admin/trader-sync/subscriptions/{subscription_id}` |
| GetTraderSyncRuntimeStatus / administrator | GetTraderSyncRuntimeStatusRequest{} | GetTraderSyncRuntimeStatusResponse{status:TraderSyncRuntimeStatus}；GET `/api/v1/admin/trader-sync/status` |

管理员列表的account_id是已批准概要过滤，仍绑定当前管理员身份的游标；“无account_id”约束针对会员请求。API facade内只用`session.AccountID(ctx)`，不能使用任意请求字段作为会员owner。

- [ ] **步骤1：写真实gateway表示与授权登记红灯测试。**Create请求分别`{"confirmationToken":"x","requestId":"a"}`、`{"confirmationToken":"x","requestId":"b","note":{"value":""}}`，断言传给业务的Note分别nil/指针空串。positionId=`90071992547409931234567890`经过实际gateway JSON往返保持字符串，available真0与unavailable nil不同。所有13个RPC都登记对应鉴权，未知方法拒绝。

```proto
message TraderSyncNoteInput { string value = 1; }
message CreateSubscriptionRequest {
  string confirmation_token = 1;
  string request_id = 2;
  TraderSyncNoteInput note = 3;
}
```

note wrapper保留存在性，不能用GetNote().GetValue()丢失nil；现有gogofast链未经验证不改用proto3 optional。运行新handler/contract测试确认方法尚未实现导致红灯。

- [ ] **步骤2：实现会员和管理员分开的SQL读取与游标。**会员读在account gate事务内验证当前grant，再查owner数据；跨owner和不存在均NotFound。管理员只选择账户身份、wallet、状态/生命周期、健康/中断及活动/结果数量，不选note/活动正文/payload/逐条delivery。分页0→50、1–100合法、负值或>100拒绝，cursor绑定身份、筛选及方向。

```sql
-- member查详情必须有owner条件；管理员另写聚合投影。
SELECT * FROM trader_sync_activities WHERE owner_id=$1 AND id=$2;
```

所有幂等结果读取之前也重查当前grant；冲突Aborted、配额ResourceExhausted、重复AlreadyExists、坏输入InvalidArgument、坏token FailedPrecondition。授权前完成的读取可以晚返回，不持锁等客户端收包。稳定SQL后 `make sqlc-local`。

- [ ] **步骤3：编写DTO/proto源并生成。**确认卡用具体String/Decimal/Bool/Time/Curve wrapper，统一availability/reasonCode/source/queriedAt；列表/详情包含有效区间、旧队列、分条结果、authorized/started/result时间和缺失状态。新增非生成ProtoMessage声明遵循现有notification_protomessage.go build tag。输入稳定后 `make protogen`、`make clientgen`；检查go-to-protobuf、gogofast、gateway、Swagger和deepcopy真实diff，再写handler映射。

```go
note:=(*string)(nil)
if req.Note!=nil { value:=req.Note.Value; note=&value }
input:=tsmodel.CreateInput{Token:req.ConfirmationToken,RequestID:req.RequestId,Note:note}
```

protogen使用GOPATH的仓库路径：执行前确认脚本实际解析到本次隔离工作区，必要时为命令设置本任务专用GOPATH并建立对应symlink；不得让生成器写到另一个用户checkout。Swagger的全局int64处理不应改变普通string金额/ID；用真实JSON测试确认casing。

- [ ] **步骤4：组合现有两进程并注册完整生命周期。**AthenaServiceSet字段、newAthenaServiceSet构造、gRPC registration、gateway registration及Run/Close全部接入；只启动一个Collector/Projector/目录任务；每个Run由Service.Run使用errgroup与同一取消context管理，Close取消并等待退出，不能重复Run开启多个实例。账户store持有pool，TS和notification adapter借用；server关闭时先停后台再关pool，notification进程另有自己的pool。复用任务6已经在NewServer中完成的权限hook注入，不延迟到newAthenaServiceSet，也不覆盖已安装hook。通知进程注册account/system/reply/summary源，共用唯一dispatcher/poller，启动前确认旧sender停止，不能自动从过期lease判定安全接管。

配置显式`ATHENA_TRADER_SYNC_HTTP_URL`、`ATHENA_TRADER_SYNC_WSS_URL`、`ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY`及spec默认节奏，来源地址/version registry由核验值初始化；读取现有代理配置语义。HMAC key由部署配置提供，不每次重启随机生成导致全部cursor失效。端点缺失明确报告不可监控，不用未配置的dRPC自动兜底，不新增独立库/服务端口；沿用5秒发送专属timeout，Bot30秒poll不被全局5秒timeout截断。

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

- [ ] **步骤5：运行 `go test ./internal/server/... ./internal/tradersync/... ./pkg/apis/application/v1alpha1 ./pkg/apiclient/tradersync ./pkg/apiclient/account` 与新reads集成测试。**检查普通登录/API Key、禁用凭据、NONE/管理员不能读member、三个admin方法只概要、别人的cursor/token/id均拒绝。生成后编译两个cmd包；提交 `feat(trader-sync): expose authorized APIs and wire service lifecycle`。

## 任务13：指标、故障矩阵与首期容量验收

**Files**
- 新增：`internal/tradersync/metrics.go`、`internal/tradersync/metrics_test.go`、`internal/tradersync/acceptance/acceptance_integration_test.go`、`internal/tradersync/acceptance/fixtures_test.go`、`internal/tradersync/acceptance/capacity_test.go`。
- 修改：`internal/tradersync/projector.go`、`internal/tradersync/collector.go`、`internal/notification/dispatcher.go`、`internal/notification/store/attempts.go`，在现有日志/指标机制上增加观测，不新建监控基础设施。
- 新增验收报告：`docs/testing/trader-sync-activity-alerts-acceptance.md`；原始临时执行产物放`.superpowers/trader-sync-acceptance/`，长期报告只引用保留的可复现数据摘要。
**Interfaces**
- 产出：`TimingSample{PublicEarliest,PublicLatest *time.Time; ReceivedAt,RecordedAt time.Time; AuthorizedAt,StartedAt,AckAt *time.Time; Cohort,Outcome,ClockSource string; ClockUncertainty time.Duration}`；`EvaluateTiming(sample TimingSample) TimingResult`；`TimingResult{PublicAssessable bool; Lower,Upper,ReceivedToRecorded time.Duration; Reason string}`。
- 验收fixture定义在`fixtures_test.go`：`newHarness(t *testing.T,owners,targets int) *harness`、`(*harness).Close()`、`Push(log types.Log)`、`Advance(d time.Duration)`、`Stats() snapshot`。该目录三个测试文件全部加integration build tag，普通go test不会连接数据库。harness持有任务1真实隔离DB、loopback HTTP/WSS/Telegram、实际Service与Dispatcher、可控clock；snapshot含活动/HTTP调用/终态/预算/扫描调用计数。test helper只在测试包，业务不加入故障开关。

- [ ] **步骤1：写时间证据红灯测试。**

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

- [ ] **步骤2：实现可测量段与不可判定证据。**分别记录received→recorded、finality等待、metadata额外等待、gate等待、recorded→authorized/started/ACK、summary oldest→start、相邻批次start间隔。结果没有ACK也必须在pending/failed/unknown总数和年龄中出现，不能从报表消失。独立公开观测区间[p_min,p_max]且时钟可信时总延迟在[recorded-p_max,recorded-p_min]，不确定度扩展区间。

```go
result.ReceivedToRecorded=sample.RecordedAt.Sub(sample.ReceivedAt)
if sample.PublicEarliest==nil || sample.PublicLatest==nil || sample.ClockSource=="untrusted" {
    result.PublicAssessable=false
    result.Reason="public_time_or_clock_unverified"
    return result
}
```

不要标称区块时间就是公开查询时刻。SLO区间不能唯一判断阈值时报告不可判定；普通集中队列与外部故障单列并仍保留总体，其他owner的正常慢样本不豁免。只用有限维度标签，不将wallet/owner/note/payload作指标标签；详证据在受控日志/DB。

- [ ] **步骤3：实现100关系两种矩阵并执行全路径故障注入。**场景A为10owner×各10互异wallet，场景B为10owner共享10wallet；断言100关系、100/10唯一目标及每个owner独立活动/备注/消息。每个fixture驱动真实过滤→raw→确认→活动→sender路径，不直接插activity冒充采集验收。

```go
func TestCapacityDistinctTargets(t *testing.T) {
    h:=newHarness(t,10,100)
    defer h.Close()
    s:=h.Stats()
    if s.Relationships!=100 || s.UniqueTargets!=100 { t.Fatal(s) }
}
```

snapshot在本步骤定义字段`Relationships,UniqueTargets,Activities,HTTPCalls,HistoricalRangeCalls int`及每owner结果map；newHarness创建成功基线并校验，无HTTP发送前直接通过测试不算容量验证。继续推送三来源fixture，等待/推进clock直到预期活动和确定终态，断言所有owner结果及调用次数。

- [ ] **步骤4：执行完整故障矩阵并保存证据。**按本计划覆盖表逐项记录测试名/命令/结果：注册ACK竞争、同秒、raw失败、baseline提交未知、403/null、removed/重组/未知版本、暂停/撤权/重绑、permit提交未知、超时/ACK落库失败/缺起点、429/5次上限、summary冻结窗口/超长/混合结果/跨owner公平、sender旧实例停止确认和时钟跳变。真实PG事务测试用channel屏障，不用随机sleep碰运气。

RPC spy同时拒绝eth_sendTransaction/eth_sendRawTransaction，交易/签名接口使用调用即失败的fake；断言所有场景没有交易、钱包或仓位副作用。低频提示及暂停旧队列提示验证API字段和通知文案，尚未设计的业务页面单列为后续页面任务，不冒充本次后端通过的UI验收。
- [ ] **步骤5：做限定真实只读来源联调与Telegram联调。**来源只查询配置的开发RPC和公开Profile/PNL，重新核对实现版本/ABI/币种、实际100OR订阅与少量已知活跃目标；按HTTP方法、WSS帧、版本/metadata/重连分别累计成本。Telegram只发送到用户明确提供或当前任务已明确授权的测试接收者；若无此信息，先完成loopback全部验收，再请求该缺失信息，不能使用仓库发现的任意chat自动外发。全六区间不可用照实报告字段原因，不把所有unavailable算资料适配通过。
- [ ] **步骤6：运行 `go test -tags=integration ./internal/tradersync/acceptance -count=1`、相关race套件，并保存容量与时效报告。**明确区分可控100目标矩阵、真实小样本、真实100活跃目标压力和长期稳定性。真实公开时刻不明时公开→站内P95/P99标不可判定，不声明该项验收通过；没有真实100活跃样本时容量生产结论仍待验证。不得用预算算例代替计费全量或吞吐证据。提交 `test(trader-sync): verify failure boundaries and capacity scenarios`。

## 任务14：运行文档、相关回归和执行交付

**Files**
- 同步：`docs/requirements/polymarket-copy-trading/target-trade-monitoring-notifications.md`、`docs/design/trading/trader-sync-activity-alerts.md`、`docs/design/identity-access/account-access-control.md`、`docs/design/notifications/account-telegram-notifications.md`、`docs/design/notifications/system-notification-operations.md`、`docs/design/development-runtime/local-runtime-orchestration.md`、`docs/testing/trader-sync-activity-alerts-acceptance.md`。
- 按实际导航同步：`docs/design/README.md`、`docs/requirements/README.md`、`README.md`；仅调整受影响条目。实现规格保留批准记录，偏离必须写清原因和对应修订，不把技术限制悄悄改成需求放宽。
**Interfaces**
- 消费：任务1–13实现、生成记录和验收证据。
- 产出：可复现运行/恢复说明与真实完成状态；不新增公共接口。

- [ ] **步骤1：更新当前实现说明和运行命令。**去掉Notification独立DSN/数据库归属；说明同库两个pool、单实例启动/停止顺序、确认旧sender已停止后恢复、Chainstack/dRPC手动切换、新实时边界和未补遗漏。清理旧1.1秒全局串行发送、旧三态/泛化自动重试描述，给出unknown、永久资格失效、摘要首条缺失和资料unavailable的排查入口。保留Telegram绑定权限与系统通知原有功能。
- [ ] **步骤2：完成受影响链的最终回归。**复用各任务已完成且源码未再变化的生成记录，不无条件重跑生成器。当前HEAD执行以下相关套件；若某项在最后改动后已执行且证据完整可引用，不重复为计数运行。

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

- [ ] **步骤3：按Superpowers进行实现代码审阅并修复实际问题。**与批准spec和任务覆盖表核对；检查权限绕过、锁序、未知结果重发、摘要成员丢失、metadata阻塞成交和生成消费者错配。修复之后只重跑受影响验证；报告遗留的真实环境/性能证据缺口，未执行的必要验收不能写成通过或把整份计划标完成。
- [ ] **步骤4：提交最终文档和验证修正，给出分支/提交与验收证据。**局部提交不默认授权推送、合并或部署；按finishing-a-development-branch进行交付。完整实施与必要验收尚未完成时停在真实状态，不发完成邮件。
- [ ] **步骤5：仅在用户确认的整份计划已完整实施且必需验证完成后，从仓库根执行一次完成通知。**这是执行阶段的仓库规则，本轮只写计划不运行。

```bash
make notify-task-complete \
  TASK_NOTIFICATION_SUBJECT='任务完成：Trader Sync Activity Alerts' \
  TASK_NOTIFICATION_BODY='已完成：目标订阅、实时活动与Telegram通知；验证：相关单元、事务、接口和验收检查完成。'
```

等待命令完成；内置重试后仍失败则报告安全的错误概要，不声称邮件已发送。主题/正文按实际已完成范围修正，不能掩盖缺失验收。

## 规格覆盖与计划自检

| 批准spec / 长期需求 | 实现任务 | 主要验证 |
| --- | --- | --- |
| §1–3进程/同库/规模；规则1–6、14、30–33 | 1、6、12、13 | 单权威迁移、pool归属、十模块、owner/配额、100关系两矩阵 |
| §4接口/隐私/幂等；验收1–9、17、27–28 | 5、6、12 | 13 RPC、登录/API Key权限、管理员摘要、CAS、cursor/token owner绑定 |
| §5身份/确认卡/P/L/备注；规则40–42、验收9、33–34 | 5、6、8、12 | 精确URL、六区间独立证据、20/21 code point、nil/空、历史快照 |
| §6来源/基线/恢复；规则7–9、12–17、24–29、44；验收3、10–13、16、24–26、36–37 | 7、8、9、10 | 自身OrderFilled、三来源、最终确认、注册/ACK边界、无回补、重组及版本 |
| §7事务/持久实体；规则10–11、18–23、27 | 1–3、6、9–11 | 同库gate、候选固定归属、活动与外发原子、去重、无自动历史TTL |
| §8许可/结果；规则34–39、43；验收14–15、18–23、29、35 | 2–4、6、10、11 | 许可前后竞争、永久墓碑、unknown不重发、ACK落库恢复、Bot offset原子 |
| §9摘要/限速；验收20–23、29–31、35 | 4、10、11、13 | 滚动端点、前10逐条、冻结到started、完整分条、公平及双60秒边界 |
| §10–11配置/证据/容量；规则45、验收38–39 | 9、12、13、14 | 心跳/头/时钟、计费全量、真实与模拟分开、不确定不当通过 |
| §12长期文档及非目标；规则17、验收32 | 12、14 | 无交易/签名入口，无新页面布局，实际消费者与运行文档同步 |

计划作者自检（编写阶段，不代表实现测试）：

- [x] 逐节核对批准spec及长期设计覆盖表；所有规则组、39条验收范围均有后端落点或明确的后续页面边界。
- [x] 扫描占位语、未定义跨任务接口及文件路径；修正引用和生成顺序。
- [x] 检查任务消费/产出类型、account/session锁序、owner引用、授权与恢复语义一致。
- [x] 校验文档链接与diff；6份文档共133个本地链接有效，改动仅文档，未执行实现命令或外发通知。
