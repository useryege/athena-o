# Worm Trading 与共用页面集中视觉提案（v22）

> 状态：所展示视觉已整批确认。用户于 2026-09-14 对六个 Worm 业务页面及六项共用适配的二十四张桌面／手机主图反馈“确认✅”，确认范围见 [v22 确认记录](previews/theme-worm-and-common-v22-approval.json)。正式 `ui/` 尚未改版。

本批承接[其余页面安排](remaining-pages-plan.md)的第三批：Worm Trading 六个实际业务页面、七条正式路由，以及管理员身份／自助账户、会员与管理员 Help 六项界面差异。新增和编辑共用一个组合编辑器；反馈矩阵是两个身份域的状态证据，不是新增业务页面或路由。一次交付十二项界面的二十四张桌面／手机主图，常规辅助状态不逐图增加审批。

沿用 v1–v21 已确认的 Nansen 单一深色、Inter／JetBrains Mono、应用壳和组件。Trader Sync 本轮排除；独立已确认的 [Token／Nansen 钱包战绩 v1](../token/wallet-analytics-page-proposal.md)保持原设计，不重复调研或改动。

本次定稿覆盖上述二十四张主图的布局、信息密度、阅读顺序、操作层级和既定主题搭配。五十七张辅助图继续作为状态覆盖与验证证据，不补写成用户逐图看过；常规继承状态不增加审批。四十张当前 React 对照图保留现状参照用途。交付原型、截图、数据、检查和审阅 JSON 共 138 份资产按确认记录保存原字节；审阅 JSON 的 `awaiting_user_review` 是交付时历史状态，后续批准由确认记录承接。

## 视觉规则与信息层级

四份成品 HTML 继续使用近黑背景 `#06080B`、面板 `#0F1114`、较亮层 `#181D22`、分隔线 `#252A30`、白色正文、灰色辅助信息 `#9FA0A1` 和青绿色强调 `#00FFA7`；主要按钮使用深色文字。正向、等待、错误与未知保留独立语义，青绿色不代替交易成功结论。面板靠明暗和细边框分层，图标采用 SVG，不新增视觉世界或装饰影像。

Inter 用于标题、正文与数字，JetBrains Mono 用于完整地址、哈希和技术标识。成品页标题桌面为 1.75rem、手机为 1.5rem，主要区块标题 1.25rem；沿用已确认字号层级和数字对齐。桌面侧栏 224px、页头 64px，手机用导航抽屉和纵向分隔记录。按钮、输入等被检查的主要操作目标至少 44px；头像采用 1.75rem 容器与紧凑行高，容纳 200% 字号。

Worm 页面先展示当前对象、独立状态和允许动作，再展开技术证据；组合编辑器桌面将市场目录与有序选择并列，手机按目录、选择顺序阅读；执行详情将冻结意图和下一项允许动作放在步骤账本之前。登录／注册使用居中单列，账户页保持 Profile 单面板、Access 模块权限在前和会话在后；Help 使用资源分隔行。完整身份、未知数值和操作后果保留在相应上下文内。

## 原型截图与审阅材料

以下入口均为仓库静态原型，不是正式业务服务。Worm Assets 为单页，其余三份 HTML 通过 `?view=` 切换页面。

