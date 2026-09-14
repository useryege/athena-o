# 管理员业务页面集中视觉基准（v19）

> 状态：四个实际管理员页面所展示的桌面／手机视觉已确认；正式 `ui/` 未修改。范围见 [v19 确认记录](previews/theme-admin-batch-v19-approval.json)。
>
> 页面范围：Trader Sync 订阅列表、订阅概要、Profit Sharing 轮次列表、轮次详情。每页桌面与手机各一张主要截图；同一页面的辅助状态不计作新增页面，也不增加逐图审批。

关联现状：[Trader Sync 列表](../../../ui/src/app/admin/pages/trader-sync/subscriptions.tsx)、[Trader Sync 概要](../../../ui/src/app/admin/pages/trader-sync/subscription-detail.tsx)、[Trader Sync 模型](../../../ui/src/app/admin/trader-sync-models.ts)、[Trader Sync 服务](../../../ui/src/app/admin/trader-sync-service.ts)、[Trader Sync 长期设计](../../design/web-ui/trader-sync-activity-alerts.md)、[Profit Sharing 页面](../../../ui/src/app/admin/pages/profit-sharing-admin.tsx)、[管理员 Profit Sharing 服务](../../../ui/src/app/admin/profit-sharing-service.ts)、[共享 Profit Sharing 服务](../../../ui/src/app/shared/services/profit-sharing-service.ts)、[共享 Profit Sharing 页面逻辑](../../../ui/src/app/shared/pages/profit-sharing-shared.tsx)和[Profit Sharing 长期设计](../../design/governance/profit-sharing.md)。

## 共用视觉与集中审阅方式

四页沿用已确认的 Nansen 单一深色主题：页面 `#06080B`、面板 `#0F1114`、抬升层 `#181D22`、分隔线 `#252A30`，白／灰文字与有限青绿强调，状态继续使用独立语义色。Inter 用于界面文字和数字；JetBrains Mono 只用于钱包、ID、slug、cursor 与技术原文。

管理员壳保持 224px 桌面侧栏、64px 顶栏、32px 桌面内容边距和 20px 手机边距，控件高度 44px；手机抽屉为 280px 且最大宽度为视口减 48px。手机弹窗按钮依次为 Cancel、主要操作，两者全宽并相隔 12px。

用户已确认 v18，并要求每批提供多个实际页面集中审阅。v19 因此将两个业务的列表与详情一次交付，共八张主图。加载、空态、错误、窄屏、文字放大及交互状态继续作为实现证据覆盖，不作为新的逐图批准关卡。

## Trader Sync 订阅列表

列表用五列组合完整信息：用户与钱包、订阅状态、当前观察、活动、关联投递。完整钱包地址直接可读并可复制；subscription ID、account ID、生命周期细节和详细数量默认折叠。手机按订阅逐条纵向排列，不依赖水平滚动。

筛选保留 account ID、canonical wallet、All 加六种精确状态和 `includeCancelled`。输入先形成草稿，只有 `Apply` 后生效；account ID trim，钱包 trim 后转小写。Apply 重置不透明 cursor。分页固定每页 50，只提供 Previous／Next 及已访问 cursor 栈，不编造总页数或全局总数。

列表返回时保留本地已应用筛选是本原型的体验提案，不能写成当前 React 已完整实现。筛选和 cursor 导航是合成本地演示，不证明远端语义。

## Trader Sync 订阅概要

概要先显示用户、目标钱包与订阅生命周期，再区分当前观察覆盖和历史中断；精确活动与关联投递数量独立展示。`lastReliableAt` 表示本次激活已持久化覆盖，不是最后成交时间。历史中断缺失 `end` 或 `recoveredAt` 时显示 `Unknown`，不能据此判断当前仍中断；后续手动激活与旧激活恢复分开，缺失期不回填。

活动计数是 lifetime。关联投递是 distinct logical messages，不是发送 attempt；不能跨订阅相加。所有大计数按返回的十进制字符串保真，`Unavailable` 与 `0` 明确不同。生命周期条件分别表达 Paused、Cancelled 和 Permission disabled；读取失败不能伪造监控状态，已有成功数据变旧时保留并标记 stale。

管理员范围只读，只提供身份、钱包、生命周期、观察和汇总数量。页面不显示私密笔记、活动正文或逐条投递，也不提供 Pause、Resume、Cancel 或 Resend。

