# ATHENA 产品上下文

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

ATHENA 有独立的普通会员与管理员应用。会员使用被授予权限的业务模块；管理员管理账户授权及各模块允许的运行概要。两种身份的页面、会话和业务数据边界分别维护。

Trader Sync 的第一阶段能力是 Activity Alerts。用户人工选择低频 Polymarket 交易者，及时查看目标正在交易的市场、Outcome、方向和公开事实，再自行判断是否前往市场操作。管理员在本功能中只查看安全概要，不查看用户私有备注、完整活动或逐条消息。

## Product Purpose

ATHENA 是面向区块链与预测市场的情报分析平台，通过 Web UI、API 和通知服务提供市场与链上数据。本文件为界面工作提供上下文，业务细则仍以相关长期需求和获批 spec 为准。

Trader Sync 第一阶段提供目标确认、独立订阅、实时成交活动和 Telegram 私聊提醒，Activity Alerts 不执行交易、签名或钱包操作。[交易板块初版需求](docs/requirements/polymarket-copy-trading/copy-trading.md)已逐项确认：事前一对一钱包绑定并由用户确认启用交易，用户自行到 Polymarket 为对应交易账户入金；收到通知后，用户在电脑或手机上的 ATHENA 内自行确认金额并提交市价订单。买入本金与手续费分开，手动买入次数由用户决定，初版不自动跟单。展示钱包全部持仓与买卖历史，保留本系统提交后的拒绝、失败、待核对记录和原始目标活动关联；普通市场及 Neg Risk 单个选项支持手动买卖与结算领取，Combo 仅展示持仓和历史。[平台校验](docs/requirements/polymarket-copy-trading/manual-trading-contract-verification.md)已完成官方契约与公开样本核对，账户入金一致性、私有执行与完整数据覆盖待实测。已进入[交易总体设计讨论](docs/design/trading/polymarket-manual-trading.md)，服务划分与页面组织为首轮提案，详细接口及完整交互尚需细化，交易代码未实现。

## Operating Context

- 用户于本轮 UI 设计中确认：Trader Sync 桌面优先，手机完整可用。手机须覆盖从 Telegram 进入详情、查看通知状态与管理订阅的流程。这不是对其他模块新增移动端改造要求。
- 已确认的主页以跨目标活动为主，桌面同时展示目标与监控状态，手机折叠目标栏。用户通过独立添加页解析和审阅目标，再明确确认订阅；备注、暂停、恢复、取消与中断记录集中在完整订阅列表和目标详情页管理。
- 活动使用独立详情，与 Telegram 深链接共用；摘要批次有独立页面展示全部分条。可见页每 5 秒更新状态，新活动提示后点击载入，保持阅读位置。管理员订阅概要独立，运行健康并入现有 Service Status。
- 首期按 10 名用户、每人最多 10 个未取消订阅设计，覆盖最多 100 个不同目标。后台监控不依赖浏览器持续打开。
- Telegram 绑定已有独立会员自助入口；管理员使用独立管理应用。新增界面应与这些实际入口衔接。

## Capabilities and Constraints

- 每位用户拥有独立订阅、备注、活动和通知资格；私有备注最多 20 个 Unicode code point，历史保留形成时快照。
- 地址或 Profile URL 均先解析并展示确认卡。身份不能核准时禁止创建；辅助资料缺失明确标为不可用，仍可确认。确认卡包含六个已定义 P/L 区间，默认 1Y；创建后不持续刷新收益。
- 基线建立后立即生效，按链上结算时间划界。断线、重启及故障期间的遗漏不补查，已有中断说明保留。
- 每条合格成交形成独立持久活动；用户级滚动 60 秒内前 10 条逐条提醒，第 11 条起按已确认规则汇总。消息未知结果不自动重发，摘要分条分别显示结果。
- 暂停/取消保留旧通知队列；撤权、解绑或重绑终止未取得发送许可的旧资格。重新获权后用户逐个恢复订阅。
- 显示结算时间、监控健康、资料缺失和通知结果，不把“无新活动”“监控失效”“投递未知”混为同一状态。
- 后端、会员与管理员页面、Notifications 返回态和 Service Status 集成已经实现。受控浏览器覆盖根路径与 `/athena`、桌面/手机、深浅主题、权限撤销、owner 隔离及关键管理流程；资料查询与运行结论以实际[验收记录](docs/testing/trader-sync-activity-alerts-acceptance.md)为准。
- 现有证据不等于生产 SLO：协议样本包含 11 条 recorded 与 1 条明确 synthetic，100 个真正活跃实网目标、公开时刻 P95/P99、供应商静默漏推完整性和长期稳定仍需外部环境验证。

