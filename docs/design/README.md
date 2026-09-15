# 后端技术设计

`docs/design/` 保存 ATHENA 面向开发者和 AI 代理的长期架构知识。文档既能表达目标方案，也能说明当前已经运行的设计，应明确区分目标、现状和未决问题，并随代码演进维护。

新任务遵循原版 Superpowers，按其 `brainstorming`、`writing-plans` 等技能开展工作，入口见 [Superpowers 开发工作流](../developer-guide/superpowers-development.md)。复杂任务的规格与实现计划默认放在 `docs/superpowers/specs/` 和 `docs/superpowers/plans/`；本目录承接其中需要长期维护的架构决定，不另外设置需求确认、设计确认和另行派发实现的审批流程。

目录中既有的 `web-ui/` 文档继续作为界面架构说明维护。前后端新任务均使用 Superpowers；具体界面设计仍遵循根 `AGENTS.md` 中适用的 UI 约定。

全站 UI 已按 Nansen 单一深色、Inter／JetBrains Mono、v1–v22 批准视觉及 [v23 一致性契约](../requirements/web-ui/theme-consistency-contract.md)完成正式实现和 T9 验收；T1–T9 独立任务审阅通过。37 条改版入口、2 条 Appearance 删除、4 条路由规则及 8 条 Trader Sync 共享影响回归见[覆盖清单](../requirements/web-ui/theme-refactor-coverage.md)。Service Status 的 Trader Sync 页签已重排；专属八路由布局暂缓。共享 T1／T2 已完成，Token 独立功能未开启。

当前设计文档描述固定深色和 realm 独立的本地视图偏好，账户主题 API／schema 已删除。[验收记录](../testing/web-ui-theme-refactor-acceptance.md)区分正式实现、受控状态、真实读取和外部未验证边界；最终审阅／环境停止／通知状态集中在该记录。下列 v1–v23 提案条目保留批准时点历史，不代表当前代码仍未实施。技术约束见[方案](../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)与[实施计划](../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)。

聊天记录不是跨任务事实来源。长期有效的技术决定应同步到对应设计文档，并与 [`docs/requirements/`](../requirements/README.md) 中的相关需求及任务规格按需互相链接。

## 设计状态

已有设计状态继续描述方案和代码的实际情况，不作为启动工作的审批开关，不要求按固定顺序流转，也不替代 Superpowers 的工作流程。

| 状态 | 含义 |
| --- | --- |
| `设计中` | 正在形成或修改目标技术方案，仍有未决内容 |
| `已确认待实现` | 目标技术方案已经获得确认，但代码尚未完全实现 |
| `已实现` | 文档描述的设计已经由当前代码实现 |

组件边界、接口或数据契约、数据模型、事务与并发、核心流程或基础设施变化时，更新对应设计和未决问题；如果原状态已不能准确反映现状，同时更新状态。不能因为工作流切换而代为确认尚在讨论的方案。

可执行代码始终是“当前运行行为”的事实来源。`设计中` 和 `已确认待实现` 文档必须在“现状与目标差距”中清楚区分当前实现与目标方案；达到 `已实现` 后，正文只保留当前有效设计，直接替换过时内容，不保存迁移历史、废弃方案或未来计划。

## 使用与维护

- 按 [Superpowers 开发工作流](../developer-guide/superpowers-development.md) 处理新任务，将本索引和相关能力文档作为上下文。
- 阅读受影响能力的业务需求、设计文档及现有任务规格，区分已决定事项和未决问题。
- 沿源码链接检查真实入口、边界和依赖，用可执行代码核实当前行为。
- 同一任务内将长期有效的方案和实际源码位置同步到相关文档，避免任务规格与长期说明互相矛盾。

## 文档地图

