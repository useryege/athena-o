# 其余页面视觉定稿安排

> 状态：v20 四页、v21 八页、v22 Worm 六页及本批六项共用适配的展示视觉均已整批确认；v22 最新反馈为“确认✅”。2026-09-14 已完成[当前入口与继承状态归属核对](theme-refactor-coverage.md)，并整理[技术方案](../../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)与[实施计划](../../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)。本文保留三批视觉设计的范围记录，正式 UI 尚未改版。

## 已确认的工作方式

沿用 [全站视觉主题](visual-theme.md)及 v1–v22 各记录已获确认的配色、字体、应用壳、组件和页面视觉。一次交付多个实际页面，列表与详情、桌面与手机一起审阅。仅呈现新的信息结构和关键操作差异；加载、空态、失败、只读、窄屏及字号放大由设计者按既定规则覆盖与验证，不逐图追加确认。

本轮排除所有 Trader Sync 页面；既有确认保留，活动详情和批次摘要的新主题设计暂不推进。此前状态答复遗漏了独立目录中的 [Nansen 钱包战绩 v1 确认](../token/wallet-analytics-page-proposal.md)：该页已经用户确认，本轮直接沿用，不重复设计或审批。Wallets 私有钱包管理与 Token 的 Wallet analytics 是不同业务。

## 三批业务页面

以下按当前 React 路由与实际页面归组；同一编辑器的新增和编辑路由计为一个页面，不把弹窗或状态计为额外页面。共 18 个实际业务页面、19 条路由，三批所展示主视觉均已依据用户反馈定稿；各批确认记录界定实际认可范围，原型及独立审阅不替代正式实现与验收。

| 批次 | 实际页面 | 当前路由 | 状态 |
| --- | --- | --- | --- |
| 第一批 v20 | Wallets 私有钱包管理 | `/wallet` | [v20 展示视觉已确认](previews/theme-member-foundations-v20-approval.json) |
| 第一批 v20 | Solana 发行候选列表 | `/solana` | [v20 展示视觉已确认](previews/theme-member-foundations-v20-approval.json)；保留首版已确认业务，仅对齐全站主题 |
| 第一批 v20 | 会员 Profit Sharing 轮次列表 | `/profit-sharing` | [v20 展示视觉已确认](previews/theme-member-foundations-v20-approval.json) |
| 第一批 v20 | 会员 Profit Sharing 轮次详情 | `/profit-sharing/:slug` | [v20 展示视觉已确认](previews/theme-member-foundations-v20-approval.json)；包含提案编辑／提交、匿名投票和结果 |
| 第二批 v21 | Market Radar 热门市场 | `/market-radar` | [v21 展示视觉已确认](previews/theme-market-intelligence-v21-approval.json) |
| 第二批 v21 | Market Radar 实时市场 | `/market-radar/realtime` | [v21 展示视觉已确认](previews/theme-market-intelligence-v21-approval.json) |
| 第二批 v21 | Market Radar 涨跌榜 | `/market-radar/movers` | [v21 展示视觉已确认](previews/theme-market-intelligence-v21-approval.json) |
| 第二批 v21 | Sports Live | `/sports-live` | [v21 展示视觉已确认](previews/theme-market-intelligence-v21-approval.json) |
| 第二批 v21 | Sports History | `/sports-history` | [v21 展示视觉已确认](previews/theme-market-intelligence-v21-approval.json) |
| 第二批 v21 | World Cup Corners | `/world-cup-corners` | [v21 展示视觉已确认](previews/theme-market-intelligence-v21-approval.json) |
| 第二批 v21 | Managed OO 提案列表 | `/managed-oo/proposals` | [v21 展示视觉已确认](previews/theme-market-intelligence-v21-approval.json) |
| 第二批 v21 | Managed OO 争议列表 | `/managed-oo/disputes` | [v21 展示视觉已确认](previews/theme-market-intelligence-v21-approval.json) |
| 第三批 v22 | Worm Trading 资产 | `/worm-trading` | [v22 展示视觉已确认](previews/theme-worm-and-common-v22-approval.json) |
| 第三批 v22 | Worm Trading 组合列表 | `/worm-trading/combinations` | [v22 展示视觉已确认](previews/theme-worm-and-common-v22-approval.json) |
| 第三批 v22 | Worm Trading 组合编辑器 | `/worm-trading/combinations/new`、`/worm-trading/combinations/:id/edit` | [v22 展示视觉已确认](previews/theme-worm-and-common-v22-approval.json)；新增／编辑合并覆盖 |
| 第三批 v22 | Worm Trading 执行预览 | `/worm-trading/combinations/:id/execute` | [v22 展示视觉已确认](previews/theme-worm-and-common-v22-approval.json) |
| 第三批 v22 | Worm Trading 执行记录 | `/worm-trading/executions` | [v22 展示视觉已确认](previews/theme-worm-and-common-v22-approval.json) |
| 第三批 v22 | Worm Trading 执行详情 | `/worm-trading/executions/:id` | [v22 展示视觉已确认](previews/theme-worm-and-common-v22-approval.json) |

