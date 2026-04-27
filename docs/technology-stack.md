
# Athena 技术选型总结：Rust、Go 与整体架构建议

## 1. 背景

Athena 是一个面向区块链交易场景的自动化交易系统，核心能力包括：

- 实时监听链上区块、交易和日志事件
- 维护 Project 市场数据
- 根据用户策略自动判断买入、卖出时机
- 创建 swap 交易
- 对 pending transaction 进行加速
- 维护订单生命周期
- 在风险出现时自动触发保护性卖出

由于 Athena 直接参与链上交易，系统设计必须遵循以下原则：

```text
Design and development across the entire project must prioritize runtime performance above all else.
```

也就是说：

```text
整个项目的设计和开发必须优先考虑运行时性能。
在区块链交易场景中，任何可避免的性能损耗都可能造成经济损失。
```

因此，技术选型不能只看开发效率，还要重点考虑：

* 运行时性能
* 延迟稳定性
* 并发安全
* 内存安全
* 交易执行可靠性
* 链上生态兼容性
* 长期可维护性

---

# 2. 总体结论

Athena 不建议简单地在 Rust 和 Go 之间二选一。

对于一个以 EVM / ETH 为主要目标的自动化交易系统，最合理的技术路线是：

```text
Go 做 EVM 生态适配和业务控制
Rust 做性能敏感的交易核心
TypeScript 做前端和管理后台
```

原因是：

```text
Go 更贴近 EVM / Geth 生态
Rust 更适合低延迟、高性能、无 GC 抖动的核心交易链路
TypeScript 更适合 UI、管理后台和策略配置界面
```

最终推荐：

```text
MVP 阶段：
  Go + PostgreSQL + Redis + Redpanda + REST/WebSocket

长期高性能生产版：
  Go + Rust + PostgreSQL + Redis + Redpanda/Kafka + ClickHouse + gRPC + WebSocket
```

一句话总结：

```text
MVP 用 Go 快速跑通 EVM 交易闭环；
长期用 Rust 承担慢一步就可能亏钱的核心模块。
```

---

# 3. 为什么 Athena 核心模块适合 Rust

优先考虑 Rust，不是因为 Rust 更高级，也不是因为 Rust 更流行，而是因为 Athena 的核心链路里：

```text
运行时性能
延迟稳定性
内存安全
并发可靠性
```

都会直接影响收益和风险。

一句话：

```text
Athena 的核心模块适合 Rust，因为它能在没有 GC 暂停的前提下，提供接近 C/C++ 的性能，同时显著降低内存错误和并发错误。
```

---

## 3.1 Rust 没有 GC，延迟更稳定

Athena 的关键交易链路是：

```text
Block Sniffer
-> Project Controller
-> Strategy Engine
-> Swap Server
-> Tx Speed Up Server
```

这些模块都对延迟非常敏感。

如果使用 Go、Java、Node.js 这类带 GC 的语言，虽然平均性能也可以很好，但在极端情况下可能出现：

```text
GC pause
内存回收抖动
tail latency 变高
p99 / p999 延迟不稳定
```

对于普通 Web 系统，这可能只是请求慢几十毫秒。

但对自动交易系统来说，几十毫秒可能意味着：

```text
更差的买入价格
错过最佳卖出窗口
保护性卖出慢一步
交易被别人抢先
gas 策略失效
```

Rust 没有 GC，内存释放由编译期所有权系统管理，因此更适合低延迟、延迟可控的交易热路径。

---

## 3.2 Rust 性能接近 C/C++，适合高吞吐链上事件处理

Block Sniffer 和 Project Controller 会持续处理大量链上数据：

```text
新区块
交易
logs
pending tx
price update
liquidity update
pool state
```

这些数据量可能非常大。

Rust 的优势包括：

```text
零成本抽象
高效内存布局
无运行时依赖
编译期优化强
CPU 利用率高
```

这意味着可以写出抽象良好的代码，同时不会像一些高级语言那样付出明显运行时成本。

