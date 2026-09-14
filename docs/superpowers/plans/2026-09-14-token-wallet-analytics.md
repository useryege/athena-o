# Token 钱包 Nansen 数据展示实施计划

> **执行代理：** 使用 `executing-plans` 按任务执行；用户选择并行实施时使用 `subagent-driven-development`。行为改动采用 TDD，完成前使用 `verification-before-completion`。复用已经取得的业务与页面确认，不重复请求未改变事项的批准。

**目标：** 在 Token 内上线钱包查询页，按已确认的 v1 清楚展示 Nansen 的钱包总览、逐币表现和交易明细。

**架构：** API Server 组合请求期间运行的 Nansen 适配模块，以自己持有的 account-state PostgreSQL pool 保存共享结果、请求协调与调用记录。公开 gRPC／HTTP 接口分区返回；React 页面消费这些接口，保留供应商数值、条件和来源。没有本地盈亏账本或独立链扫描依赖。

**技术栈：** 当前仓库 Go 1.27.1、pgx/v5、sqlc、gogofast／grpc-gateway；Node 24.14.1、React 18、TypeScript、现有请求服务、Jest／Testing Library、Playwright／axe。精确数值比较使用仓库已有的 `github.com/shopspring/decimal`，前端保留十进制字符串。

**规格：** [后端接入设计](../specs/2026-09-14-token-wallet-analytics-backend-design.md)、[有效需求](../../requirements/token/wallet-trading-performance.md)、[页面设计 v1](../../requirements/token/wallet-analytics-page-proposal.md)、[用户确认记录](../../requirements/token/previews/wallet-analytics-v1-approval.json)。执行时同时阅读这些文件与本计划。

**当前状态：** 2026-09-14 已完成实施计划整理，所有任务尚未执行。供应商样本调通和页面设计通过，不等于正式接口、缓存、权限或页面已交付。本轮只写计划与同步文档，不调用 Nansen、不启动服务、不执行生成器、不发送实施完成邮件。

## 全局约束

