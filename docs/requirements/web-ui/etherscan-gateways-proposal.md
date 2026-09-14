# 管理员 Etherscan Gateways 页面视觉基准（v17）

> 状态：所展示 Gateways 与 Live Probe 的桌面／手机视觉已确认；正式 `ui/` 未修改。具体范围见 [v17 确认记录](previews/theme-etherscan-gateways-v17-approval.json)。
>
> 视觉基础：沿用已确认的 Nansen 单一深色世界、v1 配色、v2 Inter／JetBrains Mono 字体、v3 布局、v4 状态反馈，以及 v15/v16 管理员应用壳与独立来源状态。本提案不改变现有权限、网关配置或探针服务端契约。

关联现状：[当前页面](../../../ui/src/app/admin/pages/etherscan-gateways.tsx)、[状态与探针客户端](../../../ui/src/app/shared/services/service-status-service.ts)和[已实现 Etherscan Manager](../../design/blockchain-data/etherscan-manager.md)。

## 目标与现状差距

管理员需要分别判断网关进程是否可达、实际 Etherscan 请求是否通过。v17 将这两个来源组织为 `Gateways` 与 `Live Probe` 页签，每个页签保留自己的状态摘要；网关运行不能推导探针通过，探针结果也不能覆盖当前网关健康。

页面继续使用 224px 桌面侧栏、64px 顶栏、32px 主区边距；手机使用 20px 边距，导航抽屉为 280px 且最大宽度为视口减 48px。页面采用 `#06080B` 背景、`#0F1114` 面板、`#181D22` 交互层、`#252A30` 分隔线、白／灰文字和青绿色操作／选中关系。成功、警告、失败继续使用相互独立的语义色。Inter 用于标题、正文、控件和表格数字；地址、URL、run ID 与错误原文使用 JetBrains Mono。

## Gateways 来源

页头 `Refresh gateways` 只刷新网关来源，不重新运行探针，也不修改已有探针结果。网关计数平排显示；每条记录直接展示 Address、Reachable、Runtime、Latency、Checked 和完整 Error。Base URL 默认折叠，展开后可以读取和复制。地址、URL 与错误均提供复制能力，不能只靠 tooltip 或截断读取。

桌面保留紧凑记录结构。390px 手机按每条网关纵向组织三项主要事实；320px 改为两列事实布局，避免状态与延迟碰撞。不可达、运行时错误、读取失败、未检查和无记录都必须以文字说明，不能只用颜色表达。

## Live Probe 输入与权限边界

运行表单保留两个现有输入：`Interval` 为 1–1000ms 整数、默认 10；`Requests per API key` 为 1–20 整数、默认 6。正式功能使用预配置 API key 与 gateway，并消耗真实请求额度；只有具备现有管理员写权限的用户可以启动。页面不增加在线编辑 key、管理网关、重启或取消能力；运行中禁止重复提交。

编辑表单不能改写上一轮结果所记录的参数。提交后字段与再次提交保持锁定，直到本次运行完成或失败。服务端只在完成时发布计数，页面不虚构逐请求实时进度；无需增加确认弹窗或取消按钮。本地原型只模拟运行及重试状态，不发送探针请求，也不消耗额度。

## 结果、汇总与失败语义

整体结果直接使用服务端返回的 `result`。总请求数为 API key 数量乘以 `requestsPerKey`，不再乘以 gateway 数量；成功门槛为 `ceil(total × 0.9)`。本次合成数据恰好使用 3 个网关、3 个 API key、18 次请求、16 次成功、2 次限流、所需成功 17、差 1 次，仅用于演示，不定义固定容量。

按 gateway 和按 API key 的行状态继续遵循现有前端规则：出现非 rate-limit 的严重失败时为 `Needs check`；否则成功率低于 90% 为 `Below 90%`；达到门槛且无严重失败时为 `OK`，尚无完成结果时显示 `—`。整体 `Pass` 可以与某一行 `Needs check` 同时出现，不能根据行颜色重新计算整体结果。

七种失败分类全部保留：Rate Limit、Authentication、Plan、Invalid Request、Malformed、Upstream 和 Other。用户可切换 `By Gateway`／`By API Key`。`Timing & run details` 默认收起，展开后包含 elapsed、start spread、start、finish、create 和 run ID；存在失败时，`Error samples` 默认展开并保留可复制的完整原文。

网关状态与 probe 使用独立时间源；样例中最近 probe 为 13:48、当前 gateway 为 13:50，不应合并成同一次检查。页面还需分别表达读取失败、尚未运行、启动失败、执行错误和运行中。启动失败继续保留上一轮结果，包括按 key 的汇总；本地重试和 mock 完成只切换样例展示。

## 异步与生产实现义务

当前 React 对网关健康进行 10 秒轮询；本页启动的 run ID 使用 1 秒轮询，并需处理进入页面后 latest 已处于运行中的继续跟踪。正式实现仍须检查请求取消、防陈旧响应、重复提交、权限、竞态和额度行为。静态原型没有证明后端健康、认证、真实探针或这些异步边界。