对 Athena 来说，以下能力都属于 Rust 的优势区间：

```text
事件过滤
策略规则匹配
交易构造
交易签名
序列化 / 反序列化
RPC 批量请求
内存状态维护
高频数据结构更新
```

---

## 3.3 Rust 内存安全强，适合长期运行服务

Athena 不是跑一次就结束的脚本，而是长期运行的资金相关系统。

例如：

```text
Block Sniffer 需要 24/7 监听链上事件
Strategy Engine 需要持续维护 project state
Protect Server 需要持续监控用户资产风险
Tx Speed Up Server 需要持续跟踪 pending tx
```

长期运行服务最怕：

```text
内存泄漏
野指针
数据竞争
状态错乱
偶发崩溃
```

C++ 性能很强，但内存安全风险更高。

Go / Java 内存安全也不错，但有 GC 和更高运行时成本。

Rust 的价值在于：

```text
接近 C++ 的性能
更强的内存安全
没有 GC
并发错误更容易在编译期暴露
```

对交易系统来说，很多 bug 不是马上爆，而是在高并发、高压力、长期运行之后才出现。

Rust 的编译期约束可以提前拦截大量潜在问题。

---

## 3.4 Rust 并发模型更安全，适合多任务事件系统

Athena 会有大量并发任务：

```text
多个 chain 同时监听
多个 token 同时更新
多个 strategy 同时判断
多个 wallet 同时交易
多个 pending tx 同时加速
多个 order 同时状态流转
```

这些地方容易出现：

```text
重复买入
重复卖出
nonce 冲突
订单状态覆盖
缓存状态不一致
同一个 token 被重复 sync
同一个 order 被多个 worker 同时处理
```

Rust 的类型系统和所有权模型会强制开发者认真处理：

```text
共享状态能不能被多个线程同时访问
可变引用是否唯一
跨线程数据是否 Send / Sync
异步任务里的生命周期是否安全
```

在交易系统中，这类约束不是负担，而是保护。

因为一次并发 bug 可能导致：

```text
多发一笔交易
错过卖出
错误加速 nonce
资产状态错乱
订单状态错误
```

这些问题都可能带来真实经济损失。

---

## 3.5 Rust 适合构建高性能本地内存状态

Athena 的策略判断不应该每次都访问数据库。

更合理的设计是：

```text
Projects runtime state = 内存 + Redis
Orders hot state = Redis + 内存缓存
Strategy Engine 本地维护 project state
```

Strategy Engine 应该：

```text
订阅 project.updated
本地维护 HashMap / DashMap / lock-free structure
策略判断直接读内存
必要时才访问 Redis / PostgreSQL
```

Rust 很适合这种模式，因为它可以高效管理：

```text
HashMap
BTreeMap
DashMap
Arc
RwLock
channel
async stream
zero-copy buffer
```

这对低延迟策略判断非常关键。

---

## 3.6 Rust 适合交易构造、签名和底层链交互

Swap Server 和 Tx Speed Up Server 处理的是底层交易逻辑：

```text
构造 swap transaction
估算 gas
设置 nonce
签名交易
广播交易
替换 pending transaction
重发交易
处理 receipt
解析 revert reason
```

这些逻辑对正确性和性能要求都非常高。

Rust 在这类场景中的优势包括：

```text
强类型适合表达交易结构
错误处理显式
序列化效率高
适合写底层 SDK / client
适合控制内存分配
适合精细优化 RPC 请求
```

尤其是错误处理，Rust 要求显式处理：

```text
Result<T, E>
Option<T>
```

这比很多语言里随手抛异常、返回 null 更适合交易系统。

交易系统里“没处理的异常”很危险，可能导致：

```text
订单状态没更新
交易已发但系统认为失败
保护逻辑没继续执行
pending tx 没进入加速队列
```

Rust 会强制开发者把这些状态设计清楚。

---

## 3.7 Rust 更符合 Athena 的性能优先文化

Athena 已经明确要求：

