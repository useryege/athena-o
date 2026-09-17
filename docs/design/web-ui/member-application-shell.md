# 会员应用壳

> 范围更新（2026-09-16）：Sports 三项清理后，Worm Markets 的 API-only 权限也已独立退役；Worm Trading 七条路由、登录、Wallet 和其他板块保留。下文记录当前七模块权限界面；原 main 现场数据退役不改变会员壳源码状态。

> 设计状态：已实现

> [板块访问开关](../../requirements/development-runtime/business-access-control.md)已实现。会员壳独立于账户七模块权限读取六项环境访问状态：关闭时最多 5 秒中止请求、卸载正文并保留菜单和 URL；无法确认时不开放，重新开放后重新读取，晚响应不能恢复正文。后台任务不随页面关闭停止。`worm` 统一控制 Worm Trading 路由、弹窗和浏览器交易驱动，不自动补发交易。实际 GUI 和双 realm 证据见[全栈验收](../../testing/full-stack-access-acceptance.md)。

## 范围

会员应用壳负责 ATHENA 普通账户的浏览器体验：Google 与 Phantom 登录、共享用户名注册、认证 bootstrap、Pending 访问、响应式会员导航、Account Center、API Keys、Profit Sharing、Telegram 绑定及所有业务模块页面。它也负责会员范围的授权刷新、请求取消、缓存和瞬时状态清理，以及阻止管理员进入会员应用的跳转。

共享部署与会话边界见[应用壳](application-shell.md)；管理员账户管理、Profit Sharing 治理和系统运维见[管理员应用壳](administrator-application-shell.md)。Notifications 是普通账户自助页面，不公开系统投递历史或测试发送。

## 单一深色与验收边界

两份 HTML 固定深色首屏，入口级 `AthenaThemeProvider` 在 bootstrap 前提供统一 token 和本地 Inter／JetBrains Mono；系统 light／dark 输入均不改变外观。桌面侧栏 224px、顶栏 64px、内容边距 32px，900px 以下使用手机抽屉和 20px 外侧间距。Ant 字号 rem 桥接、弹窗滚动正文／固定操作区及完整关闭目标沿共用实现。

两端 Appearance 已删除并使用各自既有 404；主题不是账户或浏览器偏好。分页、排序、侧栏、banner 和返回位置仍按 realm 隔离。主题重构时 37 条改版入口已实现（本次后续清理删除其中三个 Sports 入口），8 条 Trader Sync 专页仅共享主题回归；管理员 Service Status 的 Trader Sync 页签已按 v16 重排。Token 导航未开启。

正式 React、原生浏览器缩放、真实本地读取与外部未验证项分别见[验收记录](../../testing/web-ui-theme-refactor-acceptance.md)；完整基线与最终差量使用各自提交，不把设计批准或 fixture 成功当真实供应商成功。

## 源码入口

