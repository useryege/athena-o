---
version: 1
slug: "ui-previews-theme-service-status-v16-html-d2423ab1"
primary_target: "docs/requirements/web-ui/previews/theme-service-status-v16.html"
related_targets: []
---

# v16 管理员 Service Status 视觉提案

Mode: Operate。用户已授权继续设计既有管理员 Service Status 页；承接已确认视觉系统，只制作 HTML、截图与需求文档，不修改正式 ui/。

THESIS: 先辨认哪一个来源需要关注，再读取该来源的状态与细节。三来源页签始终保留各自状态，选中后显示完整资料。

OWN-WORLD: 沿用 visual-theme.md 的 Nansen 参考、v1 配色、v2 字体、v3 导航尺寸、v4 反馈层级及 v15 管理员导航；不重新选择视觉世界。

STORY: 管理员扫读 Services、Notifications、Trader Sync 的独立状态，查看异常服务、通知恢复与队列，或核对指标单位和时间范围。Refresh 仍针对全部来源，单一来源失败不遮盖其他来源。

FIRST VIEWPORT: 桌面 224px 侧栏、64px 顶栏、32px 内容边距。页头保留 Refresh；下方三来源页签各有状态文字，默认 Services 表格展示服务名、状态和错误。通知页将恢复提示与 System/Account 队列表格放在运行细节之前。Trader Sync 先显示采集器与 raw 可观测性，再给每条 metric 的值、单位、范围。手机 20px 内容边距，三页签并列，服务与指标表改为有分隔线的逐条记录，技术信息可换行。

FORM: 普通扩展、code-led；seed key: not-applicable。已指定现有页面与已确认主题，内容与三来源结构有明确源实现。标志性交互为保留三来源状态的页签切换，键盘方向键可操作；只有约 160ms 颜色反馈，无装饰动画。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## 来源与业务边界

以 ui/src/app/admin/pages/service-status.tsx、shared/services/service-status-service.ts、admin/notification-service.ts、admin/trader-sync-models.ts、administrator-application-shell.md 为准。Services 的 Last checked、Notification 的 Last received、Trader Sync 的 As of 各自保留，全部 UTC+8。三来源现有 10 秒可见 single-flight 与缓存边界不改变；原型不调用服务、不冒充真实轮询。

gRPC 健康值与进程运行状态不合并成全局健康判断。Notification failed/stopped 优先于 recovery；remaining/elapsed 保持精确字符串、0 不当缺失、不倒计时。System/Account 五种队列计数分别展示。Trader raw 缺值显示 Not observable，metric 不跨单位、窗口、service epoch 相加，epoch 不解释成日期。window 示例明确 synthetic。保留全部现有字段，只调整展示层次。

## QUALITY BAR

第一眼可以识别异常与所属来源；信息完整且能核对来源时间。恢复提示、刷新失败、服务不可达采用不同的准确文案，颜色之外有文字。手机不横向滚动，不把宽表缩成微型文字。状态页不增加重启、重发或配置操作。复用管理员导航与已确认的抽屉尺寸；全站单深色，无明暗开关。

## 验证与文档

对照当前 React 同一份虚构数据，GET fixture 仅用于截图，不写服务。检查 1440、390、320、720px/200% 字号，最多两轮自检，之后独立 finish reviewer 与 documenter。最终截图嵌入来源，214 份已批准摘要保持不变。

只维护 service-status-proposal.md、visual-theme.md、previews/README.md、requirements/README.md、design/README.md，状态待确认。保留旧链接与批准范围，不更新 AGENTS.md 或 PRODUCT.md 的批准规则。DESIGN.md 和 .impeccable/design.json 原先缺失，不在普通扩展中修复。借用 root 既有 Vite localhost:4000，结束关闭任务浏览器，保留共享环境。
