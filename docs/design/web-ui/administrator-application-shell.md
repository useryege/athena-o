# 管理员应用壳

> 设计状态：已实现

> [板块访问开关](../../requirements/development-runtime/business-access-control.md)已实现。Service Status 在原 Services、Notifications、Trader Sync 三个只读来源之外增加独立 Module Access 第四页签，展示六项 OPEN／CLOSED、修改人和时间，并提供显式保存；开关管理和健康始终可用，业务页面同样受限。后台任务继续运行，Token 接入延期，`worm` 只对应 Trading。实际管理员 GUI、手机／桌面和审计持久化证据见[全栈验收](../../testing/full-stack-access-acceptance.md)。

## 范围

管理员应用壳负责 ATHENA 管理专用浏览器体验：Google 登录、管理员角色 guard、账户目录与权限编辑、Profit Sharing 治理、Service Status、Etherscan Gateway、系统通知列表/详情/测试、管理员自助、响应式管理导航及管理员范围的请求清理。它不公开会员模块、API Keys、Phantom、Profit Sharing 会员操作、会员 Telegram 绑定或跨应用 switcher。

共享部署与会话边界见[应用壳](application-shell.md)；普通账户业务路由和模块授权见[会员应用壳](member-application-shell.md)。

## 单一深色与验收边界

两份 HTML 固定深色首屏，入口级 `AthenaThemeProvider` 在 bootstrap 前提供统一 token 和本地 Inter／JetBrains Mono；系统 light／dark 输入均不改变外观。桌面侧栏 224px、顶栏 64px、内容边距 32px，900px 以下使用手机抽屉和 20px 外侧间距。Ant 字号 rem 桥接、弹窗滚动正文／固定操作区及完整关闭目标沿共用实现。

两端 Appearance 已删除并使用各自既有 404；主题不是账户或浏览器偏好。分页、排序、侧栏、banner 和返回位置仍按 realm 隔离。37 条改版入口已实现，8 条 Trader Sync 专页仅共享主题回归；管理员 Service Status 的 Trader Sync 页签已按 v16 重排。Token 导航未开启。

正式 React、原生浏览器缩放、真实本地读取与外部未验证项分别见[验收记录](../../testing/web-ui-theme-refactor-acceptance.md)；完整基线与最终差量使用各自提交，不把设计批准或 fixture 成功当真实供应商成功。

## 源码入口

| 职责 | 源码 | 关键符号 |
| --- | --- | --- |
| 入口与 Shell | [ui/src/app/entry/admin.tsx](../../../ui/src/app/entry/admin.tsx)、[ui/src/app/admin/app.tsx](../../../ui/src/app/admin/app.tsx) | `AdminApp`、管理员 bootstrap/role guard、访问复查状态 |
| 登录与共享 bootstrap | [bootstrap.tsx](../../../ui/src/app/session/bootstrap.tsx)、[login.tsx](../../../ui/src/app/admin/login.tsx) | `SessionBootstrap`、`AdminLoginPage` |
| 账户、治理与系统页 | [pages](../../../ui/src/app/admin/pages) | Accounts、Profit Sharing、Service Status、Etherscan、Notifications |
| Trader Sync 管理视图 | [trader-sync-models.ts](../../../ui/src/app/admin/trader-sync-models.ts)、[trader-sync-service.ts](../../../ui/src/app/admin/trader-sync-service.ts)、[pages/trader-sync](../../../ui/src/app/admin/pages/trader-sync) | 安全 Summary DTO、列表/详情、runtime |
| 管理员读 scope | [read-scope.ts](../../../ui/src/app/admin/read-scope.ts)、[use-visible-query.ts](../../../ui/src/app/shared/use-visible-query.ts) | `beginAdminReadSession`、`endAdminReadSession`、`useAdminReadScope`、可见 single-flight |
| 系统通知 | [system-notifications.tsx](../../../ui/src/app/admin/pages/system-notifications.tsx)、[system-notification-detail.tsx](../../../ui/src/app/admin/pages/system-notification-detail.tsx)、[notification-service.ts](../../../ui/src/app/admin/notification-service.ts) | 列表、详情、测试、Notification runtime |
| 服务注册 | [services.ts](../../../ui/src/app/admin/services.ts)、[registry.ts](../../../ui/src/app/shared/services/registry.ts) | realm-owned `AdminServices`、`ensureAdminBusinessServices` |
| 服务端鉴权 | [authz.go](../../../internal/server/authz.go)、[accountaccess/controller.go](../../../internal/accountaccess/controller.go) | `administratorGRPCMethods`、`Controller.Authorize` |
| 请求、缓存与资源 realm | [requests.ts](../../../ui/src/app/shared/services/requests.ts)、[data.ts](../../../ui/src/app/components/data.ts) | admin realm/session generation、abort、cache 清理 |
| 样式 | [admin.css](../../../ui/src/app/styles/admin.css)、[admin-features.css](../../../ui/src/app/styles/admin-features.css)、[trader-sync.css](../../../ui/src/app/admin/pages/trader-sync/trader-sync.css) | 管理员 Shell、功能页与局部 Trader Sync 状态样式 |

