# 系统通知运营

> 设计状态：已实现；发送恢复与新调度器的后续范围见文末。

## 范围

系统通知运营负责发送到配置的测试/生产 Telegram 群组及论坛 Topic 的运营通知，包括内部生产者契约、持久 Topic 与投递、公平调度、管理员列表/详情/测试操作和共享运行视图。

Market Radar、Sports Live、Managed OO、Worm Markets 自行决定告警条件并保留来源侧状态；Notification 接受请求后负责投递，不选择普通账户，也不向会员暴露系统记录。私聊见[账户 Telegram 通知](account-telegram-notifications.md)。

## 源码入口

| 职责 | 源码 | 关键符号 |
| --- | --- | --- |
| 内部系统与运行契约 | [internal/notification/notification.proto](../../../internal/notification/notification.proto) | `SystemNotificationService`, `NotificationRuntimeService` |
| 系统应用逻辑 | [internal/notification/service.go](../../../internal/notification/service.go) | `SendSystemNotification`, `ListSystemNotificationDeliveries`, `GetSystemNotificationDelivery` |
| 系统持久存储 | [internal/notification/store/system_notifications.go](../../../internal/notification/store/system_notifications.go), [internal/notification/store/queries/system_notifications.sql](../../../internal/notification/store/queries/system_notifications.sql), [internal/accountstate/store/migrations/000001_init.sql](../../../internal/accountstate/store/migrations/000001_init.sql) | `EnsureSystemNotificationTopic`, 系统 Topic 与投递队列操作 |
| 公平投递与 Telegram 网关 | [internal/notification/worker.go](../../../internal/notification/worker.go), [internal/notification/sender.go](../../../internal/notification/sender.go) | `claimFairNotificationBatch`, `processClaimedSystemNotification`, `TelegramSender` |
| 公开管理员入口 | [internal/server/notification/notification.proto](../../../internal/server/notification/notification.proto), [internal/server/notification/notification.go](../../../internal/server/notification/notification.go) | 管理员列表、详情、测试及运行 HTTP 路由 |
| 管理员鉴权 | [internal/server/authz.go](../../../internal/server/authz.go) | `administratorGRPCMethods` 通知规则 |
| 管理员界面 | [ui/src/app/admin/pages/system-notifications.tsx](../../../ui/src/app/admin/pages/system-notifications.tsx), [ui/src/app/admin/pages/system-notification-detail.tsx](../../../ui/src/app/admin/pages/system-notification-detail.tsx), [ui/src/app/admin/pages/service-status.tsx](../../../ui/src/app/admin/pages/service-status.tsx), [ui/src/app/admin/notification-service.ts](../../../ui/src/app/admin/notification-service.ts) | 列表、筛选、详情、测试页面及 Notification 运行面板 |
| 内部鉴权客户端 | [internal/notification/apiclient/apiclient.go](../../../internal/notification/apiclient/apiclient.go), [internal/notification/apiclient/internal_auth.go](../../../internal/notification/apiclient/internal_auth.go) | `Clientset.System`, `Clientset.Runtime`, `NewNotificationClientset` |
| 当前告警生产者 | [internal/marketradar/mover_alerts.go](../../../internal/marketradar/mover_alerts.go), [internal/sportslive/price_alerts.go](../../../internal/sportslive/price_alerts.go), [internal/sportslive/score_alerts.go](../../../internal/sportslive/score_alerts.go), [internal/managedoo/proposed_alerts.go](../../../internal/managedoo/proposed_alerts.go), [internal/managedoo/disputed_alerts.go](../../../internal/managedoo/disputed_alerts.go), [internal/wormmarkets/notifications.go](../../../internal/wormmarkets/notifications.go) | `SendSystemNotification` 调用方 |
| 进程与部署装配 | [cmd/athena-notification/commands/athena_notification.go](../../../cmd/athena-notification/commands/athena_notification.go), [Procfile](../../../Procfile), [docker-compose.prod.yml](../../../docker-compose.prod.yml) | 单进程、单 Bot、chat ID 与共享内部凭据 |
| 持久发送许可 | [attempts.go](../../../internal/notification/store/attempts.go)、[delivery/types.go](../../../internal/notification/delivery/types.go)、[delivery_attempts.sql](../../../internal/notification/store/queries/delivery_attempts.sql) | `Authorize`、`RecordStarted`、`RecordOutcome`、`NextState` |
| HTTP 起点与结构化结果 | [send_transport.go](../../../util/telegram/send_transport.go) | `sendTransport`、`SendError` |

## 架构与权限