```text
Design and development across the entire project must prioritize runtime performance above all else.
```

Rust 和这个原则高度匹配。

Rust 会让团队自然关注：

```text
内存分配
数据结构
锁竞争
拷贝成本
异步任务调度
错误边界
状态所有权
```

这和 Athena 的项目性质一致。

如果用 Node.js / Python，团队很容易写出开发快但运行成本高的代码。

如果用 Go，整体性能不错，但在极致低延迟和内存控制方面不如 Rust。

因此可以这样判断：

```text
Go 适合效率优先的服务
Rust 适合性能优先的核心交易链路
```

Athena 的核心路径显然更偏后者。

---

# 4. 为什么不是全项目 Go

Go 是非常优秀的工程语言，尤其适合 EVM 生态和服务端业务开发。

但如果把 Go 作为所有核心热路径的第一选择，需要注意它的一些限制。

---

## 4.1 Go 的优势

Go 的优势包括：

```text
开发快
部署简单
并发模型友好
生态成熟
团队招聘容易
写 API 和 worker 很舒服
和 go-ethereum 集成方便
```

Go 很适合：

```text
API Server
Order Controller
Admin Service
内部后台服务
普通异步任务
配置管理服务
WebSocket Gateway
EVM Adapter
Project Controller
```

---

## 4.2 Go 在交易热路径上的不足

对 Athena 核心交易链路来说，Go 的不足包括：

```text
有 GC
内存布局控制弱于 Rust
泛型和类型表达能力弱于 Rust
极致性能优化空间小于 Rust
并发数据竞争仍需要非常小心
```

Go 可以做交易系统，而且在 MVP 阶段非常适合快速落地。

但如果第一原则是：

```text
runtime performance above all else
```

那么交易热路径上 Rust 更合适。

---

# 5. 为什么不是 C++

C++ 性能也非常强，甚至在某些场景可以做到极致。

但 Athena 不优先推荐 C++，主要原因是：

```text
内存安全风险更高
并发 bug 更难排查
工程复杂度更高
长期维护成本更高
新人写出危险代码的概率更高
```

Athena 是资金相关系统，不只是要跑得快，还要稳定、可维护、少出错。

可以理解为：

```text
C++：性能很强，但安全成本高
Go：工程效率高，但极致性能和延迟控制弱一些
Rust：性能强，延迟稳定，安全性好，但开发门槛更高
```

所以核心链路更适合 Rust，而不是 C++。

---

# 6. 为什么不是 Node.js / TypeScript 做核心后端

TypeScript 很适合：

```text
前端
管理后台
配置平台
非核心 API
内部工具
```

但不适合作为 Athena 的核心交易路径语言。

原因包括：

```text
单线程事件循环容易被阻塞
GC 带来延迟抖动
CPU 密集任务能力弱
内存控制弱
高频链上事件处理压力大
```

如果用 Node.js 写：

```text
Block Sniffer
Strategy Engine
Swap Server
Tx Speed Up Server
Protect Server
```

早期 MVP 可能能跑，但一旦事件量上来，容易遇到性能瓶颈。

尤其是 Protect Server 这种模块，慢一步可能就是损失。

---

# 7. EVM / ETH 场景下为什么 Go 权重要提高

如果 Athena 当前主要面向 EVM / ETH，那么 Go 的权重确实要提高。

原因是：

```text
Geth / go-ethereum 是 Go 生态
```

Geth 是 Ethereum 非常核心的 execution client，而 go-ethereum 生态提供了非常成熟的 EVM 工程能力。

这意味着 Go 在 EVM 应用层有明显优势。

---

## 7.1 Go 在 EVM 生态的优势

Go 可以直接使用 go-ethereum 中的能力，例如：

```text
ethclient
rpc
core/types
common.Address
crypto
accounts/abi
event filters
transaction / receipt types
```

因此在这些场景里 Go 非常顺手：