## 架构与权限

`AdminApp` 是 `admin/index.html` 唯一 React root。它先读取共享会话，再验证持久管理员角色，只有成功后才构造管理 service 或发管理请求。普通认证账户停留在管理员 bundle 内看到 Athena Admin 403，并有整页链接回会员根路径。

入口在 bootstrap 前把请求 realm 固定为 `admin`。所有 API 请求带 `X-Athena-Application-Realm: admin`，认证开启时只选择 HttpOnly `athena.token.admin`，loopback disabled-auth 时只选择 `local-admin`。`AccountAvatar` 使用 `athenaRealm=admin` 读取相对上传资源；外部图片 URL 不改写。realm 只选择会话，持久管理员角色与操作鉴权仍是权威。

桌面使用常驻管理侧栏，紧凑布局使用 drawer。固定导航为：

- **Account Admin**：Accounts。
- **Governance**：Profit Sharing。
- **System**：Service Status、Etherscan Gateways、Notifications、Trader Sync。

`/admin/notifications` 和 `/:id` 管理系统 Telegram 投递；Service Status 展示共享 Notification runtime。Trader Sync 使用独立管理员 service/read scope，只提供安全订阅概要和 runtime，不导入会员 model、draft 或 cache。普通账户、API Key 和会员 service 都不能调用这些管理员 RPC。

权限编辑器通过 `accountAccessDisplayModules` 隐藏 Token 控件，但 draft、reset、冲突 reload、比较和更新都保留完整七模块 aggregate；改变其他权限不得丢 Token level。管理员自己的 aggregate 仍是不可编辑的登录专用状态。

## 路由与页面流程

