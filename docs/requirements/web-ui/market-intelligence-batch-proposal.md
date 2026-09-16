# v21 市场、赛事与 Managed OO 八页集中视觉提案

> 范围更新（2026-09-15）：Sports Live、Sports History 与 World Cup Corners 已确认删除；本批 Market Radar、Managed OO 与共用视觉规则继续有效。[删除尚未实施](../development-runtime/sports-removal.md)；下文批准记录、原覆盖清单及历史验收证据保留当时事实。

> 状态：本批展示视觉已确认。用户于 2026-09-14 对集中展示的八页桌面／手机十六张主图反馈“确认”，整批确认范围见 [v21 确认记录](previews/theme-market-intelligence-v21-approval.json)。交付时的审阅记录保持原字节；本次确认不表示正式前端已实施或真实业务验收通过。

## 范围与沿用基准

按[其余页面安排](remaining-pages-plan.md)一次交付八页，桌面／手机合并审阅。第一批 [v20 四页](member-foundations-batch-proposal.md)已整批确认，其[审批资产](previews/theme-member-foundations-v20-approval.json)保持原字节。本批沿用已确认 Nansen 近黑背景、深色面板、白灰文字、mint 强调及细边框关系，不新增共用视觉系统决定。Trader Sync 本轮排除；独立已确认的 [Nansen 钱包战绩 v1](../token/wallet-analytics-page-proposal.md)直接沿用。

三份成品共同使用页面 `#06080B`、面板 `#0F1114`、抬升面 `#181D22`、边框 `#252A30`、正文 `#FFFFFF`、辅助文字 `#9FA0A1` 和强调 `#00FFA7`。Inter 用于标题、正文及数字，JetBrains Mono 用于完整地址、哈希和技术标识；沿用 28／24px 页面标题、20px 节标题、16px 正文、14px 控件、13px 辅助层级、224px 侧栏、64px 顶栏及至少 44×44px 操作目标。颜色、字体和空间来自[视觉主题](visual-theme.md)、[字体基准](typography-proposal.md)、[布局基准](layout-proposal.md)及成品嵌入 CSS；本页布局是既定规则在业务上的扩展，本批主图所展示视觉已获用户整批确认。

## 八页主图与路由

每页提供两张主图，共 16 张。链接中的 `view` 选择静态原型页面；实际应用路由只作为后续实施对照。

| 实际页面 | 当前应用路由 | 可操作原型 | 桌面主图 | 手机主图 |
| --- | --- | --- | --- | --- |
| Market Radar 热门市场 | `/market-radar` | [打开](previews/theme-market-radar-v21.html?view=hot) | [桌面](previews/theme-market-radar-v21-hot-desktop.png) | [手机](previews/theme-market-radar-v21-hot-mobile.png) |
| Market Radar 实时市场 | `/market-radar/realtime` | [打开](previews/theme-market-radar-v21.html?view=realtime) | [桌面](previews/theme-market-radar-v21-realtime-desktop.png) | [手机](previews/theme-market-radar-v21-realtime-mobile.png) |
| Market Radar 涨跌榜 | `/market-radar/movers` | [打开](previews/theme-market-radar-v21.html?view=movers) | [桌面](previews/theme-market-radar-v21-movers-desktop.png) | [手机](previews/theme-market-radar-v21-movers-mobile.png) |
| Sports Live | `/sports-live` | [打开](previews/theme-sports-v21.html?view=live) | [桌面](previews/theme-sports-v21-live-desktop.png) | [手机](previews/theme-sports-v21-live-mobile.png) |
| Sports History | `/sports-history` | [打开](previews/theme-sports-v21.html?view=history) | [桌面](previews/theme-sports-v21-history-desktop.png) | [手机](previews/theme-sports-v21-history-mobile.png) |
| World Cup Corners | `/world-cup-corners` | [打开](previews/theme-sports-v21.html?view=corners) | [桌面](previews/theme-sports-v21-corners-desktop.png) | [手机](previews/theme-sports-v21-corners-mobile.png) |
| Managed OO 提案列表 | `/managed-oo/proposals` | [打开](previews/theme-managed-oo-v21.html?view=proposals) | [桌面](previews/theme-managed-oo-v21-proposals-desktop.png) | [手机](previews/theme-managed-oo-v21-proposals-mobile.png) |
| Managed OO 争议列表 | `/managed-oo/disputes` | [打开](previews/theme-managed-oo-v21.html?view=disputes) | [桌面](previews/theme-managed-oo-v21-disputes-desktop.png) | [手机](previews/theme-managed-oo-v21-disputes-mobile.png) |

## 布局、事实与操作差异

### Market Radar 三页

