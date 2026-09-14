# Trader Sync 添加交易员页改版（v7 视觉基准）

> 状态：所展示的页面视觉已确认。用户于 2026-09-14 查看新主题桌面／手机完整截图后，对布局、信息密度和操作层级反馈“舒服”。本文记录现有 `/trader-sync/add` 页面的改版视觉基准；正式 `ui/` 尚未改版，已实现的查询、订阅、权限或通知行为继续有效。
>
> 视觉基础：沿用已确认的 [v1–v6 主题、组件与代表页面基准](visual-theme.md)。页面业务继续以[现有 Trader Sync 界面设计](../../design/web-ui/trader-sync-activity-alerts.md)及当前 React 实现为准。

## 对照范围与演示数据

当前版截图由 `ui-fixtures` 驱动实际 `add.tsx`、`confirmation-card.tsx` 与 `pnl-chart.tsx` 生成；v7 原型使用同一份[共享演示数据](previews/theme-trader-add-v7-data.json)。采集边界记录于[当前界面证据](previews/theme-trader-add-v7-before-capture.json)。两侧共享同一个受控钱包、公开资料、备注、配额、Telegram 状态和六周期 P/L 证据，便于只比较页面外观与信息层级。

所有内容均为 fixture，不是真实账户、交易或供应商查询。1Y 的金额和八点曲线明确为视觉对照用虚构数据；其余五个周期保留 `query_failed`，Predictions 保留超过 JavaScript 安全整数范围的原始字符串，Position value 与 Largest win 保留有效零值。样板不能作为真实收益、资料完整性或生产请求成功的证据。

## 本轮已确认的视觉设计

确认依据为直接展示的[桌面完整截图](previews/theme-trader-add-v7-desktop.png)与[手机完整截图](previews/theme-trader-add-v7-mobile.png)，具体范围及原件摘要见 [v7 确认记录](previews/theme-trader-add-v7-approval.json)。当前版桌面／手机以链接提供对照。本次认可覆盖默认核对状态中的资料与 P/L 分区、确认区、阅读顺序和主次操作层级；输入态、过期、缺失和精确曲线值等辅助截图未逐图确认，仍作为检查证据保存。

### 页面结构与操作层级

- 页头下方先展示完整宽度的 `Find trader`，使钱包地址或 Polymarket Profile URL 输入成为清楚的第一步。
- 取得结果后，桌面左侧展示身份资料与 P/L，右侧使用 316px sticky 确认区；视口不超过 1200px 时改为纵向阅读，手机确认区排列在身份与 P/L 核对之后。
- 已展示的核对状态中，`Resolve again` 为次要操作，`Confirm subscription` 为页面主操作。初始输入态以 `Resolve trader` 为主要操作的方案保留在辅助样例中，其页面外观不由本次反馈单独确认。
- 确认区新增“Confirmation valid until”显示，值直接来自既有 `expiresAt`。这是已有确认期限的可见表达，不新增有效期算法、延长期限或后台倒计时规则。
- 已展示的默认 1Y P/L 金额、曲线与六周期控件，以及私有备注、配额和已连接 Telegram 的视觉分层作为本页基准。其余周期结果、精确值展开及异常状态继续按各自范围确认和验证。

### 保持不变的资料与 P/L 语义

- 保留完整钱包、公开名称、明确的 `Not verified`、头像不可用原因，以及 Joined、Position value、Largest win、Predictions 四项资料。
- 私有备注继续限制为最多 20 个 Unicode code point，并保留字符计数和超限错误；备注不改写公开身份。
- P/L 继续提供 1D、1W、1M、1Y、YTD、ALL 六个周期，默认 1Y。正式业务按供应商返回的原始十进制字符串展示；本原型按同样规则读取上述虚构样例。精确值可展开核对和复制，不从图形像素反算数值。
- P/L 与资料金额继续按 Polymarket 的 `$` 展示符号呈现，并明确供应商未提供币种代码；不得擅自标成 USD、pUSD 或 USDC。缺失、有效零值与超大整数分别保留，不互相替代。

