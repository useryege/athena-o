# 关键操作的采集与结果契约

> 详细设计已于 2026-09-18 整体审阅通过，按用户要求暂不实施；不是已实现功能。入口：[总体设计](../2026-09-18-key-operation-logs-design.md)。

## 1. 可执行范围的定义

[事件清单](event-catalog.md)列出每种动作编码、实际入口、采集符号、对象及允许的详情字段；[机器清单](event-catalog.json)保存同一数据，供逐入口静态核对。清单内的业务入口是否允许会员、管理员或 API Key 调用，继续由原鉴权决定，日志不扩大权限。

不是根据 POST 或方法名自动判断是否属于关键操作。当前 `ResolveTarget` 是 POST 查询；Worm 的 `heartbeat` 和 `execute-next` 属于自动执行协调。它们显式排除。普通 Get/List、bootstrap、健康和日志查询也排除。新增业务入口必须在采集清单或排除清单中有明确归类。

## 2. 事件与一次用户操作

一次实际请求中的每项业务动作有独立 `operationId`（服务端 UUIDv4）。根请求生成 `requestId`；同一请求中注册和随后登录使用不同 operationId、相同 requestId，并设置 parentOperationId。业务幂等 commandId／requestId 单独保存为 businessRequestId，不能拿客户端 ID 覆盖或去重其他操作。

每个 operationId 最多接受一份 START 和一份 FINISH：

| 字段 | 类型与约束 |
| --- | --- |
| schemaVersion | 整数，当前只接受 1；不实现兼容分支 |
| eventId / operationId / requestId | 非零规范 UUID；eventId 在同一次事件投递重试中保持不变 |
| parentOperationId | 可空 UUID，只用于同一业务动作链 |
| phase | START 或 FINISH |
| producerId | API 进程每次启动生成的 UUID，不来自请求 |
| startedAt / occurredAt | UTC RFC3339Nano；startedAt 在同一操作内固定；FINISH occurredAt 是结果观察时间 |
| actor | accountId、usernameSnapshot、role、realm、credentialKind、identityVerified、identitySnapshotComplete、provider |
| actionCode / moduleCode | 事件目录中的固定编码 |
| resources | 主对象及相关对象引用，最多 100 个；类型来自目录，ID 为规范字符串 |
| businessRequestId | 可空字符串，最多 128 字节；只接收原业务已验证的幂等标识 |
| outcome / observation | 见结果表；START 的 outcome 为 UNKNOWN、observation 为 START_ONLY |
| businessState | 可空、最多 64 字节的既有业务状态编码 |
| effect | 已确认效果编码数组，最多 32 项，每项最多 64 字节；没有证据时为空，不以协议成功推导 |
| durationMs | FINISH 中非负 int64，使用单调时钟测量，不用墙钟相减；API JSON 返回十进制字符串 |
| grpcCode / httpStatus / reasonCode | 真实观察到的协议结果；不存在就留空，不从另一协议猜测 |
| details | 目录允许的类型化字段，最多 16 KiB；字符串值最多 256 字节；不是原始请求／响应 JSON |
| resourceCount / resourcesComplete | 完整总数（可空）及引用是否完整；省略对象时必须如实标记 |

整个事件编码上限 32 KiB。对超长诊断或对象列表，先按上述规则保留计数和截断标志；仍不合法则放弃该事件并计采集异常，不能因日志错误拒绝业务。所有标识和数字在前端保持字符串，不经 JavaScript number 损失精度。catalog 内的限长值截断使用完整 UTF-8 边界。

FINISH 必须自包含完整身份、动作、开始时间、对象和结果，即使 START 未落盘也可以单独展示。START_ONLY 记录显示“结果待记录”，解释可能仍在执行，也可能未能补记；不根据等待时长猜测业务失败。

## 3. 身份与内容来源

