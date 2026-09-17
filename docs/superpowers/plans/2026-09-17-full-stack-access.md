# 十一应用全栈启动与六板块访问控制实施计划

> **执行要求：** 使用 `subagent-driven-development` 按任务实施，每项完成后进行规格与代码质量审阅，最终执行整体验证与独立审阅。复用本轮用户对完整技术方案及开发、测试、本地真实验收的授权，不重复确认既定设计。步骤以复选框追踪。

**目标：** `make run` 默认启动十一应用，由 API Server 的六个持久访问开关拦截新的用户业务请求，管理员通过 Module Access 管理，后台任务继续运行。

**架构：** 复用现有独立服务、账户 PostgreSQL、API 认证链及前端身份边界。账户库保存设置，API 每请求读取；前端增加独立访问代次。运行器统一声明所选应用、schema、配置、就绪与停止依赖，拥有并准确回收本地资源。

**技术栈：** Go 1.27.1、PostgreSQL/pgx/sqlc、gRPC/Protobuf、Cobra、Docker、本地 Go runtime、React/TypeScript、Node 24.14.1、Jest、Playwright/系统 Chrome。

**批准规格：** [全栈启动与访问控制修订设计](../specs/2026-09-17-full-stack-access-reassessment-design.md)。用户于本轮整体批准，规格中的“待审阅／未授权实施”是此前时点事实。

## 全局约束与基线

- 工作区 `/home/yege/work/athena`，分支 `rf4`，实施前 HEAD `d230ce03`，初始工作区干净。按用户指定目录工作；不覆盖其他任务改动，不推送或合并远端。
- 五核心：UI、API Server、Wallet、Notification、Etherscan Manager。六业务：Trader Sync、Solana Discovery、Market Radar、Managed OO、Profit Sharing、Worm Trading。
- 开关键限定 `trader_sync`、`solana`、`market_radar`、`managed_oo`、`profit_sharing`、`worm`；`worm` 只指 Trading。缺行默认关闭，显式目标值持久 upsert，同值提交也更新修改信息，重启不重置。
- 开关不改变账户权限 revision、既有交易授权、订阅撤权事务、后台采集、已受理工作或通知；API Key、管理员及开发身份的业务请求同样受控。
- 外部业务请求统一经过 API，内部端口由服务器规则隔离是已确认部署前提，本轮不检查远端服务器或扫描外网端口。
- 不新增 Market Radar、Managed OO、Profit Sharing 的内部鉴权、Actor 协议、账户池或服务侧权限重构；保留既有鉴权。
- Token 接入延期；不恢复 BSC、Sports、Worm Markets。五远端 Gateway 只连接及只读健康检查，不创建、停止或修改实例。
- 全栈保持 managed；局部仅准备所选依赖，external 仅 verify 借用库。普通启动/停止不删除数据库或数据卷，不新增历史兼容或数据搬迁。
- 复用现有 Trading main/migrate、目录内聚及 Solana/Trading/Trader Sync 认证；当前六应用图、Wallet/Profit Sharing 聚合入口及 Token/Temporal 多余准备仍需改造。
- 预算：launcher 5 分钟、构建每项 10 分钟、基础设施 5 分钟、schema 每步 120 秒、普通 ready 60 秒、UI 120 秒、Notification 180 秒、总启动 30 分钟、服务停止默认 30 秒、总收尾 180 秒。
- API 配置读取最多 2 秒且服从请求期限；前端前台每 2 秒单飞读取，从请求开始按单调时钟计算最多 5 秒有效。
- 独立任务之间接口有依赖时按顺序落地；生成物从源生成，不手改。sqlc 在稳定输入批次后运行一次。
- SDS-R1–R8 按实际涉及边界记录证据；不把健康检查等同于业务验收，不把 AI 交付写成人工最终通过。

## Task 1：独立入口、schema 准备与有界停止

**文件：**
- 新增 `cmd/athena-wallet/main.go`、`cmd/athena-etherscan-manager/main.go`、`cmd/athena-market-radar/main.go`、`cmd/athena-managed-oo/main.go`、`cmd/athena-profit-sharing/main.go`。
- 修改上述 `commands/`、`cmd/athena-solana-discovery/commands/`、`internal/wallet/store/`、`internal/managedoo/store/`、`internal/profitsharing/store/`、`internal/solanadiscovery/` 的连接、验证及停止逻辑；必要共用代码放 `internal/serviceschema/` 或现有 postgres 工具目录。
- 测试放对应 command/store 包；更新直接运行说明中的实际接口。

**接口：** Wallet、Managed OO、Profit Sharing、Solana 二进制提供 `schema up|verify --timeout=120s`；schema 命令只需要该 schema 的 DSN，不启动业务。服务运行只连接并 verify。Solana 用同一 pool 验证账户 public contract 与业务 schema。

