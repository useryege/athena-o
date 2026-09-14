# v15 管理员导航与账户管理视觉基准

> 状态：v15 所展示的桌面账户管理、手机目录、手机权限详情和手机确认弹窗视觉已确认；正式 `ui/` 未修改，生产验收尚未开展。

## 范围与目标

本轮沿用 v1–v14 已形成的单一深色视觉系统，覆盖既有管理员侧栏、导航和 `/admin/accounts` 账户目录／详情，不新增业务能力。管理员从目录选择身份后直接进入 `Access`，并通过相邻的 `Profile`、`Identity` 页签核对资料与登录身份。其他管理员导航目的页只显示本地范围提示。

现状与后续实现以 [账户页](../../../ui/src/app/admin/pages/admin-accounts.tsx)、[账户服务](../../../ui/src/app/admin/accounts-service.ts)和[共享模块定义](../../../ui/src/app/shared/access-modules.ts)为直接依据，并结合[管理员应用入口](../../../ui/src/app/admin/app.tsx)与[共享模型](../../../ui/src/app/shared/models.ts)核对路由和字段。

本提案不提供创建账户、删除账户或提升管理员身份的入口。管理员账户的 access 在此只读，当前管理员自己的 profile 也在此只读；自己的资料继续由 Account Center 管理。

## 视觉与响应式布局

- 使用 `#06080B` 页面背景、`#0F1114` 面板、`#181D22` 控件底、`#252A30` 分隔线、白／灰文字和 `#00FFA7` 强调色；正文与数字使用 Inter，地址和技术标识使用 JetBrains Mono。
- 桌面保留 224px 管理员侧栏；账户工作区为 304px 目录与详情并列，详情默认显示 `Access`，并提供 `Profile`、`Identity` 页签。
- 手机按“目录 → 详情”分步展示，详情保留返回目录入口。导航抽屉宽 280px，最大宽度为视口减 48px。
- 模块说明可按需展开，完整地址允许换行。敏感权限缩减或新增 `Read & write` 前先显示影响说明；手机确认操作按纵向全宽排列，顺序为 `Cancel`、`Apply changes`。

## 目录、资料与权限契约

账户目录搜索当前已注册身份，状态可筛选；分页提供 25／50／100，默认 50。本原型固定为四个虚构账户、单页数据，只用于视觉与本地状态演示。

授权聚合保留源码定义的 11 个模块，界面展示 10 个；隐藏的 Token grant 在编辑和保存聚合时必须原样保留。Trader Sync 只允许 `No access`／`Read & write`，Solana 只允许 `No access`／`Read only`。登录、API Key、Profit Sharing 和各模块权限仍按现有授权模型处理，tier 只是展示属性，不授予访问权。

Profile 中 username 不可修改。Display name 与 tier 独立保存；display name 去除首尾空白后须为 1–80 个 Unicode 字符且不能包含控制字符。头像沿用 JPEG／PNG／WebP、最大 2 MiB 的现有边界。

Identity 展示 provider、状态、邮箱或钱包地址、账户 ID、创建时间和最后登录时间，不改变其授权含义。权限缩减须说明对登录、API Key、Profit Sharing 或模块使用的影响，新增 `Read & write` 须明确写入能力扩大。

## 正式实现义务

正式实现须保留真实的未保存草稿离开保护、revision 冲突处理、身份／session／realm 变化防陈旧，以及资料、tier、头像和权限保存竞态；还须覆盖真实目录分页、URL 状态、网络竞态及完整资料／头像并发处理。原型仅演示 Access：权限冲突会丢弃旧草稿并显示最新权威值，普通保存错误会保留草稿；这不覆盖全部真实失败路径。

原型中的搜索、筛选、切页、页签、展开、确认、复制、错误和恢复均为本地演示，不发出账户写请求。当前 React 对照仅拦截 GET；未执行真实账户写入、全栈 smoke 或生产验收。

## 审阅证据

