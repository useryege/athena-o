# Trader Sync 内部 gRPC 字段与映射契约

> 状态：本轮补全的契约草案，待审阅；尚未创建或生成正式 proto。职责、权限、事务及构建运行方式沿用两份已确认规格，不重复审批。

依据：[服务边界](2026-09-13-trader-sync-service-boundaries-design.md)、[构建与运行](2026-09-13-trader-sync-local-runtime-design.md)、[当前公开 RPC](../../../internal/server/tradersync/tradersync.proto)、[公开资源类型](../../../pkg/apis/application/v1alpha1/trader_sync_types.go)及[现有映射](../../../internal/server/tradersync/tradersync.go)。适用 SDS-R2、R4、R6、R7、R8。

## 1. 协议选择与责任

采用独立强类型内部消息：`internal/tradersync/apiclient/trader_sync.proto`，package `tradersync.internal.v1`，Go package `github.com/useryege/athena/internal/tradersync/apiclient`，服务名 `TraderSyncService`。只定义内部消息和 gRPC，不导入公共 `application.v1alpha1`、HTTP annotations、Swagger 或 API runtime。

| 可选方式 | 判断 |
| --- | --- |
| 独立消息、显式映射 | 本次推荐；内部服务所有契约可独立生成和编译，字段含义明确 |
| 直接引用公共 v1alpha1 DTO | 减少映射，但使内部服务跟随公共聚合 API 类型一起演进，本次不采用 |
| JSON bytes、Struct 或 Any 整包传递业务结果 | 将字段与 presence 约束移出 proto，容易出现数值转换和隐式契约，本次不采用 |

API 从可信会话构造 Actor，将请求字段逐项映射；Trader Sync 校验业务参数、查询当前权限、调用领域服务并形成内部结果；API 再逐项映射成现有公共 DTO。当前 `publicResolved/publicSubscription/publicActivity` 中的默认区间、观察状态合成、价格 evidence、资料缺失判断移到 Trader Sync 的结果映射中，API 不重算这些结果。

公共路径、公开消息字段号、JSON 字段名和现有业务行为保持。内部消息以现有业务视图为形状，独立持有声明；不能通过 JSON 序列化或不安全类型转换代替明确映射。

## 2. Actor、存在性与字段基础规则

```proto
syntax = "proto3";
package tradersync.internal.v1;
option go_package = "github.com/useryege/athena/internal/tradersync/apiclient";

enum ApplicationRealm {
  APPLICATION_REALM_UNSPECIFIED = 0;
  APPLICATION_REALM_MEMBER = 1;
  APPLICATION_REALM_ADMIN = 2;
}
message Actor {
  string account_id = 1;
  ApplicationRealm realm = 2;
}
message StringValue { string value = 1; }
message BoolValue { bool value = 1; }
message NoteInput { string value = 1; }
message PageInput { int32 page_size = 1; string cursor = 2; }
message PageInfo { string next_cursor = 1; }
message ActivityPageInfo {
  string next_cursor = 1;
  string refresh_cursor = 2;
  string snapshot = 3;
  string as_of = 4;
  bool has_newer = 5;
}
```

每个请求必须有 `Actor actor = 1`，缺失、非规范账户 UUID 或 realm 为零/未知值被拒绝。内部 Bearer 放在 metadata，不进入请求消息；Actor 没有 grant、administrator 布尔值、session、公开 JWT 或 API Key 字段。

API 通过 `session.AccountID(ctx)` 取得已认证账户 ID，再通过已有 `credentialMgr.Get(id).ApplicationRealm()` 取得持久 realm；不从用户提交的 `account_id` 过滤器或任意 header 构造 Actor，不要求 Bearer/API Key 新增 realm header。存储事务继续执行当前 member/grant/owner 或 administrator 检查；Actor 的 realm 不是权限证明。