- [x] 写失败用例：空库 verify 不创建版本表；当前 schema 成功；缺表/缺列/错误版本失败；schema 子命令在无业务凭据时可执行；停止遇到阻塞工作仍有界。

```go
// 测试使用临时 PostgreSQL 并比较验证前后的 catalog。
before := readCatalog(t, db)
err := verify(ctx, db)
require.Error(t, err)
require.Equal(t, before, readCatalog(t, db))
```

- [x] 运行对应测试确认缺少行为导致失败，再实现显式迁移、只读 verify、五独立 main 和取消链；复用已有 migration，不重写历史迁移。
- [x] 运行命令包和存储测试、构建五 main 与 Solana；通过实际子进程验证 TERM 的有界退出。验证错误不启动 scanner/RPC。

```bash
go test ./cmd/athena-wallet/... ./cmd/athena-etherscan-manager/... ./cmd/athena-market-radar/... ./cmd/athena-managed-oo/... ./cmd/athena-profit-sharing/... ./cmd/athena-solana-discovery/...
go build ./cmd/athena-wallet ./cmd/athena-etherscan-manager ./cmd/athena-market-radar ./cmd/athena-managed-oo ./cmd/athena-profit-sharing ./cmd/athena-solana-discovery
```

- [x] 自查范围、保存红绿测试证据，提交 Task 1；独立审阅并修复后进入 Task 2。

## Task 2：持久设置、三个接口及全部 API 准入

**文件：**
- 新增 `internal/accountstate/store/migrations/000006_module_access.sql`、`queries/module_access.sql`、账户 store 设置适配器、`internal/moduleaccess/` 的键/状态/错误/准入能力。
- 新增 `internal/server/moduleaccess/moduleaccess.proto` 与 handler；生成 `pkg/apiclient/moduleaccess/`、账户 sqlc、`internal/accountstate/schema/contract.json`。
- 修改 `internal/server/authz.go`、`athena-server.go`、原始 `worm_*.go`、`internal/walletsecret/errors.go` 及实际 Google/Phantom/开发验证入口；测试各实际注册集合。

**接口：**

```proto
rpc ListModuleAccessStates(ListModuleAccessStatesRequest) returns (ListModuleAccessStatesResponse);
rpc ListModuleAccessSettings(ListModuleAccessSettingsRequest) returns (ListModuleAccessSettingsResponse);
rpc UpdateModuleAccessSetting(UpdateModuleAccessSettingRequest) returns (UpdateModuleAccessSettingResponse);
```

HTTP 路径分别为 `/api/v1/module-access-states`、`/api/v1/admin/module-access-settings`、`/api/v1/admin/module-access-settings/{module_key}`。列表固定六项、无分页；最小状态只包含键和状态；管理接口另含最后修改者账户 ID、用户名和时间。更新只接受键及 `OPEN/CLOSED`，操作者/时间来自可信会话/数据库。

- [x] 先写持久层失败测试：缺行关闭、同值更新、重启保留、并发最后提交生效、未知键/Token/UNSPECIFIED 拒绝，不修改账户 access revision。
- [x] 稳定 SQL 输入，运行 `make sqlc-local`，再写 store 消费者；运行 `make account-state-schema-contract` 更新 contract，验证新库/既有库/缺结构与只读空库。
- [x] 添加 proto 并运行 `make protogen-fast`，按生成协议实现 handler、注册及授权。更新必须显式检查交互凭据，覆盖管理员早返回与 API Key 拒绝；local-admin 允许。
- [x] 先写真实 interceptor/handler 失败测试，覆盖：39 个现行 RPC 与实际描述符分类；原始 Trading HTTP 28 项、验证 16 项与四个 Google purpose；core/deferred 显式分类，未知业务失败关闭。

```go
// 用户原权限先成功，再检查开关；已放行请求不因之后关闭而被取消。
require.Equal(t, codes.Unavailable, status.Code(callClosedBusiness()))
require.Equal(t, "MODULE_ACCESS_CLOSED", errorReason(callClosedBusiness()))
require.Equal(t, codes.OK, status.Code(callCore()))
```

- [x] 在身份/权限成功后、handler/授权签发/副作用前统一准入；unary/stream、gRPC-Web、JSON、原始 HTTP 都走同一能力。每请求读主库，失败返回 `MODULE_ACCESS_UNAVAILABLE`，两类错误均保留 `athena.module_access` ErrorInfo 与 module_key；HTTP 保留现有 envelope、reason header、private/no-store。
- [x] 验证 Google state 先校验消费，关闭不恢复 state；重定向传专用原因。后台与内部服务不注入此开关。
- [x] 跑账户、API、原始验证及生成消费者相关测试，保存证据，提交并独立审阅。

## Task 3：双应用访问状态边界与 Module Access 页面