- 登录会话和 API Key 使用 [AuthenticatedCredential](../../../../internal/accountcredentials/types.go)，再从现有凭据投影读取用户名和持久角色。`LOGIN_SESSION` 只说明凭据类型，不声称一定来自浏览器。
- actor.role 为 MEMBER、ADMINISTRATOR、UNKNOWN；realm 为 MEMBER、ADMIN、UNKNOWN；credentialKind 为 LOGIN_SESSION、API_KEY、DEVELOPMENT、UNAUTHENTICATED。字段分别表达事实，不互相推导。
- accountId 只有在身份验证后才能填写。无法取得用户名快照时仍保留已验证 UUID，usernameSnapshot 留空、identitySnapshotComplete=false；日志不得为补名字阻止业务。
- accountId 为 null 的失败记录显示“身份未确认”。provider 表示认证方式，不保存外部身份 subject、邮箱输入、Cookie 或签名。验证通过、但尚无 ATHENA 账户时仍不能伪造 accountId。
- 角色和用户名是操作当时的快照；不得在查询时 JOIN 当前账户覆写历史。日志行不对当前账户设置级联删除外键。
- 只允许目录字段。账户资料／备注记录字段名、长度变化和版本，不记录自由文本内容；账户权限可记录受版本保护的前后枚举值。若没有同一 CAS 版本的前值，beforeAvailable=false，只显示已确认的新值，不额外读取一个可能已经变化的“旧值”。
- 私钥、API Key bearer、登录令牌、签名、OAuth code、确认令牌、协调令牌和完整业务消息不进入事件。API Key 的展示 ID 与 bearer 区分；源操作创建的展示 ID 可记录。
- 第一版不采集 IP、User-Agent、页面地址或搜索内容。HTTP route 只用匹配后的模板，不复制含 query 的 URL。

## 4. gRPC 采集位置

在 [unaryAuthInterceptor](../../../../internal/server/authz.go)使用一个请求级 Recorder，贯穿已存在的 authenticate、authorize、module admission 和 handler 调用，保持原鉴权顺序和返回错误。

1. 从固定方法目录创建 Recorder，只拷贝类型化且校验过的对象标识；调用既有授权逻辑。
2. `authorizeGRPC` 即使拒绝权限也会返回 authCtx。仅当其中确有可信凭据时绑定账户；否则记为未确认身份。拒绝路径可以只写自包含 FINISH，不能为追求 START 先把未验证输入当作用户。
3. 授权通过后，在 module admission 和业务调用前尝试写 START。日志超时不跳过或放宽后续的权限、Origin、版本及授权期限检查。
4. facade 在已有数据库／远端调用提交位置向 Recorder 标注观察事实；最终由请求级 defer 写唯一 FINISH。拦截器负责兜底，但不能仅凭 `err == nil` 判定异步业务最终成功。
5. 现有 recovery 继续处理 panic；Recorder 在独立 recover 保护中完成 UNKNOWN 结果。日志自身 panic 只报告采集错误，不替代原 handler 的 panic、响应或业务结果。

HTTP→grpc-gateway→gRPC 和 gRPC-web 共用此采集位置，不再套第二层业务记录 HTTP wrapper。网关在 proto 解码之前拒绝的请求没有可执行业务动作，不进入关键操作历史；此边界在覆盖清单中明确记录。

## 5. 原生 HTTP、登录与注册

原生 HTTP 在实际业务 handler 中注入同一窄 Recorder 接口，遵循原解码大小、Content-Type、Origin、身份与权限校验。不得通过读取完整 ResponseWriter 内容来拼日志。协议状态可由轻量 wrapper 观察，业务事实必须由 handler 显式标记。

- **普通登录**：Google callback 完成目的分派后进入登录 Recorder；Phantom Verify 对应登录 Recorder。普通登录跳转／challenge 成功不记录，初始化失败作为登录尝试的失败记录。验证身份后发现需要注册，记 ACTION_REQUIRED／REGISTRATION_REQUIRED。创建会话且 SetAthenaSessionCookie 成功，记 SUCCEEDED／SESSION_ISSUED；这不证明客户端已经接收 Cookie。
- **注册**：POST `/auth/registration` 是注册动作。`CredentialManager.RegisterExternalAccount` 的 store 已提交后即标注 ACCOUNT_CREATED 或 EXISTING_ACCOUNT_RESOLVED；该标注要放在持久提交后、publish 之前，不能丢失 commit 成功而 publish 失败的事实。后续 access publish 失败是 PARTIAL；进入 CreateExternalLogin 时创建独立的登录子动作。注册日志保留账户建立事实，不能因随后登录失败反写成“没有注册”。DELETE 是取消注册动作，GET 和用户名可用性检查不记。
- **退出**：只按已解析、已核验 realm 的会话绑定用户。Cookie 清除和服务端 revoke 分别标注；revoke 失败保留 PARTIAL／SESSION_REVOCATION_UNCONFIRMED，303 不等同完整退出。无有效会话的重复退出可记 SUCCEEDED／NO_ACTIVE_SESSION，身份留空。
- **敏感授权**：Google callback 按既有 ownsCallback 分派至对应 reauthentication／authorization 流，Phantom 使用各目的 Verify。每次完成验证只记录目的动作一次。账户绑定、lease/proof 的持久结果和失败由对应处理器标注；不能再记成一次普通登录。初始化成功及 challenge 生成不记，初始化失败可记同一目的的未完成尝试。
- **开发身份**：记录真实 development account 和 DEVELOPMENT 来源，仍经过现有 loopback 限制；不以日志功能开启 disabled-auth。
- **HTTP 写回失败**：已确认的持久业务事实保持原值，额外记录 responseWriteFailed=true；不能因客户端断线把已提交的写操作改成未执行。

