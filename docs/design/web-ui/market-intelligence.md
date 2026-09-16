# 市场、赛事与 Managed OO 会员页面

> 目标范围更新（2026-09-16）：[Sports 删除、Worm 保留](../../requirements/development-runtime/sports-removal.md)已确认，尚未实施。Sports Live／History、World Cup Corners 转为清理范围；Worm、Market Radar、Managed OO 继续按各自需求维护。

八页前端消费既有 service DTO，沿用 [v21 布局](../../requirements/web-ui/market-intelligence-batch-proposal.md)与 [v23 一致性契约](../../requirements/web-ui/theme-consistency-contract.md)。本说明记录 T6 的页面实现边界；外部 Polymarket、Polygon 和扫描链路的可用性须另行验收。

## 页面与数据阅读

| 入口 | 页面展示 |
| --- | --- |
| `/market-radar` | Hot 活动指标：24h volume、liquidity、spread、outcome probability；重复指标使用明确数值列角色 |
| `/market-radar/realtime` | Outcome 价格及 1m／5m／15m 百分点变化、warmup 与样本状态 |
| `/market-radar/movers` | 服务返回的 leader、direction 和 mover score；不在前端重算排名 |
| `/sports-live` | 比分与比赛阶段、当前 moneyline 快照、按业务时间排列的历史曲线、完整来源标识 |
| `/sports-history` | 完赛时间／比分、价格历史、独立同步状态，以及不改变比分和快照的 Scratch／Reset |
| `/world-cup-corners` | 返回数据集的实际样本、分阶段命中率；90 分钟加补时角球与包含加时的全场角球分列，点球比分独立 |
| `/managed-oo/proposals` | 问题、区块、原始 proposed price、请求时间、完整 proposer 地址和交易证据 |
| `/managed-oo/disputes` | 同一日志事实结构中的 disputer 角色和争议证据 |

Market Radar 的三个展示组件分别接受 `MarketRadarHotMarketItem`、`MarketRadarRealtimeMarketItem` 和 `MarketRadarMoverMarketItem`。桌面与手机消费同一记录模型。`MarketVolume` 将已知零显示为 `0`，缺失显示为 `Unavailable`；24h 和 total 各用各自字段。原始价格、完整 condition/token 标识及窗口样本按需展开。顶层 proto3 计数（candidate、monitored/subscribed market/token）经实际 encoding/json 网关省略时代表零；局部解码保留该默认值，显式 null／非法计数仍显示 Unknown。该规则不应用到 nested 金额、价格或时间；顶层省略布尔仍按合法 false 处理。分页默认 50，保留 10／50／100 选择；刷新保留有效页，数据收缩时修正越界页。

Sports 图表由 `sports-market-card.tsx` 的 SVG 绘图代码控制，横坐标使用有效的业务 timestamp。缺时间的点不以数组索引代替；缺 outcome 的序列不借用其他 outcome。蓝／黄／紫区分球队／outcome，反馈色另行表达成功、失败与陈旧。时间选择支持指针和键盘，并显示完整北京时间与选中值。历史加载、历史请求失败和成功无历史分别表达；没有数据时不绘造曲线。比分、moneyline 快照与价格曲线保持各自口径。

角球筛选与升降序排序在表格外统一维护，桌面和手机切换后保留相同结果。样本为零只在数据读取成功后成立；统计沿用返回比赛记录，不把局部 fixture 固定称为完整 64 场赛事。

Managed OO 的 proposed price 保持字符串原值，包括零和负值，不换算或改写为概率。详情保留 requester／proposer／disputer、交易、区块和原始日志证据。只有具备写权限时提供区块解析；解析进行中禁用重复提交，失败保留输入以便重试。

## 请求与身份边界

首次请求失败时显示错误，不同时展示成功空态或零记录。已有记录刷新失败保留旧数据并标明陈旧。八条入口按会员 accountId 与 issuer 重挂载；realm 由独立会员应用根限定。离开页面、替换身份或撤销写权限后，迟到的解析／同步请求不回写旧草稿、通知或刷新当前页面。

Sports History 的 `Reload saved data` 只重载已保存事件／价格历史，`Sync history` 是仅对可写账号开放的独立手动同步；同步进行中禁用重复提交。

页面仍调用既有读取、手动刷新和区块解析 API，没有增加后端或服务边界。验收使用 `theme:markets` 严格声明 fixture 请求；区块解析 POST 仅在用例明确声明的本地拦截中执行，不证明真实扫描、链上写入或供应商接入。

## 有界验证入口

- 展示与时间曲线单测：`market-radar-presentation.test.tsx`、`sports-market-card.test.tsx`。
- `UI_ACCEPTANCE_GREP='theme:markets' make ui-acceptance`：root 与 `/athena`，八页桌面／手机主图、数据语义、失败／空态互斥、分页、Sources、解析 pending／failed、身份切换、320px 与根字号放大。
- `UI_ACCEPTANCE_GREP='theme:a11y' make ui-a11y` 包含八个 `markets-*` 主场景；自动检查不能替代完整页面和外部系统验收。

主场景 ID 为 `markets-hot`、`markets-realtime`、`markets-movers`、`markets-live`、`markets-history`、`markets-corners`、`markets-proposals`、`markets-disputes`。所有临时截图、trace、失败轮和 harness cleanup 证据保存在独立 `.tmp/` 目录，原始批准资产保持不变。
