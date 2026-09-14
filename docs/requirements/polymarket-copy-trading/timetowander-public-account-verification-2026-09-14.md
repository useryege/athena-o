# timetowander 公开交易账户只读校验

> 校验日期：2026-09-14。用户提供朋友的 [Polymarket 资料页](https://polymarket.com/@timetowander)，明确用于只读校验。本报告补充[手动交易平台契约校验](manual-trading-contract-verification.md)，不改变已确认的交易需求，也不代表获得该账户的交易授权。

## 结果

已核准资料页对应的公开交易账户，并取得一般市场（包含 Neg Risk）的持仓、买卖历史、领取历史样本及 Combo 数据。普通成交与活动接口在所测范围内一致，跨页读取可用。**默认筛选、零价值可赎回标记、小额余量及 Combo 字段差异都需要在设计中处理。**

本次只进行了公开 GET 请求，没有登录、签名、创建账户、授权、下单、领取或入金，也没有启动 ATHENA 服务。不能凭公开账户地址验证 Wallet 托管归属、Solana 钱包关联、账户签名类型或实际交易执行。

## 账户身份与证据

| 项目 | 结果 |
| --- | --- |
| 用户提供的资料页 | `https://polymarket.com/@timetowander` |
| 官网 canonical | 与用户提供的资料页一致 |
| 公开交易账户 | `0x58b3380f71bd6c706dd398a5e60bde55fa3a653c` |
| 身份复核 | 从该页唯一成功的 `/api/profile/userData` 数据取得账户，再用 Gamma public-profile 按地址查询；名称和 `proxyWallet` 一致。未依赖模糊搜索结果 |
| 资料创建时间 | 官方资料返回 `2025-04-24T19:10:49.089231Z`；不将其当作钱包部署时间 |
| 钱包类型与控制权 | 未验证。来源字段名 `proxyWallet` 不能证明它是某种特定智能钱包，也不能证明用户有权使用它签名 |

原始响应、来源与计算结果已入库：

- [请求清单](evidence/timetowander-public-account-2026-09-14/manifest.json)：27 次公开 GET 尝试，其中 25 次 HTTP 200；2 次发生 TLS EOF、未取得 HTTP 响应，随后对同一 URL 的只读重试均成功。失败记录仍保留。
- [资料页身份提取](evidence/timetowander-public-account-2026-09-14/profile-identity.json)：canonical、SSR 身份、访问时间和缓存信息。
- [结构化校验结果](evidence/timetowander-public-account-2026-09-14/observations.json)：数量、筛选、分页、逐笔对照和异常样本。
- [Combo 字段对照](evidence/timetowander-public-account-2026-09-14/combo-field-comparison.json)：按同一组合及腿 Position ID 对照两个接口。

请求清单的 `file` 指向压缩保存的原始响应。`sha256`／`bytes` 对应解压后的 HTTP 正文，`stored_sha256`／`stored_bytes` 对应 gzip 文件；文件可独立校验。资料页响应有缓存，且各接口并非同一时刻读取，因此本报告不声称获得原子一致的账户资产快照。

## 持仓与小额余量

一般市场数据来自 [Data API v2 持仓接口](https://docs.polymarket.com/api-reference/wallet/list-positions-for-a-user-or-market)。本次分别查询默认 OPEN、放宽筛选的 OPEN、CLOSED 和小页游标。

| 查询／对照 | 实际结果 | 设计含义 |
| --- | --- | --- |
| OPEN 查询 | 889 条，包含普通市场 508 条、Neg Risk 381 条；响应状态为 OPEN 的 15 条、REDEEMABLE 的 874 条；本查询无下一页 | 查询名称 OPEN 包含已结算但仍持有的记录，不等于全部都能继续市价交易 |
| 可赎回标记 | 上述 874 条 `redeemable=true` 的记录均 `current_value=0` | 不能仅据布尔标记显示正金额「领取」；需另核对结算结果和实际余额 |
| 默认与低门槛 OPEN | 默认参数、`filter_amount=0`、`filter_amount=0.000001` 三组返回相同的 889 个 token；低门槛组均设置 `filter_type=TOKENS, include_archived=true` | 不能承诺调低此参数就能取得本钱包的全部小额余量 |
| CLOSED 查询 | 769 条，其中 504 条 `current_size=0`；另外 265 条仍返回正份额，范围为 0.0001～0.091 份，其中 104 条还有正的 `current_value` | CLOSED 不等于持仓份额为零；小额余量必须保留可见，不能直接标为完全清仓 |
| 小额针对性核对 | 一条 CLOSED 样本返回 0.001 份；按相同 condition 查询 OPEN，并设置 `filter_amount=0.000001` 仍为空 | 本次余量缺口可复现；这不是把响应缺失自行推算为余额 |
| 持仓分页 | 3 条一页，保留原钱包及筛选取得第二页；两页共 6 个 token 与一次读取结果的前 6 个顺序一致 | 验证了此次稳定排序和保留查询条件的分页方式 |

上述余额和状态均为公开索引返回，未做链上余额核验。265 条正份额 CLOSED 与所测 OPEN 集合没有 token 重叠；这支持在后续设计中联合核对两个集合，但不能推导任意市场或任意钱包完整覆盖。此次没有归档市场样本，inactive 覆盖仍未证明。

后续持仓展示应依据实际份额、市场状态与操作资格组织数据。需让用户找到可操作的仓位，同时保留已结算零价值记录及小额余量；不能因为列表较长而隐去已确认要展示的资产。

## 买卖历史、分页与领取记录

为避免账户持续交易影响对照，普通买卖查询使用固定结束时间。控制窗口为 **2026-09-13 20:33:17 至 2026-09-14 20:33:17（北京时间）**，即连续 24 小时。

| 查询 | 返回数量 | 校验结果 |
| --- | --- | --- |
| `/v2/trades`，`taker_only=false` | 740 条：BUY 587 条、SELL 153 条 | 该窗口分页耗尽 |
| 相同条件，改为 `taker_only=true` | 34 条 | 该窗口分页耗尽，返回集合属于上述 740 条 |
| `/v2/activity`，`type=TRADE`，相同起止时间 | 740 条 | 按交易哈希、token、方向、数量、价格和时间的多重集合对照，与全部角色成交一致 |

两组 trades 请求均显式使用 `filter_type=TOKENS, filter_amount=0.000001`，只有成交角色参数不同。**如果直接沿用默认只查 taker 的查询，本窗口会缺少其余 706 条成交记录。** 这进一步支持已确认的完整钱包历史范围必须包括 maker 与 taker。[官方成交参数](https://docs.polymarket.com/api-reference/feeds/list-trades)

另外使用 `start=1` 抽取普通成交两页，共 2,000 条，活动接口也读取相同两页；逐页多重集合均一致，在上述比较键下未见跨页重复。第二页仍有后续游标，因此 **2,000 条只是历史样本，不是该账户全部历史数量**。样本时间范围为 2026-09-11 20:56:21 至 2026-09-14 12:31:27 UTC。

单独按时间正序查询活动，取得最早返回成交 `2025-04-24T20:34:20Z`。这证明可访问近期样本以前的记录，不证明从该时刻至今的每一笔已经抓取或完整核对，也没有覆盖三年前边界。

领取活动另取得 10 条 `REDEEM` 样本且还有下一页，可用于验证历史中的独立操作类型。它们不能当作市价卖出，读取既有领取记录也不等于 ATHENA 已能执行领取。[活动接口](https://docs.polymarket.com/api-reference/feeds/list-account-activity)

本次多重集合的比较键仅用于这些公开样本的交叉检查，不作为生产系统的唯一成交 ID 或去重算法；同一链上交易可包含多个成交事实。

## Combo 展示与字段差异

[Combo 持仓](https://docs.polymarket.com/api-reference/wallet/list-combo-positions)取得 26 条仍有正份额的记录，状态均为 `RESOLVED_LOSS`，`redeemable=false`。[Combo 活动](https://docs.polymarket.com/api-reference/feeds/list-combo-activity)取得 35 条，其中 SPLIT 34 条、REDEEM 1 条；两组查询均无下一页。它们可用于已确认的只读展示，但 SPLIT 金额不是买入本金，不能拿 35 条生命周期记录冒充全部 Combo 买卖历史。

对同一组合及腿 Position ID，持仓与活动接口的 `leg_condition_id` 在 132 次对照中不同，涉及 103 个不同组合／腿键。重新读取前 3 条后仍有 14 个共同腿键复现该差异。示例市场为 Gamma `3945919`：

| 来源 | 同一腿的 condition 与展示标签 |
| --- | --- |
| Combo 持仓 | `0x0114debe730e3d1317aec5cf071d430ebd0000000000000000000000000000`；标签 `Arthur Fils` |
| Combo 活动 | `0x09ce36db9ba15107e04b0608a12c8a0914debe730e3d1317aec5cf071d430ebd`；标签 `No` |
| Gamma 市场详情 | `conditionId` 与活动接口一致；市场 Outcome 为 `Stefanos Tsitsipas`／`Arthur Fils` |

本次不同字段字符串长度分别为 64 与 66（均含 `0x`）。这里只记录可复现的不一致及单个 Gamma 样本的复核，不推断服务端内部原因，也不将一个样本扩展为全平台哪一个接口始终正确。

设计不能直接把两个接口中的同名字段当作同一个市场标识，更不能靠补零、截断或标签文字猜测市场。保留原始值，通过核准的市场映射形成展示；未解决的关联明确标记资料待核对。初版仍不执行 Combo 交易或领取。

## 对后续设计的补充

- 一般持仓必须核对 OPEN 与 CLOSED 中的实际余量，市价卖出前另校验真实可卖余额；CLOSED、REDEEMABLE 均不直接决定操作结果。
- 自有钱包历史查询显式包含 maker 与 taker，保留时间和筛选参数；跨页对照与缺口状态必须可追踪。
- Combo 的数据类型和市场关联单独处理；保留持仓及历史，不把生命周期当买卖，也不绕过执行范围限制。
- 资料页地址可用于只读数据接入，不能用它证明用户 Wallet 与官网交易账户的控制关系。账户启用、官网登录与入金对应关系、签名、市价订单、实际扣费和领取恢复仍需后续验证。

以上结果已补入[总体设计提案](../../design/trading/polymarket-manual-trading.md)。这是数据契约校验，不是正式页面、真实交易或完整历史验收。

[返回平台校验](manual-trading-contract-verification.md) · [返回交易需求](copy-trading.md)
