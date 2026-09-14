# Trader Sync 订阅列表改版（v8 视觉基准）

> 状态：所展示的页面视觉已确认。用户于 2026-09-14 查看新主题桌面／手机完整截图后，对信息密度、状态区分和手机排版反馈“舒服”。本文记录现有 `/trader-sync/subscriptions` 页面的改版视觉基准；正式 `ui/` 尚未改版，订阅状态、权限、通知与分页契约继续有效。
>
> 视觉基础：沿用已确认的 [v1–v7 主题、组件与代表页面基准](visual-theme.md)。列表业务继续以[现有 Trader Sync 界面设计](../../design/web-ui/trader-sync-activity-alerts.md)及当前 React 实现为准。

## 对照范围与数据真实性

当前版截图由 `ui-fixtures` 驱动实际 `subscriptions.tsx`、`subscription-state.tsx` 和共用表格组件生成；v8 原型使用同一份 [受控数据](previews/theme-trader-subscriptions-v8-data.json)。两侧共享 3 个 Current 订阅、1 个 Cancelled 订阅、3 / 10 配额、身份快照、监控状态和旧通知计数，只用于比较页面布局与信息层级。

所有账户、钱包、资料、时间、计数和队列状态均为合成 fixture，不是线上账户、真实订阅或外部 API 数据。before 截图记录现有 React 页面外观，新原型用于审阅视觉重排；对照不证明生产轮询、游标读取或通知发送已经运行。

## 本轮已确认的展示视觉

确认依据为直接展示的[桌面 Current 完整截图](previews/theme-trader-subscriptions-v8-desktop.png)与[手机 Current 完整截图](previews/theme-trader-subscriptions-v8-mobile.png)，当前版桌面／手机以链接提供对照。确认范围包括四列信息层级、逐交易员的手机分组、监控与旧通知区分，以及所展示的身份、配额和操作入口。具体范围与原件摘要见 [v8 确认记录](previews/theme-trader-subscriptions-v8-approval.json)。Cancelled 列表、空态、加载及失败等辅助截图未逐图确认。

### 桌面表格与手机分组

- 桌面保留一张完整四列表格：`Trader`、`Monitoring`、`Old notifications`、`Manage`。身份、运行事实与管理入口按列对齐，不将一条订阅拆成互不关联的卡片。
- 视口不超过 1050px 时，改为按交易员分组的纵向结构；每组依次展示身份、监控、旧通知和详情入口。完整钱包与日期不因手机宽度被省略。
- 页头保留 `Add trader` 主要入口；`Current` 与 `Cancelled` 继续作为两个独立视图，取消历史不会混入当前订阅列表。

### 保持可见的业务事实

- Trader 保留备注优先的主身份、可用公开名称、完整钱包、Polymarket Profile 入口，以及资料缺失原因和资料查询时间。地址继续使用 JetBrains Mono，并可复制完整原值。
- Monitoring 保留订阅状态、Effective 和 Last reliable observation。中断、暂停与取消状态按各自事实呈现，不把最近可靠时间写成最近交易时间。
- Old notifications 保留 pending、sending、sent、failed、unknown、cancelled 六项计数；存在相应队列说明时，明确已经排队的通知仍可能继续到达。
- 配额继续显示 3 / 10，并说明所有 Current 状态都占用名额；Cancelled 不因出现在历史视图而重新占用当前配额。

`Manage` 仍只提供 `View subscription`，进入现有详情页后再执行备注、暂停、恢复或取消。v8 不在列表行内新增 pause、resume、cancel，也不改变详情页的确认与恢复流程。

## 筛选、分页与本地状态

- `Current`／`Cancelled` 切换只改变当前视图，并保留各自的本地浏览位置；空视图分别说明没有当前订阅或取消记录。
- 正式列表继续使用每页 50 条和游标分页，底部保留 `Previous`、`Next` 与 `Latest subscriptions`，不增加搜索、总页数、虚构总数或 Token 钱包的“加载更多”。
- 当前 fixture 没有上一页或下一页游标，因此原型中的 `Previous`／`Next` 均为禁用状态。`Latest subscriptions` 只在本地恢复该视图的最新 fixture，不发起真实刷新。
- 原型可本地演示视图切换、钱包复制、首次加载、成功空态、首次失败，以及读取失败但保留旧页。失败状态不能伪装成空列表，保留旧页时仍需明确数据可能过期。

