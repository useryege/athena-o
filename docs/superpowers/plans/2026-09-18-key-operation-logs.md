# 用户关键操作日志系统实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 完整实现已批准的关键操作日志，覆盖 74 类动作、98 个采集位置，管理员能够以稳定快照查询真实观察结果。

**架构：** API 的窄 producer adapter 将 START／FINISH 写入独立 `operation_log` schema 的持久收件箱；独立服务串行发布可见历史版本，通过内部鉴权 gRPC 接受管理员查询。API 继续原业务鉴权和调用，日志故障只影响日志；React 管理员页面独立读取记录、处理状态和当前 API 采集状态。

**技术栈：** Go 1.27.1、PostgreSQL、pgx、Goose v3.25.0、sqlc、gRPC／grpc-gateway、React／TypeScript、Jest、Playwright。

**规格：** `docs/superpowers/specs/2026-09-18-key-operation-logs-design.md` 及其 `2026-09-18-key-operation-logs/` 全部附录；设计提交 `033c6e81`。长期需求 `docs/requirements/observability/key-operation-logs.md`，长期设计 `docs/design/observability/operation-logs.md`。

## 全局约束

- 用户于 2026-09-18 明确恢复并授权实施；历史暂停记录保留为历史，当前不再等待相同设计批准。
- 从 `033c6e81` 新建 `codex/key-operation-logs`，工作区 `/home/yege/work/athena/.worktrees/key-operation-logs`；根目录 `rf4` 保持原样。只在此工作区提交，不推送、合并或部署生产。
- 事件目录固定 74 类动作、98 个位置（37 gRPC、61 HTTP／认证，含 12 个仅失败位置）；按目录逐个显式接入，不能根据 POST 或方法名猜测。
- 字段只从已验证身份和动作白名单提取；不录正文、私钥、bearer、签名、OAuth code、Cookie、IP、User-Agent。标识及大整数在公共 JSON／UI 中保持字符串。
- 默认 UNKNOWN，有业务事实才确认 SUCCEEDED、ACCEPTED、FAILED、DENIED、PARTIAL、ACTION_REQUIRED、CANCELLED；START_ONLY 不随时间推断失败。FINISH 自包含，注册提交与后续登录分别记录。
- 每阶段最多 2 次、单次最多 100 ms、合计 200 ms；同请求含所有子动作共享 400 ms；日志池最多 4 连接，16 并发，无无界内存队列。补记只重试同一事件，不重执行业务。
- 每事件 32 KiB、details 16 KiB、100 资源、32 effects；资源省略保留计数与 completeness，UTF-8 边界截断。冲突、无效、容量拒绝、提交未确认分别统计，原子快照。
- schema 与 Goose 版本表仅属 operation_log；保持 public 账户目录验证不变。verify 只读，up 持共同迁移 advisory lock；不设置全局 Goose 表名。
- projector 500 ms／100 条／2 s 事务，publication 行锁序列和历史版本发布原子；退避 1/2/4/8/16/30 s。不以 ingest_id 最大值跳过晚提交。
- 默认 7 天／50 条、最大 90 天／100 条；cursor ≤4096 字节、30 分钟、不续期、32 字节独立 HMAC，绑定管理员账户、realm、会话摘要、筛选与快照；详情独立 token kind，无伪造 total。
- 所有查询都要求交互管理员凭据；内部 token 与持久 administrator／login_enabled 二次校验。查询不受业务模块开关控制，也不产生操作日志。
- 独立服务最小依赖仅 PostgreSQL、account schema、operation_log schema；默认 127.0.0.1:8124。外部只经过 API Server，不新增公网业务端口。
- 生成源稳定后运行 `make sqlc-local`、`make protogen`，禁止手改生成物。按 SDS-R1–R8 记录实际证据。
- 逐任务 TDD、独立任务审阅，末尾整分支 AI 审阅；V01–V18 逐项写实际证据，隔离测试不代替真实环境 V18。
- 采用现有 Nansen 单一深色、Inter／JetBrains Mono、管理员 read scope；英文 UI，中文计划和交付。
- 停止本任务创建的临时进程／容器，保留数据、日志和报告；不停止已有实例。累计超过 600 秒时主代理最终回复前按 AGENTS.md 固定文案通知一次。

## 文件与依赖分工