| 子系统 | 能力 | 文档 | 状态 |
| --- | --- | --- | --- |
| Development Runtime | 按实例正向选择独立服务、持久基础设施、gRPC 与同库事务、精确归属和有界停止、schema/TLS 部署 | [本地运行编排](development-runtime/local-runtime-orchestration.md) | 独立运行器与全栈已实现；持久重启及真实验收通过，[验收记录](../testing/trader-sync-independent-service-acceptance.md) |
| Developer Experience | Public LLM discovery documents, one-time Connect AI instructions, Swagger generation and embedding, unauthenticated documentation delivery, and root-path and safety boundaries | [AI Discovery Documentation](developer-experience/ai-discovery-documentation.md) | `已实现` |
| Developer Experience | Explicit task-completion email delivery through fixed Tencent Exmail transport and recipient boundaries, invocation content, configuration precedence, and bounded retries | [Task Completion Email](developer-experience/task-completion-email.md) | `已实现` |
| Identity and Access | UUID account identities, immutable public usernames, realm-scoped provider bindings, independent member/admin login cookies, persistent API Keys, Athena JWT v3, typed request credentials, and interactive-only sensitive boundaries | [Account Credentials](identity-access/account-credentials.md) | `已实现` |
| Identity and Access | 同一账号只保留一个有效客户端登录，新登录成功后旧登录立即失效 | [单客户端登录目标设计](identity-access/single-client-login.md) | `设计中`；[业务规则](../requirements/identity-access/single-client-login.md)已确认，当前会话校验及交易接收边界待落实，代码未实现 |
| Identity and Access | Realm-bound browser Google Authorization Code flow, PKCE, one-time OAuth state, administrator admission, anonymous username registration, and independent member/admin cookie issuance | [Google OIDC Login](identity-access/google-oidc-login.md) | `已实现` |
| Identity and Access | Browser-injected Phantom Solana authentication, one-time SIWS challenges, Ed25519 verification, and wallet-first username registration | [Solana Wallet Authentication](identity-access/solana-wallet-authentication.md) | `已实现` |
| Identity and Access | Realm-to-persisted-role session binding, credential-specific Wallet and Worm-selection rules, login, API Key, Profit Sharing, eleven-module access, Pending state, and transactional administrator control | [Account Access Control](identity-access/account-access-control.md) | `已实现` |
| Identity and Access | UUID-owned display profiles, immutable username presentation, display-only tiers, and realm-scoped browser-local view preferences | [Account Profile and Local View Preferences](identity-access/account-profile-and-preferences.md) | `已实现` |
| Identity and Access | Private account and Wallet avatar validation, S3-compatible object storage, distinct authorization, authenticated delivery, and orphan recovery | [Account and Wallet Avatar Storage](identity-access/account-avatar-storage.md) | `已实现` |
| Identity and Access | UUID-owned EVM and Solana custody, canonical key handling, owner-only safe metadata, persisted owner-resolved Worm Wallet selection, and separate purpose-bound Worm credential and live-execution signers | [Wallet Ownership and Custody](identity-access/wallet-ownership.md) | `已实现` |
| Identity and Access | Independent Wallet-reveal and Worm-credential leases plus selection-reconciliation, exact-Run, and exact-position-Cash-Out proof boundaries; rate-limited Google/Solana reauthentication; and durable intent-bound authorization | [Wallet Secret and Worm Credential Reauthentication](identity-access/wallet-secret-reauthentication.md) | `已实现` |
| Web UI | Shared bootstrap/session kernel, deployment-root and application-root separation, two HTML/React entry points, realm selection, and cross-realm cleanup | [Application Shell](web-ui/application-shell.md) | `已实现` |
| Web UI | 会员登录、Pending access、十一模块导航、Solana 列表、Telegram 绑定、Trader Sync 六路由与可见页单飞刷新、Account Center、API Keys、Profit Sharing 和 Wallet | [会员应用壳](web-ui/member-application-shell.md) | `已实现` |
| Web UI | 管理员登录/角色复查、账户授权、Trader Sync 安全概要、三来源 Service Status、系统通知和管理员自助 | [管理员应用壳](web-ui/administrator-application-shell.md) | `已实现` |
| Notifications | 账户 Telegram 绑定、幂等投递、Trader Sync 普通/摘要通知、Add 草稿往返、binding revision fencing 与不可达恢复 | [账户 Telegram 通知](notifications/account-telegram-notifications.md) | `已实现` |
| Notifications | 认证运维生产者、Telegram Topics、持久系统投递、account/system/reply 与 summary-head 协调调度、sender 恢复及管理员 Service Status | [系统通知运维](notifications/system-notification-operations.md) | `已实现` |
| Governance | UUID membership, immutable participant-name snapshots, entitlement-gated profit-allocation rounds, proposals, voting, and runoff resolution | [Profit Sharing](governance/profit-sharing.md) | `已实现` |
| Market Intelligence | Read-only Market Radar module, Polymarket discovery, rolling price windows, mover ranking, and alerts | [Market Radar](market-intelligence/market-radar.md) | `已实现` |
| Market Intelligence | Current Polymarket sports synchronization, price history, and price/score alerts | [Sports Live](market-intelligence/sports-live.md) | `已实现` |
| Market Intelligence | Completed ATP/WTA event synchronization, price history, status, and manual refresh | [Sports History](market-intelligence/sports-history.md) | `已实现` |
| Market Intelligence | Managed Optimistic Oracle log ingestion, market enrichment, reads, scans, and alerts | [Managed OO](market-intelligence/managed-oo.md) | `已实现` |
| Market Intelligence | Worm sports-market synchronization, rules, live state, history, and alerts | [Worm Markets](market-intelligence/worm-markets.md) | `已实现` |
| Trading | Trader Sync 实时活动、订阅基线、中断可见、摘要/发送许可、独立 gRPC 与共享事务边界 | [Activity Alerts 后端设计](trading/trader-sync-activity-alerts.md) | 独立服务已实现；最终验收通过，[结果与证据限制](../testing/trader-sync-independent-service-acceptance.md) |
| Trading | Trader Sync 站内手动交易的服务职责、钱包账户映射、提交核对与页面组织 | [手动交易总体设计提案](trading/polymarket-manual-trading.md) | `设计中`；页面组织、主要交互、一个独立交易服务的分工及账户接入顺序已确认，单客户端登录的新登录替换规则、共用 Trader Sync 权限、交易结果仅在页面／历史展示及目标卖出活动的对应持仓入口也已确认，会话与交易接收边界、接口及私有链路继续细化，代码未实现 |
| Web UI | Trader Sync 下单确认、持仓卖出／领取、系统记录与成交历史的交互提案 | [手动交易页面设计](web-ui/trader-sync-manual-trading.md) | `设计中`；2026-09-15 页面组织、报价核对与同页提交、持仓分组和历史两个视图的主要交互已逐项确认；其余字段状态与正式视觉继续细化，交易 UI 未实现 |
| Web UI | Trader Sync 活动与目标同屏、独立添加/管理/活动/摘要、稳定刷新、Telegram 衔接、管理员安全概要与受控浏览器验收 | [Activity Alerts UI 设计](web-ui/trader-sync-activity-alerts.md) | `已实现` |
| Trading | Revisioned owner-scoped selection of up to 20 Solana Wallets, removal-first official-HMAC connection/activity flows, saved combinations and previews, Cash Out, and official Web JWT live-Run orchestration | [Worm Trading](trading/worm-trading.md) | `已实现` |
| Trading | Provider-backed Worm event catalogs and owner-scoped, revisioned market-combination CRUD with trusted display snapshots | [Worm Market Combinations](trading/worm-market-combinations.md) | `已实现` |
| Trading | Durable asynchronous read-only execution previews limited to the current selected Wallet revision, with frozen Wallet and market order, authoritative Worm exposure and estimates, cumulative USDC simulation, and expiring owner-scoped review | [Worm Execution Preview](trading/worm-execution-preview.md) | `已实现` |
| Trading | Current-selection-gated Run admission, Run-bound Worm Web JWT execution, purpose-bound custodial signing, browser-led Wallet-major coordination, durable mutation attempts, reconciliation, and permanent owner-scoped history | [Worm Order Execution](trading/worm-order-execution.md) | `已实现` |
| Trading | Current-selection exact-position, fresh-proof-authorized Worm HMAC Cash Out with whole-position market Close, at-most-once dispatch, read-only recovery, and durable historical detail | [Worm Position Cash Out](trading/worm-position-cash-out.md) | `已实现` |
| Trading | Up-to-20 selected-Wallet, Wallet-major serial Cash Out batches with complete position freezing, single-operation reuse, confirmed-USDC advancement gates, durable Wallet locks, and manual recovery controls | [Worm Position Cash Out Batches](trading/worm-position-cash-out-batches.md) | `已实现` |
| Token Intelligence | 第一板块设计包：已确认时效、架构选择、跨子系统契约与容量验收 | [Token 第一板块主 spec](../superpowers/specs/2026-09-10-token-first-block-design.md) | `设计中`，分节已确认，书面待审阅 |
| Solana Intelligence | 首版范围、后续决定、集成与暂停采集状态 | [Solana 设计总览](solana-intelligence/README.md) | 首版源分支已验收；已集成到 `rf4` 并通过[集成验收](../testing/rf4-branch-integration.md)，采集继续暂停，数据保留 |
| Solana Intelligence | finalized 新 Mint 发现、持久游标、服务与授权 | [项目发现](solana-intelligence/project-discovery.md) | 已实现、已验收；`rf4` 本次集成验证已通过 |
| Solana Intelligence | 名称、符号、平台归因、字段合同与首次补全重试 | [基础信息补全与发行来源](solana-intelligence/candidate-metadata.md) | 已实现、已验收；`rf4` 本次集成验证已通过，历史队列未全部补完 |
| Web UI | Solana 独立列表、查询、资料状态和链上详情 | [Solana 列表页](web-ui/solana-discovery.md) | 首版功能已确认并验收；最终视觉遵循全站主题设计 |
| Solana Intelligence | 发行后持续研究、活动触发与每项目一小时限频 | [持续研究与刷新调度](solana-intelligence/research-lifecycle-and-refresh.md) | 后续方向已确认，调度细节为建议，尚未实现 |
| Token Intelligence | 有界发现、连续覆盖、有序活动、owner 观察屏障、七协议 Swap 及停止 | [发现与研究生命周期](token-intelligence/discovery-research-lifecycle.md) | `设计中`，尚未实现 |
| Token Intelligence | PostgreSQL 持久依赖任务、独立事实、刷新、共享配额和失败恢复 | [研究任务运行时](token-intelligence/research-task-runtime.md) | `设计中`，尚未实现 |
| Token Intelligence | 非空源码、五项共享 AI 产物与声明式部署读取 | [源码事实与读取方案](token-intelligence/source-facts-and-read-plans.md) | `设计中`，尚未实现 |
| Token Intelligence | 部署前双历史、内部归因、历史 L1、五层图和六资产同块估值 | [钱包证据与估值](token-intelligence/wallet-evidence-and-valuation.md) | `设计中`，尚未实现 |
| Token Intelligence | 分项事实、历史证据、进度、分页及管理员命令 | [研究查询与管理](token-intelligence/research-query-and-administration.md) | `设计中`，尚未实现 |
| Token Intelligence | Synchronous EVM block discovery, token validation, project initialization, and per-attempt processing diagnostics | [Token Chain Processor](token-intelligence/chain-processor.md) | `已实现` |
| Token Intelligence | On-chain ERC-20, pair, wallet, and simulation-state aggregation | [ATHENA EVM Aggregator Contract](token-intelligence/athena-contract.md) | `已实现` |
| Token Intelligence | PostgreSQL-backed six-source one-time collection, fenced workers, terminal barrier, and immutable ProjectProfile construction | [Token Collection and Project Profile](token-intelligence/collection-profile.md) | `已实现` |
| Token Intelligence | Ave token market data and canonical pair collection | [Ave Market Data Collection](token-intelligence/ave-market-data.md) | `已实现` |
| Token Intelligence | Project list/detail, collection evidence, immutable profile, wallet, and contract-source reads | [Token Project Read Model](token-intelligence/project-read-model.md) | `已实现` |
| Token Intelligence | One-time pre-deployment normal transactions for related wallets | [Project Wallet Pre-Deployment Normal Transactions](token-intelligence/wallet-normal-transactions.md) | `已实现` |
| Token Intelligence | Nansen 钱包总览、逐币表现与交易的请求适配、共享缓存及调用管理 | [钱包数据展示](token-intelligence/wallet-analytics.md) | [后端草案](../superpowers/specs/2026-09-14-token-wallet-analytics-backend-design.md)已形成，[页面设计 v1](../requirements/token/wallet-analytics-page-proposal.md)已获用户确认，[实施计划](../superpowers/plans/2026-09-14-token-wallet-analytics.md)已整理，尚未实现 |
| Token Intelligence | Token-module READ/READ_WRITE API authorization, retained grants, and the disabled member navigation entry | [Token Module Access Control](token-intelligence/access-control.md) | `已实现` |
| Blockchain Data | Finalized inbound BSC transaction indexing and lookup | [BSC Inbound Normal Transactions](blockchain-data/bsc-inbound-normal-transactions.md) | `已实现` |
| Blockchain Data | Finalized BSC V2 Swap-topic transaction indexing and lookup | [BSC V2 Swap Transactions](blockchain-data/bsc-v2-swap-transactions.md) | `已实现` |
| Blockchain Data | Etherscan API-key and Gateway request scheduling | [Etherscan Manager](blockchain-data/etherscan-manager.md) | `已实现` |