系统与账户通知是同一 `athena-notification` 部署内的逻辑域。`SystemNotificationService` 只管理系统 send/list/get；`AccountNotificationService` 管理私聊收件人；`NotificationRuntimeService` 报告共享进程状态。三者共用数据库、Bot 身份、发送器、公平 worker 和当前发送间隔。

生产者通过 `Clientset.System()` 附带内部 Bearer。API Server 是唯一公开代理：管理员能列表、检查详情、入队测试和读取运行状态；会员会话与 API Key 均不能。运行视图通过独立管理员入口提供。

## 运行流程

1. 进程校验内部 Bearer、数据库、Bot token、test/prod chat ID 及 Telegram 客户端，同步 Bot 资料，检查长轮询兼容性，启动 poller 与 worker 后才报告 SERVING。
2. 四个现有生产者仅在各自通知功能开启时创建鉴权客户端，调用 `SendSystemNotification` 提交来源、严重程度、标题/正文/链接、逻辑 chat 和 Topic 标签。
3. 服务校验请求和渲染后的 Telegram 长度。`EnsureSystemNotificationTopic` 先读取持久 `(telegram_chat,label)`；不存在时取得 PostgreSQL advisory lock，调用 Telegram 创建 Topic，再保存 message-thread ID。
4. 接受请求后插入引用 Topic 的 pending 投递并返回 ID。消息发送只在 worker 内进行；Topic 创建仍可能在入队前调用 Telegram。
5. 每轮按批量分配系统/账户队列、交替优先级、交错处理。所有消息使用共享 `TelegramSender` 并遵循发送间隔；领取不增加 attempt 数。
6. `Authorize` 锁定系统投递及持久路由，计算所有正文/路由输入的 digest；pending、到期且少于五次才插入 attempt 和改为 sending。事务提交后解析配置群组 ID，要求持久 Topic thread ID，发送转义后的 HTML。实际 RoundTrip 起点通过非阻塞握手保存，结果另开事务以 attempt/status CAS 落库。
7. `GET /api/v1/admin/system-notification-deliveries` 分页筛选系统记录；详情只接受正数 ID。测试入口始终向配置测试 chat 与提交 Topic 标签入队 `admin-ui` 来源的信息通知。
8. `/admin/notifications` 提供列表、筛选、移动端紧凑卡片、详情和单飞测试弹窗。Service Status 在进入、手动刷新及每十秒读取 `/api/v1/admin/notification-runtime/status`。

## 状态与数据

- `system_notification_topics` 以 `(telegram_chat,label)` 为键，保存正数 `message_thread_id`；逻辑 chat 仅为 test/prod。
- `system_notification_deliveries` 保存来源、严重程度、标题/正文/链接、channel、逻辑 chat、Topic、状态、provider 结果、尝试数、下一尝试时间与领取信息；Topic 为外键，`current_attempt_id` 指向当前许可。
- 状态包括 pending/sending/sent/failed/unknown/cancelled；pending 且 attempts>0 单独计入 retry。终态不自动重发。
- `notification_delivery_attempts` 保存 work kind/ID、sender incarnation、不可变 digest、授权/实际起点/结果时间、message ID、outcome/code/retry-after。系统 attempt 的 owner 为空；许可事务按 kind 核验实际投递关系。

`SystemNotificationDeliveryItem` 与 `SystemNotificationDeliveryDetail` 是内部和公开契约共享的投影，仅包含系统通知；不包含账户投递或私聊标识。除原字段外，投影保留 authorizedAt、startedAt、resultAt；缺失起点为空，不当作发送成功时间。管理员能筛选 sending/unknown/cancelled，unknown 使用明确警示和“不会自动重发”说明，运行页独立展示 sending/unknown 数量。

## 配置与部署

| 配置 | 行为 |
| --- | --- |
| `ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN` | Notification、API Server 与四个现有生产者共享的必需 Bearer。 |
| `ATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID` | `TELEGRAM_CHAT_TEST` 对应的具体群组。 |
| `ATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID` | `TELEGRAM_CHAT_PROD` 对应的具体群组。 |
| `ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN`、`..._API_URL`、`..._TIMEOUT_SECONDS` | Bot 身份、API 地址及适配器超时；worker 消息另有五秒 context 截止，长轮询使用独立客户端。 |
| `ATHENA_SERVER_POSTGRES_DSN` | 两个通知域共用的 Athena 数据库及唯一权威迁移。 |
| `ATHENA_NOTIFICATION_WORKER_SEND_INTERVAL`、`..._POLL_INTERVAL`、`..._BATCH_SIZE`、`..._LOCK_TIMEOUT` | 当前共享发送间隔、轮询、公平批量及未许可领取锁恢复。总发送尝试上限固定五次。 |
| 各生产者 `..._NOTIFICATION_ENABLED` 与 `..._NOTIFICATION_SERVER_ADDRESS` | 开启通知及选择内部 gRPC 目标。 |

