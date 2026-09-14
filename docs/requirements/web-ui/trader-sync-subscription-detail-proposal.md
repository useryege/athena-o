# Trader Sync 订阅详情改版（v9 视觉基准）

> 状态：所展示的页面视觉及历史默认摘要方式已确认。用户于 2026-09-14 查看桌面／手机完整页和手机取消确认弹窗后反馈“舒服，继续设计”。本文作为现有订阅详情页的视觉基准，正式 `ui/` 尚未改版。
>
> 视觉基础：沿用已确认的 [v1–v8 主题、组件与代表页面基准](visual-theme.md)，是既有视觉体系的普通页面扩展，不创建新的全局视觉世界。订阅状态、权限、修订、幂等、轮询与分页等业务契约继续以[现有 Trader Sync 界面设计](../../design/web-ui/trader-sync-activity-alerts.md)及当前 React 实现为准。

## 对照范围与数据真实性

当前版截图由 `ui-fixtures` 驱动实际 [`subscription-detail.tsx`](../../../ui/src/app/member/pages/trader-sync/subscription-detail.tsx)、[`subscription-state.tsx`](../../../ui/src/app/member/pages/trader-sync/subscription-state.tsx) 和 [`observation-history.tsx`](../../../ui/src/app/member/pages/trader-sync/observation-history.tsx) 生成；v9 原型使用同一份[受控数据](previews/theme-trader-detail-v9-data.json)。两侧共享交易员身份、完整钱包、保留备注、当前监控状态、观察历史和六项旧通知计数，只用于比较页面布局、信息层级和展示交互。

所有账户、钱包、资料、时间、历史和队列计数均为合成 fixture，不是线上账户、真实订阅或外部 API 数据。before 截图记录当前 React 详情页的普通状态；没有可靠的当前版取消弹窗截图，因此不建立旧弹窗对比。新原型中的写入和路由只显示本地提示，不模拟成功保存、成功取消或正式导航。

## 本轮已确认的展示视觉

### 桌面与手机阅读顺序

- 桌面采用左右分区：左侧宽栏依次放置交易员资料、当前监控和观察历史；右侧固定为 340px，依次放置备注与订阅管理、活动与旧通知。当前事实与历史记录分开，不把旧记录当成当前状态。
- 手机改为单列，按交易员资料、当前监控、备注与操作、观察历史、活动与旧通知的顺序阅读。完整钱包、资料查询时间、可靠时间和历史边界不因手机宽度省略。
- 页头保留返回订阅列表入口。首屏先回答“是谁、现在是否可靠、可以做什么”，历史与旧通知随后提供核对材料。

### 交易员与当前监控

- Trader 保留备注优先的主身份、可用公开名称、完整钱包与复制原值、Polymarket Profile 入口、资料缺失原因和资料查询时间。地址继续使用 JetBrains Mono，其余英文标题、正文和数字沿用 Inter。
- Current monitoring state 独立显示订阅状态、状态原因、Effective、Last reliable observation、关联中断记录数量和更新时间；暂停、取消与权限停用时继续显示各自的事实时间。
- 辅助状态切换按钮只用于在同一原型中合成展示多种界面状态。暂停和取消模拟时间晚于历史 `asOf`，历史页可能滞后，因此这些截图不能证明历史中的监控区间在截图时仍然有效。

### 观察历史摘要与按需展开

本轮已确认历史记录默认显示摘要，并允许逐条展开原始细节的方式：

- 监控区间摘要保留主要生效区间；展开后显示 `sortAt`、完整起止边界、`generation` 和 `epoch`。
- 中断摘要保留原因、实际开始时间（未知时明确为未知）、恢复时间和可能遗漏提示；展开后显示 `sortAt`、恢复边界、原因与不确定性原文。
- 页面始终说明遗漏活动不会回填，可能遗漏的数量未知；旧记录中的未确认恢复也不代表当前订阅状态。
- 历史读取失败只影响历史区，不改写当前监控状态；空态、最新历史及前后游标入口继续保持独立语义。

### 备注、生命周期操作与取消确认

- 私有钱包备注上限继续按 20 个 Unicode code point 计算。保存备注使用青绿色主要按钮；历史活动中的备注快照保持不变。
- 六种订阅状态沿用当前操作矩阵：

| 订阅状态 | 可用操作 | 必须保留的说明 |
| --- | --- | --- |
| Preparing monitoring | Cancel | 基线尚在准备，不提供暂停或恢复 |
| Monitoring | Pause、Cancel | 当前可暂停监控 |
| Monitoring interrupted | Pause、Cancel | 中断自动恢复，不要求手动 Resume |
| Paused | Resume、Cancel | Resume 会准备新的监控基线 |
| Disabled by access change | Resume、Cancel | 权限恢复不会自动恢复订阅 |
| Cancelled | Subscribe again | 重新订阅需要重新确认 |

- 请求结果未知是写操作恢复状态，不是第七种订阅状态。该状态锁定备注和其他写操作，只保留 `Recover request result`，要求恢复原请求结果后再发起新的变更；修订冲突仍需读取最新状态并重新选择操作。
- 取消入口使用描边危险色；确认弹窗中的 `Confirm cancellation` 使用填充危险色，并保留全部后果：取消不可恢复、释放一个订阅名额、历史和钱包备注继续保留、已排队通知继续处理且之后仍可能到达、再次订阅需要新的确认。弹窗同时保留交易员身份，避免只凭动作名称确认。