热门市场先比较 24h 成交量、流动性等美元指标与 spread；实时市场先读各 outcome 当前价格，再读 1m／5m／15m 的百分点变化；涨跌榜突出 leader、方向和加权 score，保留 1／0.6／0.3 窗口权重说明。价格、百分点与分数不混作收益或同一单位。桌面横向对齐记录，手机以市场标题开头，随后分组列出关键指标；完整 condition／token 标识及来源默认折叠、可展开复制。

快照栏分别表达 fetched、stale、sampling 连接、候选／监控数量；warmup 表示窗口尚未形成，Unknown 不替换为零。时间使用 UTC+8。三页均只读；无权示例移除市场数据，刷新仅重载本地合成快照且保留分页。沿用默认 50、可选 10／50／100 条及页码夹紧；本地 60 条扩展样本检查了翻页、改页大小和刷新保页。

来源：[当前 React](../../../ui/src/app/member/pages/market-radar.tsx)、[前端服务](../../../ui/src/app/shared/services/market-radar-service.ts)、[长期 Market Radar 设计](../../design/market-intelligence/market-radar.md)。

### Sports Live、History 与 World Cup Corners

Live／History 桌面把比分、阶段、快照价格放在左侧，右侧展示有真实时间轴的合成价格序列；手机按赛事事实、价格、图表顺序阅读，来源标识默认折叠。Live 的三 moneyline 结果分别读取其 Yes 价格，ATP 使用两个 outcome；例如 0.615 对应图表 61.5%，图表纵轴明确为 0–100%，不把原始价格或比赛得分改作另一种含义。来源折叠行最终使用一个 SVG 指示，收起向右、展开向下。

History 保留最近 72h 已完赛赛事、整场历史与同步状态。Scratch 仅遮罩价格图表，不隐藏比分、最终结果或快照；提供键盘时间滑块与复位。手动同步是需要写权限的独立操作，只读示例隐藏入口；同步中与失败分别显示，原型只演示本地反馈。

Corners 先呈现统计口径、样本与阶段命中比例，再给出可按球队、阶段和命中筛选的比赛记录。8 场数据明确为合成样本，不冒充真实 64 场赛事。O6.5 按 90 分钟加补时计算；full-match 额外包含加时，点球独立记录。整体 5/8＝62.5%、淘汰赛 3/6＝50%、平均 7.75 来自同一数组；支持 90 分钟／全场排序和口径说明。三个 Sports 页面沿当前源码不新增分页。

来源：[Live React](../../../ui/src/app/member/pages/sports-live.tsx)、[History React](../../../ui/src/app/member/pages/sports-history.tsx)、[赛事卡](../../../ui/src/app/member/pages/sports-market-card.tsx)、[Corners React](../../../ui/src/app/member/pages/world-cup-corners.tsx)；[Live 服务](../../../ui/src/app/shared/services/sports-live-service.ts)、[History 服务](../../../ui/src/app/shared/services/sports-history-service.ts)、[Corners 服务](../../../ui/src/app/shared/services/world-cup-corners-service.ts)；[Live 长期设计](../../design/market-intelligence/sports-live.md)、[History 长期设计](../../design/market-intelligence/sports-history.md)。

### Managed OO 提案与争议

页头与页签之后是有写权限者的单区块解析区，再展示已保存事件。桌面按问题／Market、block／log、原始 proposed price、request time 和参与地址排列；手机先问题，再区块／时间、完整价格串、地址和交易。完整证据采用固定头尾、正文滚动的右侧详情抽屉，辅助 raw 内容默认折叠。

`1000000000000000000`、`0`、`-1000000000000000000` 等 proposed price 保持原始字符串精度，不标作 USD，也不据此判定胜负。proposer 与 disputer 角色明确区分；详情保留 requester、完整参与地址、identifier、原始 timestamp、expiration／currency、block／txIndex／hash／contract／topic、market／condition／slug、ancillary text／hex 与 raw topics／data，缺失值保持 Unknown。

读取与解析写入口分开；只读隐藏解析入口。单区块输入验证正的 safe integer，支持 `block`／`block_number` URL 筛选及默认 50、可选 10／50／100 分页。解析只等待 250ms 后过滤本地合成记录并重置页码，明确显示未执行 Polygon scan；失败保留输入，不表示实际扫描、入库或告警成功。

来源：[当前 React](../../../ui/src/app/member/pages/managed-oo.tsx)、[前端服务](../../../ui/src/app/shared/services/managed-oo-service.ts)、[长期 Managed OO 设计](../../design/market-intelligence/managed-oo.md)。三域导航沿[会员入口](../../../ui/src/app/member/app.tsx)及既有权限分组，不新增模块类别。

## 全部截图与辅助索引