## 原型状态与交互边界

默认核对状态展示尚未过期的确认、3 / 10 配额和已连接 Telegram。原型的演示按钮可切换输入、核对资料、已过期、未绑定、配额已满、已有订阅和创建结果未知状态，用于比较同一布局下的反馈层级。

输入校验、P/L 周期切换、精确曲线值展开、备注计数及上述演示状态只在本地数据中工作。确认、恢复创建结果、Notifications、Profile、订阅管理及返回等按钮只显示本地提示，不创建订阅、不访问正式路由或外部账户。

原型不模拟生产后台计时器、真正网络请求、request ID 幂等恢复、owner 失效隔离或草稿跨路由保留。已存在的创建条件和异常恢复契约继续由正式实现负责；本轮检查不能称为完整生产行为验收。

## 审阅产物

以下文件均位于 `docs/requirements/web-ui/previews/`：

- `theme-trader-add-v7.html`
- `theme-trader-add-v7-before-desktop.png`
- `theme-trader-add-v7-before-mobile.png`
- `theme-trader-add-v7-before-input.png`
- `theme-trader-add-v7-desktop.png`
- `theme-trader-add-v7-mobile.png`
- `theme-trader-add-v7-desktop-first-screen.png`
- `theme-trader-add-v7-mobile-first-screen.png`
- `theme-trader-add-v7-curve-values.png`
- `theme-trader-add-v7-unavailable.png`
- `theme-trader-add-v7-desktop-expired.png`
- `theme-trader-add-v7-mobile-expired.png`
- `theme-trader-add-v7-desktop-input.png`
- `theme-trader-add-v7-mobile-input.png`
- `theme-trader-add-v7-checks.json`
- `theme-trader-add-v7-review.json`
- `theme-trader-add-v7-approval.json`
- `theme-trader-add-v7-data.json`
- `theme-trader-add-v7-before-capture.json`

before 截图记录当前 React 页面，其余截图展示 v7 的默认核对、首屏、输入、过期、缺失资料和精确曲线值状态；data 与 before-capture 保存同事实数据及当前版采集边界。原始 HTML、截图、数据、检查与 review 保留提交审阅时的内容和待确认标记；用户后续认可单独记录于 approval，不改写历史审阅材料。

## 验证状态

[浏览器检查](previews/theme-trader-add-v7-checks.json)已覆盖 1440px 桌面、390px 手机、320px 窄屏及 720px 下根字号 200% 场景。记录显示各场景无整页横向溢出、重复 ID、字体加载失败、页面脚本错误或原型外部请求；默认核对、精确值、缺失资料、过期、结果未知和输入状态的 axe 自动检查未发现违规，本地交互检查通过。

这些结果只适用于独立原型和受控 fixture。根字号放大不等于所有浏览器原生缩放，自动检查不替代人工无障碍审阅，也不覆盖正式 API、计时、幂等恢复、权限或草稿生命周期。独立视觉审阅结论为 `ship`：13 张截图可以提交用户查看，没有必须返工的可见问题。该审阅结论仅表示材料具备提交条件；用户随后对直接展示的桌面／手机页面反馈“舒服”，确认范围见上文及独立确认记录。正式 UI 实现与验收尚未开展。

Impeccable 仅提示 `overused-font: inter`。依据用户在 v2 已确认的字体，通过 CLI 登记只针对 v7 HTML 的 Inter 例外，其他规则保持有效。第二轮检查前已修正演示状态切换后提示条残留的问题，并收紧确认区间距；最终截图与检查同属修正后的原型。

当前 React 截图采集使用的临时 Vite（5617）已停止，端口释放，原有共享环境保留。新主题原型直接读取本地 HTML，检查结束后浏览器已关闭，不需要保留运行服务。

[返回主题需求](visual-theme.md)