## Brand Commitments

产品名称为 ATHENA；Trader Sync 第一阶段能力名为 Activity Alerts。当前界面的主题、组件和语言事实以实际 React/CSS 及界面设计文档为依据。

用户于 2026-09-13 确认全站视觉主题重构方向：单一深色主题、近黑背景、青绿色强调、清晰的文字与数据层次，以及克制的边框和动效。整体风格为深色极简科技风，带专业金融终端气质；会员与管理员界面遵循同一视觉目标，业务与权限边界继续独立。[v1 配色与按钮层级](docs/requirements/web-ui/previews/README.md)、[v2 字体与数字排版](docs/requirements/web-ui/typography-proposal.md)、[v3 导航／页头／页面密度](docs/requirements/web-ui/layout-proposal.md)的视觉效果均已确认，后续设计沿用该基准：Inter 用于英文标题、正文和数字，JetBrains Mono 用于地址与哈希；会员样板采用常驻侧栏、顶栏账户、集中查询区域，手机使用导航抽屉与逐币摘要。用户于 2026-09-14 确认了 [v4 第一组通用组件与状态](docs/requirements/web-ui/components-proposal.md)的展示视觉，包含表单与按钮状态、反馈配色、加载／空态及桌面／手机确认弹窗。[v5 列表控件](docs/requirements/web-ui/list-controls-proposal.md)的展示视觉也已确认。[v6 Trader Sync 首页](docs/requirements/web-ui/trader-sync-home-proposal.md)已确认桌面活动／目标布局、手机阅读，以及身份快照、Position ID 与来源入口默认收起、按需展开的处理；当前业务语义保持有效，其他状态不因本次反馈整体视为已确认。[v7 添加交易员页](docs/requirements/web-ui/trader-sync-add-proposal.md)所展示的桌面／手机布局、信息密度与操作层级也已确认，沿用资料与 P/L 核对、订阅确认分区及手机阅读顺序；辅助状态截图未逐图确认。[v8 订阅列表](docs/requirements/web-ui/trader-sync-subscriptions-proposal.md)已确认所展示的桌面四列、手机分组、信息密度和监控／通知状态区分；Cancelled 列表及其他辅助状态未逐图确认。[v9 订阅详情](docs/requirements/web-ui/trader-sync-subscription-detail-proposal.md)所展示的桌面／手机布局、操作层级、手机取消弹窗与历史默认摘要／按需展开方式已确认；未直接展示的辅助状态未逐图确认。[v10 Notifications](docs/requirements/web-ui/notifications-proposal.md)所展示的已连接桌面／手机布局、操作主次与桌面重新连接流程已确认；未直接展示的辅助状态未逐图确认。[v11 Account Center / Profile](docs/requirements/web-ui/account-profile-proposal.md)所展示的桌面资料页、手机编辑状态与手机离开确认视觉已确认；其他辅助状态与管理员页面未逐图确认。[v12 Account Center / Security](docs/requirements/web-ui/account-security-proposal.md)所展示的桌面／手机安全设置页及手机连接说明弹窗视觉已确认；其他辅助状态未逐图确认。[v13 Account Center / Access & session](docs/requirements/web-ui/account-access-proposal.md)所展示的桌面／手机权限与会话页及手机 Google 待授权页视觉已确认；其他辅助状态未逐图确认。[v14 会员登录与注册](docs/requirements/web-ui/auth-proposal.md)所展示的登录页与 Google 注册页桌面／手机视觉已确认；其他辅助状态未逐图确认。[v15 管理员导航与账户管理](docs/requirements/web-ui/admin-accounts-proposal.md)所展示的桌面账户管理、手机目录、手机权限详情和手机确认弹窗视觉已确认，按 [v15 确认记录](docs/requirements/web-ui/previews/theme-admin-accounts-v15-approval.json)执行；其他辅助状态未逐图确认。[v16 管理员 Service Status](docs/requirements/web-ui/service-status-proposal.md)所展示的 Services 桌面／手机、Notifications 桌面和 Trader Sync 手机视觉已确认，按 [v16 确认记录](docs/requirements/web-ui/previews/theme-service-status-v16-approval.json)执行；其他辅助状态未逐图确认。[v17 管理员 Etherscan Gateways](docs/requirements/web-ui/etherscan-gateways-proposal.md)所展示 Gateways 与 Live Probe 的桌面／手机视觉已确认，按 [v17 确认记录](docs/requirements/web-ui/previews/theme-etherscan-gateways-v17-approval.json)执行；其他辅助状态未逐图确认。用户再次明确，主旨是重构当前全站前端 UI，钱包样板用于提炼共用规则，后续页面级设计应对照现有代表业务页面；其他页面、业务展示语义、完整交互与跨平台字体验证按各自范围处理，正式主题重构尚未实现，见[全站视觉主题需求](docs/requirements/web-ui/visual-theme.md)。

