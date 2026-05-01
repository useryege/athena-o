你这个需求在成熟项目里通常不会叫“定时拉数据”，而会被设计成一个专门的 **状态同步系统**。

更准确地说，它是：

> **Project State Watcher / State Synchronizer / State Monitor**

它负责持续维护你关心的所有 Project 的最新状态，并在状态发生变化时触发事件。

---

## 一、你的需求本质是什么

你的流程可以抽象成这样：

```text
多个 Project
   ↓
按照一定间隔检查链上状态
   ↓
拿到最新链上数据
   ↓
和本地缓存状态对比
   ↓
如果没变化：跳过
   ↓
如果有变化：更新本地状态 + 触发事件
   ↓
其他模块收到事件后执行后续逻辑
```

同时你还需要：

```text
定时触发
手动触发
可能未来还有区块触发 / WebSocket 触发 / 外部事件触发
```

所以它不应该被设计成一个简单的 `for + sleep`。

它应该被设计成一个 **事件驱动 + 状态缓存 + 调度器 + 比较器** 的系统。

---

# 二、企业级项目一般怎么设计

一般会拆成几个核心模块：

```text
Project Registry
Project State Store
Project Sync Scheduler
Project State Fetcher
State Comparator
Event Bus
Project State Manager
```

可以理解成这样：

```text
                ┌──────────────────────┐
                │   Project Registry    │
                │  维护要监控的项目列表  │
                └──────────┬───────────┘
                           │
                           ↓
                ┌──────────────────────┐
                │ Project Sync Scheduler│
                │ 定时/手动/事件触发同步 │
                └──────────┬───────────┘
                           │
                           ↓
                ┌──────────────────────┐
                │  Project State Fetcher│
                │    从链上拉最新状态    │
                └──────────┬───────────┘
                           │
                           ↓
                ┌──────────────────────┐
                │   State Comparator    │
                │  比较链上状态和本地状态 │
                └──────────┬───────────┘
                           │
              changed?     │
                 yes       ↓
                ┌──────────────────────┐
                │ Project State Manager │
                │ 更新内存 / Redis / DB  │
                └──────────┬───────────┘
                           │
                           ↓
                ┌──────────────────────┐
                │      Event Bus        │
                │ 发布 ProjectChanged 事件│
                └──────────────────────┘
```

---

# 三、核心概念应该这样定义

## 1. Project Registry：维护你关注哪些项目

这个模块只负责告诉系统：

> 当前有哪些 Project 需要被监控。

例如：

```go
type Project struct {
    ID              string
    ChainID         int64
    ContractAddress string
    Type            string
    SyncInterval    time.Duration
    Enabled         bool
}
```

它的数据来源可以是：

```text
PostgreSQL：持久化项目配置
Redis：快速缓存活跃项目
内存 Map：运行时高速访问
```

企业级项目一般不会只放内存，因为重启后会丢。

比较合理的是：

```text
PostgreSQL = 项目配置的权威数据源
Redis = 分布式缓存 / 状态快照
内存 = 当前进程的高速状态副本
```

---

## 2. Project State Store：维护本地状态

你本地要保存的是每个 Project 当前已知的状态。

例如：

```go
type ProjectState struct {
    ProjectID      string
    ChainID        int64
    BlockNumber    uint64
    StateHash      string
    RawState       ProjectRawState
    UpdatedAt      time.Time
}
```

注意这里非常关键：

不要每次都拿完整结构体做深度比较。

更好的方式是维护一个：

```text
StateHash
Version
BlockNumber
UpdatedAt
```

例如：

```go
type ProjectState struct {
    ProjectID   string
    BlockNumber uint64
    StateHash   string
    Data        ProjectData
}
```

然后比较时先比较：

```go
if oldState.StateHash == newState.StateHash {
    return NoChange
}
```

这样效率高很多。

对于链上数据，常见的状态版本可以是：

```text
blockNumber
contract storage hash
important field hash
event sequence
lastUpdated timestamp
nonce / version
```

如果没有天然版本号，你可以自己根据关键字段计算 hash。

---

## 3. Project Sync Scheduler：同步调度器

