# 日志存储、消费与稳定查询

> 详细设计已于 2026-09-18 整体审阅通过，按用户要求暂不实施。入口：[总体设计](../2026-09-18-key-operation-logs-design.md)。

## 1. 数据库归属与迁移

复用 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` 指向的 PostgreSQL 数据库，在其中建立独立的 **operation_log schema**。所有日志表、索引、函数和迁移版本表均有 schema 限定名，不进入 public。

这是根据当前源码确定的边界：[账户 schema 校验](../../../../internal/accountstate/schema/schema.go)严格核对 `public.goose_db_version` 及 [public 对象目录](../../../../internal/accountstate/schema/catalog/catalog.go)。新增日志模块不能把另一套版本记录写进该表，也不能删弱现有账户校验。

目标独立入口 `cmd/athena-operation-log-migrate` 提供 `up`、`verify`；schema owner 为 operation-log。`up` 在有界、与现有迁移相同的数据库级 advisory lock 下先创建 operation_log schema，再使用已安装 Goose v3.25.0 的实例级 Provider／WithStore，版本表名为 `operation_log.goose_db_version`。不设置进程全局 goose TableName。`verify` 只读核对该 schema 的版本、表、索引、约束与函数；空库 verify 不创建任何对象。

Provider 的自定义 store 和会话锁能力以[官方文档](https://pressly.github.io/goose/documentation/provider/)及本机 v3.25.0 `provider_options.go` 为依据。迁移命令必须证明公共 schema 校验前后结果相同。

API producer 自有日志连接池；日志服务自有存储池和账户只读授权适配器。schema migration 关闭自己的连接，producer 和服务分别关闭自己的池，不借用业务 transaction。默认共用数据库意味着数据库不可用可能同时影响既有账户业务，这是已有共同依赖，不能宣称日志拥有独立数据库故障域。

采集入口是日志模块公开的持久消息协议：类型化 Go adapter＋限定 SQL，不是任意业务表访问。协议选择属于 SDS-R2 的明确例外；数据库认证识别生产者连接，服务查询仍有独立内部 RPC 鉴权。当前共享数据库账户的权限以部署配置为准，不声称应用 adapter 已提供数据库级最小权限隔离。

## 2. 持久实体

下表给出实施必须保持的列和约束。UUID 用 PostgreSQL uuid，时间用 timestamptz，计数和版本用 bigint，编码用带 CHECK 的 text，结构化负载用 jsonb。未明确允许 null 的字段必须非空。

| 表 | 列 | 约束／用途 |
| --- | --- | --- |
| `operation_log.event` | event_id、ingest_id、operation_id、phase、producer_id、schema_version、occurred_at、received_at、payload、payload_hash | event_id PK；ingest_id identity UNIQUE；UNIQUE(operation_id, phase)；payload ≤32 KiB；hash 为规范化字段编码的 SHA256；事件正文只追加 |
| `operation_log.delivery` | event_id、state、failure_count、next_attempt_at、processed_at nullable、reason_code nullable | event_id PK/FK event；state=PENDING/PROCESSED/QUARANTINED；与 event 在同一入箱事务写入 |
| `operation_log.publication` | singleton_id、last_seq、last_published_at nullable | singleton_id 固定 1；last_seq≥0；所有可查询投影发布的事务串行化边界 |
| `operation_log.entry_version` | operation_id、visible_from_seq、visible_to_seq nullable、started_at、finished_at nullable、actor_account_id nullable、actor_username nullable、actor_role、realm、credential_kind、module_code、action_code、outcome、observation、target_account_id nullable、primary_resource_type nullable、primary_resource_id nullable、request_id、parent_operation_id nullable、business_request_id nullable、duration_ms nullable、detail、source_event_ids | PK(operation_id, visible_from_seq)；visible_to_seq>visible_from_seq；每个 operation_id 只有一条 visible_to_seq=null 的当前版本；业务列由事件折叠产生，不允许应用编辑 |
| `operation_log.producer_status` | producer_id、started_at、last_seen_at、stopped_at nullable、snapshot_no、attempted_events、confirmed_events、unconfirmed_events、invalid_events、capacity_rejected_events、last_failure_at nullable、last_failure_code nullable、last_recovered_at nullable | producer_id PK；绝对计数非负；只接受更大的 snapshot_no，乱序旧上报不能覆盖新状态 |

不对账户、钱包、订阅、轮次设置删除级联外键。已处理事件和历史版本持续保留。delivery 的状态以及 entry_version 的可见区间属于投影维护元数据，可由日志进程更新；原始 event 和既有版本的业务正文保持不变。不存在用户／管理员编辑和删除接口，也不宣称能够阻止数据库管理员维护数据。

`detail` 保存完整的规范化事件视图，包括 actor、资源引用、effect、errors、数量及 completeness，前述列为筛选和排序投影。两者在同一事务写入并保持一致，不把任意上游 JSON 直接透传到 UI。

## 3. 索引

- event：`(operation_id, phase)` 唯一索引、ingest_id 唯一索引、`(received_at, event_id)`。
- delivery：state=PENDING 的 `(next_attempt_at, event_id)` 部分索引；state=QUARANTINED 的 `(event_id)` 部分索引。
- entry_version：`(operation_id, visible_from_seq DESC)`；当前版本的 operation_id 部分唯一索引；`(started_at DESC, operation_id DESC, visible_from_seq)`。
- 常用查询：`(actor_account_id, started_at DESC, operation_id DESC)`、`(lower(actor_username), started_at DESC, operation_id DESC)`、`(module_code, action_code, started_at DESC, operation_id DESC)`、`(outcome, started_at DESC, operation_id DESC)`、`(primary_resource_type, primary_resource_id, started_at DESC, operation_id DESC)`。
- 用户名支持不区分大小写的精确值和前缀，不做任意正文全文搜索；前缀索引按实际 SQL 配置 text_pattern_ops。不存在自动收集全文并建立全文索引的路径。

## 4. 入箱与幂等

producer 在一个短事务中插入 event 和 PENDING delivery。冲突时查询已有 event：eventId、operationId、phase 及规范化 payload_hash 相同则视为确认成功，不重复新建 delivery；相同键不同内容返回 EVENT_CONFLICT。收到 SQL 事务提交回执才算 confirmed。超时后仍可通过相同事件重试确认前次提交。

规范 hash 对固定字段顺序的编码计算，列表顺序有业务意义，details 字段排序；不能对任意 map 的不确定序列化结果计算。START 和 FINISH 使用不同 eventId，永远不能互相覆盖。

采集失败后业务继续，适配器更新本进程状态。无持久本地补记文件、无无限内存队列；因此“从未确认入箱”可能丢失，不能在重启后自动宣告已补齐。

## 5. 消费与事务发布

每个服务进程有一个 projector loop。可以运行多个实例，但一次只允许一个发布事务持有 publication 单行锁。空闲轮询 500 ms，每批最多 100 条、事务最长 2 秒。

1. 开始事务，尝试 `SELECT ... FROM publication WHERE singleton_id=1 FOR UPDATE SKIP LOCKED`。没有取得锁则本轮退出。
2. 按 received_at、ingest_id 选择到期 PENDING delivery，用行锁锁定本批。不能记一个 `last_ingest_id` 后跳过更早但晚提交的事件。
3. 校验 envelope、目录动作、actor 一致性和大小。无法识别的版本／动作、同一操作的冲突事实进入 QUARANTINED，保留事件与稳定原因；不清空队列或丢弃它们。
4. 对每个 operationId 读取已处理或本批验证通过的 START／FINISH，排除 QUARANTINED 及尚未验证的其他待处理事件。按 received_at、ingest_id 确定验证顺序；与既有有效事实冲突的后续事件隔离，不能在下一批又参与折叠。FINISH 的自包含事实优先；只有 START 时产生 UNKNOWN／START_ONLY。同批该操作有两阶段，只发布一份合并版本。
5. 有新的投影时在锁内取 nextSeq=last_seq+1；关闭旧版本的 visible_to_seq，插入新版本，标记本批 delivery 已处理，并更新 last_seq、last_published_at。全部在一个事务提交；无投影变化时只确认 delivery，不增加序号。
6. commit 成功后才对查询可见。进程崩溃时事务回滚，PENDING 仍可重新处理。数据库临时错误退避 1、2、4、8、16、30 秒，之后以 30 秒为上限持续重试；不把整个数据库错误归咎于单条事件并隔离所有队列。

确定的单条数据错误使用 savepoint 隔离该条并继续同批其他记录；事务／连接级错误回滚整个批次。quarantine 修复属于后续代码修复与运营范围，本版没有管理员强行改写事件的按钮。

`SKIP LOCKED` 仅用于队列与发布锁，绝不用于管理员列表查询。PostgreSQL 明确指出它会跳过锁定行，适用于队列但不适合一般一致性查询。[PostgreSQL SELECT](https://www.postgresql.org/docs/16/sql-select.html)

## 6. 状态折叠

| 已持久事实 | 查询结果 |
| --- | --- |
| START | UNKNOWN，observation=START_ONLY，显示结果待记录 |
| FINISH | FINISH 中结果，observation=FINISH_ONLY，详情说明开始记录未收录 |
| START＋FINISH 一致 | FINISH 中结果，observation=COMPLETE |
| FINISH 先到、START 晚到 | 可见结果保持 FINISH，下一版本补全 observation=COMPLETE |
| 原先未确认身份、FINISH 获得真实账户 | 新版本补充可信身份；旧查询快照保持当时视图 |
| 两阶段已验证账户／动作冲突 | 后到冲突事件隔离，保留先前视图，标注服务处理异常 |

没有结果事件不按时间合成 FAILED。第一版不追加交易最终结算事件；业务详情承接命令之后的异步生命周期。

## 7. 稳定快照分页

普通数据库 sequence 可以在事务提交前分配，且不会因事务回滚而回退；所以 `MAX(ingest_id)` 不能直接代表已提交的连续查询水位。采用 publication 行锁和同事务发布，是本设计针对该事实的选择。[PostgreSQL 事务隔离说明](https://www.postgresql.org/docs/16/transaction-iso.html)

首次查询读取已提交的 `last_seq=W`；所有页面只选择：

```sql
visible_from_seq <= W
AND (visible_to_seq IS NULL OR visible_to_seq > W)
```

再对这个快照内的版本应用用户、模块、动作、结果、对象和时间筛选，按 `(started_at DESC, operation_id DESC)` 排序。下一页增加 `(started_at, operation_id) < (last_started_at, last_operation_id)`，取 pageSize+1 确定是否还有下一页。筛选条件不得先作用于原始历史版本后再随意取最新行。

即使查询后某条 UNKNOWN 变成 SUCCEEDED，旧快照依然选中旧版本；新结果筛选不会使翻页漏项。晚到事件在 W 之后发布，只在 Refresh 新快照中出现。查询使用短只读事务，不跨 HTTP 请求保持数据库事务或连接。

cursor 包含 version=1、W、完整规范 filters（包括固定 from/to）、pageSize、最后排序键、viewerBinding、issuedAt、expiresAt，并使用独立的 32 字节 HMAC-SHA256 密钥签名。有效期 30 分钟；长度上限 4096 字节；校验失败返回 CURSOR_INVALID，过期返回 CURSOR_EXPIRED；禁止悄悄换快照继续下一页。后续页面沿用首次查询的 issuedAt／expiresAt，不通过翻页无限续期。

viewerBinding 为 API 提供的当前管理员 accountId、realm 和会话 JTI 的不可逆摘要绑定。新会话或不同管理员不能复用旧 cursor。原始 JTI 不存入日志或公开响应。cursor 签名只保证参数绑定，所有请求仍需重新鉴权。

详情可传 snapshotToken，读取 W 对应的同一版本；不传则读取最新版本。snapshotToken 包含 version=1、W、viewerBinding、issuedAt、expiresAt，以同一密钥签名并使用独立的 token kind 区分 cursor；有效期沿用该快照，不是授权凭据。详情提示当前看到的是哪次查询快照，可显式刷新到最新。Previous 使用客户端保存的真实输入 cursor；刷新页面时恢复 URL 中 cursor，不能自行推算页码或总条数。
