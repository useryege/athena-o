# 全栈启动与板块访问控制：重构后审核及修订设计

> 实施结果（2026-09-17）：本方案已在代码版本 `6b2db2ec8ca5d488a672e68785eea3272af4f50c` 实现，并完成根路径与 `/athena` 前缀的十一应用、双 realm、六开关、两次重启持久化、后台连续性及资源收尾验证。准确证据、限制和 SDS 对照见[全栈启动与访问控制验收记录](../../testing/full-stack-access-acceptance.md)。本文下方的“待审阅／未实施”和源码行号保留设计审核基线事实，不再代表当前实现状态；人工最终审查尚未由本文宣告完成。

> 状态（设计时点）：2026-09-17 技术方案整理完成，待整体审阅；当时尚未实施。用户已明确本轮聚焦当前目标，不纳入三服务内部鉴权重构。原已确认业务决定及部署网络边界继续有效，本文补齐当时源码对应的配置、数据库准备、接口、生命周期和验证安排。
>
> 审核基线：`rf4`，`6aaf7f3c9f3cd4f038d7cb0f6ddeaefde00d3adc`。只读核对源码、现行需求、旧方案及入口；本轮未启动服务、连接远端 Gateway 或执行业务请求。此前同会话六应用启动与双 realm smoke 通过，仅证明当前启动图，不能证明本文的十一应用或访问开关。
>
> 本文替代旧 spec 中与当前源码不符的实施安排，保留原批准时点事实，不把后续完成的重构再列为未完成。整体审阅后以一份实施计划承接服务适配、访问准入和运行编排；本轮授权是完成技术设计，尚未执行实现或部署。

## 1. 继承的业务决定

依据[访问需求](../../requirements/development-runtime/business-access-control.md)、[原访问设计](2026-09-15-business-access-control-design.md)和[原全栈设计](2026-09-15-local-full-stack-design.md)：

- `make run` 启动本期全部本地应用，管理员控制新的用户业务请求是否允许进入；关闭访问不停止进程、采集、已受理工作或通知。
- 六个板块为 Trader Sync、Solana、Market Radar、Managed OO、Profit Sharing、Worm Trading；`worm` 只对应 Trading。首次未配置默认关闭，管理员设置持久保存，再次启动不改写设置。
- 会员、管理员及原本允许的 API Key 业务调用均受相应开关约束；登录、账户、Wallet、Notification、健康及开关管理保持原权限边界。
- Token 详细接入继续延期；现有 Token 接口不等于本轮应当启用。BSC、Sports 和 Markets 已退役，不恢复其程序、权限或数据准备。
- 五个远端 Etherscan Gateway 仅作为外部依赖连接，不创建或停止远端副本。关闭业务访问不减少后台外部调用，不能用开关代替停机或费用控制。
- 现有交易授权、所有权、撤权事务和恢复规则继续生效；访问开关不改成账户撤权或交易取消协议。

### 1.1 后续讨论确认的部署边界

