# Trader Sync 首页改版（v6 视觉基准）

> 状态：首页整页布局、桌面／手机展示及辅助信息默认折叠已确认。用户查看前后对照后反馈“认可”。本文记录后续正式改版的页面级视觉基准，不表示正式 `ui/` 已改版，也不改变已实现的业务功能、接口或权限。
>
> 视觉基础：沿用已确认的 [v1–v5 主题与组件基准](visual-theme.md)。业务事实继续以[现有 Trader Sync 界面设计](../../design/web-ui/trader-sync-activity-alerts.md)和当前 React 实现为准。

## 对照范围与数据真实性

本轮从钱包样板提炼共用视觉规则后，回到现有代表页面 Trader Sync 首页做“当前版 → 新主题”对照。当前版截图由 `ui-fixtures` 驱动实际 React 组件生成；新主题原型读取同一份演示数据。两侧均包含相同的 3 个目标、3 条活动、3 / 10 配额、Telegram 状态及异常事实，活动形成顺序为 ID `303`、`302`、`301` 倒序。

这些内容是受控演示事实，不是线上账户、真实交易或外部 API 数据。当前版截图用于记录既有组件外观，新原型用于审阅视觉重排；对照不证明生产服务、轮询或真实通知已经运行，也不改写长期设计文档中旧版“已实现”的事实状态。

## 本轮已确认的设计

用户确认桌面活动／目标分区、市场与成交信息的阅读层级、手机默认收起目标区的展示，以及身份快照、Position ID 和来源入口默认收起、按需展开的处理。直接展示的当前桌面、新主题桌面与手机完整截图，以及确认范围和原件摘要见 [v6 确认记录](previews/theme-trader-sync-v6-approval.json)。

其余首屏、目标展开及证据展开截图保留为辅助审阅材料，不据本次反馈宣称其全部细节已逐图确认。完整网络状态、权限变化、交互验收与其他页面继续按各自范围设计和验证。

### 页面结构

- 继续使用 v3 已确认的会员壳层、顶栏、页头和单一深色主题；Telegram 保持独立状态区，不并入目标监控或活动投递结果。
- 桌面采用左侧活动主区与右侧 284px 目标区。视口不超过 1200px 时，目标区折叠为活动区上方的可展开区域；手机继续完整支持目标筛选和状态查看。
- 活动卡片先呈现市场、Outcome、BUY／SELL、成交金额和份额，使用户先读到成交事实。目标身份、完整钱包、公开名称、私有备注和结算时间仍保留。
- 身份快照的查询时间、Position ID，以及 Polymarket Profile、市场和原订阅入口移入 `Identity & position` 展开区。用户已认可这项默认可见性调整；证据和详情入口继续保留，完整钱包与成交事实仍直接可见。

### 状态与事实层级

- 目标监控中断继续在目标区显示原因和最近可靠观察时间。
- Telegram 已发送、站内独有及未知投递分别表达；未知结果保留“可能已收到且不自动重发”的边界。
- Finality anomaly 与投递状态分开显示，不能把链上最终性异常包装成通知失败。
- 完整钱包、公开名称、备注、结算时间、pUSD 成交额和份额均保留；样板只改变视觉层级，不重新计算或改名业务数据。

## 沿用的筛选与分页契约

- 日期筛选继续使用 UTC+8，结束日期包含当天；活动按 owner 内形成 ID 倒序稳定读取。
- Page size 仅提供 50 和 100；分页继续使用 `Previous`／`Next` 及已访问页栈，不展示虚构总数或总页数。
- 目标和日期变化按现有查询契约重建结果，页面仍区分当前选择、空结果、读取失败和新活动提示。

本页面不迁入 Token 钱包样板的完整合约地址搜索、每批 20 条或“加载更多”规则。v5 提供的是共用视觉参考，不替代 Trader Sync 已确认的分页和筛选业务。

## 原型交互边界

[独立 HTML](previews/theme-trader-sync-v6.html)中的日期筛选、目标折叠与选择、活动证据展开、手机目标区和页面壳层可在本地操作。Profile、市场、原订阅、添加目标、通知管理及其他路由按钮只显示本地预览提示，不跳转正式路由。

原型不实现生产 5 秒轮询、异步请求竞争、游标恢复、权限撤销、缓存或任何写操作；不访问外部账户和 API，也不保存筛选、折叠或输入状态。以上生产行为继续由现有业务设计和后续正式改版实现验证约束。

## 审阅产物

以下文件均位于 `docs/requirements/web-ui/previews/`：

- `theme-trader-sync-v6.html`
- `theme-trader-sync-v6-before-desktop.png`
- `theme-trader-sync-v6-before-mobile.png`
- `theme-trader-sync-v6-desktop.png`
- `theme-trader-sync-v6-mobile.png`
- `theme-trader-sync-v6-desktop-first-screen.png`
- `theme-trader-sync-v6-mobile-first-screen.png`
- `theme-trader-sync-v6-mobile-targets.png`
- `theme-trader-sync-v6-evidence.png`
- `theme-trader-sync-v6-checks.json`
- `theme-trader-sync-v6-review.json`
- `theme-trader-sync-v6-data.json`
- `theme-trader-sync-v6-before-capture.json`
- `theme-trader-sync-v6-approval.json`

其中 before 截图记录当前 React 组件，其他截图展示新主题的桌面、手机首屏、目标展开及证据展开状态；data 与 before-capture 保留同事实数据和当前版采集边界。HTML、截图、检查与原始 review 保留提交时状态，其中历史“待确认”标记不改写；当前确认状态以本文和 approval 为准。

## 验证状态

- 独立原型的 Playwright 检查已通过：1440px 桌面、390px 手机、320px 窄屏与 720px 下根字号 200% 场景均无整页横向溢出，字体资源正常加载。
- 上述场景中的初始、目标展开、证据展开及空结果状态，axe WCAG 2 A／AA、2.1 AA 自动检查均未发现违规；日期错误恢复、含结束日筛选、目标选择与清除、证据展开、每页数量和本地路由提示检查通过。
- 没有页面脚本错误或原型外部 HTTP 请求。根字号放大检查不等于跨浏览器原生缩放或完整人工无障碍验收；上述结果也不覆盖正式 API、轮询和生产业务验收。
- 首次检查发现演示数据文件的身份快照时间与当前 React 截图中的统一查询时间不一致，已将三个快照时间对齐为 `2026-09-14T06:42:00Z`，重新生成原型并通过完整检查。当前界面原始截图未修改。
- Impeccable 唯一提示为 `overused-font: inter`；用户已在 v2 确认字体，通过 `hooks ignore-value` 分别登记仅限 v6 HTML 与其 `.superpowers/trader-sync-v6.css` 同源样式的例外，其他规则保持有效。
- 当前界面采集使用的临时 Vite 已停止；原型检查直接读取本地 HTML，检查结束后关闭浏览器。原有共享开发环境保持原样。

原始独立视觉审阅结论为 `ship`，仅表示当时的截图可以提交用户确认。用户随后反馈“认可”，本页所列整页布局和默认折叠现已成为视觉基准；两项记录分别保存，均不表示正式 UI 改版或真实业务验收已经完成。

[返回主题需求](visual-theme.md)