| 数据 | 内部表示与映射规则 |
| --- | --- |
| revision、generation、interval epoch | `uint64`，保持完整二进制精度；公共 JSON 继续使用现有 `,string` 标签 |
| 活动/批次/投递 ID、计数、原始金额、position ID、价格分子分母、decimal | 延续现有公开契约的精确十进制 `string`，没有 double/float 中转。数据库数值的规范化与边界校验留在服务中 |
| 时间与时间过滤器 | 日期时间字段保持现有 RFC3339Nano 字符串契约。服务端按当前 `time.Parse` 规则接收过滤器并转 UTC；输出在服务端格式化为 UTC。未知非可选时间保持现有空串，可选时间用 `StringValue` 的缺失表达，不制造 Unix epoch。例外是 `CurvePoint.t`：沿用现有 Unix 秒的十进制字符串，原样传输，不改成 RFC3339 或毫秒 |
| 可选 string/bool | 使用可空消息 `StringValue` / `BoolValue`；nil 是缺失，非 nil 的 `""`、`"0"`、false 都是真实值。避免依赖现有 gogo 工具链对 proto3 optional scalar 的支持 |
| 可选对象 | 对象消息可缺失，不能经 `GetX()` 取默认对象后误当作真实证据 |
| repeated | 按服务返回顺序传输，不排序、不去重；空列表沿用现有公共 gRPC→gateway 结果。不把 Go 的 nil/空 slice 本身当作公共存在性保证 |
| evidence | 保留 availability、reason_code、source、queried_at；unavailable 时 value 缺失。资料 unavailable 与整个服务不可用是不同结果 |

`Curve.points` 和 `TradeMetadata.legs` 使用可空列表消息，保留领域中的未知与已知空列表直至 facade 映射：`CurvePointList { repeated CurvePoint items = 1; }`、`ComboLegList { repeated ComboLeg items = 1; }`。公开 repeated 字段的最终空值编码继续由现有公共 proto/gateway 决定；可观察的资料状态以 evidence 为准，不声称既有公共 protobuf 能区分 nil 和空 repeated。

Create 的 `note` 保留独立 `NoteInput`：公共未传或 JSON null → 内部 nil；公共 `{"value":""}` → 存在的空 NoteInput，表示明确清空。UpdateTargetNote 的 `note` 仍是普通 string，空串是更新值。

## 3. 16 个请求与响应根消息

以下每行定义同名 `MethodRequest/MethodResponse`。请求字段前自动加入 `actor:Actor = 1`，表中业务字段按顺序从 2 编号；响应字段按顺序从 1 编号。`?` 表示允许缺失的消息，`[]` 表示 repeated，不是 proto 语法。未列为 `?` 的响应对象必须由服务提供；缺失是内部契约错误，API 返回局部 503。

| RPC | 请求业务字段（从 2 开始） | 响应字段（从 1 开始） |
| --- | --- | --- |
| ResolveTarget | input:string | target:ResolvedTarget |
| CreateSubscription | confirmation_token:string；request_id:string；note:NoteInput? | subscription:Subscription |
| ListSubscriptions | page:PageInput?；view:string；state:string | subscriptions:Subscription[]；page:PageInfo；quota:Quota；as_of:string |
| GetSubscription | subscription_id:string | subscription:Subscription |
| PauseSubscription | subscription_id:string；expected_revision:uint64；request_id:string | subscription:Subscription |
| ResumeSubscription | subscription_id:string；expected_revision:uint64；request_id:string | subscription:Subscription |
| CancelSubscription | subscription_id:string；expected_revision:uint64；request_id:string | subscription:Subscription |
| UpdateTargetNote | wallet:string；note:string；expected_revision:uint64；request_id:string | note:TargetNote |
| ListActivities | page:PageInput?；subscription_id:string；from:string；to:string；summary_batch_id:string；refresh_cursor:string | activities:Activity[]；page:ActivityPageInfo |
| GetActivity | activity_id:string | activity:Activity |
| ListSubscriptionHistory | subscription_id:string；page:PageInput? | entries:HistoryEntry[]；page:PageInfo；as_of:string |
| GetSummaryBatch | batch_id:string | batch:SummaryBatch |
| ListSummaryParts | batch_id:string；activity_id:string；page:PageInput? | parts:SummaryPart[]；page:PageInfo；as_of:string |
| ListSubscriptionSummaries | page:PageInput?；account_id:string；state:string；wallet:string；include_cancelled:bool | summaries:SubscriptionSummary[]；page:PageInfo；as_of:string |
| GetSubscriptionSummary | subscription_id:string | summary:SubscriptionSummary |
| GetTraderSyncRuntimeStatus | 无业务字段 | status:RuntimeStatus |

前 13 个方法是 member 方法，后 3 个是 administrator 方法；最后一个请求仍含 Actor，不能使用空请求绕过权限。管理员请求中的 `account_id` 只是查询过滤器，不能替换 Actor。

例如：