| 职责 | 源码 | 关键符号 |
| --- | --- | --- |
| 入口与 Shell | [ui/src/app/entry/member.tsx](../../../ui/src/app/entry/member.tsx)、[ui/src/app/member/app.tsx](../../../ui/src/app/member/app.tsx) | `MemberApp`、bootstrap 边界、`Shell`、会员角色 guard |
| 会话 bootstrap | [ui/src/app/session/bootstrap.tsx](../../../ui/src/app/session/bootstrap.tsx) | `SessionBootstrap`、`loadAppBootstrapWithRetry` |
| 懒加载路由 | [ui/src/app/member/routes.tsx](../../../ui/src/app/member/routes.tsx) | 会员页面的 route-level import |
| 登录与注册 | [login.tsx](../../../ui/src/app/member/pages/login.tsx)、[register.tsx](../../../ui/src/app/member/pages/register.tsx) | `LoginPage`、`RegisterPage`、`PhantomProvider` |
| Account Center 与 Security | [account-center.tsx](../../../ui/src/app/shared/pages/account-center.tsx)、[account-security.tsx](../../../ui/src/app/member/pages/account-security.tsx) | 共享账户自助、会员 API Key 页面 |
| Telegram 绑定 | [notifications.tsx](../../../ui/src/app/member/pages/notifications.tsx)、[notification-service.ts](../../../ui/src/app/member/notification-service.ts)、[notification-storage.ts](../../../ui/src/app/member/notification-storage.ts) | `NotificationsPage`、`MemberNotificationService`、标签页绑定指令 |
| Trader Sync | [app.tsx](../../../ui/src/app/member/app.tsx)、[routes.tsx](../../../ui/src/app/member/routes.tsx)、[trader-sync-service.ts](../../../ui/src/app/member/trader-sync-service.ts)、[pages/trader-sync](../../../ui/src/app/member/pages/trader-sync)、[state.ts](../../../ui/src/app/member/pages/trader-sync/state.ts) | 六类页面、类型化 API、owner-scoped 草稿/页栈/读取会话、`clearTraderSyncState` |
| Solana | [solana.tsx](../../../ui/src/app/member/pages/solana.tsx)、[solana-service.ts](../../../ui/src/app/shared/services/solana-service.ts) | 新 Mint 候选列表、查询、分页、链上详情与持久扫描状态 |
| 权限与可见读取 | [access-modules.ts](../../../ui/src/app/shared/access-modules.ts)、[context.ts](../../../ui/src/app/shared/context.ts)、[use-visible-query.ts](../../../ui/src/app/shared/use-visible-query.ts) | 七模块、`AuthorizationCtx`、可见时 single-flight |
| 请求与缓存清理 | [requests.ts](../../../ui/src/app/shared/services/requests.ts)、[data.ts](../../../ui/src/app/components/data.ts) | realm/account/session scope、abort、cache generation |
| 服务注册 | [services.ts](../../../ui/src/app/member/services.ts)、[registry.ts](../../../ui/src/app/shared/services/registry.ts) | realm-owned `MemberServices` 与中立共享 facade |
| 服务端鉴权 | [authz.go](../../../internal/server/authz.go)、[tradersync.proto](../../../internal/server/tradersync/tradersync.proto) | 普通交互账户规则、Trader Sync 读写 RPC、Telegram 绑定 RPC |
| 样式 | [member.css](../../../ui/src/app/styles/member.css)、[member-features.css](../../../ui/src/app/styles/member-features.css)、[trader-sync.css](../../../ui/src/app/member/pages/trader-sync/trader-sync.css) | 会员 Shell、功能页和 Trader Sync 局部主题规则 |

## 架构

`MemberApp` 是会员 HTML 唯一 React root，管理匿名登录/注册、已认证会员 Shell 和维护/错误三类状态。bootstrap 返回认证会话后先验证 `administrator=false`，再构造会员业务服务；管理员通过整页跳转到部署相对的 `/admin`。

入口在 bootstrap 前把请求 realm 固定为 `member`。API 请求带 `X-Athena-Application-Realm: member`，认证开启时只选择 HttpOnly `athena.token.member`，loopback disabled-auth 时只选择 `local-user`。`AccountAvatar`、Wallet、Worm Trading 的浏览器原生资源请求使用 `athenaRealm=member`；外部图片 URL 不改写。realm 只选择会话槽或开发身份，角色、模块和操作级授权仍由服务端判断。

桌面使用常驻侧栏，紧凑布局使用 drawer。导航投影当前普通账户的七模块权限：

- **Markets**：Market Radar、Managed OO、Worm Trading，以及 grant 严格为 `READ_WRITE` 时的 Trader Sync。
- **Token & Risk**：Token 为禁用项，仅在 Token 为 `READ`/`READ_WRITE` 时显示；Solana 在拥有 `READ` 时显示并进入 `/solana`；Wallet 按自己的模块显示。
- **Operations**：有权益时显示 Profit Sharing；每个已认证普通交互会话均显示 Notifications。

Token 在 Trader Sync 加入前已经存在，仍没有路径、页面、service、return snapshot 或 landing。权限卡使用排除 Token 的 `accountAccessDisplayModules`，但完整七模块 aggregate 仍参与账户状态，因此 Token-only 账户仍为 Active。

Solana 只接受 `NONE` 与 `READ`，列表、查询和链上详情由[Solana 列表设计](solana-discovery.md)定义。页面刷新只重读持久数据，不触发扫描；当前十一应用全栈会启动 Solana 发现进程，局部选择未启动该服务时仍按真实请求结果显示不可用，不用空列表掩盖故障。

Trader Sync 只接受 `NONE` 与 `READ_WRITE`。非法持久/网关 `READ` 经 `normalizeModuleGrant` 归为 `NONE`，不存在合法只读回退；六条业务路由和导航都要求严格 `READ_WRITE`。后端的读 RPC 使用模块 READ requirement 是鉴权层复用规则，不代表产品支持 READ grant。

