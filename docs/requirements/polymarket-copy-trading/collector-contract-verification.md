# Trader Sync RPC 过滤、确认与免费额度复核

> 复核日期：2026-09-10；实测时间：北京时间 17:03–17:05。
>
> 范围：只读 HTTP RPC、短时 WSS 订阅、官方计费文档；未修改业务源码或运行配置。
>
> 状态：技术研究。用户随后确认 Chainstack 开发入口、dRPC 手动切换及共享 WSS 路线，见[后端设计](../../design/trading/trader-sync-activity-alerts.md)；本报告的样本不代表完整运行验收。

## 结论

Chainstack 与 dRPC 免费端点均接受两合约地址及 100 个钱包 topic OR 过滤，并实际推送匹配日志。

100 项过滤包含三只真实样本钱包和 97 个人工构造地址；这验证过滤规模与实际匹配，**不是 100 个活跃目标的吞吐压测**。

Chainstack 的 Archive 限制也作用于旧区块的 `eth_getBlockByNumber`；不能仅因不补查历史成交就宣称所有恢复确认均不受限制。

本次已知旧区块的 `eth_getBlockByHash` 和已知交易的回执查询在两家均成功，为仅处理已接收事实提供可用候选路径。

## 原始证据

- [第一轮：既有样本钱包与旧候选查询](evidence/trader-sync-capacity-evidence-2026-09-10.json)。
- [第二轮：当前活跃样本、100 项过滤与 HTTP 核验](evidence/trader-sync-capacity-evidence-active-2026-09-10.json)。

实测复用已有端点清单中的 Chainstack 与 dRPC HTTP/WSS 地址；本报告不重复复制端点。

没有登录供应商控制台，未读取真实剩余额度、账单开关或资源路由；“免费节点”沿用用户此前说明。

## 第一轮：旧样本与链头推进

运行区间为北京时间 17:03:21–17:04:05；两家日志订阅实际观察均约 41.2 秒。

使用此前已验证的三只样本钱包加 97 个人工地址；两家均取得 logs 和 newHeads 订阅 ACK。

观察期间 Chainstack 收到 27 个新区块头，dRPC 收到 28 个；没有匹配成交日志。

因此第一轮只支持 100 项过滤接受、连接与链头推进，空结果不能证明真实成交匹配。

两家的 latest 与 finalized 均持续推进，各采样八次；全部临时订阅取消响应均为 true。

第一轮 dRPC 保存了 clean close/code 1000；Chainstack 的关闭回调未及时进入证据文件，不把它写成已有 clean-close 证据。验证进程已退出。

## 第二轮：实际匹配成交

运行区间为北京时间 17:04:56–17:05:28。

从当前三个区块的 167 条日志选取以下三只当时活跃钱包，再加入 97 个人工地址；样本选择仅为验证，不构成交易者推荐。

```text
0x7cf89836b082aab85b137d45bfc6b991cf2104ae
0xeb34b86ca3ca64eb3cee6c9a0cce668385df7268
0xc69bd5567b40ef4d11922eaa57e1f9be1c642076
```

过滤形态为两个核心 V2 Exchange 地址数组与 `topics: [OrderFilled签名, null, 100项钱包topic数组]`。

| 项目 | Chainstack | dRPC |
| --- | --- | --- |
| 实际订阅观察区间 | 17:04:58.541–17:05:26.683 | 17:04:58.778–17:05:25.070 |
| 实际观察时长 | 约 28.1 秒 | 约 26.3 秒 |
| 匹配成交日志 | 43 条 | 39 条 |
| 实际匹配钱包数 | 3 | 3 |
| 新区块推送 | 19 个 | 17 个 |
| latest 高度 | 93,550,710→93,550,729 | 93,550,711→93,550,727 |
| finalized 高度 | 93,550,708→93,550,725 | 93,550,708→93,550,723 |

每条日志的合约地址、事件签名和钱包 topic 均与过滤条件一致；三只真实样本均出现实际推送。

两家共同收到的 39 条日志在地址、区块哈希、交易哈希、日志索引、topics、data、removed 上完全一致。

各自第一条日志另与 HTTP 区块头及已知交易回执核验，区块哈希及回执中的原日志内容一致。

**观察窗口不同，43 条与 39 条的差异不能作为供应商漏推或性能比较。**本次也没有证明没有收到的消息一定不存在。

Chainstack 五次 finalized 查询中的一次返回 `TypeError: fetch failed`；证据未记录更深层错误原因，其余查询持续推进。不能写成“所有请求均成功”。

第二轮四次 `eth_unsubscribe` 全部返回 true；两条 WSS 均 code 1000、`wasClean=true`。没有留下后台订阅。

## 免费额度与计量复核

