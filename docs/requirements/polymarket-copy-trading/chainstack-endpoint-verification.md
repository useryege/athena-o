# Chainstack Polygon 端点可用性验证

> 验证时间：2026-09-10 10:12–10:15（北京时间）。
>
> 范围：用户明确要求检查提供的 HTTP 与 WebSocket 端点；仅执行只读 RPC 与短时订阅。
>
> 原始证据：[HTTP、WSS、公共节点对照与权限响应](evidence/chainstack-endpoint-verification-2026-09-10.json)。

## 结论

**两个端点的实时数据能力已得到实际验证：区块时间为当前时间，HTTP 可查询近期成交，WebSocket 可收到新区块和目标钱包成交日志。当前套餐拒绝 Archive 请求，较早历史补查不可用。**

本次成功收到三个连续新区块和七条指定目标的 `OrderFilled` 日志；这七条日志与随后 Chainstack HTTP、PublicNode HTTP 的结果逐条一致，可以将该端点用于开发阶段的实时采集。用户随后明确当前不做历史补查，断线或服务恢复后只继续实时监控，因此 Archive 权限限制不构成当前需求范围的阻碍；具体边界见[Activity Alerts 需求](target-trade-monitoring-notifications.md#实时监控与中断恢复已确认)。

本结论是短时端点能力验证，不是长期稳定性、吞吐或通知时效保证，也不代表 Trader Sync 需求、技术设计或实现已完成。没有登录控制台核实套餐名称、账单设置或额度余额，没有修改业务运行配置。

## 用户提供的端点

| 协议 | 完整 URL |
| --- | --- |
| HTTPS JSON-RPC | `https://polygon-mainnet.core.chainstack.com/327c883749871f9378cea0845a639e96` |
| WSS | `wss://polygon-mainnet.core.chainstack.com/327c883749871f9378cea0845a639e96` |

网络为 Polygon PoS 主网，Chain ID 为 `137`。用户消息中的 `wss\://` 按 Markdown 转义还原为标准 `wss://` 后连接。

## 区块与连接结果

| 项目 | 实测结果 |
| --- | --- |
| HTTP 与 WSS `eth_chainId` | 均返回 `0x89`，即 137 |
| `eth_syncing` | 返回 `false` |
| 首次 `latest` | 高度 `93,534,231`，区块时间 `2026-09-10 02:13:00 UTC` |
| 首次 `finalized` | 高度 `93,534,229`，区块时间 `2026-09-10 02:12:57 UTC` |
| 后续 `latest` | 高度推进到 `93,534,322` |
| 公共节点核对 | dRPC 与 PublicNode 对高度 `93,534,231` 返回相同区块哈希，与 Chainstack 一致 |
| WSS `newHeads` | 实际收到 `93,534,278`、`93,534,279`、`93,534,280` 三个连续区块 |

首次最新区块返回时，与本地 UTC 时钟相差约 0.3 秒；这只是单次区块时间新鲜度观测，不是 RPC 延迟统计或端到端时效结论。`finalized` 在该快照中比 `latest` 低两个区块，也不能据此承诺固定最终确认延迟。

共同核对的区块哈希为：

```text
0x730d3c23c37ac974230edd29b18763ed7285c68de14b5ca6794cd234b40f25a1
```

## 目标成交日志与实时推送

先查询最近五个区块 `93,534,227` 至 `93,534,231` 的两种 V2 Exchange `OrderFilled`，得到 316 条日志；从样本中选取当时有活动的钱包，仅用于短时验证：

```text
0x674887d1ac838099a48b629dff53f25b7b87ee08
```

随后使用两种 Exchange 地址数组与以下 topics 过滤建立 WSS 日志订阅：

```json
[
  "0xd543adfd945773f1a62f74f0ee55a5e3b9b1a28262980ba90b1a89f2ea84d8ee",
  null,
  "0x000000000000000000000000674887d1ac838099a48b629dff53f25b7b87ee08"
]
```

实际收到区块 `93,534,278` 中的七条目标成交日志，钱包字段均匹配目标。随后分别通过 Chainstack 和 PublicNode，对相同区块及相同过滤条件执行 `eth_getLogs`，两者均返回七条；逐条比较合约地址、区块号、区块哈希、交易哈希、日志索引、topics 与 data，均与 WSS 结果相符。

该结果验证了多合约地址过滤及目标钱包所在的 `topics[2]` 过滤，不代表已验证任意大的目标 OR 数组或全部 Polymarket 协议版本。

收到第一批目标日志后即取消日志订阅；收到三个新区块后取消区块订阅。两次 `eth_unsubscribe` 均返回 `true`，验证进程已退出，没有留下后台监听。

## 当前套餐的历史补查限制

| 查询内容 | 单次查询跨度 | 实测结果 |
| --- | --- | --- |
| 昨天既有样本区块 `93,506,126` 的目标成交 | 1 块 | HTTP 403，RPC `-32002`，当前套餐不支持 Archive |
| 距所读链头 80 块的区块 `93,534,242` | 1 块 | HTTP 200，返回目标的五条成交日志 |
| 距所读链头 160 块的区块 `93,534,162` | 1 块 | HTTP 403，RPC `-32002`，当前套餐不支持 Archive |

拒绝响应原文为：

```text
Archive, Debug and Trace requests are not available on your current plan.
```

[官方 RU 规则](https://docs.chainstack.com/docs/request-units)将 Polygon 此类请求中距 tip 至少 127 块的数据归为 Archive，`eth_getLogs` 以 `fromBlock` 判定。这与本次 80 块前成功、160 块前被拒绝的结果一致。本次没有逐块测量精确的权限切换边界。

这是数据历史深度的套餐权限，与[免费 Developer 每次最多查询 100 块](https://docs.chainstack.com/docs/limits)是两个不同限制。**将旧数据拆成每次一个区块也不会获得 Archive 权限。**不能再仅以“把历史补查分成 100 块”认为免费套餐能够覆盖长时间中断后的恢复。

以上保留的是端点能力验证事实，不是本阶段必须实现的补查要求。用户已决定当前不做任何历史成交补查，包括仍可由近期查询取得的短时中断遗漏；开发阶段使用其实时能力即可，不再要求为补查配置其他来源或升级 Archive 套餐。本次没有升级套餐、开通付费超额或自动接入其他服务。

## 对当前候选选择的影响

Chainstack 已有本次实时数据和目标成交推送的正面证据，可作为开发实时采集入口的候选。此前 Chainnodes 端点在其验证时数据严重滞后，详见[Chainnodes 验证报告](chainnodes-endpoint-verification.md)；本次没有重新检查其是否恢复。

最终实时采集架构、容量和费用仍未完成设计确认，历史补查已明确不在当前范围。当前结论不会替代完整需求与技术设计阶段。