需要记录新的长期架构知识时使用[设计模板](template.md)；单次任务规格和实现计划采用 Superpowers 的对应格式与默认目录。

## 语言约定

新技术设计以简体中文为主。现有英文设计不为本次规则迁移而批量翻译；当某份文档进入实质设计周期时，在同一周期中将整份正文迁移为简体中文，避免长期维护双语混合段落。源码符号、协议名和其他专有名词保留其准确写法。

## 维护原则

- 本目录按稳定子系统和能力组织文档；单次任务的规格与计划保存在 `docs/superpowers/` 下。
- 存在对应长期需求时互相链接；相关 Superpowers 规格与计划按需链接。没有独立需求文档的能力无需虚构链接。
- 按需写清需求覆盖、现状差距、关键决定、组件与契约、数据与流程、事务并发、失败恢复、配置安全、可观测性、源码影响、验证方式和待确认问题；不适用的部分明确说明即可。
- 链接真实源码路径并命名关键符号，不复制大段实现。设计完成前可以记录预计影响路径，实现完成后改为实际位置。
- 实现者在同一任务内同步受影响的设计文档。设计维护不是编码后的独立补写任务。
- 代码与 `已实现` 文档不一致时，先判断偏差来源。已授权的实现或修复任务应在同一任务修复代码偏差或过时文档；单独审查任务只报告差异，等待明确修复授权。
- 根据实际影响维护文档；纯格式化、注释、文案，以及生成源、契约和设计语义均未变化的生成文件刷新等机械修改，不要求更新设计文档。

