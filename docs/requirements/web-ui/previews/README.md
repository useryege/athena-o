# 基础主题视觉样板

> 状态：v1 配色与按钮层级、v2 字体与数字排版、v3 导航／页头／页面密度、v4 第一组通用组件与状态、v5 列表控件的展示视觉、v6 Trader Sync 首页布局与辅助信息默认折叠，以及 v7 添加交易员页、v8 订阅列表和 v9 订阅详情所展示的桌面／手机视觉，以及 v9 历史默认摘要方式均已确认。v10 Notifications 所展示的已连接桌面／手机及桌面重新连接视觉已确认；v11 Account Center / Profile 所展示的桌面资料页、手机编辑状态与手机离开确认视觉已确认。v12 Account Center / Security 所展示的桌面／手机安全设置页及手机连接说明弹窗视觉已确认。v13 Account Center / Access & session 所展示的桌面／手机权限与会话页及手机 Google 待授权页视觉已确认。v14 会员登录与注册所展示的登录页与 Google 注册页桌面／手机视觉已确认。v15 管理员导航与账户管理所展示的桌面账户管理、手机目录、手机权限详情和手机确认弹窗视觉已确认。正式 UI 尚未改版。

## v21 市场、赛事与 Managed OO 八页（展示视觉已确认）

[集中提案与八页主图](../market-intelligence-batch-proposal.md#八页主图与路由)覆盖 Market Radar 三页、Sports 三页和 Managed OO 两页，一次提供 16 张桌面／手机主图。[全部辅助索引](../market-intelligence-batch-proposal.md#全部截图与辅助索引)覆盖 29 张辅助和 16 张当前 React 对照；合计 61 张，四个手机弹窗为实际 390×844 视口，其余从顶部截取全页。普通辅助状态沿用既定规则，不逐图新增审批。

[Market HTML](theme-market-radar-v21.html)、[Sports HTML](theme-sports-v21.html)、[Managed OO HTML](theme-managed-oo-v21.html)均为明确合成的静态原型，数据／检查／采集链接见[证据表](../market-intelligence-batch-proposal.md#检查独立审阅与证据)。Market 98 断言、Sports 12 视口与 24 交互记录、Managed OO 9 行为项及 208 控件样本通过；来源扫描 61 张、0 缺失。首次独立审阅覆盖 61 图，最终 14 图定向复核关闭唯一 Sports 箭头问题，未重做全量审阅。[批次 review](theme-market-intelligence-v21-review.json)保留交付时的 `ship` 和 `awaiting_user_review` 历史记录。用户于 2026-09-14 对八页十六张主图反馈“确认”，整批确认范围见 [v21 确认记录](theme-market-intelligence-v21-approval.json)；交付原型、截图、数据、检查和审阅记录共 74 份资产原字节保留，辅助图不补写成用户逐图看过。

v20 已确认主题与审批资产保持不变，正式 `ui/` 未修改。当前 React 使用本地合成 GET 及 Sports 两条只读 batchGet POST 拦截，未真实写入；本轮未启停服务，临时浏览器已关闭，借用的既有根 Vite 保留。真实供应商、权限、同步／扫描与 full-stack 验收不在本轮结论内；第三批和共用适配的后续确认见 [v22 确认记录](theme-worm-and-common-v22-approval.json)。

## v15 管理员导航与账户管理（所展示视觉已确认）

[v15 视觉基准](../admin-accounts-proposal.md)沿用已确认的近黑／青绿色系统、Inter 与 JetBrains Mono，以桌面 224px 侧栏、304px 账户目录和默认 Access 的三页签详情组织既有 `/admin/accounts`；手机按目录到详情分步展示，导航抽屉为 280px 且不超过视口减 48px。权限聚合仍为 11 项、展示 10 项并保留隐藏 Token，未新增创建、删除或管理员提升能力。

| 类别 | 证据 |
| --- | --- |
| 四张主要提案图 | [桌面](theme-admin-accounts-v15-desktop.png)、[手机目录](theme-admin-accounts-v15-mobile-list.png)、[手机 Access](theme-admin-accounts-v15-mobile-access.png)、[手机确认](theme-admin-accounts-v15-confirm-mobile.png) |
| 七张辅助图 | [桌面 Profile](theme-admin-accounts-v15-profile-desktop.png)、[手机 Profile](theme-admin-accounts-v15-profile-mobile.png)、[手机 Identity](theme-admin-accounts-v15-identity-mobile.png)、[手机导航](theme-admin-accounts-v15-navigation-mobile.png)、[手机空态](theme-admin-accounts-v15-empty-mobile.png)、[320px](theme-admin-accounts-v15-narrow.png)、[200% 文字](theme-admin-accounts-v15-zoom.png) |
| 三张当前 React 对照 | [桌面](theme-admin-accounts-v15-before-desktop.png)、[手机目录](theme-admin-accounts-v15-before-mobile-list.png)、[手机详情](theme-admin-accounts-v15-before-mobile-detail.png) |
| 可复核文件 | [HTML](theme-admin-accounts-v15.html)、[数据](theme-admin-accounts-v15-data.json)、[检查](theme-admin-accounts-v15-checks.json)、[React 采集](theme-admin-accounts-v15-before-capture.json)、[审阅](theme-admin-accounts-v15-review.json) |

14 张 PNG 均带来源，审阅副本逐字节一致且扫描 0 缺失；四个视口共 119 项检查通过，字体已加载，无横向溢出、页面错误或外部请求，抽样正文对比度不低于 4.5。当前 React 仅运行 1 项 GET fixture 检查（2.9s），没有账户写入。独立初审提出的抽屉宽度、手机确认操作和导航图标三项问题已由一个批次修正；最终 `ship` 只复核这些修正，不代表用户批准、生产实现或完整 smoke。用户对所展示桌面账户管理、手机目录、手机权限详情及手机确认四张图反馈“可以”，范围见 [v15 确认记录](theme-admin-accounts-v15-approval.json)；其他辅助状态未逐图确认。原始 HTML、截图、检查和审阅记录保持不变，当前确认以独立确认记录为准。正式 `ui/` 未修改。

## v14 会员登录与注册（所展示视觉已确认）

[会员登录与注册视觉基准](../auth-proposal.md)沿用 v1–v13 已确认的 Nansen 单一深色世界、Inter 与地址专用 JetBrains Mono，以居中 448px 单列面板承载当前任务。Google 和 Phantom 保持同级且互为独立身份；注册按已验证身份、用户名及可用状态、永久名称说明、创建动作和身份切换排列，业务访问仍由管理员另行授予。

| 类别 | 截图 |
| --- | --- |
| 四张主提案图 | 登录[桌面](theme-auth-v14-login-desktop.png)、[手机](theme-auth-v14-login-mobile.png)；注册[桌面](theme-auth-v14-register-desktop.png)、[手机](theme-auth-v14-register-mobile.png) |
| 七张辅助图 | 登录 [320px](theme-auth-v14-login-narrow.png)、[200% 文字](theme-auth-v14-login-text-200.png)、[手机 Phantom 缺失](theme-auth-v14-login-mobile-missing.png)；注册 [320px](theme-auth-v14-register-narrow.png)、[200% 文字](theme-auth-v14-register-text-200.png)、[手机 Phantom 身份](theme-auth-v14-register-mobile-wallet.png)、[手机 ticket 过期](theme-auth-v14-register-mobile-expired.png) |
| 四张当前 React 对照 | 登录[桌面](theme-auth-v14-before-login-desktop.png)、[手机](theme-auth-v14-before-login-mobile.png)；注册[桌面](theme-auth-v14-before-register-desktop.png)、[手机](theme-auth-v14-before-register-mobile.png) |

[可操作 HTML](theme-auth-v14.html)、[固定虚构数据](theme-auth-v14-data.json)、[原型检查](theme-auth-v14-checks.json)、[当前 React 采集](theme-auth-v14-before-capture.json)与[独立审阅记录](theme-auth-v14-review.json)共同记录范围和证据。11 张提案图与 4 张 React 对照图均为带来源的 Playwright Chromium 截图；审阅副本逐字节一致，来源扫描为 15 张、0 缺失。原型在 1440×900、390×844、320×844 和 720×900／根字号 200% 下没有横向溢出、外部请求或页面错误，抽样文字对比度不低于 4.5。当前 React 的 1 个 GET-only Playwright 测试通过（4.5s）并生成 4 张对照图；现有注册页普通 `ConfigProvider` 的低对比度是基线事实，`ui/` 未修改。

两轮自身检查只修正禁用态 hover／pressed 与指针中性截图。独立审阅最初的窄屏顶部裁切观察经现有 PNG 与七状态 320px 边界检查证伪并撤回，最终处置为 `ship`，没有 reviewer code fix。原型的 400ms 本地用户名检查和 sign-in／create／switch／recovery 控件只切换本地展示或显示预览反馈，不发出真实 provider、注册或取消请求，也不导航到外部授权；完整服务端用户名安全规则、ticket／CSRF／`returnTo`、移动钱包深链边界和真实流程仍是正式实现义务。用户另对所展示登录页与 Google 注册页的桌面／手机四张图反馈“没问题”，范围见 [v14 确认记录](theme-auth-v14-approval.json)；其他辅助状态未逐图确认。原始 HTML、截图、检查和审阅记录保持不变；`ship` 本身不代表用户批准、生产实现或 smoke 验收。本任务没有启动服务，复用了任务前已有的根 Vite 4000，关闭全部任务浏览器上下文后保留该环境，准确归属见采集记录。

## v13 Account Center / Access & session（所展示视觉已确认）

[Account Center / Access & session 视觉提案](../account-access-proposal.md)沿用 v1–v12 基准，把十项模块权限和 2／3／5 摘要放在 Current session 前，独立显示 API Key 与 Profit Sharing，并将时间、修订和版本默认收起。Pending 页面区分 Google／Phantom 已验证身份与尚未开通的业务访问；样板不更改真实授权。

- 当前 React 对比：[桌面 active](theme-account-access-v13-before-desktop.png)、[手机 active](theme-account-access-v13-before-mobile.png)、[手机 pending](theme-account-access-v13-before-mobile-pending.png)。
- 新主题 active：[桌面](theme-account-access-v13-desktop.png)、[手机](theme-account-access-v13-mobile.png)、[桌面展开细节](theme-account-access-v13-desktop-expanded.png)。
- Pending：[桌面 Google](theme-account-access-v13-desktop-pending.png)、[手机 Google](theme-account-access-v13-mobile-pending.png)、[手机 Phantom](theme-account-access-v13-mobile-phantom.png)、[手机刷新错误](theme-account-access-v13-mobile-error.png)。
- 适配：[320px](theme-account-access-v13-narrow.png)、[200% 文字](theme-account-access-v13-text-200.png)。
- [可操作 HTML](theme-account-access-v13.html)、[固定虚构数据](theme-account-access-v13-data.json)、[当前 React 采集](theme-account-access-v13-before-capture.json)、[原型检查](theme-account-access-v13-checks.json)、[独立审阅记录](theme-account-access-v13-review.json)。

9 张最终原型图和 3 张当前 React 图均为带来源记录的 Chromium 截图，没有 AI 生成或修图；12 张审阅副本逐字节一致，来源扫描 0 缺失。1440×900、390×844、320×844 与 720×900／根字号 200% 检查通过。当前 React 使用同一合成账户与 grants，只截获 GET，1 个测试通过（3.0s）并生成 3 张截图。两轮检查后合并重复的 Logged in 展示，并在 800px 及以下堆叠账户标志；Inter detector 例外仅限本 HTML。

原型支持本地状态选择、细节展开、复制、650ms 刷新忙碌态、退出提示、错误与恢复，不执行真实轮询、权限检查、授权变更或退出。独立审阅对全部 12 张图和源码证据给出 `ship`，未提出需要修正的视觉问题；这不是用户批准、生产实现或验收。用户另对桌面／手机默认页及手机 Google 待授权页反馈“舒服”，范围见 [v13 确认记录](theme-account-access-v13-approval.json)；其余辅助状态未逐图确认，原始 HTML、截图、检查与审阅记录保持不变，正式 `ui/` 未修改。本任务未启动服务；对照复用了任务前已有的根 Vite 4000，浏览器上下文均已关闭，已有服务保持原样。

## v12 Account Center / Security（所展示视觉已确认）

[Account Center / Security 视觉提案](../account-security-proposal.md)沿用 v1–v11 基准，以紧凑当前账户身份衔接相邻 Profile 页面，先呈现 Connect AI，再呈现 API keys。它明确区分一次性完整 AI 接入说明与密钥元数据管理；列表只显示 ID、Issued、Expires、Revoke，不增加 scope、provider、最近使用、状态或调用量。Credential ready 只表示 ATHENA 接受预期账户的凭据，不表示外部 AI 已连接。

- 当前 React 对比：[桌面](theme-account-security-v12-before-desktop.png)、[手机](theme-account-security-v12-before-mobile.png)。
- 新主题默认页：[桌面](theme-account-security-v12-desktop.png)、[手机](theme-account-security-v12-mobile.png)；创建：[桌面 Create API key](theme-account-security-v12-desktop-create-key.png)、[手机 Connect AI](theme-account-security-v12-mobile-create-ai.png)。
- AI 结果：[桌面 ready](theme-account-security-v12-desktop-ai-ready.png)、[桌面 failed](theme-account-security-v12-desktop-ai-failed.png)、[手机 ready](theme-account-security-v12-mobile-ai-ready.png)；[手机 API Key 一次性结果](theme-account-security-v12-mobile-key-result.png)、[手机撤销确认](theme-account-security-v12-mobile-revoke.png)。
- 适配：[320px](theme-account-security-v12-narrow.png)、[200% 文字](theme-account-security-v12-text-200.png)。
- [可操作 HTML](theme-account-security-v12.html)、[固定虚构数据](theme-account-security-v12-data.json)、[当前 React 采集](theme-account-security-v12-before-capture.json)、[原型检查](theme-account-security-v12-checks.json)、[独立审阅记录](theme-account-security-v12-review.json)。

11 张最终原型图和 2 张当前 React 图均为带来源记录的 Chromium 截图，没有 AI 生成或修图，来源扫描为 13 张、0 缺失。四个视口检查通过，无横向溢出、页面错误、外部请求或记录问题；长 AI 说明在手机弹窗的滚动正文中展示，固定标题和复制／Done 操作保持可见。当前 React 对比使用同一账户和三条 token metadata，只观察到 bootstrap 与 token-list GET，1 个测试在 2.1 秒内生成 2 张截图，没有真实写入。

原型使用无效合成 secret 与 `example.invalid`，所有签发、验证、撤销和导航只做本地演示；样例复制会复制完整合成载荷。独立 finish review 覆盖全部 13 张图与本地原型并给出 `ship`，未提出需要修正的视觉问题。该处置只覆盖视觉提案。用户另对桌面／手机默认页和手机 AI 连接说明弹窗反馈“舒服，继续”，范围见 [v12 确认记录](theme-account-security-v12-approval.json)；其余辅助状态未逐图确认，原始 HTML、截图、检查与审阅记录保持不变；真实创建／撤销／验证、异步防陈旧、权限与账户变化仍是正式实现验收范围。Inter detector 例外仅限本 HTML。对照采集复用了任务前已有的根 Vite 4000，本任务未启动服务；浏览器均已关闭，已有服务保持原样。

## v11 Account Center / Profile（所展示视觉已确认）

[Account Center / Profile 视觉提案](../account-profile-proposal.md)沿用 v1–v10 基准，把重复的已保存身份、首字母头像和编辑区域合并为一个 Profile 面板，明确区分只读 Username 与可编辑 Display name，并补齐未保存草稿、名称冲突、保存错误、上传错误及安全离开。目标单一深色界面移除 Appearance；Security 只用于 `apiKeyEnabled` 的会员 fixture，管理员视觉不在本轮范围。

- 当前 React 对比：[桌面](theme-account-profile-v11-before-desktop.png)、[手机](theme-account-profile-v11-before-mobile.png)。
- 新主题默认与编辑：[桌面](theme-account-profile-v11-desktop.png)、[桌面编辑](theme-account-profile-v11-desktop-edited.png)、[手机](theme-account-profile-v11-mobile.png)、[手机编辑](theme-account-profile-v11-mobile-edited.png)。
- 名称状态：[桌面冲突](theme-account-profile-v11-desktop-conflict.png)、[桌面保存错误](theme-account-profile-v11-desktop-error.png)、[手机无效输入](theme-account-profile-v11-mobile-invalid.png)；头像状态：[手机上传错误](theme-account-profile-v11-mobile-upload-error.png)。
- 离开确认：[桌面](theme-account-profile-v11-desktop-leave.png)、[手机](theme-account-profile-v11-mobile-leave.png)；适配：[320px](theme-account-profile-v11-narrow.png)、[200% 文字](theme-account-profile-v11-text-200.png)。
- [可操作 HTML](theme-account-profile-v11.html)、[固定虚构数据](theme-account-profile-v11-data.json)、[当前 React 采集](theme-account-profile-v11-before-capture.json)、[原型检查](theme-account-profile-v11-checks.json)、[独立审阅记录](theme-account-profile-v11-review.json)。

12 张最终原型图和 2 张当前 React 图均为带来源记录的 Chromium 截图，没有 AI 生成或修图。四个视口检查通过，无横向溢出、重复 ID、页面错误、外部请求或记录问题；原型本地覆盖草稿、校验、Reset、状态切换和离开确认，Save／Upload／导航不发起真实业务写入。当前 React 对比使用同一 fixture，只观察到 bootstrap GET，1 个测试生成 2 张截图；这不是完整 smoke 或后端验收。

独立 finish review 检查全部 14 张图并给出 `ship`，未提出需要修正的视觉问题；该处置只覆盖视觉原型与本地演示，不代表用户批准或生产验收。用户对桌面资料页、手机编辑状态及手机离开确认反馈“舒服”，具体范围见 [v11 确认记录](theme-account-profile-v11-approval.json)；其余辅助状态未逐图确认，原始 HTML、截图、检查与审阅记录保持不变。已有头像的 Remove、真实 Profile／Avatar 请求与 CAS、身份切换、授权及管理员页面留待正式实现与独立验收。v11 专属 Vite 5621 已停止，原有共享服务保持原样；Inter detector 例外仅限本 HTML。

## v10 现有 Notifications 页面（所展示视觉已确认）

[Notifications 视觉提案](../notifications-proposal.md)沿用 v1–v9 基准，将当前 Telegram 连接与未完成的新设置分层：现有绑定保持至替换成功，取消设置保持原绑定；三步设置同时提供同一链接、手动命令和二维码。Expired／Failed 恢复区经审阅合并为单一主要 Create new link，Failed 使用错误色，到期与需要注意使用琥珀色。此轮没有新增收件箱或其他通知渠道。

- 当前 React 对比：[桌面 Connected](theme-notifications-v10-before-desktop.png)、[手机 Connected](theme-notifications-v10-before-mobile.png)、[桌面 Pending](theme-notifications-v10-before-desktop-pending.png)、[手机 Pending](theme-notifications-v10-before-mobile-pending.png)。
- 新主题主要状态：[桌面 Connected](theme-notifications-v10-desktop.png)、[手机 Connected](theme-notifications-v10-mobile.png)、[桌面 Pending](theme-notifications-v10-desktop-pending.png)、[手机 Pending](theme-notifications-v10-mobile-pending.png)。
- 弹窗：[桌面 Disconnect](theme-notifications-v10-desktop-disconnect.png)、[手机 Disconnect](theme-notifications-v10-mobile-disconnect.png)；桌面辅助状态：[Reconnect](theme-notifications-v10-desktop-reconnect.png)、[其他标签页](theme-notifications-v10-desktop-remote.png)、[Expired](theme-notifications-v10-desktop-expired.png)、[刷新失败保留旧值](theme-notifications-v10-desktop-stale.png)。
- 手机辅助状态：[Unreachable](theme-notifications-v10-mobile-unreachable.png)、[首次错误](theme-notifications-v10-mobile-error.png)、[Failed](theme-notifications-v10-mobile-failed.png)；适配：[320px](theme-notifications-v10-narrow.png)、[200% 文字](theme-notifications-v10-text-200.png)。
- [可操作 HTML](theme-notifications-v10.html)、[固定虚构数据](theme-notifications-v10-data.json)、[当前 React 采集](theme-notifications-v10-before-capture.json)、[原型检查](theme-notifications-v10-checks.json)、[独立审阅记录](theme-notifications-v10-review.json)。

15 张最终截图覆盖上述状态。原型在 1440×900、390×844、320×844、720×900／根字号 200% 下通过横向溢出、页面错误、外部请求、重复 ID、弹窗焦点循环／Escape／焦点恢复、复制载荷、字体和对比度检查。固定 fixture 使用 `example.invalid` 演示二维码；连接、重连、取消设置、刷新、打开 Telegram 和确认解绑等业务动作只显示本地 notice，不执行 Telegram 或 API 写入；复制、导航折叠、样例状态切换和弹窗开关仍可在本地操作。Inter 是 v2 已确认字体，detector 的 `overused-font=inter` 仅对 v10 HTML 通过 CLI 持久忽略；审阅的 `ship` 仅表示两项修正 resolved 且未观察到批次回归，不代表整页或用户批准。

四张当前 React 对比图由 1 个 fixture 测试生成，仅观察 GET，不是完整 smoke、后端验收或真实 Telegram 行为验证。真实请求、竞态保护、3 秒可见轮询、倒计时、`sessionStorage` 标签页边界、账户切换、Bot 可用性、条件式返回 Trader Sync 和 owner 草稿均是未来正式实现需保留并验证的契约。任务 Vite 5620 已停止，原有用户／共享环境保持原样，详见采集记录。用户对直接展示的已连接桌面／手机及桌面重新连接完整截图反馈“可以，继续设计”，范围见 [v10 确认记录](theme-notifications-v10-approval.json)。其余辅助状态未逐图确认；原始 HTML、截图、数据、检查与审阅记录保持不变，正式 `ui/` 未修改。

## v9 现有“订阅详情”页面（展示视觉及默认摘要已确认）

[订阅详情视觉基准](../trader-sync-subscription-detail-proposal.md)延续 v1–v8 视觉基准。桌面将资料、当前监控和历史放在左侧，备注／管理及旧通知集中右侧；手机按阅读顺序纵向排列。历史记录默认保留摘要与可能遗漏提示，按需展开完整时间及边界；用户已对桌面／手机完整页及手机取消弹窗反馈“舒服，继续设计”，确认默认摘要方式、布局与操作层级。

- 当前 React：[桌面](theme-trader-detail-v9-before-desktop.png)、[手机](theme-trader-detail-v9-before-mobile.png)。
- 新主题：[桌面完整页](theme-trader-detail-v9-desktop.png)、[手机完整页](theme-trader-detail-v9-mobile.png)、[桌面首屏](theme-trader-detail-v9-desktop-first-screen.png)、[手机首屏](theme-trader-detail-v9-mobile-first-screen.png)。
- 取消确认：[桌面](theme-trader-detail-v9-desktop-cancel.png)、[手机](theme-trader-detail-v9-mobile-cancel.png)；[历史展开](theme-trader-detail-v9-history-expanded.png)。
- 辅助状态：[暂停](theme-trader-detail-v9-mobile-paused.png)、[已取消](theme-trader-detail-v9-cancelled.png)、[操作结果未知](theme-trader-detail-v9-mobile-unknown.png)、[历史读取失败](theme-trader-detail-v9-history-error.png)、[首次加载](theme-trader-detail-v9-loading.png)、[首次失败](theme-trader-detail-v9-mobile-error.png)、[更新失败保留旧值](theme-trader-detail-v9-stale.png)。
- [可操作 HTML](theme-trader-detail-v9.html)、[共享演示数据](theme-trader-detail-v9-data.json)、[当前 React 采集](theme-trader-detail-v9-before-capture.json)、[原型检查](theme-trader-detail-v9-checks.json)、[独立审阅记录](theme-trader-detail-v9-review.json)。

所有资料和历史均为虚构；保存、暂停、恢复、取消和导航只提供本地演示反馈。当前 React 对比范围为正常详情页；新版取消弹窗不附旧弹窗对照。正式 `ui/` 尚未修改。确认范围见 [v9 记录](theme-trader-detail-v9-approval.json)；桌面取消弹窗、展开细节及其余辅助状态未逐图确认，原始 HTML、图片、数据、检查与审阅记录保持不变。

## v8 现有“订阅管理列表”页面（展示视觉已确认）

[订阅列表视觉基准](../trader-sync-subscriptions-proposal.md)延续 v1–v7 基准。用户查看新主题桌面／手机 Current 完整截图后反馈“舒服”，确认信息密度、状态区分和手机排版：桌面按交易员、监控、旧通知、管理入口四列对齐，手机按交易员依次分组。Current / Cancelled、配额和资料更新时间集中在列表上方；完整钱包与时间、六项通知计数和已排队通知可能继续到达的说明保持可见。

- 当前 React 页面：[桌面](theme-trader-subscriptions-v8-before-desktop.png)、[手机](theme-trader-subscriptions-v8-before-mobile.png)、[已取消](theme-trader-subscriptions-v8-before-cancelled.png)。
- 新主题：[桌面完整页](theme-trader-subscriptions-v8-desktop.png)、[手机完整页](theme-trader-subscriptions-v8-mobile.png)、[桌面首屏](theme-trader-subscriptions-v8-desktop-first-screen.png)、[手机首屏](theme-trader-subscriptions-v8-mobile-first-screen.png)。
- 已取消视图：[桌面](theme-trader-subscriptions-v8-desktop-cancelled.png)、[手机](theme-trader-subscriptions-v8-mobile-cancelled.png)。
- 辅助状态：[更新失败保留旧页](theme-trader-subscriptions-v8-stale.png)、[手机空列表](theme-trader-subscriptions-v8-mobile-empty.png)、[首次加载](theme-trader-subscriptions-v8-loading.png)、[手机首次失败](theme-trader-subscriptions-v8-mobile-error.png)。
- [可交互 HTML](theme-trader-subscriptions-v8.html)、[共享演示数据](theme-trader-subscriptions-v8-data.json)、[当前界面采集证据](theme-trader-subscriptions-v8-before-capture.json)、[原型检查](theme-trader-subscriptions-v8-checks.json)、[独立审阅记录](theme-trader-subscriptions-v8-review.json)。

前后共享 3 个当前订阅、1 个取消记录及 3 / 10 当前配额，全部为明确标注的虚构资料。旧通知数量拆为有标签的原始计数，仍进入详情页管理。原型只有本地视图、复制和状态演示，不发起真实订阅请求；正式 `ui/` 未修改。

直接展示并获认可的是新主题桌面和手机 Current 完整截图，当前版桌面／手机以链接提供对照。Cancelled 列表、空态、加载和失败等辅助截图未逐图确认。具体范围与原件摘要见 [v8 确认记录](theme-trader-subscriptions-v8-approval.json)；HTML、截图、数据、检查与原始审阅记录保持提交时状态。

## v7 现有“添加交易员”页面（展示视觉已确认）

[添加页视觉基准](../trader-sync-add-proposal.md)延续 v1–v6 基准，将当前输入、资料核对、盈亏和订阅确认流程放入新主题。用户查看新主题桌面／手机完整截图后反馈“舒服”，确认布局、信息密度和操作层级：桌面左侧核对资料与 P/L、右侧集中确认；手机按输入、资料、P/L、确认纵向排列。

- 当前 React 页面：[桌面](theme-trader-add-v7-before-desktop.png)、[手机](theme-trader-add-v7-before-mobile.png)、[初始输入](theme-trader-add-v7-before-input.png)。
- 新主题：[桌面完整页](theme-trader-add-v7-desktop.png)、[手机完整页](theme-trader-add-v7-mobile.png)、[桌面首屏](theme-trader-add-v7-desktop-first-screen.png)、[手机首屏](theme-trader-add-v7-mobile-first-screen.png)。
- 输入态：[桌面](theme-trader-add-v7-desktop-input.png)、[手机](theme-trader-add-v7-mobile-input.png)；过期态：[桌面确认区](theme-trader-add-v7-desktop-expired.png)、[手机确认区](theme-trader-add-v7-mobile-expired.png)。
- 盈亏：[精确曲线值](theme-trader-add-v7-curve-values.png)、[缺失状态](theme-trader-add-v7-unavailable.png)。
- [可交互 HTML](theme-trader-add-v7.html)、[共享演示数据](theme-trader-add-v7-data.json)、[当前界面采集证据](theme-trader-add-v7-before-capture.json)、[原型检查](theme-trader-add-v7-checks.json)、[独立审阅记录](theme-trader-add-v7-review.json)。

前后使用同一批受控资料，1Y 八点曲线明确为虚构；超大整数、零值及其余周期缺失用于核对展示边界。`$` 保留供应商符号说明，不推定币种。HTML 中的状态按钮只切换演示，确认／恢复／导航只显示本地提示，不创建真实订阅；本轮未修改正式 `ui/`。

直接展示并获认可的是新主题桌面和手机完整截图，当前版桌面／手机以链接提供对照。已展示的默认 1Y P/L、周期控件、备注、有效期、配额和 Telegram 分层随页面视觉确认；输入、过期、缺失及精确曲线值等辅助截图未逐图确认。范围与原件摘要见 [v7 确认记录](theme-trader-add-v7-approval.json)。HTML、截图、演示数据、检查和原始审阅记录保留提交时状态。

## v6 现有 Trader Sync 首页改版（页面视觉与默认折叠已确认）

[首页视觉基准](../trader-sync-home-proposal.md)将已确认的视觉系统应用到当前业务页面，用户查看前后对照后反馈“认可”。当前 React 界面与新主题原型使用同一批受控演示数据；两侧都不是线上账户或真实交易截图。

- 当前界面：[桌面](theme-trader-sync-v6-before-desktop.png)、[手机](theme-trader-sync-v6-before-mobile.png)。
- 改版提案：[桌面完整页面](theme-trader-sync-v6-desktop.png)、[手机完整页面](theme-trader-sync-v6-mobile.png)、[桌面首屏](theme-trader-sync-v6-desktop-first-screen.png)、[手机首屏](theme-trader-sync-v6-mobile-first-screen.png)。
- 展开状态：[手机目标区](theme-trader-sync-v6-mobile-targets.png)、[活动证据](theme-trader-sync-v6-evidence.png)。
- [可交互 HTML](theme-trader-sync-v6.html)、[共享演示数据](theme-trader-sync-v6-data.json)、[当前界面采集证据](theme-trader-sync-v6-before-capture.json)、[原型浏览器检查](theme-trader-sync-v6-checks.json)、[独立审阅记录](theme-trader-sync-v6-review.json)。

本轮已确认整页信息层级、桌面活动／目标布局、手机阅读体验，以及将身份快照、Position ID 与来源入口默认放入展开区的处理。日期仍按 UTC+8，分页仍为 50／100 条和 Previous／Next。直接展示范围为当前桌面、新主题桌面和手机完整截图；其他截图作为辅助证据保存，未将其全部细节当作逐图批准。范围与原件摘要见 [v6 确认记录](theme-trader-sync-v6-approval.json)。HTML、截图、检查及原始审阅记录保留提交时状态；正式 `ui/` 尚未修改，原型只提供本地交互。

## v5 下拉菜单、表格筛选与分页（展示视觉已确认）

[列表控件视觉基准](../list-controls-proposal.md)已获用户认可，反馈为“可以”。直接展示范围是桌面菜单、手机菜单与手机筛选结果；当前范围见 [v5 确认记录](theme-list-controls-v5-approval.json)。沿用 v1–v4 基准，展示完整合约地址筛选、已实现盈亏排序下拉、可清除的条件标签和“加载更多”。

- [可交互 HTML](theme-list-controls-v5.html)、[桌面总览](theme-list-controls-v5-desktop.png)、[手机总览](theme-list-controls-v5-mobile.png)。
- [桌面菜单](theme-list-controls-v5-desktop-menu.png)、[手机菜单](theme-list-controls-v5-mobile-menu.png)。
- [桌面筛选结果](theme-list-controls-v5-desktop-filtered.png)、[手机筛选结果](theme-list-controls-v5-mobile-filtered.png)。
- [浏览器检查](theme-list-controls-v5-checks.json)、[审阅记录](theme-list-controls-v5-review.json)。

全部为本地虚构数据；样例 5 行加 3 行用于观察控件，正式每批最多 20 个的规则不变。没有外部请求或正式 UI 修改。原始 HTML、截图和审阅记录保留提交时状态；这些共用规则用于现有全站前端改版，后续页面级设计对照现有代表页面。

## v4 第一组通用组件与状态（展示视觉已确认）

[组件与状态视觉基准](../components-proposal.md)已获用户确认，2026-09-14 反馈为“舒服，可以。”。本轮确认桌面总览所展示的表单与按钮状态、四种语义反馈配色、加载与成功空态，以及桌面／手机确认弹窗的外观。

- [HTML](theme-components-v4.html)、[桌面总览](theme-components-v4-desktop.png)、[手机总览](theme-components-v4-mobile.png)。
- [桌面弹窗](theme-components-v4-desktop-dialog.png)、[手机弹窗](theme-components-v4-mobile-dialog.png)、[键盘焦点](theme-components-v4-focus.png)。
- [浏览器检查记录](theme-components-v4-checks.json)：四种尺寸／文字放大场景、初始与弹窗状态及本地交互。

各区为独立虚构样例，确认弹窗仅重置本地输入草稿，不操作真实钱包或已保存数据。

具体范围与原件摘要见 [v4 确认记录](theme-components-v4-approval.json)。HTML、截图和[原始审阅记录](theme-components-v4-review.json)保留提交时状态。完整手机总览提供了链接，键盘焦点图作为检查证据保存，未把它们当作单独展示确认。后续列表控件见上方 v5；其余组件与动效按范围继续设计。

## v3 导航、页头与页面密度（视觉已确认）

[布局视觉基准](../layout-proposal.md)已获用户确认，反馈为“舒服”。沿用 v1 配色与 v2 字体，确认截图中的 224px 桌面侧栏、64px 顶栏、页面操作与查询分组、内容间距及手机抽屉和逐币摘要行。

- [v3 HTML](theme-layout-v3.html)、[桌面截图](theme-layout-v3-desktop.png)、[手机截图](theme-layout-v3-mobile.png)、[手机导航展开](theme-layout-v3-mobile-navigation.png)。
- [浏览器检查记录](theme-layout-v3-checks.json)：四种视口、导航／抽屉与本地交互检查。

全部为虚构数据与局部交互示例，正式 Token 功能、权限、接口与数据规则不由本样板替代。

具体范围及原件摘要见 [v3 确认记录](theme-layout-v3-approval.json)。HTML、截图和原始审阅记录保留提交时的“待确认”内容；未展示的侧栏收起态、账户／二级菜单仍需设计。组件确认进度见上方 v4 及[主题需求](../visual-theme.md)。

## v2 字体与数据排版（视觉已确认）

[v2 字体与数据排版](../typography-proposal.md)的视觉效果已获用户确认，原话为“可以 舒服”。沿用下方已确认配色，Inter、地址专用 JetBrains Mono、文字层级与数字对齐作为后续视觉基准。样板还提供金额、百分比、零值和缺失值、可展开复制的完整地址与哈希，具体业务语义和交互契约不由截图认可整体确认。

- [v2 HTML](theme-typography-v2.html)、[桌面截图](theme-typography-v2-desktop.png)、[手机截图](theme-typography-v2-mobile.png)。
- [浏览器检查记录](theme-typography-v2-checks.json)：桌面、手机、320px 窄视口及 200% 文字放大检查。
- [字体来源与文件摘要](files/typography-v2-font-sources.json)：字体及许可均在 `files/` 中，使用本地资源。

确认范围与原件摘要见 [v2 确认记录](theme-typography-v2-approval.json)。v1、v2 HTML、截图与原始审阅记录保留当时“待确认”标记，避免改动实际提交审阅的版本。v2 的检查结果、已修正的文字放大挤压问题和保留 Inter 的扫描说明见字体基准文档。

## v1 配色与按钮样板（已确认）

- [可交互 HTML 样板](theme-sample-v1.html)：可直接在浏览器打开；同目录 `files/` 保存参考图。
- [桌面截图](theme-sample-v1-desktop.png)：1440 × 1050 视口下的整页截图。
- [手机截图](theme-sample-v1-mobile.png)：390 × 844 视口下的整页截图，表格支持在自身区域横向滚动。
- [浏览器检查结果](theme-sample-v1-checks.json)：仅用于这份独立样板，不是正式产品验收。

v1 审阅范围为背景与面板的深浅、文字颜色的清晰度、青绿色的使用强度，以及主要与次要操作的视觉层级。用户反馈“没问题，我喜欢这个感觉”，这部分现已确认。页面布局、字体、表格结构和业务字段不是 v1 整体确认对象；字体后续由 v2 单独确认。

原始 HTML 与截图保留提交审阅时的“待确认”和“候选”标记，作为不可混淆的审阅原件；当前确认范围及文件摘要见[确认记录](theme-sample-v1-approval.json)。后续样板另存版本，并沿用本轮配色基准。

## 参考与已确认基础色

主要参考为[仓库保存的 Nansen 官网截图](../reference/nansen-2026-09-13/README.md)。页面底色、面板、主要文字、辅助文字和青绿色来自该截图的实际像素采样；这些是截图观察值，不声称全部等同于供应商源码变量。下表基础色现按 v1 确认为基准；样板中其余控件和业务状态颜色仍需在相应设计主题中细化。

| 用途 | v1 已确认基础色 | 来源依据 |
| --- | --- | --- |
| 页面底色 | `#06080B` | 官网功能区域截图采样 |
| 内容面板 | `#0F1114` | 官网数据面板截图采样 |
| 悬停层次 | `#181D22` | 样板派生色 |
| 装饰分隔线 | `#252A30` | 样板派生色；输入与次要按钮另用更明显的边界 |
| 主要文字 | `#FFFFFF` | 官网标题截图采样 |
| 辅助文字 | `#9FA0A1` | 官网面板文字截图采样 |
| 青绿色强调 | `#00FFA7` | 官网截图采样，也与官方品牌规范一致 |
| 主要按钮文字 | `#06080B` | 使用近黑色与亮青绿色组合 |

普通文字的计算对比度：近黑按钮文字／青绿色约 15.15:1；白字／内容面板约 18.91:1；辅助文字／内容面板约 7.22:1。最终真实组件仍需在实现后验证，不能以色值计算代替整页验收。

## 样板行为与真实性

样板包含钱包查询输入框、时间选择、总览、三条演示代币、主次按钮及禁用状态。所有钱包与数字均明确标注为演示；切换时间只演示选中状态，数值保持不变。查询、刷新和加载更多均为本地交互，不访问钱包接口、不消耗 Nansen API 额度、不保存用户业务数据。

右侧可以展开 Nansen 原始参考图。[files/nansen-reference.png](files/nansen-reference.png) 是已保存官网截图的未修改副本；归属及来源见[参考说明](../reference/nansen-2026-09-13/README.md)。样板未采用 Nansen Logo；ATHENA 字样、页面内容及布局用于本轮配色演示。

## 本轮检查

- Playwright 检查桌面和手机两种视口，最终均没有整页横向溢出；手机表格内部横向滚动。
- 本地演示的时间选择、地址校验、查询与刷新状态、键盘焦点和参考图展开通过；两种视口无页面脚本错误。
- axe 的 WCAG 2 A/AA、2.1 AA 自动检查在这两个静态初始视图未发现违规，不代表全部状态或正式产品已通过完整无障碍验收。
- 小字号提示按实际可读性处理，将样板中的 10／11px 辅助文字调至至少 12px；手机网格最小宽度导致的整页溢出已修正。
- Impeccable 的 `overused-font: inter` 提示按参考方向保留，例外仅限定 v1 HTML 样板：Nansen 官方品牌规范列出 Inter，登记该例外时字体尚待另行确认，随后已在 v2 确认视觉效果。未禁用整条规则或其他文件检查。
- 检查脚本初次使用隐式浏览器上下文导致 axe 无法执行，改用独立 `browser.newContext()` 后重验；参考图路径按预览服务器的 `files/` 路由修正后，确认图像能够实际解码。
- 独立视觉审阅结论为 `ship`：这份样板足以提交本轮配色确认，无需额外修复；该结论仅覆盖样板的可审阅性，不代替用户对颜色与按钮效果的确认。

本次没有修改 `ui/` 中的正式页面，也没有执行全站主题替换。正式实现与交付仍遵循[主题需求](../visual-theme.md)中后续设计、计划和验收安排。

[返回主题需求](../visual-theme.md)

## v16 管理员 Service Status（所展示视觉已确认）

[视觉提案说明](../service-status-proposal.md)将当前纵向堆叠的三个状态来源重组为 Services、Notifications、Trader Sync 页签，并始终保留各来源状态。它沿用 v15 管理员壳和已确认的单一深色视觉；正式 `ui/` 未修改。

- 主要审阅图：[Services 桌面](theme-service-status-v16-services-desktop.png)、[Notifications 桌面](theme-service-status-v16-notification-desktop.png)、[Services 手机](theme-service-status-v16-services-mobile.png)、[Trader Sync 手机](theme-service-status-v16-trader-mobile.png)
- 其他提案图：[Notifications 手机](theme-service-status-v16-notification-mobile.png)、[Trader Sync 桌面](theme-service-status-v16-trader-desktop.png)、[stale 桌面](theme-service-status-v16-stale-desktop.png)、[不可用手机](theme-service-status-v16-unavailable-mobile.png)、[手机导航](theme-service-status-v16-navigation-mobile.png)、[320px 窄屏](theme-service-status-v16-narrow.png)、[200% 根字号](theme-service-status-v16-zoom.png)
- 当前 React 对照：[桌面](theme-service-status-v16-before-desktop.png)、[手机](theme-service-status-v16-before-mobile.png)
- 可复核材料：[HTML](theme-service-status-v16.html)、[合成数据](theme-service-status-v16-data.json)、[浏览器检查](theme-service-status-v16-checks.json)、[当前 React 采集](theme-service-status-v16-before-capture.json)、[独立审阅](theme-service-status-v16-review.json)

11 张提案图与 2 张当前 React 对照图均有来源记录；浏览器检查覆盖 1440×900、390×844、320×844 和 720×900／根字号 200%，142 项检查通过，没有浏览器错误、外部请求或整页横向溢出。当前 React 使用同一合成数据和 GET fixture，1 个测试通过（2.5s），无写请求。独立审阅处置为 `ship`，只说明提案可交用户判断。用户对所展示四张图反馈“舒服 继续推进”，所展示的 Services 桌面／手机、Notifications 桌面和 Trader Sync 手机视觉已确认，范围见 [v16 确认记录](theme-service-status-v16-approval.json)；其他辅助状态未逐图确认，原始审阅保留提交时的 `awaiting_user_review` 状态；原型不执行真实轮询、授权、服务健康或导航，正式 `ui/` 未修改。

## v17 管理员 Etherscan Gateways（所展示视觉已确认）

[视觉提案说明](../etherscan-gateways-proposal.md)把网关运行状态和实际请求测试分别组织为 `Gateways` 与 `Live Probe`，两个来源保留各自状态和时间。页面沿用 v15/v16 管理员壳及已确认的 Nansen 单一深色视觉；正式 `ui/` 未修改。

- 主要审阅图：[Gateways 桌面](theme-etherscan-gateways-v17-gateways-desktop.png)、[Live Probe 桌面](theme-etherscan-gateways-v17-probe-desktop.png)、[Gateways 手机](theme-etherscan-gateways-v17-gateways-mobile.png)、[Live Probe 手机](theme-etherscan-gateways-v17-probe-mobile.png)
- 其他提案图：[API key 与时序桌面](theme-etherscan-gateways-v17-keys-desktop.png)、[连接信息手机](theme-etherscan-gateways-v17-connection-mobile.png)、[运行中手机](theme-etherscan-gateways-v17-running-mobile.png)、[启动失败手机](theme-etherscan-gateways-v17-start-error-mobile.png)、[320px 网关](theme-etherscan-gateways-v17-narrow.png)、[200% 文字 Probe](theme-etherscan-gateways-v17-zoom.png)
- 当前 React 对照：[桌面](theme-etherscan-gateways-v17-before-desktop.png)、[手机](theme-etherscan-gateways-v17-before-mobile.png)
- 可复核材料：[HTML](theme-etherscan-gateways-v17.html)、[合成数据](theme-etherscan-gateways-v17-data.json)、[浏览器检查](theme-etherscan-gateways-v17-checks.json)、[当前 React 采集](theme-etherscan-gateways-v17-before-capture.json)、[独立审阅](theme-etherscan-gateways-v17-review.json)

10 张提案图与 2 张当前 React 对照图均有来源记录，并与 `.impeccable/review/v17/` 中对应截图逐字节一致，来源扫描为 12 张、0 缺失。样板在 1440×900、390×844、320×844 和 720×900／根字号 200% 下通过 202 项断言，没有整页横向溢出、JavaScript 错误、外部或真实 API 请求。当前 React 只截获同一组合成 GET，1 项 Playwright 通过（2.8s），没有 POST。独立审阅处置为 `ship`，只表示静态提案可交用户判断；用户对四张主图反馈“舒服”，所展示 Gateways 与 Live Probe 的桌面／手机视觉已确认，范围见 [v17 确认记录](theme-etherscan-gateways-v17-approval.json)；其他辅助状态未逐图确认，原始审阅保留提交时的 `awaiting_user_review` 状态。原型不证明真实权限、探针、轮询、竞态、额度或 full-stack smoke。Inter 与经实测不成立的 hover 对比度提示仅在本 HTML 内持久忽略，没有全局放宽规则。

## v18 管理员系统通知列表与详情（所展示视觉已确认）

[视觉提案说明](../system-notifications-proposal.md)将通知列表与详情作为同一阅读流程，沿用 v15 管理员壳和已确认的单一深色视觉。列表的七个字段合并为五列，详情依次阅读消息、投递结果、时间和技术标识；正式 `ui/` 未修改。

- 主要审阅图：[列表桌面](theme-system-notifications-v18-list-desktop.png)、[详情桌面](theme-system-notifications-v18-detail-desktop.png)、[列表手机](theme-system-notifications-v18-list-mobile.png)、[详情手机](theme-system-notifications-v18-detail-mobile.png)
- 辅助提案图：[测试弹窗手机](theme-system-notifications-v18-test-mobile.png)、[320px 详情](theme-system-notifications-v18-narrow.png)、[200% 文字列表](theme-system-notifications-v18-zoom.png)
- 当前 React 对照：[列表桌面](theme-system-notifications-v18-before-list-desktop.png)、[详情桌面](theme-system-notifications-v18-before-detail-desktop.png)、[列表手机](theme-system-notifications-v18-before-list-mobile.png)、[详情手机](theme-system-notifications-v18-before-detail-mobile.png)
- 可复核材料：[HTML](theme-system-notifications-v18.html)、[合成数据](theme-system-notifications-v18-data.json)、[浏览器检查](theme-system-notifications-v18-checks.json)、[当前 React 采集](theme-system-notifications-v18-before-capture.json)、[独立审阅](theme-system-notifications-v18-review.json)

7 张提案图与 4 张当前 React 对照图均有来源记录，并与 `.impeccable/review/v18/` 中对应截图逐字节一致，来源扫描为 11 张、0 缺失。样板在 1440×900、390×844、320×844 和 720×900／根字号 200% 下通过 194 项断言，没有整页横向溢出、JavaScript 错误或外部请求。当前 React 只截获合成 GET，1 项 Playwright 通过（case 2.5 秒，总计 3.1 秒），没有 POST。独立审阅处置为 `ship`，只表示静态提案可交用户判断；因新 reviewer 受 thread 上限阻止，复用了未参与 v18 设计的 v10 documenter 完成独立复核。v18 所展示列表／详情的桌面／手机视觉已确认；常规辅助状态由实现者沿既有规则验证，不逐图增加审批。原型不证明真实权限、Telegram 投递、异步行为或 full-stack smoke。

用户已对四张主图反馈“舒服”，具体范围见 [v18 确认记录](theme-system-notifications-v18-approval.json)。原始审阅与资产保持不变；本次认可不表示正式 UI 已实施。

## v19 四个管理员业务页面（所展示视觉已确认）

[视觉提案说明](../admin-business-batch-proposal.md)把 Trader Sync 订阅列表／概要和 Profit Sharing 轮次列表／详情作为四个实际页面一次交付。共用主题、字体、组件和管理员壳直接沿用；每页桌面／手机为集中审阅主图，辅助状态继续作为实现证据。

### Trader Sync

- 主要审阅图：[列表桌面](theme-admin-trader-sync-v19-list-desktop.png)、[概要桌面](theme-admin-trader-sync-v19-detail-desktop.png)、[列表手机](theme-admin-trader-sync-v19-list-mobile.png)、[概要手机](theme-admin-trader-sync-v19-detail-mobile.png)
- 辅助提案图：[身份展开手机](theme-admin-trader-sync-v19-identity-mobile.png)、[320px](theme-admin-trader-sync-v19-narrow.png)、[200% 文字](theme-admin-trader-sync-v19-zoom.png)
- 当前 React 对照：[列表桌面](theme-admin-trader-sync-v19-before-list-desktop.png)、[概要桌面](theme-admin-trader-sync-v19-before-detail-desktop.png)、[列表手机](theme-admin-trader-sync-v19-before-list-mobile.png)、[概要手机](theme-admin-trader-sync-v19-before-detail-mobile.png)
- 可复核材料：[HTML](theme-admin-trader-sync-v19.html)、[合成数据](theme-admin-trader-sync-v19-data.json)、[浏览器检查](theme-admin-trader-sync-v19-checks.json)、[当前 React 采集](theme-admin-trader-sync-v19-before-capture.json)

### Profit Sharing

- 主要审阅图：[列表桌面](theme-admin-profit-sharing-v19-list-desktop.png)、[详情桌面](theme-admin-profit-sharing-v19-detail-desktop.png)、[列表手机](theme-admin-profit-sharing-v19-list-mobile.png)、[详情手机](theme-admin-profit-sharing-v19-detail-mobile.png)
- 辅助提案图：[创建弹窗手机](theme-admin-profit-sharing-v19-create-mobile.png)、[320px](theme-admin-profit-sharing-v19-narrow.png)、[200% 文字](theme-admin-profit-sharing-v19-zoom.png)
- 当前 React 对照：[列表桌面](theme-admin-profit-sharing-v19-before-list-desktop.png)、[详情桌面](theme-admin-profit-sharing-v19-before-detail-desktop.png)、[列表手机](theme-admin-profit-sharing-v19-before-list-mobile.png)、[详情手机](theme-admin-profit-sharing-v19-before-detail-mobile.png)
- 可复核材料：[HTML](theme-admin-profit-sharing-v19.html)、[合成数据](theme-admin-profit-sharing-v19-data.json)、[浏览器检查](theme-admin-profit-sharing-v19-checks.json)、[当前 React 采集](theme-admin-profit-sharing-v19-before-capture.json)

统一[独立审阅](theme-admin-batch-v19-review.json)包含 30 个资产摘要。22 张 Chromium 截图中 14 张为提案、8 张为当前 React 对照，全部嵌入来源并与 `.impeccable/review/v19/` 逐字节一致，来源扫描为 22 张、0 缺失；既有 265 个已批准资产摘要保持不变。独立 reviewer 初次要求补齐 seed 来源与 Profit Sharing Create／Draft 0–5 人 roster；1 个修正批完成后最终处置为 `ship`。因 thread limit 复用了未参与 v19 实现的 v10 reviewer；该结论不代表用户批准或生产验收。

Trader Sync 四种尺寸下 202 项本地断言通过；Profit Sharing 七种截图条件覆盖同样四类尺寸，并通过键盘入口、Draft／roster、阶段门槛、匿名、冲突和本地确认检查。两份当前 React 各 1 项 Playwright 通过，全部只截获合成 GET；页面没有整页横向溢出、JavaScript 错误或外部请求。create-mobile 使用真实 390×844 视口，文档保持顶部，弹窗正文滚动 206px 到 roster 控件，不是弹窗正文顶部取景。四页八张主图所展示视觉已确认，正式 `ui/` 未修改。

用户于 2026-09-14 明确确认本批无须调整，范围见 [v19 确认记录](theme-admin-batch-v19-approval.json)。原始原型、截图与审阅记录保持不变；辅助状态按既定规则验证，不追加逐图审批，正式前端后续按本批视觉基准实施。

## v20 四个会员基础业务页面（展示视觉已确认）

[集中视觉提案](../member-foundations-batch-proposal.md)一次交付 Wallets 私有钱包管理、Solana 发行候选列表、会员 Profit Sharing 轮次列表和轮次详情四个实际页面。三份 HTML 共对应 27 张 PNG：8 张主图、11 张辅助提案图、8 张当前 React 对照图。桌面／手机一起审阅，辅助状态用于覆盖验证，不逐张增加用户审批。

### Wallets 私有钱包管理

- 主要审阅图：[桌面](theme-wallets-v20-desktop.png)、[手机](theme-wallets-v20-mobile.png)
- 辅助提案图：[320px](theme-wallets-v20-narrow.png)、[200% 根字号](theme-wallets-v20-zoom.png)、[手机详情](theme-wallets-v20-detail-mobile.png)、[手机创建](theme-wallets-v20-create-mobile.png)
- 当前 React 对照：[桌面](theme-wallets-v20-before-desktop.png)、[手机](theme-wallets-v20-before-mobile.png)
- 可复核材料：[HTML](theme-wallets-v20.html)、[合成数据](theme-wallets-v20-data.json)、[最终浏览器检查](theme-wallets-v20-checks.json)、[当前 React 采集](theme-wallets-v20-before-capture.json)

### Solana 发行候选列表

- 主要审阅图：[桌面](theme-solana-v20-desktop.png)、[手机](theme-solana-v20-mobile.png)
- 辅助提案图：[320px](theme-solana-v20-narrow.png)、[200% 根字号](theme-solana-v20-zoom.png)、[手机链上证据](theme-solana-v20-evidence-mobile.png)
- 当前 React 对照：[桌面](theme-solana-v20-before-desktop.png)、[手机](theme-solana-v20-before-mobile.png)
- 可复核材料：[HTML](theme-solana-v20.html)、[合成数据](theme-solana-v20-data.json)、[最终浏览器检查](theme-solana-v20-checks.json)、[当前 React 采集](theme-solana-v20-before-capture.json)

### 会员 Profit Sharing 轮次列表与详情

- 主要审阅图：[列表桌面](theme-member-profit-sharing-v20-list-desktop.png)、[列表手机](theme-member-profit-sharing-v20-list-mobile.png)、[详情桌面](theme-member-profit-sharing-v20-detail-desktop.png)、[详情手机](theme-member-profit-sharing-v20-detail-mobile.png)
- 辅助提案图：[手机匿名投票](theme-member-profit-sharing-v20-voting-mobile.png)、[桌面结果](theme-member-profit-sharing-v20-closed-desktop.png)、[320px](theme-member-profit-sharing-v20-narrow.png)、[200% 根字号](theme-member-profit-sharing-v20-zoom.png)
- 当前 React 对照：[列表桌面](theme-member-profit-sharing-v20-before-list-desktop.png)、[列表手机](theme-member-profit-sharing-v20-before-list-mobile.png)、[详情桌面](theme-member-profit-sharing-v20-before-detail-desktop.png)、[详情手机](theme-member-profit-sharing-v20-before-detail-mobile.png)
- 可复核材料：[HTML](theme-member-profit-sharing-v20.html)、[合成数据](theme-member-profit-sharing-v20-data.json)、[最终浏览器检查](theme-member-profit-sharing-v20-checks.json)、[当前 React 采集](theme-member-profit-sharing-v20-before-capture.json)

Wallets 的 6 张图／9 项行为和 Solana 的 5 张图／8 项行为通过本地检查；操作区域修正后，两者共 112 个几何样本均至少 44×44px。Profit Sharing 检查覆盖 8 种截图条件及既定导航、草稿、封存、匿名投票和结果流程。四类尺寸为 1440×900、390×844、320×844 及 720×900／根字号 200%；Wallets 两张手机弹窗图取实际视口、正文顶部，其余为全页图。检查未见整页横向溢出、重复 ID、JavaScript 错误或外部请求。

当前 React 对照只截获合成 GET；Wallets／Solana 合计 4 张，Profit Sharing 的 1 项 Playwright 通过（3.6 秒、4 张）。27 张 PNG 均有来源记录并与 `.impeccable/review/v20/` 镜像同字节，来源扫描为 0 缺失。[统一审阅记录](theme-member-foundations-v20-review.json)保存最终资产摘要、验证入口与独立 `ship` 处置；一个修正批已恢复 Profit Sharing 既定导航并补齐 Wallets／Solana 操作区域，复判范围与源码阅读限制见审阅原文。

用户于 2026-09-14 明确反馈“没问题，审批通过”，整批确认四页八张主图，范围与 40 份交付文件摘要见 [v20 确认记录](theme-member-foundations-v20-approval.json)。原型、截图及交付时审阅 JSON 保持不变；辅助状态继续沿既定规则覆盖与验证，不补写成用户逐图看过。三份 HTML 的一次合并 detector 仅保留三项已确认 Inter 与两项 hover 误判的文件范围例外，没有全局放宽。原型未执行真实身份、密钥托管、上传／导入、Solana 采集、提案写入或投票，正式 `ui/` 未改版。本轮未启停服务，借用的既有环境保持原样，临时浏览器已关闭；完整环境与验证边界见[提案说明](../member-foundations-batch-proposal.md)。

本批属于[18 页、4／8／6 分批安排](../remaining-pages-plan.md)的第一批，第二批 v21 八页、第三批 v22 六页及六项共用适配所展示主视觉也均已整批确认；Trader Sync 本轮排除，已独立确认的 [Nansen 钱包战绩 v1](../../token/wallet-analytics-page-proposal.md)直接沿用。

## v22 Worm Trading 与共用页面（展示视觉已确认）

[集中视觉提案](../worm-and-common-batch-proposal.md)承接第三批六个 Worm 实际业务页面（七条正式路由）与六项共用界面差异；[二十四张桌面／手机主图](../worm-and-common-batch-proposal.md#原型截图与审阅材料)集中交付，反馈状态不增加业务页面。用户于 2026-09-14 反馈“确认✅”，整批确认所展示视觉，范围及 138 份交付资产摘要见 [v22 确认记录](theme-worm-and-common-v22-approval.json)。原型、截图、数据、检查及[审阅记录](theme-worm-and-common-v22-review.json)保持原字节，`awaiting_user_review` 是交付时历史状态；辅助图不补写成用户逐图看过。

| 原型域 | 页面入口 | 证据 |
| --- | --- | --- |
| Worm Assets | [资产](theme-worm-assets-v22.html) | [数据](theme-worm-assets-v22-data.json)、[检查](theme-worm-assets-v22-checks.json)、[React 对照](theme-worm-assets-v22-before-capture.json) |
| Worm Combinations | [列表](theme-worm-combinations-v22.html?view=list)、[编辑器](theme-worm-combinations-v22.html?view=editor)、[预览](theme-worm-combinations-v22.html?view=preview) | [数据](theme-worm-combinations-v22-data.json)、[检查](theme-worm-combinations-v22-checks.json)、[React 对照](theme-worm-combinations-v22-before-capture.json) |
| Worm Executions | [记录](theme-worm-executions-v22.html?view=list)、[详情](theme-worm-executions-v22.html?view=detail) | [数据](theme-worm-executions-v22-data.json)、[检查](theme-worm-executions-v22-checks.json)、[React 对照](theme-worm-executions-v22-before-capture.json) |
| 共用适配 | [管理员登录](theme-common-adaptations-v22.html?view=login)、[共享注册](theme-common-adaptations-v22.html?view=register)、[Profile](theme-common-adaptations-v22.html?view=profile)、[Access](theme-common-adaptations-v22.html?view=access)、[会员 Help](theme-common-adaptations-v22.html?view=member-help)、[管理员 Help](theme-common-adaptations-v22.html?view=admin-help)、[反馈证据](theme-common-adaptations-v22.html?view=feedback) | [数据](theme-common-adaptations-v22-data.json)、[检查](theme-common-adaptations-v22-checks.json)、[React 对照](theme-common-adaptations-v22-before-capture.json) |

当前 121 张 PNG 中 81 张为提案（24 主图／57 辅助）、40 张为当前 React 合成对照，来源扫描 0 缺失。每页包含桌面、手机、320px 与 200% 根字号；Worm 放大视口 720×900，共用为 720×1000。十一张原生弹窗取 390×844 实际视口，其他按采集记录提供全页；详细状态与尺寸见提案及 checks。

初审逐图查看 120 张输入图及两张 v21 参考；组合错误恢复、执行证据时间、Worm 六页放大头像三项经一个命名修正批关闭，新增一张恢复保存手机图。最终复审仅重读修正包 47 张图及相关回归，结合初审处置为 `ship`。一次 detector 的八项文件范围例外分别为四项已批准 Inter 和四项深字／mint 实际渲染反证，28 样本最低 15.145:1；低对比 CLI 的 `*` 限于四个 v22 HTML，不是全局禁用规则。

正式 `ui/` 与此前 469 份预览文件保持原状，本次确认也未修改 v22 交付原件。全部合成数据和本地交互只说明原型的有界表现，真实身份、权限、Help 资源、供应商、凭据、签名、Cash Out 与订单未验。本任务没有启停服务，临时浏览器已关闭，借用的既有 localhost:4000 保留；环境与详细未验边界见提案。[覆盖归属核对](../theme-refactor-coverage.md)、[技术方案](../../../superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)及[实施计划](../../../superpowers/plans/2026-09-14-web-ui-theme-refactor.md)现已整理；用户批准的主视觉直接沿用，正式代码与真实验收尚未开始。