## 原型、截图与审阅材料

- 可操作原型：[theme-etherscan-gateways-v17.html](previews/theme-etherscan-gateways-v17.html)
- 共享合成数据：[theme-etherscan-gateways-v17-data.json](previews/theme-etherscan-gateways-v17-data.json)
- 主要审阅图：[Gateways 桌面](previews/theme-etherscan-gateways-v17-gateways-desktop.png)、[Live Probe 桌面](previews/theme-etherscan-gateways-v17-probe-desktop.png)、[Gateways 手机](previews/theme-etherscan-gateways-v17-gateways-mobile.png)、[Live Probe 手机](previews/theme-etherscan-gateways-v17-probe-mobile.png)
- 其他提案图：[API key 与时序桌面](previews/theme-etherscan-gateways-v17-keys-desktop.png)、[连接信息手机](previews/theme-etherscan-gateways-v17-connection-mobile.png)、[运行中手机](previews/theme-etherscan-gateways-v17-running-mobile.png)、[启动失败手机](previews/theme-etherscan-gateways-v17-start-error-mobile.png)、[320px 网关](previews/theme-etherscan-gateways-v17-narrow.png)、[200% 文字 Probe](previews/theme-etherscan-gateways-v17-zoom.png)
- 当前 React 对照：[桌面](previews/theme-etherscan-gateways-v17-before-desktop.png)、[手机](previews/theme-etherscan-gateways-v17-before-mobile.png)
- 可复核材料：[浏览器检查](previews/theme-etherscan-gateways-v17-checks.json)、[当前 React 采集](previews/theme-etherscan-gateways-v17-before-capture.json)、[独立审阅](previews/theme-etherscan-gateways-v17-review.json)

共 12 张 Chromium 截图，其中 10 张是提案、2 张是当前 React 对照；全部嵌入来源，并与 `.impeccable/review/v17/` 中对应截图逐字节一致，来源扫描为 12 张、0 缺失。独立 finish reviewer 的最终处置为 `ship`，五个审阅部分均完成且没有静态提案范围内的必须修正项。该结论只表示材料可以交给用户判断，不构成用户确认、正式实现或生产验收。既有 232 份已批准 artifact 摘要保持不变，其中包含 v16 本轮冻结的 18 份。

## 验证范围与限制

[浏览器检查](previews/theme-etherscan-gateways-v17-checks.json)覆盖 1440×900、390×844、320×844 和 720×900／根字号 200% 四种条件，202 项本地样板断言通过；没有整页横向溢出、JavaScript 错误、外部请求或真实 API 请求。两轮自身检查已经结束，修正了复制图标引用、320px 事实布局碰撞和 200% 文字下的输入高度，随后独立审阅未要求修改。

Impeccable detector 单次报告 Inter 饱和字体和白字／青绿 hover。Inter 是用户已确认的 v2 字体；后者与实际渲染不符，`Run Probe` hover 使用 `#06080B`／`#51FFC3`，对比度为 15.70:1，12 个 hover 样本均不低于 4.5:1。两个忽略都仅通过 CLI 限定在本 v17 HTML，没有全局关闭规则；临时执行证据 `.superpowers/v17-hover-contrast.json` 只记录本次检查，不是长期 API 契约。

[当前 React 采集](previews/theme-etherscan-gateways-v17-before-capture.json)使用同一组合成数据，只截获 GET；1 项 Playwright 测试通过，用时 2.8 秒，没有 POST，并保留源码 hash。它不是 full-stack smoke 或真实探针验收。

## 环境与确认范围

本任务没有启动新服务，所有任务浏览器均已关闭。采集沿用既有共享 root Vite：`http://localhost:4000`，PID `308228`，supervisor PID `307846`，仓库 `/home/yege/work/athena`，进程工作目录 `/home/yege/work/athena/ui`，实例 `athena-local-runtime`；运行目录及日志位于 `.run/athena-local-runtime/`，采集日志位于 `.superpowers/v17-current-playwright-artifacts/`。该环境按原归属保留，没有执行 `cd /home/yege/work/athena && make stop`。

本轮是既有视觉体系的普通扩展。任务开始前 `DESIGN.md` 与 `.impeccable/design.json` 已缺失，本轮按边界保持其缺失并记录既有漂移，不补建或修复。用户于 2026-09-14 对上一回复展示的四张主图反馈“舒服”，所展示 Gateways 与 Live Probe 的桌面／手机视觉已确认。具体范围见 [v17 确认记录](previews/theme-etherscan-gateways-v17-approval.json)；未直接展示的六张辅助图及其他状态未逐图确认。原始 HTML、合成数据、截图、检查与审阅记录保持不变，原始审阅保留提交时的 `awaiting_user_review` 状态，当前确认以独立确认记录为准。此次认可不表示正式 UI 已实现、真实探针已验收或全站设计已完成。

[返回主题需求](visual-theme.md)
