# Solana 元数据补全实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 已有与新增 Solana 候选可显示链上名称、符号和有证据的发行来源。

**Architecture:** Solana 服务负责解析、存储和异步补全，API 继续代理服务，UI 只展示持久化结果。新增后台补全与扫描共用 RPC 限速；失败和缺失通过持久化时间重试，不阻塞发现游标。

**Tech Stack:** Go、pgx/PostgreSQL、Solana JSON-RPC、protobuf/gRPC Gateway、React/Ant Design。

**Spec:** `docs/superpowers/specs/2026-09-13-solana-metadata-design.md`

## Global Constraints

- 仅在 `/home/yege/work/athena/.worktrees/solana-discovery` 开发，保留既有候选和游标，不重置数据、不动根目录环境。
- 名称与符号读取 finalized 链上账户快照；不抓取 URI、Logo 或其他链外地址。
- 发行来源通过成功的 Mint 初始化交易判断：已知祖先发行指令同时匹配程序地址、selector 和该 Mint 的指定账户。
- 扫描和补全共用配置的节点请求预算与超时，429 尊重 Retry-After。
- 每批最多 10 个候选、20 个账户；同批交易签名去重。缺失元数据 1 小时后重试，失败 5 分钟后重试。
- 名称和符号分别允许为空。已验证的字段不自动周期更新；补全失败不得覆盖已有有效名称和来源。
- 保留既有 Solana READ 授权，API 和 UI 不直接访问节点，不扩展为研究生命周期。

### Task 1: 后端解析与持久化补全

**Files:** 新增 `internal/solanadiscovery/metadata.go`、`source.go`、`enricher.go`、`enrichment_store.go` 及各自测试；新增 `migrations/002_metadata.sql`；修改 `model.go`、`parser.go`、`rpc.go`、`scanner.go`、`store.go`、相关测试和 `cmd/athena-solana-discovery/commands/command.go`。不得修改 rpcservice、proto、UI 或长期文档，由 Task 2 负责。

**Interfaces:** 现有 `NewScanner` / `Run` / `ParseBlock` 签名继续供调用；Project 增加以下字段，作为 Task 2 唯一数据合同：

```go
Name, Symbol, MetadataStatus, MetadataSource, MetadataAccount string
MetadataObservedSlot uint64
MetadataUpdatedAt time.Time
IssuanceSource, IssuanceProgram, SourceStatus string
```

metadataStatus 为 pending/ready/unavailable/error；sourceStatus 为 pending/identified/unrecognized/error；metadataSource 为 token2022_on_mint/metaplex/空；issuanceSource 为 pump_fun/raydium_launchlab/direct_token/unknown。首次新发现直接填来源状态；历史行迁移后为 pending。MetadataUpdatedAt 是本次有效观察的时刻，失败保留之前值。

- [x] 先编写失败测试：Token2022 type18自指+type19名称符号、真实 USDC Metaplex padding、错误 owner/mint/畸形长度/UTF8/外指 pointer；Pump create_v2 成功和未知三级 CPI、兄弟调用不误归因、LaunchLab 的 base_mint 与 quote_mint 不混淆。以 `.superpowers/solana-metadata/research-metadata.md`、`research-source.md` 为详细协议证据；精确 selector/账户下标已经给出，不自行猜测。真实 fixture 复制必要数据到 `internal/solanadiscovery/testdata/`，去掉无关大段交易数据也必须保留测试用到的调用树。
- [x] 运行 `go test ./internal/solanadiscovery -run 'Metadata|Issuance|Source' -count=1` 确认测试先因未实现失败，保存证据。
- [x] 实现有边界的二进制解码与来源识别；重构解析器局部类型使原区块解析与 getTransaction 历史补全复用同一来源逻辑。来源无法判定不是 scanner fatal error。rpc 使用 finalized/base64 getMultipleAccounts，逐地址对应、校验数量、null 和 context.slot；getTransaction 使用 finalized/jsonParsed/maxSupportedTransactionVersion=0。
- [x] 新建幂等 SQL 迁移，Migrate 顺序执行内嵌迁移。添加上述公开字段与内部下次补全时间/必要状态。新增待补全读取与独立结果写回：队列按到期时间、发现时间稳定排序，事务不持有远端请求；metadata 与 source 独立推进，已完成的一边不重复查询；同批签名去重。成功元数据 ready，当前缺失 unavailable，临时失败 error；worker 所有错误均不更新扫描状态。允许固定后台批间间隔 10 秒；候选为空也等待以免忙循环。
- [x] 扫描和后台共享同一节点限速器（总预算仍是配置值），取消时两者均退出。避免未知来源永久重复 getTransaction。记录必要错误到服务日志，不把密钥 URL 写入候选字段。延迟重试用数据库 next-at，重启不丢状态。首次获取 metadata 失败不阻止原候选入库。
- [x] 更新 ListProjects 返回字段并扩展查询：Mint 保持大小写敏感；名称和符号使用 `position(lower($1) in lower(name)) > 0`，不得把 `%`/`_` 当通配符。
- [x] 先写 PG 行为失败测试覆盖迁移保留游标、重启待处理、逐字段失败保留、查询；worker/RPC mock 覆盖共享预算、null/错误重试及某候选出错其他候选正常。测试数据库由主代理提供专用端口，设置 `SOLANA_TEST_POSTGRES_DSN`，不得连接预览 DB 跑破坏性测试。
- [x] 运行核心包与 command 的相关测试；报告真实通过结果、skip 数及未完成项。仅提交本任务所属文件，提交说明 `feat: enrich Solana candidates from on-chain evidence`。报告写入 task brief 对应 report 路径，附具体 RED/GREEN 证据与接口说明。