## 新增文档

- 选择稳定的子系统目录和能力导向文件名，按需使用[设计模板](template.md)。
- 基于实际代码及任务规格记录当前设计、目标和差距，并按事实填写状态。
- 补充相关需求、任务规格和源码链接，登记到上方地图。
- 检查相对链接并删除所有占位说明；本节仅说明文档组织方式，不定义任务阶段。

## v16 索引补充

| 子系统 | 能力 | 文档 | 状态 |
| --- | --- | --- | --- |
| Web UI | 管理员 Service Status 三来源页签、来源状态、完整诊断及响应式记录布局 | [v16 视觉提案](../requirements/web-ui/service-status-proposal.md) | `展示视觉已确认`，v16 所展示的 Services 桌面／手机、Notifications 桌面和 Trader Sync 手机视觉已确认，按[确认记录](../requirements/web-ui/previews/theme-service-status-v16-approval.json)执行；不改变已实现的[管理员应用壳](web-ui/administrator-application-shell.md)三来源 10 秒可见 single-flight、缓存、时间、错误隔离和授权契约，批准时正式 `ui/` 未修改 |

## v17 索引补充

| 子系统 | 能力 | 文档 | 状态 |
| --- | --- | --- | --- |
| Web UI | 管理员 Etherscan 网关健康与 Live Probe 独立来源、结果汇总、诊断展开及响应式记录布局 | [v17 视觉提案](../requirements/web-ui/etherscan-gateways-proposal.md) | `展示视觉已确认`，v17 所展示 Gateways 与 Live Probe 的桌面／手机视觉已确认，按[确认记录](../requirements/web-ui/previews/theme-etherscan-gateways-v17-approval.json)执行；其他辅助状态未逐图确认，不改变已实现的[管理员应用壳](web-ui/administrator-application-shell.md)与 [Etherscan Manager](blockchain-data/etherscan-manager.md)权限、配置、探针请求或服务端结果契约，批准时正式 `ui/` 未修改 |

