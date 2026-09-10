# Trader Sync 数据源契约核验

> 核验日期：2026-09-10。只读协议与资料研究；不代表业务实现或全部性能验收。
>
> 关联：[需求](target-trade-monitoring-notifications.md)、[后端设计](../../design/trading/trader-sync-activity-alerts.md)、[此前成交样本](onchain-trade-data-feasibility.md)、[采集契约与额度核验](collector-contract-verification.md)。

## 证据与结论

本轮重新核对官方文档、当前部署代理、验证源码、真实 RPC 响应及官方 Profile 页面消费链。以下分别区分源码契约、实时样本与仍不能推广的结论。

| 证据 | 保存内容 |
| --- | --- |
| [协议与市场响应](evidence/trader-sync-protocol-current-rpc.json) | 查询时间、RPC 请求/响应、实现槽、抵押币、getLegs、迁移腿与 Gamma 映射。 |
| [实现验证记录](evidence/trader-sync-implementation-evidence-2026-09-10.json) | 当前实现地址、Sourcify URL、验证状态、相关 ABI、响应与源码 SHA-256；不保存完整第三方源码。 |
| [Profile 资料证据](evidence/trader-sync-profile-evidence-2026-09-10.json) | 页面与 bundle 来源及 SHA-256、公开资料、搜索结果、Predictions 和相关页面数据。 |

### 当前代理实现不能照抄官方页面

