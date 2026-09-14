---
version: 1
slug: "b-ui-previews-theme-market-radar-v21-html-1c81e743"
primary_target: "docs/requirements/web-ui/previews/theme-market-radar-v21.html"
related_targets: ["ui/src/app/member/pages/market-radar.tsx"]
---

# Market Radar v21 Direction contract

## THESIS
Operate。对应既有 Hot Markets、Realtime Markets、Market Movers 三个真实路由；只制作视觉原型，FORM-SEED: not-applicable。会员需要比较市场活动、短窗概率变动与领涨跌 outcome。

## OWN-WORLD
沿用 v20、v13、v18 的已批准 Nansen 单深色世界。近黑底、灰面板、克制细分隔线；Inter 负责标题、正文、数值；JetBrains Mono 仅完整 condition/token 标识。固定 28/24/20/16/14/13 字阶与既有 224px 会员侧栏。

## STORY
页头说明数据模型和刷新；快照行区分 fetched、stale、采样连接与候选/监控数量；主要区域先展示各页关键事实。Hot显示24h volume/liquidity/spread，Realtime显示价格与1m/5m/15m百分点窗口，Movers显示leader、weighted score与方向。其余完整事实使用行内展开，未知值明确 Unknown。

## FIRST VIEWPORT
桌面采用横向可比较的分隔记录；手机每个市场标题在前，指标按2或3列重排，原始标识自然换行。320px与200%根字号保持无水平溢出；控件至少44px，键盘焦点清晰。主截图整页顶部，样本提示放页尾，辅助状态入口收起。

## FORM
刷新只重载合成快照，保留分页；每页数量10/50/100默认50，页面上下界按返回样本夹紧。原型不发起供应商、交易、真实写操作。数据只读；权限丢失示例隐藏该模块数据；错误、无数据、陈旧、窗口warmup与加载按现有状态规则处理。

## FINISH
不修改正式 ui 或旧批准资产。使用同一合成 GET 数据采集当前三页桌面手机6图后制作原型；每页四尺寸截图，必要状态、分页、复制失败、键盘与完整标识检查。同一批截图打开检查一次，所有问题统一修正一次再确认。局部原型与合成 GET 不证明真实身份、后端、实时供应商或完整权限流程。

## QUALITY BAR
三页第一屏能分别回答活动量、价格变化与领涨跌问题；保留单位、精度、unknown、warmup和UTC+8时间；状态不混用。完整ID可展开与复制。桌面手机层级一致，无外部请求、JS异常、重复ID与横向溢出；所有可操作目标>=44×44。
