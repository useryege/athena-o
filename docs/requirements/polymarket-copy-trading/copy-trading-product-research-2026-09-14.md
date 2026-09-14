# Polymarket 跟单产品调研：策略与完整业务流程

> 调研日期：2026-09-14。
>
> 用途：仅保存市面上 Polymarket 跟单产品的调研资料。用户随后将本次初版调整为 ATHENA 内手动下单；本文不向[交易板块需求](copy-trading.md)引入竞品策略或新增配置。

## 结论

所查产品将金额模式、触发筛选、成交价格、资金限额和退出方式分别设置，并提供跟单关系管理、持仓与执行记录。固定金额是其中一种常见模式；其他选项代表这些产品的范围，不代表 ATHENA 当前需要采用。

## 样本与证据范围

主要比较 Polycule、PolyCop 和 Bullpen 的 Polymarket 跟单功能。这三者有可核对的产品说明或操作文档，覆盖跟单设置及管理流程。本次评估的是功能设计完整度；没有登录实测、提交订单，也没有独立验证用户规模、长期稳定性、成交速度或收益，不能据此认定其交易服务可靠性已经得到证明。

PolyGun 的公开文档列为补充候选，但详细交易页读取失败，本报告不据此断言其策略参数。代币跟单产品中的“每个 Token 只买一次”、永续合约的仓位镜像，以及开源示例机器人均不作为本轮 Polymarket 产品行为的证据。Bullpen 同时覆盖其他市场，这里仅采用其明确标为 Polymarket 的文档部分。

| 产品 | 官方资料中明确描述的能力 | 设计观察 |
| --- | --- | --- |
| Polycule | 固定金额、比例及带上下限的比例模式；单市场投入上限；目标成交金额、价格、流动性和期限筛选；卖出单独配置；编辑、暂停和恢复。 | 金额、准入和退出分开设置；单市场投入上限不等于只买一次。[跟单指南](https://www.polycule.trade/copy-trading) |
| PolyCop | 固定金额或比例；单个 Yes/No Outcome 的持有金额上限及总持有金额上限；低于最低要求时选择跳过或上调金额；目标小额成交过滤；相对目标价格的限价偏移与期限。 | 限额可以按 Outcome 计算；低于最小值的处理是显式选项。[设置指南](https://polycop.gitbook.io/polycop-docs/copy-trading/how-to-copy) |
| Bullpen | 固定金额、目标交易比例或自己余额比例；预算、单市场及每日额度；价格和时效条件；自动跟卖、仅提醒退出或手动退出；执行历史包含成功、失败、跳过及原因。 | 退出方式独立设置；执行历史覆盖没有形成成交的处理结果。[Polymarket 跟单教程](https://cli.bullpen.fi/skill/references/copy-trading-tutorial/) |

这些是厂商文档描述的行为，不照搬其中的推荐金额、参数、资金币种或收益说法。平台约束以当前 Polymarket 官方文档为依据。

## 完整业务流程观察

| 环节 | 查到的产品行为 |
| --- | --- |
| 选择目标 | Polycule 支持输入地址并命名；PolyCop 支持地址或 Polymarket Profile 链接。来源见上表。 |
| 配置跟单 | 用户先设置金额模式与条件，再启用目标跟单。来源见上表。 |
| 管理关系 | Bullpen 提供列表、状态、编辑、暂停、恢复、结束与删除，并将执行记录单列。[跟单管理说明](https://cli.bullpen.fi/reference/commands/tracker/copy/) |
| 查看持仓与操作 | Polycule 文档提供持仓查询及从仓位创建卖出限价单的流程。[产品操作文档](https://polycule.trade/docs) |
| 手动交易 | PolyCop 支持选择 Outcome 后按预算市价买入，或填写价格和预算限价买入。[手动交易指南](https://polycop.gitbook.io/polycop-docs/manual-trading/how-to-manual-trading) |
| 查看执行历史 | Bullpen 可以按目标、状态筛选执行记录，显示市场、方向、金额、价格，并附错误或跳过原因。来源见上表的跟单教程。 |

这些样本展示了从建立关系、配置启用、持续管理，到持仓操作及历史查询的业务覆盖。本次没有核验所有产品的历史保留期限、钱包绑定限制或真实页面状态，不能推断三者行为完全相同。

## Polymarket 特有概念与平台核对

### Market、Event 与 Outcome

Polymarket 的 Event 可以包含多个 Market；普通二元 Market 对应 Yes、No 两个 Outcome。按事件、按市场和按单个 Outcome 计算限制，会得到不同结果。[官方市场与事件说明](https://docs.polymarket.com/concepts/markets-events)

PolyCop 的单个 Yes/No 限额和 Polycule 的单市场投入上限属于不同口径。另外，投入金额上限仍可能允许多次加仓，不等于限制交易次数。这是对竞品规则的比较；ATHENA 当前手动交易初版不采用每市场一次限制。

### 金额、价格与成交

Polymarket 官方区分立即全部成交否则不成交（FOK），以及立即成交可成交部分、取消剩余部分（FAK）；同时存在市场最低份额与价格精度要求。固定金额不保证足额成交，竞品文档里的最低美元金额也不能当作所有市场的统一约束。[官方下单说明](https://docs.polymarket.com/trading/place-orders)

例如目标成交价为 0.50，随后可买价格变为 0.65，即使投入金额相同，买到的份额也会不同。这是说明金额与价格区别的假设算例，不是实际成交测量或交易建议。

### 卖出与赎回

Polymarket 将通过订单簿买卖份额和市场结果确定后的获胜份额赎回区分为不同动作。[官方持仓与代币说明](https://docs.polymarket.com/concepts/positions-tokens)

### 交易事实与跟单过程

Polymarket 提供[用户订单](https://docs.polymarket.com/api-reference/trade/get-user-orders)、[成交](https://docs.polymarket.com/api-reference/trade/get-trades)和[持仓](https://docs.polymarket.com/api-reference/wallet/list-positions-for-a-user-or-market)查询。跟单产品另有目标、策略和执行过程：例如 Bullpen 的规则跳过发生在自身跟单流程中，未必形成交易所订单。两类记录覆盖的信息不同；本文仅记录这一调研观察，不确定 ATHENA 的数据设计。

## 与 ATHENA 本次范围的关系

用户已明确竞品只做调研并存入仓库，随后将自己的初版进一步简化为：

- 一个目标对应一个不同的自有 Wallet 钱包。
- 目标出现成交后，通过现有通知流程提醒用户。
- 用户在电脑或手机上的 ATHENA 内判断是否交易、确认本次金额并提交，由 ATHENA 向 Polymarket 下单。
- 不实现自动跟单；手动买入不限制次数，每次由用户决定。
- 提供自己的持仓、手动卖出、订单和交易历史。

竞品的固定／比例自动跟单、筛选器、预算配置、自动跟卖等均保留为研究资料。当前需求的未决细节以[交易板块需求](copy-trading.md)为准；全部需求确认后再进行前后端设计。

[返回交易板块需求](copy-trading.md) · [返回产品需求](README.md)
