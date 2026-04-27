下面是我建议的 **Athena 技术栈**。我会按你的性能优先原则来选：**交易热路径低延迟、事件可回放、状态强一致、链上数据高吞吐写入、行情查询快、故障后可恢复**。

---

# 一句话结论

我建议 Athena 第一版使用：

```text
Backend Language: Rust / Go
Primary DB: PostgreSQL
Cache / Hot State: Redis
MQ / Event Stream: Kafka / Redpanda
Analytics DB: ClickHouse
RPC / Internal Communication: gRPC
API Layer: REST + WebSocket
Deployment: Docker + Kubernetes
Observability: Prometheus + Grafana + OpenTelemetry
```

如果团队规模较小、想降低运维复杂度，可以先用：

```text
PostgreSQL + Redis + NATS JetStream + ClickHouse
```

但如果你们明确追求高吞吐、事件回放和后续扩展，我更推荐：

```text
PostgreSQL + Redis + Kafka/Redpanda + ClickHouse
```

---

# 1. 后端语言：Rust 优先，其次 Go

## 推荐

```text
Rust：Block Sniffer / Strategy Engine / Swap Server / Tx Speed Up Server
Go：API Server / Admin Service / Worker Service
TypeScript：前端和部分后台管理服务
```

## 原因

Athena 是区块链自动交易系统，**低延迟和稳定性直接影响收益**。核心模块不建议用 Node.js / Python 作为主执行层。

### Rust 适合：

```text
Block Sniffer
Buy Strategy Engine
Sell Strategy Engine
Swap Server
Tx Speed Up Server
Protect Server
```

原因：

* 性能强，延迟低。
* 内存控制好，没有 GC 暂停。
* 适合处理高频事件流、交易构造、签名、RPC 调用。
* 更适合写对性能敏感的 blockchain runtime 服务。

### Go 适合：

```text
API Server
Order Controller
后台任务服务
管理服务
运维服务
```

原因：

* 开发效率高。
* 并发模型简单。
* 生态成熟。
* 写 API、worker、控制服务很舒服。

### 不建议：

```text
Node.js / Python 作为交易热路径主语言
```

可以用于后台脚本、数据分析、内部工具，但不建议负责抢交易、加速交易、保护性卖出这种核心链路。

---

# 2. 主数据库：PostgreSQL

## 推荐用途

```text
用户配置
用户钱包配置
策略配置
订单主表
订单生命周期状态
交易记录
系统配置
审计日志
```

## 为什么选 PostgreSQL

Athena 里的 `Orders` 是强状态对象，涉及：

```text
Order Created
Waiting to Create Transaction
Transaction Created
Waiting for Confirmation
Transaction Confirmed
Transaction Failed
Position Updated
Closed
```

这些状态必须可靠，不能只放缓存里。

PostgreSQL 的优势是：

* 事务能力强。
* 数据一致性好。
* 适合订单、用户、策略、交易记录这种关系型数据。
* 支持 MVCC 并发控制，在并发读写下保证数据完整性。PostgreSQL 官方文档也强调其并发控制目标是在多 session 并发访问时同时保证高效访问和严格数据完整性。([PostgreSQL][1])
* 后续可以很方便做审计、回溯、报表、后台查询。

## 推荐表分层

```text
users
wallets
strategies
projects_snapshot
orders
order_events
transactions
risk_events
system_configs
```

其中我强烈建议：

```text
orders = 当前订单状态表
order_events = 订单事件流水表，只追加，不覆盖
transactions = 链上交易记录表
```

不要只维护一个 `orders.status`，还要有 `order_events`。

因为交易系统后续一定会遇到：

```text
为什么这个订单买了？
为什么没卖？
为什么保护卖出触发了？
哪个服务发出了信号？
交易卡在哪一步？
```

这些都需要事件流水回放。

## PostgreSQL 不适合做什么

不建议把所有链上 tick、log、price update 都直接高频写 PostgreSQL。

例如：