**用户确认，2026-09-17**：服务器通过访问规则拒绝外部访问业务服务端口，外部用户业务请求只经过统一 API 入口。长期约束已写入[服务开发规范](../../developer-guide/service-development-standards.md#deployment-network-boundary)及根 AGENTS.md，后续设计都须沿用；本轮未检查或修改服务器规则。

在此边界下，API Server 的访问开关可以控制新的外部用户业务请求。本文源码审核发现的内部鉴权差距继续保留，但不能把“外部用户绕过 API 直连业务端口”当作当前部署事实，也不能仅据此将三服务鉴权改造定为访问开关的必需前置。

**本轮范围决定（2026-09-17）**：用户明确“先不要过重设计，这里聚焦当前目标”，采用十一应用启动、API 访问开关和管理员页面，以及实现这些能力必需的构建、数据库结构准备、配置、就绪和停止改动。Market Radar、Managed OO、Profit Sharing 的新增内部鉴权、Actor 协议及服务侧权限重构移出本轮，不保留为待确认项，不另行自动启动该任务；既有身份、权限及内部鉴权继续保留。该任务范围决定不改变全局服务规范对其他新服务或后续边界改造的要求。

## 2. 审核结论与需要修订的证据

### 2.1 已实现部分应直接复用

| 事项 | 当前证据 | 对新方案的影响 |
| --- | --- | --- |
| 默认图仍为六应用 | `internal/devruntime/fullstack.go:18` | 十一应用扩编仍需实现，六应用 smoke 不能代替目标验收 |
| Trading 独立 main、迁移和局部运行已有 | `cmd/athena-worm-trading/main.go:10`、`cmd/athena-worm-trading-migrate/main.go:21`、`internal/devruntime/registry.go:34`、`worm_trading.go:95` | 复用这些入口，只补全栈端点、依赖和验证；不再安排重复拆分 |
| Trading 已独占按需目录，Markets 已删除 | [退役证据](../../testing/worm-markets-retirement-acceptance.md) | 删除新设计中的 Markets 准备、凭据、health 与停止节点；不重复删除数据库 |
| Solana 和 Trading 已有内部鉴权及账户读取 | `internal/solanadiscovery/rpcservice/service.go:62`、`cmd/athena-worm-trading/commands/athena-worm-trading.go:117` | 保留并回归现有可信身份、账户查询、Wallet signer 与恢复契约 |
| 前端已有请求取消、身份代次和敏感写入范围 | `ui/src/app/shared/use-visible-query.ts`、`services/requests.ts`、`sensitive-write-scope.tsx` | 在现有机制上增加独立的板块访问代次，不能把开关变化伪装成登出 |

### 2.2 当前实现差距与范围评估

1. **三个服务存在内部调用鉴权差距，本轮不改造。** Market Radar、Managed OO、Profit Sharing 的 `CreateGRPC` 分别在 `internal/marketradar/server.go:41`、`internal/managedoo/server.go:47`、`internal/profitsharing/server.go:34` 裸建 gRPC server。Profit Sharing 的 `CreateRound`（`service.go:73`）依赖请求中自报的管理员标识。旧方案“内部服务沿用既有鉴权”不能当作已满足事实；同时，用户已明确用服务器规则阻止外部直连，这些源码事实不证明外部用户能绕过统一入口。保留此审核事实，按第 1.1 节决定移出本轮实施与验收。
2. **迁移归属没有完全分离。** Wallet、Managed OO、Profit Sharing 仍在 `NewSQLStoreSource` 调用 `ConnectAndMigrate`（分别 `store/sql_store.go:112`、`:36`、`:39`）；Solana 在 command 中执行 `Migrate`。不能只改 Solana，再宣称所有 external 借用库只读验证。所有本轮新增或改造的运行入口都须拆分显式迁移与运行时连接。
3. **独立构建与独立停止都不完整。** Wallet、Profit Sharing 全栈仍经聚合 `cmd/main.go` 构建；另有 Manager、Market Radar、Managed OO 缺独立 main。相关 command 中裸用 `GracefulStop`，部分后台 `Wait` 无界。补 main 时须一起补配置正向选择、取消传播、停止预算与直接运行验证。
4. **运行器的失败与停止模型仍是旧实现。** `launch.go:340` 先等 Trader Sync 就绪，初始失败会结束整个实例；`runner.go:181` 仅记退出码。停止遍历进程直接发 TERM，并在 `:204`、`:363` 清空失败。要落实“业务失败保留核心”，必须增加逐服务结果、停止依赖和独立失败记录。
5. **全栈存储准备仍包含未选模块。** `fullstack.go:36`、`:82` 仍准备 Token、Temporal 数据库。新实例只准备所选服务实际使用的库；已有历史库保持原样，不把普通启动变成删除操作。
6. **Gateway 与状态清单没有统一事实来源。** Manager 读取 host:port，管理员读取纯 IP 并固定 6776（`internal/server/servicestatus/service_status.go:129`）；当前管理员 Services 也只有七个远端 checker（`:59`），不同于本地运行图。不能用该列表数量证明十一应用启动。

旧全栈 spec 中“业务 7/7”、Trading 在 Markets 前停止、Trading 独立入口尚未完成等表述均属于过时目标。新设计用五核心、六业务与现行依赖图替代，不修改历史批准记录来伪造实施历史。

## 3. 方案比较与推荐

| 方案 | 范围与代价 | 判断 |
| --- | --- | --- |
| 只把五个应用加进 `make run`，沿用旧访问设计 | 改动最少，但启动时数据库结构准备与停止缺口仍需处理 | 不推荐直接照搬 |
| 复用已有重构，只完成当前目标所需改动 | 十一应用启动、六板块 API 开关、管理员页面及必要消费者；三服务内部鉴权重构不纳入 | **用户已选择此范围** |
| 新增 Runtime Control、网关或分布式控制服务 | 新增部署、状态同步和故障边界，且没有关闭后台任务需求 | 不采用 |

保留 API 内统一准入和每请求读库；不为访问开关引入 Redis 缓存、事件总线或独立控制服务。实现可按依赖拆分任务，不强制增加三份独立计划或审批；整体交付覆盖启动、API 开关、页面消费者及真实验收。

## 4. 目标运行图与服务前置

### 4.1 十一应用清单

| 分类 | 应用／默认端口 | 当前入口 | 本轮所需工作 |
| --- | --- | --- | --- |
| 核心 | API Server／8080 | 独立 main 与局部注册已有 | 接入配置与准入，验证未启动 Token 不影响核心启动 |
| 核心 | UI／4000 | Vite 与局部注册已有 | 访问状态边界与管理员操作 |
| 核心 | Notification／8086 | 独立 main 与局部注册已有 | 保留 sender 恢复屏障，初始就绪预算 180 秒 |
| 核心 | Wallet／8088 | command 已有，独立 main／局部注册缺失 | 最小构建、显式迁移、运行时 verify、有界停止 |
| 核心 | Etherscan Manager／8100 | command 已有，独立 main／局部注册缺失 | 正向配置、Gateway 地址一致性、有界停止 |
| 业务 | Trader Sync／8122 | 独立 main 与局部注册已有 | 复用运行代次、权限、同库事务；补统一编排 |
| 业务 | Solana Discovery／8112 | 独立 main 与专用 profile 已有 | 接入统一 owner，迁移归属与同账户库校验 |
| 业务 | Market Radar／8092 | command 已有，独立 main／局部注册缺失 | 独立入口、声明必要依赖、就绪与有界停止；复用现行业务接口 |
| 业务 | Managed OO／8106 | command 已有，独立 main／局部注册缺失 | 同上，并分离数据库结构准备与运行连接 |
| 业务 | Profit Sharing／8108 | command 已有，独立 main／局部注册缺失 | 独立入口、结构准备／运行连接分离、有界停止；复用现行身份传递与权限逻辑 |
| 业务 | Worm Trading／8090 | 独立 main、migrate 与局部注册已有 | 复用并接入全栈，保持目录、凭据、交易恢复边界 |

“全部”明确指本期清单，不含 Token 旧九角色、未来 Trader Sync 手动交易服务、退役服务、运行器和一次性工具。五个远端 Gateway 及基础设施另行展示。

### 4.2 本轮服务改动的限制

本轮不新增三个服务的内部凭据体系、Actor 协议、账户状态连接池或服务侧权限重构，也不调整 Profit Sharing 的权限权威归属。现有 API 认证与授权继续执行，板块开关作为通过原检查后的附加准入；Solana、Trading、Trader Sync 等现有内部鉴权继续保留，受影响时做针对性回归。

业务服务改动以独立构建、按需启动、配置连接、数据库结构准备、就绪和有界停止为限。保留原有同步结果、事务与恢复契约；不把 Managed OO 同步扫描改为异步任务，不引入新的后台控制协议。

访问开关验收验证统一 API 对用户请求的控制；实际部署时验证内部业务端口的外部隔离。本地实现交付不要求另行操作远端服务器或开展外网端口扫描；部署证据未产生时如实记录。现有服务鉴权差距不作为本轮新增验收条件，不声称这些差距已经修复。

### 4.3 存储与资源归属

- 新空实例的应用数据库为 `athena`、`wallet`、`managed_oo`、`profit_sharing`、`worm_trading` 五个。账户库由账户 schema 工具准备，Solana 业务 schema 与账户 schema 位于它声明的同一库；Market Radar 内存读模型不新增业务库。数据库数量不含 PostgreSQL 系统库。
- `make run` 沿用本地 managed 模式，由运行器显式调用各 schema 的 up／verify；服务启动只连接并 verify。本轮不新增全栈 external 模式。局部 `run-service`／`run-services` 的 external 模式只验证所选服务及必要账户库，不执行 CREATE、迁移、seed 或停止借用资源。
- 保留现有服务的账户连接、池所有者和健康边界，本轮不为三个服务额外引入账户只读连接。同一物理 PostgreSQL 是共同故障域，不声称彼此数据库完全隔离。
- 普通 `run`／`stop` 不删除旧 Token、Temporal 或其他历史库；新实例“不创建未选库”与已有实例“数据保留”分别验收。不承担 Token 归档清理任务。
- 不使用零值 DSN、假服务或空数据库满足无关初始化。当前 Token 客户端为非阻塞连接，未发现 API 必须等待 Token 可达的事实；只移除全栈 Token 存储准备，核验并解除实际存在的配置耦合，不借本轮删除 Token 公共接口。配置白名单按消费者声明，API 不获得 Trading 的 Wallet signer 凭据。
- 账户 schema contract 当前校验整个 public schema（`internal/accountstate/schema/catalog/catalog.go:19`）。新增访问表必须同步 migration、sqlc 与 contract；部署前识别并协调全部实际同库消费者，停止需升级的消费者后执行 up／verify，再以匹配 contract 的二进制恢复。不假定“只新增 API 使用的表”就对其他进程无版本影响，也不重写历史迁移。

这里的“迁移”指创建或升级数据库表结构；本轮不设计历史数据搬迁、旧协议兼容或双版本运行。现有业务 migration 直接复用，新增业务表只有访问设置表；其余改动是把已有结构准备从进程启动移到明确的准备步骤，并补只读验证。

#### 数据库准备入口

| 数据库／schema | 运行器使用的准备入口 | 本轮改动 |
| --- | --- | --- |
| `athena.public` | `athena-account-state-migrate up`／`verify` | 复用独立工具；追加访问表 migration，同步 sqlc 和 contract |
| `athena.solana_discovery` | `athena-solana-discovery schema up`／`verify` | 新增子命令，复用 `Store.Migrate` 的嵌入 SQL；增加只检查本 schema 的只读 verify |
| `wallet` | `athena-wallet schema up`／`verify` | 新增子命令，复用 Wallet migration；补有效版本及必需关系、列的只读校验 |
| `managed_oo` | `athena-managed-oo schema up`／`verify` | 同上，使用 Managed OO migration |
| `profit_sharing` | `athena-profit-sharing schema up`／`verify` | 同上，使用 Profit Sharing migration |
| `worm_trading` | `athena-worm-trading-migrate up`／`verify` | 复用独立工具及现有验证强度，不另建完整 catalog 指纹 |

上述四组 `schema` 子命令是待实现接口，只读取本 schema 必需的 DSN、执行存储操作后退出，不启动扫描、通知或 RPC，也不要求业务上游凭据；统一支持 `--timeout=120s`。不新增四套迁移框架，不经聚合 `cmd/main.go`，不调用 `--module all`，不以 Goose `status` 代替只读 verify。

managed 全栈的共享库准备顺序固定为账户 up → 账户 verify → Solana up → Solana verify → 账户 verify 复核，再启动应用；其余四个业务库按各自 up／verify 完成准备。局部选择只准备所选服务声明的 schema，选择 Solana 时才执行上述账户／Solana 组合顺序；局部 external 模式全部只 verify。Wallet、Managed OO、Profit Sharing 单独运行不准备账户库或 Solana。Solana SQL 只操作 `solana_discovery` schema，不写 `public.goose_db_version`，与账户 public contract 无冲突，不需要合并迁移或放宽账户校验。全栈显式将 `ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN` 指向本实例账户库，避免专属 DSN 的优先级选到其他库。

Wallet／Managed OO／Profit Sharing 的服务运行路径从 `ConnectAndMigrate` 改为连接后只读 verify；仅设置 `ATHENA_POSTGRES_AUTO_MIGRATE=false` 不等于已有 verify。Solana 运行路径移除自动 `Migrate`，在现有同一 pool 上分别验证账户和业务 schema，不新增账户池。验证失败时不启动业务；空库验证也不得创建版本表。

账户 contract 的现有消费者为 API、Notification、Trader Sync、Trading，以及同库读取账户的 Solana；前四项已有账户 verify，Solana 本轮补齐。升级时识别并停止该实例实际运行的这些消费者，完成新结构及 contract 验证，再启动一致版本。Wallet、Market Radar、Managed OO、Profit Sharing 不因此成为账户库消费者。

## 5. 统一编排设计

运行器内单份 `ServiceSpec` 负责入口、最小依赖、配置白名单、监听与消费地址、schema owner、就绪方式、启动类别和停止依赖。`make run` 选定上述十一项；局部入口只选用户指定应用及基础设施，不隐式启动其他业务进程。

API 的健康列表独立消费其声明的客户端，不导入运行器执行器、Docker 或本地状态文件。两者可共享纯数据标识或通过覆盖测试核对名称，但“本地进程已退出”“远端 RPC 不可达”“用户访问关闭”必须独立展示。

### 5.1 配置与阶段

1. 校验 checkout／实例、所选服务配置和端口归属；选择 UI 才要求 Node，需要托管基础设施才要求 Docker。输出明确阶段与日志。已有同实例 supervisor 时核验并报告归属及状态，不重复启动、不附着热更新；其他归属不抢占。
2. 构建各独立 main 和所需 schema 工具，准备自有基础设施和 schema。局部构建不经过聚合入口。
3. 启动 Wallet、Notification、Manager，再使 API／UI 核心入口可用；双 realm bootstrap 和 UI 代理可用后输出“核心可用，业务仍在启动”。
4. 启动六项业务。每项按其依赖与真实恢复语义判定 ready；后台上游退化与进程不可用分开记录。
5. 十一项均就绪才报告全栈就绪；同时列外部依赖状态、访问设置读取结果、页面 URL、日志和停止命令。不把全栈就绪等同于业务验收。

Manager 与管理员 Gateway 检查采用同一组已选择地址。首版保持当前纯 IP＋6776 范围：解析两份配置后比较规范化地址集合，拒绝冲突，向两个消费者注入一致值；不默默选其中一份。任意 hostname／端口扩展不属于本轮。

业务端口由各服务配置键唯一解析后显式传参，并注入消费者地址。Profit Sharing 正式采用命令已有的 `ATHENA_PROFIT_SHARING_LISTEN_PORT`，移除运行器对 `ATHENA_PROFIT_SHARING_PORT` 的另一套解释，不保留兼容别名。运行器为所有本地监听显式注入 loopback 默认值，不能依赖若干命令现有的 `0.0.0.0` 默认值。UI origin、callback 和 BaseHRef 一起校验。

managed 全栈依据实际本地监听端点覆盖配置中残留的远端业务消费者地址；五个已选 Gateway 是明确保留的外部例外。内部凭据按用途持久生成／复用并一致注入两端，Trading 已有加密 key 必须复用，不能随重启轮换；显式配置与持久身份冲突时报告问题，不猜测替换。API 健康列表中的现有 Token 行不作为十一应用完成条件，保留真实探测结果；本地启动摘要和访问页签另明确 Token 本期接入延期，不伪造其健康状态。

#### 配置与就绪清单

下表端口键均带 `ATHENA_` 前缀，地址默认 `127.0.0.1`；Trader Sync 的键本身包含 host:port。Go 构建包均为 `./cmd/<程序名>`，表中“新增 main”对应第 4.1 节五个缺失入口。

| 程序／入口 | 正式端口配置键 | 必要资源与就绪依据 |
| --- | --- | --- |
| `athena-server` | `SERVER_PORT`，运行器转 `--port` | 账户 PostgreSQL、Redis、MinIO／头像 bucket；HTTP `/healthz` 与双 realm bootstrap |
| UI／现有 Vite | `UI_PORT`，转 `--port`，使用 `--strictPort` | Node、UI 依赖与 API 代理；会员／管理员 HTML、资源和代理响应 |
| `athena-notification` | `NOTIFICATION_PORT`，运行器转 `--port` | 账户 PostgreSQL、Telegram 及现有恢复屏障；空 service 名 gRPC Health |
| `athena-wallet`／新增 main | `WALLET_PORT`，运行器转 `--port` | Wallet PostgreSQL、现有加密及内部凭据；空名称 gRPC Health |
| `athena-etherscan-manager`／新增 main | `ETHERSCAN_MANAGER_PORT`，命令原生读取 | 五 Gateway 与既有凭据，无数据库；空名称 gRPC Health |
| `athena-trader-sync` | `TRADER_SYNC_LISTEN_ADDRESS` | 账户 PostgreSQL、Ethereum HTTP／WSS、既有凭据及 transport；Health service=`tradersync.internal.v1.TraderSyncService` |
| `athena-solana-discovery` | `SOLANA_DISCOVERY_PORT`，命令原生读取 | 同库账户及 Solana schema、Solana RPC、既有内部凭据；空名称 gRPC Health |
| `athena-market-radar`／新增 main | `MARKET_RADAR_LISTEN_PORT`，命令原生读取 | 内存读模型、Polymarket、启用通知时的 Notification；空名称 gRPC Health |
| `athena-managed-oo`／新增 main | `MANAGED_OO_LISTEN_PORT`，命令原生读取 | Managed OO PostgreSQL、Polymarket／Polygon、启用通知时的 Notification；空名称 gRPC Health |
| `athena-profit-sharing`／新增 main | `PROFIT_SHARING_LISTEN_PORT`，命令原生读取 | Profit Sharing PostgreSQL；空名称 gRPC Health |
| `athena-worm-trading` | `WORM_TRADING_PORT`，命令原生读取 | Trading 及账户 PostgreSQL、Wallet signer、既有 Solana 检查和恢复；空名称 gRPC Health |

监听 host 使用各程序现有 `ATHENA_<SERVICE>_LISTEN_ADDRESS`，UI 使用 Vite `--host`；Trader Sync 按其现有完整地址与 TLS／transport 配置连接。配置白名单保留各服务现行扫描、超时与限流项，不为全栈另改业务默认值。Solana 保留 start slot 0、每范围 4 slots、并发 1、每秒请求预算 1、请求超时 15 秒、轮询 2 秒。

消费者注入明确覆盖：API 的 Notification、Wallet、Market Radar、Managed OO、Profit Sharing、Trading、Solana、Trader Sync 八项 `ATHENA_<SERVICE>_SERVER_ADDRESS`；Market Radar／Managed OO 各自的 `...NOTIFICATION_SERVER_ADDRESS`；Trading 的 `ATHENA_WALLET_SERVER_ADDRESS`；UI 的 `ATHENA_API_URL` 和目前白名单缺少的 `ATHENA_SERVER_BASEHREF`。局部组合只覆盖已选服务地址，其余使用明确配置的外部依赖，不隐式启动其他应用。

核心 bootstrap 对 `/api/v1/app/bootstrap` 分别携带 member／admin realm，核对实际 realm 和会话语义，匿名登录态可接受；所有路径加实例 BaseHRef。服务健康表示当前契约下可服务，不表示 Solana 首轮扫描、市场数据回填或真实交易已完成。Manager 就绪与 Gateway 可达分开报告；五 Gateway 仅做每个 3 秒预算的并行只读健康检查，不执行额度或业务探测。Manager 当前没有本期其他应用的必需调用者，API Gateway 检查直接连接 Gateway，不虚构调用依赖。

### 5.2 失败、状态与停止

- 核心启动失败：结束本轮实例并验证自有资源收尾。业务启动失败：只清理该服务本轮残留，继续其他业务，保留核心供检查并明确“全栈启动未完成”。不改写访问设置。
- 构建或共享准备失败、启动阶段取消、总启动预算到期都收尾并非零退出；记录失败／取消原因。保留前台供检查时以状态查询判定全栈结果，不能等待 `make run` 退出码。启动完成后正常停止仅在本轮无启动、运行与收尾失败时返回 0；状态查询命令是否成功与全栈是否健康分别表达。
- 本轮新增状态分别保存阶段、每服务启动结果、当前探测结果、进程退出事实和收尾结果。运行中任何应用非预期退出，即使退出码为 0，也撤销全栈就绪并保留事件；不自动重启、不静默清空失败。
- 沿用已确认预算：launcher 5 分钟、每构建 10 分钟、基础设施 5 分钟、每 schema 步骤 120 秒、普通就绪 60 秒、UI 120 秒、Notification 180 秒、全栈启动总预算 30 分钟。deadline 覆盖实际操作；不能把 helper 的停止预算当成构建超时。
- 服务自身关闭默认 30 秒，沿用现有服务明确声明的预算与恢复契约；新增／改造入口补齐取消链。先拒绝该进程的新调用，再取消并等待本服务工作、有界停止 transport、关闭自有连接；这不写持久业务访问开关。运行器兜底不能替代独立二进制自身可停止。
- 实例停止依赖顺序为 UI／API 用户入口 → 六业务 → Wallet／Notification／Manager → 自有基础设施，同层可并行，总收尾预算 180 秒。按实际调用依赖核验，清单中不存在 Markets。
- PID／PGID、启动 ticks、boot ID、run ID、不可变可执行文件及容器 ID／标签的身份验证继续复用。旧 profile 现场只用原 owner 收尾；统一 owner 后移除 Solana 第二条全栈启动路径。
- stop/status 优先使用本实例已保存且校验通过的运行器，避免当前源码编译失败阻止停服；无可用已保存程序才执行有进度、有预算的 bootstrap。停止保留数据、日志和本轮失败事实。

## 6. 访问准入设计

### 6.1 当前覆盖与控制位置

本基线六板块公共入口仍与原设计一致：**39 个 RPC、28 项 Trading 业务 HTTP 注册、16 项 Trading 专属验证注册和四种 Google callback purpose**。这些是当前盘点，不是永久写死的安全范围；实现应枚举实际服务描述符、HTTP 方法／命令分支及认证 purpose，检查每个入口都有受控、核心或明确延期分类。

- 统一 gRPC 检查位于原身份／角色／模块授权成功后、handler 前，覆盖 unary 与 stream；HTTP JSON、直接 gRPC 和 gRPC-Web 共用。避免管理员或 Profit Sharing 的提前授权分支跳过开关。
- 原始 Trading HTTP 和验证入口显式调用同一准入能力；不能仅按 `/api` 或 `worm` 字符串拦截，也不能把整个 `/auth` 关闭。Google callback 在可信 state 校验与消费、会话和权限确认后，签发授权或产生业务副作用前重验 `worm`。
- 未分类公共业务入口失败关闭并给出可诊断配置错误；核心、协议基础设施和明确延期接口显式登记，防止误封 health/reflection 或把 Token 当作受控完成。
- 每个新请求从主库读取当前配置，操作最多 2 秒且服从上游 deadline；缺行按关闭，读库失败按无法确认。已通过检查的请求允许自然完成，关闭不撤销已经准入的工作。
- 用户开关只存在 API 准入边界，后台不受开关影响。服务内部现有身份与权限检查保持，本轮不新增第 4.2 节排除的鉴权重构；部署规则保证外部用户业务请求经过 API。`DisableAuth` 只影响开发身份选择，不能跳过开关。

### 6.2 配置、接口与错误沿用

账户库新增 `athena_module_access_setting`，键限定六个板块，保存 `is_open`、可信操作者账户 ID 和数据库时间；缺行表示默认关闭。显式目标值 upsert，提交后才返回成功；最后提交者生效，不引入 CAS、反转命令或操作队列。设置操作不修改账户权限 revision，不触发订阅撤权或通知资格撤回。

字段沿用原设计：`module_key TEXT PRIMARY KEY` 限定 `trader_sync`、`solana`、`market_radar`、`managed_oo`、`profit_sharing`、`worm`；`is_open BOOLEAN NOT NULL DEFAULT FALSE`；可空 `updated_by_account_id UUID` 引用账户；可空 `updated_at TIMESTAMPTZ`。缺行不补 seed；首次设置写入行，同值提交也更新最后修改信息。不提供删除、重置或额外操作历史。

沿用原设计的三个接口：

| RPC | HTTP | 作用与边界 |
| --- | --- | --- |
| `ListModuleAccessStates` | `GET /api/v1/module-access-states` | 已认证含 Pending 账户读取最小状态，不返回修改人 |
| `ListModuleAccessSettings` | `GET /api/v1/admin/module-access-settings` | 持久管理员查看设置与最后修改信息 |
| `UpdateModuleAccessSetting` | `PUT /api/v1/admin/module-access-settings/{module_key}` | 持久管理员且交互式会话设置明确目标值；API Key 不能写，本地 `local-admin` 可用 |

新增 proto 位于 `internal/server/moduleaccess/moduleaccess.proto`，生成到 `pkg/apiclient/moduleaccess`。状态列表固定六项、顺序与第 4.1 节业务清单一致，无分页；只返回 `module_key`／`state`。管理列表及更新响应额外返回最后修改人账户 ID／用户名和时间，缺行的修改信息为空。更新只接受 `module_key` 和 `state`，操作者与时间由服务器确定；`OPEN`／`CLOSED` 是两种合法状态，`UNSPECIFIED`、未知键和 Token 返回 `InvalidArgument`。

现有 `authorizeGRPC` 的管理员分支会提前返回，不能只把更新方法同时加入管理员和交互式方法映射就认为两项都执行。该更新 handler 必须在既有持久管理员检查后明确验证可信凭据为交互式会话；API Key 写入拒绝、`local-admin` 成功分别留测试证据。三个设置／状态接口显式登记为核心，不被业务开关反向封锁。

保存超时或响应丢失后读取实际设置，不自动重发。关闭为 503／`MODULE_ACCESS_CLOSED`，配置不可确认为 503／`MODULE_ACCESS_UNAVAILABLE`，均保留 `google.rpc.ErrorInfo` 的 `athena.module_access` domain；原始 HTTP 同样保留稳定 reason。身份和权限错误保持原语义，不能把内部故障或关闭误当登出。

原始 HTTP 当前 `internal/walletsecret/errors.go:101` 只序列化 code／message／reason；实施须同步加入可解析的 ErrorInfo 与 module_key，并验证 `ui/src/app/shared/services/requests.ts` 消费结果，不能只修改 gRPC 状态详情。保留现有错误 envelope、reason header 和 private／no-store 响应约束；前端错误详情增加模块标识，列表整体读取失败则不填模块。Google 重定向按可信 purpose 传递专用原因后重新查询访问状态，不恢复已消费 state，不把关闭归类为普通登录失败。

## 7. 页面方案与异步边界

沿用已实施的[单一深色主题](../../requirements/web-ui/visual-theme.md)及 Service Status 原三页签，在其中增加 **Module Access**。六行设置、Token 延期提示、最后修改人与时间、明确“开放／关闭”操作；关闭采用中性状态，不使用进程停止或故障文案。桌面表格、手机分隔行，行级保存进度，保存成功后更新，未知结果重新读取，无全表提交或新增确认弹窗。

本页固定说明“关闭访问会拦截新的用户请求；后台任务和通知继续运行”。健康和访问查询互不阻塞；管理员不能因关闭业务而失去管理开关入口。无新增视觉风格，新页签交互草案待本轮方案确认，实际视觉证据在实现后提供。

会员与管理员各自已认证应用持有独立访问状态 provider。按现有 `member/app.tsx`、`admin/app.tsx` 路由装配，先判断原权限再挂载业务边界；Wallet、账户、通知及管理核心不等待访问配置。Trading 七条现行路由及其弹窗统一受 `worm` 控制。

- 前台每 2 秒单次读取，状态有效期从请求开始计最多 5 秒，以单调时钟计算；过期、离线、隐藏、切回或 pageshow 后重新确认前，不显示旧开放正文。关闭也继续查询以发现重新开放。
- 访问代次独立于身份及账户授权代次。关闭／未知时取消相应请求、失效旧读取、卸载业务组件并销毁本地草稿；迟到响应、旧状态读取和签名结果不能恢复已关闭页面或触发后续写入。
- 不能直接用现有 `useVisibleQuery` 作为整个准入实现：它在错误时保留 stale data，且 focus 直接调用 loader。访问 provider 必须先失效、确认新鲜状态再允许业务 loader；通用 hook 在核心页面原有 stale 行为不改。
- Trading `worm-trading-executions.tsx:770` 已有 epoch 与 abort，可复用并接入板块代次。关闭时停止心跳、`execute-next`、排队 timer 和后续签名处理，不自动发送 pause、terminate 或撤销授权。
- 重新开放只加载权威现状，由用户明确继续操作；不因 Run 仍为 RUNNING 就自动恢复驱动。已受理的单笔／批量 Cash Out、预览构建和后台任务继续原恢复流程。
- 保留有权限的菜单、URL 和核心导航；无权限菜单按原规则隐藏，不把业务开关扩展成授权能力。

## 8. 实施分包与验证契约

按第 1.1 节已确认的范围收敛实现与验收；以下是当前目标的任务划分，不是已执行记录。内部鉴权重构已经移出本轮，无需再次确认。

1. **启动所需服务适配**：五个独立 main、相关数据库结构准备／verify 分离、有界停止和最小配置；现有 Trading／Solana／Trader Sync 边界回归。以独立构建、直接运行、就绪、停止和借用库无 DDL 等实际范围验证；不加入三服务内部鉴权重构。
2. **访问准入与页面包**：账户 schema、三个接口、完整入口分类、HTTP／callback 消费者、双应用访问边界和管理员页签。必须共同交付，不能只隐藏 UI 或遗漏直接 API、开发身份和迟到回调。
3. **十一应用编排与交付包**：统一注册表、五库按需准备、端点注入、Gateway 一致性、阶段状态、失败隔离、精准停止、文档及真实验收。不能以十一端口监听代替应用 readiness 或业务测试。

验收矩阵：

| 维度 | 必要证据 |
| --- | --- |
| 初次启动／重启 | 新实例首次六开关关闭；分别保存开放与关闭，重启保留；既有实例历史数据保留，未选库不被新建 |
| 请求闭环 | 39 RPC 与实际注册覆盖一致；Trading HTTP、开发验证、Google／Phantom 回调、API Key、管理员业务与直接协议均被检查；核心保持可用 |
| 部署网络边界 | 本地核对 loopback 默认绑定；实际部署时从外部网络验证业务端口隔离，本轮不自动执行远端操作或声称已经取得部署证据 |
| 现有权限与后台行为 | 原有认证、账户权限及既有内部鉴权保持；合法后台任务继续，现有交易权限／恢复不回退；本轮不验收三服务新增鉴权 |
| 页面竞态 | 桌面／手机、根路径／`/athena`、2 秒轮询／5 秒过期、离线恢复、迟到读取、异步签名、直接详情和保存结果未知；开关不能误登出或自动重放交易 |
| 生命周期 | 十一独立构建与局部正向选择；核心失败、单业务启动失败、运行期退出、Notification 恢复屏障；停止后失败仍可追溯 |
| 所有权 | 多 checkout／INSTANCE、自有与 external 借用库、旧 profile；进程／端口／容器退出核验、数据卷保留、远端 Gateway 未被操作 |
| 真实环境 | 双 realm bootstrap 与系统 Chrome smoke，再按业务验收关闭／开放和后台继续；受控替身用于故障路径但不能替代真实运行证据 |

按 SDS-R1／R2／R3／R4／R5／R6／R7／R8 分别保留边界、配置、事务、运行与消费者证据。涉及 proto、sqlc、账户 schema 和 API 消费者的实现按实际依赖同步生成物；不手改生成代码。

具体沿用仓库现有工具，不在本轮设计阶段执行下面的实现验证：

- schema SQL → `make sqlc-local` →账户 store／访问 handler；账户 contract 用 `make account-state-schema-contract` 按现有生成工具更新，验证新空库、已有当前版本库、缺表／缺列及只读空库失败无 DDL。
- moduleaccess proto → `make protogen-fast` →生成的 Go API、HTTP 网关与 UI 消费者；验证实际注册入口分类、管理员提前返回分支、API Key 与原始 HTTP 错误详情。
- 对受影响的 `internal/devruntime`、账户 schema／store、API／HTTP／验证回调、服务 command 及存储包执行 Go 测试；运行器故障测试核对资源身份、取消、超时和退出事件，不能只断言配置结构相等。
- UI 执行相关 Jest 测试、`yarn lint` 和构建；用受控时钟及延迟响应覆盖访问过期、realm 隔离和停止交易驱动，再以真实浏览器验证桌面／手机、根路径／前缀与两个应用。
- 从目标工作区按声明启动 `make run`，保留十一项健康、双 realm bootstrap、Chrome smoke、六开关闭环及后台继续的真实证据；无另行保留要求时，按同一 INSTANCE 执行 `make stop` 并核对进程／端口／容器退出，保留数据与报告。收尾后按项目规则交付人工审查材料。

## 9. 本轮边界与后续确认

本轮产物是完成源码核对的修订技术方案，没有新增功能、运行验收、环境变更或最终人工交付结论。用户已确认聚焦当前目标，三服务内部鉴权重构排除；不把此次范围选择写成全部技术细节批准或已授权开始实施。

2026-09-17 静态审阅已核对配置／运行生命周期和 schema／访问接口两组内容，修正了局部启动与全栈准备顺序的歧义；文档链接、五核心加六业务清单及差异格式检查通过。这些证据只对应设计文档，不表示拟新增命令、十一应用或访问开关已经运行通过。

当前未留下需要用户另选的技术分岔。下一步由用户整体审阅本文，通过后整理一份可执行实施计划并进入开发。六个开关、后台继续运行、首次关闭、重启保留、网络隔离和 Token 延期均继续沿用既有决定，无须逐项重新批准；常规实现细节由执行者处理，不新增通用治理平台、独立控制服务或无关重构。
