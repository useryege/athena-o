# 全站 UI 视觉主题重构需求

> 当前实施：单一深色正式源码已落实，T1–T9 完成并通过独立任务审阅；实现版本 `20167913`。37 条改版入口、2 条 Appearance 删除、4 条默认／兜底和 8 条 Trader Sync 共享影响回归已完成相应验证。最终审阅、环境收尾与通知状态统一见[实施与验收记录](../../testing/web-ui-theme-refactor-acceptance.md#交付状态与环境收尾)。
>
> 设计依据：Nansen 方向、v1–v22 已批准范围及[一致性修订契约](theme-consistency-contract.md)／[v23](previews/theme-consistency-v23/README.md)继续有效。本文各版本章节记录批准时点的材料与限制，不是当前实现进度。625 份批准资产（包括预览目录两份 README）保留原字节，静态原型通过不替代正式 React 或真实接入证据。
>
> 范围与实现：[覆盖清单](theme-refactor-coverage.md)、[技术方案](../../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)、[实施计划](../../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)。Trader Sync 专属页面暂缓重排，Service Status 的 Trader Sync 页签已改版。共享 T1／T2 已完成；Token／Nansen 接入与导航仍归独立计划，未在本次开启。
>
> 当前契约：[共享应用壳](../../design/web-ui/application-shell.md)、[会员应用壳](../../design/web-ui/member-application-shell.md)、[管理员应用壳](../../design/web-ui/administrator-application-shell.md)、[账户资料与本地偏好](../../design/identity-access/account-profile-and-preferences.md)。验收有准确版本与外部边界，不宣称全部真实交易或身份供应商均通过。

## 背景与问题

用户喜欢 Nansen 官网的深色主题、青绿色强调、文字层次和整体视觉质感，希望 ATHENA 整体采用相近的视觉方向，并明确不再支持白天／夜间两套主题。

ATHENA 当前已有实际业务界面、主题配置和共享组件。本次从整体视觉系统入手，进一步统一页面、控件和数据展示；业务工作台仍需满足查询、比较、核对与连续操作的使用需求。

用户要求先将决定保存在仓库，随后逐项确认视觉效果，再按完成的设计文档组织执行与交付。本文随确认结果维护；未确认提案不能作为最终实现要求。

## 已确认目标

**本轮主旨是重构 ATHENA 当前 `ui/` 下的全站前端界面。** 用户再次明确：所有 UI 设计均围绕现有前端改版展开，覆盖会员端、管理员端及其共享视觉系统，以 Nansen 为主要参考，统一主题、配色、字体、布局、组件和交互状态。

钱包样板用于具体展示和确认可复用规则；页面提案已经覆盖本轮现有页面，实施与验收安排见 coverage／spec／plan。此目标说明不改变现有业务与权限语义，不要求更换技术栈。正式实现及相应验收已完成，最终交付收尾状态见总报告；Trader Sync 专属重排继续暂缓。

> ATHENA 采用单一深色主题，以近黑背景、青绿色强调色、清晰的文字与数据层级、克制的边框和动效，形成统一的专业链上分析平台风格。

风格描述为 **“深色极简科技风，带有专业金融终端的气质”**。这是本项目用于沟通的描述性名称，不是供应商的官方风格分类。

### 主要视觉参考约束（已确认）

用户已再次明确：非常喜欢 Nansen 的 UI 风格，ATHENA 的整体 UI 必须参考 Nansen 进行设计。这是持续约束后续工作的产品偏好，应在跨任务上下文中保留。

- Nansen 是全站主题设计与视觉验收的主要参考；应逐项对照配色、字体层次、面板、空间安排和交互细节，而不是仅把界面改成深色后泛称为科技风。
- 页面与组件设计应结合本项目的业务内容，提交能够对应参考特征的视觉效果，按用户逐项确认结果落实。
- 原有橙色主题及双主题描述的是重构前代码状态，不是继续沿用旧风格的设计约束。
- 以本轮记录的参考证据和后续用户确认的视觉样例为依据。Nansen 官网以后改版，不自动改变 ATHENA 已确认的设计。
- 已确认的 v1 基础配色与主次按钮层级、v2 字体与数字排版、v3 导航／页头／页面密度、v4 第一组通用组件与状态见下节；其余页面、组件和动效仍按后续逐项确认范围落实。

### 主题与配色搭配（已确认）

用户特别强调：Nansen 的主题与配色非常契合自己的审美，ATHENA 的主题搭配必须以此为核心参考。这一偏好适用于整体颜色关系，而不只是选取一个青绿色主色。

后续设计应一起对照近黑背景、深色面板、白／灰文字、青绿色强调和低强度边框的搭配，关注明暗层次、冷暖倾向、强调色面积与各颜色的使用位置。v1 已确认为基础配色基准，其余交互状态通过后续视觉确认落实；品牌青绿色与业务盈亏、警告、错误等状态仍需保持清晰的语义区别。

| 维度 | 已确认方向 | 当前落实依据 |
| --- | --- | --- |
| 主题模式 | 全站统一为单一深色主题，取消明暗切换 | T1 已完成入口、偏好、接口和持久化清理；profile／access 契约保留 |
| 页面背景 | 沿用 v1 近黑背景、深色面板和分隔层次 | v1–v22 页面依据与修订契约，浮层消费同一 Provider |
| 强调色 | 青绿色 `#00FFA7`；沿用主次按钮与 v4 反馈语义 | 最终绘制、hover／active 和业务语义按修订契约验证 |
| 文字与数据 | Inter、JetBrains Mono、v2 字号与数字对齐 | 保留各业务精度；实际字号增幅、局部重排和跨平台中文回退纳入验收 |
| 分组与布局 | 沿用 v3 壳及各页面已确认结构 | 51 项入口归属及状态矩阵已形成，暂缓范围见 coverage |
| 边框与动效 | 克制边框、平滑反馈 | 控件强边界 `#61717B`；0.16／0.2s 反馈与 reduced-motion 按技术方案实施 |
| 页面气质 | 专业、冷静、现代，适合链上与金融数据研究 | 按已确认页面实施；本轮不新增产品影像或装饰性场景 |

### v1 配色与按钮视觉基准（已确认）

用户查看第一版样板后反馈“没问题，我喜欢这个感觉”。据本轮明确的审阅范围，确认 v1 的整体配色、背景／面板深浅、文字颜色、青绿色使用以及主次按钮的视觉层级，作为后续设计基准。

| 用途 | 已确认 v1 基础色 |
| --- | --- |
| 页面底色 | `#06080B` |
| 内容面板 | `#0F1114` |
| 较亮交互层次 | `#181D22` |
| 装饰分隔线 | `#252A30` |
| 主要文字 | `#FFFFFF` |
| 辅助文字 | `#9FA0A1` |
| 青绿色强调 | `#00FFA7` |
| 主要按钮文字 | `#06080B` |

主要操作采用亮青绿色实心按钮；次要操作采用深色底与细边框。整体保持大面积深色，突出必要的操作和信息。控件边界、焦点和业务状态的具体规则随组件设计补齐。

确认依据为[样板说明及确认记录](previews/README.md)、[桌面截图](previews/theme-sample-v1-desktop.png)和[手机截图](previews/theme-sample-v1-mobile.png)。后续字体、布局和组件样板应沿用本配色基准，另存版本，保留本轮审阅原件。

本次未整体确认样板字体、字号、页面构图、数据字段、统计定义、表格结构或全部交互状态，也不表示正式 UI 已实现。样板中的钱包和数值仍是虚构演示。

### v2 字体与数字排版视觉基准（已确认）

用户查看 v2 桌面和手机截图后，对字体大小与数字清晰度反馈“可以 舒服”。确认 Inter 用于英文标题、正文和数字，JetBrains Mono 用于地址与哈希；采用样板的字号、字重和行高层级，以及等宽数字、金额右对齐的阅读方式。

主要层级为：页面标题桌面 28px／手机 24px，区块标题 20px，正文 16px，表格与控件 14px，辅助信息与地址 13px；主要金额桌面 40px／手机 32px，中等视口 36px。完整字重、行高与适用边界见[字体与数据排版基准](typography-proposal.md)。

确认范围是截图体现的视觉效果，包括金额、百分比及辅助文字的清晰度。完整展开／复制交互、业务精度与舍入规则、缺失状态语义、页面布局及完整组件状态未由本轮整体确认；中文跨系统回退和正式页面仍需验证。

后续样板沿用 v1 配色与 v2 字体基准，保留[桌面截图](previews/theme-typography-v2-desktop.png)、[手机截图](previews/theme-typography-v2-mobile.png)及[确认记录](previews/theme-typography-v2-approval.json)作为依据，不将样板批准等同于正式 UI 已实现。

### v3 导航、页头与页面密度视觉基准（已确认）

用户查看桌面、手机及手机导航展开截图后，对整体布局与疏密程度反馈“舒服”。本轮确认截图中的常驻侧栏、导航分组、顶部位置与账户入口、页头操作位置、查询条件分组、内容留白和数据密度，以及手机导航抽屉与逐币摘要排列。

用于复现截图的主要参数是桌面侧栏 224px、顶栏 64px、宽屏内容留白 32px、手机水平留白 20px、手机抽屉宽 280px。页面标题与说明在左，页级操作在右；手机按阅读顺序重排。完整设置与范围见[布局视觉基准](layout-proposal.md)及 [v3 确认记录](previews/theme-layout-v3-approval.json)。

未展示的侧栏收起态、二级与账户展开菜单、完整组件状态、管理员及其他业务页面继续设计和验证；不将会员样板当作全站每个页面已确认。业务字段、计算、权限和 API 接入继续依据各自需求，正式 UI 尚未修改。

### v4 第一组通用组件与状态（视觉已确认）

用户于 2026-09-14 查看桌面状态总览、桌面确认弹窗和手机确认弹窗后反馈“舒服，可以。”。本轮确认已展示的表单与按钮状态、成功／警告／错误／信息反馈、静态加载骨架、成功空态，以及弹窗的遮罩、留白和操作层级。

反馈采用低饱和深色背景，配合成功绿 `#55D9A1`、警告黄 `#E6BF72`、错误粉红 `#F58C9B`、信息蓝 `#9BC6F3`，并保留图标和文字说明。确认弹窗桌面最大宽度 480px，手机左右留白 20px、按钮纵向排列；继续编辑用描边按钮，放弃草稿用粉红色实心按钮。完整色值、尺寸和范围见[组件与状态视觉基准](components-proposal.md)及 [v4 确认记录](previews/theme-components-v4-approval.json)。

完整手机总览与交互预览提供了链接，专门的键盘焦点图保存在文档中，不将它们当作全部状态均已逐图确认。下拉菜单、表格筛选、分页、其他控件、动效和业务页面继续设计；样板校验与本地草稿操作不新增全站交互契约，正式 UI 尚未修改。

### v5 下拉菜单、表格筛选与分页（展示视觉已确认）

用户查看桌面菜单展开、手机菜单展开和手机筛选结果截图后反馈“可以”。确认已展示的灰边深色菜单、深青底与青绿勾选、搜索与排序控件层级、可清除的地址筛选标签、已加载数量和“加载更多”布局，以及手机筛选后的全部加载状态。具体范围见[列表控件基准](list-controls-proposal.md)及 [v5 确认记录](previews/theme-list-controls-v5-approval.json)。

截图之外的全部交互、追加失败、实际请求与数据范围、其余页面不由本轮整体确认。原始 HTML、截图和审阅记录保持原样；钱包示例是全站组件设计的审阅载体，正式界面尚未改造。

### v6 现有 Trader Sync 首页（页面视觉与默认折叠已确认）

用户查看当前 React 桌面、新主题桌面和手机完整截图后反馈“认可”。确认左侧活动主区、右侧目标区的桌面布局，市场／买卖方向／成交额／份额的阅读层级，手机目标区默认收起和活动重排，以及身份快照、Position ID 和来源入口进入 `Identity & position`、默认收起并按需展开的处理。

完整钱包、备注和公开名称、结算时间继续直接显示；当前 Telegram 绑定、活动投递结果、监控中断和 finality 异常保持各自层级。沿用 Trader Sync 的 UTC+8 日期、50／100 条与 Previous／Next 规则。具体参数、保留字段与确认边界见[首页视觉基准](trader-sync-home-proposal.md)和 [v6 确认记录](previews/theme-trader-sync-v6-approval.json)。

当前版与新主题采用同一批演示数据，截图是本页改版依据。其他展开态截图、全部网络和权限状态、完整交互及其他页面不因本次认可整体视为已确认。原始 HTML、截图与审阅记录保持原样，批准时正式 `ui/` 尚未改版。

### v7 现有添加交易员页（展示视觉已确认）

用户于 2026-09-14 查看新主题桌面和手机完整截图后，对布局、信息密度与操作层级反馈“舒服”。确认页头下方的 Find trader 输入区、桌面左侧身份与 P/L 核对及右侧确认区，以及手机按输入、资料、P/L、确认排列的阅读顺序。已有结果时 Resolve again 为次要操作，Confirm subscription 为主要操作。

已展示的默认 1Y 金额与曲线、六周期控件、备注、确认有效期、配额和已连接 Telegram 的视觉层级作为本页基准。具体参数与边界见[添加页视觉基准](trader-sync-add-proposal.md)和 [v7 确认记录](previews/theme-trader-add-v7-approval.json)。

初始输入、过期、缺失、精确曲线值等辅助截图未逐图确认；完整交互、业务/API 契约及其他页面不因本次反馈整体视为已确认。原始 HTML、截图、演示数据、检查和审阅记录保持原样，批准时正式 `ui/` 尚未改版。

### v8 现有订阅管理列表（展示视觉已确认）

用户于 2026-09-14 查看新主题桌面／手机 Current 完整截图后，对信息密度、状态区分和手机排版反馈“舒服”。确认桌面按 Trader、Monitoring、Old notifications、Manage 四列对齐，手机按交易员依次展示身份、监控、通知和详情入口。

已展示的 Current / Cancelled 控件、3 / 10 当前配额、资料更新时间、完整钱包、监控两项时间、六类旧通知计数及队列保留说明的视觉层级作为本页基准；正常监控、暂停与中断，以及通知失败／未知各自保持清楚的状态表达。具体参数及边界见[订阅列表视觉基准](trader-sync-subscriptions-proposal.md)与 [v8 确认记录](previews/theme-trader-subscriptions-v8-approval.json)。

Cancelled 列表、空态、加载及失败等辅助截图未逐图确认；视图切换、复制、分页、权限、网络和写操作的完整流程不因截图反馈整体视为已确认。原始 HTML、截图、数据、检查与审阅记录保持原样，批准时正式 `ui/` 尚未改版。

### v9 现有订阅详情（展示视觉及默认摘要已确认）

[订阅详情视觉基准](trader-sync-subscription-detail-proposal.md)延续 v1–v8 已确认的主题、字体与组件，用同一份虚构资料对照当前 React 和新版页面。桌面左侧呈现资料、当前监控和观察历史，右侧集中备注、订阅操作及旧通知；手机按资料、当前状态、操作、历史、通知依次阅读。

用户于 2026-09-14 对桌面／手机完整页和手机取消确认弹窗反馈“舒服，继续设计”，并确认历史记录摘要／详情展开方式：默认仍显示主要区间、实际开始时间未知、恢复时间与可能遗漏提示，展开再查看记录排序时间、generation／epoch、恢复边界及不确定性。取消确认弹窗保留完整身份与全部后果，管理操作遵循现有状态和权限语义。

已展示的布局、操作层级、默认摘要方式及手机取消弹窗作为本页基准，具体范围见 [v9 确认记录](previews/theme-trader-detail-v9-approval.json)。桌面取消弹窗、展开细节与其他辅助状态未逐图确认；原型和检查记录不是正式 UI 实现或生产行为验收。

### v10 现有 Notifications 页面（所展示视觉已确认）

[Notifications 视觉提案](notifications-proposal.md)延续 v1–v9 的单一深色主题、Inter 字体、语义反馈和确认弹窗，将当前 Telegram 连接与新的设置尝试分开呈现。现有绑定在替换成功前保留；原来处于 Connected 的连接继续有效，取消设置保留原绑定；设置按打开 Bot、点击 Start、返回 Athena 三步组织，并提供同一链接、手动命令和二维码。

原型覆盖连接、设置中、重连、其他标签页、到期、失败、不可达、Bot 不可用、加载／错误和解绑确认。审阅后 Expired／Failed 恢复区只有一个主要 Create new link；Failed 徽标使用错误色，到期与需要注意使用琥珀色。Bot 不可用时禁用 Configure／Reconnect，已有绑定仍允许 Disconnect；不可达时保留绑定。完整截图与检查见[样板索引](previews/README.md)。

v10 已确认所展示的已连接桌面／手机及桌面重新连接视觉，范围见 [确认记录](previews/theme-notifications-v10-approval.json)。其余辅助状态未逐图确认。原型只使用固定虚构 fixture 和 `example.invalid` 二维码，连接、重连、取消设置、刷新、打开 Telegram 和确认解绑等业务动作仅显示本地 notice；复制、导航折叠、样例状态切换与弹窗开关保留本地交互；它没有验证真实 API、竞态、3 秒轮询、倒计时、`sessionStorage`、账户切换、Bot 可用性、条件式 Trader Sync 返回或 owner 草稿。批准时正式 `ui/` 未修改，现有错误来源和状态处理在后续实现中必须保留。

### v11 Account Center / Profile 页面（所展示视觉已确认）

[Account Center / Profile 视觉提案](account-profile-proposal.md)延续 v1–v10 的单一深色、Inter、表单反馈和确认弹窗规则，把当前重复的保存身份、首字母头像和编辑区域合并为一个 Profile 面板。桌面保留账户局部导航，手机使用 Account section 选择器；Username 保持只读，Display name 独立编辑，Standard 只作展示，名称状态和 Reset／Save 紧随表单。Appearance 从目标单一深色界面移除，非主题浏览器偏好继续保留；Security 仍只对启用 API Key 的普通会员显示。

当前 React 与 v11 使用同一无头像虚构 fixture，以真实首字母 `A` 回退对照。12 张最终原型图覆盖默认、编辑、无效输入、冲突、保存／上传错误、离开确认、320px 和 200% 文字；2 张当前 React 对比图只观察到 bootstrap GET。四个视口检查通过，独立审阅对全部 14 张图片处置为 `ship`，未提出需要修正的视觉问题。该处置只说明本地原型达到审阅交付标准。用户另对桌面资料页、手机编辑状态及手机离开确认反馈“舒服”，展示范围见 [v11 确认记录](previews/theme-account-profile-v11-approval.json)；其他辅助状态未逐图确认，批准时正式 `ui/` 未修改。真实资料与头像写入、CAS、已有头像 Remove、身份切换、授权、路由／`beforeunload` 和管理员视觉仍需后续实现与验收，详见[审阅记录](previews/theme-account-profile-v11-review.json)。

### v12 Account Center / Security 页面（所展示视觉已确认）

[Account Center / Security 视觉提案](account-security-proposal.md)在 v1–v11 已确认世界中延续 Account Center 页头、桌面局部导航、手机分区选择器和紧凑当前账户身份。主内容先呈现 Connect AI，再呈现 API keys；前者生成同一账户 API Key 的一次性完整 HTTP 接入说明，后者管理已签发元数据。列表只显示 ID、Issued、Expires、Revoke，桌面使用紧凑表格，低于 800px 使用分隔行；不新增 scope、provider、最近使用、状态或调用量。

一次性 API Key 结果只能通过 Done 关闭。AI 结果保留完整 base URL、发现文档、验证入口、预期账户 UUID、Bearer 与权限说明；手机长内容使用滚动正文和固定标题／操作区。ready 只表示 ATHENA 接受了属于预期账户的凭据，不表示外部 AI 已连接。撤销确认显示精确 ID 与即时失效后果，安全焦点位于 Keep key。Security 仍只对启用 API Key 的普通会员可见；页面不配置 Nansen 平台 Key 或外部 AI 提供商密钥，Nansen 环境变量与共享配额决定保持不变。

11 张最终原型图与 2 张当前 React 对照图均为带来源记录的 Chromium 截图。四个视口检查通过；当前 React fixture 只发出 bootstrap 与 token-list GET，没有写入。独立审阅对 13 张图和本地原型处置为 `ship`，未提出需要修正的视觉问题。该审阅结论只覆盖视觉提案。用户另对桌面／手机默认页及手机 AI 连接说明弹窗反馈“舒服，继续”，范围见 [v12 确认记录](previews/theme-account-security-v12-approval.json)；其他辅助状态未逐图确认，正式实现及生产验收仍未完成。原型仅使用无效合成 secret、固定元数据和 `example.invalid`，不执行真实签发、验证、撤销或外部 AI 调用；正式实现必须保留当前源码的错误、单飞、账户归属、陈旧结果丢弃与中止清理契约，详见[审阅记录](previews/theme-account-security-v12-review.json)。

### v13 Account Center / Access & session 页面（所展示视觉已确认）

[Account Center / Access & session 视觉提案](account-access-proposal.md)在 v1–v12 已确认世界中延续紧凑账户身份与局部导航，把十项 Module access 和 `2 full · 3 read · 5 none` 摘要放在 Current session 之前；API Key 与 Profit Sharing 独立展示，时间、修订与版本默认折叠。Token 只从展示列表排除，完整授权仍保留十一模块；Tier 只展示，权限标签保持 `Read & write`、`Read only` 与 `No access`。

Pending 清楚区分 Google／Phantom 已验证身份与尚未开通的业务访问，并保留刷新与退出。判定仍严格遵循现有模型：禁用登录为 Blocked；管理员、Profit Sharing 或任意非空模块 grant 为 Active；API Key 单独启用不构成 Active。本次 Pending 演示数据没有启用 API Key，因此截图不显示 Security，但现有规则仍允许为启用 API Key 的普通会员显示 Security。

9 张最终原型图与 3 张当前 React 对照图均有 Playwright 来源记录。四个视口检查通过，独立审阅覆盖全部 12 张图、源码契约与浏览器证据并处置为 `ship`，未提出需要修正的视觉问题。该结论只覆盖视觉提案。用户另对桌面／手机默认页及手机 Google 待授权页反馈“舒服”，范围见 [v13 确认记录](previews/theme-account-access-v13-approval.json)；其他辅助状态未逐图确认，批准时正式 `ui/` 未修改。原型只执行本地状态切换、展开、复制、忙碌、错误恢复和退出提示，不执行真实轮询、授权检查、授权变更或退出；正式实现仍须保留 15 秒／焦点刷新、真实错误、身份／realm 防护、账户切换、权限竞态与退出行为，详见[审阅记录](previews/theme-account-access-v13-review.json)。

### v14 会员登录与注册页面（所展示视觉已确认）

v14 把会员 `/login` 与共享 `/register` 延伸到 v1–v13 已确认的 Nansen 单一深色视觉体系。登录采用 448px 居中单列面板和两个同级 48px provider 按钮；注册依次展示已验证身份、用户名及可用状态、永久名称说明、青绿色创建动作和身份切换。桌面内边距 32px；手机外侧 20px、面板内侧 24px，低于 360px 时内侧 20px；标题沿用桌面 28px／手机 24px。

Google 与 Phantom 继续作为独立身份，Phantom 只使用浏览器注入的 Solana provider，手机布局不新增深链。注册仍从已验证 provider 和服务端 ticket 开始，用户名永久且以服务端安全规则与可用性为准，业务权限由管理员另行授予。完整截图、状态边界和证据见[会员登录与注册视觉提案](auth-proposal.md)。独立审阅的 `ship` 只覆盖视觉提案与本地演示。用户另对登录页与 Google 注册页的桌面／手机四张图反馈“没问题”，具体范围见 [v14 确认记录](previews/theme-auth-v14-approval.json)；其他辅助状态未逐图确认，正式 `ui/`、真实身份流程和生产验收均未完成。

## 范围与角色

整体主题目标覆盖 ATHENA 自有的会员与管理员界面，以及两者使用的共享组件。登录、账户中心、业务列表、详情、表单、弹窗、菜单和系统状态页面均应纳入后续覆盖清单；新增页面沿用完成后的统一视觉规则。

这里确认的是整体视觉方向。具体实施批次、页面优先级、代表页面和各设备的验收范围仍需列明，不预先承诺每个页面都重排布局。

会员与管理员继续使用各自的身份、会话、导航和授权边界。视觉统一不合并两个应用，不改变业务权限、数据归属或已有操作语义。

产品名称保持 ATHENA，模块名称与业务事实继续依据各自需求。参考 Nansen 的视觉方法不等于采用其名称、Logo 或把其展示数据当作 ATHENA 数据。

## 业务规则与不变量

- 目标界面只提供统一的深色外观，不再提供浅色、深色或跟随系统的主题选择。
- 系统颜色偏好、登录／退出、页面刷新和应用切换均不应把目标界面切回浅色。页面初始加载阶段也需覆盖，避免先出现浅色背景再切换。
- 青绿色的品牌强调与盈亏、成功、警告、失败等业务状态分别表达。状态文字、图标和数值含义保持准确，不能为了视觉统一而把所有状态改成同一种含义。
- 表格、图表、表单和交易明细保持可读、可操作；长地址、精确数值、复制、筛选、分页、排序和错误反馈按各自业务规则工作。
- 已有键盘焦点、禁用状态、错误提示、响应式能力和减少动态效果偏好应继续得到支持；具体视觉参数与验收矩阵在设计中确定。
- 主题重构不替代钱包战绩的 Nansen API 接入需求，也不改变其他业务的计算、采集、权限和交易规则。

## 使用过程与边界场景

用户从登录入口、业务链接或账户页面进入时，看到一致的深色界面；登录后进入其有权限的应用和模块。切换页面、展开菜单、打开弹窗、加载数据或显示异常时，相关组件遵循同一套视觉规则。

本主题目标没有独立的业务状态机。加载中、无数据、权限不足、网络失败、操作成功和危险操作等状态沿用原有业务语义，后续通过具体示例确认其外观。

需要单独核对的边界包括：系统偏好为浅色、旧浏览器缓存含主题设置、账户仍保存旧偏好、登录前后切换、会员与管理员入口、部署子路径、长数据和较窄视口。具体技术清理方式留给技术设计，不为保留旧明暗模式新增兼容路径。

## 当前正式实现

| 当前契约 | 实现与验证依据 |
| --- | --- |
| 两份 HTML 首屏固定深色；入口级 Provider 覆盖 bootstrap、登录、注册、错误及业务页 | [tokens.css](../../../ui/src/app/styles/tokens.css)、[AthenaThemeProvider](../../../ui/src/app/shared/athena-theme.tsx)、[会员 HTML](../../../ui/src/app/index.html)、[管理员 HTML](../../../ui/src/app/admin/index.html) |
| 单一颜色角色、Inter 正文／数字、JetBrains Mono 地址／哈希与本地 OFL 字体 | [最终色角色](../../../ui/src/app/shared/athena-color-roles.ts)、[字体](../../../ui/src/assets/fonts.css)；实际字体、原生 200% 缩放与根字号 200% 分开取证 |
| 两端 Appearance、theme 模型、系统监听、账户主题 API／schema／生成投影全部删除 | [账户资料与本地偏好](../../design/identity-access/account-profile-and-preferences.md)；旧 URL 落既有 404，无主题兼容重定向 |
| realm 独立本地偏好保留页大小、排序、侧栏、banner 和返回位置 | [ViewPreferencesService](../../../ui/src/app/shared/services/view-preferences-service.ts) 读写仅接受当前字段，不再读写或跨设备同步 theme |
| 身份／自助、运维、钱包／Solana／治理、市场／赛事／Managed OO、Worm 按批准页面重排 | [51 项入口与 S1–S14 状态](theme-refactor-coverage.md)及[正式验收](../../testing/web-ui-theme-refactor-acceptance.md) |
| 8 条 Trader Sync 专页保留业务布局，共享主题／字体／壳已生效并回归；Status 页签已重排 | [Trader Sync UI 设计](../../design/web-ui/trader-sync-activity-alerts.md)仍拥有业务契约；逐路由证据见覆盖清单 |

原始值精度、profile／access CAS、realm 与身份代际、撤权清理、陈旧结果、单飞和破坏性操作门槛保持当前业务契约。真实本地读取与受控成功状态分开记录；未启用服务的 503 不算业务读取成功，缺少真实记录的详情不以 fixture 冒充真实成功。

## 参考证据与适用边界

参考来源于 2026-09-13 对 [Nansen 官网](https://nansen.ai/) 的页面、公开源码与媒体素材检查：

已将实际查看的官网功能区域截图与产品视频帧保存到[仓库视觉参考](reference/nansen-2026-09-13/README.md)，供后续任务直接对照，并记录来源和观察边界。

- [官方品牌规范](https://nansen.ai/brand)列出主色 `#00FFA7`、Uniforma 标题字体和 Inter 等字体。ATHENA 的 v1 基础色与 v2 Inter 主字体、地址专用 JetBrains Mono 已分别通过样板确认；未选择 Uniforma，不把供应商的完整字体系统直接当作 ATHENA 已选方案。
- 官网使用 Framer、React／Motion、CSS 视觉效果和产品展示视频。选取它的视觉风格，不直接决定 ATHENA 更换现有技术栈。
- [首屏视频素材](https://framerusercontent.com/assets/lKYEyMEXdN62XBSWgyqvysfpQ.mp4)呈现带透视角度的产品屏幕与青绿色光影；视频制作工具未知。本次浏览器未完整加载该视频，另行查看素材帧仅用于理解视觉构成，不构成加载性能或播放流畅度结论。
- 当前参考主要来自公开官网及其产品影像，不等于已完整审查 Nansen 登录后的交互和业务界面。

工作台优先吸收颜色、文字层次、面板和交互的一致性。巨型标题、大面积产品影像与装饰性动效是否用于 ATHENA 的具体位置，继续依据页面任务确认。

## 逐项确认记录

| 项目 | 状态 | 确认范围或下一步 |
| --- | --- | --- |
| 总体目标与风格 | 已确认 | 本文“已确认目标”中的文字定义 |
| 主要视觉参考 | 已确认 | 全站 UI 必须以 Nansen 为主要视觉参考，设计与验收具体对照参考特征 |
| 主题与配色搭配 | 已确认 | Nansen 的整套主题配色关系是核心审美参考，覆盖背景、面板、文字、强调和边框 |
| 单一深色主题 | 已确认 | 不再支持明暗切换，整体采用统一深色外观 |
| 基础配色与操作层级 | 已确认 | v1 的背景、面板、文字颜色、青绿色使用和主次按钮层级作为后续基准 |
| 第一轮视觉样板 | 本轮视觉效果已确认 | [v1 样板、确认范围与检查记录](previews/README.md)；不扩展为字体、布局或正式实现的批准 |
| 字体与数据排版 | v2 视觉效果已确认 | [字体与数据排版基准](typography-proposal.md)；字体、字号层级与数字对齐作为基准，交互与业务展示语义仍按各自范围确认 |
| 导航、页头与页面密度 | v3 截图视觉已确认 | [布局基准与截图](layout-proposal.md)：侧栏、页头、信息分组、间距和手机重排；未展示状态与其他页面仍按范围确认 |
| 通用组件与状态 | v4 第一组及 v5 展示视觉已确认 | [组件与状态基准](components-proposal.md)及 [v5 列表控件基准](list-controls-proposal.md)：已展示的表单、反馈、加载／空态、确认弹窗、下拉菜单、地址筛选标签与“加载更多”；其余组件和页面继续按范围设计 |
| 现有业务页面效果 | v6–v14 所展示范围已确认 | [首页](trader-sync-home-proposal.md)、[添加页](trader-sync-add-proposal.md)、[订阅列表](trader-sync-subscriptions-proposal.md)及[订阅详情](trader-sync-subscription-detail-proposal.md)按各自确认记录执行；[v10 Notifications](notifications-proposal.md)按已展示范围确认；[v11 Account Center / Profile](account-profile-proposal.md)按所展示桌面资料页、手机编辑状态及手机离开确认范围执行；[v12 Account Center / Security](account-security-proposal.md)按所展示桌面／手机默认页及手机 AI 连接说明弹窗范围执行；[v13 Account Center / Access & session](account-access-proposal.md)按所展示桌面／手机默认页及手机 Google 待授权页范围执行；[v14 会员登录与注册](auth-proposal.md)按所展示登录页与 Google 注册页桌面／手机范围执行 |
| 管理员页面效果 | v15–v19 所展示范围已确认 | 账户、服务状态、网关、通知列表／详情与治理提案均有各自确认记录；实际状态按 coverage 验证 |
| 其余业务与共用适配 | v20／v21／v22 三批已确认 | 四页、八页、六个 Worm 页面及六项共用适配按各批范围执行；辅助状态不重新逐页审批 |
| 图表效果 | v7 默认 1Y P/L 展示视觉已确认 | [添加页视觉基准](trader-sync-add-proposal.md)中的默认金额、曲线与六周期控件；精确值展开、缺失及其他周期结果未逐图确认，其他业务图表仍待讨论 |
| 动效与装饰素材 | 本轮范围已写入方案 | 沿已确认页面，补充反馈与 reduced-motion；额外展示素材属后续独立需求 |
| 覆盖范围与交付验收 | 已整理，待实施验证 | [51 项入口及状态矩阵](theme-refactor-coverage.md)、技术方案、T1–T10 命令与完成标准 |
| 总体审阅一致性修订 | 已授权修复 | [修订契约](theme-consistency-contract.md)与 [v23 证据](previews/theme-consistency-v23/README.md)，批准时正式源码仍未改版 |

v1 配色、v2 字体与数字排版、[v3 导航／页头／页面密度](layout-proposal.md)、[v4 第一组通用组件与状态](components-proposal.md)的展示视觉均已确认，样板、截图、浏览器检查与确认记录已保存。[v5 下拉菜单、表格筛选与分页](list-controls-proposal.md)所展示的视觉也已获得认可。[v6 Trader Sync 首页](trader-sync-home-proposal.md)作为第一张现有业务页面，已通过同演示数据的前后对照确认整页效果及辅助信息默认折叠；后续继续其他页面设计，正式 UI 仍未改版。

[v7 添加交易员页](trader-sync-add-proposal.md)的桌面／手机完整截图也已获认可，确认布局、密度与操作层级。当前 React 与新主题使用相同演示数据；输入态、P/L 精确值与缺失、确认过期样例继续作为辅助证据保存，未逐图确认。当前确认范围以 [v7 确认记录](previews/theme-trader-add-v7-approval.json)为准，批准时正式前端尚未改版。

[v8 订阅管理列表](trader-sync-subscriptions-proposal.md)的桌面／手机 Current 完整截图已获认可，确认信息密度、状态区分和手机分组。前后对照使用相同演示数据；Cancelled 列表及其他辅助状态未逐图确认，当前范围见 [v8 确认记录](previews/theme-trader-subscriptions-v8-approval.json)。后续继续其他页面设计，批准时正式前端尚未改版。

[v9 订阅详情](trader-sync-subscription-detail-proposal.md)所展示的桌面／手机布局、操作层级、手机取消弹窗和历史默认摘要方式已确认，范围见 [v9 确认记录](previews/theme-trader-detail-v9-approval.json)。[v10 Notifications](notifications-proposal.md)的已连接桌面／手机及桌面重新连接视觉已确认，具体范围见 [v10 确认记录](previews/theme-notifications-v10-approval.json)；未直接展示的辅助状态未逐图确认，批准时正式前端尚未改版。

[v11 Account Center / Profile](account-profile-proposal.md)所展示的桌面资料页、手机编辑状态与手机离开确认视觉已获用户认可，范围见 [v11 确认记录](previews/theme-account-profile-v11-approval.json)。其他辅助状态和管理员页面未逐图确认；原始[审阅记录](previews/theme-account-profile-v11-review.json)保留当时的待确认状态，当前确认以本需求和确认记录为准，正式前端仍未改版。

[v12 Account Center / Security](account-security-proposal.md)所展示的桌面／手机安全设置页及手机连接说明弹窗视觉已确认，范围见 [v12 确认记录](previews/theme-account-security-v12-approval.json)。其他辅助状态未逐图确认；原始[审阅记录](previews/theme-account-security-v12-review.json)保留当时的待确认状态，当前确认以本需求和确认记录为准。正式前端仍未改版，真实凭据请求和权限边界尚未由该原型验收。

[v13 Account Center / Access & session](account-access-proposal.md)所展示的桌面／手机权限与会话页及手机 Google 待授权页视觉已确认，范围见 [v13 确认记录](previews/theme-account-access-v13-approval.json)。原始[审阅记录](previews/theme-account-access-v13-review.json)保留当时的 `awaiting_user_review` 状态；当前确认以本需求和确认记录为准，其他辅助状态未逐图确认，正式前端实现与生产验收仍未完成。

[v14 会员登录与注册](auth-proposal.md)视觉提案、11 张提案图、4 张当前 React 对照图和浏览器证据已完成，独立[审阅记录](previews/theme-auth-v14-review.json)处置为 `ship`，未要求代码修正。该处置只覆盖视觉提案与本地演示。用户对四张展示图反馈“没问题”，范围见 [v14 确认记录](previews/theme-auth-v14-approval.json)；原始审阅保留提交时的 `awaiting_user_review` 状态。其他辅助状态未逐图确认，正式前端实现、真实身份流程及生产验收仍待后续完成。

[v15 管理员导航与账户管理](admin-accounts-proposal.md)已形成 11 张提案图、3 张当前 React 对照图及本地证据，所展示的桌面账户管理、手机目录、手机权限详情和手机确认弹窗视觉已确认。它沿用 224px 管理员侧栏、280px 手机抽屉、单一深色配色与既有字体，将 `/admin/accounts` 组织为 304px 目录及默认 Access 的三页签详情。最终[审阅记录](previews/theme-admin-accounts-v15-review.json)仅确认初审发现的抽屉、手机确认操作和导航图标三项修正已经完成；用户另对四张展示图反馈“可以”，具体范围见 [v15 确认记录](previews/theme-admin-accounts-v15-approval.json)；其他辅助状态未逐图确认，原始审阅保留提交时的 `awaiting_user_review` 状态。正式前端及真实账户流程未修改或验收。

第一版[HTML](previews/theme-sample-v1.html)、[桌面截图](previews/theme-sample-v1-desktop.png)与[手机截图](previews/theme-sample-v1-mobile.png)保留原始审阅状态，包括当时的“待确认”标记；当前确认状态以本需求及[确认记录](previews/theme-sample-v1-approval.json)为准。

v2 的 [HTML](previews/theme-typography-v2.html)、截图与[原始审阅记录](previews/theme-typography-v2-review.json)同样保留当时状态，当前确认范围以 [v2 确认记录](previews/theme-typography-v2-approval.json)为准。

v3 的 [HTML](previews/theme-layout-v3.html)、三张截图和[原始审阅记录](previews/theme-layout-v3-review.json)保留提交时的“待确认”状态，当前范围以 [v3 确认记录](previews/theme-layout-v3-approval.json)为准。

v4 的 [HTML](previews/theme-components-v4.html)、截图和[原始审阅记录](previews/theme-components-v4-review.json)同样保留提交时状态，第一组视觉的当前确认范围以 [v4 确认记录](previews/theme-components-v4-approval.json)为准。

按用户后续要求，同一批可以集中展示多个页面供审阅，复用通用规则的辅助状态不逐页重复审批。已确认决定持续有效；实质业务分歧单独记录，不能把原型或截图视为功能已实现。

## 后续执行与验收依据

已形成可执行的[全站实施计划](../../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)，执行顺序如下：

1. 采用 v1–v22 的已确认依据和总体审阅修订，实施 T1／T2 的公共主题、偏好清理、壳、组件及验证接线。
2. 按 T3–T8 的页面归属实施，保留既有业务、精度与权限；Trader Sync 专属布局暂缓，只回归共享影响。
3. Token 页面按独立计划，在共同 T1／T2 完成后接入；其后端接入可独立推进。
4. 完成 T9／T10 的真实页面、无过滤回归、文档与环境收尾后交付。

可据当前目标明确的验收方向：

- 会员与管理员的目标覆盖页面遵循统一深色外观；无主题选择入口，系统浅色偏好不改变页面外观。
- 登录前后、刷新、路由切换和弹出组件没有混入旧浅色或旧品牌主题；初始化外观一致。
- 主要操作、辅助信息、业务状态、焦点和禁用状态可清楚区分；图表和精确数值可正常核对。
- 既有权限、表单、列表、详情、复制、筛选和分页等流程按受影响范围验证，没有因换肤改变业务语义。
- 最终截图和实际页面与用户确认的视觉基准及一致性修订相符；尺寸、页面和场景清单按现有 coverage／plan 执行。
- 视觉审阅对照 Nansen 的已记录参考与用户确认样例，核对配色、文字、面板、空间和交互是否体现约定风格。

详细验收阈值、执行命令和页面矩阵已经写入技术方案与 T1–T10。当前只完成设计、规划和修订参考；旧主题测试或静态样板通过均不代表正式全站重构已完成。

[返回需求索引](../README.md)

## v16 管理员 Service Status（所展示视觉已确认）

[管理员 Service Status 视觉提案](service-status-proposal.md)沿用 v1–v15 已确认的单一深色主题、字体、布局、状态反馈与管理员应用壳，将当前纵向堆叠的 Services、Notifications、Trader Sync 改为始终保留各来源状态的三个页签。默认 Services 渲染返回的全部服务，本次合成 fixture 为 10 条；Notifications 先显示恢复与 System／Account 五类队列，再显示运行细节；Trader Sync 保留 raw 可观测性及返回的全部指标，本次 fixture 为 7 条，并逐项展示值、单位及 gauge／window／epoch 范围。手机将服务和指标表转换为有分隔线的逐条记录。

v16 已形成 11 张提案图、2 张当前 React 对照图和本地证据，独立审阅处置为 `ship`，没有必须修正项。该处置只覆盖视觉提案。用户另对四张展示图反馈“舒服 继续推进”，所展示的 Services 桌面／手机、Notifications 桌面和 Trader Sync 手机视觉已确认，具体范围见 [v16 确认记录](previews/theme-service-status-v16-approval.json)；其他辅助状态未逐图确认，原始审阅仍保留提交时的待确认状态。正式 `ui/`、真实三来源请求、10 秒可见 single-flight、缓存／授权边界及生产验收均未修改或验证。

| 项目 | 状态 | 确认范围或下一步 |
| --- | --- | --- |
| 管理员 Service Status | v16 所展示视觉已确认 | 审阅 [Services 桌面](previews/theme-service-status-v16-services-desktop.png)、[Notifications 桌面](previews/theme-service-status-v16-notification-desktop.png)、[Services 手机](previews/theme-service-status-v16-services-mobile.png)和 [Trader Sync 手机](previews/theme-service-status-v16-trader-mobile.png)；按 [v16 确认记录](previews/theme-service-status-v16-approval.json)所展示范围执行，其他辅助状态继续确认，批准时正式 UI 尚未改版 |

## v17 管理员 Etherscan Gateways（所展示视觉已确认）

[管理员 Etherscan Gateways 视觉提案](etherscan-gateways-proposal.md)沿用已确认的 Nansen 单一深色主题、Inter／JetBrains Mono、管理员应用壳和语义状态色，将网关运行状态与实际请求测试分为 `Gateways`／`Live Probe` 两个独立来源页签。网关地址、运行状态、延迟、检查时间和错误直接可读，Base URL 按需展开；Probe 保留两个既有参数、服务端整体结果、gateway／API key 汇总、七种失败分类、时序和错误样本。手机使用纵向记录，320px 进一步两列重排。

v17 已形成 10 张提案图、2 张当前 React 对照图和本地验证证据。四种尺寸下 202 项样板断言通过；当前 React 使用同一合成 GET fixture，1 项 Playwright 通过且没有 POST。独立审阅处置为 `ship`，只表示静态材料可交用户判断。用户另对四张主图反馈“舒服”，所展示 Gateways 与 Live Probe 的桌面／手机视觉已确认，具体范围见 [v17 确认记录](previews/theme-etherscan-gateways-v17-approval.json)；其他辅助状态未逐图确认，原始审阅保留提交时的待审阅状态。批准时正式 `ui/` 尚未改版，真实探针、权限、轮询、竞态、额度与 full-stack smoke 未由本轮验证。

| 项目 | 状态 | 确认范围或下一步 |
| --- | --- | --- |
| 管理员 Etherscan Gateways | v17 所展示视觉已确认 | 审阅 [Gateways 桌面](previews/theme-etherscan-gateways-v17-gateways-desktop.png)、[Live Probe 桌面](previews/theme-etherscan-gateways-v17-probe-desktop.png)、[Gateways 手机](previews/theme-etherscan-gateways-v17-gateways-mobile.png)和 [Live Probe 手机](previews/theme-etherscan-gateways-v17-probe-mobile.png)；按 [v17 确认记录](previews/theme-etherscan-gateways-v17-approval.json)所展示范围执行，其他辅助状态继续确认，批准时正式 UI 尚未改版 |


## 确认方式：按设计差异分组

用户在 2026-09-14 指出剩余页面较多，担心逐页确认造成过多轮次。本轮继续“系统通知列表与详情”，后续将页面覆盖与用户审阅分开管理：

- 已确认的主题、配色、字体、导航及通用组件直接沿用；同类页面使用同一套规则，不重复请求用户确认。
- 列表与详情、桌面与手机按业务流程合成一组审阅；常规加载、空态、错误、窄屏和字号放大由实现者沿既定规则完成并验证，不默认各占一次用户确认。
- 新的信息结构、图表表达、复杂表单或关键操作出现实质差异时，展示代表页面与必要差异；余下同类页面保留覆盖清单，在整体验收时核对一致性。
- 过去记录中的“未逐图确认”继续是历史事实，不补写成用户看过或批准过；这些条目也不自动成为必须单独新增确认轮次的关卡。
- 页面覆盖仍包括会员端、管理员端和共享界面。减少重复审阅不缩减实施、业务语义、权限、响应式或真实环境验收范围，不表示正式前端已完成。

本次系统通知列表与详情作为一个审阅组，重点判断记录筛选、消息正文、投递结果与时间信息的阅读层级。复用既有组件的辅助状态一并提供证据，不逐张追加问题。

## v18 管理员系统通知列表与详情（所展示视觉已确认）

[系统通知列表与详情视觉提案](system-notifications-proposal.md)沿用已确认的 Nansen 单一深色主题、Inter／JetBrains Mono、语义反馈色与管理员应用壳。列表将七个现有字段组织为五列，手机改为分隔记录；详情按消息、投递结果、五个独立时间和默认收起的标识排列，完整保留 17 个投影字段。Unknown 明确表示 Telegram 可能已经收到且不会自动重发；Test Notification 的 queued 只表示进入 pending，不表示送达。

v18 已形成 7 张提案图、4 张当前 React 对照图和本地证据。四种尺寸下 194 项样板断言通过；当前 React 只使用合成 GET，1 项 Playwright 通过且没有 POST。独立审阅处置为 `ship`，只表示静态材料可交用户判断。v18 所展示列表／详情与桌面／手机视觉已确认；批准时正式 `ui/` 尚未改版，真实权限、Telegram 投递、异步行为和 full-stack smoke 未在本轮验证；常规辅助状态沿用既有规则完成覆盖与验证，不增加逐图审批。

| 项目 | 状态 | 确认范围或下一步 |
| --- | --- | --- |
| 管理员系统通知列表与详情 | v18 所展示视觉已确认 | 已确认 [列表桌面](previews/theme-system-notifications-v18-list-desktop.png)、[详情桌面](previews/theme-system-notifications-v18-detail-desktop.png)、[列表手机](previews/theme-system-notifications-v18-list-mobile.png)和 [详情手机](previews/theme-system-notifications-v18-detail-mobile.png)；沿用已确认的筛选、正文、真实投递结果与时间层级，常规辅助状态由实现者验证，批准时正式 UI 尚未改版 |

用户已对四张主图反馈“舒服”，具体范围见 [v18 确认记录](previews/theme-system-notifications-v18-approval.json)。原始审阅与资产保持不变；本次认可不表示正式 UI 已实施。

用户于 2026-09-14 进一步明确：希望每次先完成多个页面，再集中提供审阅。后续按批次交付多个实际页面，桌面和手机一起展示；不是仅把一个页面的多个状态叫作一批。每批沿用已确认视觉规则，必要差异一次说明，收到反馈后针对指出部分修改。

## v19 四个管理员业务页面（所展示视觉已确认）

[v19 管理员业务页面集中视觉提案](admin-business-batch-proposal.md)一次覆盖 Trader Sync 订阅列表／概要和 Profit Sharing 轮次列表／详情四个实际页面，每页提供桌面与手机主图。它沿用已确认的 Nansen 单深色、Inter／JetBrains Mono、语义色和管理员壳；常规辅助状态作为实现证据覆盖，不变成逐图批准关卡。

Trader Sync 保留完整身份与钱包、生命周期／当前观察／历史中断边界、精确字符串计数和不透明 cursor，只提供管理员只读概要。Profit Sharing 保留 Draft → Collecting → Voting → Closed、sealed／匿名边界、exact-five Open 门槛和 0–5 人 Create／Draft roster。创建手机图的文档位于顶部，但弹窗正文滚动 206px 到 roster 控件，并保持底部操作可见。

本批 14 张提案图、8 张当前 React 对照图均有来源证据。独立 reviewer 经 1 个修正批后最终处置为 `ship`，只表示静态材料可交付。四页八张主图所展示视觉已确认；批准时正式 `ui/` 尚未改版，真实权限、后端读写、竞态和 full-stack smoke 未在本轮验证。

| 项目 | 状态 | 确认范围或下一步 |
| --- | --- | --- |
| 管理员 Trader Sync 与 Profit Sharing 四页 | v19 所展示视觉已确认 | 已确认两个列表、两个详情的[八张桌面／手机主图](admin-business-batch-proposal.md#原型截图与审阅材料)；辅助状态由实现者沿既有规则验证，不增加逐图审批，批准时正式 UI 尚未改版 |

用户于 2026-09-14 明确确认本批无须调整，范围见 [v19 确认记录](previews/theme-admin-batch-v19-approval.json)。原始原型、截图与审阅记录保持不变；辅助状态按既定规则验证，不追加逐图审批，正式前端后续按本批视觉基准实施。

## v20 四个会员基础业务页面（展示视觉已确认）

[v20 会员基础业务页面集中视觉提案](member-foundations-batch-proposal.md)覆盖 Wallets 私有钱包管理、Solana 发行候选列表、会员 Profit Sharing 轮次列表与详情四个实际页面，提供八张桌面／手机主图、十一张辅助图及八张当前 React 对照图。它沿用已确认的 Nansen 单一深色、Inter／JetBrains Mono、会员导航和分隔记录；Wallets 以名称及完整地址为主，Solana 分开表达候选、扫描状态与来源，Profit Sharing 按草稿／封存、匿名投票和结果组织个人任务。

用户于 2026-09-14 明确反馈“没问题，审批通过”，整批确认四页八张主图，范围见 [v20 确认记录](previews/theme-member-foundations-v20-approval.json)。原型、截图及 [v20 审阅记录](previews/theme-member-foundations-v20-review.json)保持原字节；审阅 JSON 的 `awaiting_user_review` 为交付时历史状态。辅助状态继续沿既定规则覆盖与验证。静态原型及合成 GET 对照不证明真实身份、密钥、采集、提交／投票、权限或全栈流程，批准时正式 `ui/` 尚未改版。

## 本轮其余页面安排

用户本轮明确排除 Trader Sync，已确认的共用规则和既有页面确认继续有效。[Nansen 钱包战绩 v1](../token/wallet-analytics-page-proposal.md)在独立目录中已经用户确认，直接沿用；它与 Wallets 私有钱包管理是不同页面，不重复设计或审批。

[其余页面视觉定稿安排](remaining-pages-plan.md)根据当前路由列出本轮 18 个实际业务页面、19 条路由，分为 4／8／6 三批。第一批 v20 四页、第二批 [v21 八个市场、赛事及 Managed OO 页面](market-intelligence-batch-proposal.md)、第三批 [v22 六个 Worm Trading 页面及六项共用适配](worm-and-common-batch-proposal.md)所展示主视觉均已整批确认。常规辅助状态继续沿既定规则覆盖和验证，不逐图增加确认；其实现及验收归属已写入[完整覆盖清单](theme-refactor-coverage.md)与[实施计划](../../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)。

## v21 八个市场、赛事与 Managed OO 页面（展示视觉已确认）

[v21 集中视觉提案](market-intelligence-batch-proposal.md)覆盖 Market Radar 热门／实时／涨跌榜、Sports Live／History、World Cup Corners 与 Managed OO 提案／争议八个实际页面，沿用 v20 已确认主题及共用规则。八页提供 16 张桌面／手机主图、29 张辅助图和 16 张当前 React 合成对照，共 61 张，辅助状态不逐图新增审批。

独立首次审阅覆盖全部 61 图，唯一 Sports 来源重复箭头问题经一个修正批关闭；最终审阅只复核 14 张修正图及相关回归，结合初审得出 `ship`。用户随后于 2026-09-14 对八页十六张主图反馈“确认”，整批确认所展示视觉，范围见 [v21 确认记录](previews/theme-market-intelligence-v21-approval.json)。原型及[批次记录](previews/theme-market-intelligence-v21-review.json)保留原字节，其中 `awaiting_user_review` 表示交付时历史状态；本次批准由独立确认记录承接。辅助状态沿既定规则覆盖与验证，不补写逐图审批。局部原型与截获合成数据不证明真实供应商、扫描／同步、鉴权／权限、竞态或全栈流程；批准时正式 `ui/` 未改版。第三批 [v22 六页和共用适配](worm-and-common-batch-proposal.md)随后也已获整批确认，当前已完成[覆盖归属核对](theme-refactor-coverage.md)并整理正式技术方案与实施计划，批准时代码与真实验收尚未开始。

## v22 Worm Trading 与共用页面（展示视觉已确认）

[v22 集中视觉提案](worm-and-common-batch-proposal.md)覆盖 Worm 资产、组合列表／编辑器、执行预览、执行记录／详情六个实际业务页面（七条路由），以及管理员登录、共享注册的管理员身份、管理员 Profile／Access、会员／管理员 Help 六项界面差异。反馈矩阵只覆盖两身份域现有状态，不新增业务路由。沿用既定 Nansen 颜色关系、Inter／JetBrains Mono、应用壳及主次操作层级。

四份 HTML 提供二十四张桌面／手机主图、五十七张辅助图及四十张当前 React 合成对照，共 121 张 PNG。初次独立审阅打开全部 120 张输入图及两张 v21 参照，指出组合失效选择清理后保存仍禁用、执行步骤时间矛盾、六页 200% 头像溢出三项问题；修正后增补一张恢复证据。最终复审仅重读 47 张修正包图及相关回归，三项均 resolved，结合初审处置为 `ship`。范围、摘要和原文见[批次记录](previews/theme-worm-and-common-v22-review.json)，其 `awaiting_user_review` 保留交付时历史状态。

用户于 2026-09-14 对二十四张主图反馈“确认✅”，整批确认布局、信息密度、阅读顺序、操作层级与既定 Nansen 主题搭配，范围见 [v22 确认记录](previews/theme-worm-and-common-v22-approval.json)。138 份交付原型、截图、数据、检查及审阅记录保持原字节；辅助图不补写成用户逐图看过，也不追加常规状态审批。

每页已覆盖桌面、手机、320px 与 200% 字号，十一张原生弹窗保留真实视口。未展示的 Help 动态配置分支仍沿既定规则覆盖与验证，不新增逐状态审批。真实后台、身份、权限、资源、凭据及交易均未验证，批准时正式 `ui/` 未改版；本任务未启停服务。总覆盖和实施准备见[覆盖清单](theme-refactor-coverage.md)及[技术方案](../../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)；Trader Sync 专属布局暂缓，独立 Token／Nansen 钱包设计保持不变。
