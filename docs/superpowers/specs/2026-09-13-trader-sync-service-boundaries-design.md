# Trader Sync 独立服务：职责、接口与事务设计

日期：2026-09-13。状态：第 1、2 板块已获用户确认，待实现；没有实施进程拆分。

## 目标与已确认范围

落实本轮第 1 项“服务边界”和第 2 项“接口与事务归属”。独立业务服务方向、API facade 职责、保留撤权同库原子事务，已在讨论中接受。用户进一步明确：本次采用单个活跃采集实例，允许重启中断，恢复后展示中断；不要求本次实现多副本高可用或不中断采集的滚动升级。

2026-09-13：用户审阅自审修订稿后确认上述两个板块，包括职责、内部调用与权限、事务归属及相关所有权/故障约束。后续设计沿用这些决定，无需重复确认。具体 proto 消息字段、构建运行入口、配置与部署注入、局部资源编排和实施验收步骤仍需细化；本次确认不表示这些后续部分已经设计完成或实施。

本设计受[服务开发规范](../../developer-guide/service-development-standards.md)约束。业务依据为[Trader Sync 产品需求](../../requirements/polymarket-copy-trading/README.md)；当前实现证据见[Activity Alerts 后端设计](../../design/trading/trader-sync-activity-alerts.md)。独立启动与局部清理的具体命令、容器和端口配置属于后续运行入口设计，本轮只确定其必须满足的依赖和生命周期契约。

保留现有业务规则：权限与所有者隔离、订阅名额、确认 token、request_id 幂等、revision 并发控制、完成基线后生效、故障期间不补查、撤权以持久发送许可为界、未知投递结果不自动重发。Copy Trading 和交易执行不在本设计内。

## 方案取舍

采用“独立 Trader Sync 业务进程 + 内部 gRPC + 受控同库事务适配器”。它把采集与业务生命周期从 API 移出，同时保留当前业务的原子性。代价是账户、Trader Sync、Notification 的部分存储契约仍需协调演进；不能宣称数据库层已完全解耦。

对比方案：

| 方案 | 影响 | 结论 |
| --- | --- | --- |
| 独立进程，业务调用走 gRPC，原子不变量保留窄事务适配器 | 可以独立运行；保留 schema 和小范围存储契约耦合 | 本次采用 |
| 独立进程、独立数据库，撤权和投递资格通过事件协调 | 必须重新设计撤权完成条件、失败恢复和发送许可竞争 | 本次不采用，不改变已确认业务语义 |
| 将订阅 API、Collector、Projector 再拆成多个进程 | 必须把同实例基线快照变成跨进程协议，并分别定义 worker 所有权 | 当前没有独立扩容或高可用需求支撑，不采用 |

## 1. 服务职责

| 所有者 | 负责内容 | 调用和依赖边界 |
| --- | --- | --- |
| API Server | 公共 HTTP/gRPC、登录/session、公开请求鉴权、协议转换、聚合和 facade；现有账户权限写入口 | 业务通过内部 gRPC 调用 Trader Sync；不得持有其 Service、Collector 或其他 runtime 对象 |
| Trader Sync 进程 | 目标解析、订阅增改和查询、基线登记与兑现、WSS 采集、原始事实持久化、确认与活动投影、目录刷新、业务状态查询 | 自有 PostgreSQL pool、HTTP/WSS 客户端、worker 生命周期和内部 gRPC listener |
| Notification 进程 | Telegram 绑定和 Bot updates、摘要冻结与首条协调、发送许可、调度、外发与结果恢复 | 自有 PostgreSQL pool；通过受控存储适配器读取活动资格并冻结摘要，不调用 Trader Sync runtime |
| 账户权限模块 | 权限的权威写入和撤权事务 | 本次仍由现有 API 进程承载；通过窄适配器在本地事务内完成相关资格撤销，不新增账户服务进程 |

Trader Sync 将整个订阅/基线/Collector/Projector 组合迁出。已有 `NewService` 要求订阅的 baseline registrar 与运行 Collector 是同一对象，独立进程内继续保持这个约束。API 不负责预先写入基线请求，也不代替 Trader Sync 查询业务表。

