# 管理员日志接口与页面契约

> 详细设计已于 2026-09-18 整体审阅通过，按用户要求暂不实施；所有新增接口和页面均为目标能力。入口：[总体设计](../2026-09-18-key-operation-logs-design.md)。

## 1. API 清单与权限

公共 proto 目标：`internal/server/operationlog/operationlog.proto`，package `operationlog`，service `OperationLogService`。内部 proto 目标：`internal/operationlog/api/operationlog.proto`，package `operationlog.internal.v1`。所有方法动词在前，查询条件放在 message 字段中。

| RPC | HTTP | 实际处理者／最长预算 |
| --- | --- | --- |
| ListOperationLogs | GET `/api/v1/admin/operation-logs` | API facade → 日志服务，3 秒 |
| GetOperationLog | GET `/api/v1/admin/operation-logs/{operation_id}` | API facade → 日志服务，2 秒 |
| GetOperationLogRuntimeStatus | GET `/api/v1/admin/operation-log-runtime` | API facade → 日志服务，1 秒 |
| GetOperationLogCaptureStatus | GET `/api/v1/admin/operation-log-capture-status` | 当前 API 进程的采集适配器，只读内存、100 ms |
| ListOperationLogActions | GET `/api/v1/admin/operation-log-actions` | API 编译期目录，只读、100 ms |

内部服务提供前三个同名 RPC；不提供用户可调用的写日志、修改日志、删除日志或执行补偿业务的 RPC。后两个 API-only 方法不依赖日志服务可用。

全部加入 administratorGRPCMethods，同时明确要求交互登录能力；沿用 realm 和持久账户管理员校验，API Key、普通会员和匿名用户不可查询。内部服务校验 Bearer 服务身份及 Viewer，再通过账户只读适配器核验当前持久 administrator=true、login_enabled=true。Viewer.accountId 和 credentialKind 只能由 API 根据已验证凭据填入，HTTP 参数不能覆盖。

Viewer 字段为 account_id（UUID）、realm（ADMIN）、credential_kind（LOGIN_SESSION／DEVELOPMENT）、session_binding（32 字节摘要）、access_revision（uint64）。它不包含原始会话令牌。内部服务继续执行权限复核；持久账户读取失败返回 Unavailable，不从缓存断言仍有权限。

新增查询不受六个业务模块开关控制；日志服务故障也不改变这些开关。使用 `Cache-Control: no-store`，不开放会员数据接口。日志查询不会再记录一条用户操作日志。

## 2. ListOperationLogs 契约

请求使用以下固定字段；proto 字段使用 snake_case，公共 JSON 使用明确 camelCase tags，int64/uint64 在 JSON 中为十进制字符串。

| 字段 | 类型／默认值／验证 |
| --- | --- |
| from / to | RFC3339 时间字符串；from 包含、to 不包含；都省略时为服务端 now-7d 到 now；只传一个拒绝；单次跨度上限 90 天，可选择更早的历史区间 |
| actor_query | 最多 128 字节；规范 UUID 精确匹配 accountId，其他合法用户名按不区分大小写前缀匹配 usernameSnapshot |
| actor_role | ALL（默认）、MEMBER、ADMINISTRATOR、UNKNOWN |
| credential_kind | ALL（默认）或已定义来源枚举 |
| module_code / action_code | 可空；非空必须在目录，动作与模块矛盾时 InvalidArgument |
| outcome | ALL（默认）或结果枚举，UNKNOWN 包含 START_ONLY 和明确结果未知 |
| target_account_id | 可空规范 UUID，只筛选被管理员操作的账户；与操作者分开 |
| resource_type / resource_id | 同时为空或同时有值，类型为目录枚举、ID ≤128 字节；主对象精确匹配 |
| page_size | int32，默认 50，允许 1–100 |
| cursor | 可空，最长 4096 字节；有值时其筛选绑定必须与本次规范化参数相同 |

响应：`items: OperationLogSummary[]`、`page: {nextCursor, snapshotToken, snapshotSequence, snapshotAt, pageSize}`、`appliedFilters`。无 nextCursor 表示没有下一页；不返回伪造 total 或推算出的页数。snapshotSequence 为十进制字符串；snapshotAt 是查询快照创建时间，不是最后一条业务时间。