## Profit Sharing 轮次列表

列表保留 `Round`（title + slug）、`Phase`、`Participants`、`Submitted`、`Voted`。阶段依次为 Draft、Collecting、Voting、Closed；只有 Voting／Closed 显示投票进度。轮次名称使用原生键盘可访问控件进入详情。

列表用于判断轮次所处治理阶段和下一步条件，不把治理分配比例描述成美元盈亏或已完成结算。所有示例均为合成治理数据。

## Profit Sharing 轮次详情与 roster

主要详情展示 Collecting 轮次达到 5/5 submissions 后可以 `Publish proposals`，但发布前内容仍 sealed，管理员只能查看状态。发布后才公开匿名 proposal 并进入 Voting；作者和票数保持隐藏。只有全员投票后才能 close；平票会对最高平票 proposal 新开匿名 runoff，直到产生唯一赢家。

Draft 创建允许 0–5 名参与人，title 必填且最多 120 字符，slug 使用小写字母、数字与单连字符。已有 Draft 的 slug 只读，标题与 roster 可编辑。Create 与 Draft 都能添加、移除并从 eligible accounts 选择账户；display name／username 身份快照只读，baseline responsibility 可编辑且必填，同时校验空字段、重复账户和最多五人。空 roster 可以保存 Draft；只有保存后恰好五名不同、非管理员、可登录且拥有 Profit Sharing 权限的参与人，才能打开 Collecting。打开后 roster 锁定。

更新定义以及打开、发布、关闭操作携带 expectedRevision；创建不携带预期版本。409 冲突要求丢弃本地变更并刷新当前轮次，其他错误保留当前状态和服务端错误。这是既有业务契约，不代表真实并发已验收。创建、保存、打开、发布、关闭、确认和导航在原型中都只改变本地展示，不发出写请求。

Profit Sharing 是沿用既定视觉体系的普通 code-led 扩展，`FORM-SEED: not-applicable`。最终 create-mobile 使用真实 390×844 视口：文档保持顶部，弹窗正文滚动 206px 到参与人编辑区，以同屏展示账户选择、只读身份、职责、Remove／Add 和固定操作栏；该截图不是弹窗正文顶部取景。

## 原型、截图与审阅材料

### Trader Sync

- 原型与数据：[HTML](previews/theme-admin-trader-sync-v19.html)、[合成数据](previews/theme-admin-trader-sync-v19-data.json)
- 主要审阅图：[列表桌面](previews/theme-admin-trader-sync-v19-list-desktop.png)、[概要桌面](previews/theme-admin-trader-sync-v19-detail-desktop.png)、[列表手机](previews/theme-admin-trader-sync-v19-list-mobile.png)、[概要手机](previews/theme-admin-trader-sync-v19-detail-mobile.png)
- 辅助提案图：[身份展开手机](previews/theme-admin-trader-sync-v19-identity-mobile.png)、[320px](previews/theme-admin-trader-sync-v19-narrow.png)、[200% 文字](previews/theme-admin-trader-sync-v19-zoom.png)
- 当前 React 对照：[列表桌面](previews/theme-admin-trader-sync-v19-before-list-desktop.png)、[概要桌面](previews/theme-admin-trader-sync-v19-before-detail-desktop.png)、[列表手机](previews/theme-admin-trader-sync-v19-before-list-mobile.png)、[概要手机](previews/theme-admin-trader-sync-v19-before-detail-mobile.png)
- 验证材料：[浏览器检查](previews/theme-admin-trader-sync-v19-checks.json)、[当前 React 采集](previews/theme-admin-trader-sync-v19-before-capture.json)

### Profit Sharing