1. 匿名 `/admin/*` 访问显示 `/admin/login` 并保留校验后的管理员本地 return target；仅提供 Google。OIDC 使用 `athenaRealm=admin`，成功注册写 `athena.token.admin`。
2. bootstrap 只选择 admin cookie/`local-admin` 并核持久角色。普通账户在 management service 构造前收到 403；管理员建立 accountId+iss+generation 的独立 read session。
3. `/admin` 重定向 `/admin/accounts`。账户页分页/搜索安全身份资料，以 expected revision 一次更新普通账户完整可变权限 aggregate；管理员行只读。
4. Governance 使用共享 `ListRounds`/`GetRound` 的管理员权限，只提供 lifecycle/roster 管理，不创建会员 proposal 或 vote。
5. 系统通知页提供 keyword、delivery status、`test`/`prod` chat 筛选；详情显示 provider-facing 规范化结果。Test Notification 单飞提交，提交中不可关闭，成功后刷新列表，失败保留弹窗和错误。
6. 管理员 Trader Sync 路由为 `/admin/trader-sync/subscriptions` 与 `/admin/trader-sync/subscriptions/:id`，并与 `/admin/service-status` 互链。列表 filter 先保留 draft，只有 Apply 才生效；accountId trim、wallet trim 并小写，includeCancelled=true 表示当前与已取消全部，默认 pageSize=50。
7. Trader Sync 列表 Previous 使用本页真实输入 cursor，Next 使用响应 nextCursor，不推算 total。只显示用户安全身份、完整钱包、生命周期、观察、活动数和关联逻辑 delivery 数；不同订阅行的 Associated deliveries 不可求和。详情无备注、完整活动、消息正文、逐条 delivery 或 Pause/Resume/Cancel/Resend。
8. Service Status 对 Services、Notification Runtime、Trader Sync 各维护独立 10 秒可见 single-flight；Module Access 使用独立 5 秒可见 single-flight。hidden 不发新请求，visible/focus 和手动 Refresh 使用各自 reload。某一来源 pending/失败不阻塞其余来源，失败保留该来源最后成功值、时间和 stale 提示；访问设置无法确认时禁用修改，不伪造 CLOSED。
9. Profile、Access、Help 只操作当前管理员 UUID；管理员应用不创建 API Key。Logout 只撤销/清除 `athena.token.admin`，会员会话不受影响。

其他路由为 `/admin/profit-sharing`、`/admin/profit-sharing/:slug`、`/admin/etherscan-gateways`、`/admin/notifications`、`/admin/notifications/:id`、`/admin/account/profile`、`/admin/account/access` 与 `/admin/help`。管理路由都位于 `/admin` 应用根下，已移除的 Appearance 地址统一落入既有 404 兜底。

## 身份清理与访问复查

`beginAdminReadSession(user)` 以 accountId+iss 建立 generation；Shell logout、401、维护型 503、失 admin、身份变化或卸载调用 `endAdminReadSession()`，先清读 scope 和 cache，再进行导航或状态切换。页面普通卸载只取消本页 query，不结束整个管理员 session。旧 query 的成功、失败或 finally 都必须通过 generation fence。

普通业务 503 保留各来源旧数据和独立错误，不视为失权。401 或 `503 + code 14 + 系统维护中` 立即清 scope 并跳登录。403 且 reason=`ACCOUNT_ADMIN_REQUIRED` 时立即清屏，再调用真实 `/api/v1/session/userinfo` 复查角色；复查期间以 `Administrator access check` region 取代业务页，显示 `Checking administrator access`、`aria-busy=true`，禁用 `Retry access check`。

复查网络/普通 503 失败显示 `Could not verify administrator access` 和原错误，role=alert，并开放 `Retry access check`。重试 single-flight；只有最新复查成功且 `loggedIn=true`、`administrator=true`，才通过原 refresh 建立合法新 session 并重新读取当前路由。返回失管理员显示 Forbidden；之后的新 401 或先前 userinfo 晚成功都不能恢复业务页。

## 数据与 Service Status 语义

管理员授权投影包含 account UUID、持久管理员角色、profile、安全 provider 身份、access revision 和登录专用 aggregate。请求/cache key 包含 admin realm、viewer UUID、issuer 与 generation；持久 UI key 使用 `athena.admin.*`，不读取会员 drafts、filters、return positions 或 feature cache。

Trader Sync Summary DTO 从源头白名单化：ID 与计数保持 string，不经 JavaScript number；缺值显示 `Unavailable`/`Unknown`，raw 不可观测不补 0。健康六态为 pending_baseline、healthy、interrupted、paused、permission_disabled、cancelled；healthy 显示 `Monitoring`，interrupted 显示 `Monitoring interrupted`。