时间默认值只在没有 cursor 的新查询中计算。有 cursor 时先读取其中已签名的 from/to 等规范参数，再核对显式传入的筛选；不能每次用 now 重算默认时间。页面从 appliedFilters 保存精确区间后再翻页。

Summary 字段：operationId、startedAt、actor（accountId、usernameSnapshot、role、realm、credentialKind、identityVerified、identitySnapshotComplete）、moduleCode、actionCode、primaryResource、outcome、observation、reasonCode、durationMs、businessState、resourceCount、resourcesComplete、responseWriteFailed。所有可缺失字段使用 presence／nullable，不以 0、空时间或 Unknown 用户名伪装事实。

## 3. GetOperationLog 契约

请求：`operation_id` 必填规范 UUID；`snapshot_token` 可选。不传读取最新视图；传入则按该快照读取，过期和会话不匹配与列表同规则处理。

响应 `item: OperationLogDetail` 包含 Summary 全部字段，并增加 finishedAt、requestId、parentOperationId、businessRequestId、provider、effect、resources、changes、counts、protocolResult、sourceFacts。

- Resource：type、id、relation（PRIMARY／RELATED）、referenceVerified。仅来自未经业务核验的规范输入时 referenceVerified=false，页面提示“请求目标”；不能将它呈现为确认存在／有权访问的对象。
- Change：fieldCode、beforeValue nullable、afterValue nullable、beforeAvailable、valueKind。只接受目录中的 enum／bool／integer／字段名，不承载自由正文。
- Counts：requested／confirmed／failed／unknown 各自可缺失；不要求未知项补零，也不把钱包原子批量接口显示成逐条部分成功。
- ProtocolResult：grpcCode nullable、httpStatus nullable、reasonCode nullable、responseWriteFailed。
- SourceFacts：producerId、firstReceivedAt、lastReceivedAt、phasesReceived、snapshotSequence。只显示运营所需事实，不暴露数据库凭据或内部 token。

列表快照之后创建的对象在旧 snapshotToken 下返回 NotFound，刷新到最新后才出现。不存在的记录与当前范围不可见记录均不泄漏额外内容。详情不提供“进入会员钱包／交易详情”的越权链接；如未来增加业务链接，必须是管理员已有权限的路由。

## 4. 目录与状态接口

ListOperationLogActions 返回目录版本、模块和动作数组，每项包含 code、moduleCode、label、resourceType；用于筛选及 UI 文案。目录版本用于识别部署是否一致，不保存历史兼容实现。未知 action 不悄悄展示为成功记录，服务将其隔离并报告目录不一致。

RuntimeStatus 包括：serviceEpoch、checkedAt、queryReady、projectionState（READY／BACKLOG／ERROR）、lastPublishedAt nullable、publicationSequence、pendingEvents、oldestPendingReceivedAt nullable、quarantinedEvents、lastProcessingErrorCode nullable、observedProducers[]。计数为 int64 字符串；不可获取时返回错误，不补零。

observedProducers 为持久化采集实例摘要：producerId、startedAt、lastSeenAt、stoppedAt、attemptedEvents、confirmedEvents、unconfirmedEvents、invalidEvents、capacityRejectedEvents、lastFailureAt、lastFailureCode、lastRecoveredAt。默认返回最近 24 小时有更新或未明确停止的实例，最多 100 项，返回 totalObserved 与 producersComplete；存在更多项时明确展示数量限制。

CaptureStatus 返回处理本次查询的 API 实例及同一组进程累计计数、checkedAt、persistenceReachable（true／false／unknown）、lastConfirmedAt nullable。该接口只描述当前 API 实例，不代表整个部署全部实例。

生产者上报每 5 秒一次，lastSeenAt 超过 30 秒标记“状态未更新”，不推断已停止；stoppedAt 只来自正常停机的尽力上报。存在 unconfirmedEvents 时长期保留“本实例曾有记录未确认入箱”的事实，即使当前恢复，也不能声称历史完整。新进程的零计数只属于新 serviceEpoch／producerId。

## 5. 错误语义