会员路由通过 `React.lazy` 加载，入口依赖图不导入管理员页面或 `AdminNotificationService`。中立组件、模型和 transport 可以进入共享 chunk，但会员与管理员业务 service/state 不能交叉消费。

## 路由与运行流程

1. 匿名访问显示 `/login`。Google OIDC 带 `athenaRealm=member`；Phantom 使用会员专属浏览器注入 SIWS。未知身份携带服务端注册 ticket 进入共享 `/register`。
2. bootstrap 只选择 member cookie/`local-user`，返回账户、资料、偏好、角色和完整权限 aggregate。管理员在任何会员业务请求前离开；普通账户进入会员授权上下文。
3. Pending 账户默认进入 `/account/access`，仍可使用 Profile、Access、Help、Notifications 和 Logout，不挂载业务模块页面。Active 普通账户默认进入 `/account/profile`。
4. 普通模块路由要求对应 READ，写控件要求 `READ_WRITE`；Profit Sharing 使用独立权益。Notifications 是直接自助例外，服务端仍要求普通交互登录并拒绝管理员/API Key。
5. `/notifications` 读取 Bot、binding 与 attempt，显示 `Unavailable`、`Not connected`、`Waiting for Telegram`、`Link expired`、`Setup failed`、`Connected` 或 `Needs attention`。Configure 返回深链、准确 fallback command、到期倒计时和浏览器本地 Ant Design `QRCode`。
6. 未过期 attempt 在页面可见且没有 mutation 时每 3 秒刷新；focus、`visibilitychange`、手动刷新与 mutation 恢复共用同一 single-flight read。Cancel 删除 attempt；Reconnect 在新 token 成功前保留原 binding；Disconnect 经确认后删除 binding 和 attempt。
7. Trader Sync 六条路由为 `/trader-sync`、`/trader-sync/add`、`/trader-sync/subscriptions`、`/trader-sync/subscriptions/:subscriptionId`、`/trader-sync/activities/:activityId`、`/trader-sync/summaries/:batchId`。主页、列表、详情、活动和摘要读取在可见时每 5 秒 single-flight，隐藏停止，恢复 visible 或 focus 立即读取。
8. 活动列表使用固定 snapshot、refresh cursor 和 `hasNewer`；刷新只更新本页固定成员，用户点击提示后才取新 snapshot。Previous/Next 使用真实 opaque cursor 与会话页栈，不推算页数。详情返回在目标 DOM 提交后恢复筛选、页栈、滚动与选择位置。
9. 从 Add 打开 Notifications 时，navigation state 只允许精确 `returnTo === '/trader-sync/add'`；仅当前 owner 仍有内存草稿才显示返回入口。备注、确认 token 和请求内容不进入 navigation state、URL、`localStorage` 或 `sessionStorage`。
10. 会员 Shell 在退出、accountId/issuer/admin 身份变化，以及 Trader Sync 从 `READ_WRITE` 降为任何较低值时，先使旧 generation 失效并调用 `clearTraderSyncState()`。它清除添加草稿、订阅编辑/页栈、活动与详情会话；旧请求即使不可取消地晚成功或晚 finally，也不能恢复内容。
11. 用户信息读取在可见时按 freshness 间隔、focus/visible 返回、稳定拒绝及 Pending 显式刷新触发并去重。普通写入结果未知时保留同 payload 供用户显式恢复，不自动重放；背景刷新失败保留最后成功数据和陈旧提示。
12. Logout 清理会员敏感状态，只撤销/清除 `athena.token.member`，整页返回 `/login`；同源管理员标签和 `athena.token.admin` 不受影响。

其余业务路径包括 `/wallet`、`/worm-trading/*`、`/market-radar/*`、`/managed-oo/*`、`/profit-sharing/*`、`/account/*` 与 `/help`。会员应用没有 Service Status 或 Etherscan Gateway 路由。

## 状态与数据

会员授权投影包含 account UUID、不可变 username、安全身份展示、profile、access revision、Profit Sharing 权益和完整七模块。UUID、member realm、issuer 与 session generation 共同限定请求、缓存和 Trader Sync 内存状态；username/display name 只用于展示。

私钥和新签发 API Key 只存在于当前 React result state。离开页面、结束会话、改变账户或失去权限会丢弃它们并取消工作。浏览器不持久化 provider token、wallet signature、外部 subject、registration ticket、管理员断言或 HttpOnly cookie。