[官方 Contracts 页](https://docs.polymarket.com/resources/contracts)列出的交易所代理地址仍与本次观察一致，但其中 Combo Exchange 与 CombinatorialModule 的实现地址已经与链上不同。

在区块 `93,550,754`（`0x59378a2`）、哈希 `0x1d64636dcdfa966bb1c6564f0ff3f4322bfc23150c2579f737eb36c9bb5028ad` 查询 EIP-1967 implementation slot，得到下表。Combo Exchange 与 CombinatorialModule 经 dRPC/PublicNode 双节点对照；BinaryModule 本次仅由 dRPC 读取，不能宣称三者均完成双节点验证。

| 代理 | 当前实现 |
| --- | --- |
| Combos Exchange `0xe3333700cA9d93003F00f0F71f8515005F6c00Aa` | `0x641b40ec414a076b9e79e703fc7bf4ebec248bb7` |
| CombinatorialModule `0x30000034706C7d8e12009DAB006Be20000c031A8` | `0xf96968a44022b17240b42c557693e7c383d2d8a3` |
| BinaryModule `0x1000008dD9001B968442c1000017eaE6E0dA00Ba` | `0xf6428c0b5fa9361c0708cddb95468cf54c56e9a2` |

随后通过 Sourcify 取得[当前 Exchange](https://sourcify.dev/server/v2/contract/137/0x641b40ec414a076b9e79e703fc7bf4ebec248bb7?fields=abi,metadata,sources)、[当前组合模块](https://sourcify.dev/server/v2/contract/137/0xf96968a44022b17240b42c557693e7c383d2d8a3?fields=abi,metadata,sources)和[当前 BinaryModule](https://sourcify.dev/server/v2/contract/137/0xf6428c0b5fa9361c0708cddb95468cf54c56e9a2?fields=abi,metadata,sources)的 ABI 与源码。三者 `runtimeMatch=match`；这是验证服务的匹配结果，不是本次独立重编译或 `exact_match` 声明。没有调查升级发生时刻。

后端须绑定已核验的来源地址、实现版本和解码器。旧实现源码、官方地址表或相同事件签名，均不能单独证明代理当前所有行为；事件 ABI 相同也不表示交易 calldata 内部布局相同。

### 成交归属、粒度与金额

普通 CTF `0xE111180000d2663C0091e4f400237545B87B996B`、Neg Risk `0xe2222d279d744050d28e00520010520000310F59` 与上述 Combo Exchange 的当前相关事件均使用：

`OrderFilled(bytes32,address,address,uint8,uint256,uint256,uint256,uint256,bytes32,bytes32)`

`topics[2]` 保存自身订单资金钱包；主动 Taker 同样产生自己的 `OrderFilled`。`topics[3]` 是对手方，不因目标在这里出现而再生成自身成交；`OrdersMatched` 也不另计活动。普通 V2 的固定[事件逻辑](https://github.com/Polymarket/ctf-exchange-v2/blob/ccc0596074f4dfd62c944fbca4de252893b82b4b/src/exchange/mixins/Events.sol)与当前 Combo `Exchange.sol` 的 `_emitOrderFilled`、`_emitTakerEvents` 支持该归属。

一条自身日志是本方案的源记录单位。主动订单日志可能涉及多个对手方，不宣称与所有 Data API 行永久一一对应，更不等同于一张完整订单。

| 事件方向 | 抵押币原始量 | 份额原始量 |
| --- | --- | --- |
| BUY / `side=0` | `makerAmountFilled` | `takerAmountFilled` |
| SELL / `side=1` | `takerAmountFilled` | `makerAmountFilled` |

三个 Exchange 的当前抵押币 getter 都返回 `0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB`，该币本次返回 `symbol=pUSD`、`decimals=6`。费用独立保存；已核验逻辑中 BUY 另收费用、SELL 从收入扣费，成交额与含费现金变化分开。份额的六位单位依据官方[持仓操作](https://docs.polymarket.com/trading/positions/manage)与[Combo 请求契约](https://docs.polymarket.com/trading/combos/requesters)。金额不通过二进制浮点，不因 Data API 的 `usdcSize` 字段名而改标币种。

### Combo 两腿链上映射样本

当前 `getLegs(bytes31)` ABI 返回 `uint256[]`。PositionId 的高 31 字节表示结构化 condition，低 8 位表示自身 Outcome；当前验证源码仍保持这一布局。

样本 PositionId `1677415912060011761505225895152832125576889787899132551452768910625300545536` 对应 condition `0x03b5623e2ec8037416edce55a0ed93553a0000000000000000000000000000`。本次向 CombinatorialModule 代理调用 `getLegs`，返回两个 module ID 为 1 的 Binary 腿，与此前生命周期样本一致。

对两腿分别向当前 BinaryModule 调用 `legacyConditionId(bytes31)` 与 `getLegacyPositionId(bytes32,uint256)`，得到原 CTF condition/token。随后直接以链上返回的 token 查询 Gamma，明确使用 `closed=true`，各命中一个真实市场：

| Gamma market ID | 市场 | 链上 token 匹配的 Gamma Outcome |
| --- | --- | --- |
| `3978602` | Sporting CP vs. Galatasaray SK: Both Teams to Score | Yes |
| `3978659` | Spread: VfB Stuttgart (-1.5) | VfB Stuttgart |

这条链路是 `Combo PositionId → getLegs → Binary 迁移映射 → Gamma token → 市场及 Outcome`，不依赖当前钱包持仓或同一交易包含 SPLIT。相同样本的当前 Combo positions 查询为空，进一步说明“没有当前持仓”不能用于否定成交。生命周期资料中的第二腿 `Yes` 对应 Gamma 的球队名，不能盲目拿 Yes/No 覆盖实际 Outcome 文案。

该样本仅验证迁移 Binary 腿。Native Binary、Neg Risk 及将来其他模块须使用各自身份映射；不能把结构化腿 PositionId 当作 Gamma `clobTokenIds`。Gamma 查询需处理开放与关闭市场，不以默认筛选返回空数组认定无市场。

本轮另读取 Combo markets 两页共 200 个条目，按目录给出的 market ID 精确查询 Gamma。module 2 市场 `3977566` 的两枚 `positionIds`、condition ID 与目录逐项一致，Gamma `negRisk=true`；module 1 市场 `4408301` 也一致。后者的 legacy 查询失败，不能进一步称其为已证明的 Native Binary 样本。证据见[精确市场对照](evidence/trader-sync-combo-metadata-precise.json)、[目录与查询记录](evidence/trader-sync-combo-metadata-check.json)。这些是元数据映射样本，不是新增成交验证。

因此可由目录建立 `PositionId → market ID` 索引，再精确查询 Gamma 并核对其 `positionIds`；该路线不依赖未文档化的 PositionId 查询参数。目录只有 `limit/cursor/exclude`，两页后仍有下一页，不声称已遍历完整目录。缓存须保留已经见过的关闭市场；全新实例未命中的历史腿仍可能缺资料。

[Combo 持仓](https://docs.polymarket.com/api-reference/core/get-user-combo-positions)默认会省略很小的开放余额，增量模式有可见性安全滞后；[Combo activity](https://docs.polymarket.com/api-reference/core/get-user-combo-activity)是用户生命周期记录，不能保证所有成交都有对应项。[Combo markets](https://docs.polymarket.com/api-reference/combo-markets/get-combo-markets)是活跃可组腿目录，也不是任意已成交组合的直接查询接口。缺失资料应保留精确 PositionId 与明确的不可用状态，不能丢弃已确认成交。

组合 YES 表示各腿条件的合取；组合 NO 是整个合取的补集。组合的 BUY/SELL 属于整份组合，不是每腿各自发生了一次相同方向交易。[官方组合语义](https://docs.polymarket.com/trading/positions/combinatorial)

## 公开身份与确认卡

### Profile URL

本次访问[官方 `@GCR` 页面](https://polymarket.com/@GCR)，canonical 为 `/@gcr`，页面公开数据明确给出 `proxyWallet=0x5c52d767c32cc18100c72d7471209d9618216058`；以该地址再次查询 Gamma public-profile，身份一致。

相同文字的 public-search 首条却是另一钱包的 `GCRClassic`，精确 `gcr` 在后续结果中。这直接说明搜索排序不能作为 Profile URL 身份映射。允许受限读取用户指向的官方 Profile 页面、提取明确身份并复核钱包；页面结构不受公开 API 契约保证，解析失效或身份冲突时必须失败，不能猜测或退回相似名称。

地址输入仍通过[官方 public-profile](https://docs.polymarket.com/api-reference/profiles/get-public-profile-by-wallet-address)查询。`proxyWallet` 是来源字段名，不能据此排除现行 Deposit、Safe 等实际交易钱包类型。样本没有覆盖全部 handle、重命名与本地化路径。

### Predictions 与其他统计

本次核对官方 Profile bundle、页面数据与 `/traded` 响应：页面的 `Predictions` 直接取 `/traded?user=...` 的 `traded` 原值，三处样本均为 2。现在可以在保留该消费证据的前提下使用此官方原值；不能自己重算或用另一个恰好相等的统计字段替代。

当前页面还提供 `user-stats` 的 `largestWin`、`joinDate` 等字段，以及持仓价值。各字段保存自己的来源、查询时间与可用性。字段不可用不填 0，返回真实 0 也不误标为缺失；不使用全量仓位自行拼出最大盈利或其他未核实口径。

### P/L 六区间

当前官方页面及其 bundle 支持 `1D/1W/1M/1Y/YTD/ALL`；本次页面公开数据取得 ALL 与 1W 的 `(t,p)` 序列。官方客户端请求 User PNL API 的基础区间为 `1D/1W/1M/ALL`，`1Y/YTD` 使用 ALL 数据按官方逻辑裁切；展示值还涉及首尾差值及可用历史长度，不能把末点直接用作所有区间的 Profit/Loss。

直接 User PNL GET 本次超时，六区间的完整请求链未逐项通过实时验证。后端可采用严格的官方页面/资料适配器；仅在所选区间的来源和展示规则已验证时返回可用数值与曲线，否则返回 `unavailable`。不为填满卡片推导、造零或使用排行榜冒充曲线。这沿用辅助资料缺失不阻止订阅的已有决定。

进一步逐段核对得到[六区间参数与显示规则](profile-pnl-contract-verification.md)：1Y 是 365 天，1M 裁切 30 天但历史不足判断使用 31 天；请求精度随 ALL 数据跨度改变。YTD 使用执行 JavaScript 的本地时区，参考时间 hook 和金额舍入函数尚未在当前 bundle 中闭合。适配结果必须记录实际时间、时区、请求参数和来源；无法核准某字段时独立标记 unavailable，不默认用 UTC、末点或自选舍入规则冒充官方显示。

官方页面与 bundle URL、观察时间、内容 SHA-256 保存在 Profile 证据文件中。它们是当前页面消费行为证据，不是平台对长期稳定接口的保证。

## 结论边界

- 已验证：当前主要 Exchange 归属与金额解释；当前 Combo 实现及 getLegs；两个迁移 Binary 腿的完整链上映射；一个精确 Profile URL；Predictions 当前消费来源。
- 有来源但尚未完成运行矩阵：其他 Combo 腿类型、全部收益区间、所有 Profile 路由、重组与重启故障链路。
- 未验证：长期稳定性、100 个活跃目标的实际吞吐、最终确认或通知端到端 P95/P99。

上述限制由设计定义明确失败/缺失行为，并由后续实现验收覆盖；不要求用户选择技术事实真假，也不把短时成功扩大为全部产品验收通过。

## PositionValue 独立补证

2026-09-10T17:03:15.616Z 对固定 GCR 钱包的公开 `/value?user=...` 做了一次受限 GET，返回 HTTP 200、唯一匹配钱包与原始数字 `0`。[完整记录与 SHA-256](evidence/trader-sync-position-value-2026-09-10.json)独立保存，不覆盖此前 SSR 样本。官方 [v1 总持仓价值契约](https://docs.polymarket.com/api-reference/core/get-total-value-of-a-users-positions)明确该数组的 `user`、`value` 口径；确认卡可以保存该直接原值与本次查询来源、时间。缺失、null、模式变化、钱包不匹配或请求失败仅使该字段 unavailable，不从持仓列表求和。

文档将 v1 标记为 Legacy；[v2](https://docs.polymarket.com/api-reference/wallet/get-portfolio-value)要求 bearer 认证并采用不同 envelope。本次实现采用仍在提供服务且已获批准的公开 v1 单一路径，不增加 v2 认证、兼容读取或失败 fallback。这份补证不证明 `/user-pnl` 全部实时区间可用。
