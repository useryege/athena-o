# Trader Sync 手动交易：平台契约校验

> 校验日期：2026-09-14。范围为官方文档、公开接口实测、现有源码和独立十进制计算；不是已实现功能的验收。既有[手动交易需求](copy-trading.md)继续有效，不重新引入自动跟单或 Combo 执行。

> 后续范围更新（2026-09-15）：用户要求不再关注 Combo，后续校验聚焦普通市场与 Neg Risk 单个选项，必要时可更换公开钱包样本。本文保留已取得的 Combo 事实，不继续追查字段差异或将其作为设计推进前提；未验证的内容仍不得宣称通过。

## 结论

普通市场与 Neg Risk 单个选项的手动市价买卖、价格保护、含费余额检查和手动领取，在官方接口中有对应能力，可以继续开展设计。现有仓库尚未具备 Polymarket 交易账户启用、订单签名与提交能力。

**不能把本次结果表述为「交易链路和全部历史已验证通过」。** 私有账户的交易权限、签名、实际成交、费用扣取及领取尚未实测；默认持仓和历史查询存在筛选，完整覆盖需要进一步核对。入金操作范围已补充确认：用户自行到 Polymarket 入金，见末节。

## 证据范围

- [官方资料清单](evidence/manual-trading-2026-09-14/official-documents.json)：访问地址、时间、HTTP 状态及内容摘要值。资料由 Polymarket 官方提供；仓库旧文档快照不作为当前规则的唯一依据。
- [公开接口证据](evidence/manual-trading-2026-09-14/public-api.json)：29 次 GET，27 次返回 200，2 次持仓裸游标请求返回 400。保留查询参数、响应和时间；400 是需要记录的契约差异，不计作通过。
- [市场样本](evidence/manual-trading-2026-09-14/market-samples.json)：从本次 Gamma 响应选取的普通、Neg Risk 市场身份及交易约束，用于与 CLOB 对照。
- [十进制边界检查](evidence/manual-trading-2026-09-14/decimal-boundaries.json)：144 组价格边界计算，以及份额尾数、费用示例。仅验证计算方向，不代表 SDK 或交易所接受这些订单。
- [源码核对](evidence/manual-trading-2026-09-14/source-review.json)：本次查看的源码路径、内容摘要值和能力边界。
- 后续用户提供的 [timetowander 公开账户复核](timetowander-public-account-verification-2026-09-14.md)另保存 27 次 GET 尝试的证据，补充较大历史样本、小额余量和 Combo 字段差异；不计入上述初轮 29 次请求。

样本取自已有公开钱包核验记录和官方公开成交返回的钱包地址。没有使用用户密钥、创建交易账户、生成授权或订单签名、提交交易、领取或划转资金；没有启动 ATHENA 服务。

## 平台规则与需求的对应