- 仅 Ethereum Mainnet；任意合法的完整 20 字节钱包地址，无须关联 Wallets。`period_days` 仅 7／30／90，缺省 30；不增加自定义日期。
- 总览只显示 Nansen 的已实现盈亏与交易过的代币数量；胜率、ROI、次数和 Top 5 不进入接口或页面。无本地买卖分类、基础币筛选、成本计算、报价换算或多跳还原。
- 代币以完整合约地址精确筛选，作用于两份列表，不改变总览。两类列表每批最多 20 项；逐币按已实现 USD 盈亏排序，默认降序，可反向；交易时间降序。
- 成功结果不足 10 分钟复用，满 10 分钟由下次请求刷新；不足 24 小时可作对应条件的失败回显，满 24 小时不可返回。成功完成时间决定边界，读取、失败和新建查询均不续期。
- 手动刷新绕过有效缓存；同次网络重发沿用 `request_id`，用户再次重试生成新值。没有用户请求时不刷新供应商数据。
- Key 只通过服务端 `NANSEN_API_KEY` 加载，平台用户共用该账户额度。前端没有 Key 配置能力；开发使用[已指定凭据](../../requirements/token/wallet-trading-performance.md#后续开发使用的-api-key已确认)。
- 所有新 RPC 要求当前 Token `READ`／`READ_WRITE`，包括缓存命中、查询创建和额度查询；长请求返回前再核对权限。普通会员、管理员及请求 realm 沿用现有边界。
- 不设每日查询次数上限。低于 1,000 credits 提示，恢复至 1,000 或以上解除；不据低额度阈值拦截仍可支付的请求，不自动充值、订阅或连续重试。
- 固定上游 `https://api.nansen.ai`；Key 仅在 `apikey` 请求头。共享主动限速 5 次／秒、最多 4 个在途 HTTP 请求，遵守更严格的供应商限流与 `Retry-After`。
- 单次业务请求预算最多 60 秒，单次 HTTP 默认 20 秒、响应最多 8 MiB；逐币两组合计最多 20 次 HTTP／64 MiB。租约固定 75 秒，网络等待期间不持有数据库事务。
- 采用单一 Nansen 深色视觉：近黑 `#06080B`、面板 `#0F1114`、青绿 `#00FFA7`；Inter 文字和数字，JetBrains Mono 地址和哈希。钱包页按 v1 实现，不将旧橙色或双主题当成新设计目标。
- 原有项目发现与研究、Wallets、Trader Sync 和管理员业务保持各自职责。钱包页所需导航与样式接线在本计划内，全站其他页面的主题重构不并入本计划。
- 只从 SQL／proto 源生成产物；不手改 sqlc、pb、gateway、Swagger 或 schema contract。按[同步技能](../../../.codex/skills/sync-athena-changes/SKILL.md)在稳定源批次后生成，再改消费者。
- 按[服务规范](../../developer-guide/service-development-standards.md)记录 SDS-R1–R8 的适用证据：库形态、显式授权、共享 pool 归属、独立配置与故障、取消和停止。不得发布仅后端而遗漏本计划所需消费者的成品 PR。

## 源码依据与开发顺序

| 当前源码事实 | 实施落点 |
| --- | --- |
| `internal/server/athena-server.go` 拥有 account-state pool、服务注册、gateway 和 Close | 组合适配器；先取消请求与关自有 HTTP transport，再由 API owner 关闭 pool。 |
| `internal/server/authz.go` 使用显式 `moduleGRPCRules` | 新增五个方法；从 `util/session.AccountID(ctx)` 取身份，不信任请求体中的账号。 |
| `internal/accountstate/schema/schema.go` 运行时只读 Verify，迁移由单独工具执行 | 加入 `000004_wallet_analytics.sql`，按迁移 → contract 生成 → 只读验证执行。执行前若编号被占用，使用紧邻的下一号并同步此计划，不能覆盖他人迁移。 |
| `sqlc.yaml` 的 account-state schema 已被多个 adapter 消费 | 新增 wallet analytics 查询输出包，并核对既有同库消费者。 |
| `hack/generate-proto.sh` 使用 gogofast，扫描 `internal/**/*.proto` | 使用可缺失 message 包装十进制值，不假定支持 proto3 `optional`；生成到独立 Go 包。 |
| `internal/devruntime/registry.go` 按服务筛选环境变量，生产 Compose 也显式列举变量 | Nansen 配置同时进入 API 本地白名单与 `docker-compose.prod.yml` 的 `athena-server.environment`。 |
| `ui/src/app/member/app.tsx` 的 Token 入口禁用且没有 landing path | 新增钱包子页、READ 路由与 landing path；不启用旧 Token 项目页面。 |
| 前端 `readNumber` 会转成 Number，现有请求服务负责 realm／账户／撤权清理 | 本页用专用字符串映射，复用请求 scope 与 abort，不绕过现有授权请求层。 |
| 现有 Playwright `testMatch` 只列 Trader Sync；smoke 只验证应用壳 | 明确新增钱包测试匹配与真实业务验收入口，不能只新增一个不会被执行的文件。 |

任务 1–7 完成并验证后端，任务 8–10 实现页面和整体验收。中间里程碑可独立审阅，但整个功能必须在消费者和真实验收完成后才标记交付。

## 文件职责

以下 `新增` 路径是目标，不表示当前文件或命令已经存在。

| 目标文件／目录 | 职责 |
| --- | --- |
| 新增 `internal/tokenwalletanalytics/model/model.go`、`decimal.go`、`query.go` | 无 IO 的结果、时间、地址、数值与分页模型；子包统一引用，避免循环依赖。 |
| 新增 `internal/tokenwalletanalytics/config.go`、`runtime.go`、`query_service.go`、`token_service.go`、`transaction_service.go`、`quota_service.go` | 配置、授权入口、请求协调及独立区域行为。 |
| 新增 `internal/tokenwalletanalytics/nansen/client.go`、`decode.go`、`errors.go`、`collector.go` | 固定端点、字面量解析、错误和分页收集。 |
| 新增 `internal/tokenwalletanalytics/store/store.go`、`snapshots.go`、`operations.go`、`provider.go`、`queries/*.sql`、生成 `sqlc/` | 共享 pool adapter；缓存、游标、租约、限速、额度与 attempt。 |
| 新增 `internal/tokenwalletanalytics/testdata/manifest.json`、`testkit/`、对应 `*_test.go` | 保存样本来源清单；可控时钟、HTTP transport 与 PostgreSQL 场景，测试工具不进入运行代码。 |
| 新增 `internal/server/tokenwalletanalytics/wallet_analytics.proto`、`server.go`、`mapping.go` | 五个公开 RPC、结果外壳和错误映射；不实现业务缓存。 |
| 生成 `pkg/apiclient/tokenwalletanalytics/wallet_analytics.pb.go`、`.pb.gw.go`，更新 `assets/swagger.json` | 公开客户端、gateway、Swagger。 |
| 新增 `internal/server/wallet_analytics.go`、`wallet_analytics_test.go`、`wallet_analytics_gateway_integration_test.go` | 组合配置、授权、生命周期和真实 gateway 回归。 |
| 修改 `internal/accountstate/store/migrations/`、`internal/accountstate/schema/contract.json`、`sqlc.yaml` | account-state 的显式迁移与验证契约。 |
| 修改 `internal/devruntime/registry.go`、`internal/devruntime/environment_test.go`、`docker-compose.prod.yml`、`.env` | API 配置注入；生产凭据仍由运行环境提供，不自动部署或改写 `.env.prod`。 |
| 新增 `ui/src/app/member/wallet-analytics-models.ts`、`wallet-analytics-service.ts` | HTTP 契约、专用精度映射、可取消请求；加入 `member/services.ts`。 |
| 新增 `ui/src/app/member/pages/wallet-analytics/` 下 `page.tsx`、`query-controller.ts`、`format.ts`、`summary.tsx`、`tokens.tsx`、`transactions.tsx`、`status.tsx`、`wallet-analytics.css` 及测试 | 页面、查询生命周期和展示组件。 |
| 修改 `ui/src/app/member/app.tsx`、`routes.tsx`；新增 `ui/src/assets/fonts/wallet-analytics/` | Token 导航、路由、账户切换清理和已确认字体资产。 |
| 新增 `ui/e2e/wallet-analytics-fixtures.ts`、`wallet-analytics.spec.ts`、`wallet-analytics-live.spec.ts`、`wallet-analytics-a11y.spec.ts`、`wallet-analytics-real.spec.ts` | 截获 HTTP 的展示场景、真实本地 API＋供应商替身、axe、真实 Nansen 查询，四类证据分别标注。 |
| 修改 `ui/playwright.config.ts`、`ui/scripts/acceptance-runner.mjs` 及其测试；新增 `internal/tokenwalletanalytics/acceptance/ui_harness_integration_test.go` | 选择钱包验收 harness、根路径与 `/athena`，保留原套件行为。 |
| 新增 `docs/testing/token-wallet-analytics.md` | 实际执行记录、来源、额度变化、局限、资源归属与停止证据。 |

## 跨任务契约

### 内部模型

任务 1 定义下列 Go 类型。可缺失数字用 `*DecimalValue`，nil 代表未知；`Value` 保留供应商数字原文，比较和展示时另行规范化，不改写原值。时间为 `time.Time`／`*time.Time`，公开边界转 RFC3339 UTC。代码示例中的 `model` 指 `internal/tokenwalletanalytics/model`。

```go
type DecimalValue struct { Value string }
type Range struct { From, To time.Time }
type Query struct {
    ID, Wallet string
    PeriodDays int
    ReferenceRange Range
    ForceRefresh bool
    ExpiresAt time.Time
}
type Issue struct { Code, Message, Field string; RetryAt *time.Time }
type Coverage struct {
    TokensComplete, TransactionsLastPage bool
    ProviderPages int
    Description string
}
type Meta struct {
    SnapshotID, Source, Wallet, Chain, State string
    PeriodDays int
    Range Range
    FetchedAt, FreshUntil, RetainUntil *time.Time
    Coverage Coverage
    Issues []Issue
}
type Summary struct { RealizedPnLUSD, TradedTokenCount *DecimalValue }
type Token struct {
    ID, Symbol, Address string
    BoughtUSD, SoldUSD, HoldingAmount, HoldingUSD *DecimalValue
    RealizedPnLUSD, UnrealizedPnLUSD *DecimalValue
    Sources []string
}
type Asset struct {
    Symbol, Address, Chain string
    Amount, PriceUSD, ValueUSD *DecimalValue
}
type Assets struct { Present bool; Items []Asset }
type Transaction struct {
    ID, Hash, RawTimestamp, TimestampBasis string
    Timestamp *time.Time
    VolumeUSD *DecimalValue
    Sent, Received Assets
}
type Result[T any] struct { Data *T; Meta Meta; NextCursor string }
type Quota struct {
    Remaining *DecimalValue
    ObservedAt *time.Time
    Low, Exhausted bool
    Issue *Issue
}
type RegionRequest struct { QueryID, RequestID string }
type ListRequest struct {
    QueryID, RequestID, TokenAddress, Cursor, SnapshotID, Order string
}
```

`SnapshotID` 用于逐币反向排序复用当前完整快照；它是只读引用，不是权限凭证。交易只接受游标继续原页，不接受客户端自报页号或日期。`Order` 仅 `desc`／`asc`，只在逐币接口生效。

### 公开契约

五个 RPC 使用 proto package `tokenapi`、Go package `github.com/useryege/athena/pkg/apiclient/tokenwalletanalytics`；不修改旧 `TokenCatalogService`。

| 方法 | 请求字段 | 返回字段 | HTTP |
| --- | --- | --- | --- |
| `CreateWalletAnalyticsQuery` | `wallet_address: string`、`period_days: int32`、`refresh: bool` | `query: WalletAnalyticsQuery` | `POST /api/v1/tokens/wallet-analytics/queries`，body `*` |
| `GetWalletAnalyticsSummary` | `query_id`、`request_id` | `result: WalletAnalyticsSummary`、`meta: WalletAnalyticsMeta` | `GET /api/v1/tokens/wallet-analytics/queries/{query_id}/summary` |
| `ListWalletTokenPerformances` | `query_id`、`request_id`、`token_address`、`order`、`cursor`、`snapshot_id` | `result: WalletTokenPerformances`、`meta`、`next_cursor` | `GET /api/v1/tokens/wallet-analytics/queries/{query_id}/tokens` |
| `ListWalletTransactions` | `query_id`、`request_id`、`token_address`、`cursor` | `result: WalletTransactions`、`meta`、`next_cursor` | `GET /api/v1/tokens/wallet-analytics/queries/{query_id}/transactions` |
| `GetWalletAnalyticsQuota` | `request_id` | `quota: WalletAnalyticsQuota` | `GET /api/v1/tokens/wallet-analytics/quota` |

除表中指定 int32／bool 外，请求字段均为 string。Query 对应内部模型，字段按 snake_case 命名；金额 message 为 `WalletAnalyticsDecimal { string value = 1; }`。Summary、Token、Transaction、Asset、Assets、Issue、Coverage、Meta、Quota 对应内部同名模型，message 前缀统一 `WalletAnalytics`。`WalletTokenPerformances`／`WalletTransactions` 各含 `repeated ... items = 1`，用外层 message 存在性区分成功空集合和无结果失败。字段号在首次 proto 中顺序分配并由生成契约测试固定。

数字 message 缺失与 `{value:"0"}` 区分；资产集合包含 `present`。缺失成功时间不生成空时间或当前时间。公开响应不包含原始 payload、Key 配置摘要、发起账号或其他用户信息。该新增接口固定使用 gateway 生成的 camelCase JSON 字段；前端不维护第二套旧 snake_case 返回兼容分支。

参数／身份错误使用 gRPC 标准错误；已授权的供应商失败返回区域 `meta.state=failed`，可回显时为 `stale`。所有五种成功或区域失败外壳均带 HTTP `Cache-Control: no-store, private` 和 `Vary: Cookie, Authorization, X-Athena-Application-Realm`；HTTP 错误路径也设置 no-store。

### 对外业务入口

任务 5 的 Runtime 对 transport 暴露以下方法，均从 ctx 解析并验证当前身份。`Runtime` 实现 `Close() error`，拥有本模块上下文与自有 HTTP transport，不拥有数据库 pool。

```go
CreateQuery(ctx context.Context, wallet string, days int, refresh bool) (model.Query, error)
GetSummary(ctx context.Context, req model.RegionRequest) (model.Result[model.Summary], error)
ListTokens(ctx context.Context, req model.ListRequest) (model.Result[[]model.Token], error)
ListTransactions(ctx context.Context, req model.ListRequest) (model.Result[[]model.Transaction], error)
GetQuota(ctx context.Context, requestID string) (model.Quota, error)
```

## 任务 1：可精确核对的数据模型与样本映射

**文件：** 新增 `model/{model,decimal,query}.go`、`config.go`、`nansen/decode.go` 及同名测试；新增 `testdata/manifest.json` 和 `testkit/fixtures.go`。

**接口：** `model.ReadDecimal(json.RawMessage) (*DecimalValue, error)`、`model.NewQuery(wallet string, days int, refresh bool, now time.Time) (Query, error)`；`nansen.DecodeSummary`／`DecodeTokens`／`DecodeTransactions` 接收 `[]byte`，返回对应数据、`[]model.Issue` 与结构性 error，列表还返回 `PageInfo{Page, PageSize int; Last bool}`。`LoadConfig(lookup func(string)(string,bool)) (Config,error)` 提供后端 spec §7 的六项配置。

- [ ] **1.1 写字段精度和地址／日期失败测试。** 用实际 JSON 字面量而非已被 float64 解析的数据；测试缺失、null、0、-0.0、科学计数及微量持仓。保存原文，数值有效性通过 decimal 验证。

```go
func TestDecimalKeepsSupplierLiteral(t *testing.T) {
    for _, raw := range []string{"0", "-0.0", "0.00000181579280252", "1.81579280252e-6"} {
        value, err := model.ReadDecimal(json.RawMessage(raw))
        require.NoError(t, err)
        require.Equal(t, raw, value.Value)
    }
    missing, err := model.ReadDecimal(json.RawMessage("null"))
    require.NoError(t, err)
    require.Nil(t, missing)
    _, err = model.ReadDecimal(json.RawMessage(`"not-a-number"`))
    require.Error(t, err)
}
```

- [ ] **1.2 运行红灯。** `go test ./internal/tokenwalletanalytics/model ./internal/tokenwalletanalytics/nansen -run 'TestDecimal|TestQuery|TestDecode' -count=1`；记录缺少实现或不满足断言，不能把测试编写错误当行为红灯。
- [ ] **1.3 实现纯模型、配置和映射。** JSON 外壳使用 `map[string]json.RawMessage` 或含 RawMessage 的 DTO；金额不经 float64。`model.NewQuery` 校验 `(?i)^0x[0-9a-f]{40}$`，小写用于规范化，零地址合法；固定 UTC 秒、闭区间与 UUID，缺省天数为 30。单字段异常写 Issue，集合或分页结构损坏返回 error。

```go
func ReadDecimal(raw json.RawMessage) (*DecimalValue, error) {
    literal := strings.TrimSpace(string(raw))
    if literal == "" || literal == "null" { return nil, nil }
    if _, err := decimal.NewFromString(literal); err != nil { return nil, err }
    return &DecimalValue{Value: literal}, nil
}
```

在此函数外先验证原始类型是 JSON number；不把对象、布尔或字符串数字静默接受为正常供应商数字。公开字符串只来自已经验证的数值字面量。限制异常大指数／位数的解析成本，按字段不可用处理并保留原文在服务端证据中。

- [ ] **1.4 建立只读样本清单。** `testkit.RecordedResponse(t testing.TB, name string) []byte` 按 manifest 读取研究文件的 `response` 原始字节；先核对 SHA256，再交给 decoder。至少收录 `nansen-integration-sample-130328/{pnl-summary,pnl,pnl-show-realized-false,transactions}.json` 与 `nansen-semantics-132128/{active-summary,active-pnl-true,active-pnl-false,active-transactions,transactions-cate-filter,transactions-usdt-filter}.json`；执行前以真实文件名核对清单，不复制 Key 或原研究脚本到运行包。
- [ ] **1.5 逐字段核对。** 总览只读 `realized_pnl_usd`、`traded_token_count`；逐币字段为 `token_symbol/token_address/bought_usd/sold_usd/holding_amount/holding_usd/pnl_usd_realised/pnl_usd_unrealised`；交易保留完整 sent／received 数组、数量符号、`volume_usd`、原时间和哈希。原生 ETH 标识保留；不解析类型或拆分一条供应商记录。
- [ ] **1.6 通过验证并提交本任务。** `go test ./internal/tokenwalletanalytics/model ./internal/tokenwalletanalytics/nansen -count=1`；检查所有命中字段对照原样本。提交范围仅本任务文件，提交说明 `feat: add wallet analytics models and provider mappings`。

## 任务 2：持久结果、游标与协调状态

**文件：** 新增迁移 `internal/accountstate/store/migrations/000004_wallet_analytics.sql`、`store/queries/{queries,snapshots,operations,provider}.sql` 与 `store/{store,snapshots,operations,provider}.go`、对应 integration tests；修改 `sqlc.yaml`，生成 `store/sqlc/` 和 schema contract。

**接口：** `store.New(pool *pgxpool.Pool) *Store`；Store 不关闭借用 pool。输出 `InsertQuery/GetQuery`、`FindSnapshot/LoadSnapshot/PublishSnapshot`、`CreateCursor/ResolveCursor`、`AcquireOperation/FinishOperation`、`AcquirePermit/ReleasePermit`、`BeginAttempt/FinishAttempt`、`ReadProviderState/ObserveProviderState`、`Prune`。所有方法首参为 context；业务主键、状态和期限的字段以本节表格固定，sqlc 类型只留在 store 包内。

| 表 | 必须保存的字段与约束 |
| --- | --- |
| `wallet_analytics_queries` | UUID PK、Key scope、wallet、days、from/to、force_refresh、created/expires；days IN (7,30,90)，expires=created+24h。 |
| `wallet_analytics_snapshots` | UUID PK、semantic_key、scope、region、mapping_version、实际请求 JSON、映射结果 JSON、供应商原始响应 bytea、fetched/fresh/retain；只插入不可变成功结果。 |
| `wallet_analytics_lookups` | kind=`latest`／`cursor`、key PK、snapshot_id、不可变 cursor binding JSON、expires；latest 可更新指针，cursor 引用不得换页；过期索引。 |
| `wallet_analytics_operations` | kind=`build`／`build_result`／`delivery`／`permit`、key、owner UUID、generation bigint、lease_until、status、result_snapshot_id、issue JSON、expires；(kind,key) 主键；generation 单调。build_result 按 build key＋generation 保存不可变完成结果，避免下一代覆盖等待者所需结果。 |
| `wallet_analytics_provider_state` | scope PK、余额及存在性／观测时间、冷却时间、滑动限速窗口；不存 Key 原值。 |
| `wallet_analytics_call_attempts` | UUID PK、operation、actor、固定 endpoint、started/finished、upstream request ID、HTTP status、issue、报价／实扣／余额的值及存在性；持续保留，外键不能因缓存清理级联删除。 |

- [ ] **2.1 写 PostgreSQL 红灯。** 使用已有 `internal/testutil/pgtest` 的随机独立测试数据库；迁移后验证六张表。覆盖不可变快照、十进制字符串 round-trip、24h 截止、游标跨条件拒绝及清理不删除 attempts。

```go
func TestWalletTablesExistAfterMigration(t *testing.T) {
    db := pgtest.New(t, migrations.FS, migrations.Dir)
    for _, name := range []string{"queries", "snapshots", "lookups", "operations", "provider_state", "call_attempts"} {
        var exists bool
        err := db.Pool.QueryRow(context.Background(),
            `SELECT to_regclass($1) IS NOT NULL`, "public.wallet_analytics_"+name).Scan(&exists)
        require.NoError(t, err)
        require.True(t, exists, name)
    }
}
```

- [ ] **2.2 运行红灯并写 SQL 源。** `go test -tags=integration ./internal/tokenwalletanalytics/store -run TestWallet -count=1`。按下列模式实现租约取得和发布条件；`PublishSnapshot` 的插入快照、更新 latest、完成 operation 在一个短事务内，全程验证 owner／generation／未过期；事务锁顺序固定 operation → lookup。

```sql
-- name: AcquireWalletAnalyticsOperation :one
INSERT INTO wallet_analytics_operations
  (kind, key, owner, generation, lease_until, status, expires)
VALUES ('build', $1, $2, 1, $3, 'running', $4)
ON CONFLICT (kind, key) DO UPDATE
SET owner = EXCLUDED.owner,
    generation = wallet_analytics_operations.generation + 1,
    lease_until = EXCLUDED.lease_until, status = 'running'
WHERE wallet_analytics_operations.status <> 'running'
   OR wallet_analytics_operations.lease_until <= $5
RETURNING owner, generation, lease_until;
```

同次 delivery 的复用由上层先读取状态决定，不重做已经完成的同次投递。build 完成后，新的一次显式刷新可取得新 generation，不必等旧 75 秒租期；已等待的请求固定读取原 generation 的 build_result。发布时 `UPDATE ... WHERE owner=$owner AND generation=$generation AND lease_until>$now AND status='running'` 必须恰好影响一行，否则回滚快照写入；同一事务保存 build_result。租约失效的旧 owner 无发布权。失败完成也保存该代次结果并释放 running 状态，不自动重发。

- [ ] **2.3 固定生成输入并生成一次。** 在 `sqlc.yaml` 新增 schema 指向 account-state migrations、queries 指向新 store/queries、输出到新 store/sqlc 的条目；执行 `make sqlc-local` 并审阅 diff。该目标实际通过 `go run -mod=mod github.com/sqlc-dev/sqlc/cmd/sqlc generate` 运行，无须 `dist/sqlc`；执行阶段确认 Go 依赖可用，不要绕过生成手写 sqlc。
- [ ] **2.4 生成并检查 schema contract。** 准备开发 PostgreSQL 后配置 `ATHENA_TEST_PG_ADMIN_DSN`，运行 `make account-state-schema-contract`；该工具创建并清理自己拥有的临时库，不迁移用户现有库。补充只读 Verify 拒绝缺表、缺索引、旧版本的回归。
- [ ] **2.5 完成 store adapter 与时间边界测试。** 清理入口分批删除过期 cursor／lookup／query／operation／snapshot，保留仍被有效 cursor 引用的快照至其自身 retain 截止；所有读取均独立检查 expires，物理未删除也不能使用。
- [ ] **2.6 通过同库检查并提交。** `go test -tags=integration ./internal/tokenwalletanalytics/store ./internal/accountstate/schema ./internal/accountstate/store ./internal/notification/store -count=1`，保存命令及退出结果。提交说明 `feat: persist wallet analytics snapshots and coordination`。

## 任务 3：固定供应商客户端、共享限速与实际调用记录

**文件：** 新增 `nansen/{client,errors}.go`、`provider_executor.go`、对应测试；补齐任务 2 的 provider／attempt SQL 与 adapter。SQL 若发生变化，完成这一批再运行一次 sqlc。

**接口：** `nansen.NewClient(apiKey string, timeout time.Duration, transport http.RoundTripper) *Client`；`Client.Request(ctx context.Context, endpoint Endpoint, body []byte) (Response,error)`。Endpoint 只有 `SummaryEndpoint`、`PnLEndpoint`、`TransactionsEndpoint`、`AccountEndpoint`；Response 包含 `Body []byte`、`Headers http.Header`、`Status int`、`StartedAt/FinishedAt time.Time`。根包 `Executor.Execute(ctx, actorID, operationID, endpoint, body)` 调用 Store 协调，再调用该客户端；返回同一 Response／error。actorID、operationID 为 string，其余与 Client 相同。

- [ ] **3.1 写不会访问互联网的 HTTP 红灯。** 测试 transport 注入只在 Go 构造参数中存在；URL 与 `apikey` 由客户端固定生成。Account 使用 GET，另外三个端点使用 POST。

```go
type roundTripFunc func(*http.Request) (*http.Response, error)
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClientUsesFixedOriginAndKeepsResponseBytes(t *testing.T) {
    raw := `{"realized_pnl_usd":0.00000181579280252,"traded_token_count":0}`
    transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
        require.Equal(t, "https://api.nansen.ai/api/v1/profiler/address/pnl-summary", r.URL.String())
        require.Equal(t, "test-key", r.Header.Get("apikey"))
        require.Equal(t, http.MethodPost, r.Method)
        return &http.Response{StatusCode: 200, Header: make(http.Header),
            Body: io.NopCloser(strings.NewReader(raw))}, nil
    })
    client := nansen.NewClient("test-key", 20*time.Second, transport)
    response, err := client.Request(context.Background(), nansen.SummaryEndpoint, []byte(`{}`))
    require.NoError(t, err)
    require.Equal(t, raw, string(response.Body))
}
```

- [ ] **3.2 运行红灯并实现客户端。** `go test ./internal/tokenwalletanalytics/nansen -run TestClient -count=1`。关闭重定向或拒绝所有跨域重定向，避免 Key 被转送；`io.LimitReader` 读取 8 MiB＋1 后检查超界；所有响应关闭 body；不自动重试。请求 context 与 HTTP timeout 同时限制读取。
- [ ] **3.3 实现发送前 attempt 与共享许可。** 在短事务内锁定 scope 对应 provider_state，移除已过期许可、检查冷却／滑动 1 秒窗口／4 个有效 permit；获准后写 permit、窗口记录和 attempt，再提交事务并发送 HTTP。许可包含独立 UUID 和失效时间；完成、取消均释放，进程崩溃由期限回收，不永久占槽。

```text
Execute
  检查 ctx 剩余预算 → AcquirePermit
  许可不足：等待给出的下一时刻或 ctx 取消，不发送 HTTP
  BeginAttempt 失败：释放许可，返回 LOCAL_STORAGE_UNAVAILABLE
  Client.Request 一次 → FinishAttempt（真实头部/状态/未知结算）
  ObserveProviderState → ReleasePermit
  持久化失败：保留已发送事实，不重新发送上游请求
```

许可等待与账户核对均计入区域总 60 秒预算；当前请求收到 429 后结束这一尝试，不能在同一次请求里睡眠后再发。5 次／秒和 4 并发通过两个独立 pool 的 integration test 验证，不能用单进程 mutex 代替共享约束。

- [ ] **3.4 固定错误映射和额度存在性。** 401→`PROVIDER_AUTH_FAILED`；套餐／credits 错误只按响应体明确错误码区分，不能将所有 403 都当额度耗尽；429→`PROVIDER_RATE_LIMITED`，解析秒数或 HTTP date 的 Retry-After；超时→TIMEOUT；5xx／连接故障→UNAVAILABLE；结构不合法→DATA_INVALID。完整错误码以 spec §6 为准。额度、报价、实扣缺失分别为未知，不写 0，不用余额差代替实际扣款。

保存样本的对应头为 `x-nansen-credits-cost`（报价）、`x-nansen-credits-used`（实扣）、`x-nansen-credits-remaining`（余额）、`x-request-id`（供应商请求 ID）。响应可能同时出现多层限流头，按同窗口最严格的有效限制处理，不能用更宽松的一层放大主动上限；缺失／无法解释的头不解除已存在的冷却。
- [ ] **3.5 加入失败路径与限流集成测试。** 覆盖 body 超限、取消、401、429、耗尽、账户失败、attempt 写入前失败不得出网、结果写入后失败不得重发、6 个并发请求等待、75 秒许可／租约回收。使用同步 channel 与可控时钟确定顺序，不用长 sleep 猜时序。
- [ ] **3.6 验证并提交。** `go test ./internal/tokenwalletanalytics/nansen -count=1`；`go test -race -tags=integration ./internal/tokenwalletanalytics -run 'TestExecutor|TestRate|TestAttempt' -count=1`。提交说明 `feat: add metered Nansen request executor`。

## 任务 4：逐币完整集合与交易分页

**文件：** 新增 `nansen/collector.go`、`model/sort.go`、`token_service.go`、`transaction_service.go`、相关测试；按需求补齐 store cursor 查询。

**接口：** `model.SortTokens(rows []Token, order string) ([]Token,error)`；`nansen.CollectTokens(ctx context.Context, fetch TokenPageFetcher, request PnLRequest, limits CollectionLimits) (TokenCollection,error)`。`TokenPageFetcher` 为 `func(context.Context, PnLRequest) (Response,error)`；`PnLRequest` 含 `Wallet/TokenAddress string`、`Range model.Range`、`ShowRealized bool`、`Page/PageSize int`；`CollectionLimits` 含 `MaxRequests int`、`MaxBytes int64`；`TokenCollection` 含 `Rows []model.Token`、`Responses []Response`、`ProviderPages int`。fetch 由 Executor 包装提供，不绕过限速／计费。

- [ ] **4.1 写全局排序红灯。** 构造 41 行测试集合，将最高盈利放在第三个供应商页；另外加入相同金额、空值、负数及大整数旁的小数差。该集合明确标为受控测试数据，不能作为真实钱包样本。

```go
func TestSortTokensUsesExactNumbersAndKeepsMissingLast(t *testing.T) {
    rows := []model.Token{
        {Address: "b", RealizedPnLUSD: &model.DecimalValue{Value: "9007199254740992.01"}},
        {Address: "a", RealizedPnLUSD: &model.DecimalValue{Value: "9007199254740992.02"}},
        {Address: "unknown"},
    }
    got, err := model.SortTokens(rows, "desc")
    require.NoError(t, err)
    require.Equal(t, []string{"a", "b", "unknown"}, []string{got[0].Address, got[1].Address, got[2].Address})
    got, err = model.SortTokens(rows, "asc")
    require.NoError(t, err)
    require.Equal(t, "unknown", got[2].Address)
}
```

- [ ] **4.2 运行红灯并实现两组收集。** `go test ./internal/tokenwalletanalytics/model ./internal/tokenwalletanalytics/nansen -run 'TestSortTokens|TestCollect' -count=1`。true 组全部分页后读取 false 组；每组显式已实现盈亏降序，上游页大小默认 1,000。每次成功接收完整响应记录顺序，按链＋规范化资产地址选最后完整一行；未知资产地址用来源页＋行号保留独立行，不能按符号去重。

```text
两组完成 = true.is_last_page && false.is_last_page
完整前：不发布 snapshot，不输出部分排名
完整后：精确金额排序 → 相同金额按资产地址 → 未知地址按稳定行 ID
反向排序：只翻转已知金额的方向，缺失金额仍置后
展示分页：固定快照的切片，每批 20 行
```

- [ ] **4.3 验证覆盖异常。** 检查页号、页大小与 is_last_page 的存在性；非末空页、重复整页、同组重复推进、任组失败、超过 20 次／64 MiB／60 秒都停止，不发布部分成功。单响应上限仍由客户端执行。少于页大小不能替代末页信号。保存响应足以核对最后完整行的来源，不跨响应相加。
- [ ] **4.4 实现交易页固定绑定。** 请求固定 `block_timestamp DESC`、`hide_spam_token=true`、每页 20；过滤用上游 `token_address`。cursor 保存 wallet／days／实际 from/to／token／page／order／首页 lineage；两次相同哈希保留两条记录，行 ID 由 snapshot/page/index 构成。
- [ ] **4.5 写交易边界测试。** 使用已保存第一页、`transactions-page2.json`、CATE／USDT filter 响应；覆盖同时间两条记录、未知时间保留但附字段问题、非末空页、页号不符、跨页已知时间反序、客户端篡改条件、游标到期。跨页比较只比较可解析的已知时间，缺失时间不自行补成查询时间。失败不产生“已加载全部”。
- [ ] **4.6 固定 snapshot 引用。** 反向排序携带当前逐币 snapshot_id，回到第 1 批；该快照存在、范围与筛选匹配且未满 24 小时时读取完整集合，不请求 Nansen；缓存过了 10 分钟的主动新查询仍按刷新规则处理。交易后续页严格继承首页范围，不借用新查询参考时间。新 cursor 不延长旧快照自身留存期。
- [ ] **4.7 验证并提交。** `go test ./internal/tokenwalletanalytics/model ./internal/tokenwalletanalytics/nansen -count=1`；`go test -tags=integration ./internal/tokenwalletanalytics -run 'TestToken|TestTransaction|TestCursor' -count=1`。提交说明 `feat: organize complete token results and stable transaction pages`。

## 任务 5：分区缓存、请求合并、权限与额度恢复

**文件：** 新增 `runtime.go`、`query_service.go`、`quota_service.go`、`testkit/rig.go`；补齐 token／transaction 服务和操作存储；新增缓存、并发、权限、quota integration tests。

**接口：** `NewRuntime(opts Options) (*Runtime,error)`；Options 为 `Pool *pgxpool.Pool`、`Config Config`、`Transport http.RoundTripper`、`Now func() time.Time`、`Authorize func(context.Context)(string,error)`。入口与返回类型使用前述跨任务契约。`Authorize` 返回当前可信账号且检查 Token READ，不能由请求提供账号。`testkit.NewRig(t *testing.T, now time.Time)` 创建迁移后的独立库、可控时钟、脚本 HTTP 与 Runtime，返回 `Rig{Runtime, Pool, Clock, Provider}`；Clock 提供 `Set(time.Time)`，Provider 提供 `Count() int`、`FailNext(code string)`、`BlockNext() (started <-chan struct{}, release func())`。Rig 关闭本测试创建的资源。

- [ ] **5.1 写缓存四个边界的红灯。** 同时断言调用数、真实 range、fetched_at 和失败外壳；以下测试扩展为表驱动，并以固定假时钟运行。

```go
func TestCachedReadDoesNotRenewSuccessfulTime(t *testing.T) {
    start := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
    rig := testkit.NewRig(t, start)
    q, err := rig.Runtime.CreateQuery(context.Background(), "0xfb67ef6fe609edab1e0595e6815634e8e4db9cf7", 30, false)
    require.NoError(t, err)
    first, err := rig.Runtime.GetSummary(context.Background(), model.RegionRequest{QueryID:q.ID, RequestID:uuid.NewString()})
    require.NoError(t, err)
    calls := rig.Provider.Count()
    rig.Clock.Set(start.Add(9*time.Minute + 59*time.Second))
    cached, err := rig.Runtime.GetSummary(context.Background(), model.RegionRequest{QueryID:q.ID, RequestID:uuid.NewString()})
    require.NoError(t, err)
    require.Equal(t, calls, rig.Provider.Count())
    require.Equal(t, first.Meta.FetchedAt, cached.Meta.FetchedAt)
    require.Equal(t, first.Meta.Range, cached.Meta.Range)
    require.Equal(t, "cached", cached.Meta.State)
}
```

- [ ] **5.2 运行红灯，实现清楚分离的查找键。** `go test -tags=integration ./internal/tokenwalletanalytics -run 'TestCached|TestStale|TestRefresh' -count=1`。semantic key 包含 Key 的服务端 SHA256 scope、wallet、days、region、token、mapping version；snapshot 保存真正的日期和参数。cursor／排序不是新的供应商语义查询，不能把不同日期在途请求合并。

```text
Authorize → 校验 query/参数/期限
  先查同次 delivery：有完成结果则复用，返回前再次 Authorize
  普通请求且命中 <10m 成功快照：cached
  否则固定实际参数 → 获得 build lease 或等待同一 build
  成功：原子发布 snapshot + latest + delivery，fresh
  失败：同 semantic 且 <24h 的旧成功 snapshot → stale + issue
        无可用结果 → failed + issue（Data=nil）
  返回前再次 Authorize
```

- [ ] **5.3 实现幂等与跨实例合并。** delivery key 绑定 query_id、region、筛选／cursor、排序／snapshot 引用及 request_id，记录本次响应结果，保留不超过 query／snapshot 的实际期限。同 request_id 改参数为 InvalidArgument。build key 用真正请求参数；只有完全等价才合并。owner 取消或停机结束在途获取；等待者不接手后台采集，等待者取消也不替 owner 再发。首次调用账号记一次 attempt，其余等待用户不产生虚假消耗。
- [ ] **5.4 验证租约和撤权。** 两个 pool／Runtime 竞争相同 key，使用同步屏障确认只有一个 owner；推进假时钟超过 lease，新 owner 成功后旧 owner 发布必须失败。等待期间撤销 Token 权限，数据已经产生也不能返回该用户。缓存、quota、query 和 cursor 入口都验证；管理员身份不自动授予会员 Token 权限。
- [ ] **5.5 完成时间与错误测试矩阵。** 9:59 为 cached；10:00 发新请求；23:59:59 刷新失败可 stale；24:00 不回显。满 24 小时 query 也过期，测试必须创建新的同条件 query 来验证旧 snapshot 边界，不能把 query 过期与 cache 过期混为一个断言。新钱包／日期／筛选失败不能显示另一条件的旧数据；局部失败不改变其余区域。
- [ ] **5.6 完成 quota 按需获取。** 新响应中存在的余额优先形成带时间观测，缺失头不清零；快照未知／满 10 分钟时查询 account，同 scope 合并。已明确耗尽后，用户显式刷新／重试使用新 request_id，可核对一次账户来发现充值；同次重发复用核对，不形成自动循环。确认充值后解除拦截；account 核对失败不从旧的 0 推断永久耗尽。缓存读取不以 quota 成功为前提。
- [ ] **5.7 实现生命周期。** 每个入口带 runtime 根取消和请求 deadline；并发等待、HTTP 读取、限速等待都可取消。housekeeping 仅清本模块过期数据，不取供应商信息。Close 取消、等待有限退出、停止清理 timer、关闭自有 HTTP idle connections；不关闭注入的借用 transport／pool。config 缺失或非法映射为模块不可用，实例仍允许其他 API 正常运行。
- [ ] **5.8 验证并提交。** `go test -race -tags=integration ./internal/tokenwalletanalytics/... -count=1`；确保没有跳过需要 PostgreSQL 的测试。提交说明 `feat: coordinate authorized wallet queries and shared cache`。

## 任务 6：公开接口、API 组合与实际环境注入

**文件：** 新增 `internal/server/tokenwalletanalytics/{wallet_analytics.proto,server.go,mapping.go}` 及测试、`internal/server/wallet_analytics.go` 和测试；修改 `athena-server.go`、`authz.go`／`authz_test.go`、运行白名单、Compose 及其测试；生成公开 Go、gateway、Swagger。

**接口：** transport 的 `NewServer(runtime *tokenwalletanalytics.Runtime) *Server` 实现五个 RPC；内部 `summaryResponse`、`tokensResponse`、`transactionsResponse` 将 model.Result 映射到前述对应公开 Response。`AthenaServer` 新增 `walletAnalytics *Runtime` 与 `serviceSet.WalletAnalytics`。独立组合函数 `newWalletAnalyticsRuntime(pool, authorize, lookup)` 负责配置加载，参数分别为 `*pgxpool.Pool`、`func(context.Context)(string,error)`、`func(string)(string,bool)`；返回可用或带配置 issue 的 Runtime，不使无关接口启动失败。

- [ ] **6.1 先写传输契约红灯。** 验证实际 gateway 输出数字为字符串，未知不变 0；authz 使用真实 session／账户 API Key 的账号、Token READ／READ_WRITE 通过，无权限、撤权、无登录拒绝。Create 是建立查询，即使名为 Create 也只要求 READ。

```go
func TestSummaryJSONKeepsMissingCountDistinctFromZeroPnL(t *testing.T) {
    response := &api.GetWalletAnalyticsSummaryResponse{
        Result: &api.WalletAnalyticsSummary{
            RealizedPnlUsd: &api.WalletAnalyticsDecimal{Value: "0"},
        },
    }
    body, err := (&jsonpb.Marshaler{}).MarshalToString(response)
    require.NoError(t, err)
    require.Contains(t, body, `"realizedPnlUsd":{"value":"0"}`)
    require.NotContains(t, body, `"tradedTokenCount"`)
}
```

- [ ] **6.2 写 proto 源。** message 严格对应跨任务表；Query／Meta 的内部 Range 展开为 `range_from/range_to`，Wallet 对应 `wallet_address`，所有公开时间字段为 RFC3339 UTC string。缺失时间保持未设置。结果字段不可用值使用缺失 decimal message；不使用 string 的空串代表已知 0。

```proto
syntax = "proto3";
package tokenapi;
import "google/api/annotations.proto";
option go_package = "github.com/useryege/athena/pkg/apiclient/tokenwalletanalytics";

service TokenWalletAnalyticsService {
  rpc CreateWalletAnalyticsQuery(CreateWalletAnalyticsQueryRequest) returns (CreateWalletAnalyticsQueryResponse) {
    option (google.api.http) = {post: "/api/v1/tokens/wallet-analytics/queries" body: "*"};
  }
  rpc GetWalletAnalyticsSummary(GetWalletAnalyticsSummaryRequest) returns (GetWalletAnalyticsSummaryResponse) {
    option (google.api.http) = {get: "/api/v1/tokens/wallet-analytics/queries/{query_id}/summary"};
  }
  rpc ListWalletTokenPerformances(ListWalletTokenPerformancesRequest) returns (ListWalletTokenPerformancesResponse) {
    option (google.api.http) = {get: "/api/v1/tokens/wallet-analytics/queries/{query_id}/tokens"};
  }
  rpc ListWalletTransactions(ListWalletTransactionsRequest) returns (ListWalletTransactionsResponse) {
    option (google.api.http) = {get: "/api/v1/tokens/wallet-analytics/queries/{query_id}/transactions"};
  }
  rpc GetWalletAnalyticsQuota(GetWalletAnalyticsQuotaRequest) returns (GetWalletAnalyticsQuotaResponse) {
    option (google.api.http) = {get: "/api/v1/tokens/wallet-analytics/quota"};
  }
}
message WalletAnalyticsDecimal { string value = 1; }
```

所有 request/response message 使用跨任务表中同名定义，与模型字段一一对应，不导入旧 Token 项目 API 模型。代码块展示服务源；消息的字段表是本计划的完整契约输入。

- [ ] **6.3 稳定源后生成。** 执行 `make protogen`；核对新 package 的 pb／gateway、Swagger 五条路由和无无关生成变更。随后实现 handlers，不在生成输出上补字段。测试真实 grpc-gateway 的 camelCase JSON，再由任务 8 对照生成契约。
- [ ] **6.4 组合服务并登记授权。** `newWalletAnalyticsRuntime` 通过可信 session AccountID 和 `accessController.Authorize(id, accountaccess.RequireModule(accountaccess.ModuleToken, accountaccess.AccessLevelRead))` 授权。把五个完整方法名登记 `moduleRead(accountaccess.ModuleToken)`，注册 gRPC 与 gateway，纳入 `serviceSet`。不能因为缓存共用而绕过授权。
- [ ] **6.5 实现 no-store 与停止。** 在已有 `translateGRPCResponseHeaders` 添加五类响应；钱包 HTTP 路径的统一前置 middleware 同样设置 no-store，覆盖 401／403／参数错误。`AthenaServer.Close()` 在 accountStateStore.Close 前调用 Runtime.Close；构造中途失败也释放自有 HTTP 资源。支持根路径和部署前缀，避免在另一个 handler 写死根 URL。
- [ ] **6.6 注入配置。** 把 Key 和五个工程项加入 API 的 `apiConsumerEnvironment`；确认全栈实际选用同一环境筛选链。Compose 的 `athena-server.environment` 加入下列映射。`.env` 中装载用户已指定 Key；不修改其他进程的 Key 白名单，不添加 UI 变量或管理接口。

```yaml
NANSEN_API_KEY: ${NANSEN_API_KEY:-}
ATHENA_NANSEN_HTTP_TIMEOUT: ${ATHENA_NANSEN_HTTP_TIMEOUT:-20s}
ATHENA_NANSEN_QUERY_TIMEOUT: ${ATHENA_NANSEN_QUERY_TIMEOUT:-60s}
ATHENA_NANSEN_PNL_PAGE_SIZE: ${ATHENA_NANSEN_PNL_PAGE_SIZE:-1000}
ATHENA_NANSEN_PNL_MAX_REQUESTS: ${ATHENA_NANSEN_PNL_MAX_REQUESTS:-20}
ATHENA_NANSEN_PNL_MAX_RESPONSE_BYTES: ${ATHENA_NANSEN_PNL_MAX_RESPONSE_BYTES:-67108864}
```

- [ ] **6.7 增加配置／组合失败测试。** Key 缺失、HTTP 超时大于总超时、总超时大于 60 秒、非法页大小均仅使模块不可用；合法未配置场景的 bootstrap 仍工作。测试 API 进程能取得配置，UI／notification／trader-sync 取不到 Key；不启动 Token API／链扫描也能通过新公开服务完成查询。
- [ ] **6.8 验证并提交。** `go test ./internal/server/tokenwalletanalytics ./internal/server ./internal/devruntime -count=1`；`go test -tags=integration ./internal/server -run 'TestWalletAnalytics' -count=1`；`go build ./cmd/athena-server`；`bash hack/production-compose_test.sh`。提交说明 `feat: expose authorized wallet analytics API`。环境白名单／Compose 测试不能替代实际启动验证。

## 任务 7：后端阶段验收与最小真实 Nansen 核对

**文件：** 新增 `internal/tokenwalletanalytics/live_integration_test.go`、`internal/server/wallet_analytics_gateway_integration_test.go`；新增实际验证文档 `docs/testing/token-wallet-analytics.md`。测试专用脚本和原始证据写入 `.tmp/`，持久文档只记录必要来源和结果。

**接口：** `TestNansenLiveRoundTrip` 只在 `ATHENA_NANSEN_LIVE=1` 时允许真实供应商请求，读取服务端 `NANSEN_API_KEY`。该开关只控制测试，不是产品功能开关。测试模式明确 live；缺 Key 时失败而不是宣布验证通过。

- [ ] **7.1 先执行离线／数据库验收。** 任务 1–6 的失败、精度、排序和权限用保存样本／本地替身验证；冻结红绿记录、生成检查点和测试退出结果，避免使用付费调用调试确定性错误。
- [ ] **7.2 准备目标环境与显式迁移。** 按运行文档选择 Node、确认 repository／worktree／实例所有权；复用健康环境，缺失时在该工作区运行 `make run` 并保存会话和日志。先执行 account-state 显式迁移，再启动只读 Verify 的 API；不通过 API 启动建表，不 reset 数据库。

```bash
go run ./cmd/athena-account-state-migrate up
go run ./cmd/athena-account-state-migrate verify
ATHENA_NANSEN_LIVE=1 go test -tags=integration ./internal/tokenwalletanalytics -run '^TestNansenLiveRoundTrip$' -count=1 -v
```

记录实际使用的服务端 DSN 与 Key 配置归属，不打印凭据到验收报告；这里的迁移、真实调用命令仅在执行阶段运行。

- [ ] **7.3 做一轮最小真实核对。** 使用已验证钱包 B `0xfb67ef6fe609edab1e0595e6815634e8e4db9cf7`、30 天：一次摘要、两组逐币第一页、一次交易；只在 is_last_page 为 false 时继续必要逐币页。请求前后取额度观测，实际消费按响应记录，约 4 credits 仅为单页样本估计，不能预填为结算结果。测试与 API 调用共用本轮证据，不重复请求同一目的数据。
- [ ] **7.4 把返回与原响应逐字段对照。** 当前响应和本次映射结果比较，不与 9 月 13 日的历史金额比较相等；记录 UTC 实际范围、获取时间、两组页数／字节／耗时、来源行及数字字符串。再次普通查询应命中缓存；另一个有权限账号读取相同语义缓存不增加数据端点调用。手动刷新恰好形成新操作，网络重发不形成另一组数据请求。
- [ ] **7.5 验证真实分页能力边界。** 用供应商真实响应验证页大小 1,000 被接受；存在下一页时核对真实翻页。41 行、异常排序、重复页和超界通过受控数据证明算法与防护，不把它们写成真实大钱包吞吐证据。若找不到足够大且可完成的真实集合，如实记录“真实大集合耗时未覆盖”，不伪造覆盖率或额外研究成本算法。
- [ ] **7.6 完成后端里程碑。** 保存公开 HTTP／gRPC 请求、返回与 attempts 对照，证明 Key 缺失不影响 bootstrap、READ 可以查询、撤权不能读缓存、扫描服务未启动。后端全部必要用例通过后进入页面编码；仍未完成的外部依赖写入具体失败证据，不把隔离测试当整个功能交付。提交说明 `test: verify wallet analytics backend integration`。

## 任务 8：页面服务、精度格式化与请求生命周期

**文件：** 新增 `ui/src/app/member/wallet-analytics-models.ts`、`wallet-analytics-service.ts`、`pages/wallet-analytics/{format,query-controller}.ts` 和测试；修改 `member/services.ts`。

**接口：** `WalletAnalyticsService` 提供 `createQuery(wallet, days, refresh)`、`getSummary(queryId, requestId)`、`listTokens(queryId, requestId, {tokenAddress, order, cursor, snapshotId})`、`listTransactions(queryId, requestId, {tokenAddress, cursor})`、`getQuota(requestId)`，返回有 `abort(): void` 的 Promise。TypeScript model 为公开 camelCase 契约的类型化结果，decimal 转为 `string | undefined`；没有 Number 金额。

`formatUSD(value: string | undefined, pnl?: boolean): string`、`formatQuantity(value: string | undefined): string`、`normalizeDecimal(value: string): string` 为纯函数。`WalletQueryController` 提供 `query(wallet: string, days: 7|30|90, refresh?: boolean): Promise<void>`、`filter(token: string): Promise<void>`、`sort(order: 'asc'|'desc'): Promise<void>`、`loadMore(region: 'tokens'|'transactions'): Promise<void>`、`retry(region: 'summary'|'tokens'|'transactions'): Promise<void>`、`dispose(): void`、`subscribe(listener): () => void`；listener 参数为下面约定的 WalletQueryState。

`WalletQueryState` 包含 activeQuery、tokenFilter、order、quota、summary／tokens／transactions 三份独立 RegionState；RegionState 含 status、data、meta、issue、nextCursor、loadingMore。新输入草稿与 activeQuery 分开，数据标签始终使用 meta 的实际 wallet／range。

- [ ] **8.1 写精度与协议红灯。** 选择大整数、小数、科学计数、负零及微量；未定义值必须输出 Unavailable。复制／Full values 保留原字符串。

```ts
test('preserves tiny values and separates zero from missing', () => {
    expect(formatUSD('-0.0', true)).toBe('$0.00');
    expect(formatUSD('0.0001', true)).toBe('+<$0.01');
    expect(formatUSD(undefined)).toBe('Unavailable');
    expect(formatQuantity('0.00000181579280252')).toBe('≈0.000001816');
    expect(normalizeDecimal('1.81579280252e-6')).toBe('0.00000181579280252');
});
```

- [ ] **8.2 运行红灯并实现格式化。** `cd ui && yarn test --runInBand --runTestsByPath src/app/member/pages/wallet-analytics/format.test.ts`。把已确认预览中的 BigInt 十进制格式化提炼为纯 TS；增加科学计数到普通十进制的字符串展开与输入长度／指数界限，不先 Number 再转字符串。金额千分位两位、盈亏符号、数量舍入及完整值遵循 v1。
- [ ] **8.3 接入现有请求服务。** scope 固定 `{module: AccountDataModule.Token, mode:'read'}`，所有五个方法都使用它。方法 path 以 `/tokens/wallet-analytics` 开头，由现有 requests 处理 `/api/v1`、部署前缀、realm 和认证；结果解析检查 outer result／meta，失败不能转成空数组。

```ts
const scope = {module: AccountDataModule.Token, mode: 'read' as const};
const req = requests.post('/tokens/wallet-analytics/queries', scope).send({
    walletAddress: wallet, periodDays: days, refresh
});
const promise = req.then(response => parseQuery(response.body.query)) as AbortablePromise<WalletAnalyticsQuery>;
promise.abort = () => req.abort();
```

`parseQuery(body: unknown): WalletAnalyticsQuery` 定义在 models.ts，按公开契约校验 UUID、合法天数和日期；模型中只接受单一 camelCase 格式，金额字段必须是 value 包装的字符串，不对异常类型补零。

- [ ] **8.4 实现独立请求和竞态保护。** 创建 query 后同时启动三个区域与 quota；不要 `Promise.all` 任一失败就清整页。钱包／日期 Query 使用新 generation，迟到的旧响应被丢弃；filter 只更新两列表的 generation；sort 只影响逐币。dispose、退出、账号切换或 Token 撤权时 abort 全部请求并清当前页面数据；不把地址结果写到跨账号的浏览器持久缓存。

一批实际数据请求完成后，再读取一次 quota，以展示这些响应新带回的余额；有新鲜余额观测时此调用只读服务端状态，不再访问 account。控制器不为额度提示设置常驻供应商轮询，quota 的迟到响应也受 generation 约束。
- [ ] **8.5 实现筛选、排序与加载。** 应用完整合约地址后清两列表页链、保留摘要，不裁掉交易里其他资产；清除过滤回到钱包两列表。反向排序传现有 snapshotId 并重置逐币展示第 1 批。加载失败保留已有行和 cursor，用户重试用新的 requestId；同次网络重新投递沿用原 requestId。
- [ ] **8.6 写可观察的控制器回归。** 测试旧钱包请求晚到不能覆盖新钱包；摘要成功而逐币失败仍可读；先加载 20 再失败不显示“已全部加载”；过滤后的总览 ID 与原值不变；有效缓存显示其原日期；旧数据过 retainUntil 隐藏并提供重新查询。时间提示可本地更新，但没有查询时不自动取数。
- [ ] **8.7 注册服务并验证。** 在 MemberServices 增加 `walletAnalytics: WalletAnalyticsService`，仅在会员业务服务初始化中构造；沿用 registry 的账户与 realm 管理。运行本任务 service／models／controller／format 的 Jest 指定路径测试和 `yarn lint`，提交说明 `feat: add wallet analytics client and query state`。

## 任务 9：按已确认 v1 实现正式钱包页

**文件：** 新增 `ui/src/app/member/pages/wallet-analytics/{page,summary,tokens,transactions,status}.tsx`、同目录 `wallet-analytics.css` 和 `page.test.tsx`；修改 `ui/src/app/member/{app,routes}.tsx` 与现有 `ui/src/app/app.test.tsx`；复制两份已确认字体到 `ui/src/assets/fonts/wallet-analytics/`。

**接口：** `WalletAnalyticsPage({ownerId}: {ownerId: string})` 为会员路由组件。SummaryPanel、TokenTable、TransactionsList、RegionStatus 只消费任务 8 的 RegionState 与回调；所有取数由 controller 负责。页面路由 `/tokens/wallet-analytics`，Token 子菜单同名；把 Token 加入 moduleLandingPaths，READ 用户可直接到达。

- [ ] **9.1 写用户操作红灯。** Testing Library 渲染页面并使用可控 service：输入钱包、选择 30 天、Query wallet，先看到摘要再看到两份列表；验证页面没有 Win rate、样本切换器、Key 输入或本地交易分类。用角色和可访问名称选择元素，不依赖内部组件结构。

```tsx
await user.type(screen.getByRole('textbox', {name: 'Wallet address'}), wallet);
await user.click(screen.getByRole('button', {name: 'Query wallet', exact: true}));
expect(await screen.findByRole('heading', {name: 'Token performance'})).not.toBeNull();
expect(screen.queryByText('Win rate')).toBeNull();
```

测试所用 wallet 是上述 B 地址，user 由 `userEvent.setup()` 创建；使用当前 Jest 的标准 matcher 和 Testing Library 查找，视觉可见性在浏览器中验证，不为单个断言引入额外依赖。

- [ ] **9.2 复刻业务内容与本地样式。** 完整预览 `docs/requirements/token/previews/wallet-analytics-v1.html` 是视觉依据；不得拿构建模板当页面产物。复用现有会员 shell 和路由，不复制出第二套假导航。钱包 route 为 shell 添加 `wallet-analytics` surface 标识，在本页 CSS 内应用已确认的背景、侧栏／页头密度、字体和控件；离开页面清理标识。本页始终深色，钱包 route 不显示旧主题切换入口；全站其他页面按各自改版任务推进。

```css
html[data-ui-surface='wallet-analytics'] {
    color-scheme: dark;
    --athena-bg: #06080B;
    --athena-panel: #0F1114;
    --athena-text: #FFFFFF;
    --athena-muted: #9FA0A1;
    --athena-primary: #00FFA7;
    --athena-focus: #00FFA7;
}
.wallet-analytics-page .num { font-variant-numeric: tabular-nums lining-nums; }
.wallet-analytics-page .address { font-family: 'JetBrains Mono', monospace; }
html[data-ui-surface='wallet-analytics'],
html[data-ui-surface='wallet-analytics'] body,
html[data-ui-surface='wallet-analytics'] #app { min-width: 0; }
```

以完整 v1 的 token 值核对其余边框／弱文字／按钮文字色，不仅换一个强调色。字体来自 `docs/requirements/web-ui/previews/files/InterVariable.woff2` 与 `JetBrainsMono-Regular.woff2`，记录来源和摘要；构建以本地文件提供，不临时加载在线字体。全站改版若已在执行分支提供相同公共 tokens，直接使用实际公共实现，避免重复同一套值。

- [ ] **9.3 展示实际元数据。** 正式页移除研究样本 banner、样本 A／B 和模拟状态选择器；`request started` 改为真实 meta.fetchedAt，分别展示三部分状态和实际范围。完整值仍可查看，缺字段为 Unavailable；来源说明沿用供应商口径，不新增费用、胜率或现金收益解释。
- [ ] **9.4 实现完整交互。** 地址校验、日期切换、Query、Refresh data、合约筛选／清除、两种排序、两份 Load more、分区 Retry、低额度与耗尽提示分别连到 controller。Refresh 对当前已执行查询和可见筛选生效，输入草稿不改写旧结果身份。quota 失败独立提示，不遮挡数据。
- [ ] **9.5 实现桌面和手机阅读。** 桌面三段内容保持 v1 顺序，列表 760px 及以下转分组行；手机持仓数量与价值整行显示。复制失败展示可选中的全文回退；ETH 原生标识没有 ERC-20 链接，交易哈希格式正确才生成 Etherscan 链接。金额颜色同时有正负号，错误同时有正文，focus 可见。
- [ ] **9.6 验证导航和账号边界。** Token READ／READ_WRITE 导航与直接 URL 均可进入，NONE 不能渲染数据；只有 Token 权限的用户 landing 正确。Wallets 保留自己的入口和权限。既有 account change 清理及 request abort 要实际触发 controller.dispose；清理不能只在正常离开路由时发生。
- [ ] **9.7 验证并提交。** 运行指定的页面／路由 Jest、`yarn lint`、`yarn build`；保存 v1 对照截图前先通过数据交互回归。提交说明 `feat: render the approved wallet analytics page`。

## 任务 10：浏览器业务验收、文档和环境收尾

**文件：** 新增文件表中的五个钱包 e2e 文件与 `internal/tokenwalletanalytics/acceptance/ui_harness_integration_test.go`；新增 `internal/server/wallet_analytics_ui_harness.go`（仅 `uiharness` 构建）、修改 Playwright config／acceptance runner 及其测试；更新测试文档、长期需求和设计。

**接口：** 新验收选择参数 `UI_ACCEPTANCE_FEATURE=wallet-analytics` 仅供测试 runner 使用，默认行为仍执行已有套件；`UI_ACCEPTANCE_MODE=wallet-real` 是新增的真实钱包业务模式，独立于只检查应用壳的 `smoke`。测试配置同步接受 `ATHENA_UI_E2E_MODE=wallet-real`。真实模式只连接已准备的目标环境，不创建供应商替身，不拥有该环境生命周期。

- [ ] **10.1 先写 runner 路由红灯。** 在 `ui/scripts/acceptance-runner.test.mjs` 验证钱包 feature 选择自己的 `TestWalletAnalyticsUIHarness`，隔离模式仍逐个运行根路径与 `/athena`；a11y 选择钱包 axe 文件；wallet-real 只选择真实 spec 和系统 Chrome。拒绝未知 feature／mode、非 loopback URL、不合法路径前缀。原默认套件匹配和退出状态不能被改变。
- [ ] **10.2 准备独立钱包 harness。** 新 harness 使用现有 account-state 测试建库、session／bootstrap 和真实钱包服务注册，固定 Nansen RoundTripper 读取保存样本，生成 member A／B、Token NONE、管理员会话。供应商延迟、429、耗尽、写库失败和撤权通过只在测试构建中的控制入口触发。清理只处理该 harness 的资源；不启动 Token 扫描或 Trader Sync 业务任务。

```text
ui-fixtures：真实 React → 截获 ATHENA 响应，核对布局／字段／交互
live：真实 React → 本地 gateway／权限／PostgreSQL → Nansen 替身
wallet-real：真实 React → 目标开发 API／PostgreSQL → api.nansen.ai
smoke：已有会员／管理员应用壳检查，继续保持原证据范围
```

- [ ] **10.3 接入测试匹配。** feature 为钱包时，Playwright 的 ui-fixtures／live／a11y 项目分别匹配 `wallet-analytics.spec.ts`／`wallet-analytics-live.spec.ts`／`wallet-analytics-a11y.spec.ts`；wallet-real 匹配 `wallet-analytics-real.spec.ts`。runner 真正传入 feature，不能只修改 config 或新增未被执行的测试文件。钱包长请求用例单独设置 90 秒测试超时，覆盖后端 60 秒预算和清理，不提高其他测试超时。
- [ ] **10.4 写桌面／手机与异常用例。** 1440、390、320 宽度和 200% 文字尺寸；查询、精确过滤、清除、反向排序、每批 20、第二页失败重试、完整值／复制回退、无数据、单字段缺失、负零、微量、独立区域失败、旧数据范围、quota 低于／恢复阈值、无权限与请求竞态。对 Etherscan 链接核对 href，不在验收中批量访问外部交易页面。

```ts
test('token failure does not remove the successful wallet summary', async ({page}) => {
    await installWalletAnalyticsRoutes(page, 'tokens-failed');
    await page.goto(memberPath('/tokens/wallet-analytics'));
    await page.getByRole('textbox', {name: 'Wallet address'}).fill('0xfb67ef6fe609edab1e0595e6815634e8e4db9cf7');
    await page.getByRole('button', {name: 'Query wallet', exact: true}).click();
    await expect(page.getByText('+$64,430.55', {exact:true})).toBeVisible();
    await expect(page.getByRole('region', {name:'Token performance'}).getByRole('button', {name:'Retry'})).toBeVisible();
});
```

`installWalletAnalyticsRoutes(page, scenario)` 与 `memberPath(path)` 定义在新 fixtures.ts：前者安装对应 fixture 且记录请求 ledger，后者用 `ATHENA_UI_E2E_PATH_PREFIX` 构造会员路径；不能从 Trader Sync 的业务 fixture 引入耦合。page 的区域标签与按钮名称按任务 9 实现为真实可访问名称。

- [ ] **10.5 运行隔离和无障碍检查。** 以下是任务 10 完成 runner 接入后的命令，本计划整理时尚不可当作已有钱包验证入口使用。

```bash
make ui-acceptance UI_ACCEPTANCE_FEATURE=wallet-analytics
make ui-a11y UI_ACCEPTANCE_FEATURE=wallet-analytics
```

保留原始 axe、trace、截图、HTTP ledger、fixture／live 分类和实际退出码。使用 Impeccable 对正式页面截图与已确认 v1 做最终 UI 审阅；修复实质差异，保留既定品牌及数字语义。不能把文档预览旧截图拿来当正式页面验收截图。

- [ ] **10.6 复用或准备真实环境。** 用 `make runtime-status`、运行进程 cwd 和 `.run/instances/<instance>/state.json` 核对目标；未启动则从执行工作区按本地运行文档 `make run`，保存持久会话和日志。核对 `/`、`/admin/` 及两个 realm 的 bootstrap 后运行已有 smoke。连接失败需排查和重验，不因隔离模式通过而结束。

```bash
make ui-acceptance UI_ACCEPTANCE_MODE=smoke
make ui-acceptance UI_ACCEPTANCE_MODE=wallet-real UI_ACCEPTANCE_FEATURE=wallet-analytics
```

显式非默认地址通过 `UI_ACCEPTANCE_BASE_URL` 传入，值必须是本任务核对过的 loopback 目标。真实账号来自目标开发环境的合法会话；如使用保存 storageState，必须来自该环境真实登录，不能拿隔离 harness 会话代替。使用 disabled-auth 的开发身份时在报告中说明，不能宣称验证了 Google／Phantom 登录。

- [ ] **10.7 完成真实钱包流程。** 至少一次钱包 B／30 天查询、有效缓存再查、精确合约过滤、清除、反向排序、可用时真实交易下一页及手动刷新。默认先复用任务 7 同一环境已取得数据；必要刷新记录真实新消耗。过滤向上游发起完整合约查询，不局限于已加载页。实时余额和金额以本轮响应对照，未返回下一页时不能捏造一次真实分页通过。
- [ ] **10.8 核对请求和结果。** 浏览器只访问 ATHENA，不收到 Nansen Key；API attempts 与真实出网逐一对应。记录实际数据端点、页数、缓存命中、请求 UUID、发生时间与 credits 的已知／未知状态。401／429／额度耗尽／24 小时等故障只在受控场景验证，不为制造错误而消耗真实额度或修改账户套餐。
- [ ] **10.9 完成范围自检与最终回归。** 结合下表逐项附证据；仅在代码又发生变化或仍有未解问题时重跑相应检查，不无理由重跑全部生成器。整体验收文件写明哪一条是真实供应商、哪一条是保存样本或受控故障，以及真实大集合限制。更新需求、长期设计和本计划完成标记为实际事实。
- [ ] **10.10 审阅、提交与收尾。** 按 `requesting-code-review` 对实际改动审阅并解决问题；提交说明 `test: accept wallet analytics flows and document operation`。停止本任务创建的临时服务、harness 和容器，保留数据卷与证据；已有共享环境不停止。使用所属工作区的 `make stop INSTANCE=<已记录实例名>` 或 `make stop-instance INSTANCE=<已记录实例名>`，不是任意默认实例；核对进程、端口和容器确实退出。工具自动清理失败也要排查并记录。
- [ ] **10.11 仅在完整实施和必要验收完成后通知。** 所有任务完成、必要真实验收通过并完成收尾后，从仓库根目录发送一次以下通知，再返回最终交付；计划阶段或任何必要验收未完成时不发送。

```bash
make notify-task-complete \
  TASK_NOTIFICATION_SUBJECT='任务完成：钱包 Nansen 数据展示' \
  TASK_NOTIFICATION_BODY='已完成：钱包总览、逐币表现和交易明细接入；验证：后端、页面与真实环境验收通过。'
```

## 存储接口细化

为避免任务间各自发明签名，任务 2 的 adapter 使用以下具名结构和方法。全部属于 store 包，不对浏览器公开；时间比较使用调用方注入的同一 now 并在事务内核对 generation。所有 UUID 以规范化 string 传入 store，SQL 使用 uuid 类型。

```go
type Snapshot struct {
    ID, SemanticKey, Scope, Region string
    MappingVersion int
    Meta model.Meta
    Request, Payload json.RawMessage
    RawResponses []byte
}
type Cursor struct {
    ID, QueryID, Scope, Wallet, TokenAddress, Region, SnapshotID, Order, LineageID string
    PeriodDays, Page, Offset int
    Range model.Range
    ExpiresAt time.Time
}
type OperationKey struct { Kind, Key string }
type Lease struct { OperationKey; Owner string; Generation int64; LeaseUntil time.Time }
type OperationResult struct {
    Status, SnapshotID string
    Issues []model.Issue
    ExpiresAt time.Time
}
type Operation struct { Lease; Result OperationResult }
type Permit struct { ID, Scope string; ExpiresAt time.Time }
type PermitDecision struct { Permit *Permit; RetryAt time.Time; Issue *model.Issue }
type Attempt struct {
    ID, ActorID, OperationID, Scope, Endpoint string
    StartedAt time.Time
}
type AttemptOutcome struct {
    FinishedAt time.Time
    ProviderRequestID string
    HTTPStatus *int
    Issue *model.Issue
    QuotedCredits, ChargedCredits, RemainingCredits *model.DecimalValue
}
type ProviderState struct {
    Scope string
    Remaining *model.DecimalValue
    ObservedAt, CooldownUntil *time.Time
    RecentRequestTimes []time.Time
    Windows []RateWindow
}
type RateWindow struct {
    WindowSeconds, Limit int
    Remaining *int
    ObservedAt, ResetAt time.Time
}
type StoreContract interface {
    InsertQuery(context.Context, string, model.Query) error
    GetQuery(context.Context, string, string, time.Time) (model.Query, error)
    FindSnapshot(context.Context, string, time.Time) (*Snapshot, error)
    LoadSnapshot(context.Context, string, time.Time) (*Snapshot, error)
    PublishSnapshot(context.Context, Lease, Snapshot, time.Time) error
    CreateCursor(context.Context, Cursor) error
    ResolveCursor(context.Context, string, time.Time) (Cursor, error)
    AcquireOperation(context.Context, OperationKey, string, time.Time, time.Time) (Operation, bool, error)
    GetOperation(context.Context, OperationKey, time.Time) (Operation, error)
    FinishOperation(context.Context, Lease, OperationResult, time.Time) error
    AcquirePermit(context.Context, string, time.Time, time.Time) (PermitDecision, error)
    ReleasePermit(context.Context, Permit) error
    BeginAttempt(context.Context, Attempt) error
    FinishAttempt(context.Context, string, AttemptOutcome) error
    ReadProviderState(context.Context, string) (ProviderState, error)
    ObserveProviderState(context.Context, ProviderState) error
    Prune(context.Context, time.Time, int) (int64, error)
}
```

`InsertQuery` 的 string 是 scope；`GetQuery` 两个 string 为 scope、query ID。AcquireOperation 的 string 为 owner，两个时间为 now、expires；返回 bool 表示本次是否取得租约，lease_until=now+75s。GetOperation 可读取 build 或绑定 generation 的 build_result；等待者只等待其加入的 generation。AcquirePermit 两个时间为 now、请求 deadline。FindSnapshot 使用 semantic key；LoadSnapshot 使用 snapshot ID。这些返回数据只经 Runtime 条件和权限检查后才能交给 transport。

provider_state 的并发更新必须在行锁内合并：缺失余额不覆盖已有观测，较旧观测不覆盖更新观测，冷却不会被乱序成功响应清零，窗口记录不丢写。BeginAttempt 与 AcquirePermit 的原子性可在 store 的同库事务适配中组合；不让网络 IO 持有事务。共享次数上限计算包括账户端点。

## 需求覆盖与交付证据

| 需求／约束 | 实施任务 | 必须留下的证据 |
| --- | --- | --- |
| Nansen 原字段、零／未知／微量、原生 ETH、时间依据、只选定指标 | 1、8、9 | 原样本 SHA 与逐字段对照、精度测试、正式页面截图。 |
| 任意合法 Ethereum 地址、7／30／90、实际 UTC 范围 | 1、5、6、8 | 地址边界、缺省天数、闭区间与旧缓存原范围测试。 |
| 总览不受合约筛选影响；上游精确筛选且保留交易全部资产 | 4、8–10 | 请求 ledger、真实筛选响应、总览不变断言。 |
| 完整逐币排序、20 行切片、反向复用、两组失败不发布排名 | 2、4、5 | 41 行受控集合、同金额／空值、A／B 样本、资源边界与旧 snapshot 指针。 |
| 交易原记录、同哈希、同时间、20 条按需后续页 | 4、5、8、10 | 两页保存样本、cursor 固定条件、错误不标末页。 |
| 10m fresh、24h retain、手动刷新与同次重发 | 2、5、7、8 | 四边界、请求去重、原时间不续期、实际二次请求命中缓存。 |
| 分区失败、字段错误、无成功结果不填零 | 1、5、6、8–10 | 独立区域状态、失败外壳和局部错误可读性。 |
| 跨用户共享、两实例合并、租约 fencing、取消 | 2、3、5 | 两 pool 竞争、旧 owner 发布拒绝、等待／取消、attempt 次数。 |
| READ 权限、创建／quota／缓存均授权，撤权返回前再查 | 5、6、9、10 | gRPC／HTTP 权限矩阵、等待期间撤权、账户切换清空、landing。 |
| Key 环境注入、固定端点、模块故障隔离 | 3、6、7 | 本地白名单与 Compose 测试、无 Key bootstrap、无链扫描查询、浏览器未出网。 |
| 额度低提醒、耗尽后恢复、未知扣款、共享限速 | 2、3、5、7、10 | 真实响应头／attempt 对照、受控 429／充值、5/s 与 4 并发测试。 |
| Nansen v1 视觉、键盘、手机、复制回退与本地字体 | 8–10 | 1440／390／320／文字 200% 截图、axe 原始报告、键盘和焦点检查、最终 UI 审阅。 |
| schema 显式迁移、生成源与消费者、运行和关闭归属 | 2、6、7、10 | sqlc/proto 生成检查点、contract、同库 Verify、实例日志和停止证据，SDS-R1–R8。 |

## 执行前准备与本轮整理验证

执行时使用 `using-git-worktrees` 核对隔离工作区；若需要新 worktree，放在 `.worktrees/`，分支使用 `codex/` 前缀。本计划及相关设计当前可能尚未提交，必须把本功能所需的已确认文档与样本带入执行工作区并核对摘要，不能从旧默认分支开始后遗失它们。保留用户和其他任务的改动，不自动 stash 或提交无关文件。

本轮已核对 Go 与 Node 版本、gogofast 生成方式、现有 API 注册、权限、环境白名单、显式 schema 工具和浏览器 runner；确认 sqlc-local 通过 Go 运行，不依赖 `dist/sqlc`。数据库、供应商、真实 UI 和新测试入口均未在本轮启动或执行。

本计划整理的完成检查只验证文档完整性：现有引用可定位、新增路径明确、任务间接口一致、每项业务要求映射到任务、无空白实现步骤、预览确认文件及其产物摘要不变。执行结果另写入 `docs/testing/token-wallet-analytics.md`，不能把本节文档检查当成实施测试通过。
