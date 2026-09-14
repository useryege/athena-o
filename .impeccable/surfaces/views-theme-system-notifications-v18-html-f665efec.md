---
version: 1
slug: "views-theme-system-notifications-v18-html-f665efec"
primary_target: "docs/requirements/web-ui/previews/theme-system-notifications-v18.html"
related_targets: []
---

# v18 系统通知列表与详情

Mode: Operate。继续已授权的全站 UI 静态设计，列表与详情、桌面与手机成组展示。已确认的通用规则直接沿用；常规辅助状态由实现者验证，不逐图增加审批。正式 ui/ 本轮不修改。

## Direction contract

THESIS: 管理员从通知记录定位投递问题，进入详情后先读消息与真实结果，再核对时间及标识。

OWN-WORLD: 沿用用户确认的 Nansen 单深色主题、Inter、标识专用 JetBrains Mono、v15 管理员壳及 v4 反馈色。近黑页面、深色面板、白灰文字，青绿色限于主操作与选中。管理员长时间查看密集运行数据，减少大面积亮色。

STORY: 按关键词、状态、目标 chat 筛选，打开记录查看消息；未知结果明确可能已送达且不会自动重发。必要时发送一条独立的测试通知到已配置 test chat，排队成功不等于送达。

FIRST VIEWPORT: 桌面侧栏224、顶栏64、内容32px；列表标题及 Refresh、Test Notification 在上，筛选与五列记录在下。详情先未知警告，再左侧消息、右侧投递记录，五个独立时间字段顺序列出。手机20px边距，列表分隔行，详情按消息、结果、时间阅读。

FORM: 普通扩展、code-led；seed key not-applicable。列表与详情为同一阅读流程，标识默认收起，保留当前测试弹窗。无装饰动效；短颜色反馈，减少运动时关闭。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## 事实与范围

以 ui/src/app/admin/pages/system-notifications.tsx、system-notification-detail.tsx、notification-service.ts 和 docs/design/notifications/system-notification-operations.md 为准。保留关键词、All/pending/sending/sent/failed/unknown/cancelled、All/test/prod、分页、刷新、列表到详情、返回，以及 Topic Label 测试弹窗。现有通用状态契约含 cancelled，系统表无取消记录，不伪造 cancelled 样本或添加取消功能。无新增 severity/source 筛选、重发、删除、真实 Telegram 调用。

详情17个投影字段均可找到；Created、Send authorized、HTTP started、Result recorded、Sent 独立展示，UTC+8，缺失值不推测。HTTP(S)外链才可点击。未知结果的说明保留当前语义。测试提交trim且必填，单次提交中不能重复或关闭；失败保留输入，成功提示 queued 并加入 pending 演示记录。

## QUALITY BAR

列表扫读标题、严重性、目标、投递状态和创建时间；Topic不遗漏。详情正文优先，Unknown与Failed不混淆；支持长原文、窄屏和200%字号。沿用组件与导航，不增加不必要的KPI。全部示例明确为合成数据。

## 验证与落档

当前React列表/详情桌面和手机用相同fixture拦截GET，仅作视觉对比；不提交POST。原型1440、390、320、720px/200%字号检查；7张原型图加4张React图；最多两轮自身截图检查，再独立finish reviewer和documenter。249份既有批准摘要保持不变。新v18仍待用户整体审阅。保留既有DESIGN.md及.impeccable/design.json缺失事实。借用根工作区既有Vite localhost:4000/PID308228；任务只关闭浏览器，不停止共享环境。
