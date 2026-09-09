# Trader Sync 链上成交数据可行性调研

> 调研日期：2026-09-09
>
> 性质：外部数据能力与只读样本研究；不是已确认技术设计，不授权实现。
>
> 关联需求：[Activity Alerts](target-trade-monitoring-notifications.md)
>
> 样本证据：[RPC 请求、响应与市场映射](evidence/onchain-trade-data-sample.json)

## 结论与证据边界

**当前普通 CTF 和 Neg Risk 两种 V2 Exchange 的日志，足以识别“哪个实际交易钱包，在交易哪个 Token、什么方向、多少数量”。目标钱包是可过滤的 indexed 字段，节点可以直接返回指定目标集合的事件，无需 ATHENA 接收全链或全站所有用户的交易。Token 对应的市场、Outcome 和页面链接可以由公开 API 补齐，本次已经用真实成交完成该链路。**

这是部署源码、公开 RPC 和实际市场查询共同支持的可行性结论。完成核心识别的难度为中等；持续可靠运行还需要处理区块连续性、确认状态、重连补查、去重、目标集合变化与市场资料缺失，工程难度为中等偏高。此难度判断不等于开发工期或性能承诺。

该结论不应扩展成“现有产品已经实现”或“全部 Polymarket TRADE 已完成覆盖”。官方还有 Combos Exchange；其源码与本次实际日志均支持事件识别和账户归属，一个组合样本也已通过关联生命周期记录补齐两个市场腿，但通用的组合映射方法与展示规则仍需核实。高频容量、长期稳定性、端到端时延和费用均未由本次抽样证明。

## 本次实际做了什么

- 从官方 Data API 读取 30 条公开成交，仅用于选择交易哈希和交叉核对，不将其预先确认为生产监控依赖。
- 读取 17 个实际链上交易回执，涉及普通 CTF 与 Neg Risk 合约，以及买入、卖出、被动 Maker 和主动 Taker。
- 使用 Polygon 官方文档列出的 PublicNode 与 dRPC 公共节点，核对 Chain ID 为 137；分别读取日志与回执。选定补充样本在两个节点的原始日志完全相同。
- 对区块 `93506126` 读取两类交易所全部 `OrderFilled`，再执行单目标及多目标过滤，比较过滤结果与完整响应中的对应记录。
- 对 `93506106` 至 `93506126` 的 Neg Risk 主动成交作有界补充查询，取得 122 条主动成交事件，其中 26 条为 SELL；另取一个实际 SELL 回执复核。
- 对 `93505900` 至 `93506126` 的 Combos Exchange 作一次有界查询，取得 111 条 `OrderFilled`，涉及 46 笔交易和 72 个订单所属钱包；其中 46 条为主动订单自身事件。此项没有逐笔获取完整回执。
- 查询两个 Token 的市场归属、实际 Outcome、标题和 event slug，并核对目标公开 Profile 返回的钱包。
- 对一个实际 Combo 主动成交，执行四次公开 GET 查询，通过同交易、同 PositionId 的生命周期记录补齐两条市场腿；没有将该辅助记录算成额外成交。
- 查询链上已最终确认高度，以及现行抵押代币的 `decimals` 和 `symbol`。这不构成最终确认延迟测量。

以上均为只读外部数据研究，没有修改业务源码，没有运行单元、集成或端到端测试，也没有部署、发单或创建订阅。

## 部署合约与源码依据