- 原型与数据：[HTML](previews/theme-admin-profit-sharing-v19.html)、[合成数据](previews/theme-admin-profit-sharing-v19-data.json)
- 主要审阅图：[列表桌面](previews/theme-admin-profit-sharing-v19-list-desktop.png)、[详情桌面](previews/theme-admin-profit-sharing-v19-detail-desktop.png)、[列表手机](previews/theme-admin-profit-sharing-v19-list-mobile.png)、[详情手机](previews/theme-admin-profit-sharing-v19-detail-mobile.png)
- 辅助提案图：[创建弹窗手机](previews/theme-admin-profit-sharing-v19-create-mobile.png)、[320px](previews/theme-admin-profit-sharing-v19-narrow.png)、[200% 文字](previews/theme-admin-profit-sharing-v19-zoom.png)
- 当前 React 对照：[列表桌面](previews/theme-admin-profit-sharing-v19-before-list-desktop.png)、[详情桌面](previews/theme-admin-profit-sharing-v19-before-detail-desktop.png)、[列表手机](previews/theme-admin-profit-sharing-v19-before-list-mobile.png)、[详情手机](previews/theme-admin-profit-sharing-v19-before-detail-mobile.png)
- 验证材料：[浏览器检查](previews/theme-admin-profit-sharing-v19-checks.json)、[当前 React 采集](previews/theme-admin-profit-sharing-v19-before-capture.json)

统一[独立审阅清单](previews/theme-admin-batch-v19-review.json)包含原文五部分和 30 个资产摘要。共 22 张 Chromium 截图，其中 14 张提案、8 张当前 React 对照；全部嵌入来源，并与 `.impeccable/review/v19/` 对应截图逐字节一致，来源扫描为 22 张、0 缺失。既有 265 个已批准资产摘要保持不变。

独立 reviewer 初次处置为 `fix`，随后一个明确修正批补齐 Profit Sharing 的 seed 来源及 Create／Draft 0–5 人 roster、校验和 exact-five Open 门槛；最终处置为 `ship`，没有遗留的静态提案修正项。因新建 reviewer 受 thread limit 限制，复用了未参与 v19 实现的 v10 reviewer。`ship` 本身只表示本批材料可以交用户集中判断；用户确认另见独立记录，正式实现与生产验收仍未由此完成。

## 验证范围与限制

Trader Sync 在 1440×900、390×844、320×844、720×900／根字号 200% 四种条件下通过 202 项本地断言。Profit Sharing 的七种截图条件覆盖相同四类尺寸，并通过键盘轮次入口、Draft 与 0–5 roster、阶段门槛、匿名、确认、冲突及本地错误提示检查。两者都没有整页横向溢出、JavaScript 错误或外部请求。

Trader Sync 自身截图检查 1 轮。Profit Sharing 自身审美检查 2 轮，修正手机弹窗正文滚动与固定操作可见性、放大文字时页头纵向排列；随后针对键盘入口和 Draft 事实更新脚本并核对资产，没有开启第三轮自主 polish。独立 reviewer 另有上述 1 个修正批。

两份当前 React 各有 1 项 Playwright 通过：Trader Sync 总计 3.4 秒、用例 2.8 秒；Profit Sharing 总计 3.5 秒。全部只截获合成 GET，没有写请求。验证不覆盖真实授权、后端读写、eligibility、runoff、409 竞争、异步竞态或 full-stack smoke。

合并 detector 单次只报告两份 HTML 的 Inter 与主按钮 hover。Inter 是已确认品牌规则；20 个目标渲染样本的 hover 均为 `#06080B` 字／`#51FFC3` 底，最低对比度 15.70:1。两份 HTML 各自仅持久忽略这两项，没有全局关闭规则或二次扫描；临时证据 `.superpowers/v19-detector-triage.json` 不是长期契约。

## 环境与确认范围

本批没有启动服务，任务浏览器均已关闭。采集借用既有共享 root Vite：`http://localhost:4000`，PID `308228`，supervisor PID `307846`，仓库 `/home/yege/work/athena`，进程工作目录 `/home/yege/work/athena/ui`，实例 `athena-local-runtime`，日志位于 `.run/athena-local-runtime/`。该环境按原归属保留，没有执行 `cd /home/yege/work/athena && make stop`。

用户于 2026-09-14 明确反馈“没有需要调整的地方，确认通过。”，v19 四个页面所展示的八张桌面／手机主图已确认，范围见 [v19 确认记录](previews/theme-admin-batch-v19-approval.json)。原始审阅保留提交时的 `awaiting_user_review` 状态，当前确认以独立确认记录为准。辅助状态是实现证据，不自动成为已批准截图，也不新增逐图审批。本轮是既有视觉体系的普通扩展；任务开始前 `DESIGN.md` 与 `.impeccable/design.json` 已缺失，本轮按边界保持其缺失并记录既有漂移。正式 `ui/` 未修改。

[返回主题需求](visual-theme.md)
