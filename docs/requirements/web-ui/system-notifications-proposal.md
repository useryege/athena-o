# 管理员系统通知列表与详情视觉基准（v18）

> 状态：所展示列表与详情的桌面／手机视觉已确认；正式 `ui/` 未修改。范围见 [v18 确认记录](previews/theme-system-notifications-v18-approval.json)。
>
> 视觉基础：沿用已确认的 Nansen 单一深色主题、Inter／JetBrains Mono、v4 反馈色与 v15 管理员应用壳。本提案不改变通知服务、权限、投递或数据契约。

关联现状：[当前列表](../../../ui/src/app/admin/pages/system-notifications.tsx)、[当前详情](../../../ui/src/app/admin/pages/system-notification-detail.tsx)、[通知客户端](../../../ui/src/app/admin/notification-service.ts)、[分页约定](../../../ui/src/app/shared/pagination.ts)和[系统通知操作设计](../../design/notifications/system-notification-operations.md)。

## 目标与现状差距

管理员应先在通知记录中定位投递问题，再进入详情优先阅读消息与真实结果，最后核对时间和技术标识。v18 将现有列表的七个字段压缩为五个桌面列，将现有详情的单一键值区重组为消息、投递记录、时间和按需查看的标识，同时保留全部业务事实。

页面继续使用 224px 桌面侧栏、64px 顶栏和 32px 内容边距；手机使用 20px 内容边距，导航抽屉为 280px 且最大宽度为视口减 48px。近黑页面、深色面板、白／灰文字和克制的青绿色操作／选中状态承接已确认主题；成功、警告、失败使用独立语义色。Inter 用于标题、正文、控件和数字，JetBrains Mono 用于投递标识、source、provider 值及原始错误。

## 列表与筛选

页头保留 `Refresh` 与 `Test Notification`。筛选保留 keyword；状态保留 All、pending、sending、sent、failed、unknown、cancelled；目标 chat 保留 All、test、prod。系统通知表实际没有 cancelled 状态，但公共界面筛选词汇仍包含它；样例不伪造 cancelled 系统记录，也不增加取消操作。

桌面把现有七个字段组织为五列：`Title + Topic`、`Severity`、`Chat + Channel`、`Status`、`Created`。Topic、chat 和 channel 仍可直接读取，合并列不丢失字段。手机将每条记录重排为有分隔线的纵向内容，依次保留标题／Topic、严重性、投递状态、目标／channel 和 UTC+8 创建时间，不缩小成微型宽表。

分页继续使用实际默认值 50，并提供 10、50、100 三个 page size 选项。本地原型中的关键词筛选、状态／chat 选择、分页和返回列表时保留筛选，只演示本地交互；其中筛选保留是新提案，不能声称当前 React 已完整实现，也不能把本地演示写成后端模糊搜索字段或远端分页契约。

## 详情、时间与投递语义

详情按消息、投递记录、时间组织，并保留全部 17 个投影字段：id、source、severity、topicLabel、title、body、link、channel、status、telegramChat、providerMessageId、errorMessage、createdAt、authorizedAt、startedAt、resultAt、sentAt。正文和真实投递结果优先；技术标识默认收起，需要时再展开核对。

五个时间按 `Created`、`Send authorized`、`HTTP started`、`Result recorded`、`Sent` 的顺序独立展示，统一使用 UTC+8。缺失时间显示 `—`，不能从其他时间推测。`Provider result` 区域保留 errorMessage 原文，直接可读。只有 HTTP(S) 链接可以点击，其他协议或无效值保持纯文本。

`unknown` 与 `failed` 必须区分。未知结果显示原意完整的说明：`Telegram may have received this message. It will not be resent automatically.`，不能暗示一定失败、一定送达或会自动重发。页面不增加重发、取消或删除投递操作。

## Test Notification

测试弹窗只保留唯一输入 `Topic Label`，提交前 trim 且必填。目标是固定配置的 test chat；测试记录使用 `source=admin-ui` 和 `severity=info`。提交 pending 时锁定输入、重复提交、Cancel、关闭按钮和 Escape。失败保留输入供修正；成功提示 queued，并在本地样例中加入 pending 记录。排队成功不等于 Telegram 已送达，也不重发当前选中的记录。

静态原型只使用本地定时器切换状态，不发送 POST。正式测试入口可能调用 Telegram 准备 Topic 并发送真实测试通知，本轮没有执行，也没有证明真实服务、权限或投递行为。

## 原型、截图与审阅材料