Notification 中的摘要冻结属于发送调度的一部分，保留其执行归属。共享的是无独立后台任务的存储适配器；业务 runtime、连接池对象和内存锁不跨进程共享。同一进程内的适配器可以借用该进程所有的 pool，不能关闭它。

### 独立运行与单实例所有权

- Trader Sync 的必要运行依赖是已经初始化相应 schema 的 PostgreSQL 和所声明的外部数据源。API、UI、Notification 不是其启动前提；站点 URL 是链接配置，不要求该站点当时可访问。
- API 启动时初始化 Trader Sync 客户端及自身必需配置，不等待 Trader Sync 在线才能对外服务。缺失自身必需客户端配置属于配置错误；已配置的下游暂时不可达只影响相关请求。
- Notification 停止时，Trader Sync 仍可形成活动及持久待发送记录；实际发送和摘要调度等待 Notification 恢复。Trader Sync 停止时，Notification 可按既有资格规则处理已保存工作。
- 启动顺序固定为：初始化自身配置/pool/listener（未就绪）→取得整组运行所有权并提交旧 epoch 的接替状态→将 session/token 注入 Collector 和运行期存储、启用事务校验→完成旧 epoch 的逐账户收尾与 pending 恢复→启动三个 worker 并确认初始化完成→开放业务就绪。恢复写入也必须先具备运行期校验，不能先开放 RPC 再等待 Collector 初始化 token。
- 第二个实例获取失败时保持未就绪并退出，不能短暂投影或刷新目录。Collector 使用入口已经取得的 session，不重复获取。数据库锁丢失或无法确认所有权时，立即关闭业务准入并取消整组 worker。
- 正常停止先关闭就绪及新请求准入，在同一个总停止预算内完成在途请求、取消并等待 worker 和持久写收尾，最后释放 session、pool 和网络客户端。`GracefulStop`、`Service.Close` 或 `Pool.Close` 不能被当作天然有界；预算耗尽时由进程入口/监管者强制终止该进程，不在旧 worker 仍运行时主动释放所有权来允许接替。数据库确认旧连接结束后才释放会话锁，不承诺崩溃后立即接替。
- 崩溃后由后继实例按既有 epoch 恢复规则记录中断；不承诺中断期间成交完整性。

现有 `Collector.Run` 内部获取并关闭 session 的职责一起移至 runtime owner；Collector 成为 session 借用者。不能只改为注入已取得的 session，却保留 Collector 在其他 worker 尚未退出时自行释放所有权的路径。

#### 整组写入的所有权校验

现有 Collector 的 session lock 和 `checkCollectorFence` 只保护其特定写入，不能直接证明 Projector、Directory 或订阅命令也不会由失去所有权的实例提交。整组单活需要以下新增协议，不把它写成现有能力：

1. 新增独立的 `trader_sync_runtime_control` 单行状态，保存运行实例 owner 与单调 generation。新实例取得现有 session advisory lock 后，在同一接替事务中先排他锁定 runtime 行，再更新 runtime generation 和既有 Collector control/epoch。该事务提交是接替生效点；逐账户恢复在提交后进行。
2. 每个 Trader Sync 自有写事务开始时，先以 `FOR SHARE` 锁定 runtime 行，验证实例 owner/generation，持有到提交或回滚；之后才取得账户、钱包、来源记录、Collector control 等业务锁。旧实例已经取得共享锁的事务允许先完成，新实例等待它们结束后才能提交接替；接替后旧 generation 的新事务一律拒绝。
3. 该校验属于 Trader Sync 运行期事务入口，覆盖确认 token、订阅命令、基线、raw intake、确认/投影证据、活动、metadata、目录及其收尾写入。必须覆盖 `Begin`、`BeginTx`、账户事务入口以及其他直接写路径；只包装 Directory 的事务接口不够。纯读取保留自身权限校验，不要求持有写入锁。
4. API 撤权和 Notification 摘要/发送适配器使用调用方事务，不取得 Trader Sync runtime 锁，不重新开始事务，也不重复获取账户锁。不能把运行期校验植入这些共享适配器，导致 Trader Sync 离线时撤权或发送被禁用。
5. Directory 的两次事务都要校验。现有第二次事务持有目录行锁调用外部 HTTP，并允许原事务有限收尾；这段已有预算同时约束 runtime 共享锁的持有时间。接替等待它结束，不能在发 HTTP 前释放校验锁，也不能为收尾另开一个绕过校验的事务。这里保留已有目录节流协议，不把 HTTP 与数据库误称为分布式原子提交。
6. 接替/释放所有权只持有 runtime→Collector control 的锁，提交后再逐账户清理。所有正常运行事务使用 runtime→既有业务锁的顺序；禁止账户锁取得后再反向申请 runtime 锁。关闭清理只能修改自身 generation，不能清除后继实例的状态。