```text
每个 block 的所有 logs
每个 token 的秒级价格
大量 pending tx 状态
高频行情快照
```

这些更适合 Kafka + ClickHouse。

---

# 3. 缓存与热状态：Redis

## 推荐用途

```text
Projects 热数据缓存
token 最新价格
token 最新流动性
最新 block height
策略运行中的临时状态
订单热状态
分布式锁
限流
RPC 节点健康状态
pending transaction 状态
```

Redis 官方文档定位它可作为 cache、primary database、streams/pubsub 等用途，适合低延迟的内存数据访问场景。([Redis][2])

## Athena 里 Redis 应该放什么

### 1. Project 热数据

```text
project:{chain_id}:{token_address}
```

内容包括：

```text
price
liquidity
volume
holder_count
risk_score
last_block
last_updated_at
```

Strategy Engine 读取 Redis，比每次查 DB 快很多。

### 2. 最新 block height

```text
chain:{chain_id}:latest_block
```

Block Sniffer、Project Controller、Protect Server 都可能用。

### 3. pending tx 状态

```text
tx:{chain_id}:{tx_hash}
```

内容包括：

```text
nonce
gas_price
status
created_at
last_speed_up_at
speed_up_count
```

Tx Speed Up Server 可以快速判断是否需要替换交易。

### 4. 分布式锁

例如：

```text
lock:order:{order_id}:sell
lock:wallet:{wallet_address}:nonce
lock:token:{token_address}:sync
```

尤其是 nonce 管理和防止重复卖出，Redis 锁非常有用。

## Redis 注意点

Redis 不应该作为唯一数据源。

正确方式：

```text
Redis = 热状态 / 缓存 / 短期状态
PostgreSQL = 订单事实数据
Kafka = 事件事实流
ClickHouse = 链上和行情分析数据
```

对于订单核心状态，Redis 只能加速，不能替代 PostgreSQL。

---

# 4. MQ：推荐 Kafka 或 Redpanda，不优先 RabbitMQ

## 结论

对于 Athena，我推荐：

```text
首选：Kafka / Redpanda
备选：NATS JetStream
不优先：RabbitMQ
```

## 为什么 Kafka 更适合 Athena

Athena 本质上不是普通业务消息队列，而是 **事件驱动交易系统**。

你们需要的是：

```text
block event stream
project sync event stream
strategy signal stream
order event stream
transaction event stream
risk event stream
```

这些事件有几个特点：

* 吞吐量高。
* 需要保留历史。
* 需要重放。
* 需要多个 consumer group 并行消费。
* 需要故障后从 offset 继续。
* 后续可能接入风控、回测、监控、分析多个下游。

Kafka 官方定义就是高性能数据管道、流式分析、数据集成、关键业务应用使用的分布式事件流平台。([Apache Kafka][3])

所以 Athena 里 Kafka 很合适。

## 推荐 topic 设计

```text
chain.block.detected
chain.tx.detected
chain.log.detected

project.sync.requested
project.updated

strategy.buy.signal
strategy.sell.signal

order.created
order.status.changed
order.position.updated

tx.create.requested
tx.created
tx.speedup.requested
tx.speedup.submitted
tx.confirmed
tx.failed

risk.detected
protect.sell.requested
```

## Kafka 的核心价值

### 1. 可回放

如果 Strategy Engine 出 bug，可以从某个 offset 重新消费历史事件，重新计算策略结果。

这对交易系统很重要。

### 2. 多消费者独立消费

同一个 `project.updated` 可以同时给：

```text
Buy Strategy Engine
Sell Strategy Engine
Protect Server
Analytics Worker
WebSocket Push Service
```

互不影响。

### 3. 适合高吞吐链上事件

Block Sniffer 可能瞬间产生大量事件，Kafka 比 RabbitMQ 更适合这种持续事件流。

## Kafka vs RabbitMQ

RabbitMQ 更适合：

```text
任务队列
复杂路由
后台 job
邮件/通知
普通业务异步任务
```

