# Solana 新项目发现实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** 实现从 Solana 节点发现新 Mint、持久保存并在 ATHENA 独立列表页查看的完整第一步。

**Architecture:** 独立发现服务持有扫描、存储和查询；API 经认证 gRPC 代理；UI 使用独立 Solana READ 模块。采用 finalized HTTP 区块扫描及 PostgreSQL 检查点，不实现项目研究或活动刷新。

**Tech Stack:** Go 1.27.1、已有 solana-go、pgx/PostgreSQL、gRPC/Gateway、React/Ant Design、Node 24.14.1。

**Spec:** [第一步设计](../specs/2026-09-13-solana-discovery-design.md)。

## 全局约束

- 只实现发现与列表；保留后续持续研究需求，不引入研究任务或交易。
- Solana Mainnet Beta，finalized，原始 Token Program 与 Token-2022，顶层及 CPI 初始化，成功交易。
- 独立进程和 schema；API 不持有运行时；独立内部认证、可信账户身份、业务服务自身 Solana READ 检查。
- 原始候选不作 NFT/LP/投资项目分类；不猜名称、当前权限或发行人。
- 先写失败行为测试再实现；修改源后生成，禁止手改生成文件。
- 独立 worktree 开发，保留其他任务工作；真实验收不得以单测替代。

## 接口约定

公共与内部共用 internal/server/solana/solana.proto，go_package 为 pkg/apiclient/solana。服务名 SolanaService，方法 ListProjects、GetDiscoveryStatus。ListProjectsRequest 为 page、pageSize、query；响应 items、totalSize。Project 为 mint/tokenProgram/signature/feePayer/mintAuthority/freezeAuthority（string）、decimals（uint32）、slot（uint64）、blockTime/discoveredAt（int64）。GetDiscoveryStatusResponse 为 status、startSlot、lastProcessedSlot、latestFinalizedSlot、lastSuccessAt、totalProjects、lastError。HTTP 与 JSON 使用上述 camelCase 字段。

内部服务接收唯一 x-athena-account-id metadata，API 从认证凭据生成，绝不转发浏览器同名输入；authorization 为独立服务 Bearer token。查询参数默认 page=1、pageSize=25，上限100；query最大128字符，按 Mint 文本过滤。

### 任务 1：解析、持久化与发现循环

文件：新增 internal/solanadiscovery/model.go、parser.go、rpc.go、scanner.go、store.go、migrations/ 及对应测试；业务状态只归此包。

- [x] 编写解析失败测试，构造真实形状 jsonParsed 交易，手写期望 mint/authority，不用解析器生成期望。
  ```go
  got, err := ParseBlock(42, []byte(fixture))
  require.NoError(t, err)
  require.Equal(t, expectedMint, got[0].Mint)
  ```
- [x] 运行 go test ./internal/solanadiscovery 并记录预期失败，再实现解析器；核对程序身份、成功状态、顶层/CPI、完整字段与去重。
- [x] 用 httptest RPC 替身与真实 store 行为测试固定起点、错误重试不越过、事务进度及取消，运行失败后实现最小扫描循环。
- [x] 新建独立迁移和 pgx store，schema/进度/候选在同库短事务保存。若使用 sqlc，完成源批次后由主代理统一生成。
- [x] 运行单测、真实 PostgreSQL 集成测试及 go vet；记录命令、结果、文件和接口到 .superpowers/solana-discovery/task-1-report.md。

### 任务 2：独立权限、生成契约和查询代理

文件：internal/accountaccess/、internal/accountstate/store/、internal/server/account/、新增 internal/server/solana/、pkg/apiclient/solana/（生成）、internal/solanadiscovery/rpcservice/ 与 apiclient/，以及服务器客户端配置/注册。

- [x] 编写 Solana READ/NONE、完整11模块矩阵与权限更新保留其他模块的失败测试，再增加 ModuleSolana；迁移和查询一起更新，account proto 新值13。
- [x] 按接口约定编写 proto；与主代理协调，在所有本批源稳定后运行 make protogen 与 make sqlc-local，不手改输出。
- [x] 对 gRPC 服务先写无token/错token/无身份/无权限拒绝和READ可查的测试，再实现权威授权和列表/状态查询。具体依赖使用任务1产出的 store 适配。
- [x] API代理从 util/session 已认证凭据获取 AccountID，添加唯一内部 metadata，10秒deadline。注册GW与authz，服务不可用返回可解释错误，API不启动扫描。
- [x] 验证 accountaccess/accountstate、服务和代理测试；更新相关权限设计中的模块数量及Solana授权，并记录 task-2-report.md。

### 任务 3：成员端 Solana 列表

文件：ui/src/app/shared/access-modules.ts、shared/services/solana-service.ts、member/services.ts、member/routes.tsx、member/app.tsx、member/pages/solana.tsx 与测试；相关共享完整权限fixture。

- [x] 读取当前 Market Radar 页、AppPage/ResourceTable/useAsyncData 与 Impeccable craft-floor，继承当前 Operate 布局。
- [x] 编写页面行为测试：有权加载、无权隐藏、空态/错误、刷新、分页、复制与外链；先验证失败再实现。
- [x] 新增 Solana 独立只读模块，完整权限矩阵及管理员选择器同步；路由 /solana，按计划接口请求 /solana/projects 和 /solana/status。
- [x] 展示候选说明、状态和表格；时间/int64输入须安全转换，地址区分大小写，链接只使用固定Solscan域和编码后的地址。
- [x] 运行对应 Jest 与 yarn lint；记录 task-3-report.md。浏览器验收由主代理结合真实运行环境完成。

### 任务 4：独立启动、集成与真实验收

文件：cmd/athena-solana-discovery/、Makefile、Procfile、Procfile.solana-discovery、Procfile.solana-preview、hack/solana-local.sh、配置样例、运行和验收文档。

- [x] 按任务1/2接口创建独立命令，读取自身配置；提供正向build/run/stop，停止仅回收自己的进程，全栈显式加入Solana。
- [x] 启动行为测试覆盖无效配置、有界SIGTERM及最小依赖选择；修复后验证ShellCheck与构建。
- [x] 统一生成剩余源变更，运行受影响Go及UI检查；审阅任务diff和整体集成，修复发现。
- [x] 在归属明确的本地环境运行新服务，核对主网genesis，取得真实Mint样本并验证数据库、API及页面可见结果。
- [x] 准备目标make run环境，检查member/admin bootstrap，再执行smoke；浏览器检查Solana桌面/手机、明暗主题、刷新和外链。
- [x] 同步长期需求/设计/验收记录，标清实际实现与后续范围。全部实现和验收通过后，记录运行地址与停止方法。完成通知在最终文档提交后执行一次，结果单独记录。

## 执行记录

每项实现证据与审阅记录存放 .superpowers/solana-discovery/。仅将最终成果、实际验证和必要决定写入长期文档。


2026-09-13：四项任务及任务/整体审查完成，真实主网数据、浏览器和重启恢复通过；详细命令、样本、限制和保留环境见[验收记录](../../testing/solana-discovery.md)。用户保留的研究规则未实现。工作树和分支保留供查看结果。