这保证数据库接替生效后旧实例不能继续提交运行期业务写入；不能据此承诺瞬间撤回旧实例已经发出的外部只读 HTTP 请求。后台任务仍需正常传播取消并有界收尾。

#### 就绪与采集状态

标准 gRPC health 的业务服务项表示“可以接收并正确处理业务 RPC”：自身初始化完成、持有所有权、数据库可用、worker 已启动且未 fatal、未进入停止。它不把 WSS 已连接作为所有 RPC 的共同前提。

暂时 WSS 断开时保持服务可处理查询、暂停、取消等请求，Collector 按现有规则重连；创建/恢复可以提交 `pending_baseline`，只有基线实际完成才生效。管理员运行状态独立显示连接、中断、积压和证据缺失。所有权丢失、数据库不可用或 worker fatal 则停止业务准入并报告未就绪；公共 API 其他模块的健康不随之失效。

## 2. 内部 RPC 契约

公共契约继续由 `internal/server/tradersync/tradersync.proto` 管理，API 实现改为 facade。内部契约拟放在 `internal/tradersync/apiclient/trader_sync.proto`，使用独立的 `tradersync.internal.v1` 命名空间，不携带公共 HTTP 路由或 Swagger 配置。

内部协议由 Trader Sync 所有，定义传输消息及生成客户端，不导入 API Server 或其他服务的实现。API 负责公共消息与内部消息的映射；金额、时间、缺失证据、分页和状态等既有语义必须对应一致。公共页面不因本次部署边界变化而改变业务语义。

| 调用类别 | 内部 RPC | 权限语义 |
| --- | --- | --- |
| 目标与订阅 | ResolveTarget、CreateSubscription、ListSubscriptions、GetSubscription | 当前成员身份、Trader Sync grant、资源 owner；创建含确认 token 和名额检查 |
| 订阅命令 | PauseSubscription、ResumeSubscription、CancelSubscription、UpdateTargetNote | grant、owner、request_id、expected_revision |
| 活动与历史 | ListActivities、GetActivity、ListSubscriptionHistory | grant、owner 及现有分页/快照语义 |
| 摘要读取 | GetSummaryBatch、ListSummaryParts | grant、owner，读取持久事实 |
| 管理员 | ListSubscriptionSummaries、GetSubscriptionSummary、GetTraderSyncRuntimeStatus | 数据库中当前有效管理员身份，仅返回既有安全概要 |

每个业务请求包含明确的 `Actor`，带规范化的 `account_id` 和已认证账户的持久应用 realm；实际业务参数与 Actor 分开。API 保留 cookie 的入口 realm 校验，Bearer/API Key 不因此新增必须提供 realm header 的要求。Actor 不携带可直接当作授权结果使用的 grant 或管理员布尔值。资源 ID、过滤器、分页和排序放在请求字段中，RPC 使用动词开头的 PascalCase。

基线兑现、Collector 工作和撤权事务钩子不新增为远程 RPC。健康检查使用标准 gRPC health；业务运行状态仅由管理员 RPC 返回。

### 身份、内部认证与业务授权

采用“专用服务凭据 + 可信 Actor + 服务内查库授权”，沿用现有 Notification 内部 Bearer 的调用方认证模式，但 Trader Sync 使用独立凭据。

