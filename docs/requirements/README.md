# 需求目标设计

`docs/requirements/` 保存 ATHENA 长期维护的业务需求与目标行为。这里回答“为什么做、为谁做、系统应表现成什么样”，不提前规定组件拆分、数据库表、RPC 字段或其他实现方案。

新任务遵循原版 Superpowers，按其 `brainstorming`、`writing-plans` 等技能开展工作，入口见 [Superpowers 开发工作流](../developer-guide/superpowers-development.md)。复杂任务的规格与实现计划默认放在 `docs/superpowers/specs/` 和 `docs/superpowers/plans/`；本目录承接其中需要长期维护的业务知识，不另外设置需求、技术设计和实现的审批流程。

聊天记录不是跨任务事实来源。会影响后续工作的业务决定应同步到对应长期文档，并按需链接任务规格。明确区分已经决定的内容、仍在讨论的问题和当前实现，不能因工作流切换而将未决业务标记为已确认。

## 状态

已有需求状态继续描述内容的确定程度，维护时按事实更新；它们不是启动设计或实现的审批开关，也不替代 Superpowers 的工作流程。

| 状态 | 含义 |
| --- | --- |
| `讨论中` | 目标、范围、业务规则或验收行为仍有待决定的内容 |
| `已确认` | 文档记录的业务决定已经获得确认；不表示相关代码已经实现 |

业务规则、权限、状态、范围或可观察行为变化时，更新相应内容及待决问题；如果原状态已不能准确反映确定程度，同时更新状态。措辞和排版调整不改变业务决定。

## 内容范围

按能力需要记录：

- 背景、目标和非目标；
- 使用者、角色与权限边界；
- 业务术语、规则和必须保持的不变量；
- 主流程、关键状态变化、异常与边界场景；
- 对外可见的输入、输出和成功标准；
- 已确认决定与待确认问题，以及未决问题的影响。

需求文档可以描述现状问题和外部约束；后端组件、数据模型、事务、并发或基础设施等长期技术知识写入 [`docs/design/`](../design/README.md)，任务中的方案讨论保存在对应 Superpowers 规格中。

## 组织与维护

- 按稳定业务域和能力组织文档，不按日期、聊天、任务、Issue、PR 或版本建立一次性文件。
- 一个大型能力可以按独立业务主题拆成多份需求；各文档按实际情况维护状态。
- 新文档从[需求模板](template.md)开始，并在本索引或所属业务域索引中登记。
- 已有对应需求和技术设计时互相链接；相关 Superpowers 规格与计划按需链接。没有独立技术设计时如实说明，不为满足流程而补造文档。
- 不删除仍然有效的业务规则；发生实质变化时直接更新目标内容，不把聊天或提交历史复制进正文。

## 文档地图