| 类别 | 证据 |
| --- | --- |
| 四张主要提案图 | [桌面目录与 Access](previews/theme-admin-accounts-v15-desktop.png)、[手机目录](previews/theme-admin-accounts-v15-mobile-list.png)、[手机 Access](previews/theme-admin-accounts-v15-mobile-access.png)、[手机敏感变更确认](previews/theme-admin-accounts-v15-confirm-mobile.png) |
| 七张辅助图 | [桌面 Profile](previews/theme-admin-accounts-v15-profile-desktop.png)、[手机 Profile](previews/theme-admin-accounts-v15-profile-mobile.png)、[手机 Identity](previews/theme-admin-accounts-v15-identity-mobile.png)、[手机导航](previews/theme-admin-accounts-v15-navigation-mobile.png)、[手机空态](previews/theme-admin-accounts-v15-empty-mobile.png)、[320px](previews/theme-admin-accounts-v15-narrow.png)、[200% 文字](previews/theme-admin-accounts-v15-zoom.png) |
| 三张当前 React 对照 | [桌面](previews/theme-admin-accounts-v15-before-desktop.png)、[手机目录](previews/theme-admin-accounts-v15-before-mobile-list.png)、[手机详情](previews/theme-admin-accounts-v15-before-mobile-detail.png) |
| 可复核文件 | [可操作 HTML](previews/theme-admin-accounts-v15.html)、[固定虚构数据](previews/theme-admin-accounts-v15-data.json)、[原型检查](previews/theme-admin-accounts-v15-checks.json)、[当前 React 采集](previews/theme-admin-accounts-v15-before-capture.json)、[独立审阅记录](previews/theme-admin-accounts-v15-review.json) |

11 张提案图与 3 张 React 对照图均为带来源信息的 Playwright Chromium 截图，没有 AI 生成或后期修图；`.impeccable/review/v15` 中的审阅副本逐字节一致，14 张来源扫描为 0 缺失。四个视口 1440×900、390×844、320×844、720×900／根字号 200% 共 119 项检查通过，字体已加载，没有横向溢出、页面错误或外部请求；抽样正文对比度均不低于 4.5。当前 React 的 GET-only fixture 检查 1 项通过（2.9s），没有写请求。

两轮自身检查后，独立初审提出三项视觉一致性问题；一个修正批次恢复了 280px 手机抽屉、手机纵向全宽确认操作，以及 Profit Sharing／Etherscan Gateways 的独立语义图标。最终 `ship` 处置只复核这三项已经修正，四张预期 PNG 更新，其余十张图像数据不变；它不是新的全表面审计、用户批准或生产通过。

检测器中 Inter 是已确认字体；hover 低对比报告为计算误报，实际青绿色按钮使用深色文字，对比度 15.15:1，五个抽查 hover 状态最低 11.27:1。两项 file-scoped CLI 例外继续保留。两个固定 overlay 使用实际视口截图以避免无效的 full-page 延伸，其他 full-page 截图均从页面顶部采集。

## 环境与确认范围

用户于 2026-09-14 对上一回复所展示的桌面账户管理、手机账户目录、手机权限详情及手机敏感变更确认四张截图反馈“可以”。本次确认其布局、疏密、阅读顺序与操作层级，具体范围见 [v15 确认记录](previews/theme-admin-accounts-v15-approval.json)。其他辅助状态未由本次反馈逐图确认。

本轮没有启动服务，任务浏览器上下文均已关闭。对照采集借用既有 `http://localhost:4000` Vite：PID 308228，工作目录 `/home/yege/work/athena/ui`，supervisor PID 307846，实例 `athena-local-runtime`，日志位于 `.run/athena-local-runtime/`。该环境不属于本任务，未执行 `cd /home/yege/work/athena && make stop`，保持原样。

正式 `ui/` 未修改，195 个既有已确认视觉文件摘要保持不变。项目根 `DESIGN.md` 与 `.impeccable/design.json` 在本轮前已缺失；这是既有 Impeccable 上下文漂移，本次普通扩展不修复。v15 按四张展示图的确认范围进入后续改版基准。原始 HTML、数据、检查、截图和审阅记录保持不变；原始审阅 JSON 保留提交时的 `awaiting_user_review` 状态，当前确认以独立确认记录为准。
