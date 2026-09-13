# 以太坊主网新发代币基础币调研

> 调研日期：2026-09-13。范围：Ethereum Mainnet，chain ID `1`。
>
> 用途：为钱包战绩的原本地计算方案选择基础币名单提供依据；用户当时选择 **WETH、USDC、USDT**。2026-09-13 已改为[接入 Nansen API](wallet-trading-performance.md)，原基础币筛选及本地结算不再作为接入开发要求。本文保留调研证据与当时选择，以下“首版”等表述均属于[原方案历史](wallet-trading-performance-local-calculation-history.md)，不证明 Nansen 采用相同范围，也不表示已实现。

## 调研建议与最终选择

**最终选择：WETH、USDC、USDT。** 下述调研阶段建议供理解选择依据；涉及增加其他基础币的建议未被纳入首版需求。

**首版优先考虑 WETH、USDC，USDT 可作为扩大覆盖范围的补充选项；如果覆盖 Uniswap V4，建议同时考虑原生 ETH。DAI、WBTC 暂不必为了凑齐“五大／六大基础币”而加入，BNB 暂不纳入。**

这个优先级结合下述有限样本、协议资料与预期覆盖范围，属于功能覆盖建议，不是全市场排名。WETH 的新池和新部署代币证据最直接；USDC 有近期池实例；USDT 有实例但本批池流动性很小。将 USDT 列为补充选项是为允许分析此类交易对的产品取舍，不能据此称其与 WETH 一样常用。