[官方合约清单](https://docs.polymarket.com/resources/contracts)列出的两种核心交易所为：

| 类型 | Polygon 地址 |
| --- | --- |
| CTF Exchange V2 | `0xE111180000d2663C0091e4f400237545B87B996B` |
| Neg Risk CTF Exchange V2 | `0xe2222d279d744050d28e00520010520000310F59` |

已读取两个地址的 PolygonScan 已验证 ABI 与源码，将 `CTFExchange.sol`、`Trading.sol`、`Events.sol`、`ITrading.sol`、`Structs.sol` 与官方固定提交 `ccc0596074f4dfd62c944fbca4de252893b82b4b` 比较，五个文件均逐字一致。该核对依赖浏览器的已验证源码记录，本次没有自行编译并重现部署字节码。

- [普通 CTF 部署源码](https://polygonscan.com/address/0xE111180000d2663C0091e4f400237545B87B996B#code)
- [Neg Risk 部署源码](https://polygonscan.com/address/0xe2222d279d744050d28e00520010520000310F59#code)
- [固定版本事件声明](https://github.com/Polymarket/ctf-exchange-v2/blob/ccc0596074f4dfd62c944fbca4de252893b82b4b/src/exchange/interfaces/ITrading.sol)
- [固定版本结算逻辑](https://github.com/Polymarket/ctf-exchange-v2/blob/ccc0596074f4dfd62c944fbca4de252893b82b4b/src/exchange/mixins/Trading.sol)
- [固定版本实际发事件逻辑](https://github.com/Polymarket/ctf-exchange-v2/blob/ccc0596074f4dfd62c944fbca4de252893b82b4b/src/exchange/mixins/Events.sol)

## 日志提供哪些信息

| 信息 | 数据位置 | 能力与限制 |
| --- | --- | --- |
| 来源合约 | `log.address` | 必须校验受认可的 Exchange 地址；不能只相信事件签名。 |
| 事件类型 | `topics[0]` | V2 `OrderFilled` 的签名哈希。 |
| 订单哈希 | `topics[1]` | 能关联订单，但同一订单可以多次部分成交，不能单独作为成交去重键。 |
| 本订单实际资金钱包 | `topics[2]`，ABI 名称 `maker` | 可以在 RPC 侧过滤，是本次账户归属的关键。 |
| 对手方地址 | `topics[3]`，ABI 名称 `taker` | 可能是另一账户或 Exchange；地址出现于此不应额外算一条自身成交。 |
| 方向 | `data` 中的 `side` | 该订单视角的 BUY / SELL。 |
| 交易标的 | `data` 中的 `tokenId` | 可查询对应市场和实际 Outcome；数值应按整数或字符串保存。 |
| 成交数量与金额 | `makerAmountFilled`、`takerAmountFilled` | 根据 side 区分抵押代币数量和 Outcome Token 数量，不能交换含义。 |
| 手续费 | `fee` | 与成交金额分别解释，不把数量比值直接当成含费成本。 |
| 链上定位 | 区块号、区块哈希、交易哈希、日志索引 | 可定位、补查并判断重复读取；仍需处理规范链变化。 |
| 时间 | 查询对应区块头 | 获得的是区块结算时间，日志不提供精确的链下撮合时刻。 |
| 市场标题、Outcome 文案、页面链接 | 不在成交日志中 | 通过 Polymarket 公开市场资料补齐。 |

V2 的事件签名为：

```text
OrderFilled(bytes32,address,address,uint8,uint256,uint256,uint256,uint256,bytes32,bytes32)
0xd543adfd945773f1a62f74f0ee55a5e3b9b1a28262980ba90b1a89f2ea84d8ee
```

## 为什么能同时识别 Maker 和 Taker

**事件的 `maker` 字段代表这张订单所属的资金钱包，不等于订单簿中的流动性 Maker 角色。**

部署源码在被动订单结算时，将 `makerOrder.maker` 写入此字段；主动吃单订单结算时，同样将 `takerOrder.maker` 写入此字段，并生成其自身的 `OrderFilled`。两种 V2 合约的直接买卖、批量铸造和合并结算分支均遵循这一归属方式。

因此，对这两种合约，按 `topics[2] = 目标实际资金钱包` 可以同时发现目标的被动成交和主动成交。仅为了识别目标交易的 Token、方向及自身结算数量，无需再把 `topics[3]` 命中的对手方事件当成新活动。

外层交易由 operator 提交，`transaction.from` 不能用于代替订单资金钱包。公开 Profile 或用户输入的地址必须先解析到实际交易钱包；钱包地址能够识别的是链上账户，不能据此推断其背后的自然人或该人的其他钱包。[官方钱包说明](https://docs.polymarket.com/trading/wallets-auth)

## 从海量事件中过滤目标的实际结果

在区块 `93506126`，区块时间为 `2026-09-09 14:30:22 UTC`，执行相同范围的查询：

| 查询条件 | 返回数量 | 核对结果 |
| --- | ---: | --- |
| 两种 V2 Exchange + `OrderFilled` | 116 | 作为该区块对照集，包含两种合约。 |
| 再限定 `topics[2]` 为一个目标钱包 | 2 | 与对照集中该钱包的两条原始日志完全相同。 |
| `topics[2]` 设置为四个钱包的 OR 集合 | 7 | 与对照集中这四个钱包的七条原始日志完全相同。 |

单目标为 `0x4ebc2722adc772bde8680792d0a6fdf15499a33d`。实际请求形态如下，完整请求和响应已保存至证据文件：

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_getLogs",
  "params": [{
    "fromBlock": "0x592ca4e",
    "toBlock": "0x592ca4e",
    "address": [
      "0xE111180000d2663C0091e4f400237545B87B996B",
      "0xe2222d279d744050d28e00520010520000310F59"
    ],
    "topics": [
      "0xd543adfd945773f1a62f74f0ee55a5e3b9b1a28262980ba90b1a89f2ea84d8ee",
      null,
      "0x0000000000000000000000004ebc2722adc772bde8680792d0a6fdf15499a33d"
    ]
  }]
}
```

其中 `address` 是发日志的交易所合约，目标钱包放在 `topics[2]`；将钱包直接填入 `address` 会筛错对象。`null` 表示不限定订单哈希。多个目标可放在同一个 topic 位置的数组中，表达 OR；不同 topic 位置之间是 AND。[JSON-RPC 过滤语义](https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_newfilter)

这证明我们可以在节点返回数据前过滤目标，而不是在 ATHENA 内遍历所有用户。它不证明任意大的目标数组都被服务商接受，也不意味着节点处理没有成本。目标分组规模、区块范围限制、连接数、查询配额和费用仍需按选用的 RPC 服务核实。

WebSocket 的 `eth_subscribe("logs", filter)` 也接受合约及 topic 过滤；本次实际调用验证的是 HTTP `eth_getLogs`。尚未进行长期 WebSocket 连接、断线重连或吞吐观察，不能将二者的证据混为一谈。[Geth 日志订阅说明](https://geth.ethereum.org/docs/interacting-with-geth/rpc/pubsub)

## 一个完整的真实成交例子

[交易 `0xd3a257…b242f`](https://polygonscan.com/tx/0xd3a257f1b9ea8cc8ef806c76ad274d283e1b3452c278ef86548eee250d0b242f)中，外层提交者为 `0xd41ce901ebe93c74e3e3cb3a1e145395331da879`，目标实际交易钱包为 `0x4ebc2722adc772bde8680792d0a6fdf15499a33d`。

该交易包含四条 `OrderFilled`：三条属于对手方订单，一条属于目标自身的主动订单。目标自身记录的日志索引为 `1238`，方向为 BUY：

| 字段 | 链上或 API 查询结果 |
| --- | --- |
| Token ID | `23804098144125113012541486155890795152578817099470541864882745171440736854422` |
| 目标本次结算数量 | `911000000` 个原始单位，按本次样本单位为 911 份 |
| 本次结算金额 | `52749000` 个原始抵押代币单位，即 52.749 pUSD |
| 费用 | `2484730` 个原始抵押代币单位，即 2.48473 pUSD，独立于上述金额 |
| Condition ID | `0x84f3b2402f5471ff48cd85a43fa5ff8b144afc5b6c3afcc8ac7437578aba6774` |
| 市场 | Counter-Strike: PARIVISION vs FURIA (BO3) - FISSURE PLAYGROUND Group B |
| Outcome | PARIVISION |
| Event slug | `cs2-prv-furia-2026-09-09` |

通过 `/markets-by-token/{tokenId}` 查到 Condition ID，再以 Gamma `/markets?clob_token_ids=...` 返回的 `clobTokenIds` 与 `outcomes` 对应关系，识别出 PARIVISION，并取得标题和 event slug。这两个 Token 查询及对应 Gamma 查询均返回 HTTP 200。[Token 对应市场接口](https://docs.polymarket.com/api-reference/markets/get-market-by-token)

这条记录证明的是本次结算，不是整张原始订单的总量。目标在三条对手方日志中也出现在 `taker` 字段；若把“任一字段出现目标”都算自身新成交，会错误增加提醒并混淆 Token 和方向。

另外，最初 30 条 Data API 样本，均能按“交易哈希 + 资金钱包 + Token + 方向”匹配到恰好一条所读回执中的自身成交事件。该结果只证明这组样本的账户和标的对应关系，不证明所有数据源记录在数量、价格、时间和粒度上永远一一对应。

## Combos 的补充样本与剩余限制

对官方 Combos Exchange 代理 `0xe3333700cA9d93003F00f0F71f8515005F6c00Aa` 的有界日志查询，取得 111 条实际 `OrderFilled`。其实现合约已验证源码同样将自身订单资金钱包写入 `topics[2]`；发日志地址是代理地址，不能用实现地址代替过滤条件。[官方合约清单](https://docs.polymarket.com/resources/contracts)

其中，[交易 `0x1535ca…664a5`](https://polygonscan.com/tx/0x1535ca2707356702fe919176eecf4f3ff5b2ad8be0a4e9b3ddad5a67030664a5)的主动订单归属钱包 `0x237419de121f3aecf3e1e0a613ecb9dc82b9a426`，PositionId 为 `1677415912060011761505225895152832125576889787899132551452768910625300545536`。

普通 CLOB Token 查询返回 404，Gamma Token 查询返回空数组。Combo 持仓第一页未命中且还有下一页，因此不能认定该持仓不存在。通过官方 `/v1/activity/combos?user=...&limit=100` 找到了同交易、同 PositionId 的 `PositionsSplit` 记录，返回两条 legs：

| 组合中的市场 | Outcome |
| --- | --- |
| Sporting CP vs. Galatasaray SK：Both Teams to Score | Yes |
| VfB Stuttgart vs. Viking FK：VfB Stuttgart (-1.5) | Yes |

这证明本次组合样本能够补齐关联市场。`PositionsSplit` 仅作为资料补全证据，不是第二个新成交，也不能把组合腿各自表述为目标分别下了独立订单。本次没有证明每笔 Combo 成交都伴随同交易的生命周期记录；通用的 PositionId 到组合资料查询、完整分页和一条组合活动的展示仍需继续对齐。[官方 Combo 活动接口](https://docs.polymarket.com/api-reference/core/get-user-combo-activity)

## 需要避免的解析错误

- **把事件名中的 maker 当成被动交易者。** 主动 Taker 的自身事件也使用这个字段保存自己的资金钱包。
- **同时累加自身事件、对手方事件和 `OrdersMatched`。** 后者再次描述主动订单结算，不应另生成通知。
- **只按订单哈希去重。** 同一订单可以多次部分成交；日志定位和业务事件含义都需要保留。
- **根据 Token 流入流出猜测买卖。** 双 BUY、双 SELL、负风险转换等资产流动不能统一解释为简单的一买一卖；按对应成交事件的自身 side 解释。
- **把 primary/secondary 固定当成 Yes/No。** 另一实际样本中，`secondary_token_id` 对应的是 MOUZ；应查询真实 Outcome 文案。
- **盲信活动记录的显示辅助字段。** 本次公开样本出现 `outcomeIndex=999`、空 `eventSlug`；市场 API 能补齐，不应伪造默认值。
- **把 CLOB 响应的 `v:"v1"` 当成部署合约版本。** 本次该字段与 V2 实际结算并存，其公开语义未查明；ABI 选择应以日志来源合约及已核验版本为依据。
- **将链上 pUSD 金额直接标成 USDC。** 本次链上读取现行抵押代币返回 `symbol=pUSD`、`decimals=6`；现有需求中的金额单位需在数据契约对齐时核准，不能因 API 保留 `usdcSize` 字段名而混同。[官方抵押代币说明](https://docs.polymarket.com/concepts/pusd)

## 实现难度判断

| 工作 | 难度判断 | 依据 |
| --- | --- | --- |
| 两种 V2 合约按目标过滤 | 低至中 | indexed 字段明确，单目标和多目标查询均有真实结果。 |
| 钱包、Token、方向、结算量解析 | 中 | 数据足够，需要正确处理自身订单视角、单位和费用。 |
| Token 到普通市场资料映射 | 低至中 | 已完成真实查询；仍需缺失、缓存更新与限流处理。 |
| 部分成交、重复读取和重复语义处理 | 中 | 数据定位明确，但一条源记录和一条产品活动的映射不能想当然。 |
| 持续运行、确认状态与断线补查 | 中等偏高 | 需要持久进度、区块连续性检查、可观测故障及补查边界。 |
| Combos 全范围支持 | 中等偏高，范围仍需细化 | 事件与单笔市场映射样本可行；不能依赖每笔交易必有同交易生命周期记录，产品表达未对齐。 |

仓库已有 [Polygon 增量日志扫描](../../../internal/managedoo/log_sync.go)和 [BSC finalized 扫描](../../../internal/bscswap/scanner.go)可参考。前者处理预言机事件，后者固定 BSC 与 Swap 语义；都不能直接启用来宣称 Trader Sync 已经具备上述能力。

持续运行可以采用按区块增量查询，也可以采用日志推送并补读缺失区间；这一传输方式选择不改变本次已经证实的账户过滤能力。订阅只推送当前事件，不能代替历史补查；收到未最终确认日志也不等于已经不可逆。[订阅限制](https://geth.ethereum.org/docs/interacting-with-geth/rpc/pubsub)、[Polygon 最终确认](https://docs.polygon.technology/pos/concepts/finality/finality)

## 对现有需求的影响与后续边界

本次没有改变已确认业务条目，也没有锁定主数据源、轮询间隔、组件、表结构或实现方式。研究证明链上数据不是核心能力障碍，但正式采用前仍需处理：

1. **活动粒度与时间。** 链上自身 Taker 记录可能覆盖多个对手方成交，区块时间也不是链下撮合时间；需与现有“每条公开成交记录独立触发”和订阅生效边界对齐。
2. **实际账户类型与金额单位。** 公开接口仍使用 `proxyWallet` 字段名，但现行平台已有 Deposit、Proxy、Safe 等钱包类型；目标身份应按真实交易钱包核准。现行 pUSD 的展示口径也需与需求对齐。
3. **全部 TRADE 的协议覆盖。** 官方 `/activity` 存在 `isCombo` 活动；两个核心 V2 合约并不等同于全部交易范围。Combos 使用独立 Exchange 代理，需覆盖对应事件与组合腿数据，不能静默忽略。[官方 Data OpenAPI](https://docs.polymarket.com/api-spec/data-openapi.yaml)、[Combos 请求流程](https://docs.polymarket.com/trading/combos/requesters)
4. **性能与可靠性。** 尚未核准最大目标数、RPC 配额、长时间运行效果、端到端时效或源数据异常覆盖。不能把单区块过滤成功当成已达到产品全部验收要求。

这些发现支持继续推进链上数据路线的评估；若对产品活动粒度、范围或时间定义产生实质影响，应先写回关联需求并确认。整份需求仍为 `讨论中`，本报告不改变其状态。