**文件：**
- 新增 `ui/src/app/shared/module-access.tsx`、`module-access-service.ts` 及测试；修改共享 `services/requests.ts` 错误解析。
- 修改 `ui/src/app/member/app.tsx`、`admin/app.tsx`、对应路由、`admin/pages/service-status.tsx` 与测试。
- 修改 `member/pages/worm-trading-executions.tsx` 及必要签名/交易消费者，保持现有敏感写入与身份代次。

**接口：** 每个已认证 realm 自有 provider；业务边界消费 `module_key` 及独立访问 epoch。原权限先判定；核心页面不等待访问读取。provider 单飞刷新、单调时钟 freshness、模块作用域 abort/invalidations。

- [x] 先写受控时钟/延迟请求失败测试：2 秒轮询；请求发起后 5 秒过期；隐藏/离线/focus/pageshow 先失效；旧开放响应不能恢复；realm 不串状态；关闭仍轮询。

```tsx
// 行为断言针对真实 provider 和业务组件。
expect(screen.queryByText('business body')).toBeNull();
await resolveLatestStates({worm: 'OPEN'});
expect(screen.getByText('business body')).toBeVisible();
advanceMonotonicTime(5001);
expect(screen.queryByText('business body')).toBeNull();
```

- [x] 实现访问边界，关闭/未知时卸载业务正文和草稿、取消相应请求、拒绝迟到读取及签名后续写入。重新开放读取权威状态，Trading 不自动恢复 heartbeat/execute-next/排队 timer，不发送 pause/terminate。
- [x] Service Status 增加第四页签，六行设置与 Token 延期说明；桌面表格、手机分隔行；行级保存，明确目标值，无新增确认弹窗；保存未知后重新读取实际设置，禁止自动重发。
- [x] 沿用 Nansen 深色、既有组件与可访问性，健康和访问查询独立；保留有权限菜单、URL 和核心导航。
- [x] 运行相关 Jest、`yarn lint`、`yarn build`，保存结果并提交；独立审阅竞态及视觉消费者。

## Task 4：统一十一应用注册、配置与按需 schema

**文件：** `internal/devruntime/{registry,fullstack,environment,launch,postgres,worm_trading}.go` 及测试，必要配置代码拆为职责明确文件；`cmd/athena-local-runtime/`、`Makefile`、`hack/` 的实际消费者。

**接口：** 单份 `ServiceSpec` 定义独立 BuildPackage、最小基础设施、白名单、schema owner、端点、ready、核心类别和停止层级；`FullStackServices()` 选十一项；局部 `ResolveServices` 只选显式应用。

- [x] 先写正向选择测试：五新增命令可独立构建；Wallet/Managed OO/Profit Sharing 单选不要求账户库；Solana 才准备账户和 Solana schema；external 无 DDL/seed/借用资源停止。
- [x] 新实例仅建 `athena/wallet/managed_oo/profit_sharing/worm_trading`；移除 Token/Temporal 普通准备，历史库保留。账户 up→verify→Solana up→verify→账户 verify，其他四库各自 up/verify；全部准备结束才启动匹配 contract 的消费者。
- [x] 显式 loopback 监听、解析唯一正式端口键，Profit Sharing 使用 `ATHENA_PROFIT_SHARING_LISTEN_PORT`；覆盖八个 API 下游、两个通知生产者、Trading Wallet、UI API/BaseHRef。本地全栈覆盖残留远端消费者，局部只覆盖已选服务。
- [x] 持久生成/复用内部凭据，两端一致；冲突报错，Trading 加密 key 重启不轮换，API 不接收 Wallet signer 凭据。
- [x] Manager host:port 与 API Gateway IP 配置规范化比较，限定纯 IP:6776、冲突拒绝；五远端地址并行只读 health，每项 3 秒。
- [x] 运行 registry/environment/schema 集成测试并构建运行器，提交并独立审阅。

## Task 5：分阶段就绪、失败隔离、持久事件及精准停止

**文件：** `internal/devruntime/{launch,runner,state,process_linux,helper,infrastructure}.go`、相关生命周期测试、`cmd/athena-local-runtime/` 状态展示与 launcher/stop 脚本。

**接口：** 状态分离阶段、服务启动结果、当前探测、退出事件和收尾结果；成功执行 status 与 full-stack ready 分别表达。保留既有 PID/PGID/ticks/boot/run/executable/container identity 检查。