Service Status 各来源时间不能混用：Services 的 `Last checked` 来自 server `checkedAt`；Notification 的 `Last received` 是客户端最近成功读取时间；Trader Sync 的 `As of` 来自 server `asOf`；Module Access 的修改时间是服务端 RFC3339Nano 审计值。全部时间按 UTC+8 展示。

Notification 保留 System/Account pending、retry、failed、sending、unknown 等计数及 recovery。`remainingMillis`/`elapsedMillis` 是服务端精确 string，合法 `0` 显示 `0 ms`，缺失显示 `Unavailable`，浏览器不倒计时。initializing/waiting 显示 recovering；顶层 failed/stopped 优先但仍保留 recovery 的 reason、started 和 clock。

Trader Sync 显示 Collector Connected/Disconnected/Unavailable、collectorEpoch、filterRevision、raw 可观测性与全部 runtime metric。gauge 标记 `Current gauge` 并保留存在的 serviceEpoch；window 显示 windowStart→windowEnd；epoch 显示 opaque `Service epoch`。不同 unit、window 和 producer 不求和，serviceEpoch 不解释为 UTC 时间。当前 window fixture 是明确 synthetic，不能冒充真实进程窗口。

## 配置、不变量与故障恢复

浏览器没有管理员 allowlist；角色只来自 bootstrap 与当前服务端授权。`ATHENA_ADMIN_GOOGLE_EMAIL` 仅是未知 Google 身份进入 admin realm 时创建唯一管理员 persona 的服务端准入规则。Notifications/Trader Sync 页面没有浏览器端 endpoint、Bearer、chat 或授权配置。

- 只有持久管理员角色能进入管理 Shell 或调用管理操作；路由可见性不是授权依据。
- admin 入口只选择 `athena.token.admin`，不回退 member cookie；realm header/query 冲突认证失败。
- 管理员角色不满足会员模块、API Key、Profit Sharing 参与者或会员 Telegram 绑定要求。
- 管理员依赖图不导入 member route/service/state；会员 bundle 也不导入管理员通知或 Trader Sync 概要页面。
- 账户权限编辑只针对普通账户并使用完整 aggregate revision CAS；管理员 aggregate 不可变。
- Trader Sync 认证身份不由 account_id filter、wallet、资源 ID 或 cursor 覆盖。
- 管理员登出不结束会员会话；跨 realm 导航使用部署根整页 URL。

列表/详情失败留在各自 `AppPage`；测试发送失败不宣布入队成功。首次 Service Status 来源失败显示该来源错误，后续失败保留上次成功结果；request cancel 不显示成业务失败。缺失/非法 admin realm 认证失败，不选择会员 cookie；未知管理员 route 显示管理员品牌 404。

## 可观测性与维护检查

稳定 reason 区分 administrator-required、revision conflict、maintenance 与 authentication。系统通知详情供操作员查看规范化 provider 结果；Service Status 显示三个来源的更新时间、恢复、积压、raw 边界和 metric 单位。日志只用安全 account UUID 与 operation metadata，不记录 cookie、JWT、Google token、subject、API Key bearer 或管理 secret。

- [ ] 管理员登录、角色 guard、路由、System→Trader Sync 导航和响应式 Shell 保持同步。
- [ ] 普通账户在管理 service 构造/请求前被拒绝；admin/member bundle 边界保持单向中立共享。
- [ ] Accounts、Governance、Service Status、Etherscan、Notifications 与 Trader Sync 保留显式管理员规则。
- [ ] Trader Sync 安全 DTO、两路由、filter/cursor 和 Associated deliveries 口径保持一致。
- [ ] 三个状态来源的 10 秒及 Module Access 的 5 秒可见 single-flight、错误隔离、单位/window/epoch 与 UTC+8 保持一致。
- [ ] 401、维护、`ACCOUNT_ADMIN_REQUIRED` 清屏、可见复查失败和 Retry access check 保持当前行为。
- [ ] 源码链接和[设计索引](../README.md)保持正确。
