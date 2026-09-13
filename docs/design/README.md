# 后端技术设计

`docs/design/` 保存 ATHENA 面向开发者和 AI 代理的长期架构知识。文档既能表达目标方案，也能说明当前已经运行的设计，应明确区分目标、现状和未决问题，并随代码演进维护。

新任务遵循原版 Superpowers，按其 `brainstorming`、`writing-plans` 等技能开展工作，入口见 [Superpowers 开发工作流](../developer-guide/superpowers-development.md)。复杂任务的规格与实现计划默认放在 `docs/superpowers/specs/` 和 `docs/superpowers/plans/`；本目录承接其中需要长期维护的架构决定，不另外设置需求确认、设计确认和另行派发实现的审批流程。

目录中既有的 `web-ui/` 文档继续作为界面架构说明维护。前后端新任务均使用 Superpowers；具体界面设计仍遵循根 `AGENTS.md` 中适用的 UI 约定。

全站 UI 的[视觉主题重构目标](../requirements/web-ui/visual-theme.md)已确认采用单一深色、近黑背景与青绿色强调，[v1 配色与按钮层级](../requirements/web-ui/previews/README.md)、[v2 字体与数字排版](../requirements/web-ui/typography-proposal.md)、[v3 导航／页头／页面密度](../requirements/web-ui/layout-proposal.md)的视觉效果均已确认；下一项为通用组件及其状态，其他业务及管理员页面仍按范围设计，正式重构尚未实现。下列已实现文档中的明暗模式、橙色主题和账户主题偏好继续描述当前代码；后续按新目标实施时同步替换相关设计。重构的完整技术设计与执行计划尚未形成。

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
| Identity and Access | Realm-bound browser Google Authorization Code flow, PKCE, one-time OAuth state, administrator admission, anonymous username registration, and independent member/admin cookie issuance | [Google OIDC Login](identity-access/google-oidc-login.md) | `已实现` |
| Identity and Access | Browser-injected Phantom Solana authentication, one-time SIWS challenges, Ed25519 verification, and wallet-first username registration | [Solana Wallet Authentication](identity-access/solana-wallet-authentication.md) | `已实现` |
| Identity and Access | Realm-to-persisted-role session binding, credential-specific Wallet and Worm-selection rules, login, API Key, Profit Sharing, eleven-module access, Pending state, and transactional administrator control | [Account Access Control](identity-access/account-access-control.md) | `已实现` |
| Identity and Access | UUID-owned display profiles, immutable username presentation, display-only tiers, and cross-device theme preferences | [Account Profile and Preferences](identity-access/account-profile-and-preferences.md) | `已实现` |
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
