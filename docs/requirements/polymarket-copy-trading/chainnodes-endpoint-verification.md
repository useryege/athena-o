# Chainnodes Polygon 端点可用性验证

> 验证时间：2026-09-10 09:47–09:50（北京时间）。
>
> 范围：用户明确要求验证提供的 HTTP 与 WebSocket 端点；仅执行只读 RPC 与短时订阅。
>
> 原始证据：[请求、响应、订阅记录与对照节点](evidence/chainnodes-endpoint-verification-2026-09-10.json)。

## 结论

**两个端点均可连接，返回 Polygon 主网 Chain ID 137；但此端点当前提供的数据严重滞后，不能用于 Trader Sync 实时监控。**

HTTP 首次读取与约三分钟后的复查，以及 WebSocket 的高度查询，均返回区块 `83,521,701`。这个区块的时间为 `2026-02-27 03:37:10 UTC`。同期 dRPC 与 PublicNode 返回同一个区块 `93,533,253`，时间为 `2026-09-10 01:48:33 UTC`，区块哈希也一致。

这次对照的高度差为 **10,011,552 块**，区块时间差约 **195 天**。近期区块与成交日志请求明确报缺失数据或上游未同步。连接成功、HTTP 200 或取得订阅 ID 均不足以判定业务可用。

**195 天是返回数据与同期链上区块的时间差，不是已证实的故障持续时间。**本项目首次观测到该端点异常的时间为 2026-09-10 09:47（北京时间），没有此前该端点的连续观测记录，因此无法判断是当天出现还是已经持续一段时间。

该结果针对本次提供的端点和观察时间，不证明全部 Chainnodes 节点均不可用，也不确定控制台套餐、账户状态或服务端路由的具体原因。尚未修改业务配置或接入该端点。

## 用户提供的端点

```text
https://polygon-mainnet.chainnodes.org/2c76fdcb-f1e1-4836-8e6c-4909b9fea1cc
wss://polygon-mainnet.chainnodes.org/2c76fdcb-f1e1-4836-8e6c-4909b9fea1cc
```

用户消息中的 `wss\://` 按 Markdown 转义还原为标准 `wss://` 后连接。

## 实测结果

| 项目 | 实际结果 | 判断 |
| --- | --- | --- |
| HTTP JSON-RPC | 返回 HTTP 200，`eth_chainId = 0x89` | 可达，网络标识正确；未返回鉴权错误 |
| `eth_blockNumber`、`latest` 区块 | 均为 `83,521,701`，复查未推进 | 数据滞后 |
| `finalized` 区块 | 同样返回 `83,521,701` | 没有获得当前最终确认高度 |
| `eth_syncing` | 返回阶段状态对象，`currentBlock = 83,521,701` | 结合缺失区块和明确上游错误，不能判断为已同步 |
| 读取对照节点的近期区块 `93,533,253` | RPC `-32014`，`block not found`，`ErrEndpointMissingData` | 近期数据不可用 |
| 读取既有 Polymarket 样本区块 `93,506,126` 的目标日志 | RPC `-32603`，`2 upstream not synced` | 成交日志查询失败 |
| 相同目标日志请求发往 dRPC | HTTP 200，返回目标的两条 `OrderFilled` | 对照证明样本存在、该请求可获得结果 |
| WebSocket 连接、链 ID 与高度 | 连接成功，Chain ID 137，高度仍为 `83,521,701` | 连接可用，数据仍滞后 |
| `newHeads` 订阅 | 返回订阅 ID；观察约 22 秒无新区块通知 | 未证明能获得实时区块 |
| `logs` 订阅 | 部分过滤写法被拒绝；另一种写法取得订阅 ID | 见下节；未验证真实成交推送 |

日志错误中列出的上游为 `cn-dp-zrh-polygon` 与 `cn-dp-fra-polygon`。在失败响应中，前者可见高度为 `83,521,701`，后者报告为 0；它们是服务端返回的诊断值，不代表已确定相应节点的真实运维状态。

## WebSocket 过滤写法观察

本次使用已有样本的 V2 `OrderFilled` 签名及目标钱包 `0x4ebc2722adc772bde8680792d0a6fdf15499a33d`：

| 请求形态 | 结果 |
| --- | --- |
| 两个合约地址 + `topics: [signature, null, walletTopic]` | `invalid logs filter` |
| 单个合约地址 + 相同 topics | `invalid logs filter` |
| 不指定合约地址 + 相同 topics | `invalid logs filter` |
| 单个合约地址 + `topics: [signature]` | 取得订阅 ID |
| 单个合约地址 + `topics: [[signature], [], [walletTopic]]` | 取得订阅 ID |

上述最后一种写法只证明请求被接受；由于节点没有提供当前区块，本次没有证据证明其真实推送和目标匹配效果。不能直接将其作为已确认的实现契约。[Chainnodes 官方 Polygon 订阅文档](https://www.chainnodes.org/docs/polygon/eth_subscribe)列出地址、地址数组与 topics 过滤能力，但实际端点仍需以返回行为为准。

全部成功建立的临时订阅均已执行 `eth_unsubscribe`，响应为 `true`，连接随后关闭。没有提交链上交易或创建持久监控任务。

## 对后续选择的影响

先前对 Chainnodes Core 的优先推荐基于免费额度与公开限制；本次实际结果表明，当前这个端点不能直接用于实时采集。需要 Chainnodes 检查该 Polygon 端点的上游同步或路由状态。对外说明可直接附上本报告中的旧高度、同期对照高度和 `2 upstream not synced` 错误；本次没有代用户联系服务商。

2026-09-10 补充查阅[官方免费层限制](https://www.chainnodes.org/docs/FAQs/rate_limits)与[错误码说明](https://www.chainnodes.org/docs/FAQs/error_codes)：公开的免费限制涉及额度、吞吐、连接数及繁忙时的优先级，没有将数月前的区块数据列为免费套餐规则；文档中的配额与频率限制通常返回 429。本次观测到的缺失区块及上游未同步错误，更符合节点数据或路由异常的表现，但这仍是诊断推断。没有付费端点对照或服务商确认，不能排除免费套餐所用上游池单独异常，也不能承诺升级套餐会解决。

本次公开检索未找到能核实该异常开始时间、影响套餐和恢复时间的官方故障记录。未找到公告不代表没有故障；旧区块时间也不能代替故障开始时间。上述三项需要服务商的历史监控或诊断记录确认。

原始记录可供排查，但不将本次短时验证扩展为长期稳定性、吞吐、完整故障恢复或通知时效结论。