| 任务 | 主要文件／职责 | 依赖 |
| --- | --- | --- |
| 1 | `internal/operationlog/event/`、`ingest/`、`record/`：事件、目录、白名单、结果观察与有界采集 | 已批准目录 |
| 2 | `internal/operationlog/store/`、`schema/`、`cmd/athena-operation-log-migrate/`、`sqlc.yaml`：持久协议、迁移、投影 | 任务 1 |
| 3 | `internal/operationlog/query/`、`access/`、`api/`、`transport/`、`apiclient/`、`cmd/athena-operation-log/`、公共 facade/proto：查询和服务 | 任务 1、2 |
| 4 | `internal/server/authz.go`、`athena-server.go`、37 项 facade：gRPC 采集与 API 服务接线 | 任务 1–3 |
| 5 | 目录 61 个 HTTP／认证位置及 `util/session/` 必要提交 hook：原生采集 | 任务 1、4 |
| 6 | `ui/src/app/admin/`、共享请求 registry、路由／导航／样式：管理员列表与详情 | 任务 3、4 |
| 7 | `internal/devruntime/`、migration all、Makefile、Dockerfile／Compose：独立编排 | 任务 2–4 |
| 8 | Go／PostgreSQL／Playwright 验证、长期文档、人工审查材料：全链路交付 | 任务 1–7 |

后续任务以此前已提交的导出类型为实际接口，不重复定义同一 DTO。若命名需要微调，更新此计划接口说明与调用者；语义必须与规格一致。

### Task 1: 类型化事件、结果观察与有界采集

**文件：** 新增 `internal/operationlog/event/{event.go,catalog.go,validation.go,catalog.json,event_test.go}`、`internal/operationlog/ingest/{producer.go,status.go,producer_test.go}`、`internal/operationlog/record/{recorder.go,context.go,recorder_test.go}`。

**接口：** `event.Event` 表达规格 envelope；`event.Validate(Event) error`，`event.Canonical(Event) ([]byte,error)`；`ingest.Sink` 的 `Append(context.Context,event.Event) error`、`PublishStatus(context.Context,ingest.Status) error` 供任务 2 实现。`record.WithRequest(context.Context,*ingest.Producer) context.Context` 建立共享 requestId／预算；`record.Begin(ctx, actionCode)` 创建 recorder；通过 context 注入，提供 nil-safe 身份、资源、派发、效果、详情及协议结果观察方法，`Finish` 至多调用一次。名字可按包风格调整，但报告必须给实际签名。

- [ ] 先写事件非法字段、未验证账户、结果折叠、目录数量和重试／预算／容量失败用例，运行失败证据。

```go
func TestCatalogCoverage(t *testing.T) {
    entries := event.Catalog()
    actions := map[string]bool{}
    for _, e := range entries { actions[e.ActionCode] = true }
    if len(entries) != 98 || len(actions) != 74 { t.Fatalf("entries=%d actions=%d", len(entries), len(actions)) }
}
```

- [ ] 从批准 JSON 复制编译期目录，测试逐字段保持一致；事件验证按目录收敛字段类型、权限对象键、数字、UUID、枚举、大小、UTF-8、presence。规范 hash 固定字段结构、details 排序。
- [ ] 实现 producer：mutex 原子状态、有限并发、独立超时、同 eventId 重试、同请求预算；Sink 失败／panic 不影响调用者；5 秒上报与有界 Close。失败日志固定代码、30 秒限频。
- [ ] 实现 recorder：默认 UNKNOWN、单调 duration、FINISH 自包含、父子关系、nil/no-op 安全、仅失败初始化模式、提交事实优先、响应失败保留 effect。不要在基础层猜测 HTTP 2xx 等于业务成功。
- [ ] 执行 `go test -race ./internal/operationlog/event ./internal/operationlog/ingest ./internal/operationlog/record`；覆盖 V01、V06、V07、V14、V15 的基础层。
- [ ] 提交 `feat(operation-log): add event contracts and bounded recorder`，写明红绿证据及实际接口，交独立审阅。

### Task 2: 独立 schema、持久收件箱及原子投影

**文件：** 新增 `internal/operationlog/store/migrations/`、`store/queries/`、`store/sqlc/`、`store/{store.go,ingest.go,projector.go,status.go,*_test.go}`、`internal/operationlog/schema/`、`cmd/athena-operation-log-migrate/`；修改 `sqlc.yaml`。

