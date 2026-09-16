# 全站主题重构覆盖清单

> 范围修订（2026-09-16）：[Sports 删除、Worm 保留](../development-runtime/sports-removal.md)已确认，尚未实施。目标覆盖只移除 Sports Live／History 和 World Cup Corners；Worm 全部页面、路由和验收项恢复为保留范围。下文批准记录和历史证据保留当时事实，共用视觉与其他页面决定继续有效。

> 状态：51 项入口／规则已逐项核对，37 条改版、2 条删除、4 条既有路由规则和 8 条 Trader Sync 共享回归已完成 T9 验证；T1–T9 独立任务审阅通过。最终审阅、环境收尾和通知见[交付状态](../../testing/web-ui-theme-refactor-acceptance.md#交付状态与环境收尾)。设计批准、源码实施、受控状态和真实接入采用独立证据，不相互替代。

本清单将此前逐项确认的视觉设计映射到当前 `ui/`。本轮三批 18 个业务页面的主视觉和六项共用适配已确认；其余既有页面使用对应 v10–v19 记录。没有发现本轮需要重新逐页确认的新增页面。常规状态按已确认规则实施并验证，不增加逐图审批。

## 统计口径

[机器清单](theme-refactor-coverage.json)逐条覆盖会员入口 35 项、管理员入口 16 项，共 51 项路由／入口规则。包含组件外处理的共享注册和管理员登录；同一个通配路由只计一次。

| 处理方式 | 规则数量 | 含义 |
| --- | ---: | --- |
| 按已确认设计改版 | 37 | 对应 36 个实际页面；组合新增／编辑共用一个页面，注册的管理员身份为同一路由的差异 |
| 删除旧 Appearance | 2 | 会员／管理员各一条；删除菜单、入口及主题偏好，不保留旧页重定向兼容 |
| Trader Sync 暂缓页面重排 | 8 | 会员六条、管理员两条；共享主题和公共壳生效，原业务布局保留并回归 |
| 默认入口和未找到／兜底 | 4 | 两个应用各一组；保留路由及身份分流语义，套用共用反馈 |

以上是现有入口核对数量，不是又增加 51 张待确认页面。根路径和 `/athena` 部署使用同一清单；管理员前缀是应用 base，不能混入 API base。

## 当前页面与实现任务

任务编号对应[正式重构实施计划](../../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)。视觉依据以各版本 approval 为准，原型与历史 review 原字节保留。

### 会员入口

| 页面／入口 | 路由 | 当前源码 | 视觉依据 | 处理／任务 |
| --- | --- | --- | --- | --- |
| 默认入口 | `/` | [源码](../../../ui/src/app/member/app.tsx) | v3 + v15 + v22 | 共用规则覆盖 · T2 |
| Wallets | `/wallet` | [源码](../../../ui/src/app/member/pages/wallets.tsx) | v20 | 按确认设计改版 · T5 |
| Worm Assets | `/worm-trading` | [源码](../../../ui/src/app/member/pages/worm-trading.tsx) | v22 | 按确认设计改版 · T7 |
| 组合列表 | `/worm-trading/combinations` | [源码](../../../ui/src/app/member/pages/worm-trading-combinations.tsx) | v22 | 按确认设计改版 · T7 |
| 组合编辑器：新增 | `/worm-trading/combinations/new` | [源码](../../../ui/src/app/member/pages/worm-trading-combinations.tsx) | v22 | 按确认设计改版 · T7 |
| 组合编辑器：编辑 | `/worm-trading/combinations/:id/edit` | [源码](../../../ui/src/app/member/pages/worm-trading-combinations.tsx) | v22 | 按确认设计改版 · T7 |
| 执行预览 | `/worm-trading/combinations/:id/execute` | [源码](../../../ui/src/app/member/pages/worm-trading-execution-preview.tsx) | v22 | 按确认设计改版 · T8 |
| 执行记录 | `/worm-trading/executions` | [源码](../../../ui/src/app/member/pages/worm-trading-executions.tsx) | v22 | 按确认设计改版 · T8 |
| 执行详情 | `/worm-trading/executions/:id` | [源码](../../../ui/src/app/member/pages/worm-trading-executions.tsx) | v22 | 按确认设计改版 · T8 |
| 热门市场 | `/market-radar` | [源码](../../../ui/src/app/member/pages/market-radar.tsx) | v21 | 按确认设计改版 · T6 |
| 实时市场 | `/market-radar/realtime` | [源码](../../../ui/src/app/member/pages/market-radar.tsx) | v21 | 按确认设计改版 · T6 |
| 涨跌榜 | `/market-radar/movers` | [源码](../../../ui/src/app/member/pages/market-radar.tsx) | v21 | 按确认设计改版 · T6 |
| Solana 发行候选 | `/solana` | [源码](../../../ui/src/app/member/pages/solana.tsx) | v20 | 按确认设计改版 · T5 |
| Sports Live | `/sports-live` | [源码](../../../ui/src/app/member/pages/sports-live.tsx) | v21 | 按确认设计改版 · T6 |
| Sports History | `/sports-history` | [源码](../../../ui/src/app/member/pages/sports-history.tsx) | v21 | 按确认设计改版 · T6 |
| World Cup Corners | `/world-cup-corners` | [源码](../../../ui/src/app/member/pages/world-cup-corners.tsx) | v21 | 按确认设计改版 · T6 |
| Managed OO 提案 | `/managed-oo/proposals` | [源码](../../../ui/src/app/member/pages/managed-oo.tsx) | v21 | 按确认设计改版 · T6 |
| Managed OO 争议 | `/managed-oo/disputes` | [源码](../../../ui/src/app/member/pages/managed-oo.tsx) | v21 | 按确认设计改版 · T6 |
| 活动首页 | `/trader-sync` | [源码](../../../ui/src/app/member/pages/trader-sync/home.tsx) | v6 | 暂缓重排；共享影响回归 · T9 |
| 订阅列表 | `/trader-sync/subscriptions` | [源码](../../../ui/src/app/member/pages/trader-sync/subscriptions.tsx) | v8 | 暂缓重排；共享影响回归 · T9 |
| 订阅详情 | `/trader-sync/subscriptions/:subscriptionId` | [源码](../../../ui/src/app/member/pages/trader-sync/subscription-detail.tsx) | v9 | 暂缓重排；共享影响回归 · T9 |
| 添加交易员 | `/trader-sync/add` | [源码](../../../ui/src/app/member/pages/trader-sync/add.tsx) | v7 | 暂缓重排；共享影响回归 · T9 |
| 活动详情 | `/trader-sync/activities/:activityId` | [源码](../../../ui/src/app/member/pages/trader-sync/activity-detail.tsx) | 暂缓新主题定稿 | 暂缓重排；共享影响回归 · T9 |
| 批次摘要 | `/trader-sync/summaries/:batchId` | [源码](../../../ui/src/app/member/pages/trader-sync/summary-detail.tsx) | 暂缓新主题定稿 | 暂缓重排；共享影响回归 · T9 |
| Notifications 连接设置 | `/notifications` | [源码](../../../ui/src/app/member/pages/notifications.tsx) | v10 | 按确认设计改版 · T3 |
| Profile | `/account/profile` | [源码](../../../ui/src/app/shared/pages/account-center.tsx) | v11 | 按确认设计改版 · T3 |
| 旧 Appearance | `/account/appearance` | [源码](../../../ui/src/app/shared/pages/account-center.tsx) | Nansen 单一深色决定 | 删除旧入口 · T1 |
| Security | `/account/security` | [源码](../../../ui/src/app/member/pages/account-security.tsx) | v12 | 按确认设计改版 · T3 |
| Access & session | `/account/access` | [源码](../../../ui/src/app/shared/pages/account-center.tsx) | v13 | 按确认设计改版 · T3 |
| Profit Sharing 轮次列表 | `/profit-sharing` | [源码](../../../ui/src/app/member/pages/profit-sharing.tsx) | v20 | 按确认设计改版 · T5 |
| Profit Sharing 轮次详情 | `/profit-sharing/:slug` | [源码](../../../ui/src/app/member/pages/profit-sharing.tsx) | v20 | 按确认设计改版 · T5 |
| Help | `/help` | [源码](../../../ui/src/app/shared/pages/help.tsx) | v22 | 按确认设计改版 · T3 |
| 未找到／兜底 | `/*` | [源码](../../../ui/src/app/member/app.tsx) | v3 + v15 + v22 | 共用规则覆盖 · T2 |
| 会员登录 | `/login` | [源码](../../../ui/src/app/member/pages/login.tsx) | v14 | 按确认设计改版 · T3 |
| 共享注册：member／admin | `/register` | [源码](../../../ui/src/app/member/pages/register.tsx) | v14 + v22 | 按确认设计改版 · T3 |

### 管理员入口

| 页面／入口 | 路由 | 当前源码 | 视觉依据 | 处理／任务 |
| --- | --- | --- | --- | --- |
| 默认入口 | `/admin/` | [源码](../../../ui/src/app/admin/app.tsx) | v3 + v15 + v22 | 共用规则覆盖 · T2 |
| 账户管理 | `/admin/accounts` | [源码](../../../ui/src/app/admin/pages/admin-accounts.tsx) | v15 | 按确认设计改版 · T4 |
| 管理轮次列表 | `/admin/profit-sharing` | [源码](../../../ui/src/app/admin/pages/profit-sharing-admin.tsx) | v19 | 按确认设计改版 · T5 |
| 管理轮次详情 | `/admin/profit-sharing/:slug` | [源码](../../../ui/src/app/admin/pages/profit-sharing-admin.tsx) | v19 | 按确认设计改版 · T5 |
| 管理员订阅列表 | `/admin/trader-sync/subscriptions` | [源码](../../../ui/src/app/admin/pages/trader-sync/subscriptions.tsx) | v19 | 暂缓重排；共享影响回归 · T9 |
| 管理员订阅概要 | `/admin/trader-sync/subscriptions/:id` | [源码](../../../ui/src/app/admin/pages/trader-sync/subscription-detail.tsx) | v19 | 暂缓重排；共享影响回归 · T9 |
| Service Status | `/admin/service-status` | [源码](../../../ui/src/app/admin/pages/service-status.tsx) | v16 | 按确认设计改版 · T4 |
| Etherscan Gateways | `/admin/etherscan-gateways` | [源码](../../../ui/src/app/admin/pages/etherscan-gateways.tsx) | v17 | 按确认设计改版 · T4 |
| 系统通知列表 | `/admin/notifications` | [源码](../../../ui/src/app/admin/pages/system-notifications.tsx) | v18 | 按确认设计改版 · T4 |
| 系统通知详情 | `/admin/notifications/:id` | [源码](../../../ui/src/app/admin/pages/system-notification-detail.tsx) | v18 | 按确认设计改版 · T4 |
| 管理员 Profile | `/admin/account/profile` | [源码](../../../ui/src/app/shared/pages/account-center.tsx) | v22 | 按确认设计改版 · T3 |
| 旧 Appearance | `/admin/account/appearance` | [源码](../../../ui/src/app/shared/pages/account-center.tsx) | Nansen 单一深色决定 | 删除旧入口 · T1 |
| 管理员 Access & session | `/admin/account/access` | [源码](../../../ui/src/app/shared/pages/account-center.tsx) | v22 | 按确认设计改版 · T3 |
| 管理员 Help | `/admin/help` | [源码](../../../ui/src/app/shared/pages/help.tsx) | v22 | 按确认设计改版 · T3 |
| 未找到／兜底 | `/admin/*` | [源码](../../../ui/src/app/admin/app.tsx) | v3 + v15 + v22 | 共用规则覆盖 · T2 |
| 管理员 Google 登录 | `/admin/login` | [源码](../../../ui/src/app/admin/login.tsx) | v22 | 按确认设计改版 · T3 |

## 已有设计与特殊边界

- 共享 `/register` 由会员 bundle 的 `RegistrationBootstrap` 承载，`athenaRealm=admin` 显示 v22 管理员注册差异；没有 `/admin/register`。管理员只提供 Google 登录，会员 Google／Phantom 保持独立身份。
- 会员 API key 许可、Profit Sharing 资格与各模块 grant 保留独立判断；管理员自己的会员模块权限不自动变为可用。页面隐藏不替代服务端授权。
- `/admin/service-status` 的 Trader Sync 页签属于 v16 已确认运维页，继续按 v16 改版；本轮暂缓的是八条 Trader Sync 专属业务路由的重新排版。
- Token 导航在当前源码中为 disabled，没有已注册的 Wallet analytics 路由。[Nansen 钱包 v1](../token/wallet-analytics-page-proposal.md)已确认，并有[独立接入计划](../../superpowers/plans/2026-09-14-token-wallet-analytics.md)；沿用其页面、共享主题及组件契约，不能把静态样板当作已上线入口。本计划不开发供应商接入或启用该导航。
- 会员／管理员 Help 的 `/llms.txt`、`/docs/ai/safety.md`、`/swagger-ui` 是已有资源入口，不属于额外 React 业务页；有配置时才出现下载／聊天，会员 Connect AI 仍依赖 API key 许可。
- 去除 Appearance 还涉及 HTML 首屏脚本、注册 Provider、两套账户菜单、本地 `ViewPreferences` 和服务端主题专用 preferences。清理范围及生成顺序见[技术方案](../../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)，不改分页、排序和导航偏好。

## 继承状态矩阵

“有视觉依据”表示已有批准规则或原型证据可沿用；下面全部正式代码场景仍需在实现后验证。每一项均有对应任务，不能把辅助图当作真实网络和权限验收。

| 编号 | 状态／差异 | 视觉与业务依据 | 正式实现必须验证 | 任务 |
| --- | --- | --- | --- | --- |
| S1 | 首屏加载、成功空态、首次失败、保留旧结果／局部失败 | v4、各页面 checks | 空数组与请求失败区分；loading 不清除仍有效结果；独立来源重试与时间不混用 | T2–T8 |
| S2 | 表单校验、草稿、冲突、提交失败／单飞 | v4、v11、v12、v14、v15、v19、v20、v22 | 错误关联字段；失败保留输入；离开保护；revision 冲突不覆盖；请求中禁止重复提交 | T3–T5、T7–T8 |
| S3 | Pending、无模块、只读、管理员失权、会话失效 | v13、v15、v22；现有 guards | API key 与模块资格独立；撤权清读写 scope；晚返回不恢复旧身份数据；两 realm 退出相互独立 | T2–T4、T9 |
| S4 | 键盘导航、弹窗、下拉、复制拒绝 | v2、v4、v5 与页面状态图 | 打开／关闭焦点、Escape、背景不可聚焦；复制完整值；拒绝时允许选中全文；pending 时关闭策略沿原业务 | T2–T8 |
| S5 | 1440×900、390×844、320×844、200% 根字号 | 各版桌面／手机及辅助图 | 无整页横向溢出、裁字或覆盖；行内滚动只留给必要数据；固定操作不挡正文；金额与完整标识可读 | T1–T9 |
| S6 | Help 下载／聊天有配置与空配置 | v22 规则，当前 Help 实现 | 真实配置决定入口，按部署 base 生成路径；管理员不出现会员 Security；资源可达性另查 | T3、T9 |
| S7 | Notifications 绑定／替换、Security 密钥 | v10、v12 | 旧连接替换前仍有效；失败保留旧连接；密钥一次展示／撤销、账户归属、未完成请求清理 | T3 |
| S8 | 管理员来源页签、测试发送、网关 Probe | v16–v18 | 三来源时间／状态独立；测试发送单飞和结果；Probe 与网关健康不混合 | T4 |
| S9 | 会员／管理员 Profit Sharing 四阶段 | v19、v20 | Draft／Collecting／Voting／Closed、exact-five、封存、匿名、未公布结果与权限隔离 | T5 |
| S10 | Worm 钱包、连接、Cash Out 单仓／批次 | v22 与现有交易契约 | 0–20 选择限制、只读、五分钟授权、完整冻结、USDC 严格增加门槛、未知隔离与控制动作 | T7 |
| S11 | Worm 组合与 Run | v22 与现有交易契约 | 无效选择移除后可恢复保存；revision；1–20 执行子集；Prepare／授权／Start 分离；暂停／终止等待收敛；未知只读核对 | T7–T8 |
| S12 | 市场、赛事、Managed OO 数据差异 | v21 | 三类市场字段分别映射、0 与缺失区分；比分不当价格；完整原始精度、证据角色和时间 | T6 |
| S13 | 共享变化对暂缓页面的影响 | Trader Sync 已有实现及 v6–v9／v19 | 八条路由仍可达、角色／owner／cursor 不变、按钮与弹窗可读可操作；不据此声称专属布局已改版 | T9 |
| S14 | 首次访问、系统浅色、主题旧缓存 | v1 单深色决定 | 两份 HTML 首屏和共享注册始终深色；系统切换不会改色；不再读写／同步主题偏好 | T1 |

## 核对结论与后续

本轮路由已全部归类，继承状态有明确实施与验证归属；没有以“未逐图确认”新增审批。T1–T10 已按[方案](../../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)与[计划](../../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)完成正式 UI、相关后端／数据库变更及约定验收，并通过独立任务和整分支审阅。最终生产 `78e473b7`，临时环境已停止并保留数据；实际版本、外部未验证范围及通知状态见[总验收记录](../../testing/web-ui-theme-refactor-acceptance.md)。

## 实际验收归属

[机器清单](theme-refactor-coverage.json)每条 `routes[].evidence` 指向 T1–T9 稳定报告以及最终 JSON 的精确 pointer／case ID；改版路线指向[逐页面证据](../../../.tmp/ui-theme-refactor/task-9/route-visual-review-20167913.json)的 38 个主场景（37 条入口加共享注册 admin 变体）。每场景包含批准图、四条件正式 React、人工看图和版本明确的原生缩放／axe 映射。

2 条 Appearance、4 条默认／未知和 8 条 Trader Sync 专页分别指向[状态证据](../../../.tmp/ui-theme-refactor/task-9/state-coverage-20167913.json)的 `appearanceAndFallbacks`、`traderSyncEightRoutes`；S1–S14 均由 `state_evidence` 指向各自原始测试归属。规则库存检查只核路由存在性，不替代浏览器证据。

完整 acceptance 1018 的版本为 `e7ec272b`；`82f25ad8` 提供新主图／语义差量 38、无过滤 a11y 176、native 38、真实呈现 56、认证边界 4 与实际字体；`20167913` 仅对最终 Close 与选中图形修正复验 acceptance 14、a11y 24、native 弹窗 1、smoke 2。没有把整套基线改记成 201 重跑。

真实呈现 56 是 24 次业务 GET 成功、22 次未启用来源 503 和 10 次没有业务 GET，另有 10 个来源页签及 2 项 Help 资源。Worm 编辑／预览／执行详情、会员／管理员轮次详情、管理员通知详情共六项缺少真实成功记录；其成功状态由受控正式 React 验证。外部 OAuth／provider／签名和实体手机软键盘未验证，详见[总验收记录](../../testing/web-ui-theme-refactor-acceptance.md)。

最后整分支 I1 修正 `78e473b7` 仅修改组合列表删除逻辑及两个测试文件，无 CSS／DOM 差量；13 项新增集成测试、全 Jest 38 suites／420 tests、两部署局部40项和真实 smoke2通过。[最终交付索引](../../../.tmp/ui-theme-refactor/final-delivery-78e473b7.json)将其与原 e7／82／201 证据分开，旧主图与 a11y 保留原 SHA。76 个批准图引用对应74份唯一图片，共享注册复用图片，逐项哈希不变。