```text
监听 logs
解析 receipt
解析 event ABI
调用 debug_traceTransaction
调用 txpool_content
读取 pending nonce
构造 EIP-1559 transaction
处理 access list / blob tx 等 EVM transaction type
```

所以对于 EVM / ETH 生态，Go 很适合：

```text
Block Sniffer
EVM Adapter
Project Controller
Order Controller
API Server
WebSocket Gateway
Swap Server 第一版
```

原因是：

```text
1. 直接使用 go-ethereum
2. 解析 block / tx / logs / receipt 更自然
3. ABI decode、交易构造、签名、RPC 调用更成熟
4. 和 Geth 的对象模型更贴近
5. 工程落地速度快
```

---

## 7.2 但 Geth 是 Go 写的，不代表 Athena 必须全 Go

Geth 是 execution client，主要职责是：

```text
区块同步
交易池
EVM 执行
状态管理
区块验证
JSON-RPC 服务
```

Athena 是上层自动交易系统，主要职责是：

```text
监听链上事件
维护项目状态
判断买卖时机
构造交易
管理订单
加速交易
保护性卖出
```

两者相邻，但不是同一个问题。

Geth 用 Go 写，说明：

```text
Go 在 EVM 节点实现和 EVM 工程生态里很成熟
```

但 Athena 还要解决：

```text
低延迟策略判断
高频事件处理
订单状态一致性
nonce 管理
交易广播竞争
保护性退出速度
```

这些问题里，Rust 仍然有明显优势，尤其是在低延迟和无 GC 抖动方面。

---

# 8. Rust 对区块链开发是否全面

结论：

```text
Rust 对区块链开发非常全面，但不是在所有生态里都是最省事的选择。
```

更准确地说：

```text
Rust 在区块链底层、节点、执行客户端、高性能交易系统、Solana/Substrate/CosmWasm 等生态里非常强。

但在 EVM 应用层，Go 和 TypeScript 仍然有一些工程便利性优势。
```

---

## 8.1 Rust 在区块链开发中的覆盖范围

Rust 已经覆盖很多区块链核心场景：

```text
区块链节点 / execution client
RPC client
交易构造与签名
智能合约开发
链上程序开发
高频链上数据处理
本地开发工具链
MEV / trading bot / indexer
跨链 / Cosmos / Polkadot / Solana 生态
高性能后端服务
```

所以如果问：

```text
Rust 能不能做区块链开发？
```

答案是：

```text
能，而且非常适合。
```

但如果问：

```text
Rust 是不是所有区块链场景最成熟、最方便？
```

答案是：

```text
不是，要看具体生态。
```

---

## 8.2 Rust 在非 EVM 生态非常强

Rust 在以下生态中非常核心：

```text
Solana
Substrate / Polkadot
CosmWasm
```

如果 Athena 未来支持 Solana，那么 Rust 权重应该明显提高。

如果 Athena 未来支持多链，更合理的设计是：

```text
Core Engine: Rust
EVM Adapter: Go 或 Rust Alloy
Solana Adapter: Rust
Cosmos Adapter: Rust
API / Admin: Go
Frontend: TypeScript
```

也就是：

```text
Core trading engine 用 Rust
不同链的 adapter 按生态选择语言
```

---

## 8.3 Rust 在 EVM 生态也越来越成熟

虽然 EVM 生态传统上更偏：

```text
Go：节点 / geth / 后端集成
TypeScript：dApp / 脚本 / ethers.js / hardhat
Solidity：智能合约
```

但 Rust 在 EVM 生态也已经非常重要。

代表性项目包括：

```text
Reth：Rust 实现的 Ethereum execution client
Foundry：Rust 写的 Ethereum 开发工具链
Alloy：Rust 版 Ethereum / EVM 交互库
revm：Rust EVM implementation
```

所以 Rust 不是不能做 EVM，实际上已经可以做得很深。

只是 EVM 应用层如果大量依赖 Geth 语义，Go 仍然会更顺手。

---

# 9. 推荐语言分层

## 9.1 Athena EVM 第一版

如果 Athena 当前主要做 ETH / EVM，第一版建议：