1. API 验证浏览器 session/API Key 等现有公开凭据及 realm，从已认证上下文构造 Actor；不接受客户端提交的 account_id 冒充 Actor。
2. API 使用专用内部凭据调用 Trader Sync。拟用 `ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN`；它与 cursor HMAC key 是不同用途的配置。单个凭据只代表明确授权的 API 调用方，不默认授权其他服务。
3. Trader Sync 先验证服务凭据和 Actor 格式，再检查请求方法与 realm、数据库账户角色是否匹配；用户的业务权限由存储层逐次查询当前权威状态。
4. 写请求的 grant、owner 和并发校验在对应账户事务中完成。读请求也沿用已有事务内授权，不信任 API 进程缓存的 grant。
5. 本次业务 RPC 都是用户或管理员委托请求。进程内部后台任务不伪造用户 Actor；以后新增系统 RPC 时，另行定义服务身份和允许动作，不能以任意 account_id 获得用户权限。

保持公开凭据与后台产品资格的区别：登录或 API Key 被停用后，API 按现有规则拒绝相应新请求；这不等于撤销 Trader Sync 产品 grant，不能额外停止已授权后台监控。当前 13 个会员方法均无新增的 interactive-only 限制，不能因引入 Actor 而禁止原本可用的 API Key。

没有采用“把公共 cookie/JWT 原样转发，让服务再接入完整公开登录体系”，因为那会把 realm、cookie 和公开凭据解析职责复制到业务服务。服务信任经过内部认证的 API 作身份断言，同时独立判断该身份现在是否拥有业务权限。

内部 listener 不作为公网用户入口。跨主机调用使用校验服务端证书的 TLS；本机开发允许显式 loopback 明文连接。内部 Bearer 不替代传输保护。具体证书与部署注入方式在后续运行配置设计中落定。

### 超时、错误与幂等

- 每次内部调用使用“上游剩余 deadline 与本地预算取较早值”，并传递取消。建议初始预算为读取/管理状态 5 秒、目标解析/写命令 15 秒；这是待实现和验证的配置默认值，不是已测得的性能承诺。
- 服务端无论请求是否携带 deadline，都执行“请求剩余预算与该类服务端上限取较早值”。下游 HTTP/RPC 不能重新获得超过父请求剩余时间的预算。
- API 不自动重试写命令。请求超时或连接中断可能发生在提交之后，不能向用户声称已回滚；重放必须使用相同 owner、操作、request_id 和载荷，保持现有幂等结果与 revision 语义。
- 保留既有 InvalidArgument、NotFound、AlreadyExists、Aborted、FailedPrecondition、ResourceExhausted、PermissionDenied 等业务状态与必要错误细节；连接不可用为 Unavailable，预算耗尽为 DeadlineExceeded。
- 内部 listener 对缺失/错误服务凭据、非法 Actor 返回带稳定内部原因的认证或契约错误。公共 facade 将这类 API→Trader Sync 的服务认证/调用契约故障转换为 Unavailable（HTTP 503），记录内部原因且不向用户暴露凭据；不能把它透传为用户的 401。用户自身公开凭据失效仍由 API 返回 Unauthenticated，合法 Actor 的业务权限不足仍保留原授权拒绝语义。
- Trader Sync worker fatal 导致自身未就绪/退出，API 不因此退出。状态查询不可达时显示明确的不可用状态，不以空活动、零积压或“健康”替代缺失证据。

## 3. 事务归属与允许的存储协作

三个进程连接同一 `athena` PostgreSQL，各自拥有独立 pool；共用一个权威 schema/迁移集。数据库故障是已知共同故障域。进程隔离不等于数据库隔离。