生产 Compose 清除无关服务的内部凭据，仅注入 Notification、API Server、Market Radar、Sports Live、Managed OO、Worm Markets。非 Notification 应用容器不接收 Bot token 和具体群组 ID；生产者及 API Server 只使用鉴权 gRPC 地址。生产秘密重置独立轮换通知内部凭据，不与 Wallet、Wallet signer、Worm Trading 凭据共用。

## 不变量

- 系统操作不接受或推断普通账户 UUID，不调用 AccountNotificationService。
- 会员与 API Key 不能读取系统记录、详情、运行状态或入队测试。
- 入队前逻辑 chat 与 Topic 必须解析为持久正数 thread ID。
- 系统与账户交错领取，每轮再追加至多一条 Bot 绑定回复；三者共享 Bot、持久 attempt 和现有串行发送间隔，回复不在 poller 内直发。reply 有独立 outbox，不写入系统通知记录。
- Provider 错误不修改 Notification 数据库中的生产者告警状态；来源侧接受边界由各服务自行负责。
- 当前模型不含旧通用 topic/delivery 表和 Notifications 账户模块，也不增加历史兼容路径。
- 只有已提交许可才可发送；结果更新绑定当前 attempt 与 sending，sent/failed/unknown/cancelled 不可复活。

## 故障与恢复

无效 Bot、chat、数据库或内部凭据阻止构造/启动；配置 webhook 也阻止进入 SERVING。语法合法但不匹配的生产者凭据得到 Unauthenticated，不插入系统记录。

仅 pending 的领取锁可在超时后重领。明确 Telegram 429 或服务端暂时拒绝为 retryable，且尝试数少于五才回 pending；延迟取 1/2/4/8 秒退避与 Retry-After 的最大值。其他明确拒绝为 failed。HTTP 起点之后的断连、超时、取消、读取/解析失败归 unknown，不重发；发送禁止跟随重定向，SDK 的单次 POST 不提供可回卷 body 或幂等重放头。

成功消息后的数据库故障只重试结果 CAS，不再调用 Telegram。提交确认丢失时先按 attempt UUID 查结果；确认不了则保留消耗后的许可。成功回执不依赖 started 已写入；缺失起点如实保留。终态错误保存稳定类别，不存原始 provider 描述、响应体、URL 或 Bot 凭据。

Topic 创建在 Notification 请求之间串行，但 Telegram 创建与数据库插入跨外部事务边界。创建后插入/提交失败会留下未持久化映射；之后请求可能重新创建 Topic。消息许可流程不改变这一独立 Topic 风险。

## 可观测性与后续范围

gRPC health 报告生命周期就绪。共享运行 RPC 报告 Bot 可用性/ID/名称、poller 状态、最近成功轮询/处理时间、系统/账户 pending/retry/failed/sending/unknown 数量和不可达绑定数；公开投影仅供管理员 Service Status。

系统记录展示来源、严重程度、逻辑 chat、Topic、channel、结果、message/error、创建/授权/实际起点/结果/发送时间。日志使用投递/attempt ID 和逻辑 chat，不记录内部 Bearer 或 Bot token。

当前 worker 保留交错串行调度，账户/系统批量之外每轮追加至多一条 reply。Bot update 的绑定、消费记录、offset 和回复在同一数据库事务提交；回复以 `work_kind=reply` 复用许可/结果路径和五次上限，unknown 不重发。崩溃后遗留 sending 不会被超时领取；确认旧 sender 已停止后终结未知结果、跨 chat 并发与统一配额属于 [Trader Sync 后续实现](../trading/trader-sync-activity-alerts.md)，尚未实现。不能在启动时按不同 incarnation 批量终结仍存活 sender 的任务。

## 维护检查

- 系统、账户和运行契约在单进程内保持分离。
- 生产者只调用鉴权 SendSystemNotification，保留各自来源接受语义。
- Topic 串行化、持久许可、公平领取、实际 HTTP 起点、结果 CAS 与有限重试保持一致。
- 管理员列表、筛选、详情、测试和运行面板匹配公开契约与鉴权表。
- 系统记录与运营代码不进入会员 bundle 或 API。
- 配置注入、凭据轮换、health、运行字段与安全日志同步维护。
- 源码链接和[设计索引](../README.md)保持正确。
