# 管理员 Service Status 页面视觉基准（v16）

> 后续实施说明（2026-09-17）：v16 的三来源样板与批准证据保持原样；正式页面已在其后增加独立 **Module Access** 第四页签，展示六个访问键、OPEN／CLOSED、修改人和 RFC3339Nano 时间，并提供显式保存。该页签使用独立 5 秒可见 single-flight，未知结果禁用修改且不把不可读冒充 CLOSED；原三个状态来源仍各自 10 秒、彼此隔离。真实桌面／手机与保存、重启证据见[全栈验收](../../testing/full-stack-access-acceptance.md)。这项后续事实不表示早期 v16 截图已展示第四页签。

> 状态：所展示的 Services 桌面／手机、Notifications 桌面和 Trader Sync 手机视觉已确认；正式 `ui/` 未修改。具体范围见 [v16 确认记录](previews/theme-service-status-v16-approval.json)。
>
> 视觉基础：沿用已确认的 Nansen 单一深色世界、v1 配色、v2 字体与数字排版、v3 布局、v4 状态反馈，以及 v15 管理员导航与应用壳。本提案不改变 Service Status 的业务、授权、请求或缓存契约。

关联现状：[当前页面](../../../ui/src/app/admin/pages/service-status.tsx)、[服务状态读取](../../../ui/src/app/shared/services/service-status-service.ts)、[通知运行时读取](../../../ui/src/app/admin/notification-service.ts)、[Trader Sync 运行时模型](../../../ui/src/app/admin/trader-sync-models.ts)、[Trader Sync 读取](../../../ui/src/app/admin/trader-sync-service.ts)和[管理员应用壳设计](../../design/web-ui/administrator-application-shell.md)。

## 目标与现状差距

管理员应先辨认 Services、Notifications、Trader Sync 三个来源中哪一个需要关注，再读取该来源的完整状态、时间和诊断细节。当前 React 将三个来源从上到下连续堆叠；字段完整，但异常归属、来源状态和长页扫读成本较高。

v16 提议用三个页签组织来源，同时让每个页签一直显示自己的状态文字。默认打开 Services；选中页签后只显示该来源的完整资料。来源之间仍是并列而独立的事实，页签不把三者合成一个全局健康结论，某一来源失败也不遮盖另外两个来源的状态。

页面继续复用管理员应用壳：桌面 224px 侧栏、64px 顶栏和 32px 内容边距；手机使用 20px 内容边距，导航抽屉为 280px 且最大宽度为视口减 48px。页头保留 `Refresh`。字体继续使用 Inter；不透明 epoch、恢复 reason 和 clock 等需要逐字符核对的标识使用 JetBrains Mono。颜色保持已确认的 `#06080B` 页面、`#0F1114` 面板、`#181D22` 交互层、`#252A30` 分隔线、白／灰文字与 `#00FFA7` 强调关系。

## 三个来源的展示组织

### Services

Services 是默认页签，直接显示服务端 `checkedAt` 对应的 `Last checked`，并渲染接口返回的全部服务记录；本次合成 fixture 恰有 10 条，不定义生产数量。桌面按 Service、Status、Error 三列读取；手机改为带分隔线的逐条记录，不缩小成微型宽表。状态与完整错误信息同时保留，`Unreachable` 以文字表达，不能只靠颜色，也不能把 gRPC 可达性与进程运行活动合并为一个推断状态。

首次读取没有记录时显示明确空态；首次失败显示该来源的完整错误。已有成功结果后再失败，继续显示上一次成功值、原时间和 stale 提示。生产实现仍须保留源码错误细节，不能用统一短文案抹去来源错误。

### Notifications

Notifications 先显示客户端最近一次成功读取对应的 `Last received`。内容顺序固定为恢复状态、System／Account 队列比较、bot／poller／runtime 细节，使管理员先判断恢复和积压，再核对运行信息。

`failed`／`stopped` 顶层状态优先于 recovery，但 recovery 快照继续可见。`remainingMillis` 与 `elapsedMillis` 按服务端精确字符串显示，合法 `0` 显示 `0 ms`，缺失显示 `Unavailable`，浏览器不做本地倒计时。恢复 reason、started time 和 clock source 默认收起并可展开核对；reason 和 clock 作为标识采用 JetBrains Mono。

System 与 Account 分开显示 pending、retry、failed、sending、unknown 五类计数，不能合并两类队列，也不能省略零值。运行细节继续保留 bot、poller 和 runtime 提供的所有现有事实。

### Trader Sync

Trader Sync 使用服务端 `asOf` 作为 `As of`。先显示 collector 连接状态、不透明 collector epoch、filter revision 和 raw observation；随后显示 raw queue depth、raw persist in flight，并列出接口返回的全部 metric。本次合成 fixture 提供 7 条，这不是固定的生产数量。

每条 metric 保留原始值、unit 及 gauge／window／epoch 范围。超过 JavaScript 安全整数范围的字符串必须逐字保留；raw 缺失显示 `Not observable`，不能伪装成零；缺失 unit 显示 `Unavailable`。epoch 是不透明标识，不能解释成日期。不同 unit、window 或 service epoch 的数值不能相加或推导全局健康；`synthetic_window` 只是一条明确标注的演示 window metric，不表示现有 producer 已产生该指标。桌面使用四列表格，手机改为有分隔线的逐条 metric 记录。

## 时间、刷新与失败边界

三个来源的时间不能混用：Services `Last checked` 来自服务端 `checkedAt`，Notifications `Last received` 来自客户端最近成功读取，Trader Sync `As of` 来自服务端 `asOf`；全部按 UTC+8 展示。