| 服务商 | 当前公开免费额度 | 本次相关计量 | 官方来源 |
| --- | --- | --- | --- |
| Chainstack Developer | 每月 3M RU、25 RPS | Global Node 近期调用、建立订阅和每条推送通常各 1 RU | [价格](https://chainstack.com/pricing/)、[RU](https://docs.chainstack.com/docs/request-units) |
| dRPC Free | 每 30 天 210M CU，从注册日期每 30 天重置 | 订阅、通知、区块与回执相关调用各 20 CU | [免费限制](https://drpc.org/docs/howitworks/ratelimiting)、[WSS](https://drpc.org/docs/pricing/subscriptions/evm)、[方法](https://drpc.org/docs/pricing/compute-units) |

dRPC 免费规则还列通常每 IP 每分钟 120,000 CU，区域高需求时可降至 50,400 CU；部分网络有自定义限制。免费请求最长 2 秒、日志最多 10,000 条、batch 最多三项。

这些分钟额度不能换算成任意一秒都保证通过的吞吐；月额度、突发限制、数据完整性与端到端时效是不同约束。

## 30 天候选预算

以下仅为参数算例：100 个不同目标，每目标每日 100 条真实匹配日志，持续 30 天。实际事件率尚未由“10 人规模”确定。

候选采集方式为仅常驻目标 logs WSS、共享 finalized 每 2 秒、共享 latest 健康查询每 10 秒，不永久订阅全链 newHeads。

| 项目 | 30 天次数 |
| --- | ---: |
| 匹配日志推送 | 300,000 |
| finalized，每 2 秒一次 | 1,296,000 |
| latest，每 10 秒一次 | 259,200 |
| 候选区块头，按每日志一次计 | 300,000 |
| 已知交易回执，按每日志一次计 | 300,000 |
| 合计 | **2,455,200** |

按上述近期调用口径，合计 **2.4552M RU，占 Chainstack 免费额度 81.84%**；或 **49.104M CU，占 dRPC 免费额度约 23.38%**。

固定的两类控制查询加日志为 1.8552M 次；上表再加入最多每日志各一次头部与回执查询，得到这个简化模型中的保守计算值。

区块头可按 blockHash、回执可按 txHash 共享去重，因此这些候选查询可能少于 600,000 次。不能把 2.4552M 当成真实系统月用量上限。

订阅建立、取消、重连、重复推送、查询重试、其他链上核验与同账户其他开发进程均未计入；实际事件率也可能高于算例。所有这些须另加，市场资料 API 和 Telegram 则有独立限制。

若 latest 与 finalized 都每 2 秒查询，仅二者就需 2.592M 次/月，加 300k 日志与 300k 区块查询为 **3.192M**，已超过 Chainstack 免费额度。

按短测约每 1.5 秒一个块推算，永久 newHeads 约 1.728M 条/月，加每 2 秒 finalized 与 300k 日志即 **3.324M**，尚未计候选查询。该区块间隔是观测算例，不是链速率保证。

## Archive 与已接收事实的恢复确认

本次针对仓库既有样本块 93,534,227，使用已记录的 blockHash 与 transactionHash 查询；没有枚举中断期间未接收的成交。

| 方法 | Chainstack 免费端点 | dRPC 免费端点 |
| --- | --- | --- |
| `eth_getBlockByNumber(oldHeight,false)` | HTTP 403、RPC -32002，Archive 拒绝 | 成功 |
| `eth_getBlockByHash(knownHash,false)` | 成功，原哈希及高度一致 | 成功 |
| `eth_getTransactionReceipt(knownTxHash)` | 成功，原区块定位一致 | 成功 |

[Chainstack RU 文档](https://docs.chainstack.com/docs/request-units)明确将 Polygon 距 tip 至少 127 块的 `eth_getBlockByNumber` 等方法归为 Archive；`eth_getBlockByHash` 与 `eth_getTransactionReceipt` 不在该受块龄影响列表。文档写其余 EVM 方法不因块龄改变 full 计量，本次实测支持方法间差异。

因此，不历史补查仍需处理“已收但尚未确认的候选在长时间停机后恢复”这一独立问题；仅持久保存原始日志并不能替代确认凭据。

建议候选路径为：共享 finalized 水位、查询已知交易的当前回执核对原日志及链定位、按已知 blockHash 取得结算时间；只处理原已收事实，不把回执中的其他日志新增为补查来源。

**查到旧 blockHash，加上 finalized 高度已超过它，不足以单独证明这个旧哈希仍属于规范链。**需要当前回执中的定位与原日志内容一致，近期时可再按高度交叉核验；该判断依赖可信、健康的 RPC 数据。

null、超时、403 不等于已证实孤块，应保留未确认状态并暴露错误，不静默确认或丢弃。已持久确认及已形成活动可以按原业务规则继续处理。

以上不是实际崩溃、重组或长期保留验证，也不承诺任意旧候选永远可查。[Chainstack 回执](https://docs.chainstack.com/reference/polygon-gettransactionreceipt)、[dRPC 回执](https://drpc.org/docs/polygon-api/transactionsinfo/eth_getTransactionReceipt)。

## 秒级基线建议及证据边界

可在目标所属全部过滤 ACK 后读取新鲜 latest 区块 H，再以 `max(下一数据库整秒, H.timestamp + 1秒)` 作为候选生效时间；生效时仍校验权限、revision 与未中断的 epoch，失败就重建实时边界。

短测看到链头 timestamp 偶尔领先本地请求起点约一秒。增加 H 的时间约束可排除 ACK 前已存在区块，避免仅依赖数据库取整；但本次没有运行完整基线算法，参数仍为建议。

WSS ACK、健康响应、连续头部不能证明日志永不静默漏推；不补历史约束下，已检测中断、未知完整性与恢复后新边界必须如实记录。短测不证明端到端通知时效、100 活跃目标承载、全部协议覆盖或长期稳定性。