| 核验项 | 已核准内容及对本系统的影响 | 证据层级 |
| --- | --- | --- |
| 市价执行 | FAK 可以立即成交可成交部分并取消余量，与不挂单、不自动补单的要求相容。后续技术设计可采用该契约，不增加用户策略选项。 | [下单文档](https://docs.polymarket.com/trading/place-orders#market-order-types)；未实际下单 |
| 金额口径 | CLOB 买入 `amount` 为买入本金，适用费用另加；`maxSpend` 会约束含费支出并缩减签名本金。维持用户确认的本金与费用分列方式。 | [支出上限](https://docs.polymarket.com/trading/place-orders#cap-market-buy-spending) |
| 最小数量与价格步长 | 本次普通、Neg Risk 两个订单簿样本均返回最小数量 5 份、价格步长 0.01。这是样本规则，不是全平台固定金额；市场约束必须实时读取。 | `book-1163699`、`book-2252244`；[市场约束](https://docs.polymarket.com/market-data/market-details) |
| 金额与份额精度 | 官方构单表随 tick 调整价格和金额精度，输入份额采用两位小数，链上编码使用六位单位。资产余额精度不能直接当作订单输入精度。 | [构单规则](https://docs.polymarket.com/trading/place-orders#place-a-limit-order) |
| 手续费 | 市场费率与价格共同影响费用。本次两个市场的 `fd.r` 分别为 0.04、0.05，均 `e=1`、`to=true`，与 Gamma 样本一致；不能直接按本金固定扣 4% 或 5%。 | `clob-info-*`；[手续费](https://docs.polymarket.com/trading/fees)、[费率字段](https://docs.polymarket.com/market-data/market-details#trading-fees) |
| 成功状态 | 下单受理、撮合、上链和最终成功不同。成交状态中的 MATCHED、MINED、RETRYING 均非最终成功；界面与历史须继续核对结果。 | [订单生命周期](https://docs.polymarket.com/concepts/order-lifecycle) |
| 普通与 Neg Risk | 两类市场都有独立可交易 token；Neg Risk 需选择正确的交易及授权合约。普通多市场事件不必然都是 Neg Risk，不能靠标题猜类型。 | 两类 CLOB/Gamma 样本；[Neg Risk](https://docs.polymarket.com/concepts/negative-risk) |
| 领取 | 官方领取流程按 condition 处理钱包对应 Outcome 余额，需市场已结算和相应 adapter 授权。它是链上操作，不能复用市价卖出的成功判定。 | [管理持仓](https://docs.polymarket.com/trading/positions/manage#redeem-resolved-positions)；未实际领取 |

本次没有发送低于最小数量或精度错误的真实订单，因此没有验证交易所对各类 FAK 边界的实际拒绝行为。[官方错误说明](https://docs.polymarket.com/resources/error-codes#order-processing-errors)列出了数量、tick、重复订单和余额／授权错误；这些仍须在后续集成验收中覆盖。

### 价格保护与卖出尾数

以下是为保持已确认要求而做的独立计算推导，不是平台新增策略：

- 买入保护上限为确认报价乘以 `1 + 偏差比例`；对齐可用价格步长时只能收紧，不能提高用户的上限。卖出下限按 `1 - 偏差比例` 计算，对齐时不能降低用户的下限。
- 示例：确认报价 0.53、偏差 2%、步长 0.01，买入上限为 0.54，卖出下限为 0.52。不能把卖出下限向下舍入成 0.51。
- 比例为 0%、报价靠近价格区间端点或步长改变时，若没有满足保护要求的有效价格，应重新报价或提示不可下单，不能放宽保护。144 组计算均检查了这一不变量。
- 示例持仓 5.123456 份，在两位份额精度下选择 100% 后，最多表达 5.12 份，剩余 0.003456 份不能从展示中消失，也不能宣称余额为零。低于最小可卖数量、可用余额变化与部分成交分别展示原因。

报价必须针对本次买卖数量和当前深度核算。后续界面需区分预计均价、受保护的最高／最低成交单价与目标的历史成交价；不能拿目标历史价或盘口中间价冒充当前可执行报价。最终构单金额还需验证实际有效单价仍在用户边界内。

费用计算例：仅假设单一成交价 0.50、买入本金 10、费率参数 0.05、指数 1 且无 builder 费用，平台费为 0.25，合计 10.25。该例用于说明本金不等于总支出；实际多档成交须按适用规则核算，未验证真实扣费及逐笔舍入。

## 钱包与可用资金

| 内容 | 校验结果 |
| --- | --- |
| 自有 Wallet | 当前 Wallet 管理 EVM 和 Solana 密钥；现有签名接口专用于 Worm。Polymarket 的 Polygon 交易需要 EVM 签名身份，不能直接复用 Solana 签名接口。 |
| 实际交易账户 | 官方新账户默认使用 Deposit Wallet，需区分它与签名 EOA。建立目标绑定不会自动把原 Wallet 地址中的资金或历史迁移到交易账户。 |
| 账户准备 | 创建用户 Deposit Wallet 需要集成方 Builder 能力；连接既有账户、无 gas 钱包操作与订单认证有各自凭据要求。当前项目尚未完成对应接入。 |
| 交易授权 | 买卖需要对应交易合约的代币额度／操作授权，领取还涉及 adapter。准备成功与资金充足是不同状态；每次操作继续校验实际条件。 |
| SDK 默认副作用 | 官方 SDK 文档描述了下单时自动补齐缺失授权的能力。选型时必须核对默认行为，不能让构建页面客户端、报价或普通绑定隐式触发账户部署、签名或授权。 |
| 资金币种 | 当前官方结算资产是 Polygon 上六位精度的 pUSD。历史数据字段名含 `usdc` 不代表可以直接使用同名字段推定当前交易账户的可用资金。 |

来源：[钱包与授权](https://docs.polymarket.com/trading/wallets-auth)、[pUSD](https://docs.polymarket.com/concepts/pusd)；现有实现见 [Wallet 契约](../../../internal/wallet/wallet.proto)、[钱包归属设计](../../design/identity-access/wallet-ownership.md)。本次未核验集成方凭据、真实账户授权、部署或资金到账。

初版仍维持「用户主动启用，逐笔手动交易」。普通 EVM 地址也不能被默认视为可直接发单的账户，官方文档对直接 EOA 交易有准入条件；具体账户接入方案留给技术设计。

## 持仓与历史的实际覆盖

### 普通持仓及买卖

当前公开 Data API v2 使用 `{ data, pagination }` 和游标，与仓库旧 `/positions`、`/trades` 数组及 offset 形式不同。现有 [DataClient](../../../util/polymarket/data.go)不能仅改基础地址就变成 v2 客户端。[官方持仓接口](https://docs.polymarket.com/api-reference/wallet/list-positions-for-a-user-or-market)

| 查询 | 官方规则／实测结果 | 对「全部展示」的影响 |
| --- | --- | --- |
| 普通成交 | `/v2/trades` 默认 `taker_only=true`，用户历史默认最近三年；使用 `start=1` 请求完整时间范围，`taker_only=false` 包含 maker 行。 | 不设这些参数会遗漏其他渠道的部分成交。[成交文档](https://docs.polymarket.com/api-reference/feeds/list-trades) |
| 小额成交 | 交易筛选默认下限 0.01，传 0 仍使用默认值。本次显式传 0.000001，接口接受。 | 0 不能当作取消过滤；未取得极小额边界样本，不保证该参数已证明全部覆盖。[成交文档](https://docs.polymarket.com/api-reference/feeds/list-trades) |
| 普通持仓 | 默认 OPEN，包括尚持有的可赎回仓位；默认存在 0.1 份筛选且不含归档市场。`include_archived=true` 仍排除 inactive 市场。CLOSED 独立查询。 | 显式核对小额、归档、inactive 与已关闭范围，不能将缺失行解释为零资产。[持仓文档](https://docs.polymarket.com/api-reference/wallet/list-positions-for-a-user-or-market) |
| 活动历史 | `/v2/activity` 支持 `type=TRADE`、`start=1`；分页必须保持原筛选。 | 用于交叉核对成交，不能用账户生命周期事件伪造成交。[活动文档](https://docs.polymarket.com/api-reference/feeds/list-account-activity) |

对既有公开样本钱包 `0x5c52d767c32cc18100c72d7471209d9618216058` 的控制变量查询：

- 同为 `start=1`、`limit=1000`、`filter_amount=0.000001`，`taker_only=true` 返回 6 条，`false` 返回 35 条；两次均 `has_more=false`。
- `activity?type=TRADE&start=1` 返回 35 条。按交易哈希、token、方向、数量与时间做多重集合对照，与上述 35 条成交样本一致。这个比较只用于核验本样本，不是生产去重键；一笔链上交易可能有多条成交。
- 3 条一页的成交查询已取得第二页；两条普通持仓拆成两页，与一次返回的两条 token 集合一致。
- 所见历史来自 2024 年，未证明三年前记录、任意钱包或全链历史均完整；分页耗尽只证明当前查询返回的集合耗尽。

### 已复现的分页差异

官方普通持仓文档说翻页只需 `cursor`。本次对同一游标两次仅传 `cursor` 均返回 400，原因是缺少 `user` 或 `condition`；增加原 `user` 后返回第二条持仓，携带全部原筛选也成功。

后续接入应保留原账户与查询条件、按端点契约使用原样游标。证据中的 `gcr-positions-page2-*` 保留失败及成功请求。这里只确认当前公开接口的实际要求，不推断服务端内部原因。

### Combo 与领取资格

Combo 的 `/v2/positions/combos`、`/v2/activity/combos` 返回独立结构。公开样本 `0x237419de121f3aecf3e1e0a613ecb9dc82b9a426` 两个接口各取得 3 条且仍有下一页，增量持仓查询也取得含零余额的历史行。其生命周期活动出现 SPLIT；SPLIT 金额不能当作 BUY 本金。增量数据还有稳定性滞后，不能用它判定订单刚刚失败。[Combo 持仓](https://docs.polymarket.com/api-reference/wallet/list-combo-positions)、[Combo 活动](https://docs.polymarket.com/api-reference/feeds/list-combo-activity)

这些样本验证了 Combo 展示数据的可取得性，**未证明全部 Combo 买卖都由生命周期接口覆盖**。其历史成交仍须结合 [Polymarket 实际成交源](source-contract-verification.md)核对；不能因为初版不支持 Combo 执行，就缩小已确认的 Combo 历史展示范围。

普通持仓实测还出现 `redeemable=true`、`current_price=0`、`current_value=0` 同时存在的记录。这不足以确定实际领取金额；领取入口必须核对市场结算结果、当前余额及适用合约，不能把布尔标记直接显示成有正金额可领。

### 用户提供的 timetowander 账户复核

用户随后提供朋友的公开资料页用于只读校验。[独立报告](timetowander-public-account-verification-2026-09-14.md)核准资料页与 Gamma 对应同一交易账户；固定 24 小时窗口中，包含 maker／taker 的 740 条成交与活动接口一致，只查 taker 为 34 条。两页共 2,000 条历史也与活动逐页一致，但仍有更多历史未读取。

该样本的 OPEN 查询返回 889 条，其中 874 条可赎回标记为真但当前价值为零；CLOSED 另返回 265 条正份额小额余量。OPEN 传 `filter_amount=0` 或 `0.000001` 都没有取得这些余量，故不能仅凭调低过滤值承诺资产完整，也不能把 CLOSED 直接显示为余额归零。

Combo 持仓与活动还出现同一腿 `leg_condition_id` 与标签不同的情况，重复请求及一个 Gamma 市场复核已记录。完整资产展示需处理该映射差异；Combo 生命周期仍不能代表全部买卖。此次未验证钱包控制权、签名、真实余额或任何交易写入，原私有链路验证缺口保持有效。

### 本系统下单记录

CLOB 默认订单列表返回当前订单，已知 ID 可查单个订单；Session Key 与账户 owner 的可见范围也有限制。它不能代替全钱包历史，更无法补回只在 ATHENA 本地被拒绝的提交。[管理订单](https://docs.polymarket.com/trading/manage-orders)

因此原有决定保持不变：ATHENA 记录用户提交、拒绝／失败／未知状态和原始目标关联，真实订单与资产结果向 Polymarket 核对。本次没有使用私有 CLOB 凭据验证历史可见性、按 ID 恢复或丢失响应场景。

## 现有能力与后续验证

| 能力 | 仓库现状 | 后续需要落实 |
| --- | --- | --- |
| 活动和通知 | [Trader Sync](../../../internal/tradersync/apiclient/trader_sync.proto)已有订阅、活动与通知相关读取。 | 衔接用户手动下单上下文；保留 Activity Alerts 不补目标历史的既有规则。 |
| 市场与盘口 | [CLOBClient](../../../util/polymarket/clob_market_data.go)有只读盘口、价格、tick 和费率读取。 | 验证报价深度、精度、市场可交易状态与过期处理。 |
| 持仓及历史 | DataClient 有旧版公开接口，尚无此次核验的完整 v2 查询与覆盖流程。 | 普通及 Combo 的全量、分页、刷新、缺口和来源身份核对。 |
| 钱包与执行 | Wallet 有归属校验、EVM/Solana 托管及 Worm 专用签名，没有 Polymarket 执行能力。 | 账户准备、认证、受限签名、授权、订单提交与领取。 |

进入开发前及集成验收还需验证：

1. 选定账户类型及可用 Builder／Relayer 凭据；验证用户在 Polymarket 登录及入金的账户与 ATHENA 执行账户一致，核对对应地址、可用余额、授权、签名与恢复路径。
2. 真实普通及 Neg Risk 买卖，包括最小额、价格保护、份额尾数、费率变动、余额不足、部分成交、延迟及未知结果；SDK 不隐式改本金或新建重试订单。
3. 真实领取的正金额、零金额、已领取、授权不足与重复提交；按条件核对最终资产变化。
4. 普通市场及 Neg Risk 单个选项的更早历史、inactive／归档市场、小额余额、极小成交及不同签名渠道的覆盖。若公开索引不足，需设计对 Polymarket 链上事实的补证，不能自行缩小这些市场的「全部展示」要求。既有目标监控不补历史的决定不适用于自有钱包历史范围；Combo 不再列为后续专项校验任务。

以上是验证缺口，不是重新询问用户技术事实，也不要求本轮启动尚未实现的产品服务。

## 入金方式确认

用户于 2026-09-14 明确确认：**入金由用户自行到 Polymarket 完成。** ATHENA 初版展示实际交易账户、可用余额及必要的入金提示，不提供站内充值或从 Wallet 划款的功能。

对应业务待确认项已闭合。后续仍需验证 Polymarket 网站账户与 ATHENA 执行账户的一致性、入金后的余额读取；用户返回页面不等于到账。ATHENA 内手动买卖、普通市场与 Neg Risk 单个选项的结算领取，以及其他已确认规则继续有效。

[返回交易需求](copy-trading.md) · [返回产品入口](README.md)