| 原子操作 | 事务所有者 | 同一次事务中的行为 |
| --- | --- | --- |
| 账户撤权 | API 内的账户权限存储 | 权限更新、关闭 Trader Sync 区间/基线/订阅、撤销 membership 和尚未获许可的 delivery |
| 创建/恢复订阅 | Trader Sync | grant/owner/名额或 revision 校验、订阅变化、同 Collector 基线登记、幂等结果 |
| 基线兑现 | Trader Sync Collector | 校验登记意图、权限、generation/epoch 和 Collector fence，写入有效区间 |
| 活动形成 | Trader Sync Projector | 保存 activity、冻结资格/membership，并创建普通通知的持久待发送记录 |
| 摘要冻结与首条协调 | Notification | 按现有发送调度协议冻结 batch/parts，完成对应首条许可协调 |
| 发送许可与结果 | Notification | 短事务持久化 attempt/许可；网络发送在事务之外；结果按 CAS 独立补记 |

账户撤权使用 Trader Sync 所有的窄 persistence adapter，只接收当前事务和必要的业务输入，不构造 `tradersync.Service`、不读取链节点配置、不启动后台工作、不拥有连接池。活动形成时借用 Notification 的入队适配器也是同一原则。

这个设计允许以下行为成立：Trader Sync 已停止时，管理员撤权仍能在账户数据库事务内完成；Notification 恢复后不能给已撤销资格重新发放发送许可。已经获得持久许可的 attempt 继续记录真实结果，撤权不等待外部网络回执，也不声称撤回已授权发送。

账户事务 gate 继续使用数据库级协调，在不同进程和连接池之间保持序列化。Trader Sync 自有写事务先取得运行期共享校验锁，再沿用现有账户、钱包、Collector control 等业务锁顺序；API/Notification 的外部事务不加入运行期锁。不能用进程内 mutex 代替。

禁止在本地事务中调用远程服务来声称跨服务原子提交，也禁止通过 RPC 传递 transaction、连接池或锁对象。共享适配器只能覆盖已列明的原子不变量，不能成为 API 任意读取/写入 Trader Sync 业务表的通道。

## 4. 预期源码边界

以下是预计位置，不代表已经新增：

- `internal/tradersync/`：领域逻辑与独立 runtime/server 组合；自有客户端、pool、单实例准入与 worker 生命周期。
- `internal/tradersync/apiclient/`：内部 proto、生成消息/客户端与专用服务认证。
- `internal/server/tradersync/`：公共 proto 与 facade，只依赖内部客户端和公共契约；移出当前持有领域 Service 的实现。
- `internal/server/trader_sync_runtime.go`：现有 API 内组合直接替换，不保留旧同进程路径或切换开关。
- `internal/tradersync/store/` 与 `internal/accountstate/store/`：保留并收窄撤权 adapter 契约，区分事务借用者和 pool 所有者。
- 权威迁移集及 Trader Sync sqlc 查询：增加 runtime control 状态和运行期事务入口校验；同步生成产物。共享适配器的外部事务保持原边界。
- `internal/notification/`：保留摘要和发送调度归属；只使用需要的受控存储适配器。
- 独立 `cmd/athena-trader-sync/`：作为后续构建入口设计的目标，不经由导入所有服务实现的聚合入口构建。

## 5. 后续实现必须提供的验证证据