### Task 2: API、列表与长期文档

**Files:** `internal/server/solana/solana.proto`、`internal/solanadiscovery/rpcservice/service.go` 及测试、`pkg/apiclient/solana/` 生成文件和 `assets/swagger.json`、`ui/src/app/shared/services/solana-service.ts`、`ui/src/app/member/pages/solana.tsx` 及测试、`docs/requirements/solana/README.md`、`docs/design/solana-intelligence/project-discovery.md`、`docs/design/web-ui/solana-discovery.md`。

**Interfaces:** 消费 Task 1 Project；protobuf 新增 string name=11、symbol=12、metadataStatus=13、metadataSource=14、metadataAccount=15，uint64 metadataObservedSlot=16，int64 metadataUpdatedAt=17，string issuanceSource=18、issuanceProgram=19、sourceStatus=20。时间空值为0，非空 Unix秒；UI 使用相同 lowerCamelCase 名称。

- [x] 增加 UI 行为失败测试：带名称/符号/Pump.fun 的首列与来源；pending/unavailable/error 不同文案；未知源不会冒充发行平台；查询名称会回第一页且提交字符串；长文字/详情包含完整可核验地址。运行现有页面 Jest 确认失败。
- [x] 按字段合同更新 proto 并从隔离 GOPATH 运行 `make protogen`，审查并还原本任务没有改变源文件的无关生成漂移；rpcservice 映射所有字段并覆盖字段/时间与既有授权。
- [x] UI 首列显示 name、symbol 与 Mint；来源独立列，程序/观察证据放详情。保留复制、交易和 Solscan 链接、手动刷新与当前分页；搜索文案 Name, symbol or Mint，字段缺失不伪造。UI 时间与枚举解析保持现有风格。
- [x] 同步长期需求与设计：原“名称/符号不在第一步”改为本轮已纳入，说明支持与未识别边界、自动补全及非研究调度、API 字段和验收证据。
- [x] 运行 scoped Jest、Go rpcservice/server 测试、UI lint，检查 git diff；提交本任务文件，提交说明 `feat: show Solana token identity and issuance source`。

### Task 3: 真实预览验收和交付

**Files:** 新增 `docs/developer-guide/acceptance-records/2026-09-13-solana-metadata.md`，更新本计划的事实状态。

- [x] 使用本任务独立 PostgreSQL 测试实例完成新测试及原 scanner/store/command 回归；检查没有隐藏 skip。生成 diff 评审包，任务评审包含 spec compliance 与 quality 两个结论，有问题先修复再定向复验。
- [x] 确认 PID/worktree 后仅重启 solana-preview：`make stop ATHENA_RUN_PROFILE=solana-preview`，使用 Node24 PATH 执行 `make run ATHENA_RUN_PROFILE=solana-preview` 持续运行并保存新日志；不改根目录进程，不删除数据卷。
- [x] 比较重启前后的 StartSlot、LastProcessedSlot、候选数，确认已有记录补全、后续新记录仍在入库；通过 API 搜索一个真实 Pump 候选验证名称、符号、平台与 metadata 观察证据；调用失败状态不能当作全部任务完成。
- [x] 在 1440px 桌面与 390px 手机浏览 `/solana`，验证字段、详情、搜索、复制可访问性及表格自身滚动。运行 `make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://127.0.0.1:14000`，检查真实退出结果与报告。
- [x] 完成最终代码评审和验收记录，保存本地提交并保留运行环境。
- [x] 全部通过后从 worktree 根执行一次 `make notify-task-complete TASK_NOTIFICATION_SUBJECT='任务完成：Solana 名称与发行来源' TASK_NOTIFICATION_BODY='已完成：Solana 候选名称、符号、发行来源与历史补全；验证：后端、前端及真实预览验收通过。'`，等待退出。最终回复用户可访问页面、可查看内容和实际限制。

## 执行事实

实现、评审修复和真实预览验收已完成；历史补全队列持续运行。最终验证包含100项后端PG/race、7项UI、API、lint及系统Chrome实际业务检查与两项smoke。完整证据与当前运行方式见[验收记录](../../developer-guide/acceptance-records/2026-09-13-solana-metadata.md)。本地分支和预览保留，未合并或发布PR。
