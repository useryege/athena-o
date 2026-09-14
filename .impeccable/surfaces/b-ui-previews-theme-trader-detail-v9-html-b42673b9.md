---
version: 1
slug: "b-ui-previews-theme-trader-detail-v9-html-b42673b9"
primary_target: "docs/requirements/web-ui/previews/theme-trader-detail-v9.html"
related_targets: []
---

# Trader Sync 订阅详情视觉提案

模式：Operate。范围为现有 `/trader-sync/subscriptions/:id`，使用同一份明确标注的合成资料对照当前 React 和新主题。沿用用户已确认 v1–v8；不修改正式 `ui/`。

## Direction contract

THESIS：先核对当前监控，再理解历史中断，最后管理订阅。当前状态与旧记录不可混淆，不把停止监控理解为清空旧通知。

OWN-WORLD：延续近黑页面、深色面板、白灰文字、青绿操作；Inter 正文、JetBrains Mono 完整钱包，继承会员侧栏、顶栏与 v4 取消确认弹窗。

STORY：读取交易员身份、当前状态与可靠时间；修改保留备注或暂停/恢复，取消前核对身份与后果；按需展开每条观察记录的原始时间和未知边界，再查看活动与通知。

FIRST VIEWPORT：页头提供返回列表；桌面左侧宽栏先身份与监控、下方历史，右侧340px栏集中备注与操作，再列旧通知。手机按身份、监控、操作、历史、通知阅读。主操作为保存备注，取消为次要危险操作；只有确认弹窗内填充危险色。

FORM：已有详情页在批准世界中的精确扩展，采用代码原型，无新概念抽签，seed不适用。特征交互是观察记录详情展开：摘要保留状态、主要时间、原因与可能遗漏提示，展开保留排序时间、generation/epoch、未知起点和恢复边界；全部数据可核对。状态/焦点即时反馈，减少动态效果受尊重。

FINISH：unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

新增的历史摘要/详情展开以及本页布局待用户通过截图确认；不改变历史恢复、资料精度、权限、修订和幂等规则。原型点击仅显示本地提示或演示弹窗，六态等通过预览控件切换。完成需截图、原型检查、独立审阅和页面提案；作为现有系统扩展不创建缺失的全局 DESIGN.md 或 sidecar。