RabbitMQ 官方文档重点围绕 broker 管理、monitoring、exchange、queue、binding 等传统消息代理模型展开，exchange + queue 的路由模型很灵活。([RabbitMQ][4])

但 Athena 需要的不是简单“把任务投递给某个 worker”，而是：

```text
事件流
事件保留
事件回放
高吞吐 fanout
按 offset 恢复
```

所以我不建议用 RabbitMQ 作为主 MQ。

## Kafka 还是 Redpanda？

如果你们想要 Kafka 生态，但希望运维更简单，可以考虑：

```text
Redpanda
```

它兼容 Kafka API，架构更轻，通常部署和维护会比传统 Kafka 简单。

我的建议：

```text
团队有 Kafka 运维经验：Kafka
团队没有 Kafka 运维经验，但要 Kafka 能力：Redpanda
MVP 阶段追求简单：NATS JetStream
```

---

# 5. NATS JetStream：MVP 可选，但不是最终首选

NATS JetStream 也可以做持久化消息流。NATS 官方文档说明 JetStream 是内置持久化引擎，可以存储消息并在之后 replay 给消费者。([NATS Docs][5])

它适合：

```text
MVP
低运维复杂度
服务间低延迟消息
中等规模事件流
```

但如果后续你们要做：

```text
大量链上日志消费
多策略回放
历史数据重算
复杂数据管道
数据分析接入
```

Kafka / Redpanda 更稳。

我的建议：

```text
MVP：NATS JetStream 可以接受
生产高吞吐版：Kafka / Redpanda 更合适
```

---

# 6. 分析数据库：ClickHouse

## 推荐用途

```text
链上 logs
交易事件
价格历史
token 行情快照
策略信号历史
风控事件
回测数据
系统性能指标的业务侧分析
```

ClickHouse 官方文档和项目说明都强调其 column-oriented DBMS 适合实时分析报告；它非常适合大规模事件、日志、行情、时间序列类分析。([ClickHouse][6])

## 为什么 Athena 需要 ClickHouse

PostgreSQL 适合订单和配置，不适合大量 append-only 链上事件分析。

比如这些查询：

```sql
过去 24 小时哪些 token 交易量突然放大？
某个 token 在买入前 10 分钟的流动性变化？
某个策略过去 7 天胜率是多少？
保护性卖出触发前价格下跌速度是多少？
某个钱包最近 1000 笔交易的确认延迟分布？
```

这些放 ClickHouse 更合适。

## 数据流推荐

```text
Block Sniffer
  -> Kafka
  -> ClickHouse Sink

Strategy Engine
  -> Kafka
  -> ClickHouse Sink

Order Controller
  -> PostgreSQL
  -> Kafka
  -> ClickHouse Sink
```

ClickHouse 用来查历史、分析、回测，不作为交易热路径依赖。

---

# 7. Projects 数据存储设计

你的架构里 `Projects` 是核心热数据源。

我建议：

```text
Projects runtime state：内存 + Redis
Projects snapshot：PostgreSQL
Projects history：ClickHouse
```

也就是：

```text
内存：策略引擎本地最快访问
Redis：跨服务共享最新状态
PostgreSQL：保存关键快照
ClickHouse：保存完整历史变化
```

## 示例

```text
Project Controller
  -> 更新本地内存 map
  -> 写 Redis 最新 project state
  -> 定期写 PostgreSQL snapshot
  -> 发 project.updated 到 Kafka
  -> Kafka sink 写 ClickHouse history
```

不要让 Strategy Engine 每次判断都查 PostgreSQL。

正确方式是：

```text
Strategy Engine 订阅 project.updated
维护本地内存状态
必要时从 Redis 补数据
几乎不访问 PostgreSQL
```

这符合你的性能优先原则。

---

# 8. Orders 数据存储设计

`Orders` 是用户资产和订单状态，必须可靠。

推荐：

```text
orders 当前态：PostgreSQL
order_events 事件流水：PostgreSQL + Kafka
orders 热状态：Redis
订单历史分析：ClickHouse
```

## 状态更新方式