| 页面／界面差异 | 现有正式路由 | 静态原型 | 桌面主图 | 手机主图 |
| --- | --- | --- | --- | --- |
| Worm Assets | `/worm-trading` | [打开](previews/theme-worm-assets-v22.html) | [桌面](previews/theme-worm-assets-v22-desktop.png) | [手机](previews/theme-worm-assets-v22-mobile.png) |
| 组合列表 | `/worm-trading/combinations` | [打开](previews/theme-worm-combinations-v22.html?view=list) | [桌面](previews/theme-worm-combinations-v22-list-desktop.png) | [手机](previews/theme-worm-combinations-v22-list-mobile.png) |
| 组合编辑器（新增／编辑） | `/worm-trading/combinations/new、/worm-trading/combinations/:id/edit` | [打开](previews/theme-worm-combinations-v22.html?view=editor) | [桌面](previews/theme-worm-combinations-v22-editor-desktop.png) | [手机](previews/theme-worm-combinations-v22-editor-mobile.png) |
| 执行预览 | `/worm-trading/combinations/:id/execute` | [打开](previews/theme-worm-combinations-v22.html?view=preview) | [桌面](previews/theme-worm-combinations-v22-preview-desktop.png) | [手机](previews/theme-worm-combinations-v22-preview-mobile.png) |
| 执行记录 | `/worm-trading/executions` | [打开](previews/theme-worm-executions-v22.html?view=list) | [桌面](previews/theme-worm-executions-v22-list-desktop.png) | [手机](previews/theme-worm-executions-v22-list-mobile.png) |
| 执行详情 | `/worm-trading/executions/:id` | [打开](previews/theme-worm-executions-v22.html?view=detail) | [桌面](previews/theme-worm-executions-v22-detail-desktop.png) | [手机](previews/theme-worm-executions-v22-detail-mobile.png) |
| 管理员登录 | `/admin/login` | [打开](previews/theme-common-adaptations-v22.html?view=login) | [桌面](previews/theme-common-adaptations-v22-login-desktop.png) | [手机](previews/theme-common-adaptations-v22-login-mobile.png) |
| 共享注册（管理员身份） | `/register?athenaRealm=admin` | [打开](previews/theme-common-adaptations-v22.html?view=register) | [桌面](previews/theme-common-adaptations-v22-register-desktop.png) | [手机](previews/theme-common-adaptations-v22-register-mobile.png) |
| 管理员 Profile | `/admin/account/profile` | [打开](previews/theme-common-adaptations-v22.html?view=profile) | [桌面](previews/theme-common-adaptations-v22-profile-desktop.png) | [手机](previews/theme-common-adaptations-v22-profile-mobile.png) |
| 管理员 Access & session | `/admin/account/access` | [打开](previews/theme-common-adaptations-v22.html?view=access) | [桌面](previews/theme-common-adaptations-v22-access-desktop.png) | [手机](previews/theme-common-adaptations-v22-access-mobile.png) |
| 会员 Help | `/help` | [打开](previews/theme-common-adaptations-v22.html?view=member-help) | [桌面](previews/theme-common-adaptations-v22-member-help-desktop.png) | [手机](previews/theme-common-adaptations-v22-member-help-mobile.png) |
| 管理员 Help | `/admin/help` | [打开](previews/theme-common-adaptations-v22.html?view=admin-help) | [桌面](previews/theme-common-adaptations-v22-admin-help-desktop.png) | [手机](previews/theme-common-adaptations-v22-admin-help-mobile.png) |

## Worm Trading 业务边界

### 资产与 Cash Out

资产页按已选钱包、余额／连接／活动独立状态、持仓／请求分层；完整钱包地址与原始数值精度保留，USDC 或 P/L 未知显示不可用，部分活动不等于空仓。钱包选择保留 0–20 限制与移除保护，连接管理保留五分钟授权和未知凭据风险说明。

单仓 Cash Out 先核对精确持仓身份，批次冻结全量持仓及顺序；二者授权即排队，不套用执行 Run 的另行 Start。批次继续依赖逐笔观察到 USDC 严格增加的门槛，钱包总额观察不证明某一笔专属到账。暂停／继续／终止与只读核对仅按已有业务语义演示。

### 组合、编辑与执行预览

组合名称保留去首尾空白、1–80 Unicode 字符、拒绝控制字符和账户内唯一规则；每个市场只选择一个方向，顺序可调，删除事件说明其选项一并移除，未保存离开有草稿保护。失效市场仍显示原因并禁止保存；移除失效选项后，有效组合可以恢复保存。历史成交价使用精确美分及完整小数提示，不当作可成交报价。

预览保留 Combination → Wallets → Checks → Review 四步及冻结 revision。未建立预览时钱包子集为空，只展示已选且 CONNECTED 的钱包，用户显式选择、清空、移除和排序；执行子集保留 1–20 边界。防护条件不能关闭；已持仓或钱包范围未覆盖的进行中操作会使匹配及该钱包后续步骤跳过。SOL 余额是信息提示，执行信任边界单独说明。

共享合成示例为三个钱包 × 两个市场，组合 revision 7、钱包选择 revision 4，四步 ready／两步 skipped；Orion 因 `MARKET_POSITION_EXISTS` 跳过。最大本金 20.000000、费用 0.040000、所需总额 20.040000 USDC。Polymarket 5／Hyperliquid 1 USDC 与 1× 沿用已有后端固定规则；有效 partial-fill Estimate 不被错误排除。BUILDING／FAILED 不给出已知总额；过期、选择变化或无可执行步骤阻止 Prepare。Build 只展示这组示例，其他选择提示演示范围。

Prepare 只冻结 Run，随后独立授权；静态 Prepare 说明弹窗不构成新增正式产品确认关卡。原型不会创建 plan、Run 或发出订单。

### 执行记录与详情

历史分别表达 terminal、completed、skipped 等事实；completed 只表示已观察到 Open Position。详情按钱包优先顺序保留完整 Run、步骤、事件、市场、请求和持仓证据。冻结 Run、独立授权、显式 Start 保持分离，允许的 Pause／Continue／Terminate 来自 Run 状态。

活动步骤的暂停和终止等待收敛；未知结果仅做只读状态核对，不重复发出操作，终止也不声称撤回既有订单或消除未知隔离。完成观察、步骤 Last update 与 Run updated 各自表达时间事实；修正示例依次为 16:01:30、16:01:35、16:02:00（UTC+8）。历史 Run 没有对应旧 plan 示例时，不提供错误的共享预览链接。