61 张原生 Chromium PNG＝45 张提案（16 张主图＋29 张辅助）＋16 张当前 React 对照。Market Radar 为 21 张（15＋6），Sports 为 26 张（20＋6），Managed OO 为 14 张（10＋4）。每页主图视口为桌面 1440×900、手机 390×844；适配图为 320×844 和 720×900／根字号 200%。除以下四张实际 390×844 弹窗视口外，其余均从文档顶部截取全页，最终 PNG 高度可超过视口：Market 实时复制回退、Sports Live 复制回退、Corners 口径说明、Managed OO 提案证据。弹窗正文从顶部开始且可滚动，不要求单张截图展示全部正文。

辅助状态由既定规则覆盖与验证，不逐图新增审批；本次整批确认不补写成用户逐图看过这 29 张辅助图。

| 页面 | 窄屏与字号辅助图 | 具名状态辅助图 | 当前 React 对照 |
| --- | --- | --- | --- |
| Market Radar 热门市场 | [320px](previews/theme-market-radar-v21-hot-narrow.png)、[200%](previews/theme-market-radar-v21-hot-zoom.png) | — | [桌面](previews/theme-market-radar-v21-before-hot-desktop.png)、[手机](previews/theme-market-radar-v21-before-hot-mobile.png) |
| Market Radar 实时市场 | [320px](previews/theme-market-radar-v21-realtime-narrow.png)、[200%](previews/theme-market-radar-v21-realtime-zoom.png) | [copy-fallback-mobile](previews/theme-market-radar-v21-realtime-copy-fallback-mobile.png)、[facts-mobile](previews/theme-market-radar-v21-realtime-facts-mobile.png)、[stale-mobile](previews/theme-market-radar-v21-realtime-stale-mobile.png) | [桌面](previews/theme-market-radar-v21-before-realtime-desktop.png)、[手机](previews/theme-market-radar-v21-before-realtime-mobile.png) |
| Market Radar 涨跌榜 | [320px](previews/theme-market-radar-v21-movers-narrow.png)、[200%](previews/theme-market-radar-v21-movers-zoom.png) | — | [桌面](previews/theme-market-radar-v21-before-movers-desktop.png)、[手机](previews/theme-market-radar-v21-before-movers-mobile.png) |
| Sports Live | [320px](previews/theme-sports-v21-live-narrow.png)、[200%](previews/theme-sports-v21-live-zoom.png) | [copy-fallback-mobile](previews/theme-sports-v21-live-copy-fallback-mobile.png)、[no-history-mobile](previews/theme-sports-v21-live-no-history-mobile.png)、[stale-mobile](previews/theme-sports-v21-live-stale-mobile.png) | [桌面](previews/theme-sports-v21-before-live-desktop.png)、[手机](previews/theme-sports-v21-before-live-mobile.png) |
| Sports History | [320px](previews/theme-sports-v21-history-narrow.png)、[200%](previews/theme-sports-v21-history-zoom.png) | [failed-mobile](previews/theme-sports-v21-history-failed-mobile.png)、[scratch-mobile](previews/theme-sports-v21-history-scratch-mobile.png)、[syncing-mobile](previews/theme-sports-v21-history-syncing-mobile.png) | [桌面](previews/theme-sports-v21-before-history-desktop.png)、[手机](previews/theme-sports-v21-before-history-mobile.png) |
| World Cup Corners | [320px](previews/theme-sports-v21-corners-narrow.png)、[200%](previews/theme-sports-v21-corners-zoom.png) | [filtered-mobile](previews/theme-sports-v21-corners-filtered-mobile.png)、[method-mobile](previews/theme-sports-v21-corners-method-mobile.png) | [桌面](previews/theme-sports-v21-before-corners-desktop.png)、[手机](previews/theme-sports-v21-before-corners-mobile.png) |
| Managed OO 提案列表 | [320px](previews/theme-managed-oo-v21-proposals-narrow.png)、[200%](previews/theme-managed-oo-v21-proposals-zoom.png) | [evidence-mobile](previews/theme-managed-oo-v21-proposals-evidence-mobile.png) | [桌面](previews/theme-managed-oo-v21-before-proposals-desktop.png)、[手机](previews/theme-managed-oo-v21-before-proposals-mobile.png) |
| Managed OO 争议列表 | [320px](previews/theme-managed-oo-v21-disputes-narrow.png)、[200%](previews/theme-managed-oo-v21-disputes-zoom.png) | [read-only-mobile](previews/theme-managed-oo-v21-disputes-read-only-mobile.png) | [桌面](previews/theme-managed-oo-v21-before-disputes-desktop.png)、[手机](previews/theme-managed-oo-v21-before-disputes-mobile.png) |

旧 React 观察保持原样：Market 三个手机页面文档宽 408px／视口 390px；Managed OO 两个手机表格宽 860px／856px；Sports Live／History 手机图表横溢及 Corners 旧阶段比例裁切见采集记录。它们是当前实现对照，未被裁切隐藏，也不作为新原型回归。