不要到处直接改 `orders.status`。

建议统一通过：

```text
Order Controller
```

由它负责：

```text
校验状态流转是否合法
写 order_events
更新 orders 当前态
发 order.status.changed 事件
同步 Redis 热状态
```

这样可以避免多个服务乱改订单状态。

---

# 9. RPC 和服务通信：gRPC + Protobuf

## 推荐

服务之间使用：

```text
gRPC + Protobuf
```

适合：

```text
Strategy Engine -> Swap Server
Protect Server -> Swap Server
Swap Server -> Tx Speed Up Server
API Server -> Order Controller
API Server -> Project Controller
```

## 原因

* 性能比 JSON REST 更好。
* schema 明确。
* 适合内部服务调用。
* 多语言支持好，Rust / Go / TypeScript 都能接。
* 比 HTTP JSON 更适合低延迟内部通信。

## 外部 API

给前端用：

```text
REST：普通查询和控制接口
WebSocket：订单状态、价格、项目更新实时推送
```

不要让前端直接消费 Kafka。

---

# 10. 区块链 RPC 层

建议单独抽象一层：

```text
Chain RPC Client / Provider Manager
```

它负责：

```text
多 RPC 节点管理
健康检查
延迟统计
失败重试
限流
请求熔断
按 chain 分组
自动切换最快节点
```

## 推荐设计

```text
Block Sniffer
Swap Server
Tx Speed Up Server
Project Controller
```

都不要直接裸调用某个 RPC URL，而是通过统一 RPC provider manager。

## 需要缓存的数据

```text
latest_block
gas_price
base_fee
priority_fee
token metadata
pair address
pool state
wallet nonce
```

尤其 nonce 不要乱查，交易系统里 nonce 管理很容易出事故。

---

# 11. Observability：必须做，不是可选

推荐：

```text
Prometheus
Grafana
OpenTelemetry
Loki / Vector
Sentry
```

重点监控：

```text
block delay
event lag
Kafka consumer lag
strategy decision latency
swap create latency
tx confirmation latency
tx speed up count
protect trigger count
RPC error rate
RPC p95 / p99 latency
Redis latency
PostgreSQL slow query
ClickHouse insert lag
```

对 Athena 来说，最重要的性能指标不是普通 QPS，而是：

```text
从链上事件出现 -> 策略判断 -> 交易创建 -> 广播成功
```

这个端到端延迟要打点。

---

# 12. 推荐最终架构图

```text
                    UI
                     |
              API Server
            /     |      \
           /      |       \
  Project Query  Order Query  Manual Trade
       |           |            |
       v           v            v
    Redis     PostgreSQL     Swap Server
       |           ^            |
       |           |            v
       |      Order Controller  Tx Speed Up Server
       |           ^
       v           |
Block Sniffer -> Kafka / Redpanda -> Strategy Engines
       |              |              |
       |              |              v
       |              |          Swap Server
       |              |
       v              v
Project Controller  ClickHouse
       |
       v
Redis + In-Memory Projects
       |
       v
Buy / Sell Strategy Engine

Protect Server
   -> reads Redis / PostgreSQL
   -> emits protect.sell.requested
   -> Swap Server
   -> Tx Speed Up Server if urgent
```

---

# 13. 推荐技术栈表