Telegram 服务端拥有 binding/attempt。浏览器只在标签页键 `athena.member.notifications.telegram-attempt` 保存 `attemptId`、`deepLink`、`fallbackCommand`；仅当服务端 pending attempt ID 匹配时复用。绑定成功、失败/替换、过期、身份或会话变化、取消和解绑都会清除。

Trader Sync 的订阅、活动、通知证据均由 owner-scoped API 提供；这不扩展为通用账户通知历史。日期和时间明确按 UTC+8 展示，活动结束日期转换为 `[from,to)` 的次日边界。金额、ID 和 revision 保留字符串精度。

会员持久键使用 `athena.member.*`。固定深色与字体为共享展示；会员 filter、draft、return position 和 feature cache 不供管理员使用；Trader Sync 会话只在内存中存在，reload 后不承诺恢复。

## 配置与不变量

会员浏览器没有角色或模块 allowlist 配置；权限来自 `GetAppBootstrap` 与 `GetUserInfo`。应用 base 和部署 base 来自会员 HTML；realm/header/query 名是固定协议值。Notifications 的 3 秒和 Trader Sync 的 5 秒可见刷新周期、UTC+8 与响应式断点是前端常量。

- 只有 `administrator=false` 的账户能进入认证会员 Shell。
- member 入口只选择 `athena.token.member`，不回退到管理员 cookie；header/query realm 冲突认证失败。
- 每个业务路由与服务端操作都重复授权；Trader Sync 六路由严格要求 `READ_WRITE`。
- Pending 不发起业务模块或 Profit Sharing 请求，但可使用 Telegram 绑定。
- Telegram 绑定只接受普通交互登录；会员 bundle 不包含系统通知详情、测试发送或管理员页面。
- 七模块不等于七个页面或导航项；Token 没有业务页面，导航仍按权限显示禁用项。
- accountId、issuer、realm 和 generation 是身份敏感状态的 key；撤权清理发生在跳转前。
- 登出与角色错配使用整页导航，不提供跨 realm switcher；会员登出不结束管理员会话。

## 故障恢复与可观测性

bootstrap 对暂时失败做有界重试，并把维护状态与匿名状态分开。授权拒绝刷新真实 aggregate，不合成 grant；刷新失败保留明确错误/重试边界。初次读取失败显示错误，不伪装空列表；后续失败保留最后成功数据和更新时间。

Telegram action 只有服务端 mutation 成功后才更新本地状态；Reconnect 失败保留原 binding，Disconnect 失败仍保持 connected。`sessionStorage` 不可用时，新深链只在当前 render 有效；reload 后若服务端存在 pending attempt 而本地指令不匹配，页面说明其属于原标签页并提供取消/新建恢复。

Trader Sync 的资源 ID 不存在或跨 owner 返回 NotFound/404；签名游标或其 owner/filter/kind/pageSize 上下文错误返回 InvalidArgument/400 与通用正文。失权清屏及晚响应 fence 不借 URL/query 推导 owner。unknown mutation 不自动重放，刷新错误不删除已有事实。

稳定错误原因区分维护、模块、Profit Sharing、交互账户、API Key 与认证失败。Notifications 状态、到期、绑定身份与时间不暴露投递历史；Trader Sync 明确显示 unavailable、monitoring interruption、unknown 和 stale。日志与 UI 错误不包含 cookie、JWT、API Key bearer、provider token、wallet signature、private key、registration ticket、一次性深链或 fallback command。

## 维护检查

- [ ] 会员路由、七模块导航、Solana `READ`、严格 Trader Sync `READ_WRITE`、Notifications 和 lazy imports 保持同步。
- [ ] 管理员在会员 service 构造和请求前离开。
- [ ] Pending、Telegram 临时状态、授权刷新、request abort、cache/Trader Sync state 清理与 logout 保持一致。
- [ ] Trader Sync 六路由、5 秒可见刷新、游标/返回位置、UTC+8 与失权晚响应 fence 保持一致。
- [ ] Notifications 只增加 Add 草稿往返，不扩展系统历史或测试发送；解绑/重绑资格文案与后端一致。
- [ ] Wallet、Worm、Profit Sharing、Token 禁用项和权限展示边界与对应文档一致。
- [ ] 源码链接和[设计索引](../README.md)保持正确。