**接口：** 消费任务 1 `event.Event`、`ingest.Status`，`store.Store` 实现 `ingest.Sink`；导出 `Open(ctx,dsn)`、`Close()`、`Project(ctx)`，查询只读访问同一 store。schema 导出 `Up(ctx,dsn)` 和 `Verify(ctx,pool)`；不得导入 API runtime。

- [ ] 在独立测试数据库先写集成用例：verify 空库不创建；up 前后 public catalog 与版本字节一致；同 eventId／operation+phase 同正文确认、不同正文冲突；START／FINISH 正反序、一阶段、冲突隔离。

```sql
SELECT visible_from_seq, visible_to_seq, outcome
FROM operation_log.entry_version
WHERE operation_id = $1 AND visible_from_seq <= $2
  AND (visible_to_seq IS NULL OR visible_to_seq > $2);
```

- [ ] 一次完成五表、约束、索引和查询源，再运行 `make sqlc-local`，检查仅预期生成差异。表列与索引逐项遵守 storage-and-query.md。
- [ ] 使用 Goose Provider／WithStore 限定版本表、共同有界 advisory lock；verify 用只读事务核对版本、表、索引、约束、函数，不改 public 验证器。
- [ ] 实现 append 原子事务和 hash 幂等，producer_status 仅接受更大 snapshot_no；服务自有连接池和 producer 专用池分离。
- [ ] 实现 publication `FOR UPDATE SKIP LOCKED`、到期队列、同批折叠、排除隔离／未验证事件、savepoint 单条错误、历史区间和 delivery 同事务发布。循环处理暂时故障与退避由任务 3 runtime 消费。
- [ ] 真实 PostgreSQL 验证 V08、V09、V10、V12：低 ingest_id 晚提交、双 projector、事务回滚、重启、积压、临时数据库错误不整体隔离。测试命令与 DSN 配置写报告，测试库仅归本任务。
- [ ] 提交 `feat(operation-log): persist and publish operation history` 并独立审阅。

### Task 3: 稳定查询、内部服务与管理员公共协议

**文件：** 新增 `internal/operationlog/{query,access,transport,apiclient,api}/`、`cmd/athena-operation-log/`、`internal/server/operationlog/{operationlog.proto,server.go,server_test.go}`；修改生成配置／Swagger registry 的必要源，生成 `pkg/apiclient/operationlog/`。

**接口：** 内部 package `operationlog.internal.v1` 提供 ListOperationLogs、GetOperationLog、GetOperationLogRuntimeStatus；公共 package `operationlog` 的 OperationLogService 同三方法及 GetOperationLogCaptureStatus、ListOperationLogActions。消息／URL／presence／Viewer 完整遵守 api-and-admin-ui.md，不使用任意 JSON payload 代替类型化公开 DTO。

- [ ] 先写 filter、签名与服务权限测试：时间成对／90 天、目录组合、分页上限；篡改、过期、会话变更、不同管理员、token kind、参数变更均拒绝。

```go
// 不同会话的摘要必须使同一管理员的旧游标无效。
if _, err := codec.Decode(cursor, otherSessionViewer); err == nil {
    t.Fatal("cursor accepted for another session")
}
```

- [ ] 定义完整 proto 字段，按 grpc-rpc-naming 检查方法名，稳定后执行 `make protogen`；同步生成客户端／gateway／Swagger，禁止手改生成结果。
- [ ] 查询采用 W 内历史版本后筛选、keyset pageSize+1；首查询固定 from/to、issued/expires；返回 appliedFilters 和 snapshot，详情同 W／最新分支明确。所有查询有 deadline 和更短 statement_timeout。
- [ ] access 只读核验持久 administrator 和 login_enabled；transport 校验内部 bearer、Viewer 和 64 KiB 容量；错误带 ErrorInfo 固定 reason，不透传 Go error。角色撤销及账户库故障分别 403／503。
- [ ] 独立服务只验证 schema，健康反映依赖；schema 恢复后重新验证并恢复消费；500 ms poll、暂时错误退避、30 秒停止，TLS 与 Secret resolver 采用既有模式。
- [ ] 公共 facade 接受可信 Viewer resolver，当前 capture／目录不依赖服务；缺配置的客户端返回不可用，不使 API 启动失败。
- [ ] 执行相关包单测、数据库查询测试及 `go build ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate`，覆盖 V10、V11、V13、V15、V17 服务侧；提交并独立审阅。