正式实现必须保持三组独立的 10 秒可见 single-flight、各自请求与缓存、stale 值保留、安全取消，以及管理员 realm、身份和授权边界。手动 Refresh 可以共同触发 reload，但一个来源 pending 或失败不能阻塞其他来源。原型中的 Refresh 只恢复固定样例数据并展示本地忙碌态，不执行真实调度，也没有模拟上述并发行为。

本页面继续只读，不增加 restart、resend、配置或其他修复动作。指向其他管理员目的地的入口只显示原型范围提示，不执行真实导航。

## 原型、截图与审阅材料

- 可操作原型：[theme-service-status-v16.html](previews/theme-service-status-v16.html)
- 共享合成数据：[theme-service-status-v16-data.json](previews/theme-service-status-v16-data.json)
- 主要审阅图：[Services 桌面](previews/theme-service-status-v16-services-desktop.png)、[Notifications 桌面](previews/theme-service-status-v16-notification-desktop.png)、[Services 手机](previews/theme-service-status-v16-services-mobile.png)、[Trader Sync 手机](previews/theme-service-status-v16-trader-mobile.png)
- 其他提案图：[Notifications 手机](previews/theme-service-status-v16-notification-mobile.png)、[Trader Sync 桌面](previews/theme-service-status-v16-trader-desktop.png)、[stale 桌面](previews/theme-service-status-v16-stale-desktop.png)、[不可用手机](previews/theme-service-status-v16-unavailable-mobile.png)、[手机导航](previews/theme-service-status-v16-navigation-mobile.png)、[320px 窄屏](previews/theme-service-status-v16-narrow.png)、[200% 根字号](previews/theme-service-status-v16-zoom.png)
- 当前 React 对照：[桌面](previews/theme-service-status-v16-before-desktop.png)、[手机](previews/theme-service-status-v16-before-mobile.png)
- 可复核记录：[浏览器检查](previews/theme-service-status-v16-checks.json)、[当前 React 采集](previews/theme-service-status-v16-before-capture.json)、[独立审阅](previews/theme-service-status-v16-review.json)

共 13 张 Chromium 截图，其中 11 张是提案，2 张是当前 React 对照；均带来源记录，没有 AI 生成或修图。交付目录与 `.impeccable/review/v16/` 的截图副本逐字节一致，来源扫描为 13 张、0 缺失。独立 reviewer 覆盖全部截图、源码契约和浏览器证据，处置为 `ship`，没有必须修正项；该结论只表示提案材料可以交给用户判断，不构成用户确认或生产验收。审阅记录保存 17 个交付物摘要，既有 214 个已确认材料摘要保持不变。

## 验证范围与限制

[浏览器检查](previews/theme-service-status-v16-checks.json)在 1440×900、390×844、320×844，以及 720×900／根字号 200% 四种条件下完成 142 项检查。记录覆盖三来源字段和时间、页签键盘操作、恢复优先级及展开信息、精确零值和缺失值、全部服务和 metric、移动记录布局、导航抽屉、stale／首次失败／加载／空态、字体、横向溢出、浏览器错误、外部请求及抽样对比度；结果没有失败、浏览器错误、外部请求或整页横向溢出，抽样文字对比度均不低于 4.5:1。

Impeccable detector 的 Inter 提示沿用用户已确认的 v2 字体例外。白字／青绿色 hover 提示来自未使用的 primary CSS 与继承文字组合；页面实际没有 primary 按钮，5 个实际 hover 样本的最低对比度为 7.2188:1。两项例外都只限定本 v16 HTML，未放宽全局规则。自身两轮检查修正了缺失 SVG 引用、warning 颜色优先级和文字放大时头像尺寸；之后独立审阅未要求 UI 修正。

[当前 React 采集](previews/theme-service-status-v16-before-capture.json)使用同一批合成事实和截获的 GET fixture，1 个测试通过，用时 2.5 秒；没有写请求。原型没有外部或真实服务请求，未测试实时认证、真实服务健康、生产 10 秒轮询、完整异步失败转移或全栈 smoke。真实独立请求处理、缓存／realm 边界和完整来源错误仍是后续实现与生产验收义务。

## 环境与确认范围

本任务没有启动服务，所有任务浏览器均已关闭。采集借用了既有共享 root Vite：`http://localhost:4000`，PID `308228`，supervisor PID `307846`，仓库为 `/home/yege/work/athena`，进程工作目录为 `/home/yege/work/athena/ui`，实例 `athena-local-runtime`，日志在 `.run/athena-local-runtime/`，当前 React 采集材料在 `.superpowers/v16-current-playwright-artifacts/`。该共享环境按原归属保留，没有执行 `cd /home/yege/work/athena && make stop`。

本轮是既有视觉体系的普通扩展；任务开始前 `DESIGN.md` 与 `.impeccable/design.json` 已缺失，本轮保持这一既有状态并记录该漂移，不越界补建或修复。

用户于 2026-09-14 对上一回复的四张主要截图反馈“舒服 继续推进”，确认其三来源页签、信息层级、桌面／手机布局和视觉效果。具体范围见 [v16 确认记录](previews/theme-service-status-v16-approval.json)；其他辅助状态未逐图确认。原始 HTML、数据、截图、检查与审阅记录均保持不变，原始审阅保留提交时的 `awaiting_user_review` 状态，当前确认以独立确认记录为准。正式 `ui/` 实现、真实请求行为与生产验收均不在本轮范围。

[返回主题需求](visual-theme.md)