```text
Go:
  Block Sniffer
  EVM Adapter
  Project Controller
  Order Controller
  API Server
  WebSocket Gateway
  Swap Server

Rust:
  Buy Strategy Engine
  Sell Strategy Engine
  Protect Engine
  Tx Speed Up Scheduler
  Hot State Runtime

TypeScript:
  Frontend
  Admin Dashboard
```

这个方案的重点是：

```text
Go 快速打通 EVM 交易闭环
Rust 承担性能敏感核心逻辑
TypeScript 负责前端和配置界面
```

---

## 9.2 长期高性能生产版

长期高性能版本建议：

```text
Go:
  EVM Adapter
  Project Controller
  Order Controller
  API Server
  WebSocket Gateway

Rust:
  Core Trading Engine
  Strategy Engine
  Protect Engine
  Swap Execution Worker
  Tx Speed Up Engine
  Hot State Runtime
  Gas / Nonce Manager

TypeScript:
  UI
  Strategy Config Panel
```

长期架构的核心思想是：

```text
EVM 生态接入层：Go
业务控制层：Go
交易决策核心：Rust
风控保护核心：Rust
热状态和高频执行：Rust
前端和后台：TypeScript
```

---

## 9.3 如果只能选一种后端语言

如果团队只能接受一种后端语言，那么建议按阶段判断。

### MVP 阶段

如果当前目标是快速验证 Athena 闭环，并且主要做 EVM：

```text
优先选 Go
```

原因：

```text
1. Geth / go-ethereum 是 Go 生态，集成更顺手
2. ethclient、types、abi、crypto、txpool、debug API 等工具成熟
3. 开发速度快
4. 团队招聘和维护成本低
5. 更适合快速验证业务闭环
```

### 高性能生产阶段

如果团队 Rust 能力强，并且目标是极致性能：

```text
核心交易路径优先 Rust
```

尤其适用于：

```text
高频交易
MEV-like 场景
对 p99 / p999 延迟极度敏感
愿意接受更高开发成本
核心交易路径要长期极致优化
```

---

# 10. 模块级语言选择标准

最核心的判断标准是：

```text
如果这个模块慢 50ms 会影响交易收益或风险，就优先 Rust。
如果这个模块只是管理、查询、配置、展示，就可以 Go / TypeScript。
```

按照这个标准：

```text
Block Sniffer：
  EVM 第一版可用 Go
  高频多链或极致性能场景可用 Rust

Project Controller：
  EVM 数据适配用 Go 更自然
  核心同步 worker 可后续 Rust 优化

Buy Strategy Engine：
  Rust

Sell Strategy Engine：
  Rust

Protect Server / Protect Engine：
  Rust

Swap Server：
  第一版 Go
  后续高性能交易执行 worker 可 Rust

Tx Speed Up Server：
  调度和核心执行建议 Rust

Order Controller：
  Go

API Server：
  Go

WebSocket Gateway：
  Go

Admin Backend：
  Go / TypeScript

Frontend：
  TypeScript
```

---

# 11. 数据库、缓存、MQ 与基础设施选型

除了语言，Athena 还需要合理的数据和消息架构。

推荐技术栈：

```text
Primary DB: PostgreSQL
Cache: Redis
MQ / Event Stream: Redpanda 或 Kafka
Analytics DB: ClickHouse
Internal RPC: gRPC
External API: REST + WebSocket
Observability: Prometheus + Grafana + OpenTelemetry
Deployment: Docker / Kubernetes
```

---

## 11.1 PostgreSQL：主数据库

PostgreSQL 用于可靠落库，适合保存强一致状态。

推荐存储：

```text
users
wallets
strategies
orders
order_events
transactions
risk_events
system_configs
```

重点区分：

```text
orders = 当前订单状态
order_events = 订单事件流水
transactions = 链上交易记录
```

订单状态、交易历史、策略配置不能只放缓存。

Athena 应该使用 PostgreSQL 作为事实状态来源之一，尤其是订单和交易相关数据。