- 可操作原型：[theme-system-notifications-v18.html](previews/theme-system-notifications-v18.html)
- 合成数据：[theme-system-notifications-v18-data.json](previews/theme-system-notifications-v18-data.json)
- 主要审阅图：[列表桌面](previews/theme-system-notifications-v18-list-desktop.png)、[详情桌面](previews/theme-system-notifications-v18-detail-desktop.png)、[列表手机](previews/theme-system-notifications-v18-list-mobile.png)、[详情手机](previews/theme-system-notifications-v18-detail-mobile.png)
- 辅助提案图：[测试弹窗手机](previews/theme-system-notifications-v18-test-mobile.png)、[320px 详情](previews/theme-system-notifications-v18-narrow.png)、[200% 文字列表](previews/theme-system-notifications-v18-zoom.png)
- 当前 React 对照：[列表桌面](previews/theme-system-notifications-v18-before-list-desktop.png)、[详情桌面](previews/theme-system-notifications-v18-before-detail-desktop.png)、[列表手机](previews/theme-system-notifications-v18-before-list-mobile.png)、[详情手机](previews/theme-system-notifications-v18-before-detail-mobile.png)
- 可复核材料：[浏览器检查](previews/theme-system-notifications-v18-checks.json)、[当前 React 采集](previews/theme-system-notifications-v18-before-capture.json)、[独立审阅](previews/theme-system-notifications-v18-review.json)

共 11 张 Chromium 截图，其中 7 张是提案、4 张是当前 React 对照；全部嵌入来源，并与 `.impeccable/review/v18/` 中对应截图逐字节一致，来源扫描为 11 张、0 缺失。独立 finish reviewer 最终处置为 `ship`，五个审阅部分均完成且没有静态提案范围内的必须修正项。因新建 reviewer 受 task thread 上限阻止，父任务复用了未参与 v18 设计的 v10 documenter 完成独立复核；这一替代不构成用户确认、正式实现或生产验收。

## 验证范围与限制

[浏览器检查](previews/theme-system-notifications-v18-checks.json)覆盖 1440×900、390×844、320×844 和 720×900／根字号 200% 四种条件，194 项本地样板断言通过；没有整页横向溢出、JavaScript 错误或外部请求。两轮自身截图检查没有产生视觉修补；检查脚本曾因已关闭弹窗中的旧 callout 导致选择器歧义，限定为当前页面 callout 后通过。

Impeccable detector 单次报告 Inter、主按钮 hover 对比度和弹窗底部留白。Inter 是既有明确选择；实际 hover 为近黑字／青绿底，对比度 15.70:1；弹窗子操作区底部留白为 20px，左右为 20／24px。三个忽略都只通过 CLI 限定在本 v18 HTML，没有全局放宽规则；临时证据 `.superpowers/v18-detector-triage.json` 只记录本次判断，不是长期契约。

[当前 React 采集](previews/theme-system-notifications-v18-before-capture.json)使用合成 GET fixture，1 项 Playwright 测试通过（case 2.5 秒，总计 3.1 秒），没有 POST。真实身份权限、请求取消、防陈旧、异步状态、Telegram 投递和 full-stack smoke 均未验证。

## 环境与确认方式

本任务没有启动服务，所有任务浏览器均已关闭。采集借用既有共享 root Vite：`http://localhost:4000`，PID `308228`，supervisor PID `307846`，仓库 `/home/yege/work/athena`，进程工作目录 `/home/yege/work/athena/ui`，实例 `athena-local-runtime`，日志位于 `.run/athena-local-runtime/`。该环境按原归属保留，没有执行 `cd /home/yege/work/athena && make stop`。

用户于 2026-09-14 对上一回复展示的四张主图反馈“舒服”，所展示范围已确认，详见 [v18 确认记录](previews/theme-system-notifications-v18-approval.json)。原始审阅保持提交时的 awaiting_user_review 状态，当前确认以独立记录为准。已确认的主题、字体、导航与通用组件直接沿用；常规加载、空态、错误、长内容、窄屏、字号放大、键盘和弹窗状态由实现者覆盖与验证，不默认变成逐图审批。新的信息结构和关键操作差异仍需呈现；后续每次先完成多个页面再集中交付用户审阅。辅助图没有因此补写成逐图确认。

本轮是既有视觉体系的普通扩展。任务开始前 `DESIGN.md` 与 `.impeccable/design.json` 已缺失，本轮按边界保持其缺失并记录既有漂移，不补建或修复。正式 `ui/` 未修改。

[返回主题需求](visual-theme.md)
