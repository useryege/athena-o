---
version: 1
slug: "web-ui-previews-theme-trader-add-v7-html-2f02da5c"
primary_target: "docs/requirements/web-ui/previews/theme-trader-add-v7.html"
related_targets: []
---

# Trader Sync 添加页视觉提案

模式：Operate。范围为现有 `/trader-sync/add` 的独立视觉样板，沿用用户已确认的 v1–v6 视觉系统；保持输入、资料查询、身份核对、备注、六周期盈亏和明确确认订阅的业务流程。仅以同一批受控演示数据对照当前 React 界面，不修改正式 `ui/`。

## Direction contract

THESIS：让用户先核准交易员身份与资料，再确认监控订阅。身份、收益资料和订阅状态分组，创建条件保持完整。

OWN-WORLD：使用已确认近黑页面、深灰面板、青绿色操作、Inter 正文与地址专用 JetBrains Mono；继承 v6 的侧栏、顶栏和控件，不创建新品牌。

STORY：输入钱包或资料链接，查看完整钱包、公共名称、验证与资料来源，核对统计及六周期盈亏，填写私有备注，在独立确认区检查通知与配额后确认。

FIRST VIEWPORT：页头下方是完整宽度的输入区；桌面资料区在左，较窄确认区在右，手机按输入、身份、盈亏、确认的阅读顺序纵向排列。主按钮在最终确认区；有结果时查询按钮降为次要层级。

FORM：在既有独立添加页面与已确认视觉世界中调整信息层级，采用代码样板；属于精确页面扩展，无新方向抽签或 seed。保留 `$` 供应商符号说明、资料缺失原因和精确曲线值，不增加交易执行或新的指标。

FINISH：unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

本轮为既有视觉系统扩展，按文档边界保留现状：不因根目录尚无 DESIGN.md 或 sidecar 就创建或更换全局视觉系统。结束需完成桌面/手机截图、原型检查、独立审阅及需求提案；新页面效果待用户确认。
