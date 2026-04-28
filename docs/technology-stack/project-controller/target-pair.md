# Target Pair Data Synchronization

推荐设计是：

```text
Project Controller 是 target pair 的唯一权威来源
Block Sniffer 维护本地内存快照（只读热路径）
Project Controller 更新后发布增量事件
Block Sniffer 按版本号有序应用，并在异常时全量回补
```

目标是同时满足三件事：

```text
1) 更新可达：Controller 一更新，Sniffer 能尽快跟上
2) 一致可靠：重启、断线、乱序、重复都不丢不乱
3) 高性能：扫块热路径不依赖远程调用，不被重锁阻塞
```

---

# 方案总览：全量快照 + 增量事件 + 版本号 + 对账回补

```text
Project Controller
  -> target_pairs.updated (Redpanda)
Block Sniffer
  -> consume + version check + atomic swap cache
```

不要让 Block Sniffer 在扫块过程中频繁请求 Project Controller。

---

# 1. 数据模型（必须字段）

## 1.1 全量快照响应

```json
{
  "chain_id": 1,
  "snapshot_version": 1024,
  "pairs": [
    {
      "pair_address": "0x...",
      "token0": "0x...",
      "token1": "0x...",
      "project_id": "..."
    }
  ]
}
```

## 1.2 增量事件

```json
{
  "event_id": "target-pairs:1025",
  "chain_id": 1,
  "version": 1025,
  "op": "UPSERT",
  "pair_address": "0x...",
  "token0": "0x...",
  "token1": "0x...",
  "project_id": "...",
  "effective_from_block": 22000001,
  "updated_at_ms": 1710000000000
}
```

删除事件：

```json
{
  "event_id": "target-pairs:1026",
  "chain_id": 1,
  "version": 1026,
  "op": "REMOVE",
  "pair_address": "0x...",
  "effective_from_block": 22000010,
  "updated_at_ms": 1710000005000
}
```

字段约束：

```text
event_id: 全局唯一，用于幂等去重
version: 单链单调递增，全局连续（按 chain_id）
effective_from_block: 从哪个区块开始生效，避免边界块歧义
```

---

# 2. 启动与重连：避免“快照后到订阅前”丢更新

必须避免这个窗口：

```text
拉完 snapshot -> 还没开始消费事件 -> 期间发生更新
```

MVP 可用策略（简单可靠）：

```text
1) Block Sniffer 先启动 Redpanda consumer（不处理业务，只缓存拉到的事件）
2) 调 GetTargetPairs(chain_id) 拿 snapshot_version
3) 从缓存/队列里只应用 version > snapshot_version 的事件
4) 进入正常消费模式
```

等价要求：无论实现方式如何，都要保证“启动衔接期间不会漏 version”。

---

# 3. 事件应用规则（Sniffer 侧）

核心规则：

```text
1) event.version <= local.version: 忽略（重复/旧消息）
2) event.version == local.version + 1: 正常应用
3) event.version > local.version + 1: 判定缺口，触发 full resync
```

补充规则：

```text
同一 event_id 只处理一次（幂等）
full resync 失败时保持旧快照并告警，不要清空缓存
```

伪代码：

```go
func HandleTargetPairEvent(event TargetPairEvent) {
    cur := cache.Load()

    if dedup.Exists(event.EventID) || event.Version <= cur.Version {
        return
    }

    if event.Version != cur.Version+1 {
        snapshot, err := projectClient.GetTargetPairs(event.ChainID)
        if err != nil {
            metrics.ResyncFail.Inc()
            return
        }
        cache.Replace(snapshot) // atomic swap
        return
    }

    next := cur.Clone() // copy-on-write
    switch event.Op {
    case "UPSERT":
        next.Pairs[event.PairAddress] = event.PairMeta
    case "REMOVE":
        delete(next.Pairs, event.PairAddress)
    }
    next.Version = event.Version
    cache.Replace(next) // atomic swap
}
```

---

# 4. 内存结构与性能（热路径）

建议结构：

```go
type PairSnapshot struct {
    Version uint64
    Pairs   map[common.Address]PairMeta
}

var pairCache atomic.Value // stores *PairSnapshot
```

读取路径（扫块）只读：

```go
snapshot := pairCache.Load().(*PairSnapshot)
if pair, ok := snapshot.Pairs[pairAddress]; ok {
    _ = pair // emit sync event
}
```

更新路径使用 copy-on-write + atomic swap，避免读写锁竞争。

---

# 5. Redpanda topic 设计（MVP）

推荐：

```text
topic: target_pairs.updated
partition key: chain_id
```

原因：

```text
同一链单分区天然有序
version 检查简单
排障成本低
MVP 吞吐通常足够
```

不建议 MVP 直接用 `chain_id:pair_address` 分区；会引入多分区乱序与更复杂对账逻辑。

---

# 6. 定期 reconciliation（兜底）

即便使用 Redpanda，也应定期对账（如 30s/60s）：

```text
Block Sniffer -> Project Controller: GetTargetPairVersion(chain_id)
if remoteVersion != localVersion:
    GetTargetPairs(chain_id) full resync
```

可防御：

```text
消费异常
长时间断线
配置错误
人工修复导致的状态漂移
```

---

# 7. 可观测性与 SLO（必须）

关键指标：

```text
target_pair_update_lag_ms
target_pair_version_gap
target_pair_resync_count
target_pair_resync_fail_count
target_pair_event_duplicate_count
```

建议 SLO（可按实际压测调整）：

```text
P99 target_pair_update_lag_ms < 2000ms
version_gap 常态为 0
resync_fail_count 持续为 0
```

---

# 最终推荐流程

```text
Project Controller 是唯一写入方和权威来源

Controller 更新 target pair:
  1) 更新数据库/内存
  2) version += 1
  3) 发布 target_pairs.updated（含 effective_from_block）

Sniffer 启动/重连:
  1) 建立消费能力（保证衔接不漏）
  2) 拉 full snapshot，得到 snapshot_version
  3) 应用 version > snapshot_version 的事件
  4) 进入正常消费

Sniffer 运行中:
  1) 扫块热路径只读本地 cache
  2) 按 version 有序应用事件
  3) 跳号即 full resync
  4) 定期对账兜底
```

一句话总结：

**Project Controller 主动发布增量，Block Sniffer 本地快照原子更新；用 version 保证顺序与完整性，用启动衔接和周期对账消除漏更风险，用 `effective_from_block` 保证链上边界一致性。**