## v18 索引补充

| 子系统 | 能力 | 文档 | 状态 |
| --- | --- | --- | --- |
| Web UI | 管理员系统通知列表筛选、详情阅读、测试通知弹窗与响应式记录布局 | [v18 视觉提案](../requirements/web-ui/system-notifications-proposal.md) | `展示视觉已确认`，所展示列表／详情及桌面／手机视觉已确认；常规辅助状态由实现者验证，不新增逐图审批；不改变已实现的[管理员应用壳](web-ui/administrator-application-shell.md)与[系统通知操作](notifications/system-notification-operations.md)权限、投递或数据契约，批准时正式 `ui/` 未修改 |

用户已对四张主图反馈“舒服”，具体范围见 [v18 确认记录](../requirements/web-ui/previews/theme-system-notifications-v18-approval.json)。原始审阅与资产保持不变；本次认可不表示正式 UI 已实施。

## v19 索引补充

| 子系统 | 能力 | 文档 | 状态 |
| --- | --- | --- | --- |
| Web UI | 管理员 Trader Sync 订阅列表／概要与 Profit Sharing 轮次列表／详情的集中视觉批次 | [v19 视觉提案](../requirements/web-ui/admin-business-batch-proposal.md) | `设计中`，四页八张桌面／手机主图所展示视觉已确认；辅助状态由实现者验证，不新增逐图审批；不改变已实现的 [Trader Sync](web-ui/trader-sync-activity-alerts.md) 与 [Profit Sharing](governance/profit-sharing.md)权限、数据和治理契约，批准时正式 `ui/` 未修改 |