你现在的需求有两个触发方式：

```text
每 10 秒自动同步一次
用户手动触发同步一次
```

成熟系统里通常不会把这两个逻辑写成两套，而是统一成一个 Sync Request。

例如：

```go
type SyncReason string

const (
    SyncReasonInterval SyncReason = "interval"
    SyncReasonManual   SyncReason = "manual"
    SyncReasonStartup  SyncReason = "startup"
    SyncReasonBlock    SyncReason = "block"
    SyncReasonEvent    SyncReason = "event"
)
```

然后不管是定时还是手动，最后都变成：

```go
SyncProject(projectID, reason)
```

也就是说：

```text
定时器触发 → 创建 SyncRequest
手动触发 → 创建 SyncRequest
新区块触发 → 创建 SyncRequest
WebSocket 触发 → 创建 SyncRequest
```

统一进入同一个队列。

---

# 四、推荐的整体架构

我建议你设计成这样：

```text
                  ┌──────────────────────┐
                  │      API Layer        │
                  │ 手动触发 / 查询状态    │
                  └──────────┬───────────┘
                             │
                             ↓
                    Manual Sync Request
                             │
                             ↓
┌───────────────┐     ┌──────────────────────┐
│ Interval Timer│────▶│    Sync Request Queue │
└───────────────┘     └──────────┬───────────┘
                                 │
                                 ↓
                        ┌─────────────────┐
                        │  Worker Pool    │
                        │ 并发处理同步任务 │
                        └────────┬────────┘
                                 │
                                 ↓
                        ┌─────────────────┐
                        │ Chain Data Fetcher│
                        │ RPC / multicall  │
                        └────────┬────────┘
                                 │
                                 ↓
                        ┌─────────────────┐
                        │ State Comparator │
                        └────────┬────────┘
                                 │
                    changed?     │
                       yes       ↓
                        ┌─────────────────┐
                        │ State Store      │
                        │ memory/Redis/DB  │
                        └────────┬────────┘
                                 │
                                 ↓
                        ┌─────────────────┐
                        │ Event Publisher  │
                        │ ProjectChanged   │
                        └─────────────────┘
```

---

# 五、最关键的设计：不要让每个 Project 自己开一个定时器

这是很多新项目容易犯的错误。

比如你有 10,000 个 Project，如果你给每个 Project 启一个 goroutine 和 ticker：

```go
for _, project := range projects {
    go func(project Project) {
        ticker := time.NewTicker(10 * time.Second)
        for range ticker.C {
            SyncProject(project.ID)
        }
    }(project)
}
```

这个设计前期能跑，但是后面会出问题：

```text
定时器数量太多
goroutine 难管理
RPC 请求容易同时爆发
无法统一限流
无法统一重试
无法统一优先级
无法统一暂停和恢复
```

企业级项目一般不会这样做。

更好的方式是：

```text
中央调度器 + 任务队列 + Worker Pool
```

也就是：

```text
Scheduler 只负责决定谁该同步了
Queue 负责排队
Worker Pool 负责执行
Rate Limiter 控制 RPC 压力
```

---

# 六、调度器应该怎么做

你可以用一个优先队列维护每个 Project 下一次同步时间。

```go
type SyncPlan struct {
    ProjectID string
    NextSyncAt time.Time
    Interval time.Duration
}
```

调度器内部维护：

```text
min-heap by NextSyncAt
```

每次取出最早需要同步的 Project。

流程：

```text
1. 从 heap 里取出 NextSyncAt 最早的 Project
2. 如果还没到时间，就 sleep 到那个时间
3. 到时间后生成 SyncRequest
4. 把该 Project 的 NextSyncAt 更新为 now + interval
5. 放回 heap
```

这样比每个 Project 一个 ticker 更可控。

---

# 七、手动触发怎么设计

手动触发不要直接调用链上 RPC。

应该也是进入 Sync Queue。

例如：

```text
POST /projects/{id}/sync
        ↓
create SyncRequest {
    ProjectID: id,
    Reason: manual,
    Priority: high
}
        ↓
push to sync queue
```

这样做的好处是：