**Nansen 是全站 UI 必须采用的主要视觉参考，主题与配色搭配尤其契合用户审美。** 后续设计与验收应整体对照其背景、面板、文字、强调色和边框之间的颜色关系，同时对照字体层次、间距与交互细节，按已确认样例落实。通用的“深色科技风”描述或单独采用一个青绿色不能替代这套参考，现有橙色／双主题也不构成保留旧视觉方向的要求。具体参考证据和确认状态以主题需求为准。

用户于 2026-09-14 提出逐页确认次数过多的顾虑。后续[按设计差异分组审阅](docs/requirements/web-ui/visual-theme.md#确认方式按设计差异分组)，沿用已确认的共用视觉规则，同类页面、列表与详情及桌面／手机合并展示；常规辅助状态由设计与验收覆盖，不逐张增加确认。历史确认边界保持准确，新的信息结构或关键操作差异仍需呈现。

[v18 系统通知列表与详情](docs/requirements/web-ui/system-notifications-proposal.md)所展示桌面／手机视觉已确认，按 [v18 确认记录](docs/requirements/web-ui/previews/theme-system-notifications-v18-approval.json)落实。用户于 2026-09-14 进一步明确：希望每次先完成多个页面，再集中提供审阅。后续按批次交付多个实际页面，桌面和手机一起展示；不是仅把一个页面的多个状态叫作一批。每批沿用已确认视觉规则，必要差异一次说明，收到反馈后针对指出部分修改。

[v19 管理员业务页面](docs/requirements/web-ui/admin-business-batch-proposal.md)所展示的 Trader Sync 订阅列表／概要、Profit Sharing 轮次列表／详情及桌面／手机八张主图已获用户整批确认，无须调整；按 [v19 确认记录](docs/requirements/web-ui/previews/theme-admin-batch-v19-approval.json)落实。后续沿用本批布局、阅读层级和既定 Nansen 主题配色，不重复确认相同决定；辅助状态继续覆盖与验证，正式 `ui/` 尚未改版。

[v20 会员基础业务页面](docs/requirements/web-ui/member-foundations-batch-proposal.md)所展示的 Wallets 私有钱包管理、Solana 发行候选、会员 Profit Sharing 轮次列表／详情及桌面／手机八张主图已获用户整批确认；按 [v20 确认记录](docs/requirements/web-ui/previews/theme-member-foundations-v20-approval.json)落实。沿用本批布局、阅读层级和既定 Nansen 主题配色，辅助状态继续覆盖与验证；本轮剩余页面按[分批安排](docs/requirements/web-ui/remaining-pages-plan.md)推进，Trader Sync 本轮排除，正式 `ui/` 尚未改版。

[v21 市场、赛事与 Managed OO 页面](docs/requirements/web-ui/market-intelligence-batch-proposal.md)所展示的 Market Radar 三页、Sports 三页及 Managed OO 两页的桌面／手机十六张主图已获用户整批确认；按 [v21 确认记录](docs/requirements/web-ui/previews/theme-market-intelligence-v21-approval.json)落实。后续重构沿用本批布局、数据阅读层级、操作主次及既定 Nansen 主题配色，辅助状态继续覆盖与验证，不重复确认相同决定。本轮页面进度以[分批安排](docs/requirements/web-ui/remaining-pages-plan.md)为准，Trader Sync 本轮排除，正式 `ui/` 尚未改版。

[v22 Worm Trading 与共用页面](docs/requirements/web-ui/worm-and-common-batch-proposal.md)所展示的 Worm 六个实际业务页面及管理员登录／共享注册／Profile／Access、会员／管理员 Help 六项共用适配的桌面／手机二十四张主图已获用户整批确认；按 [v22 确认记录](docs/requirements/web-ui/previews/theme-worm-and-common-v22-approval.json)落实。沿用本批布局、阅读顺序、操作层级与既定 Nansen 主题搭配，辅助状态继承覆盖与验证，不追加逐图审批。三批 18 个业务页面及本批共用适配的主视觉均已确认；[总覆盖清单](docs/requirements/web-ui/theme-refactor-coverage.md)、[技术方案](docs/superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)与[实施计划](docs/superpowers/plans/2026-09-14-web-ui-theme-refactor.md)已整理。Trader Sync 专属页面暂缓重排，共享主题影响须回归；正式 `ui/` 尚未改版。

总体审阅后，用户要求先修复一致性问题。[修订契约](docs/requirements/web-ui/theme-consistency-contract.md)及 [v23 证据](docs/requirements/web-ui/previews/theme-consistency-v23/README.md)已补齐最终颜色、文字放大、状态、数字列和图标规则；全站与 Token 计划共同消费同一主题及字体。正式前端尚未改版。

## Evidence on Hand

- [平台说明](README.md)
- [全站视觉主题需求](docs/requirements/web-ui/visual-theme.md)
- [业务需求](docs/requirements/polymarket-copy-trading/target-trade-monitoring-notifications.md)
- [已确认后端 spec](docs/superpowers/specs/2026-09-10-trader-sync-activity-alerts-design.md)
- [已整体确认 UI spec](docs/superpowers/specs/2026-09-10-trader-sync-activity-alerts-ui-design.md)
- [长期 UI 设计](docs/design/web-ui/trader-sync-activity-alerts.md)
- [前后端联合实现计划](docs/superpowers/plans/2026-09-10-trader-sync-activity-alerts.md)
- [最终验收记录](docs/testing/trader-sync-activity-alerts-acceptance.md)
- [会员应用壳](docs/design/web-ui/member-application-shell.md)、[管理员应用壳](docs/design/web-ui/administrator-application-shell.md)
- [共享样式实现](ui/src/app/styles/shared.css)、[会员入口](ui/src/app/member/app.tsx)、[管理员入口](ui/src/app/admin/app.tsx)

## Product Principles

- 优先让用户理解目标实际交易了什么，以及可前往哪个市场查看。
- 数据缺失、监控中断和消息未知必须如实表达，不伪造完整性或成功。
- 用户隔离、管理员最小可见范围和既有授权决定落实到实际界面与接口。
- 页面设计与后端契约共同形成可验收的用户流程，已有技术设计可根据有依据的接口缺口修订。