## 共用界面差异

管理员登录只提供 Google。新身份复用 `/register?athenaRealm=admin`，没有新增 `/admin/register`；明确 Administrator account、永久全局用户名和 ticket 失效。Profile 区分只读用户名与可编辑显示名，覆盖 Unicode 1–80、控制字符、头像格式／2MiB 边界、草稿、重置、离开确认、冲突与保存失败。

管理员和会员使用不同 UUID、用户名及独立会话。管理员 Access 的模块权限全为 NONE，API key 与会员 Profit Sharing 为 No，登录资格单列；时间、版本和 revision 默认收起，未知 API version 不伪装成确定值。管理员不是拥有所有会员模块权限的会员。

Help 仅展示现有 `/llms.txt`、`/docs/ai/safety.md`、`/swagger-ui` 三个固定资源；会员仅在 API key 许可开启时显示 Connect an AI，管理员不显示会员 Security 入口。样本的 `binaryUrls` 为空，不构造客服、下载包或 chatUrl。现有动态聊天／下载有配置分支未在本批构造，后续沿同一规则覆盖和验证，不新增逐状态审批关卡；真实资源可达性未验证。

[共用反馈桌面](previews/theme-common-adaptations-v22-feedback-desktop.png)及[手机](previews/theme-common-adaptations-v22-feedback-mobile.png)并列呈现两个身份域的未找到、无权限、会话失效和加载。会员无模块权限实际前往 Access；会话失效回相应登录并保留 returnTo，没有新增独立 403／expired 路由。管理员准入失败、角色重验与失败恢复保留独立说明。旧 Appearance 清理属于已确认的单深色方向，正式入口与持久化清理仍待实施方案。

## 验证证据与审阅范围

当前共 **121 张 PNG：81 张提案图（24 主图、57 辅助图）及 40 张当前 React 对照图**。初审为 120 张，随后补一张失效选择清理后恢复保存的手机证据。最新数量、文件摘要与审阅原文以[批次审阅记录](previews/theme-worm-and-common-v22-review.json)为准；来源扫描为 121 张、0 缺失。此前 469 份预览文件保持原字节。

| 域 | 提案／React 对照 | 合成数据 | 原型检查 | React 采集 |
| --- | --- | --- | --- | --- |
| Assets | 13／2 | [data](previews/theme-worm-assets-v22-data.json) | [13 图、5 项检查](previews/theme-worm-assets-v22-checks.json) | [before](previews/theme-worm-assets-v22-before-capture.json) |
| Combinations | 22／6 | [data](previews/theme-worm-combinations-v22-data.json) | [22 图、22 项检查](previews/theme-worm-combinations-v22-checks.json) | [before](previews/theme-worm-combinations-v22-before-capture.json) |
| Executions | 12／4 | [data](previews/theme-worm-executions-v22-data.json) | [12 图、15 项检查](previews/theme-worm-executions-v22-checks.json) | [before](previews/theme-worm-executions-v22-before-capture.json) |
| 共用适配 | 34／28 | [data](previews/theme-common-adaptations-v22-data.json) | [34 图、6 组检查](previews/theme-common-adaptations-v22-checks.json) | [before](previews/theme-common-adaptations-v22-before-capture.json) |

每项实际页面均覆盖 1440×900 桌面、390×844 手机、320×844 窄屏与 200% 根字号；Worm 字号放大使用 720×900，共用适配使用 720×1000。全页截图从文档顶部采集，等待字体就绪、关闭动画并移开指针。十一张 native dialog 采用真实 390×844 视口、正文起点为 0、固定头尾与可滚动正文：Assets 四张、Combinations 三张、Executions 两张、共用两张；其余按记录采集全页。窄屏和字号放大图为辅助证据，不计算额外业务页面。

关键状态入口包括[钱包选择](previews/theme-worm-assets-v22-selection-mobile.png)、[单仓 Cash Out](previews/theme-worm-assets-v22-cashout-mobile.png)、[批次核对](previews/theme-worm-assets-v22-batch-review-mobile.png)、[批次暂停](previews/theme-worm-assets-v22-batch-paused-mobile.png)、[草稿离开](previews/theme-worm-combinations-v22-editor-dirty-mobile.png)、[失效选择](previews/theme-worm-combinations-v22-editor-unavailable-mobile.png)与[清理后恢复](previews/theme-worm-combinations-v22-editor-unavailable-recovered-mobile.png)、[预览防护](previews/theme-worm-combinations-v22-preview-guards-mobile.png)、[独立授权](previews/theme-worm-executions-v22-authorize-mobile.png)、[终止核对](previews/theme-worm-executions-v22-terminate-mobile.png)、[未知结果](previews/theme-worm-executions-v22-unknown-desktop.png)、[步骤证据](previews/theme-worm-executions-v22-evidence-mobile.png)、[资料离开确认](previews/theme-common-adaptations-v22-profile-leave-mobile.png)和[复制失败](previews/theme-common-adaptations-v22-access-copy-error-mobile.png)。完整采集与其他状态见各域 checks。