### Task 4: gRPC 业务事实与 API 生命周期接入

**文件：** 修改 `internal/server/authz.go`、`internal/server/athena-server.go`、服务器 option／配置／关闭入口、目录所列 37 个 facade 方法；新增 `internal/server/operation_log.go`、覆盖测试。

**接口：** `AthenaServer` 自有 producer 和内部查询 client；授权拦截器创建唯一 recorder，handler 通过 context 标注事实；任务 1 recorder 的实际签名是唯一接口来源。查询服务加入 ServiceSet、gRPC、gateway、read scope 及 administratorGRPCMethods，要求交互凭据。

- [ ] 先写拦截器行为测试：普通 session／API Key／development 归属、授权拒绝 authCtx、模块拒绝、日志 sink 失败仍调用一次业务、gateway／grpc-web 唯一记录。
- [ ] recorder 贯穿 authenticate／authorize／admission／handler；授权失败仅可信 FINISH；授权通过 START 在 admission 前；panic 不吞原 panic；nil producer 仍安全。
- [ ] 37 项逐方法注入白名单请求目标和已确认业务结果。dispatch 前错误与远端未知分开，异步通知／probe 标 ACCEPTED；成功响应不替代业务状态。钱包批量原子结果不假造部分完成。
- [ ] 同步 account／wallet／subscription／round／token 等资源 ID、revision、effect，私有备注仅字段名，API key 仅展示 ID；权限前值无 CAS 证据时 beforeAvailable=false。
- [ ] 注册 API 五个查询及身份绑定、no-store、前缀，当前 producer 建独立 pool、定时状态上报和关闭；日志配置失败只呈现异常。
- [ ] 运行 `go test ./internal/server/...` 及受影响包；描述符和实现覆盖测试证明 37 项与排除列表；覆盖 V01–V04、V06–V07、V13–V15。提交并独立审阅。

### Task 5: 原生 HTTP、注册登录与多步操作接入

**文件：** 修改目录内 `internal/googleoidc/`、`internal/phantomauth/`、`internal/authregistration/`、`internal/server/logout/`、头像／密钥 handler、`internal/server/worm_*.go`、`util/session/` 中 CredentialManager 的必要 hook；增加对应测试。

**接口：** 同一 request context 共享 requestId 和预算；注册父操作与登录子操作独立 operationId；使用 nil-safe recorder 显式观察提交点，轻量 ResponseWriter 仅观察状态／写回错误，不捕获正文。

- [ ] 先写提交后 publish 失败、注册成功登录失败、Google 取消／需注册、退出 revoke 失败、HTTP 写回失败和 Worm 部分移除后失败回归测试。
- [ ] 所有 61 个采集位置按 source+symbol+purpose 显式接入；12 个初始化位置只记录失败。Google callback 完成目的分派后创建对应 recorder，不重复普通登录。
- [ ] RegisterExternalAccount 持久提交后 publish 前标注 ACCOUNT_CREATED／EXISTING_ACCOUNT_RESOLVED；随后创建登录子动作。会话签发和 Cookie 成功才 SESSION_ISSUED，取消授权 CANCELLED。
- [ ] 退出仅绑定已核验 realm 会话；cookieCleared、revocationConfirmed、NO_ACTIVE_SESSION 准确。私钥 reveal 只记行为，原 Origin／reauth／版本流程不改。
- [ ] Worm CONNECTED／NOT_CONNECTED／处理中／部分效果显式映射，命令受理与目标持久达成分开；wallet selection 多步以 effect 记录 PARTIAL；不采集 heartbeat／execute-next。
- [ ] 执行 handler／manager 测试及 V01–V07、V14，覆盖清单关联到每个真实 hook 和测试；提交并独立审阅。

### Task 6: 管理员日志列表、详情与独立状态源

**文件：** 新增 `ui/src/app/admin/operation-log-service.ts`、`pages/operation-logs.tsx`、`pages/operation-log-detail.tsx`、相应测试／样式；修改管理员 registry、路由、导航及请求 feature `admin-operation-logs`。

**接口：** 只消费任务 3 的五个公共查询；UI ID／计数保持字符串，read scope generation／abort 沿用项目模式；会员 bundle 不导入管理员服务。

