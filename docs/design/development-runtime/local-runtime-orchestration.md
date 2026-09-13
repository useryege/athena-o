# 本地运行编排

> 设计状态：现有全栈编排已实现；Trader Sync 独立构建、本地运行与部署接入目标已确认待实现

## 范围

本地运行编排负责 `make run`、`make stop`、`make run-reset`、Foreman/Goreman 进程图、可复用 PostgreSQL/Redis/MinIO 容器，以及 API Server 与各业务服务的本地配置边界。Google OIDC 与 Phantom Solana 认证仍属于 API Server；运行层提供固定公开 origin、认证与 step-up 的 Redis 状态、持久账户/通知/Wallet/Worm Trading 数据、私有头像存储和两个 realm-selected disabled-auth 身份所需的 reset 边界。

Trader Sync 在现有 `athena-server` 和 `athena-notification` 两个进程内组合，不增加服务进程、独立 Notification 数据库或跨进程连接池对象。

> **服务开发规范差距（2026-09-13）：**上述组合和本文件列出的 `make run`、`make stop`、`make run-reset` 是当前实现事实与源码证据，不是新服务的目标运行模型。原“不增加服务进程”是既有实现决定，不能限制今后的服务边界改造。当前 [Procfile](../../../Procfile) 以 `ATHENA_BINARY_NAME` 复用 `go run ./cmd/main.go`；该入口在 [cmd/main.go](../../../cmd/main.go) 聚合导入全部命令实现，因此局部服务构建仍会耦合无关实现。当前 `make run` 还启动固定资源和整套进程图，`make stop`/`make run-reset` 分别面向整套清理或固定资源重置，不能当作服务的局部生命周期。按[服务开发规范 SDS-R3、SDS-R5](../../developer-guide/service-development-standards.md#sds-r3)，新增独立业务服务需要以目标服务和最小依赖正向选择的可验证构建、部署、启动、测试和停止入口，且局部编排只能回收其拥有资源；当前未实现 `run-service` 等局部命令，故不能作为现有命令列出。既有同库受控事务和连接池 owner 关系仍按 [SDS-R6](../../developer-guide/service-development-standards.md#sds-r6) 保持，改造不得跨 RPC 传递 transaction。

## 已确认的独立运行目标

Trader Sync 的[独立服务职责、接口与事务边界](../../superpowers/specs/2026-09-13-trader-sync-service-boundaries-design.md)及[独立构建与本地运行设计](../../superpowers/specs/2026-09-13-trader-sync-local-runtime-design.md)均已于 2026-09-13 获用户确认，待实现：

- 本机独立 Go 二进制配合最小 PostgreSQL 依赖；服务、API 和 Notification 使用独立入口，Trader Sync 镜像不经过 UI/聚合构建。
- 默认每个开发实例使用独立持久库；同实例的三个进程共享权威 schema，各自持有 pool。显式复用外部库时，借用方没有数据库清理权限，原 owner 的停库和故障仍会影响借用方。
- 配置使用统一的 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`，移除旧 API 命名；独立 schema 命令负责准备和只读校验，业务进程不隐式迁移。API 的 Trader Sync 客户端故障只使对应 facade 不可用。
- 局部编排按服务正向选择、按 checkout/实例核验资源归属；正常停止保留数据，显式重置只面向拥有的数据。旧全栈清理也须避免误停局部实例；停止预算、TLS 和部署迁移时序按已确认规格执行。

上述入口、变量更名和资源编排尚未实现。下面的命令与源码表继续描述当前行为；[内部字段契约](../../superpowers/specs/2026-09-13-trader-sync-grpc-contract-design.md)及[实施计划](../../superpowers/plans/2026-09-13-trader-sync-independent-grpc-service.md)已获用户确认，正在执行。已有运行证据不证明独立服务改造已完成。

## 源码入口

| 职责 | 源码 | 关键内容 |
| --- | --- | --- |
| 用户命令 | [Makefile](../../../Makefile) | `run`、`stop`、`run-reset` |
| 进程图 | [Procfile](../../../Procfile) | migration、API/UI 与业务服务 |
| 本地生命周期 | [hack/local-runtime.sh](../../../hack/local-runtime.sh) | 启动、停止、reset、exclude、端口与 coverage 清理 |
| 依赖容器 | [start-postgres-with-password.sh](../../../hack/start-postgres-with-password.sh)、[start-redis-with-password.sh](../../../hack/start-redis-with-password.sh)、[start-minio.sh](../../../hack/start-minio.sh) | 固定 volume 与初始化 fingerprint |
| 双前端入口 | [vite.config.ts](../../../ui/vite.config.ts)、[member HTML](../../../ui/src/app/index.html)、[admin HTML](../../../ui/src/app/admin/index.html) | `/`、`/admin`、部署 base 与 realm transport |
| realm 选择 | [application_realm.go](../../../internal/server/application_realm.go)、[athena-server.go](../../../internal/server/athena-server.go) | header/query 校验、gateway/native HTTP 身份选择 |
| 认证配置与暂态 | 根目录 `.env`、[googleoidc](../../../internal/googleoidc)、[phantomauth](../../../internal/phantomauth)、[authregistration](../../../internal/authregistration) | OAuth、SIWS、注册 ticket 与 scoped proof |
| 权威账户/通知迁移 | [000001_init.sql](../../../internal/accountstate/store/migrations/000001_init.sql) | UUID、权限、资料、API Key、Trader Sync 与全部 Notification 表 |
| Notification 运行 | [athena_notification.go](../../../cmd/athena-notification/commands/athena_notification.go)、[internal/notification](../../../internal/notification) | 同库独立 pool、内部三域、单 Bot/poller/sender、显式恢复 |
| Trader Sync 运行 | [config.go](../../../internal/tradersync/config.go)、[trader_sync_runtime.go](../../../internal/server/trader_sync_runtime.go)、[trader-sync-local.sh](../../../hack/trader-sync-local.sh) | HTTP/WSS、proxy、Collector/Projector/Directory、API 进程生命周期 |
| Wallet | [wallet migration](../../../internal/wallet/store/migrations/000001_init.sql)、[internal/wallet](../../../internal/wallet) | UUID-owned custody 与头像元数据 |
| Worm Trading | [athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go)、[internal/wormtrading](../../../internal/wormtrading)、[execution migration](../../../internal/wormtrading/store/migrations/000004_execution_runs.sql) | 独立数据库、凭据加密、HMAC/Web client、持久 live Run |
| 生产边界 | [docker-compose.prod.yml](../../../docker-compose.prod.yml)、[prod-remote-deploy.sh](../../../hack/prod-remote-deploy.sh) | env 注入、disabled-auth 拒绝、私有服务网络 |

## 架构与数据归属

`make run` 先启动依赖容器，再以前台 Procfile 启动进程组。PostgreSQL 存储 UUID 账户、不可变用户名、权限、profile/preferences、API Keys 及业务状态；Redis 存储撤销快照、5 分钟 OAuth/SIWS/scoped reauthentication/Run proof、独立 Wallet/Worm lease 和 15 分钟匿名注册 ticket；MinIO 存储私有账户与 Wallet 头像。Vite 端口 4000 代理 `/auth` 和 `/api`，本地 Google callback 与 SIWS domain/URI 固定为 `http://localhost:4000`，不从请求头推导。

API Server 与 Notification 都连接 `ATHENA_SERVER_POSTGRES_DSN` 指向的 `athena` 数据库，并使用唯一 accountstate migration 集。两个进程各自创建、持有和关闭自己的 `pgxpool.Pool`；“同库”不表示共享同一个内存 pool。Notification 的系统投递、Telegram Topic、账户 binding/attempt/delivery、reply outbox、polling offset 与 Trader Sync 数据都位于该数据库。Wallet 仍使用独立 `wallet` 数据库，Worm Trading 使用独立 `worm_trading` 数据库。

Goreman 监督 API Server、UI、Notification、Wallet、Profit Sharing、Token 与市场情报服务。主要独立进程如下：

| 进程 | 端口 | PostgreSQL 数据库 |
| --- | ---: | --- |
| `worm-markets` | 8084 | `worm_markets` |
| `notification` | 8086 | `athena`（进程自持 pool） |
| `wallet` | 8088 | `wallet` |
| `worm-trading` | 8090 | `worm_trading` |
| `market-radar` | 8092 | 无 |
| `sports-live` | 8094 | `sports_live` |
| `sports-history` | 8104 | `sports_history` |
| `managed-oo` | 8106 | `managed_oo` |
| `profit-sharing` | 8108 | `profit_sharing` |

Notification 一个进程承载 `SystemNotificationService`、`AccountNotificationService` 和 `NotificationRuntimeService`，拥有一个 Bot、一个长轮询 consumer 和一个公平 dispatcher。内部非 health RPC 要求同一 `ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN`；API Server 消费账户与 runtime 域，既有生产者只消费系统域。webhook 与 long poll 不并存。

API Server 内一个 process-scoped Trader Sync `Service` 运行 Collector、Projector 和 DirectoryRefresher，并复用已安装 access hook 的 store 与 API Server account pool。listener restart 不创建第二实例；后台 fatal 返回 CLI restart loop。取消时先 join 全部 borrower，再由 owner 关闭 service、HTTP/RPC transport 和 pool。

Wallet 非 health RPC 使用独立 `ATHENA_WALLET_INTERNAL_AUTH_TOKEN`。`ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN` 是 Wallet/Worm Trading 专属 capability Bearer，必须不同于 Wallet 通用 token；API Server 和无关进程明确 unset。Worm Trading 只在本地 loopback/Compose 私网监听，拥有 `worm_trading` 数据库和独立凭据加密 key，不读取 Wallet 私钥；只有它接收专用 Solana RPC、Worm HMAC/Web endpoint 与 signer capability。

disabled-auth 启动创建/复用 `local-user` 与 `local-admin` 两个完整 aggregate。每个请求用严格 `X-Athena-Application-Realm: member|admin` 选择；浏览器原生资源用一致的 `athenaRealm` query。缺失、重复、非法或 header/query 冲突都不认证。API Server 在非 loopback listener 上拒绝 disabled-auth，生产脚本与 Compose 另行固定拒绝。

## 本地生命周期

1. 正常认证首次使用前在 `.env` 配置 Google Web client、准确 callback、client secret、`ATHENA_ADMIN_GOOGLE_EMAIL` 与稳定 `ATHENA_JWT_SECRET`。Phantom 桌面登录使用注入 provider，不增加 App ID、secret、RPC 或 per-wallet 环境变量。
2. 当前 schema 与本地 fingerprint 不兼容时，操作员显式运行 `make run-reset`。它先停进程，再删除仓库拥有的 PostgreSQL/Redis/MinIO volume 和默认 scratch，不自动重启，也不访问 Telegram 或 Worm 网络。
3. `make run` 创建或复用固定 volume，运行唯一权威迁移并启动进程；会员入口是 `http://localhost:4000/`，管理员入口是 `http://localhost:4000/admin/`。`ATHENA_RUN_EXCLUDE` 可在 IDE 单独运行组件时过滤 Procfile。正常 run 不删除 volume 或暗中 reset。
4. 未知 provider 身份只取得 15 分钟 registration ticket，提交永久 username 后 PostgreSQL 才生成 UUID 并原子创建完整 aggregate。Google 与 Solana 身份不合并；同一 Google subject 可有独立 member/admin UUID。
5. member realm 始终创建普通 Pending 候选；admin realm 只在验证邮箱匹配 `ATHENA_ADMIN_GOOGLE_EMAIL` 时允许创建唯一管理员。管理员 aggregate 关闭所有会员 entitlement 与模块。
6. `make stop`、前台退出或 `Ctrl+C` 先通知 Goreman，按有界 grace period 升级清理仓库拥有的 process group/container/listener，保留 volume。它不会执行 `make run-reset`。
7. Redis 重启会丢失待处理 OAuth/SIWS/step-up/Run proof 和 lease，但不删除 Wallet/Worm/账户/通知持久数据。已持久 Run 授权仍需当前 Session 与 access revision 才能继续。
8. Worm Trading 启动时迁移并 ping `worm_trading`，核 Solana mainnet，第一轮 probe 成功前 health 为 `NOT_SERVING`。preview worker、凭据维护与 live-Run recovery 一起启动；Open/Finalize dispatch marker 和 isolation 跨正常 stop 保留，已标 dispatched 的 mutation 不重放。

生产中，不兼容的 current-state schema 通过全新持久部署替换；hot deploy 保留 PostgreSQL/MinIO volume，因此不是不兼容初始 schema 的升级机制。

## Trader Sync 与 Notification 启停顺序

API Server 先从环境加载并验证 Trader Sync HTTP/WSS、站点 URL 和游标 HMAC，再构造共享单实例。`ATHENA_TRADER_SYNC_HTTP_URL`、`ATHENA_TRADER_SYNC_WSS_URL`、`ATHENA_URL`、`ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY` 缺失或协议非法会在资源启动前失败。正常停止时先取消 process context、join Trader Sync Service，再关闭 transport 与 API Server pool；不要在 listener restart 中并行启动第二个 Collector。

本地 API Procfile 在 Goreman dotenv 后调用 `hack/trader-sync-local.sh`。只有 `ATHENA_TRADER_SYNC_PROXY_URL` **未定义**时，脚本才注入 WSL gateway `:10809`；变量已定义为空表示直连。Trader Sync 明确为 HTTP/WSS transport 设置 proxy，因此 dotenv 恢复的全局 proxy 不会静默改路。生产 Compose 不调用本地脚本。

开发默认以 Chainstack 作为单一 HTTP/WSS provider；切换 dRPC 时同时人工更新对应 HTTP/WSS 配置并重启 API Server。系统不双采、不自动 failover，也不把两个 provider 混为连续 epoch。切换、断线、进程重启和故障均建立新的实时边界，不补查遗漏；已可靠持久化且仍合格的候选可以继续处理。

Notification CLI 打开自己的 `athena` pool并运行同一权威迁移，随后先处理 sender 准入，再构造 Bot/summary：

1. `NewServer` 在 `Start` 前从自身 `SQLStore.BorrowPool()` 借同物理 pool 给 `ConfigureSummaries(pool, ATHENA_URL)`；adapter 不拥有或关闭 pool。
2. `Service.Start` 先为新 incarnation 取得独占 sender session/登记。存在未确认停止的旧 sender 时安全失败；不能用 lease 消失、端口空闲或 PID 变化代替停止证明。
3. sender 准入成功后才同步 Bot profile、启动 poller，再启动 account/system/reply/summary 的单 dispatcher。恢复屏障完成前不授权发送。
4. 正常 Stop 先 cancel，停止 poller，join worker/summary/结果补记；随后保存 graceful stop、恢复未决结果并关闭 sender session；最后由 CLI 唯一关闭 pool。

异常停止后，操作员须先从对应 supervisor/container/主机确认**已登记旧进程确实退出**，再运行：

```bash
athena-notification --recover-stopped-sender=<incarnation-UUID>
```

该命令本身是停止确认，只锁库恢复对应 incarnation 的 attempt 并退出；它在 Telegram client、summary、RPC/WSS 组合前执行，不发消息。恢复把无法确认的已消费许可记为 unknown，unknown、sent、failed、cancelled 不复活或自动重发。除可证明首次无实例/attempt 历史外，正常新启动在授权前等待完整 60 秒 monotonic 恢复屏障；未解除 Retry-After 可能更久。

## 持久状态与 reset 边界

本地 volume 为 `athena-local-postgres-data`、`athena-local-redis-data`、`athena-local-minio-data`。PostgreSQL fingerprint 包含 image、database/user/password 和有序初始化输入，不兼容状态不会静默复用。

`make run-reset` 删除本地所有 UUID、用户名、角色、grant、profile/preferences、API Keys、session/revocation、OAuth/SIWS/registration/step-up/Run proof、Profit Sharing 引用、Wallet custody、Worm connection/credential/combination/preview/Run/dispatch/isolation、Trader Sync 订阅/活动/观察/通知、系统 Topic/delivery、Telegram binding/attempt/reply/poll offset 与头像对象。它还删除默认 `/tmp/athena-local`、已知 coverage 目录和运行控制状态，但不删除这些精确默认之外的自定义临时路径。

reset 不调用 Worm 网络。若远端 credential 需要撤销，先显式 disconnect；若有 live/unknown Run，先在 Worm 侧核对和协调。直接 reset 会删除 ATHENA 的 ciphertext、request correlation 与 durable isolation，不可用来解决远端不确定操作。

## 配置

| 配置 | 本地行为 |
| --- | --- |
| `ATHENA_GOOGLE_OIDC_CLIENT_ID`、`...CLIENT_SECRET`、`...CLIENT_SECRET_FILE`、`...REDIRECT_URI` | Google Web client；direct secret 优先；redirect 必须精确匹配 `http://localhost:4000/auth/google/callback`。 |
| `ATHENA_ADMIN_GOOGLE_EMAIL` | admin realm 未知 Google 身份创建唯一管理员 persona 的准入邮箱，不限制同 subject 的 member persona。 |
| `ATHENA_JWT_SECRET` | 稳定本地 HS256 key；轮换使现有 cookie/API Key 失效。 |
| `ATHENA_SERVER_DISABLE_AUTH` | 仅 loopback；`true` 时使用 `local-user`/`local-admin`，仍走角色与模块鉴权。 |
| `ATHENA_SERVER_POSTGRES_DSN` | API Server 与 Notification 共用的 `athena` 数据库；两进程各自建 pool。 |
| `ATHENA_TRADER_SYNC_HTTP_URL`、`ATHENA_TRADER_SYNC_WSS_URL` | 单一链 provider 的 HTTP/WSS；开发先 Chainstack，dRPC 仅手动切换。 |
| `ATHENA_TRADER_SYNC_PROXY_URL` | Trader Sync HTTP/WSS 专用 proxy；unset 触发本地 WSL 默认，空串直连。 |
| `ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY` | 稳定签名游标 key。 |
| `ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES` | Projector 同时处理 source 上限，默认 100，必须为正整数。 |
| `ATHENA_URL` | 公开站点 URL；API 生成链接，Notification 在启动前配置 summary。 |
| `ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN` | Notification/API Server/系统生产者共享 Bearer，至少 32 个无空白字节。 |
| `ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN`、`...API_URL`、`...TIMEOUT_SECONDS` | 单 Bot 身份、API endpoint 与 transport timeout。 |
| `ATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID`、`...PROD_TELEGRAM_CHAT_ID` | 管理员系统通知群组；会员投递只用 `athena` 表中已绑定私聊。 |
| `ATHENA_NOTIFICATION_WORKER_CONCURRENCY` | 跨 chat 并发默认 12；Bot/私聊/群组预算与最多五次尝试是实现规则。 |
| `ATHENA_WALLET_ENCRYPTION_KEY`、`ATHENA_WALLET_INTERNAL_AUTH_TOKEN`、`ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN`、`ATHENA_WALLET_POSTGRES_DSN` | Wallet 加密、通用 Bearer、专用 signer capability 与独立数据库。 |
| `ATHENA_ACCOUNT_AVATAR_S3_*`、`ATHENA_ACCOUNT_AVATAR_MAX_BYTES` | 私有账户/Wallet 头像对象存储。 |
| `ATHENA_WORM_TRADING_LISTEN_ADDRESS`、`...PORT`、`...SERVER_ADDRESS` | 默认 loopback `127.0.0.1:8090`；Compose 私网监听/服务名。 |
| `ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN`、`...POSTGRES_DSN`、`...CREDENTIAL_ENCRYPTION_KEY` | Worm Trading 专用 Bearer、独立数据库与凭据加密。 |
| `ATHENA_WORM_TRADING_SOLANA_RPC_URL`、`...WORM_API_ATTEMPT_TIMEOUT`、`...POSITION_BUDGET`、`...POSITION_CONCURRENCY` | 专用 mainnet RPC、官方 HMAC 5 秒调用、20 秒页面预算、默认 4/最大 32 并发。 |

Telegram long-poll timeout 与有限退避是实现常量；`next_update_id` 位于 `athena` 通知表，没有环境 override。生产 secret reset 分别生成 Notification、Wallet、Wallet signer、Worm Trading Bearer 与 Worm encryption passphrase，并拒绝短值、空白或不应相同的凭据。

## 故障恢复与排查

- Notification 在数据库连接/迁移、内部 token、Bot profile 或 webhook 检查失败时不进入 `SERVING`。运行后暂时 poll 失败按有限后台退避，health 仍为 `SERVING`，runtime 显示 `poller_active=false`；binding/offset 事务失败不推进 `next_update_id`。
- Notification 数据库运行错误或 sender session 失锁会取消新授权、停止 poller、health 失败并让命令退出。先核 sender 登记与 recovery：unknown 不自动重发；缺 `started_at` 的 sent 仍是成功但耗时 unavailable；recovery remaining/elapsed 缺失与合法 `0` 不同。
- 摘要首条未开始时检查 `summary_head`、waiting membership、冻结/许可、恢复屏障、Bot/chat budget、Retry-After 与 account gate 指标。首条缺失不能通过改形成时间、重发 unknown 或跳过 60 秒恢复屏障修复。
- Trader Sync Collector 与资料目录独立：WSS/heartbeat/latest/finality 失败影响观察状态并建立新实时边界；Profile/Gamma/metadata unavailable 不得阻塞已经确认的成交形成，也不应单独关闭健康 Collector。页面如实展示资料 unavailable。
- Chainstack/dRPC 只人工切换。切换前记录旧 provider 与 collector epoch，停止 API 单实例，修改成对 HTTP/WSS 后重启；不把切换窗口或故障遗漏表述为已回补。
- Wallet/API token 不匹配时 health 仍可探测但业务 RPC unauthenticated；修正两端并重启，不 reset 数据。Notification、Worm Trading 的内部 token 同理。
- Redis 故障阻止新 OAuth/SIWS/registration/step-up/Run proof，但撤销快照初始化后的既有 session 和持久业务数据仍可使用。Google/JWKS 故障只阻止新 Google callback；Phantom 校验不依赖远端 IdP/RPC。
- Worm provider 暂时不可达时保持进程并每 30 秒重试；chain/mint/decimals/batch capability 明确不匹配进入 `configuration_error` 直到修配置重启。`CONNECT_OUTCOME_UNKNOWN`、Open/Finalize 不确定结果必须人工协调，禁止盲目重放。
- 只有 fingerprint/current-state schema 不兼容时才 stop 后显式 `make run-reset`。正常 run/stop 永不自动选择破坏性 reset。

## 可观测性

`make run` 前台输出全部服务日志；容器状态和单服务日志用于诊断依赖。Notification 标准 health 表示进程生命周期，runtime 单列 Bot、poller、最近 update、system/account 队列、sending/unknown、recovery 与不可达 binding，不暴露 token/Bearer。管理员 Service Status 以独立 10 秒可见 single-flight 读取 Services、Notification、Trader Sync；各来源失败保留自己的最后成功值。

Trader Sync runtime 单列 Collector connection/epoch/filter revision、raw 可观测边界、投影/资料/通知指标及各自 unit/window/opaque serviceEpoch。公开时间未知、资料 unavailable、无 ACK、clock anomaly 和等待成员不填 0 或改称通过。前台日志可显示有界错误类别，但不记录账户私密正文、请求参数、内部 query、credential 或 digest。

Worm Trading 只有专用 Solana probe 成功后 gRPC health 才为 `SERVING`；后续暂时 provider 失败可显示 degraded。公开状态只暴露 credential store readiness 与脱敏 reachability/error，不暴露 URL、HMAC、signer token、Web JWT、原始/签名交易或 signature。

## 维护检查

- [ ] `make run`/`stop`/`run-reset`、固定 volume 和清理边界与脚本一致。
- [ ] OIDC、SIWS、realm 注册、disabled-auth 与生产拒绝边界保持当前行为。
- [ ] `athena` 同库、API/Notification 两个独立 pool、唯一迁移和十模块保持一致。
- [ ] Notification 单 Bot/poller/dispatcher、sender 停止证明、恢复命令和先恢复后启动顺序保持一致。
- [ ] Trader Sync 单实例、Chainstack/dRPC 手动切换、proxy 空值语义、新实时边界与不补遗漏保持一致。
- [ ] Wallet/Worm 数据库、lease、proof、signer capability、停止/恢复与 reset 警告保持一致。
- [ ] 依赖所有权、失败隔离、可观测字段和安全日志保持一致。
- [ ] 源码链接和[设计索引](../README.md)保持正确。