原型检查覆盖指定尺寸、整页横向溢出、重复 ID、JavaScript 异常、外部请求与被检查主要控件的 44px 目标；相应结果通过。键盘、Escape／焦点恢复、错误恢复、只读与合成权限差异按 checks 的有界用例验证。旧 React 执行详情桌面记录 `documentWidth=1614 > 1440`，作为现状证据保留，不写成所有旧界面均无溢出。

独立首次审阅逐张打开全部 120 张输入图和两张 v21 参照，按关键合同抽样阅读四份 HTML、数据、检查、业务源码与长文档，未通读全部实现。初判 `fix`：体系延续和多数 fidelity 项通过，但下列三项阻止达到本批 QUALITY BAR，不能把局部 pass 扩大成无任何问题。

| 初审问题 | 修正及复审结论 |
| --- | --- |
| 组合编辑器移除失效市场后仍不能保存 | Save 按当前失效选择计算，保留原错误图，新增仅剩有效 Bitcoin、Save 可用的恢复图及操作记录；resolved |
| 执行完成证据时间晚于步骤 Last update | 从同一步 fixture 字段渲染时间并同步相关 Run／unknown 更新，保留全部证据，新增两项时间顺序断言；resolved |
| Worm 六页 200% 顶栏头像字母溢出 | 三份 Worm HTML 同步 rem 头像规则，重捕六张同名 zoom 及默认桌面／手机回归图；resolved |

最终复审逐张重读修正包 47 张 required PNG（包含新增恢复图），只复核上述三项及相关回归，结合初次完整审阅给出 `ship`；没有重跑整站验收或第二次 detector。静态图片本身不能证明完整动态交互或正式业务健康。

一次合并 detector 的八项例外已通过 CLI 按文件落库：四项是用户明确批准的 Inter，四项是 white-on-mint hover 静态误判。后者有 28 个实际深字／mint 渲染样本支持，最低对比为 15.145:1。低对比规则在 CLI 中只能使用 `*` 标识，配置范围限制于四个 v22 HTML；这不是全局禁用所有规则，也不是重新选择字体和主题的依据。

## 业务与实现依据

页面映射以[会员入口](../../../ui/src/app/member/app.tsx)、[管理员入口](../../../ui/src/app/admin/app.tsx)及[共享账户页](../../../ui/src/app/shared/pages/account-center.tsx)、[共享 Help](../../../ui/src/app/shared/pages/help.tsx)、[注册页](../../../ui/src/app/member/pages/register.tsx)为准。Worm 语义沿用 [Worm Trading](../../design/trading/worm-trading.md)、[单仓 Cash Out](../../design/trading/worm-position-cash-out.md)、[Cash Out 批次](../../design/trading/worm-position-cash-out-batches.md)、[市场组合](../../design/trading/worm-market-combinations.md)、[执行预览](../../design/trading/worm-execution-preview.md)与[订单执行](../../design/trading/worm-order-execution.md)。管理员字段与权限沿用[账户访问控制](../../design/identity-access/account-access-control.md)。这些已实现业务契约不等于 v22 主题重构已落地。

## 未验证边界与下一步

全部界面使用 local synthetic 数据；当前 React 通过截获合成响应生成对照，不发送真实 API 写入。Google／Phantom 身份、管理员权限执行、真实 Help 资源、凭据创建／撤销、实时余额与供应商数据、plan／revision 并发、签名、Run／下单／Cash Out／链交易及 full-stack smoke 均未在本批验证。原型中的按钮反馈、链接、授权和交易状态只是静态演示，不证明后台接入或生产验收。

本任务未启动或停止服务／容器；采集使用的临时浏览器上下文已关闭。借用的既有 [localhost:4000](http://localhost:4000) Vite 保持原状：PID 308228，工作目录 `/home/yege/work/athena/ui`，既有运行日志位于根目录 `.run/athena-local-runtime`，由原环境所有者管理；不因本批设计收尾停止共享环境。DESIGN.md 与 `.impeccable/design.json` 为既有缺失，本次作为已确认世界的普通扩展只记录，不创建或修复。

v20、v21、v22 三批所展示的主视觉均已确认。后续[总覆盖清单](theme-refactor-coverage.md)已完成入口与继承状态归属核对，[技术方案](../../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)及[实施计划](../../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)已整理；已确认决定直接沿用。Trader Sync 专属页面仍暂缓重排，不能将本批批准扩大为无范围限定的全站定稿；正式代码与真实验收尚未开始。