```text
手动同步和自动同步走同一套逻辑
可以复用限流
可以复用去重
可以复用状态比较
可以复用事件发布
```

手动触发可以设置更高优先级：

```go
type SyncRequest struct {
    ProjectID string
    Reason    SyncReason
    Priority  int
    CreatedAt time.Time
}
```

例如：

```text
manual priority = 100
interval priority = 10
startup priority = 50
```

---

# 八、必须做任务去重

假设一个 Project 每 10 秒同步一次，但用户在第 5 秒手动同步了一次。

你不希望出现这种情况：

```text
第 5 秒：手动同步
第 10 秒：定时同步
第 10.1 秒：又同步一次
```

如果链上状态没变，这就是浪费 RPC。

所以需要对每个 Project 做 in-flight 控制：

```go
type SyncState struct {
    InFlight bool
    Pending  bool
}
```

逻辑：

```text
如果 Project 正在同步：
    新请求不要重复执行
    标记 Pending = true

当前同步结束后：
    如果 Pending = true
        再执行一次
```

或者更简单：

```text
同一个 Project 在短时间窗口内只允许存在一个 SyncRequest
```

例如：

```go
dedupeKey := "project:" + projectID
```

---

# 九、状态比较怎么做

不要直接比较完整对象。

推荐三层比较：

## 第一层：blockNumber

如果你拉到的数据对应的区块高度没有变化：

```go
if new.BlockNumber <= old.BlockNumber {
    return NoChange
}
```

这通常可以直接跳过。

但是注意，有些状态虽然在新区块中没有变化，blockNumber 变了也不代表状态变了。

所以第二层比较：

## 第二层：StateHash

把你关心的字段序列化后计算 hash：

```go
stateHash := Hash(project.TotalLiquidity, project.Price, project.Status, project.ConfigVersion)
```

比较：

```go
if old.StateHash == new.StateHash {
    return NoChange
}
```

## 第三层：Diff

如果 hash 不一样，再计算具体哪些字段变了：

```go
type StateDiff struct {
    ProjectID string
    ChangedFields []string
    OldState ProjectState
    NewState ProjectState
}
```

然后发布事件：

```go
ProjectStateChanged {
    ProjectID
    OldState
    NewState
    Diff
    BlockNumber
}
```

---

# 十、事件应该怎么设计

当状态变化后，不要在 State Synchronizer 里面直接执行后续业务逻辑。

比如不要这样：

```go
if changed {
    updateState()
    executeTradingStrategy()
    sendNotification()
    updateDashboard()
}
```

这样耦合度太高。

成熟系统会这样：

```go
if changed {
    updateState()
    publish(ProjectStateChanged)
}
```

然后其他模块订阅这个事件：

```text
Trading Engine 订阅 ProjectStateChanged
Notification Service 订阅 ProjectStateChanged
Dashboard Gateway 订阅 ProjectStateChanged
Risk Engine 订阅 ProjectStateChanged
```

也就是：

```text
状态同步模块只负责发现变化
其他模块自己决定如何响应变化
```

---

# 十一、用 Redis、PostgreSQL、Redpanda 怎么分工

结合你当前技术栈，我建议：

## PostgreSQL

存这些：

```text
Project 配置
Project 基础信息
Project 是否启用
Project 同步间隔
Project 关注字段配置
历史状态快照，可选
```

PostgreSQL 是权威数据源。

---

## Redis

存这些：

```text
Project 当前状态
Project StateHash
Project 最近同步时间
Project in-flight lock
Project active set
```

Redis 适合高频读写状态。

例如：

```text
project:state:{projectID}
project:hash:{projectID}
project:sync_lock:{projectID}
project:last_sync_at:{projectID}
```

---

## 内存

存这些：

```text
活跃 Project 列表
调度 heap
最近状态快照
RPC client 缓存
Project runtime metadata
```

内存是最快的，但不能作为唯一真相。

---

## Redpanda / Kafka

发布这些事件：

```text
ProjectStateChanged
ProjectSyncFailed
ProjectSyncSucceeded
ProjectAdded
ProjectRemoved
ProjectDisabled
```

例如 topic：