| gRPC／HTTP | reason | UI 行为 |
| --- | --- | --- |
| Unauthenticated／401 | 既有认证原因 | 清除管理员 scope，返回登录 |
| PermissionDenied／403 | ACCOUNT_ADMIN_REQUIRED | 沿用管理员身份复查与清屏流程 |
| InvalidArgument／400 | FILTER_INVALID、CURSOR_INVALID、SNAPSHOT_INVALID | 指向筛选或分页问题，不显示空结果代替错误 |
| FailedPrecondition／412 | CURSOR_EXPIRED、SNAPSHOT_EXPIRED | 保留筛选，提供“重新查询”，显式建立新快照 |
| NotFound／404 | OPERATION_LOG_NOT_FOUND | 详情不存在，提供返回列表 |
| Unavailable／503 | OPERATION_LOG_UNAVAILABLE、OPERATION_LOG_SCHEMA_UNAVAILABLE | 保留最后成功的数据与来源时间，显示未能读取 |
| DeadlineExceeded／504 | OPERATION_LOG_QUERY_TIMEOUT | 保留筛选和旧数据，允许用户重试；不自动切换无筛选查询 |

各错误附 ErrorInfo 的稳定 reason；错误正文使用固定映射，不透传可能包含业务正文的任意 Go error。新功能的普通 503 不冒充既有“系统维护中”的认证清除信号。

## 6. 页面结构与交互

采用已有管理员应用视觉系统。定位是查询操作记录，主要界面为列表与独立详情，不建立新的布局风格、主题、统计大屏或编辑模式。

| 列表阅读顺序 | 桌面 | 手机 |
| --- | --- | --- |
| 页头 | Operation Logs、简短说明、Refresh | 标题、说明、Refresh |
| 状态提示 | 查询与采集异常分别提示，正常时保留最后读取时间 | 同样分来源，信息换行，不遮挡筛选 |
| 筛选 | 时间、操作者、模块、动作、结果；More filters 中放角色、凭据来源和目标 | 使用同样标签的纵向筛选区，Apply／Reset 明确可达 |
| 记录 | 时间、操作者、模块、动作、对象、结果，六列 | 每条先显示动作和结果，再列用户、时间、对象 |
| 分页 | Previous／Next、每页数量、快照时间 | Previous／Next 全宽可触达；不显示不存在的总页数 |

默认 7 天由服务端 appliedFilters 固定，日期输入按 UTC+8 展示，发送时转成带时区时间。修改筛选先形成草稿，Apply 才查询；Reset 回到默认 7 天和 50 条。Refresh 使用已应用筛选的新快照，从第一页开始；不会自动清除用户筛选。关键词只查操作者，不搜索备注正文。

列表不自动插入新行。状态源每 10 秒可见 single-flight 刷新，hidden 暂停；列表由 Apply／Refresh／翻页触发。详情默认保持进入时快照，Refresh 显式读取最新。列表、运行状态、当前 API 采集状态三个来源独立加载和失败。

详情依次是：动作与结果 → 操作者与时间 → 对象与变更字段 → 结果解释／错误原因 → 默认折叠的关联标识与采集事实。UNKNOWN 显示“结果未确认”，START_ONLY 补充“尚未收到结果记录”；ACCEPTED 明确显示“已受理，后续结果由对应业务记录确认”。长标识可换行和复制，复制按钮提供可访问名称，复制成功用现有反馈组件。

过滤条件、真实输入 cursor 和 snapshotToken 放在 URL／页面导航 state 中；返回列表保持原位置。游标过期时停留并解释重新查询，不伪造恢复上一页。切换管理员身份或退出，沿用 read-scope 的 generation 检查、取消请求和缓存清理，旧响应不能回填。

英文 UI 文案沿现有应用：Operation Logs、Apply、Reset、Refresh、Succeeded、Accepted、Failed、Denied、Unknown、Partial、Action required、Cancelled。文档使用中文解释；不顺带重做全站 i18n。

## 7. UI 验证矩阵

桌面 1440×900、手机 390×844、200% 缩放下核对：六列阅读顺序／手机记录、可访问标签、键盘筛选／详情／返回、加载／空记录／筛选无结果、长用户名／ID、八种结果、START_ONLY／FINISH_ONLY、独立状态源失败、旧数据提示、游标过期、401／403 复查及迟到响应清理。沿用既有主题颜色和状态文字，不因通用 UI 检测规则改变批准的品牌。