Google 取消授权使用 CANCELLED，不是身份验证失败；不把未完成挑战／注册 ticket 本身保存为日志关联凭据。流程关联只使用新生成的不具授权能力的 operationId／requestId。

## 6. 结果判定表

| outcome | 判定事实 | 示例 |
| --- | --- | --- |
| SUCCEEDED | 同步请求承诺的业务结果已确认 | 修改资料、取消订阅、保存组合；取消操作本身成功仍是 SUCCEEDED |
| ACCEPTED | 命令／异步工作已可靠受理 | 测试通知入队、网关探测创建、Worm 执行／Cash Out 受理；不声称发送或交易最终成功 |
| FAILED | 明确校验／业务失败，确认该动作未完成 | 无效参数、版本冲突、业务明确返回失败 |
| DENIED | 身份／权限／Origin／再认证拒绝 | 普通用户调用管理员操作、模块关闭；module admission 的业务拒绝与依赖未知分别处理 |
| UNKNOWN | 没收到结果，或副作用可能已发生但无法核实 | 远端调用超时、断线、提交后返回内容损坏、panic |
| PARTIAL | 确知部分效果已产生，但未完整达到动作目标 | 移除部分 Worm 钱包后后续添加失败；账户已建立但发布访问状态失败 |
| ACTION_REQUIRED | 当前阶段完成，还需要用户完成下一步 | 已验证身份，需要提交注册 |
| CANCELLED | 用户取消当前登录／验证流程 | Google 明确返回 access_denied；不用于普通业务取消命令的成功结果 |

实现采用“默认 UNKNOWN，证据足够再标注”的合并规则。`dispatched=false` 的确定校验错误可记 FAILED／DENIED；已向写服务发出请求的 Unavailable、DeadlineExceeded、Canceled、Internal 默认 UNKNOWN。稳定业务拒绝码只有在原接口保证没有副作用时才能映射 FAILED／DENIED。

针对多步动作，以显式 effect 集合判断 PARTIAL；HTTP 200/202 和 gRPC OK 本身不覆盖业务失败状态。START 和 FINISH 中 action、producer、startedAt、原已验证 accountId 必须一致；未确认身份只允许在可信 FINISH 中补充，绝不允许从一个已验证账户换成另一个。

身份一致性按阶段语义比较，不按到达顺序判断：未确认身份的 START 与可信 FINISH 兼容，即使 FINISH 先入箱也一样。资源引用以 FINISH 为准；主对象的选择由动作目录约定，targetAccountId 仅从账户管理动作已规范化的目标账户取值，不从 actor 自动复制。

## 7. 可靠性与采集开销

每个阶段最多两次数据库提交尝试，共享 200 ms 总预算；单次最多 100 ms，重试不重放业务动作。一次动作两阶段合计最多增加 400 ms 采集等待；同一 HTTP 请求含注册及登录子动作时，所有 Recorder 另共享 400 ms 请求级总预算，不能按子动作数量累加等待。结果采集使用解除原请求取消但受独立超时约束的 context。日志专用连接池最大 4 个连接，每个 API 进程最多 16 个并发记录任务；无槽位立即报告 capacity_exceeded，不排无界队列。

每次都使用相同 eventId 和 `(operationId, phase)` 幂等约束。提交回执丢失不等于记录丢失：未收到确认的次数命名为 unconfirmedEvents；相同 payload 重试确认存在即可成功，不同 payload 冲突为采集错误。

计数按唯一事件的一轮有界采集计算，不按 SQL 尝试次数计算。attemptedEvents 在进入采集时加一；轮次结束后只落入 confirmedEvents、unconfirmedEvents、invalidEvents 或 capacityRejectedEvents 中一类。第一次回执超时但第二次确认成功，只计 confirmedEvents；预算耗尽仍不知是否提交才计 unconfirmedEvents。事件冲突归 invalidEvents 并保留原因。四类终态计数加上当前进行中事件数等于 attemptedEvents；状态快照须原子读取这些计数。以后即使消费服务发现一条曾未确认的事件，也不回写生产者当时的历史观测，页面不能将 unconfirmedEvents 显示为确定丢失数量。

生产者每 5 秒以有界请求发布本进程累计采集状态；这是采集适配器的状态上报，不在 API 中运行日志消费服务。它失败时保留本进程内存计数，下一次成功发布绝对计数；停止后计数可能不完整，状态页明确标示未知历史。长期保存且查询恢复的保证只覆盖已经持久入箱的事件。