1. 公共 facade 的 16 个方法正确映射内部 RPC，保持字段、缺失值、分页和业务错误语义；区分内部调用契约错误与公共用户认证错误。
2. 无/错误内部凭据、非法 Actor、跨 realm、成员访问管理员接口、跨 owner、授权撤销均被拒绝；客户端不能覆盖 API 注入的 Actor。内部 token 不匹配时会员及管理员会话保留、其他模块可用、Trader Sync 局部显示不可用；公开凭据失效仍按原规则退出。有效 API Key 继续可调用，登录/API Key 停用不意外撤销产品 grant。
3. 创建/恢复与撤权并发、活动形成与撤权并发、发送许可与撤权并发仍符合原子边界；内部 RPC 已提交但响应丢失时，UI 现有恢复操作按原 request_id 和载荷重放，不重复产生业务结果。保留用户主动重新解析目标并建立新意图的既有语义。
4. 不启动 API/Notification 仍能独立启动 Trader Sync，执行授权业务调用和采集；Notification 停止不阻止活动持久化。隔离测试使用真实账户/权限 fixture，不增加产品绕过认证的路径。
5. 第二个 Trader Sync 实例在任意 worker 工作前被拒绝；模拟旧实例暂停、所有权连接丢失、后继接替和旧实例恢复，验证旧 generation 对各类运行期写入均被拒绝。覆盖正在收尾的 Directory 事务、API 撤权和 Notification 许可并发，证明接替等待、锁顺序及外部事务独立性。
6. Trader Sync 离线时 API 其他模块仍可用，账户撤权仍可提交；恢复后中断可见且不补查遗漏成交。
7. 对受影响 proto、生成 Go 客户端、公共契约及真实消费者执行项目要求的同步与契约验证。具体命令、隔离运行资源与真实环境验收步骤写入后续实施计划。
8. 就绪前请求不能使用未初始化的 Collector token；仅 WSS 断开时读取/暂停/取消仍可用，创建/恢复保留 pending 语义；worker fatal 和所有权失效撤销准入。人为阻塞请求/worker，验证总停止预算和强制退出路径，不以提前释放所有权实现“有界关闭”。

## 当前源码核验入口

- [API 组合](../../../internal/server/athena-server.go)：安装账户撤权 hook、构造 runtime、注册公共服务。
- [现有 Trader Sync runtime](../../../internal/server/trader_sync_runtime.go)：借用账户 pool，构造 Collector/Projector/Directory。
- [领域 Service](../../../internal/tradersync/service.go)：同 Collector 校验、三个 worker 当前并发启动及资源关闭边界。
- [订阅事务](../../../internal/tradersync/subscriptions.go)与[Collector 所有权](../../../internal/tradersync/store/collector_session.go)：基线登记、session lock 与 epoch/fencing。
- [账户权限事务](../../../internal/accountstate/store/sql_store.go)与[撤权适配器](../../../internal/tradersync/store/revocation.go)：提交前同步撤权。
- [活动形成](../../../internal/tradersync/store/activities.go)、[摘要冻结](../../../internal/tradersync/store/summaries.go)与[发送许可](../../../internal/notification/store/attempts.go)：资格和持久投递边界。
- [Notification 内部认证](../../../internal/notification/apiclient/internal_auth.go)：当前已有的专用内部 Bearer 模式。

## 自审记录（2026-09-13）

本次对照实际源码复核职责、16 个 RPC、权限、事务、故障和生命周期，并修订设计。以下是文档审阅结果，不是新协议已通过运行测试的声明。

| 自审发现 | 修订结果 | 核验依据 |
| --- | --- | --- |
| 内部服务认证错误直接透传 401 会清除正常用户会话 | facade 转换为下游不可用 503；公共用户认证仍按原规则处理 | [会员全局错误处理](../../../ui/src/app/member/app.tsx)、[管理员全局错误处理](../../../ui/src/app/admin/app.tsx) |
| Collector 既有锁不能直接覆盖整组运行期写入 | 新增 runtime generation 校验、接替提交生效点及事务锁序；外部撤权/通知事务豁免 | Collector session、投影写入和 Directory 两事务源码 |
| 恢复、worker 启动、RPC 就绪和 session 释放顺序不够明确 | 先启用事务校验再恢复；worker 初始化后开放就绪；runtime 在所有借用者结束后释放 session | Collector.Run、RegisterTx、Service.Run/Close |
| “有界停止”和“健康”缺少可实施语义 | 总停止预算与强制退出；RPC 就绪与 WSS 连接状态分开 | 既有关闭等待、基线登记和重连行为 |

已排除两项不应扩大改造的疑点：当前 UI 的未知结果恢复已经保留 request_id 与载荷；现有存储适配器没有反向导入 Trader Sync runtime。登录/API Key 停用也不应被误改为后台产品撤权。这些原有规则纳入后续回归证据。

本次自审仅修订设计文档，随后第 1、2 项已获用户确认。独立入口、部署注入、局部资源编排及验收命令仍按已划定的后续工作设计；确认范围不代表整个拆分任务已经具备全部运行配置或已实施。