## 检查、独立审阅与证据

| 领域 | 数据 | 局部原型检查 | 当前 React 采集 |
| --- | --- | --- | --- |
| market-radar | [data](previews/theme-market-radar-v21-data.json) | [checks](previews/theme-market-radar-v21-checks.json) | [before-capture](previews/theme-market-radar-v21-before-capture.json) |
| sports | [data](previews/theme-sports-v21-data.json) | [checks](previews/theme-sports-v21-checks.json) | [before-capture](previews/theme-sports-v21-before-capture.json) |
| managed-oo | [data](previews/theme-managed-oo-v21-data.json) | [checks](previews/theme-managed-oo-v21-checks.json) | [before-capture](previews/theme-managed-oo-v21-before-capture.json) |

Market 98 项断言通过；Sports 记录 12 组视口与 24 项交互／状态，修正后另有 8 组来源折叠定向检查；Managed OO 记录 10 张图、9 个行为项及 208 个控件几何样本，全部通过。原型记录未发现整页横溢、重复 ID、页面异常或外部请求，检查覆盖字体、操作目标、完整标识、复制失败回退、键盘及相应分页／筛选／只读／加载／错误状态。计数属于各自记录口径，不合并冒充统一测试套件。

独立 reviewer 的[首次原始审阅](../../../.superpowers/v21-finish-review-initial.txt)打开了全部 61 张图及 v20 桌面／手机参照，按角色五块记录唯一必须修正项：Sports Live／History 来源行字符箭头与 SVG 重复。一个具名修正批删除字符箭头并统一 SVG 方向，14 张受影响图在原路径重拍。[最终原始审阅](../../../.superpowers/v21-finish-review.txt)按角色两块格式给出 `verdict: resolved`、`remaining: clear` 与 `disposition: ship`，只复核这 14 图与该修正引入的回归，结合首次完整审阅关闭唯一待修项；没有重做 61 图完整审阅。两份原始记录均保留其原有结构。[批次审阅记录](previews/theme-market-intelligence-v21-review.json)的 `awaiting_user_review` 保留为交付时历史状态；后续用户批准单独保存在 [v21 确认记录](previews/theme-market-intelligence-v21-approval.json)。

三 HTML 只合并运行一次 detector，7 条提示见[分诊记录](../../../.superpowers/v21-detector-triage.json)。6 个文件范围例外分别是 3 条已确认 Inter 决策与 3 条实际 hover 误报；稳定态 12 个主按钮颜色样本最低 15.145:1，Market 没有 primary 控件。初次过渡未结束的采样无效，未把它当作 UI 问题。剩余 Market 旧 account-menu 的 30px 阴影已按具名问题机械重建，原 15 张提案图同步更新，未运行第二轮 detector。

[完整栅格 manifest](../../../.superpowers/v21-raster-manifest.json)记录路径、摘要、尺寸与四个实际视口弹窗；[来源扫描](../../../.superpowers/v21-raster-provenance-scan.txt)为 61 张、0 缺失。所有 PNG 携带来源，审阅镜像与原件字节一致；本批没有生成图片或截图后修图。

## 合成边界与环境

静态原型不请求真实 provider、不扫描、不同步、不交易。当前 React 对照在独立浏览器上下文以合成 GET 截获，Sports 另将两条只读 POST `price-history:batchGet` 用本地 `route.fulfill` 返回；这不是写入，所有未知或真实写请求拒绝，不向后端透传。真实鉴权、后端权限、竞态、供应商数据与 full-stack 验收未执行；局部检查不能替代这些验收。

用户已整批确认八页十六张主图，后续重构沿用本批布局、阅读层级及操作主次。交付原型、截图、数据、检查与交付时审阅记录共 74 份文件保持原字节，摘要见 [v21 确认记录](previews/theme-market-intelligence-v21-approval.json)；此前 394 份预览资产亦保持不变。正式 `ui/`、主题技术方案与实施尚未完成；第三批 Worm Trading 六页仍待设计，共用适配仍待覆盖，全站设计尚未全部定稿。

本轮未启动或停止服务／容器，所有临时浏览器已关闭。当前 React 对照借用任务前已有的 [localhost:4000](http://localhost:4000) 根仓库 Vite（PID 308228，cwd `/home/yege/work/athena/ui`，实例为根仓库既有运行环境），日志位于 `/home/yege/work/athena/.run/athena-local-runtime/`。环境属于用户或其他任务，按原归属保留；由原归属从 `/home/yege/work/athena` 执行 `make stop` 停止。本批没有自建临时服务需要收尾，也没有将现有 Vite 可访问视为真实业务验收通过。