### 活动与旧通知

- pending、sending、sent、failed、unknown、cancelled 六项计数各自保留，不合并为总成功或总失败。
- 当前 Telegram 绑定、已排队通知仍可能到达的说明，以及前往活动结果和 Notifications 的入口继续可见。停止当前监控不等于清空旧通知。
- 原型中的入口只显示本地提示；是否有权访问、真实结果读取和路由恢复仍需正式实现验证。

## 原型状态与实现边界

原型覆盖正常监控、历史展开、备注超长、取消弹窗、暂停、中断、取消、准备基线、权限停用、请求结果未知、历史失败、历史空态、首次加载、首次失败和保留旧数据等状态。辅助状态按钮是合成展示工具，不表示这些状态已经由用户逐项确认，也不新增业务状态或接口契约。

正式实现仍需验证真实权限与 owner 边界、订阅及备注修订、幂等 request ID、未知结果恢复、5 秒可见页轮询、历史游标分页、异步竞争和生产网络行为。视觉原型没有发出外部 HTTP 请求或 mutation，不能作为这些契约的验收证据。

## 原型与审阅材料

- 可操作原型：[theme-trader-detail-v9.html](previews/theme-trader-detail-v9.html)
- 当前 React 对照：[桌面普通详情](previews/theme-trader-detail-v9-before-desktop.png)、[手机普通详情](previews/theme-trader-detail-v9-before-mobile.png)
- 新版完整页：[桌面](previews/theme-trader-detail-v9-desktop.png)、[手机](previews/theme-trader-detail-v9-mobile.png)、[桌面首屏](previews/theme-trader-detail-v9-desktop-first-screen.png)、[手机首屏](previews/theme-trader-detail-v9-mobile-first-screen.png)
- 关键交互：[历史展开](previews/theme-trader-detail-v9-history-expanded.png)、[桌面取消确认](previews/theme-trader-detail-v9-desktop-cancel.png)、[手机取消确认](previews/theme-trader-detail-v9-mobile-cancel.png)
- 生命周期与恢复：[手机暂停](previews/theme-trader-detail-v9-mobile-paused.png)、[已取消](previews/theme-trader-detail-v9-cancelled.png)、[手机请求结果未知](previews/theme-trader-detail-v9-mobile-unknown.png)
- 读取状态：[历史失败](previews/theme-trader-detail-v9-history-error.png)、[首次加载](previews/theme-trader-detail-v9-loading.png)、[手机首次失败](previews/theme-trader-detail-v9-mobile-error.png)、[保留旧数据](previews/theme-trader-detail-v9-stale.png)
- 适配检查：[320px 窄屏](previews/theme-trader-detail-v9-narrow.png)、[根字号 200%](previews/theme-trader-detail-v9-text-200.png)
- 可复核记录：[浏览器检查](previews/theme-trader-detail-v9-checks.json)、[同事实数据](previews/theme-trader-detail-v9-data.json)、[当前 React 采集](previews/theme-trader-detail-v9-before-capture.json)、[独立视觉审阅](previews/theme-trader-detail-v9-review.json)

原始 HTML、截图、数据、检查和独立审阅记录保留提交时的内容与待确认标记。后续用户反馈应另行记录确认范围，不能改写本次审阅原件或把未展示状态一并视为批准。

## 验证状态

[第二轮浏览器检查](previews/theme-trader-detail-v9-checks.json)退出码为 0。在 1440×900、390×844、320×844，以及 720×900 且根字号 200% 的四种条件下，覆盖上述页面、交互与辅助状态；各状态的 axe 违规、页面错误、外部请求和横向溢出均为 0。首轮发现的取消弹窗焦点循环已补齐，第二轮复核覆盖连续 Tab 切换、Esc 关闭和关闭后焦点返回。

[当前 React 采集记录](previews/theme-trader-detail-v9-before-capture.json)为 1 项通过，测试内部循环桌面和手机两种尺寸，用时 2.3 秒；网络只有 GET fixture，无 mutation。独立 reviewer 已逐张查看 16 张新主题截图和 2 张当前 React 截图，结论为 `ship`，无必须修正项，即材料可提交用户视觉审阅。

这些证据仍有边界：根字号 200% 是原型的根字号放大，不是原生浏览器缩放；钱包复制由测试替换 clipboard，只校验完整 payload；自动 axe 检查不等于全量无障碍验收。字体检测只对 v9 HTML 中已确认的 Inter 用法登记单文件例外，其他规则未放宽。临时 Vite 5619 已停止，原有共享环境保持不变。

用户随后确认了直接展示的桌面／手机正常详情、手机取消弹窗，以及历史默认摘要／按需展开方式，具体范围见 [v9 确认记录](previews/theme-trader-detail-v9-approval.json)。桌面取消弹窗、历史展开细节和其他辅助状态未逐图确认；正式 `ui/`、生产权限、轮询、分页及写操作尚未改版或验收。

[返回主题需求](visual-theme.md)
