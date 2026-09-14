---
version: 1
slug: "previews-theme-admin-trader-sync-v19-html-16e6b169"
primary_target: "docs/requirements/web-ui/previews/theme-admin-trader-sync-v19.html"
related_targets: []
---

# v19 管理员 Trader Sync 列表与详情

Mode: Operate。与 Profit Sharing 两页组成四页面批次，沿用用户已确认的视觉体系，只设计静态原型、截图与文档，正式 ui/ 不修改。

## Direction contract

THESIS: 管理员按账户与钱包定位订阅，区分订阅生命周期、当前观察覆盖和历史中断，再核对活动与关联投递数量。

OWN-WORLD: 继承 Nansen 单深色、Inter 与标识专用 JetBrains Mono、v15管理员壳、v18记录与详情阅读层级。管理员长时间检查状态，近黑背景和深色面板承载白灰文字，青绿用于操作和选中，语义色单独表示状态。

STORY: 应用账户ID、规范钱包、状态和包含取消的筛选；打开订阅概要，先看所属用户与目标，再看当前观察和历史边界。所有数量来自返回值，关联投递不能跨订阅相加。

FIRST VIEWPORT: 224侧栏、64顶栏、32内容边距；页头Refresh与Service Status，明确管理员概要。列表上方紧凑筛选，下面订阅身份/状态/观察/活动/投递五列，完整钱包可换行。手机20px边距，按条纵排。详情先身份和订阅状态，左侧当前观察与历史中断，右侧精确数量；时间与技术ID按需展开。

FORM: 普通扩展、code-led；seed key not-applicable。不重新选择视觉世界。列表到详情组成同一只读流程；无订阅写操作。字段折叠有明确标题，常规反馈复用既定控件。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## 事实与边界

以 ui/src/app/admin/pages/trader-sync/subscriptions.tsx、subscription-detail.tsx、trader-sync-models.ts、trader-sync-service.ts及长期trader-sync-activity-alerts设计为准。管理员仅安全概要：身份/钱包/生命周期/观察/数量；不含备注、活动正文或逐条Telegram投递，不可Pause/Resume/Cancel/Resend。

过滤区先编辑后Apply；accountID trim，wallet trim+lowercase，精确state及includeCancelled，pageSize固定50，只有Previous/Next已访问游标栈，不虚构全局总数/页数。活动与投递计数保持十进制字符串、零和Unavailable有别，不用Number损失精度。投递含total及6种状态，distinct logical deliveries，不是attempt且跨行不可相加。

asOf与所有时间UTC+8；lastReliableAt表示当前激活已持久化覆盖，不是最后成交时间。历史中断缺失终点显示Unknown，不说明现在仍中断；后续手动激活与旧激活恢复不同，缺失期不回填。生命周期条件显示Paused/Cancelled/Permission disabled。读取失败不伪造监控状态，stale保留上次成功数据并明确标记。

## QUALITY BAR

完整钱包、用户身份、生命周期与观察状态易找到；当前和历史不混淆；活动/投递精确且范围可读。桌面易扫读，手机不靠水平滚动；ID、原因和时间按需展开但内容不丢。导航/复制/筛选/只读详情/游标演示可操作，无真实请求。

## 验证与交付

当前React四图用同批合成GET fixture，借用root localhost4000/PID308228，不调用任何POST，不改ui。原型1440、390、320与720/200%字号，主图4张，辅助3张以内；最多2轮自身合批检查。父任务统一与ProfitSharing进行一次detector、来源嵌入、独立审阅和文档化。265份既有已批准摘要保持不变。普通扩展保留既有DESIGN.md及sidecar缺失，不新增服务，所有任务浏览器关闭，共享环境保留。
