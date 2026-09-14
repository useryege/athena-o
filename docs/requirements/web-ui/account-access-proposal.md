# Account Center / Access & session 页面视觉基准

> 状态：v13 所展示的桌面／手机权限与会话页及手机 Google 待授权页视觉已确认；正式 `ui/` 未修改。
>
> 范围：现有会员 `/account/access` 在 v1–v12 已确认视觉体系中的普通扩展，只调整信息层级和展示方式，不改变登录、授权、刷新或退出规则。

关联现状：[Account Center 页面](../../../ui/src/app/shared/pages/account-center.tsx)、[展示模块](../../../ui/src/app/shared/access-modules.ts)、[权限摘要](../../../ui/src/app/shared/account-access.ts)、[账户模型](../../../ui/src/app/shared/models.ts)、[身份文案](../../../ui/src/app/shared/account-presentation.tsx)、[会员应用壳](../../../ui/src/app/member/app.tsx)与[版本服务](../../../ui/src/app/shared/services/version-service.ts)；长期契约见[账户访问控制](../../design/identity-access/account-access-control.md)、[账户资料与偏好](../../design/identity-access/account-profile-and-preferences.md)和[会员应用壳设计](../../design/web-ui/member-application-shell.md)。

## 目标与视觉方向

页面应先回答“当前账户可以使用哪些模块”，再让会员按需核对身份、会话时间、修订与版本。v13 沿用已确认的 Nansen 单一深色世界、Inter、地址专用 JetBrains Mono、紧凑 Account Center 页头、桌面局部导航和手机 Account section 选择器；模块权限置于 Current session 之前，辅助技术信息默认收起。

Appearance 只因已确认的单一深色目标从新界面移除，其他账户能力继续按既有权限显示。权限均为只读展示，本提案不增加编辑授权、申请开通、升级套餐、管理员入口、Nansen Key 设置或终止全部会话。

## 模块权限与账户标志

展示严格使用源码 `accountAccessDisplayModules` 的十项及顺序：Market Radar、Sports Live、Sports History、Managed OO、Worm Markets、Worm Trading、World Cup Corners、Solana、Wallet、Trader Sync。Token 只从本页展示中排除；解析、更新与 Active／Pending／Blocked 判定仍保留完整十一模块模型。

固定 active fixture 显示 `2 full · 3 read · 5 none`，权限标签保持源码文案 `Read & write`、`Read only`、`No access`。API Key 与 Profit Sharing 是彼此独立的账户标志，不并入模块权限汇总；Tier 只展示，不授予访问。

## Current session 信息

现有 ActiveAccessPage 的 15 项事实全部保留，并按阅读优先级分布：模块摘要；API Key 与 Profit Sharing 标志；只出现一次的 `Logged in Yes`；Username、Identity provider、完整可复制的 provider 身份、Role、Tier；以及默认折叠的 Account created、Last sign-in、Access revision、Issuer、UI version、API version 六项细节。身份时间为零时继续显示 `Not yet`，未知 provider 与缺失版本继续使用源码回退；API 版本来源保持 `/api/version`。

样板的 UTC+8、固定 `Last checked 10:00` 与 `v0.8.0-demo` 仅是合成演示上下文。当前 React 的 last-checked 使用采集时钟，UI version 已核对为 `latest`；未展示的时间、provider 和版本变体不因此获得视觉确认。

## Pending、身份与访问判定

Pending 表示 Google 邮箱或 Phantom 钱包所有权已验证，但业务访问尚未开通；它不是登录失败，也不能代表被禁用账户。Google 与 Phantom 必须保留各自的 provider 名称、标题、身份标签和完整可复制值，使“已验证身份”和“业务权限”保持清楚区分。

判定继续严格使用 `accountStatusForAccess`：`loginEnabled=false` 为 Blocked；管理员、`profitSharingEnabled=true` 或完整模块矩阵中任意非 `None` grant 为 Active；`apiKeyEnabled=true` 本身不会变成 Active。本次 Pending 演示数据为 `apiKeyEnabled=false`，所以截图没有 Security；这不建立“所有 Pending 用户都没有 Security”的新规则，普通会员启用 API Key 时现有 Account Center 规则仍允许显示 Security。

Pending 保留 Profile、刷新权限和退出。生产界面继续每 15 秒及窗口重新获得焦点时刷新，并保留身份、realm、权限竞态与退出防护；失败信息继续来自 `requestErrorMessage`。

## 原型交互与证据边界

[可操作 HTML](previews/theme-account-access-v13.html)支持本地切换状态、展开细节、复制、650ms 刷新忙碌态、退出提示、刷新失败与恢复；刷新时两个动作均禁用。它不执行真实轮询、权限检查、授权变更或退出，失败时保留当前身份与权限上下文。

[固定数据](previews/theme-account-access-v13-data.json)使用虚构 Google／Phantom 身份、权限和 API 版本。9 张原型图与 3 张当前 React 对照图均为带 Playwright 来源记录的 Chromium 截图，没有 AI 生成或修图；全部 12 张已经独立打开核对，审阅副本逐字节一致，来源扫描为 12 张、0 缺失。

当前 React 使用相同合成账户与 grants，只截获 bootstrap 和 version GET；1 个测试通过（3.0s）并生成 3 张截图，没有写请求。原型在 1440×900、390×844、320×844 与 720×900／根字号 200% 四个视口通过字体、对比度、横向溢出、页面错误、外部请求、文案、展开、复制、Pending 导航、忙碌态和键盘检查。两轮自身检查合并了重复登录状态，并在 800px 及以下堆叠独立账户标志；detector 仅依据 v2 已确认字体为本 HTML 保留 Inter 单文件例外。

## 审阅、环境与确认边界

用户于 2026-09-14 查看桌面默认页、手机默认页和手机 Google 待授权页后反馈“舒服”。这三张截图的阅读顺序、疏密、状态展示及操作层级成为已确认基准，具体范围见 [v13 确认记录](previews/theme-account-access-v13-approval.json)。未直接展示的辅助状态、管理员页面与真实服务行为未由本次反馈确认；原始 HTML、截图、检查与审阅记录保持不变。

[v13 独立审阅记录](previews/theme-account-access-v13-review.json)覆盖全部 12 张截图、源码契约与浏览器证据，处置为 `ship`，未提出需要修正的视觉问题。该结论只表示视觉提案与本地演示达到交付标准，不是用户视觉批准、正式实现或生产验收。真实 15 秒／焦点刷新、错误内容、账户与 realm 切换、权限竞态、身份防护和退出仍是正式改版验收义务；本地 fixture 对照不能替代全系统 smoke。

本任务没有启动服务。采集借用了任务前已存在的根目录 Vite `http://localhost:4000`（PID 308228，工作目录 `/home/yege/work/athena/ui`），所有新建浏览器上下文均已关闭，已有服务保持原样；该已有环境的 owner 停止命令是从仓库根目录执行 `make stop`，本任务未执行，完整归属见[采集记录](previews/theme-account-access-v13-before-capture.json)。根目录 `DESIGN.md` 与 `.impeccable/design.json` 继续保持不存在。