```proto
message CreateSubscriptionRequest {
  Actor actor = 1;
  string confirmation_token = 2;
  string request_id = 3;
  NoteInput note = 4;
}
message CreateSubscriptionResponse { Subscription subscription = 1; }
```

## 4. 结果消息完整字段目录

下面从当前公开资源定义逐字段核对，内部消息去掉 `TraderSync` 前缀；字段号沿用对应资源字段号。字段类型按第 2 节转换，可选对象保持可选；非可选对象映射时必须存在。该目录只是本轮的设计输入快照，正式 proto 成为实现后的权威契约。

| 内部消息 | 字段：编号 名称:类型 |
| --- | --- |
| `FieldEvidence` | 1 `availability:string`；2 `reason_code:string`；3 `source:string`；4 `queried_at:string` |
| `StringField` | 1 `evidence:FieldEvidence`；2 `value:StringValue?` |
| `DecimalField` | 1 `evidence:FieldEvidence`；2 `value:StringValue?` |
| `BoolField` | 1 `evidence:FieldEvidence`；2 `value:BoolValue?` |
| `TimeField` | 1 `evidence:FieldEvidence`；2 `value:StringValue?` |
| `CurvePoint` | 1 `t:string`；2 `p:string` |
| `Curve` | 1 `evidence:FieldEvidence`；2 `points:CurvePointList?` |
| `PnLView` | 1 `period:string`；2 `amount:DecimalField`；3 `curve:Curve`；4 `interval:string`；5 `fidelity:string`；6 `reference_time:TimeField`；7 `timezone:StringField` |
| `ResolvedTarget` | 1 `wallet:string`；2 `canonical_profile_url:string`；3 `avatar:StringField`；4 `display_name:StringField`；5 `verified:BoolField`；6 `joined_at:TimeField`；7 `position_value:DecimalField`；8 `largest_win:DecimalField`；9 `predictions:DecimalField`；10 `pn_l:PnLView[]`；11 `default_period:string`；12 `confirmation_token:string`；13 `expires_at:string`；14 `usage_notice:string`；15 `saved_note:TargetNote?`；16 `existing_subscription:ExistingSubscription?`；17 `quota:Quota` |
| `TargetNote` | 1 `wallet:string`；2 `note:string`；3 `revision:uint64` |
| `Quota` | 1 `used:int32`；2 `limit:int32` |
| `ExistingSubscription` | 1 `id:string`；2 `status:string`；3 `revision:uint64` |
| `TargetDisplay` | 1 `display_name:StringField`；2 `avatar:StringField`；3 `profile_url:StringField` |
| `Subscription` | 1 `id:string`；2 `wallet:string`；3 `status:string`；4 `revision:uint64`；5 `generation:uint64`；6 `note:string`；7 `note_revision:uint64`；8 `created_at:string`；9 `updated_at:string`；10 `paused_at:StringValue?`；11 `cancelled_at:StringValue?`；12 `permission_disabled_at:StringValue?`；13 `current_interval:Interval?`；14 `observation:Observation`；15 `binding_status:string`；16 `queue_notice:string`；17 `queue_counts:StatusCounts`；18 `target_display:TargetDisplay` |
| `Interval` | 1 `effective_at:string`；2 `ended_at:StringValue?`；3 `generation:uint64`；4 `epoch:uint64` |
| `Observation` | 1 `state:string`；2 `reason:string`；3 `last_reliable_at:StringValue?`；4 `latest_interruption:Interruption?`；5 `interruption_count:string` |
| `Interruption` | 1 `start:StringValue?`；2 `end:StringValue?`；3 `recovered_at:StringValue?`；4 `reason:string`；5 `uncertainty:string`；6 `possible_missing:bool` |
| `HistoryEntry` | 1 `id:string`；2 `kind:string`；3 `sort_at:string`；4 `interval:Interval?`；5 `interruption:Interruption?` |
| `Activity` | 1 `id:string`；2 `subscription_id:string`；3 `source_record_id:string`；4 `wallet:string`；5 `side:string`；6 `position_id:string`；7 `collateral_raw:string`；8 `shares_raw:string`；9 `fee_raw:string`；10 `collateral_symbol:string`；11 `collateral_decimals:int32`；12 `shares_decimals:int32`；13 `price_numerator:string`；14 `price_denominator:string`；15 `price_evidence:FieldEvidence`；16 `source_version:string`；17 `settled_at:string`；18 `received_at:string`；19 `recorded_at:string`；20 `public_time_evidence:FieldEvidence`；21 `metadata:TradeMetadata`；22 `note_snapshot:string`；23 `notification_mode:string`；24 `notification_reason:string`；25 `delivery:Delivery?`；26 `summary_progress:SummaryProgress?`；27 `target_display_snapshot:TargetDisplay`；28 `finality_anomaly:FinalityAnomaly?`；29 `source_location:SourceLocation` |
| `SourceLocation` | 1 `chain_id:string`；2 `exchange_address:string`；3 `transaction_hash:string`；4 `block_hash:string`；5 `block_number:string`；6 `log_index:string` |
| `FinalityAnomaly` | 1 `reason:string`；2 `detected_at:string`；3 `published_block_hash:string`；4 `conflicting_block_hash:StringValue?` |
| `MarketRef` | 1 `evidence:FieldEvidence`；2 `id:string`；3 `title:string`；4 `url:string`；5 `condition_id:string`；6 `position_id:string`；7 `outcome:string` |
| `ComboLeg` | 1 `position_id:string`；2 `market:MarketRef` |
| `TradeMetadata` | 1 `market:MarketRef`；2 `legs_evidence:FieldEvidence`；3 `legs:ComboLegList?`；4 `relationship:string` |
| `Delivery` | 1 `id:string`；2 `status:string`；3 `reason:string`；4 `authorized_at:StringValue?`；5 `started_at:StringValue?`；6 `result_at:StringValue?`；7 `message_id:StringValue?`；8 `attempt_count:string`；9 `latest_attempt:Attempt?` |
| `Attempt` | 1 `index:string`；2 `authorized_at:string`；3 `started_at:StringValue?`；4 `result_at:StringValue?`；5 `status:string`；6 `reason:string` |
| `StatusCounts` | 1 `total:string`；2 `pending:string`；3 `sending:string`；4 `sent:string`；5 `failed:string`；6 `unknown:string`；7 `cancelled:string` |
| `SummaryProgress` | 1 `phase:string`；2 `reason:string`；3 `batch_id:StringValue?`；4 `related_part_counts:StatusCounts`；5 `batch_part_counts:StatusCounts`；6 `oldest_at:string`；7 `first_started_at:StringValue?` |
| `TargetCount` | 1 `wallet:string`；2 `count:string` |
| `SummaryBatch` | 1 `id:string`；2 `oldest_at:string`；3 `settled_from:string`；4 `settled_to:string`；5 `recorded_from:string`；6 `recorded_to:string`；7 `first_started_at:StringValue?`；8 `activity_count:string`；9 `target_counts:TargetCount[]`；10 `part_counts:StatusCounts`；11 `as_of:string` |
| `SummaryPart` | 1 `id:string`；2 `index:int32`；3 `total:int32`；4 `delivery:Delivery`；5 `associated_activity_count:string` |
| `SubscriptionSummary` | 1 `subscription_id:string`；2 `account_id:string`；3 `username:string`；4 `email:string`；5 `wallet:string`；6 `status:string`；7 `created_at:string`；8 `updated_at:string`；9 `paused_at:StringValue?`；10 `cancelled_at:StringValue?`；11 `permission_disabled_at:StringValue?`；12 `observation:Observation`；13 `activity_count:string`；14 `associated_delivery_counts:StatusCounts`；15 `as_of:string` |
| `RuntimeMetric` | 1 `name:string`；2 `value:string`；3 `unit:string`；4 `kind:string`；5 `window_start:StringValue?`；6 `window_end:StringValue?`；7 `service_epoch:StringValue?` |
| `RuntimeStatus` | 1 `collector_connected:bool`；2 `collector_epoch:string`；3 `filter_revision:string`；4 `metrics:RuntimeMetric[]`；5 `as_of:string` |