```text
project.state.changed
project.sync.status
project.lifecycle
```

---

# 十二、MVP 阶段可以怎么做

第一期不要一上来就搞得太复杂。

我建议 MVP 这样设计：

```text
PostgreSQL：保存 Project 配置
Redis：保存当前 Project 状态和 hash
Go 内存：维护活跃 Project 调度器
Go channel：作为本地 Sync Queue
Worker Pool：并发同步
Event Bus：先用 Go channel
```

也就是说，第一版可以不用 Redpanda。

结构：

```text
ProjectController
    ├── ProjectRegistry
    ├── ProjectScheduler
    ├── ProjectSyncQueue
    ├── ProjectSyncWorkerPool
    ├── ProjectStateStore
    └── ProjectEventBus
```

代码层面可以分成：

```text
/internal/project/
    controller.go
    registry.go
    scheduler.go
    sync_queue.go
    sync_worker.go
    state_store.go
    comparator.go
    event_bus.go
```

---

# 十三、后期高性能版本怎么演进

当 Project 数量变多后，比如：

```text
1,000 个 Project
10,000 个 Project
100,000 个 Project
```

你需要逐步引入：

```text
Redis Cluster
Redpanda / Kafka
分片调度
RPC 批量请求
Multicall
链上事件监听
状态分层缓存
Rust 热路径
```

尤其是链上数据，不要每个 Project 单独 RPC 拉。

如果多个 Project 在同一条链上、同一个合约类型，可以批量拉：

```text
chain_id + contract_type 分组
        ↓
multicall
        ↓
一次 RPC 拿多个 Project 状态
```

这个对性能影响非常大。

---

# 十四、非常重要：链上状态不要只靠轮询

你现在说的是“每 10 秒拉一次”。

这是可以的，但在链上系统里，成熟设计一般是：

```text
事件监听为主
轮询校验为辅
手动触发兜底
```

也就是：

```text
WebSocket / Logs / New Block
        ↓
发现可能变化
        ↓
触发 Project Sync

定时轮询
        ↓
防止漏事件 / RPC 中断 / 节点不同步

手动触发
        ↓
人工排查 / 立即刷新
```

推荐最终模型：

```text
链上 Event / New Block 触发：快速响应
定时 Polling：数据校验
Manual Sync：人工触发
```

所以不是：

```text
每 10 秒全量扫一遍所有 Project
```

而是：

```text
有事件的 Project 优先同步
长时间没同步的 Project 定期校验
用户手动触发的 Project 高优先级同步
```

---

# 十五、推荐的事件类型

你可以定义这些事件：

```go
type ProjectEventType string

const (
    ProjectStateChanged ProjectEventType = "project.state.changed"
    ProjectSyncStarted  ProjectEventType = "project.sync.started"
    ProjectSyncSucceeded ProjectEventType = "project.sync.succeeded"
    ProjectSyncFailed   ProjectEventType = "project.sync.failed"
    ProjectSyncSkipped  ProjectEventType = "project.sync.skipped"
)
```

核心事件：

```go
type ProjectStateChangedEvent struct {
    ProjectID   string
    ChainID     int64
    OldHash     string
    NewHash     string
    OldState    ProjectState
    NewState    ProjectState
    Diff        StateDiff
    BlockNumber uint64
    ChangedAt   time.Time
}
```

---

# 十六、推荐的 Go 接口设计

可以这样抽象：

```go
type ProjectStateFetcher interface {
    Fetch(ctx context.Context, project Project) (ProjectState, error)
}
```

```go
type ProjectStateStore interface {
    Get(ctx context.Context, projectID string) (ProjectState, error)
    Set(ctx context.Context, state ProjectState) error
}
```

```go
type ProjectStateComparator interface {
    Compare(oldState, newState ProjectState) StateCompareResult
}
```

```go
type ProjectEventPublisher interface {
    Publish(ctx context.Context, event ProjectEvent) error
}
```

```go
type ProjectSyncService interface {
    Sync(ctx context.Context, projectID string, reason SyncReason) error
}
```

这样以后你可以替换具体实现：