| 业务域 | 入口 | 状态 | 关联技术设计 |
| --- | --- | --- | --- |
| Observability | [用户关键操作日志](observability/key-operation-logs.md) | `已确认`；2026-09-18 详细方案整体审阅通过，按用户要求暂不进入实施 | [日志技术设计](../design/observability/operation-logs.md)、[详细规格](../superpowers/specs/2026-09-18-key-operation-logs-design.md) |
| Service Operations | [删除 Worm Markets，仅保留 Worm Trading](development-runtime/worm-markets-removal.md) | `已实施并完成原 main 退役`；按需市场目录及权限复核归 Trading，Markets 源码、运行入口、五来源待发通知和专属数据库已退役，正常重启未重建 | [内聚与退役设计](../superpowers/specs/2026-09-16-worm-trading-market-query-design.md)、[实施计划](../superpowers/plans/2026-09-16-worm-trading-market-query.md)、[退役与跨层验收](../testing/worm-markets-retirement-acceptance.md) |
| Service Operations | [板块访问开关：简化方案](development-runtime/business-access-control.md) | `已实现，真实环境证据已记录`；六键首次缺行 CLOSED，管理员设置与审计重启保持，只拒绝新的用户业务请求，程序、后台任务和通知继续运行；Token 接入延期，三服务内部鉴权不在本轮范围 | [2026-09-17 修订方案](../superpowers/specs/2026-09-17-full-stack-access-reassessment-design.md)、[全栈验收](../testing/full-stack-access-acceptance.md)、[当前本地编排](../design/development-runtime/local-runtime-orchestration.md)、[管理员应用壳](../design/web-ui/administrator-application-shell.md)；[原整组运行需求](development-runtime/business-group-control.md)继续暂停，旧访问设计仅保留批准时点 |
| Blockchain Data | [两个 BSC 索引器删除需求](blockchain-data/bsc-indexer-removal.md) | `代码删除已实施`；已移出本地及生产目标服务清单，现场状态见验收记录 | [删除与清理配套设计](../superpowers/specs/2026-09-16-module-removal-cleanup-design.md)自查及补充已完成，[删除清理实施计划](../superpowers/plans/2026-09-16-module-removal-cleanup.md)已整理，四库直接删除策略已批准，逐环境实际结果见[验收记录](../testing/module-removal-cleanup-acceptance.md)；[普通交易索引现状](../design/blockchain-data/bsc-inbound-normal-transactions.md)、[Swap 索引现状](../design/blockchain-data/bsc-v2-swap-transactions.md) |
| Service Operations | [Sports 删除与 Worm 边界](development-runtime/sports-removal.md) | `代码清理已实施`；现场状态见[验收记录](../testing/module-removal-cleanup-acceptance.md)；该任务当时保留 Worm Markets／Trading，随后 Markets 由独立任务退役，Trading 与共用 Worm 能力继续保留；后续 Markets 结果不计入旧 Sports 验收 | [删除与清理配套设计](../superpowers/specs/2026-09-16-module-removal-cleanup-design.md)、[Sports 现状](../design/market-intelligence/sports-live.md)、[界面现状](../design/web-ui/market-intelligence.md) |
| Service Operations | [本地与生产服务清单核对](development-runtime/service-inventory-review.md) | `本地十一应用已实现`；五核心／六业务、五数据库、局部 selected-only 与五 Gateway 只读健康已验收；生产 Compose 和 Token 旧九角色继续按实际现状分列 | [2026-09-17 修订方案](../superpowers/specs/2026-09-17-full-stack-access-reassessment-design.md)、[当前运行设计](../design/development-runtime/local-runtime-orchestration.md)、[全栈验收](../testing/full-stack-access-acceptance.md) |
| Web UI | [全站视觉主题重构](web-ui/visual-theme.md) | `正式实现、T9 验收及本地 rf4 整合完成`，T1–T10 及整分支独立审阅通过；37 改版／2 删除／4 规则／8 Trader Sync 共享回归。最终审阅见[实施交付状态](../testing/web-ui-theme-refactor-acceptance.md#交付状态与环境收尾)，合并、fresh 验证、证据同步与新实例收尾见[整合记录](../testing/rf4-ui-theme-integration.md)；[覆盖](web-ui/theme-refactor-coverage.md)、[方案](../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)、[计划](../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)明确范围。共享 T1／T2 已完成，Token 独立接入未开启。 | [共享壳](../design/web-ui/application-shell.md)、[会员壳](../design/web-ui/member-application-shell.md)、[管理员壳](../design/web-ui/administrator-application-shell.md)、[账户资料与本地偏好](../design/identity-access/account-profile-and-preferences.md) |
| Web UI | [总体审阅一致性修订](web-ui/theme-consistency-contract.md) | v23 最终颜色、字号、组件状态、金额列与管理员图标规则已随主题重构正式实施并整合到本地 `rf4`；设计、两份实施计划与 [v23 样板证据](web-ui/previews/theme-consistency-v23/README.md)保留 | [全站技术方案](../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)、[整合记录](../testing/rf4-ui-theme-integration.md) |
| Identity and Access | [单客户端登录](identity-access/single-client-login.md) | `已确认`；同一账号只允许一个有效客户端，新登录成功后旧登录立即失效，代码未实现 | [目标设计](../design/identity-access/single-client-login.md)及[完整规格第 4 节](../superpowers/specs/2026-09-15-trader-sync-manual-trading-design.md#4-单客户端登录与权限)已整理当前会话、交换恢复与交易接收顺序，完整规格待整体审阅 |
| Trader Sync | [产品需求](polymarket-copy-trading/README.md) | Activity Alerts 与[手动交易需求](polymarket-copy-trading/copy-trading.md)均 `已确认`；手动交易代码未实现 | Activity Alerts [后端](../design/trading/trader-sync-activity-alerts.md)与[UI](../design/web-ui/trader-sync-activity-alerts.md)已实现，见[验收记录](../testing/trader-sync-activity-alerts-acceptance.md)；手动交易三段详细设计及[桌面／手机页面方案](web-ui/trader-sync-manual-trading-proposal.md)已确认，[完整规格](../superpowers/specs/2026-09-15-trader-sync-manual-trading-design.md)已整理、待整体审阅；[平台校验](polymarket-copy-trading/manual-trading-contract-verification.md)已有官方契约与公开样本，账户入金一致性、私有执行与完整覆盖待实测 |
| Token Intelligence | [Token 业务设计与研究资料](token/README.md) | `讨论中`（其中部分独立需求已确认） | [Token Intelligence 当前及目标技术设计](../design/README.md) |
| Solana Intelligence | [Solana 项目发现与研究](solana/README.md) | 首版范围已确认、源分支已验收并集成到 `rf4`；旧 `solana-preview` profile 现场继续暂停并保留数据，2026-09-17 明确授权的新 managed 实例已纳入十一应用全栈并完成当前主网 v1 只读与后台连续性验证；研究细节延后 | [设计总览](../design/solana-intelligence/README.md)、[项目发现](../design/solana-intelligence/project-discovery.md)、[列表页](../design/web-ui/solana-discovery.md)、[全栈验收](../testing/full-stack-access-acceptance.md) |

## v16 索引补充

以下提案条目记录当时批准材料；当前正式实现与实际验收统一见全站主题交付记录。

| 业务域 | 入口 | 状态 | 关联技术设计 |
| --- | --- | --- | --- |
| Web UI | [管理员 Service Status 视觉提案](web-ui/service-status-proposal.md) | `展示视觉已确认`（v16 所展示的 Services 桌面／手机、Notifications 桌面和 Trader Sync 手机视觉已确认，按[确认记录](web-ui/previews/theme-service-status-v16-approval.json)执行；其他辅助状态未逐图确认，批准时正式 `ui/` 未修改） | 现有[管理员应用壳](../design/web-ui/administrator-application-shell.md)；三来源请求、缓存、时间与授权契约保持不变 |

## v17 索引补充

| 业务域 | 入口 | 状态 | 关联技术设计 |
| --- | --- | --- | --- |
| Web UI | [管理员 Etherscan Gateways 视觉提案](web-ui/etherscan-gateways-proposal.md) | `展示视觉已确认`（v17 所展示 Gateways 与 Live Probe 的桌面／手机视觉已确认，按[确认记录](web-ui/previews/theme-etherscan-gateways-v17-approval.json)执行；其他辅助状态未逐图确认，批准时正式 `ui/` 未修改） | 现有[管理员应用壳](../design/web-ui/administrator-application-shell.md)与 [Etherscan Manager](../design/blockchain-data/etherscan-manager.md)；权限、配置、真实请求和服务端结果契约保持不变 |

## v18 索引补充

| 业务域 | 入口 | 状态 | 关联技术设计 |
| --- | --- | --- | --- |
| Web UI | [管理员系统通知列表与详情视觉提案](web-ui/system-notifications-proposal.md) | `展示视觉已确认`（v18 所展示列表／详情与桌面／手机视觉已确认；常规辅助状态沿既有规则验证，不增加逐图审批，批准时正式 `ui/` 未修改） | 现有[管理员应用壳](../design/web-ui/administrator-application-shell.md)与[系统通知操作](../design/notifications/system-notification-operations.md)；权限、投递和数据契约保持不变 |

用户已对四张主图反馈“舒服”，具体范围见 [v18 确认记录](web-ui/previews/theme-system-notifications-v18-approval.json)。原始审阅与资产保持不变；本次认可不表示正式 UI 已实施。

## v19 索引补充

| 业务域 | 入口 | 状态 | 关联技术设计 |
| --- | --- | --- | --- |
| Web UI | [管理员 Trader Sync 与 Profit Sharing 四页集中提案](web-ui/admin-business-batch-proposal.md) | `已确认`（四个实际页面、八张桌面／手机主图所展示视觉已确认；辅助状态由实现者验证，不增加逐图审批，批准时正式 `ui/` 未修改） | 现有 [Trader Sync UI 设计](../design/web-ui/trader-sync-activity-alerts.md)与 [Profit Sharing](../design/governance/profit-sharing.md)；只读概要、治理阶段、权限和数据契约保持不变 |

用户于 2026-09-14 明确确认本批无须调整，范围见 [v19 确认记录](web-ui/previews/theme-admin-batch-v19-approval.json)。原始原型、截图与审阅记录保持不变；辅助状态按既定规则验证，不追加逐图审批，正式前端后续按本批视觉基准实施。

## v20 与其余页面安排

| 业务域 | 入口 | 状态 | 关联技术设计 |
| --- | --- | --- | --- |
| Web UI | [会员 Wallets、Solana 与 Profit Sharing 四页集中提案](web-ui/member-foundations-batch-proposal.md) | `已确认`（v20 四页八张主图已整批确认，见 [v20 确认记录](web-ui/previews/theme-member-foundations-v20-approval.json)；辅助图按既有规则覆盖，批准时正式主题未实现） | 现有[钱包归属与托管](../design/identity-access/wallet-ownership.md)、[Solana 列表](../design/web-ui/solana-discovery.md)、[Profit Sharing](../design/governance/profit-sharing.md)；业务、权限与阶段契约保持不变 |
| Web UI | [其余页面视觉定稿安排](web-ui/remaining-pages-plan.md) | 本轮 18 个业务页面／19 条路由，分 4／8／6 三批；v20、v21、v22 三批主视觉及六项共用适配均已整批确认，[总入口与继承状态](web-ui/theme-refactor-coverage.md)已映射到实施任务；本文保留视觉批次范围 | [会员应用壳](../design/web-ui/member-application-shell.md)、[管理员应用壳](../design/web-ui/administrator-application-shell.md) |
| Web UI | [市场、赛事与 Managed OO 八页集中提案](web-ui/market-intelligence-batch-proposal.md) | `已确认`（v21 八页十六张主图已整批确认，见 [v21 确认记录](web-ui/previews/theme-market-intelligence-v21-approval.json)；辅助状态沿既有规则覆盖，不追加逐图审批） | 批准时 Sports 图属于当时范围，后续 Sports 已退役；Market Radar、Managed OO 与共用视觉规则继续有效。历史图不代表已删除入口仍存在 |
| Web UI | [Worm Trading 六页与共用界面集中提案](web-ui/worm-and-common-batch-proposal.md) | `已确认`（v22 六个业务页面／七条路由及六项共用界面差异的 24 张主图已整批确认，见 [v22 确认记录](web-ui/previews/theme-worm-and-common-v22-approval.json)；辅助状态继承覆盖与验证） | Worm Trading 与共用视觉／身份能力继续有效；Markets 后续退役不撤销 Trading 的视觉批准，也不保留 Markets API 或权限 |

本轮排除 Trader Sync；[Nansen 钱包战绩 v1](token/wallet-analytics-page-proposal.md)已独立确认，直接沿用，不与 Wallets 私有钱包管理混同或重复审批。
