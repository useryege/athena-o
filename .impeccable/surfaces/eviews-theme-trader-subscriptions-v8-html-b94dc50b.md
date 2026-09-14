---
version: 1
slug: "eviews-theme-trader-subscriptions-v8-html-b94dc50b"
primary_target: "docs/requirements/web-ui/previews/theme-trader-subscriptions-v8.html"
related_targets: []
---

# Trader Sync 订阅管理列表视觉提案

模式：Operate。现有 `/trader-sync/subscriptions` 在已确认 v1–v7 视觉系统中的页面扩展。本轮制作同数据前后对照与独立原型，正式 `ui/` 不改动。

## Direction contract

THESIS：让用户逐行核对交易员、监控与旧通知，而不是从零散状态中拼凑订阅情况。保留 Current / Cancelled 与详情管理入口。

OWN-WORLD：继承已确认近黑背景、深色面板、青绿强调、Inter 正文与 JetBrains Mono 地址；沿用侧栏、顶栏、按钮、反馈色及间距。

STORY：切换当前订阅或取消历史，核对完整钱包与身份、有效时间、最近可靠观察和六类旧通知计数，再进入该订阅详情管理。配额按全部当前状态计数。

FIRST VIEWPORT：页头左侧标题、右侧 Add trader 主操作。下方左侧 Current / Cancelled 控件，右侧 3 / 10 配额；一张宽表分 Trader、Monitoring、Old notifications、Manage 四列。手机每位交易员按身份、监控、通知、详情入口重排，保留完整地址及日期。底部沿用 Previous / Next / Latest subscriptions。

FORM：代码样板，在指定已有列表中扩展已批准视觉规则；无新世界或开放概念抽签，seed 不适用。保留每页50与游标，不加搜索、总页数、行内暂停/取消或新统计。特征交互是 Current / Cancelled 保持各自本地浏览位置；状态和按键反馈克制，内容默认可见、支持减少动态效果。

FINISH：unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

用户已同意按当前对照 → 新版截图 → 审阅推进；本页新视觉仍待确认。保留既有全局文档边界，不因 DESIGN.md 或 sidecar 缺失创建新系统。结束需完成截图、检查、独立审阅和本页需求提案。