## 5. 校验、分页、错误与超时

- 服务校验 request_id、confirmation_token、UUID、规范正十进制资源 ID、非零钱包、note 长度、revision、过滤器和时间范围，复用当前 Service/store 的校验及错误。API 只完成可信身份与传输映射，不在跨进程前重新生成请求意图或修正业务载荷。
- page 缺失或 page_size=0 使用当前默认 50；合法范围 1–100。空 cursor 是首屏；非空 cursor/refresh_cursor/snapshot 原样传递。签名、owner、查询种类、过滤条件、page size、快照和方向仍由 Trader Sync 校验。API 不解码、重签或“修复”游标。
- 写请求按原 owner/操作/request_id/载荷幂等。API 无自动写重试；连接取消和 DeadlineExceeded 均不证明未提交。UI 的现有重放保留原 request_id，主动重新解析仍可创建新意图。
- 内部认证/Actor 格式错误使用 `google.rpc.ErrorInfo`：domain 固定 `tradersync.internal.v1`。缺失/错误 token 使用 Unauthenticated，reason 分别为 `SERVICE_AUTH_MISSING`、`SERVICE_AUTH_INVALID`；Actor 格式错误使用 InvalidArgument，reason 为 `ACTOR_INVALID`。不放凭据、原请求或数据库错误。API 只识别该 domain/reason 作为内部调用契约故障并转成 Unavailable/HTTP 503。
- 合法 Actor 的数据库权限不足、realm 与允许业务角色不符保持 PermissionDenied；公共身份缺失/公开凭据失效仍由 API 返回 Unauthenticated。既有 InvalidArgument、NotFound、AlreadyExists、Aborted、FailedPrecondition、ResourceExhausted 继续保持含义；连接不可达返回 Unavailable。
- 收到缺失的必需响应对象等内部无效消息时返回 503 并记录安全类别；不能用成功空结果代替，也不能恢复同进程调用。未知服务内部错误使用现有公共安全错误处理，不泄露 SQL/凭据。
- 读与管理员状态默认上限 5 秒，Resolve/写命令默认上限 15 秒；API 和服务端分别取本地上限与父 deadline 的较早值，取消向下传递。不为内部请求重新获得完整预算。
- 两端声明并注入已有 `ATHENA_GRPC_MAX_SIZE_MB`，默认 200 MiB，与公共入口一致；独立 apiclient 自己配置消息上限，不能因复用公共客户端工厂导入 API。超限保持 ResourceExhausted，不截断结果或把缺失解释为成功。

