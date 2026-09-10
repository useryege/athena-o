# dRPC Polygon 端点可用性验证

> 验证时间：2026-09-10 10:53–10:55（北京时间）。
>
> 范围：用户明确要求检查提供的 HTTP 与 WSS 端点；执行只读 RPC、短时实时订阅及当前成交样本核对，不做历史补查。
>
> 原始证据：[HTTP、WSS 与其他节点对照](evidence/drpc-endpoint-verification-2026-09-10.json)。

## 结论

**两个端点的实时能力验证通过：HTTP 返回当前 Polygon 主网区块，WSS 实际收到三个连续新区块和一条指定目标钱包的成交日志。该成交与 dRPC HTTP、PublicNode HTTP 的结果逐项一致。**

可以作为当前只做实时监控的开发候选。本次没有发现连接、鉴权或订阅错误；用户随后明确该端点使用免费节点，此信息来自用户说明，验证过程未登录控制台读取剩余额度或账单设置。本次短时结果不代表长期稳定性、吞吐或产品通知时效已得到验证。

## 用户提供的端点

| 协议 | 完整 URL |
| --- | --- |
| HTTPS JSON-RPC | `https://lb.drpc.live/polygon/Ai_L7JAEMk5hgVgUsOe8aKuLbjItrMIR8Yhzzu2G7ZgM` |
| WSS | `wss://lb.drpc.live/polygon/Ai_L7JAEMk5hgVgUsOe8aKuLbjItrMIR8Yhzzu2G7ZgM` |

用户消息中的 `wss\://` 和 `Ai\_` 按 Markdown 转义还原为标准协议及下划线。网络为 Polygon PoS 主网，Chain ID 为 `137`。

## 区块与 HTTP 结果

| 项目 | 实测结果 |
| --- | --- |
| HTTP 与 WSS `eth_chainId` | 均返回 `0x89`，即 137 |
| HTTP `eth_syncing` | 返回 `false` |
| 首次 `latest` | 高度 `93,535,858`，区块时间 `2026-09-10 02:53:40 UTC` |
| 后续 `latest` | 高度推进到 `93,535,948`，区块时间 `2026-09-10 02:55:55 UTC` |
| 其他节点核对 | Chainstack、PublicNode 对高度 `93,535,858` 返回的区块哈希均与 dRPC 一致 |
| 最近三个区块的成交查询 | `93,535,856` 至 `93,535,858`，两种核心 V2 Exchange 共返回 254 条 `OrderFilled` |

上述 HTTP 请求均成功。新区块持续推进且与独立查询结果一致，支持本次数据新鲜度结论。没有将单次请求耗时当作服务商性能排名或延迟保障。

## 目标钱包过滤与实时推送

从近期样本中选择以下当时活跃的钱包，仅用于短时验证；该钱包在上述三个区块内有 14 条成交日志，不构成产品中的交易者推荐或频率判定：

```text
0xfe787d2da716d60e8acff57fb87eb13cd4d10319
```

WSS 成交订阅同时使用两个核心 V2 Exchange 合约地址，并按目标钱包过滤：

```json
{
  "address": [
    "0xE111180000d2663C0091e4f400237545B87B996B",
    "0xe2222d279d744050d28e00520010520000310F59"
  ],
  "topics": [
    "0xd543adfd945773f1a62f74f0ee55a5e3b9b1a28262980ba90b1a89f2ea84d8ee",
    null,
    "0x000000000000000000000000fe787d2da716d60e8acff57fb87eb13cd4d10319"
  ]
}
```

实际结果：

- `newHeads` 收到 `93,535,914`、`93,535,915`、`93,535,916` 三个连续区块。
- `logs` 收到区块 `93,535,916` 的一条目标成交，目标字段与过滤地址一致。
- 成交交易哈希为 `0xe09890d5df86cf77456d7e017abf8b1c9fa7e0c9ab4f3e54ccc03c7dc48c1f2a`，日志索引为 `0x1e2`。
- 随后对该当前成交样本分别调用 dRPC 和 PublicNode 的 `eth_getLogs`，相同区块及过滤条件均返回一条。合约、区块号与哈希、交易哈希与索引、日志索引、topics、data 和 removed 均与 WSS 一致。

这验证了多合约地址过滤和 `topics[2]` 目标过滤的实际推送，没有仅凭订阅 ID 判断日志可用。尚未验证大型目标 OR 数组或全部 Polymarket 协议类型。

两个 `eth_unsubscribe` 均返回 `true`，WSS 正常关闭（code 1000、`wasClean=true`），验证进程已退出，没有留下后台监听。

## 对当前需求的影响

本次近期日志读取仅用于选取验证样本和核对实际推送，不增加历史补查要求。当前范围继续按[实时监控与中断恢复](target-trade-monitoring-notifications.md#实时监控与中断恢复已确认)执行：中断后继续接收新成交，遗漏不补查，市场资料仍通过 API 补齐。

dRPC 与 [Chainstack](chainstack-endpoint-verification.md) 均已有实时可用的短时证据；没有据此配置多供应商自动切换、修改业务运行配置或改变需求及技术设计状态。

[返回服务商调研](hosted-polygon-rpc-providers.md)