原型中的 `Add trader`、`View subscription`、Profile 及其他路由按钮只显示本地提示，不跳转正式路由。原型不模拟生产 5 秒读取、真实游标栈、异步竞争、owner 失效、权限检查、缓存或任何写操作；本轮不能称为完整生产行为验收。

## 原型与审阅材料

- 可操作原型：[theme-trader-subscriptions-v8.html](previews/theme-trader-subscriptions-v8.html)
- 当前 React 对照：[桌面 Current](previews/theme-trader-subscriptions-v8-before-desktop.png)、[手机 Current](previews/theme-trader-subscriptions-v8-before-mobile.png)、[桌面 Cancelled](previews/theme-trader-subscriptions-v8-before-cancelled.png)
- 新版完整页：[桌面 Current](previews/theme-trader-subscriptions-v8-desktop.png)、[手机 Current](previews/theme-trader-subscriptions-v8-mobile.png)、[桌面首屏](previews/theme-trader-subscriptions-v8-desktop-first-screen.png)、[手机首屏](previews/theme-trader-subscriptions-v8-mobile-first-screen.png)
- Cancelled 视图：[桌面](previews/theme-trader-subscriptions-v8-desktop-cancelled.png)、[手机](previews/theme-trader-subscriptions-v8-mobile-cancelled.png)
- 状态截图：[保留旧页](previews/theme-trader-subscriptions-v8-stale.png)、[手机空态](previews/theme-trader-subscriptions-v8-mobile-empty.png)、[首次加载](previews/theme-trader-subscriptions-v8-loading.png)、[手机首次失败](previews/theme-trader-subscriptions-v8-mobile-error.png)
- 可复核记录：[浏览器检查](previews/theme-trader-subscriptions-v8-checks.json)、[同事实数据](previews/theme-trader-subscriptions-v8-data.json)、[当前 React 采集](previews/theme-trader-subscriptions-v8-before-capture.json)、[独立视觉审阅](previews/theme-trader-subscriptions-v8-review.json)
- 用户确认：[v8 确认范围与原件摘要](previews/theme-trader-subscriptions-v8-approval.json)

原始 HTML、截图、数据、检查和独立审阅记录保留提交时的内容与待确认标记。后续用户反馈单独归档于确认记录，避免改写实际审阅版本。

## 验证状态

[第二轮浏览器检查](previews/theme-trader-subscriptions-v8-checks.json)退出码为 0。在 1440×900、390×844、320×844，以及 720×900 且根字号 200% 的四种条件下，分别覆盖 Current、Cancelled、保留旧页、Current 空态、Cancelled 空态、首次加载和首次失败七种状态；记录中 axe 违规、横向溢出、字体加载错误、页面错误和外部 HTTP 请求均为 0。检查还覆盖同 fixture 完整数据、完整钱包复制 payload、六项旧通知计数、视图切换后的滚动恢复、3 / 10 配额、空态与失败区分，以及原型内的 Latest、Retry 和路由提示。复制检查使用测试替换的 clipboard，不代表真实浏览器权限或系统剪贴板集成已经验收。

[当前 React 采集记录](previews/theme-trader-subscriptions-v8-before-capture.json)为 2 项通过；网络仅有 GET fixture，无 mutation。它证明 before 截图与受控事实一致，不证明线上数据或生产写入行为。首轮检查发现 Cancelled 与 Paused 使用相同图标，现已改为独立取消图标并由第二轮复核覆盖。

字体规则检查仅命中 `overused-font:inter`；该 HTML 按用户已确认的 Inter 字体基准作单文件限定保留，其余规则未放宽。临时 Vite 5618 已停止，原有环境保持不变。以上证据只覆盖独立视觉原型和本地交互；精度、真实业务、生产网络、轮询、权限及写操作仍未验收。独立视觉审阅结论为 `ship`：13 张截图可提交用户视觉确认，无必须修正项。用户随后对直接展示的桌面／手机 Current 页面反馈“舒服”，其确认范围见上文及独立确认记录；正式 UI 实现与验收尚未开展。根字号放大也不等同于原生浏览器缩放。

[返回主题需求](visual-theme.md)