用户于 2026-09-14 明确确认本批无须调整，范围见 [v19 确认记录](../requirements/web-ui/previews/theme-admin-batch-v19-approval.json)。原始原型、截图与审阅记录保持不变；辅助状态按既定规则验证，不追加逐图审批，正式前端后续按本批视觉基准实施。

## v20 与其余页面安排

| 子系统 | 能力 | 文档 | 状态 |
| --- | --- | --- | --- |
| Web UI | Wallets 私有元数据、Solana 发行候选、会员 Profit Sharing 轮次列表与阶段详情的集中视觉批次 | [v20 视觉提案](../requirements/web-ui/member-foundations-batch-proposal.md) | `展示视觉已确认`，四页八张主图已整批确认，见 [v20 确认记录](../requirements/web-ui/previews/theme-member-foundations-v20-approval.json)。沿用现有[钱包托管](identity-access/wallet-ownership.md)、[Solana 列表](web-ui/solana-discovery.md)和 [Profit Sharing](governance/profit-sharing.md)契约，批准时正式主题未实现 |
| Web UI | 本轮其余业务页面与共用适配的视觉覆盖 | [其余页面视觉定稿安排](../requirements/web-ui/remaining-pages-plan.md) | 18 页／19 条路由，分 4／8／6 三批；三批主视觉及 v22 六项共用适配均已整批确认，总路由与继承状态归属已映射，批准时正式重构尚未开始 |
| Web UI | Market Radar 三页、Sports 三页与 Managed OO 两页的集中视觉批次 | [v21 视觉提案](../requirements/web-ui/market-intelligence-batch-proposal.md) | `展示视觉已确认`，八页十六张主图已整批确认，见 [v21 确认记录](../requirements/web-ui/previews/theme-market-intelligence-v21-approval.json)；辅助状态继续覆盖，批准时正式 UI 实施与真实业务验收尚未完成 |
| Web UI | Worm 资产、组合与执行流程及管理员自助／Help 的集中视觉批次 | [v22 视觉提案](../requirements/web-ui/worm-and-common-batch-proposal.md) | `展示视觉已确认`，六个业务页面及六项共用界面差异的 24 张主图已整批确认，见 [v22 确认记录](../requirements/web-ui/previews/theme-worm-and-common-v22-approval.json)；反馈属于状态覆盖，不追加逐图审批。总覆盖与实施计划已整理，批准时正式重构及真实交易尚未验收 |

Trader Sync 本轮排除；独立确认的 [Nansen 钱包战绩 v1](../requirements/token/wallet-analytics-page-proposal.md)直接沿用。本批静态原型、合成 GET 对照与视觉审阅不证明真实身份、钱包写入、采集、治理命令、权限或全栈验收通过。
