---
version: 1
slug: "nts-web-ui-previews-theme-sports-v21-html-298aca4a"
primary_target: "docs/requirements/web-ui/previews/theme-sports-v21.html"
related_targets: ["ui/src/app/member/pages/sports-live.tsx","ui/src/app/member/pages/sports-history.tsx","ui/src/app/member/pages/world-cup-corners.tsx"]
---

# Sports v21 surface brief

Mode: Operate. Scope: three existing member pages, Sports Live, Sports History, and World Cup Corners. Existing approved Nansen world; production React remains untouched. Source: respective React pages, sports-market-card.tsx and shared sports services, market-intelligence/sports-live.md and sports-history.md.

## Direction contract

THESIS: 让赛事阶段、比赛结果与价格轨迹并列可读；历史回看与角球统计保持各自任务结构。

OWN-WORLD: 沿用批准的近黑、深面板、灰白文字及 mint 操作色，Inter 字体和固定字号；价格曲线使用带文字图例的语义区分色。默认源资料折叠，赛事事实直接显示。

STORY: 会员先识别当前比分和价格或已完赛结果，再检查时间序列；历史同步明确状态与影响；角球页先看90分钟结算口径，再核对样本量、阶段命中比例和单场数据。

FIRST VIEWPORT: 桌面沿用224侧栏和64顶栏；Live/History每个赛事为左侧事实、右侧真实轴图表；手机按事实、价格、图表顺序。角球页为简洁摘要、阶段条形图和完整可筛选比赛记录。刷新操作位于页头。

FORM: code-led ordinary extension. FORM-SEED: not-applicable. 采用既定应用壳和业务记录布局，无新世界或栅格mock。Scratch以逐步揭示价格序列为特征交互，并提供键盘可用的时间滑块。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

QUALITY BAR: 每页1440/390主图、320和根字号200%辅图均无整页横溢；比赛标识完整、价格原始精度和未知值保留；曲线是合成series映射的0–100%价格序列，时间使用UTC+8；角球比例由8场合成样本计算，不冒充64场真实数据；同步只演示不发请求，按权限隐藏写入口。所有截图作者打开。父任务合并一次审阅与文档，当前代理不单独执行reviewer/documenter/detector。
