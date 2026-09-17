# 账户访问控制

> 范围更新（2026-09-16）：Sports 三项清理后，`worm_markets` 权限又由追加迁移 `000005_remove_worm_markets_access.sql` 精确删除；当前矩阵为七项，只保留 `worm_trading`。旧 Markets grant 不转换成 Trading grant，受影响账户 revision 按既有机制更新。代码、隔离升级与原 main 现场迁移见[退役验收](../../testing/worm-markets-retirement-acceptance.md)；原两个账户 revision 由 1 更新为 2，删除两条 Markets grant，其余 14 条 grant 与 flags 保持。

> 设计状态：已实现

> 新增目标：[单客户端登录](../../requirements/identity-access/single-client-login.md)已确认，新登录生效后旧登录立即失效，由[单客户端登录目标设计](single-client-login.md)承接。该账号级会话约束尚未实现；本文既有模块权限与 realm 边界继续有效。

> 交易目标：用户已确认[手动交易共用现有 Trader Sync 权限](../trading/polymarket-manual-trading.md#shared-trader-sync-access)，钱包操作权限和归属继续独立核验。该交易能力尚未实现；当前模块矩阵仍按下文所列源码运行。

> 访问接入（2026-09-17 已实现）：普通 Wallet、登录与账户权限保持核心能力；六个业务板块在原账户授权之后再检查环境访问设置。Worm 用户钱包选择、连接、交易及其专属二次验证统一受 `worm` 控制；关闭时不签发新的 Worm 凭据或交易授权，已受理后台工作与内部目的绑定签名按原规则处理。详见[现行需求](../../requirements/development-runtime/business-access-control.md)及[全栈验收](../../testing/full-stack-access-acceptance.md)。

## 范围

账户访问控制负责 ATHENA 的持久授权模型：外部登录开关、独立 API Key 与 Profit Sharing 权益、完整七模块矩阵、revision 乐观更新、Pending/Active/Blocked 状态、RPC 鉴权，以及会员与管理员应用的严格边界。所有聚合和权限查找使用稳定账户 UUID。Wallet 操作还叠加凭据能力约束；精确行所有权由 Wallet 服务独立核验。

`member` 与 `admin` 是独立 application realm。同一 Google subject 可以拥有两个独立身份：普通会员 UUID 与唯一管理员 UUID，用户名全局不同，权限、资料、偏好、API Key、Wallet 和业务关系都不合并。

[账户凭据](account-credentials.md)负责 UUID、不可变用户名、JWT 验证与持久管理员事实。[Google OIDC](google-oidc-login.md)和[Solana Wallet 登录](solana-wallet-authentication.md)把已验证身份交给统一注册边界；普通账户在完成用户名设置后才创建完整零授权聚合。通过本层授权后，具体业务状态仍由各业务服务负责。

## 源码位置

| 职责 | 源码 | 关键符号 |
| --- | --- | --- |
| 权限模型与要求 | [internal/accountaccess/access.go](../../../internal/accountaccess/access.go) | `Access`, `Module`, `AccessLevel`, `Requirement`, `IsPending`, `Validate` |
| 快照与持久 CAS | [internal/accountaccess/controller.go](../../../internal/accountaccess/controller.go) | `Controller`, `NewController`, `Register`, `Get`, `Update`, `Authorize` |
| PostgreSQL 适配器与查询 | [internal/accountstate/store/sql_store.go](../../../internal/accountstate/store/sql_store.go), [internal/accountstate/store/queries/account_access.sql](../../../internal/accountstate/store/queries/account_access.sql) | `ListAccountAccess`, `GetAccountAccess`, `UpdateAccountAccess` |
| 账户目录 | [internal/accountstate/store/queries/account_directory.sql](../../../internal/accountstate/store/queries/account_directory.sql) | `CountAccountDirectory`, `ListAccountDirectoryPage` |
| 公开账户契约 | [internal/server/account/account.proto](../../../internal/server/account/account.proto), [internal/server/account/account.go](../../../internal/server/account/account.go) | `AccountAccess`, `AccountStatus`, `ListAccounts`, `UpdateAccountAccess` |
| RPC 鉴权 | [internal/server/authz.go](../../../internal/server/authz.go) | `moduleGRPCRules`, `authorizeGRPC`, `authorizeAccountSelfService` |
| Telegram 绑定鉴权 | [internal/server/authz.go](../../../internal/server/authz.go), [internal/server/notification/notification.go](../../../internal/server/notification/notification.go) | 普通会员交互凭据边界、Telegram 绑定 RPC、服务端注入账户 UUID |
| 原生敏感操作与 Worm 管理鉴权 | [internal/server/wallet_secret.go](../../../internal/server/wallet_secret.go), [internal/server/wallet_avatar.go](../../../internal/server/wallet_avatar.go), [internal/server/worm_connection.go](../../../internal/server/worm_connection.go), [internal/server/worm_wallet_selection.go](../../../internal/server/worm_wallet_selection.go), [internal/server/worm_combinations.go](../../../internal/server/worm_combinations.go), [internal/server/worm_execution_plans.go](../../../internal/server/worm_execution_plans.go), [internal/server/worm_executions.go](../../../internal/server/worm_executions.go), [internal/server/worm_position_cash_outs.go](../../../internal/server/worm_position_cash_outs.go), [internal/server/worm_position_cash_out_batches.go](../../../internal/server/worm_position_cash_out_batches.go), [internal/server/worm_execution_authorization.go](../../../internal/server/worm_execution_authorization.go) | `authenticateWalletSecretHTTP`, `authenticateInteractiveWormTradingHTTP`, `authenticateWalletAvatarHTTP`, Wallet 选择、组合、Preview、Run、Cash Out 路由及 exact-origin / proof 边界 |
| 会话投影 | [internal/server/session/session.go](../../../internal/server/session/session.go), [internal/server/appbootstrap/appbootstrap.go](../../../internal/server/appbootstrap/appbootstrap.go) | `ProjectUserInfo`, `GetAppBootstrap` |
| realm 传输与 Cookie 鉴权 | [internal/server/application_realm.go](../../../internal/server/application_realm.go), [util/http/http.go](../../../util/http/http.go), [ui/src/app/shared/services/requests.ts](../../../ui/src/app/shared/services/requests.ts) | `applicationRealmFromIncomingContext`, `authenticateRealmLoginCookie`, `RealmAuthCookieName`, `configureAuthorizationRealm` |
| 浏览器路由与刷新 | [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts), [ui/src/app/shared/context.ts](../../../ui/src/app/shared/context.ts), [ui/src/app/member/app.tsx](../../../ui/src/app/member/app.tsx), [ui/src/app/admin/app.tsx](../../../ui/src/app/admin/app.tsx) | 会员模块刷新、管理员角色校验, `AuthorizationCtx`, `canRead`, `canWrite` |
| 管理员工作区 | [ui/src/app/admin/app.tsx](../../../ui/src/app/admin/app.tsx), [ui/src/app/admin/pages/admin-accounts.tsx](../../../ui/src/app/admin/pages/admin-accounts.tsx) | 管理员路由图, `AdminAccountsPage`, `AccountAccessEditor` |


## 架构与角色

`Access` 包含持久 `Administrator` 投影、`LoginEnabled`、`APIKeyEnabled`、`ProfitSharingEnabled`、七个模块权限及正 `Revision`。PostgreSQL 是权威来源；单 API Server 的 `Controller` 保存分离快照，只发布已提交且校验通过的聚合。`Register(accountID)` 幂等发布已提交账户，并保留更高 revision。

管理员角色只来自 `athena_account.administrator`，读取时关联到聚合，不从用户名、邮件、JWT 文本或请求推断。`Access.Validate` 固定管理员登录开启、API Key/Profit Sharing 关闭、所有会员模块为 NONE。只有显式 `RequirementAdministrator` 允许管理员操作；管理员角色不隐含任何会员能力，不能进入 Wallet、Worm Trading、Token、市场、Profit Sharing 或 Telegram 绑定业务。账户管理、治理、服务状态和 Etherscan 管理分别使用显式管理员规则。公开 Wallet 契约不接受角色或调用方指定 owner。

浏览器认证请求通过 `X-Athena-Application-Realm` 声明 member/admin；无法设置 header 的 GET 使用经校验的 `athenaRealm` query。realm 只选择 `athena.token.member` 或 `athena.token.admin` Cookie，之后必须验证签名账户的持久角色与 realm 一致。客户端不能借 realm 提升权限；两种 Cookie 可共存，但不能相互回退。Logout 只清当前 realm；只有 token 所属账户的持久 realm 与槽位一致才撤销它。会员退出清会员敏感 lease，管理员退出保留会员会话及 lease。

## 模块授权

下列最大权限适用于普通会员及 `local-user`，不是管理员默认值。

| 模块 | 允许授予 |
| --- | --- |
| `market_radar` | NONE / READ |
| `managed_oo` | NONE / READ / READ_WRITE |
| `worm_trading` | NONE / READ / READ_WRITE |
| `token` | NONE / READ / READ_WRITE |
| `solana` | NONE / READ |
| `wallet` | NONE / READ / READ_WRITE |
| `trader_sync` | NONE / READ_WRITE |

鉴权仍采用 `NONE < READ < READ_WRITE`；只读模块拒绝 RW grant。Trader Sync grant 的 READ 由 Go、API 与 SQL CHECK 拒绝，但 READ requirement 合法，RW 可以满足它。模块之间不传递权限，模块与 API Key / Profit Sharing 权益也彼此独立。protobuf 的 Trader Sync enum 使用 12，Solana 使用 13，保留 2、3、6、7、10 不复用；已删除的 2、3、7 及其原名称均声明 reserved。Solana 公开查询在 API 检查 READ 后代理到独立发现服务；API 从已认证凭据注入唯一账户 UUID，独立服务验证内部 Bearer 并从账户数据库重读 Solana READ 权限。管理员没有会员 Solana 查询权限。

账户模块权限与环境访问设置是串行且独立的准入条件。账户聚合固定七项，开关固定六键，不包含 Wallet 或 Token；开关写入不递增账户 revision、授予权限或改变 Pending/Active。缺失设置按 CLOSED，读取失败按暂不可用关闭处理。板块关闭错误携带稳定 reason 与 `module_key`；HTTP 继续使用 snake_case 和数值状态，管理员设置审计时间为 RFC3339Nano。Service Status 的设置写入只允许管理员，普通状态读取仍须已认证，核心入口不受六键阻断。

Worm Trading 的目录及组合 Create／Update 还在服务侧验证内部 Bearer、恰好一个规范 `x-athena-account-id`，组合写的 owner 必须匹配该身份，并读取当前 LoginEnabled 和 `worm_trading` READ／READ_WRITE。该只读账户连接独立于 API runtime；管理员、API Key 或历史 Markets-only grant 均无绕过。

## Telegram 账户自助

Telegram 绑定不属于产品模块矩阵。API Server facade 只接受普通会员的交互凭据，从认证上下文取得规范 UUID 并注入内部请求；浏览器不能选择账户。Pending 会员与隔离的 `local-user` 开发身份可使用；管理员和 API Key 被拒绝。无需任何产品模块或 Profit Sharing grant，绑定状态与进行中绑定请求都不影响 Pending/Active 推导。

## Wallet 与 Worm Trading 能力

模块权限、typed credential 能力与 Wallet 服务所有权共同决定访问。

| 操作 | 登录会话 | API Key | 额外要求 |
| --- | --- | --- | --- |
| Wallet 列表与详情 | 允许 | 允许 | Wallet READ |
| 当前所选 Solana Wallet 摘要、SOL/USDC 余额 | 允许 | 允许 | Worm Trading READ；服务端推导 owner selection |
| 所选 Wallet 连接状态、持仓与进行中请求 | 允许 | 允许 | Worm Trading READ；服务端推导 owner selection |
| 读取持久 Wallet selection | 允许 | 拒绝 | 交互式 Worm Trading READ；无需 lease |
| 全账户连接管理目录 | 允许 | 拒绝 | 交互式 Worm Trading RW；无需 lease |
| 完整替换 Wallet selection | 允许 | 拒绝 | 交互式 RW、exact origin、revision CAS、0–20 项；保存无需 lease |
| Worm 事件目录及已保存组合读取 | 允许 | 拒绝 | 交互式 Worm Trading READ |
| 创建、替换、删除组合 | 允许 | 拒绝 | RW、exact origin、owner/revision；无需 lease |
| 读取执行 Preview 与 Steps | 允许 | 拒绝 | owner 范围内交互式 READ |
| 创建执行 Preview | 允许 | 拒绝 | RW、exact origin、精确来源与 selection revision；无需 lease |
| 读取 Run 与 Steps | 允许 | 拒绝 | owner 范围内交互式 READ |
| 从可用 Preview 创建 Run | 允许 | 拒绝 | RW；source owner/revision/expiry/usability |
| 授权 Run | 新鲜 provider proof 后允许 | 拒绝 | RW；绑定精确 Run/plan/Session/access；不是复用 lease |
| Start / Continue / Heartbeat / Execute Next | Run 授权后允许 | 拒绝 | RW、当前 Session/access；原生命令 exact origin |
| Pause / Terminate Run | 允许 | 拒绝 | RW、exact origin、owner/revision；不能推进执行 |
| 对不确定 Step 做只读 Reconcile | 允许 | 拒绝 | RW、exact origin、owner/revision；不重放修改 |
| 创建或 reconcile 单 Wallet Cash Out | 允许 | 拒绝 | 交互式 RW、exact origin、当前 selection；Close 需操作 proof |
| 创建或控制 Cash-Out batch | 允许 | 拒绝 | 交互式 RW、exact origin、当前 selection、最多 20 个 Wallet、batch proof |
| 读取上传的 Wallet avatar | 允许 | 允许 | Wallet READ 或 Worm Trading READ |
| 修改备注、preset、上传/重置 avatar | 允许 | 允许 | Wallet RW |
| 创建或导入 Wallet | 允许 | 拒绝 | Wallet RW |
| 显示私钥 | 重新认证后允许 | 拒绝 | Wallet RW、owner、登录 Cookie、same origin、五分钟 lease |
| Worm credential connect/reconnect/disconnect | Worm 专用重新认证后允许 | 拒绝 | Worm Trading RW |

Worm Trading 摘要入口不授予普通 Wallet 列表/详情能力。API Server 从当前账户加载所选 Wallet ID，经可信 Wallet 边界解析，只投影 wallet ID、地址、备注、avatar 及对应余额、连接、持仓、进行中请求。没有 selection 时只返回未配置说明。avatar GET 的例外用于这些摘要展示；所有 Wallet 修改仍要求 Wallet RW。

连接与 selection 管理采用原生 HTTP 边界，不是公开 RPC 权限。selection GET 要求交互式 READ，全量管理目录要求交互式 RW；owner 与 Solana 过滤由服务端给出，读取不改凭据，故无需 lease。替换 selection 要求 exact origin 和 expected-revision CAS，范围为 0–20。每次 connect、reconnect、regenerate 或 retirement-disconnect 还需独立五分钟 `worm.api_credential.manage` lease 与 owner 范围内 Solana Wallet 查询。所有管理入口拒绝 API Key；Wallet 签名仅在内部 Wallet Bearer 及精确 purpose-bound challenge 验证后执行。

组合目录/list/detail GET 要求交互式 READ；POST/PUT/DELETE 要求 RW、exact origin、owner 与 revision，但不需要 Wallet 权限或 Worm 凭据 lease，因为只修改账户模板。创建或完整替换前，API Server facade 核验交互式准入，只转发组合、事件等 ID 与可信账户身份元数据；Trading 核验当前 RW、owner 与 revision，重取权威事件目录、校验事件归属和方向，并构造可信展示快照。浏览器标题或 availability 不能作为授权证据。API Key 即使能读 Assets 投影也不能进入此 facade。

Preview 的 owner-scoped plan/step GET 要求交互式 READ；创建 POST 要求 RW 与 exact origin，只接收组合 UUID、expected revision、当前 selection revision，以及按顺序排列的 1–20 个当前已选 Wallet ID，不执行 step-up。API Server 核验每个 Wallet owner 后才持久化异步 preview 请求。浏览器的组合/Wallet 选择和 Refresh 需要 RW；owner 范围内 `?planId=` Review 保持 READ 可用，且不请求管理连接目录。API Key 不能创建或读取 Preview。

Run/Step list/detail 需要交互式 READ。创建、证明身份、coordinator 获取/续约、Start/Pause/Continue/Terminate、选择下一 Step 及 Reconcile 都需要 RW；原生 JSON 命令还需要 exact origin、command UUID、expected Run revision 与当前 owner/session/access 绑定。Google、Phantom 或回环开发 proof 绑定精确 Run、冻结 plan digest、账户、登录 Session-JTI digest 与 access revision，不替代可复用 Worm 凭据 lease。Start/Continue/Heartbeat/Execute Next 要求持久 Run 授权；Pause/Terminate/只读 Reconcile 不能推进 Run。API Key 不能进入任何 Run 路由。浏览器不能提交 Wallet 地址、市场、方向、资金、Worm 凭据、交易或签名。

## 运行流程与原子更新

1. 服务启动在账户 store 创建后立即设置必需 `AccessChangeHook`，先于开发账户初始化和 Controller 加载；缺失通过既有 `errorsutil.CheckError` 启动错误路径处理。读取所有 access head、数据库角色与七模块行；零账户合法，缺失、重复、未知、不完整或无效聚合使启动失败。
2. member realm 中 Google 或 Solana 注册创建 login=true、API Key/Profit Sharing=false、revision=1 与七项 NONE。只有获准 Google 身份在 admin realm 可创建固定管理员聚合。管理员配置邮件进入 member realm 仍创建普通 Pending 身份。Controller 在提交后才发布对应独立 UUID。
3. 每次会话/API Key 请求检查 LoginEnabled，API Key 另查 APIKeyEnabled；Profit Sharing member RPC 查权益且业务层继续核验 round membership；产品 RPC 查显式模块 requirement。敏感原生入口同时执行上节凭据、owner、proof 与 origin 条件。未知已认证 RPC 拒绝，不能绕过规则进入业务代码。
4. 管理员只能对普通账户以 expected revision 完整替换三个 flags 与七模块矩阵。Controller 按账户串行；SQL 使用共享 account gate，在同事务读取真实 previous，执行 head revision CAS 与七行替换，再于 commit 前执行 hook。hook 错误回滚整笔变更；成功后才发布缓存。管理员聚合不能经该路径编辑。
5. `trader_sync` RW→NONE 的真实 hook 在同一账户事务关闭区间、标记非取消订阅 permission_disabled、终止 pending baseline attempt 与未冻结摘要成员，并为全部旧 Trader Sync delivery 写永久资格墓碑（含 sending）；pending 投递取消。已许可 attempt 仍记录真实成功/unknown，明确失败则 cancelled。重授不恢复订阅或旧 attempt。Login/API Key flags 不冒充产品撤权。基线登记、摘要成员与发送许可边界见 [Trader Sync 设计](../trading/trader-sync-activity-alerts.md)。
6. 状态优先为 login=false 的 BLOCKED；非管理员 login=true 且所有模块 NONE、Profit Sharing=false 为 PENDING；其余 ACTIVE。管理员即使没有会员 grant 也是 Active，API Key 单独开启不会让会员 Active，Telegram 状态不参与推导。
7. 管理员目录搜索用户名、Google 验证邮件、Solana 地址、显示名及精确 UUID；支持 All/Pending/Active/Blocked、Profit-Sharing-eligible 过滤、总数与从 1 起的分页，默认 50、最大 100。排序为 Pending 优先，再最近登录、用户名、UUID。
8. 会员浏览器固定 member realm，可见时至多每 15 秒刷新权限，并在 focus/visibility 返回、Pending 手动刷新及稳定权限拒绝后刷新。模块失权中止相应请求、清 member/account/session 范围缓存、忽略晚响应，不可访问路由转 `/account/access`。管理员应用独立固定 admin realm，bootstrap 只读自己 Cookie，在构建 realm services 前拒绝角色不符，不能回退使用另一个 realm 的会话。
9. Pending 会员可使用 Profile、Appearance、Access、Telegram setup、Help、Logout，不启动业务请求；Security 仅在 API Key 开启时出现。首次有已交付 UI 的模块 grant 导航至 canonical 可读模块；仅 Profit Sharing 权益时转 `/profit-sharing`。Token 无业务页面或 landing，其禁用菜单项仅在 Token READ/RW 时显示；仅 Token 权益仍 Active，但不自动进入业务页。Trader Sync 导航和六条路由只在 grant 严格等于 READ_WRITE 时开放；非法 READ 归 NONE，降权立即清业务状态。管理员应用只创建管理员与账户自助/安全概要服务，普通账户在任何管理请求前被拒绝。

管理员编辑器与账户中心使用 `accountAccessDisplayModules` 隐藏 Token 控件、卡片及展示计数；完整 `accountDataModules` 仍驱动解析、克隆、替换、比较、状态与提交，编辑其他权限保留 Token grant。Trader Sync 共享 `allowedAccessLevels` 只给 NONE/RW；解析、编辑、克隆及服务序列化对非法 READ 关闭为 NONE，不提升为 RW。会员壳的 `revokeLostModuleAccess` 使用相同矩阵执行中止和缓存清理。

## 持久状态与不变量

`account_access.account_id UUID` 保存三个 flags 和正 revision；`account_module_access` 主键为 `(account_id,module)`，每账户恰好七行，两者都引用同一 UUID 账户。角色只在父账户持久化并关联读取，授权表不含用户名。

普通账户保留，可禁用登录或改变 grant；没有删除、角色提升、改用户名、重绑外部身份、合并或转移 API。唯一管理员由注册创建，由角色唯一索引和固定聚合校验保护。外部身份键包含 `(identity_provider,identity_subject,administrator)`；同一 Google subject 的两种 persona 不共享 UUID 或从属聚合，用户名依然全局唯一。

必须保持：管理员能力不满足模块或 Profit Sharing member requirement；角色不从用户名推断；每个已认证 RPC 有明确边界；CAS 同时发布全部 flags/模块或全部不发布；所有 owner 由认证上下文提供。模块、API Key 与 Profit Sharing 独立。Run 授权不扩大冻结计划，不替代当前权限、Wallet owner 或 coordinator 排他性。历史 Preview/Run/单次及批量 Cash Out 继续使用自身 owner-scoped READ 边界；新的 selection revision 控制新准入，不删除历史可见性。

追加 `000004_remove_sports_access.sql` 与 `000005_remove_worm_markets_access.sql` 分别在 Goose 事务中精确清除旧授权并收紧模块及最大级别约束；000001–000003 保留原文。`000005` 删除 grant、递增受影响账户 revision，且不授予 `worm_trading`。新库与升级库均为七行聚合且通过 schema contract。发布顺序为全部实际同库消费者退出、authority up、verify、一致新版本启动，不兼容旧代码继续写入；原 main 实际升级证据由 Task 10 补录。

## 配置

权限没有逐账户环境变量，身份、角色与聚合来自 PostgreSQL。`ATHENA_SERVER_DISABLE_AUTH=true` 创建或复用两个回环开发聚合：`local-user` 获全部模块最大权限和 API Key/Profit Sharing，`local-admin` 保持隔离固定管理员权限。每个 realm 在 synthetic claims 中选择对应 UUID，两种应用可同时使用，无需启动时选择角色。正常外部认证拒绝两种开发身份。API Server 拒绝非回环监听的 disabled-auth；生产脚本与 Compose 拒绝或固定该设置为 false。

## 失败恢复

启动非法状态关闭访问。revision 冲突保留数据库与缓存，SQL 或 hook 错误回滚完整事务。注册已经提交但运行时发布或 Cookie 签发失败时，账户仍持久存在，下次已知 subject 登录重载权限后再签发会话。

禁用 API Key 只暂停，不删除元数据；重开仅恢复未删除且未过期的 Key。登录关闭在下次请求优先于其他权益生效。任意 access revision 变化使 Wallet-secret lease 失效，因为每次 reveal 都比较当前 revision；Worm 凭据 lease 在每次连接修改作同样校验。管理目录每次也重新鉴权，浏览器不能凭旧快照继续 selection reconciliation。

组合每次请求重新检查交互凭据与当前 access revision；失权后即使浏览器保留 builder 状态也不能继续目录读取、保存或删除。组合 revision 冲突不改变模板或 items，需重载当前 owner revision。selection 替换使用 owner-scoped CAS；失权、换账户或并发替换保留原 selection。

Preview 的 POST/GET 都重复当前凭据/账户鉴权。失权阻止创建或继续 polling，但不改变持久 Preview。POST 拒绝来源/selection revision 冲突与未选 Wallet；GET 保留 owner 快照可读，对后续来源变化标记不可消费，不从陈旧数据赋权。

Run 每条命令重新核验交互凭据、owner、module、Session、access revision 与 Run revision。失权或 revision 更新阻止新 Step 与 coordinator 续约，不删除 Run、不重放修改。恢复后可通过新鲜 provider proof 重绑符合条件的 Run；撤权不把未知结果改成功，不解除隔离，也不解锁其他账户资源。Trader Sync 的产品墓碑更严格：重授绝不恢复旧 delivery 后续 attempt。

## 可观测性

稳定错误区分维护、管理员要求、模块拒绝、API Key、Profit Sharing 与 revision 冲突。模块拒绝暴露 module/required/effective 元数据。Wallet-secret HTTP 另有 login-session、reauthentication-required、reauthentication-unavailable；Worm 连接使用独立稳定原因，不能把 step-up 拒绝误作全局登录失败。Wallet 乐观 CAS 冲突与权限拒绝分别返回。Run proof 区分 `WORM_EXECUTION_LOGIN_SESSION_REQUIRED`、`WORM_EXECUTION_AUTHORIZATION_REQUIRED` 与 `WORM_EXECUTION_AUTHORIZATION_UNAVAILABLE`；持久 Run 授权保存 Session-JTI digest 和 access revision，后续推进命令不能依赖陈旧授权快照。

日志记录账户 UUID 与授权边界，不输出 identity subject、Wallet 签名、JWT、JTI 或 bearer。管理员目录展示 UUID、用户名、安全 provider 展示信息及时间，不返回 Google subject。Wallet-secret/Worm-management 日志仅含有界 provider/stage/reason，不含私钥、凭据、challenge、签名、lease 或 Session JTI。

组合的拒绝/冲突输出有界，原生响应不返回 owner UUID，也不接受客户端 owner。Preview 只投影安全 Wallet 展示、余额、连接、市场快照、估算、Step 分类、生命周期与稳定 failure/usability code。Run 另返回冻结安全快照、状态/计数、允许动作、当前 Step、数字 provider request ID/state、授权类型、coordinator 状态及有界错误；隐藏 owner UUID、Session-JTI digest、access binding、Worm JWT、原始/已签交易、Wallet 签名与 provider 凭据。coordinator token 只在专用控制响应返回。

## 变更核对

维护本文时同时核对七模块矩阵、三个权益与管理员状态，UUID 注册/CAS/提交后发布，RPC 与 Pending UI，Telegram 的普通会员交互/服务端 UUID 边界，Wallet/Worm typed credential 与 owner 约束，selection/组合/Preview/Run 的 origin、revision、proof、lease 和 API Key 限制，以及管理员目录分页排序。Trader Sync grant 和 requirement 的合法集合必须分开；产品撤权必须保留真实结果与永久墓碑。入口与实现状态在[设计索引](../README.md)保持一致。