## 6. 映射落点与验证

拟新增 `internal/tradersync/transport/`，负责内部 server、参数校验、领域→内部 DTO 映射及 internal auth/deadline；运行组合根在 `internal/tradersync/runtime.go`。该 transport 不导入 `internal/server` 或 `pkg/apis/application/v1alpha1`。

公共 facade 保留在 `internal/server/tradersync/`，改为生成内部客户端和 Actor resolver 的使用者；`mapping.go` 只负责内部→公共字段和 presence 的对应。既有公开 proto、v1alpha1 Trader Sync 类型和 UI 类型无需因内部命名空间而换成内部类型。

必须取得的契约证据：

1. 16 个 RPC 全覆盖请求字段/响应根映射，包括管理员过滤器与 Actor 不混淆。
2. 字段链经过真实内部 protobuf 编解码、公开 protobuf 编解码及生产 JSON marshaler。覆盖 >2^53 的 ID、uint64 最大值、长整数金额、价格为零、false、空 string、未知时间/投递许可/消息 ID。
3. note 未传、null、显式空；不可用 evidence、无效源 Bool/Time、六个 PnL 区间、Unix 秒字符串 `CurvePoint.t` 原样保留及相同时间的重复曲线点；legs 未知/已知空/部分值在内部保真，公开结果与原 gateway 保持。
4. next/refresh/snapshot 原样传递、has_newer=false、空结果、改变 owner/过滤条件/page size 的游标拒绝。
5. 正常 member、admin、有效 API Key、无 realm header 的 Bearer、错误内部 token、跨 realm/owner、撤权、响应丢失后相同请求重放。
6. 无父 deadline 的内部请求仍受服务端上限；短父预算不被延长；取消释放请求资源。TLS 和消息大小分别测试，不由健康探测替代业务契约。

本文件不声称上述测试已执行。完整任务顺序、文件与命令见[实施计划](../plans/2026-09-13-trader-sync-independent-grpc-service.md)；代码落地时按 sync-athena-changes 稳定 proto 后生成，再修改实际消费者。

## 7. 本轮审阅记录

已对照公开 proto、Go 类型和实际映射逐项核验，并完成一次独立的只读契约审阅：16 个 RPC、34 个资源消息、215 个资源字段均有对应，未发现字段号或 presence 遗漏。修正了时间规则中的 `CurvePoint.t` 例外，明确 Unix 秒字符串；补齐了内部认证/Actor 错误的 gRPC code 与稳定 ErrorInfo reason。两跳、授权及运行测试列入实施计划，目前没有执行，本文仍为待用户审阅的新增契约。