- [x] 先写实际进程/故障测试：核心失败整体收尾；单业务初始失败清理自身、保留核心并继续其他业务；运行期退出码 0 也撤销 ready；停止后历史失败仍存在。
- [x] 核心 Wallet/Notification/Manager→API/UI；HTTP health、双 realm bootstrap 的 realm/会话及 UI proxy/资源验证后输出核心可用；逐业务 ready 后才十一项 ready。Trader Sync 使用专属 health service，Notification 等待 sender 恢复屏障。
- [x] 为 build/infra/schema/start 操作施加真实 deadline，总启动 30 分钟；取消/构建或共享准备失败收尾并非零。业务失败前台留存时 status 清楚报告未完成。
- [x] 停止按 UI/API→六业务→Wallet/Notification/Manager→自有基础设施；总预算 180 秒，独立服务仍需自身有界停止。只停止本次拥有资源，保留数据/日志/失败。
- [x] stop/status 先用校验后的已保存运行器，不依赖当前源码可编译；Solana 移除第二条全栈 owner 路径，旧现场只由原 owner 收尾。
- [x] 运行生命周期、取消、身份、停止预算与异常退出回归；提交并独立审阅。

## Task 6：整体验证、真实十一应用及页面闭环

**文件：** 新增/扩展 `ui/e2e/` 的实际验收用例、`docs/testing/full-stack-access-acceptance.md`；日志与截图存本任务独立证据目录。

- [x] 核对每项改动和生成消费者，跑受影响 Go 测试、UI Jest/lint/build；独立整分支审阅，对重要问题修复并复验。
- [x] 记录已有容器/进程/端口归属，选择项目 Node，以本任务独立 INSTANCE 从根目录执行真实 `make run`，保留持久会话和日志。

```bash
make run INSTANCE=full-stack-access
make runtime-status INSTANCE=full-stack-access
```

- [x] 用真实应用就绪/恢复与十一项状态验证启动；member/admin bootstrap 分别核对 realm；使用系统 Chrome smoke，检查退出码与报告。
- [x] 六开关首次关闭→管理员开放→合法业务可达→关闭返回专用错误→核心仍可用；分别保存开/关并重启验证持久性，核对五库和历史数据保留。
- [x] 观察关闭前后真实后台扫描/游标/已受理任务或通知的连续性，记录真实证据；受控故障和签名竞态测试补充不可安全触发路径，不以替身冒充真实后台证据。
- [x] 桌面/手机、根路径/`/athena`、双 realm、直接详情、离线恢复、旧响应、保存未知及重新开放不自动驱动交易；必要时使用隔离浏览器用例补齐矩阵并区分真实与受控来源。
- [x] 按需求逐项写证据和限制，网络隔离只记已确认部署前提，不声称远端实测。

## Task 7：长期文档、环境收尾及人工审查交付

**文件：** `docs/requirements/development-runtime/{business-access-control,service-inventory-review}.md`、`docs/design/development-runtime/local-runtime-orchestration.md`、访问控制长期设计、`docs/developer-guide/running-locally.md`、双应用壳文档、需求/设计索引及本轮验收记录。

- [x] 将文档现状更新为实测结果，保留既有批准时点，不将三服务鉴权排除误写为已修复；写清局部 schema 命令、Token 延期、Gateway 与后台继续的边界。
- [x] 按同一 checkout/INSTANCE 停止本任务应用和容器，核验进程退出、端口释放、容器停止；数据库/卷、日志、截图和报告保留，共享基础设施不动。

```bash
make stop INSTANCE=full-stack-access
make runtime-status INSTANCE=full-stack-access
git diff --check
```

- [x] 使用 `athena-human-review` 交付 `docs/testing/human-review/full-stack-access/R1/{ai-delivery,review-guide,human-report}.md`，绑定准确代码版本及证据；有未完成项如实标明。
- [x] 最终核对 git 状态与本轮提交、验证和资源清单；累计超过 600 秒时从根目录发送固定通用通知一次并等待结果。
- [x] 最终说明实现、验证、限制与收尾，使用“本轮 AI 交付完成，待人工审查”或实际未完成状态；不记录人工最终验收通过。

## 计划自检

| 关联 | 产出与消费 | 核对 |
| --- | --- | --- |
| Task 1→4 | 独立 main、schema up/verify、运行只读验证 | 命令名、预算和 DSN 与批准规格一致 |
| Task 2→3 | 三 RPC/HTTP、ErrorInfo、六键/状态 | UI 依据生成协议消费，授权和访问代次分离 |
| Task 2→4 | 账户迁移与 contract | 所有同库消费者升级后统一准备，不同时运行不匹配版本 |
| Task 4→5 | ServiceSpec/端点/依赖/schema | 正向选择、就绪与停止使用同一声明 |
| Task 1–5→6 | 独立测试及实现 | 真实十一应用、双 realm、Chrome 与后台证据另外执行 |
| Task 6→7 | 准确版本、实测报告与资源归属 | 收尾后交付人工材料，未实测内容保留限制 |

各任务测试和实现目标一致；没有未决设计分岔。测试中的 `verify/readCatalog/callClosedBusiness` 表示对应测试夹具职责，实现时使用实际包的接口，测试必须运行真实被测逻辑。常规技术细节按批准规格与现有代码确定，实质冲突才向用户提出。