---

## 11.2 Redis：缓存和热状态

Redis 用于低延迟读取和热状态维护。

适合存储：

```text
Project 最新状态
token 最新价格
最新 block height
pending tx 状态
nonce lock
order hot state
RPC 节点健康状态
分布式锁
```

Redis 的定位是：

```text
热状态 / 缓存 / 短期状态 / 分布式锁
```

但 Redis 不应该作为订单和交易的唯一事实源。

正确边界是：

```text
Redis = 性能加速层
PostgreSQL = 订单和配置事实数据
Kafka/Redpanda = 事件事实流
ClickHouse = 历史分析数据
```

---

## 11.3 Kafka / Redpanda：事件流

Athena 更适合使用 Kafka / Redpanda，而不是 RabbitMQ 作为主事件流。

原因是 Athena 需要：

```text
高吞吐
事件保留
事件回放
多个 consumer group
故障后 offset 恢复
```

推荐事件 topic：

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

RabbitMQ 更适合普通任务队列，而 Athena 更需要事件流、回放和高吞吐 fanout，因此不建议作为主 MQ。

如果团队希望降低 Kafka 运维复杂度，可以优先考虑：

```text
Redpanda
```

它保留 Kafka 生态能力，同时部署和维护相对更轻。

---

## 11.4 ClickHouse：链上历史和分析

ClickHouse 用于大量 append-only 历史数据和分析查询。

适合存储：

```text
链上 logs
价格历史
交易事件
策略信号历史
风控事件
回测数据
token 行情快照
```

例如这些查询更适合 ClickHouse：

```text
过去 24 小时哪些 token 交易量突然放大？
某个 token 在买入前 10 分钟的流动性变化？
某个策略过去 7 天胜率是多少？
保护性卖出触发前价格下跌速度是多少？
某个钱包最近 1000 笔交易的确认延迟分布？
```

PostgreSQL 存关键状态。

ClickHouse 存大量历史分析数据。

---

## 11.5 gRPC：内部服务通信

内部服务之间建议使用：

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

优势：

```text
性能比 JSON REST 更好
schema 明确
适合内部服务调用
多语言支持好
Rust / Go / TypeScript 都能接
```

---

## 11.6 REST + WebSocket：外部 API

前端和管理后台建议使用：

```text
REST：普通查询和控制接口
WebSocket：订单状态、价格、项目更新实时推送
```

不要让前端直接消费 Kafka / Redpanda。

---

## 11.7 Observability：必须做

Athena 必须建设完整可观测性，不是可选项。

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
Kafka / Redpanda consumer lag
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

对 Athena 来说，最重要的性能指标是：

```text
从链上事件出现 -> 策略判断 -> 交易创建 -> 广播成功
```

这个端到端延迟必须打点。

---

# 12. 推荐整体架构

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
Block Sniffer -> Redpanda/Kafka -> Strategy Engines
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

# 13. Projects 与 Orders 的存储边界

## 13.1 Projects

Projects 是 token / project 级别的市场数据，用于策略判断。

推荐存储方式：

```text
Projects runtime state：内存 + Redis
Projects snapshot：PostgreSQL
Projects history：ClickHouse
```

数据流：

```text
Project Controller
  -> 更新本地内存 map
  -> 写 Redis 最新 project state
  -> 定期写 PostgreSQL snapshot
  -> 发 project.updated 到 Redpanda/Kafka
  -> Sink 写 ClickHouse history
```

Strategy Engine 不应该每次判断都查 PostgreSQL。

正确方式是：

```text
Strategy Engine 订阅 project.updated
维护本地内存状态
必要时从 Redis 补数据
几乎不访问 PostgreSQL
```

---

## 13.2 Orders

Orders 是用户资产和订单状态，必须可靠。

推荐存储方式：

```text
orders 当前态：PostgreSQL
order_events 事件流水：PostgreSQL + Redpanda/Kafka
orders 热状态：Redis
订单历史分析：ClickHouse
```