```text
内存 Store → Redis Store
Go channel EventBus → Redpanda EventBus
普通 RPC Fetcher → Multicall Fetcher
Go Worker → Rust Worker
```

架构不会被推翻。

---

# 十七、我建议你的命名

你现在说的这个东西，可以叫：

```text
Project State Manager
Project State Watcher
Project Sync Engine
Project Monitor
```

如果放在你的系统里，我建议：

## 模块名

```text
Project Controller
```

它是总控。

内部包含：

```text
Project Registry
Project Scheduler
Project State Manager
Project Sync Engine
Project Event Publisher
```

更清晰一点：

```text
Project Controller
    ├── Registry
    ├── Scheduler
    ├── Sync Engine
    ├── State Store
    └── Event Publisher
```

---

# 十八、推荐最终设计

我建议你采用这个设计：

```text
Project Controller
负责维护所有 Project 的生命周期和状态同步。

核心流程：

1. 从 PostgreSQL 加载所有 enabled Project
2. 在内存中维护 active project registry
3. Scheduler 根据每个 Project 的 sync interval 生成 SyncRequest
4. 手动触发也生成 SyncRequest
5. SyncRequest 进入统一队列
6. Worker Pool 消费队列
7. Fetcher 从链上拉取最新状态
8. Comparator 和本地状态比较
9. 如果状态没变，只更新 last_checked_at
10. 如果状态变化，更新 Redis / 内存状态
11. 发布 ProjectStateChanged 事件
12. 其他模块订阅事件并执行后续逻辑
```

架构图：

```text
                 ┌────────────────────┐
                 │   PostgreSQL        │
                 │ Project Config      │
                 └─────────┬──────────┘
                           │
                           ↓
                 ┌────────────────────┐
                 │ Project Registry    │
                 └─────────┬──────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ↓                  ↓                  ↓
 Interval Trigger     Manual Trigger     Chain Event Trigger
        │                  │                  │
        └──────────────────┴──────────────────┘
                           ↓
                 ┌────────────────────┐
                 │  Sync Request Queue │
                 └─────────┬──────────┘
                           ↓
                 ┌────────────────────┐
                 │   Worker Pool       │
                 └─────────┬──────────┘
                           ↓
                 ┌────────────────────┐
                 │ Chain State Fetcher │
                 └─────────┬──────────┘
                           ↓
                 ┌────────────────────┐
                 │ State Comparator    │
                 └─────────┬──────────┘
                           ↓
                 ┌────────────────────┐
                 │ Redis / Memory      │
                 │ Current State Store │
                 └─────────┬──────────┘
                           ↓
                 ┌────────────────────┐
                 │ Event Publisher     │
                 │ ProjectStateChanged │
                 └────────────────────┘
```

---

# 十九、最小可行版本我建议这样落地

第一版不要太复杂。

你可以先做：

```text
1. PostgreSQL 存 Project 配置
2. Go 内存维护 activeProjects map
3. Go heap 调度 nextSyncAt
4. Go channel 作为 syncQueue
5. 固定数量 Worker Pool
6. Redis 存 Project 当前状态 hash
7. 状态变化后通过内部 EventBus 发布事件
```

暂时先不引入：

```text
Redpanda
Kafka
Rust
复杂分布式调度
```

等你项目数量大了，再扩展。

---

# 二十、重点结论

你的需求应该设计成：

> **Project Controller 内部的 Project State Sync Engine**

它不是简单定时任务，而是一个：

```text
统一调度
统一队列
统一限流
统一状态比较
统一事件发布
支持定时触发
支持手动触发
支持未来链上事件触发
```

的状态同步系统。

我建议你的核心设计原则是：

```text
定时器不要散落在每个 Project 里
所有同步都变成 SyncRequest
所有 SyncRequest 都进入统一队列
所有状态变化都通过事件发布
本地状态用 StateHash 快速比较
PostgreSQL 管配置
Redis 管高频状态
内存管运行时高速调度
```

一句话总结：

> **Project Controller 不应该“每 10 秒遍历所有 Project 拉一次数据”，而应该维护一个可调度、可去重、可限流、可事件驱动的 Project State Sync Engine。**