需要更正前面讨论中的技术表述：**Uniswap V2/V3 池由 ERC-20 资产组成，涉及 ETH 时核心池使用 WETH；Uniswap V4 可以直接以原生 ETH 建池，也可以使用 WETH。** V2/V3 也可以是 USDC/USDT 等不含 WETH 的池。“ETH 只能以 WETH 形成交易对”不能适用于所有协议。用户最终选择的三项名单不包含原生 ETH，这是业务范围选择。[Uniswap V2 官方说明](https://blog.uniswap.org/uniswap-v2)、[V3 以太坊部署文档](https://developers.uniswap.org/docs/protocols/v3/deployments/v3-ethereum-deployments)、[V4 白皮书第 4 节](https://app.uniswap.org/whitepaper-v4.pdf)

## 调研方法与范围

本次同时读取公开索引数据、直接查询主网工厂事件，并查阅协议官方资料。没有使用 Base、Solana 或 BNB Smart Chain 的池来推断以太坊主网。

| 证据 | 本次取得的范围 | 能说明什么 |
| --- | --- | --- |
| GeckoTerminal Ethereum `new_pools` | 100 个去重池，涉及 6 个协议版本 | 近期被收录的新池使用了哪些配对资产 |
| Ethereum RPC 工厂日志 | Uniswap V2/V3 官方工厂，1,001 个区块，18 个建池事件 | 在确定工厂和区块范围内，链上实际创建了哪些池 |
| DEX Screener 对照 | 指定 5 个池，核对两边币种地址及创建时间 | 两个索引服务对这些字段的记录是否一致 |
| 代币部署核验 | 对指定代币追溯创建交易 | 对应代币是否也在近期部署；不自动证明其首池 |
| 官方协议资料 | V2/V3/V4 资产支持方式及基础币身份 | 哪些配对在协议上成立，如何区分 ETH/WETH |

GeckoTerminal 数据在 **2026-09-13 07:28:53–07:28:56 UTC（北京时间 15:28:53–15:28:56）**取得；池创建时间落在 **2026-09-12 22:26:59 至 2026-09-13 07:20:23 UTC**。原计划读取 10 页，实际第 1、3、4、5、6 页成功，每页 20 条；第 2 页连接失败，第 7–10 页限流。因此这是有缺页的便利样本，不能称为该时段全部新池或连续最近 100 池。分页期间数据可能更新，已按池标识去重。[公开新池入口](https://www.geckoterminal.com/explore/new-crypto-pools/eth)、[原始响应及请求记录](research-data/2026-09-13-base-assets/geckoterminal-pages.json)

**新池不等于新币，也不等于官方首发池。** 旧代币可以创建新池，同一代币也可以创建多个费率或不同基础币的池。未经核验的代币不称为“刚发行”；建池也不自动证明有真实交易需求。报告按池两侧的合约地址识别基础资产，不凭名称、符号或索引网站的 base/quote 排序判断。

## 近期新池样本

以下各项以固定地址匹配池的任意一侧。本批没有两侧同时属于所列六项资产的池，因此各行互斥；20 个“其他”池单独保留。

| 配对资产 | 池数 | 对侧不同代币地址数 | 索引报告池流动性至少 1,000 USD 的池数 |
| --- | ---: | ---: | ---: |
| WETH | 30 | 30 | 9 |
| USDC | 30 | 13 | 4 |
| 原生 ETH | 12 | 9 | 6 |
| USDT | 6 | 4 | 0 |
| WBTC | 2 | 2 | 0 |
| DAI | 0 | 0 | 0 |
| 其他配对 | 20 | — | 17 |
| 合计 | 100 | 不跨行加总 | 36 |

来源：[完整池明细 CSV](research-data/2026-09-13-base-assets/new-pools.csv)、[按地址计算的统计结果](research-data/2026-09-13-base-assets/summary.json)。

这组数字需要结合以下事实理解：

- **WETH 与 USDC 都是 30 个池，不代表同样普遍。** WETH 对应 30 个不同代币地址；USDC 只有 13 个，其中同一个 BLUECHIP 地址贡献了 9 个 USDC 池。
- 100 个池里有 70 个来自 Uniswap V4、24 个来自 Uniswap V2、1 个来自 Uniswap V3、2 个来自 Balancer V3、1 个来自 DODO、2 个来自 Sushi V3。协议构成会影响观察结果。
- 6 个 USDT 池报告的流动性全部低于 17 USD；2 个 WBTC 池分别约为 24.89 USD、0.50 USD。它们证明配对存在，对“新币通常选择什么”只能提供很弱的证据。
- 1,000 USD 仅用于展示结果对微小流动性池的敏感程度，**不是已确认的产品过滤条件**。这些是索引服务估计值，未逐池核验储备、价格可信度或成交真实性，不使用金额加总衡量市场份额。
- 样本中有一个名称为 `USDT/HYPE` 的池，其“USDT”地址为 `0x21c3fcd4cf0e0e30458481ec70319805489e46f2`，不是 Tether 主网 USDT；已归入“其他”。这直接说明白名单必须按链和资产地址识别。[该池](https://www.geckoterminal.com/eth/pools/0xf3eec126b5a43f69ea17d0210f51afb5c666d6c7)、[Tether 官方主网地址](https://tether.to/en/supported-protocols/)

## 链上工厂事件核验

通过公开 Ethereum RPC 查询两个官方工厂：

- Uniswap V2：`0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f`，`PairCreated`。
- Uniswap V3：`0x1F98431c8aD98523631AE4a59f267346ea31F984`，`PoolCreated`。

区块范围 **25,966,019–25,967,019（含首尾）**，边界区块时间为 **2026-09-13 04:08:35–07:28:59 UTC**。返回 V2 17 个、V3 1 个建池事件，**18 个池全部包含规范 WETH 地址**，USDC、USDT、DAI、WBTC 均未出现。[原始工厂日志](research-data/2026-09-13-base-assets/factory-logs.json)、[工厂地址、事件签名与解码明细](research-data/2026-09-13-base-assets/chain-sample.json)、[边界区块响应](research-data/2026-09-13-base-assets/window-blocks.json)

其中 JWCN/WETH、LSK/WETH、$HLA/WETH 的创建时间与 GeckoTerminal 一致；它们属于交叉核验，不能再加到前述 100 池上形成“118 个独立样本”。这个窗口没有查询 V4，也没有覆盖其他 DEX，18/18 只适用于本次工厂和时间范围。

## 可以逐项核对的实例

表内时间均为 UTC；名称来自索引服务，仅作显示，不代表项目身份认证。

| 交易池 | 协议 | 池创建时间 | 核验与含义 |
| --- | --- | --- | --- |
| [JWCN/WETH](https://www.geckoterminal.com/eth/pools/0xe84d9ac2dfda02adba5587cb11750012855e7f44) | Uniswap V2 | 2026-09-13 07:20:23 | 索引、DEX Screener 与工厂事件一致；代币在 24 秒前部署 |
| [BERRY/WETH](https://www.geckoterminal.com/eth/pools/0x47b940dddf7a0ae54fb53e6cfb079f1488ecca2f) | Uniswap V2 | 2026-09-13 06:42:23 | 索引、DEX Screener 与工厂事件一致；代币约 14 小时 24 分前部署 |
| [SEND/USDC](https://www.geckoterminal.com/eth/pools/0x5fd8a51042972b2777e0480817d0c7478a5a3dc2863acfd400ffa50a5c81a638) | Uniswap V4 | 2026-09-13 04:44:47 | 两个索引服务字段一致；代币在约 2 天 9 小时前部署 |
| [kARGO/ETH](https://www.geckoterminal.com/eth/pools/0xbad43e58b62214d83c9c5508fe5e75e092fc109c67c775674682be3e0273022c) | Uniswap V4 | 2026-09-13 00:29:23 | 两个索引服务均标识原生 ETH；代币部署与池观察时间同秒 |
| [BLUECHIP/USDT](https://www.geckoterminal.com/eth/pools/0xad8a6463107576955b9b298c1e66af05b90a6a1af1892a5323fc9747a911ae48) | Uniswap V4 | 2026-09-13 00:57:35 | 两个索引服务字段一致；采集时流动性约 16.60 USD |
| [MCFUN/WBTC](https://www.geckoterminal.com/eth/pools/0xf361a08966b534a9aad1579033256ecd9e836437) | Balancer V3 | 2026-09-12 22:35:35 | 单一索引实例，采集时流动性约 24.89 USD |

前五项的独立索引响应保存在 [DEX Screener 对照数据](research-data/2026-09-13-base-assets/dexscreener-crosscheck.json)。V4 链接中的长十六进制标识是池 ID，不能当成 V2/V3 的独立池合约地址。

JWCN 的目标代币地址为 `0x3cc315d17bf63020ef5ed7d36dda8b3def1231d5`，部署于区块 25,966,974、2026-09-13 07:19:59 UTC；随后在区块 25,966,976 建立 WETH 池。这是本次可直接核对的“新部署代币很快与 WETH 建池”实例。[部署交易](https://etherscan.io/tx/0x7a22a4943cdcdc67830e8df74e68c3a4c6e82fa41c95030ecf0492f976332859)、[建池交易](https://etherscan.io/tx/0xb3bc07498b7e4901e03725f5644fc47d8ae88d122cfd75e15a823e2efeee2762)

BERRY 的目标代币地址为 `0x6f76a14a66a5b335cafad523217522fb3c5c1054`，部署于区块 25,962,478、2026-09-12 16:17:47 UTC；观察到的池于约 14 小时 24 分后创建。两项部署时间通过 Blockscout 的地址创建交易与交易返回的 `created_contract` 精确匹配核验；没有枚举代币全部历史池，故均不承诺为官方首池。[BERRY 部署交易](https://etherscan.io/tx/0x2a718b306d365c1bf45d095708e3a54e365f00912389bad400a5b04cf57a0af2)、[建池交易](https://etherscan.io/tx/0x8a370f6c5c0c8cca7bc9f23e9412e0d1e850625ce278d259347932c74f8b29cf)

另对 SEND 与 kARGO 核验了创建交易内的内部 CREATE 记录，并按目标地址匹配：SEND 部署于 2026-09-10 19:44:11 UTC，比表中 USDC 池早约 2 天 9 小时；kARGO 部署于 2026-09-13 00:29:23 UTC，与表中 ETH 池观察时间同秒。这支持原生 ETH 的同期新代币／新池实例，但本次未取得该 V4 池初始化交易哈希，不能断言部署与建池发生在同一笔交易。[SEND 部署交易](https://etherscan.io/tx/0xcbc59e75e793efcc8ec5ee1142bb1d15accc7021f4a461a82016dfbeb4467714)、[kARGO 部署交易](https://etherscan.io/tx/0x433713081fcd8afea93019ef6b8a7b9a295a49d8fcb0b8734bc1c60d637ab050)、[四项部署核验与原始证据](research-data/2026-09-13-base-assets/README.md)

## 对基础币名单的影响

| 资产 | 本次依据 | 用户确认后的首版选择 |
| --- | --- | --- |
| WETH | 当前新池、直接工厂日志及新部署代币实例均有证据 | 已确认纳入 |
| USDC | 当前样本有 30 个池、13 个对侧代币地址，包含独立索引核验实例 | 已确认纳入 |
| USDT | 有规范地址配对实例，但本批都很小 | 已确认纳入；不由此宣称与 WETH/USDC 同等常见 |
| 原生 ETH | 当前样本有 12 个 V4 池；官方协议明确支持 | 不作为首版基础币 |
| WBTC | 本批仅有 2 个微小流动性池 | 不作为首版基础币 |
| DAI | 本批没有观察到 | 不作为首版基础币；不能由零样本推断全链不存在 |
| BNB | 没有取得本范围内常见配对证据 | 不作为首版基础币；不能把 BSC 的 BNB/WBNB 配对直接套用到主网 |

BNB 是 BNB Smart Chain 的原生资产；Ethereum 上同名或封装形式必须另外确定具体资产身份。本次没有把任何同名 ERC-20 自动认定为 BNB。[BNB Chain 官方说明](https://docs.bnbchain.org/bnb-smart-chain/developers/quick-guide/)

调研阶段比较了 **WETH、USDC、USDT** 三项名单与增加原生 ETH 的四项名单，用户已选择前者。首版无需凑固定的五项或六项资产，也不因 USD 计量就只选择稳定币。

样本中的 kARGO/ETH、EDOG/ETH 等池不能凭 WETH 入选而自动纳入。另一方面，用户钱包支付 ETH、路由器封装 WETH 后进入 AAA/WETH 池，实际池资产仍是 WETH；“钱包支付币种”和“池的基础资产”应分别处理。多跳路径如何归属仍沿用需求中的待讨论状态。[V2 路由与封装说明](https://blog.uniswap.org/uniswap-v2)

资产身份核对如下；前三项进入已确认名单，原生 ETH 仅保留为调研对照：

| 资产 | Ethereum Mainnet 身份 | 核对来源 |
| --- | --- | --- |
| WETH | `0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2` | [Uniswap 主网部署文档](https://developers.uniswap.org/docs/protocols/v3/deployments/v3-ethereum-deployments) |
| USDC | `0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48` | [Circle 官方地址](https://developers.circle.com/stablecoins/usdc-contract-addresses) |
| USDT | `0xdac17f958d2ee523a2206206994597c13d831ec7` | [Tether 官方地址](https://tether.to/en/supported-protocols/) |
| 原生 ETH | 主网原生资产，无 ERC-20 合约；本次 V4 索引以全零地址表示 | [V4 白皮书](https://app.uniswap.org/whitepaper-v4.pdf)及上述 ETH 池响应 |

本次选择不改变已确认的“最新价格折算 USD、报价落库、按需缓存 10 分钟、移动加权平均成本”等规则，也未确定支持的 DEX、路由归属或流动性过滤条件。首版名单已确认；若另需精确的“新币首池占比”，还需先定义新币时间范围，再按部署交易和首次建池去重做更长时段的系统统计。