| 模块                 | 推荐技术                 | 原因                |
| ------------------ | -------------------- | ----------------- |
| Block Sniffer      | Rust                 | 高吞吐、低延迟、无 GC      |
| Project Controller | Rust / Go            | 数据同步频繁，要求稳定       |
| Strategy Engine    | Rust                 | 交易判断热路径，性能优先      |
| API Server         | Go / TypeScript      | 开发效率、生态成熟         |
| Order Controller   | Go / Rust            | 状态一致性重要           |
| Swap Server        | Rust                 | 交易创建、签名、广播属于核心热路径 |
| Tx Speed Up Server | Rust                 | 对延迟敏感             |
| Protect Server     | Rust / Go            | 风控需要低延迟，但业务逻辑也较复杂 |
| Primary DB         | PostgreSQL           | 订单、策略、用户配置强一致     |
| Cache              | Redis                | 热状态、锁、低延迟读取       |
| MQ                 | Kafka / Redpanda     | 事件流、回放、高吞吐        |
| Analytics DB       | ClickHouse           | 链上事件、行情、策略历史分析    |
| Internal RPC       | gRPC                 | 高性能、schema 明确     |
| Frontend Realtime  | WebSocket            | 推送订单和行情状态         |
| Metrics            | Prometheus + Grafana | 性能监控              |
| Tracing            | OpenTelemetry        | 端到端延迟追踪           |
| Logs               | Loki / Vector        | 日志采集与排查           |
| Deployment         | Kubernetes           | 服务编排、扩缩容、容灾       |

---

# 14. MVP 版本建议

如果你们现在还在早期，不建议一上来堆太多东西。

## MVP 可以这样选

```text
Backend: Go + Rust
DB: PostgreSQL
Cache: Redis
MQ: NATS JetStream 或 Redpanda
Analytics: 暂时 PostgreSQL 分区表，后续迁 ClickHouse
Deploy: Docker Compose / 单机 Kubernetes
```

## MVP 不建议一开始就做太复杂

可以先不要：

```text
复杂 Kafka 集群
复杂 ClickHouse 集群
多区域部署
过度微服务拆分
```

但要保留架构边界。

也就是说，即使用单体或轻量服务，也要保留：

```text
Project Controller
Strategy Engine
Order Controller
Swap Server
Tx Speed Up Server
Protect Server
```

这些逻辑边界。

---

# 15. 生产版本建议

生产高性能版本：

```text
Language:
  Rust + Go

Database:
  PostgreSQL primary
  read replica optional

Cache:
  Redis Cluster

MQ:
  Redpanda / Kafka

Analytics:
  ClickHouse

RPC:
  gRPC

Realtime:
  WebSocket Gateway

Infra:
  Kubernetes
  Prometheus
  Grafana
  OpenTelemetry
  Loki
```

生产中建议：

```text
PostgreSQL 只存关键状态
Redis 存热数据
Kafka/Redpanda 存事件流
ClickHouse 存历史分析
Strategy Engine 本地内存做最快判断
```

---

# 16. 我最推荐的组合

如果只让我选一套，我会选：

```text
Rust + Go
PostgreSQL
Redis
Redpanda
ClickHouse
gRPC
WebSocket
Prometheus + Grafana + OpenTelemetry
Kubernetes
```

理由：

* Rust 保证交易核心链路性能。
* Go 保证 API 和控制服务开发效率。
* PostgreSQL 保证订单和配置一致性。
* Redis 保证热数据读取低延迟。
* Redpanda 提供 Kafka 能力但运维更轻。
* ClickHouse 承接链上事件和策略分析。
* gRPC 降低内部通信开销。
* WebSocket 满足前端实时状态。
* Observability 能帮助定位延迟和交易失败原因。

最终原则：

```text
交易热路径：内存 + Redis + gRPC
事实状态：PostgreSQL
事件流：Kafka / Redpanda
历史分析：ClickHouse
```

这是比较适合 Athena 的技术栈边界。

[1]: https://www.postgresql.org/docs/current/mvcc.html?utm_source=chatgpt.com "PostgreSQL: Documentation: 18: Chapter 13. Concurrency Control"
[2]: https://redis.io/docs/latest/?utm_source=chatgpt.com "Docs - Redis"
[3]: https://kafka.apache.org/?utm_source=chatgpt.com "Apache Kafka"
[4]: https://www.rabbitmq.com/docs?utm_source=chatgpt.com "RabbitMQ Documentation | RabbitMQ"
[5]: https://docs.nats.io/nats-concepts/jetstream?utm_source=chatgpt.com "JetStream | NATS Docs"
[6]: https://clickhouse.com/docs?utm_source=chatgpt.com "ClickHouse Docs | ClickHouse Docs"