状态更新应统一通过：

```text
Order Controller
```

Order Controller 负责：

```text
校验状态流转是否合法
写 order_events
更新 orders 当前态
发 order.status.changed 事件
同步 Redis 热状态
```

不要让多个服务直接随意修改 `orders.status`。

---

# 14. MVP 版本建议

如果当前还在早期阶段，不建议一开始就把架构做得过重。

MVP 推荐：

```text
Language:
  Go 为主
  Rust 用于最核心策略模块，或先预留边界

Database:
  PostgreSQL

Cache:
  Redis

MQ:
  Redpanda

API:
  REST + WebSocket

Deployment:
  Docker Compose / 单机 Kubernetes
```

MVP 先跑通：

```text
监听链上事件
同步 Project
触发策略
创建交易
加速交易
更新订单
保护性卖出
```

MVP 阶段可以暂缓：

```text
复杂 Kafka 集群
复杂 ClickHouse 集群
多区域部署
过度微服务拆分
全量 Rust 化
```

但必须保留逻辑边界：

```text
Block Sniffer
Project Controller
Strategy Engine
Order Controller
Swap Server
Tx Speed Up Server
Protect Server
```

即使早期是单体或轻量服务，也要保留这些边界，方便后续拆分和优化。

---

# 15. 长期生产版本建议

长期高性能生产版推荐：

```text
Language:
  Go + Rust + TypeScript

Database:
  PostgreSQL primary
  PostgreSQL read replica optional

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

Observability:
  Prometheus
  Grafana
  OpenTelemetry
  Loki
```

生产中建议：

```text
PostgreSQL 只存关键状态
Redis 存热数据
Redpanda/Kafka 存事件流
ClickHouse 存历史分析
Strategy Engine 本地内存做最快判断
```

---

# 16. 最终推荐技术栈

如果只选一套长期目标技术栈，建议：

```text
Backend:
  Go + Rust

Frontend:
  TypeScript

Primary DB:
  PostgreSQL

Cache:
  Redis

Event Stream:
  Redpanda / Kafka

Analytics DB:
  ClickHouse

Internal Communication:
  gRPC + Protobuf

External API:
  REST + WebSocket

Observability:
  Prometheus + Grafana + OpenTelemetry + Loki

Deployment:
  Docker + Kubernetes
```

模块分工：

```text
Go:
  EVM Adapter
  Block Sniffer 第一版
  Project Controller
  Order Controller
  API Server
  WebSocket Gateway
  Swap Server 第一版

Rust:
  Core Trading Engine
  Buy Strategy Engine
  Sell Strategy Engine
  Protect Engine
  Tx Speed Up Engine
  Hot State Runtime
  Gas Manager
  Nonce Manager
  高频事件过滤器
  后续高性能 Swap Execution Worker

TypeScript:
  Frontend
  Admin Dashboard
  Strategy Config UI
```

---

# 17. 最终结论

Athena 当前如果主要做 EVM / ETH：

```text
第一版建议 Go 为主。
```

原因是：

```text
Go 更贴近 Geth / go-ethereum 生态
EVM RPC、ABI、receipt、txpool、debug API 集成更顺手
开发速度更快
更适合 MVP 快速落地
```

但从长期高性能交易系统角度：

```text
Rust 应该作为核心性能模块语言。
```

原因是：

```text
Rust 没有 GC，延迟更稳定
Rust 性能接近 C/C++
Rust 内存安全强
Rust 并发模型更安全
Rust 更适合本地高性能内存状态
Rust 更适合慢一步就可能亏钱的交易热路径
```

最终架构建议：

```text
EVM 生态接入层：Go
业务控制层：Go
交易决策核心：Rust
风控保护核心：Rust
热状态和高频执行：Rust
前端和后台：TypeScript
```

一句话总结：

```text
MVP 用 Go 快速跑通 EVM 交易闭环；
长期用 Rust 承担慢一步就可能亏钱的核心模块。
```
