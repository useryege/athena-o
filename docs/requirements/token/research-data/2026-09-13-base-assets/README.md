# 基础币调研数据

关联报告：[以太坊主网新发代币基础币调研](../../new-token-base-assets-research.md)。采集日期为 2026-09-13，固定 Ethereum Mainnet（chain ID 1）。这是一次性研究附件，不是产品数据源配置或自动刷新任务。

## 文件与来源

| 文件 | 内容与来源 |
| --- | --- |
| [geckoterminal-pages.json](geckoterminal-pages.json) | `https://api.geckoterminal.com/api/v2/networks/eth/new_pools?include=base_token,quote_token,dex&page=N` 的分页响应，保留请求 URL、取得时间及失败信息 |
| [new-pools.csv](new-pools.csv) | 成功页按池 ID 去重后的 100 行；币种地址由响应的 `included` 按 relationship ID 关联 |
| [summary.json](summary.json) | 按池两侧地址匹配 WETH、USDC、USDT、DAI、WBTC 和原生 ETH 的统计；原生 ETH 全零地址是数据源表示方式，不是 ERC-20 |
| [factory-logs.json](factory-logs.json) | `https://ethereum-rpc.publicnode.com` 返回的 Uniswap V2/V3 工厂事件；块范围 25,966,019–25,967,019 |
| [window-blocks.json](window-blocks.json) | 同一 RPC 的窗口边界区块，用于时间核验 |
| [chain-sample.json](chain-sample.json) | RPC 地址、工厂、topic、范围和 18 个解码事件，便于按区块和交易重查 |
| [dexscreener-crosscheck.json](dexscreener-crosscheck.json) | 指定五池的 DEX Screener `/latest/dex/pairs/ethereum/{pairIds}` 响应及请求 URL、时间 |
| [token-creation-checks.json](token-creation-checks.json) | JWCN、BERRY、SEND、kARGO 的代币部署地址、区块、时间与对应池观察时间 |

部署核验来源为 `https://eth.blockscout.com/api/v2`：先查询 `/addresses/{address}` 的 `creation_transaction_hash`，再查询 `/transactions/{hash}`；内部部署另外查询 `/transactions/{hash}/internal-transactions` 并精确匹配目标合约。

| 代币 | 地址响应 | 创建交易响应 | 内部创建证据 |
| --- | --- | --- | --- |
| JWCN | [地址](blockscout_jwcn_address.json) | [交易](blockscout_jwcn_creation_tx.json) | 顶层 `created_contract` 匹配 |
| BERRY | [地址](blockscout_berry_address.json) | [交易](blockscout_berry_creation_tx.json) | 顶层 `created_contract` 匹配 |
| SEND | [地址](blockscout_send_address.json) | [交易](blockscout_send_creation_tx.json) | [内部 CREATE](blockscout_send_creation_internal.json) |
| kARGO | [地址](blockscout_kargo_address.json) | [交易](blockscout_kargo_creation_tx.json) | [内部 CREATE](blockscout_kargo_creation_internal.json) |

## 复算口径

1. GeckoTerminal 成功页按 `data.id` 去重，不把缺页当作空页。V4 ID 是池标识，不是独立池合约地址。
2. 对两侧 token 的 `address` 转为小写，匹配 `summary.json` 中的六个固定资产身份，不依赖符号。命中某资产的池数按池计；对侧不同代币数按对侧地址去重。
3. 流动性敏感性列使用采集时 `reserve_in_usd >= 1000`。无值不计入达标数；该门槛仅是报告视角，不是产品规则或资产真实性保证。
4. 工厂日志仅统计对应工厂和事件签名，按日志取得池的两侧地址后匹配基础币。结果与 GeckoTerminal 可能重合，两组样本不直接相加。
5. DEX Screener 对照匹配链、池标识、双方代币地址和 `pairCreatedAt`；没有将两个索引服务的一致视作已验证全部链上状态。
6. 部署核验只证明合约创建时间及本次池观察时间的关系，没有穷举首池、确认发行人身份或验证交易可获利性。

保留原始响应用于核对；重新请求动态 API 可能得到不同的池、价格与流动性。报告数值以本目录快照为准。