第一批先解决身份／地址、只读发现列表与会员提案表单的差异。第二批集中解决市场、赛事和预言机的列表与时序阅读。第三批将组合编辑、执行前核对与执行结果作为完整交易流程审阅，不拆成零散按钮确认。

## 共用页面的补齐覆盖

共用视觉已经确认；以下是将规则应用到现有界面的覆盖项，不另起一轮主题讨论，也不默认每项增加一次审批。与三批业务页面一起完成，出现实质交互差异时随当批材料说明。v22 六项界面差异的十二张桌面／手机主图已随本批整批确认；两身份域反馈作为状态证据，不增加业务路由或补写为用户逐图看过。Help 动态聊天／下载的有配置分支未在该空配置样本构造，后续继承覆盖与验证，不另设逐状态审批。

| 覆盖项 | 处理依据 | 状态 |
| --- | --- | --- |
| 管理员登录及共享注册／管理员入场结果 | 沿用 v14 身份页的单列布局，保留管理员独立身份、会话与准入语义 | [v22 主图视觉已确认](previews/theme-worm-and-common-v22-approval.json)；入场辅助状态继承覆盖 |
| 管理员 Profile 与 Access & session | 沿用 v11／v13，共享组件保留管理员字段及会话差异 | [v22 展示视觉已确认](previews/theme-worm-and-common-v22-approval.json) |
| 会员／管理员 Help | 同一阅读页面规则，保留各自入口与真实帮助配置 | [v22 展示视觉已确认](previews/theme-worm-and-common-v22-approval.json)；动态配置分支继续验证 |
| 会员／管理员未找到、无权限、会话失效与首屏加载 | 沿用已确认反馈与操作层级，分别核对两个应用 | [v22 状态证据已提供](worm-and-common-batch-proposal.md#共用界面差异)，沿既定规则覆盖与验证，不追加逐图审批 |
| 旧 Appearance 入口 | 单一深色决定已确认；后续实现中清理入口及相关主题路径，保留其他偏好 | 已确认方向，待实现方案 |

## 每批交付与定稿标准

- 依据当前源码和长期业务设计制作独立 HTML 预览，保留字段、权限、时间、数据精度和操作后果。
- 每个实际页面有桌面／手机主图；复杂表单和关键状态提供必要辅助证据。演示数据明确标识，不执行真实交易、密钥操作或供应商请求。
- 对照当前界面，完成有界的浏览器检查和独立视觉审阅。静态原型检查不替代正式业务接入与全栈验收。
- 用户认可后单独保存确认范围和资产摘要，更新本清单；未确认页面保持事实状态。
- 三批主视觉及 v22 六项共用适配已确认；[完整覆盖清单](theme-refactor-coverage.md)已将当前入口、继承状态和暂缓项对应到[实施计划](../../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)的 T1–T10。已确认决定直接沿用，实际代码和验收尚未开始。

路由依据：[会员入口](../../../ui/src/app/member/app.tsx)、[管理员入口](../../../ui/src/app/admin/app.tsx)。本轮不修改生产 `ui/`、不重做已批准资产，不将页面设计阶段写成实现完成。