- [ ] 使用 impeccable 并读取已批准主题和管理员 shell 约定。先写 service／DOM 测试：草稿 Apply、Reset 默认、Refresh 保持应用筛选并换快照、真实 Previous cursor、详情返回与 URL 恢复、身份代次迟到响应丢弃。
- [ ] 桌面六列、手机记录：动作和结果优先，UTC+8 输入／时区转换。筛选完整覆盖时间、用户、模块、动作、结果、角色、来源、账户目标和主对象，无 total／页数推算。
- [ ] 列表不自动插行；三状态源独立；状态 10 秒可见 single-flight、hidden 暂停；503 保留旧数据和时间，401／403 清理，cursor 过期显式重新查询。
- [ ] 详情展示八种结果、START_ONLY／FINISH_ONLY、对象与变更、reason、counts、协议结果、默认折叠关联事实，长 ID 可复制／换行，不链接越权业务页。
- [ ] 执行 `cd ui && yarn test --runInBand` 的相关测试、TypeScript、ESLint、构建；V16 Playwright 覆盖 1440×900、390×844、200%、键盘、空态、长 ID、故障和 scope 清理；提交并独立审阅。

### Task 7: 独立运行、迁移编排与构建产物

**文件：** 修改 `internal/devruntime/{registry.go,schemas.go,config.go,environment.go,environment_graph.go,fullstack.go}` 及实际消费者、`cmd/athena-migrate/` 实际 all 路由、Makefile、生产 Dockerfile／Compose 和 `.env.example`；增加运行回归。

**接口：** `operation-log` 服务 registry，listen/server address 默认 8124、独立 internal token／HMAC key，schemas 为 account 和 operation-log；局部启动仅 PostgreSQL；完整栈包含日志，API 不因日志不可用拒绝业务。

- [ ] 先写 registry／schema owner／环境白名单／选择图测试，断言 operation-log 独立启动不选择 Wallet、Worm、Trader Sync、Redis 或 UI。
- [ ] 接入 migrate up／verify，generic all 明确路由自有 schema owner；配置使用已批准名称，内部 token 和 HMAC 持久保存／互不复用，TLS 参数透传。
- [ ] 加独立 binary／image build 和内部 Compose 服务，不执行生产部署。readiness／runtime status／stop 包含新服务，统一 INSTANCE 精确资源归属。
- [ ] 执行 devruntime／迁移／Compose 测试、独立构建及 `make run-service SERVICE=operation-log INSTANCE=key-operation-logs`；验证鉴权、健康、最小依赖、同实例停止、数据保留，写 V12／V17 证据。
- [ ] 同步运行文档实际命令与限制；提交并独立审阅。

### Task 8: 全链路验证、整分支审阅、环境收尾及人工审查

**文件：** 新增 `docs/testing/key-operation-logs-acceptance.md`、`docs/testing/human-review/key-operation-logs/R1/{ai-delivery.md,review-guide.md,human-report.md}`；更新长期需求／设计、索引、管理员 shell／本地运行文档；补全实际发现问题对应测试。

- [ ] 汇总 V01–V18，每条记录命令、退出码、具体用例、日志／截图／报告路径和事实边界；测试不覆盖项保留未完成，不将静态检查写成真实通过。
- [ ] 按 athena-browser-acceptance 准备独立真实环境，确认地址、所属进程、Node；从本 worktree 执行 `make run INSTANCE=key-operation-logs-acceptance` 并保存会话／日志，检查 member/admin bootstrap。
- [ ] V18 真实会员关键操作→持久入箱→独立消费→管理员列表及详情；停止日志进程后业务继续、采集继续，恢复后积压消化；根路径与部署前缀分别检查。外部凭据确实不可用时记录尝试、失败原因和未执行范围。
- [ ] 执行适用 Go、真实 PostgreSQL、race、前端测试／构建和完整目录核对。独立最强审阅代理做整分支审阅；在批准范围内修复并重验，不扩大需求。
- [ ] 按资源归属用同 INSTANCE 的停止入口停止本任务环境，检查进程、端口、容器，保留数据库、日志、截图和验收报告。
- [ ] 使用 athena-human-review 交付 R1，版本绑定最终代码提交与验证证据；AI 未完成／阻塞时如实标注，不提前写最终交付完成。
- [ ] 更新计划 checkbox 和长期文档当前事实；全部本地提交，最终汇报分支／工作区／实现／验证／未完成／环境。超过 600 秒按固定模板发送一次通知，等待命令结束。
