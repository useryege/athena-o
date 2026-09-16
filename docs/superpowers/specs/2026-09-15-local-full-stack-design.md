# make run 全栈启动配套设计

> 状态：2026-09-16 方案已获用户采用，尚未实施；源码现状核对日期为 2026-09-15，Worm 补充核对日期为 2026-09-16。本文承接[访问开关需求](../../requirements/development-runtime/business-access-control.md)、[访问接口与页面设计](2026-09-15-business-access-control-design.md)及[服务清单](../../requirements/development-runtime/service-inventory-review.md)。
>
> 本轮只维护设计，没有执行 `make run`、连接远端 Gateway、恢复 Solana 采集或修改环境配置。当前实际命令行为仍见[本地运行说明](../../developer-guide/running-locally.md)。

> 范围修订（2026-09-16 最新决定）：[Markets 删除与 Trading 内聚](2026-09-16-worm-trading-market-query-design.md)方案 A 已采用，Markets 专属数据直接删除。目标调整为 11 个应用、5 个应用数据库；`worm` 只对应 Trading。删除与接入均尚未实施。

## 1. 目标与实现选择

保留 `make run` 作为日常人工验收的入口：一次命令准备本实例基础设施、构建并启动本期全部服务、打印可直接打开的地址。它保持前台运行并显示进度；管理员控制用户访问，运行器管理真实进程，两者职责明确。

继续扩展现有 `internal/devruntime`。它已有实例身份、持久资源、进程归属、日志和停止机制，不新增第二套运行器。与改为整套 Compose 或回到 Procfile 相比，这能直接复用已实现的 checkout／INSTANCE 隔离和独立服务入口。

本设计是本地编排配套；生产沿用自己的部署工具和进程管理器，不在生产机器执行这套开发命令。服务清单与配置职责应对应，但生产不启动 Vite，本地也不管理远端 Gateway 的进程。

## 2. 本期服务清单

目标覆盖 **11 个本地常驻应用程序**，不包括运行器、一次性准备工具及基础设施容器。下面是拟实施清单，不是当前 `make run` 已有能力；默认端口均绑定 loopback，可按实例显式改配。

| 运行器标识 | 程序 | 分类 | 默认端口 | 构建入口 |
| --- | --- | --- | --- | --- |
| `api-server` | `athena-server` | 核心 | 8080 | 已有独立入口 `./cmd/athena-server` |
| `ui` | Vite／Node | 核心 | 4000 | 已安装的 UI 依赖 |
| `wallet` | `athena-wallet` | 核心 | 8088 | 补齐 `./cmd/athena-wallet` 独立 main |
| `notification` | `athena-notification` | 核心 | 8086 | 已有独立入口 `./cmd/athena-notification` |
| `etherscan-manager` | `athena-etherscan-manager` | 核心 | 8100 | 补齐 `./cmd/athena-etherscan-manager` 独立 main |
| `trader-sync` | `athena-trader-sync` | 业务 | 8122 | 已有独立入口 `./cmd/athena-trader-sync` |
| `solana-discovery` | `athena-solana-discovery` | 业务 | 8112 | 已有独立入口 `./cmd/athena-solana-discovery` |
| `market-radar` | `athena-market-radar` | 业务 | 8092 | 补齐 `./cmd/athena-market-radar` 独立 main |
| `managed-oo` | `athena-managed-oo` | 业务 | 8106 | 补齐 `./cmd/athena-managed-oo` 独立 main |
| `profit-sharing` | `athena-profit-sharing` | 业务 | 8108 | 补齐 `./cmd/athena-profit-sharing` 独立 main |
| `worm-trading` | `athena-worm-trading` | 业务 | 8090 | 保留现有 commands，补齐独立 main 与运行器接入 |

- 五个远端 Etherscan Gateway 作为已有外部核心服务连接，既不构建也不启动本地副本；Manager 与管理员网关检查使用同一组已选择的地址。
- Token 完成重构后再确定进程接入，不把旧九角色加入本期全栈。启动摘要显示“Token：接入延期”，不显示启动成功或虚假可用状态。
- 两个 BSC 索引器、Sports、World Cup Corners 不加入目标清单。Trading 保留；Markets 从目标清单移出，其交易目录由 Trading 承接。Temporal 没有核对到当前运行消费者，本期不创建其空数据库或常驻进程。
- `FullStackServices()`、单服务注册表和服务状态读取共用同一份程序定义，取消全栈私下覆盖 Wallet／Profit Sharing 构建方式的做法。每项定义包含构建入口、参数、环境白名单、存储准备、探测方式及时间预算。
- 单独选择服务只准备其声明的基础设施；不隐式启动 API、UI 或其他业务程序。需要组合时使用显式 `run-services` 集合。

## 3. 配置、基础设施与数据准备

### 3.1 本地配置

沿用 `ENV_FILE=.env`，导出的环境变量覆盖文件值；dotenv 作为数据解析，不能通过 shell source 执行。只把各服务声明的配置传给其进程，错误提示说明服务、配置项、来源文件和日志位置。

全栈为 `DB_MODE=managed`，由本实例生成并注入数据库、Redis、MinIO 和所选本地服务的消费者地址；这些本地地址优先于文件中残留的生产地址。外部 RPC、Gateway 与 Telegram 等供应商配置仍使用明确提供的值，不自动更换 provider 或凭据。输出应区分“本地进程／本地存储／远端共享 Gateway／其他外部依赖”。

所有本地监听地址默认 loopback。端口由现有配置或 `--port` 明确传入；Solana 已读取 `ATHENA_SOLANA_DISCOVERY_PORT`，运行器据此计算端点并显式传参。Market Radar／Managed OO 使用现有 `...LISTEN_PORT`，Manager 使用 `...PORT`；注册表必须统一计算监听和消费者地址，不能依赖各自猜测默认值。 修改 UI 端口或前缀时，同时校验 `ATHENA_URL`、BaseHRef、Google callback 和 SIWS origin 的一致性；登录信任来源仍由明确配置决定，不从请求 Host 或转发头推导。

新增实例所需的内部凭据沿用持久文件管理方式，按服务目的分别生成或读取，在对应 API／服务两端一致注入；重启复用。不能把内部 token 当作访问开关，也不生成供应商密钥。

本地存储隔离不代表第三方资源隔离：同一 Telegram Bot 的长轮询和资料更新不能被当作多个独立 Bot。配置与运行检查须报告既有 poller／webhook 冲突，遵循 Notification 的独占与恢复规则；不为让启动通过而接管其他环境、清除 webhook 或跳过恢复。网关额度是否独立仍由实际 API key 决定。

### 3.2 存储清单

| 持久资源 | 本期使用者与准备 |
| --- | --- |
| 本实例 PostgreSQL 的账户状态库 | API、Trader Sync、Notification、Solana 各自连接；包含账户、访问开关和相应业务 schema |
| `wallet` 数据库 | Wallet 的迁移与验证 |
| `managed_oo` 数据库 | Managed OO 的迁移与验证 |
| `profit_sharing` 数据库 | Profit Sharing 的迁移与验证 |
| `worm_trading` 数据库 | Worm Trading 的迁移与验证，已有数据保留 |
| Redis | API 当前会话与认证等依赖 |
| MinIO 及私有头像 bucket | API 头像资源；按既有配置准备并保留 |

Market Radar 当前读模型在内存中，Manager 不需要新增数据库。按实际服务选择创建上述数据库，不调用 `--module all`。停止后再次启动复用原 volume、账户、配置、业务记录与采集 checkpoint。

账户 schema 继续用独立 `athena-account-state-migrate` 执行 up／verify；Wallet／Managed OO／Profit Sharing／Worm Trading 的迁移工具增加独立 main，按显式模块选择运行，不经聚合 `cmd/main.go` 构建所有业务程序。

Solana 当前在服务启动内执行 `Store.Migrate`。接入时将迁移动作放到该服务命令的独立 `schema up`／`schema verify` 子命令，均只处理存储并退出；运行路径只读验证，不启动扫描后再准备 schema。上述子命令为待实现接口。它与账户 schema 在同一个本实例数据库执行，避免另起一个没有账户权限表的空库。

保留最小开发身份的既有初始化，重复启动不重新授予被撤销的权限，不自动创建订阅、下单或制造业务数据。访问开关表缺行按关闭处理，正常启动不执行“全部设为关闭”或“全部设为开放”。

当前 `fullStackModules()` 仍包含待删除模块与旧 Token 的准备。实施时移除这些全栈准备和强制配置依赖；Trading 数据库及配置保留；Markets 准备及配置删除。Markets、Sports 与两个 BSC 索引器的专属历史数据已确认直接删除，使用独立退役步骤处理；普通启动和停止不执行该删除。旧 Token 数据不在此次删除范围。若 API 初始化仍强依赖这些未选择模块，需解除该启动依赖，不能保留无效 DSN 或假服务来满足检查；这不扩展为旧 Token 业务重构。

### 3.3 Worm Trading 接入

Trading 拥有按需市场目录、组合、预览、交易、Cash Out 与恢复；API 只承接用户入口和准入，不启动 Trading 后台工作。目录组件无采集循环和 Markets 存储。具体业务与服务侧授权见[内聚设计](2026-09-16-worm-trading-market-query-design.md)。

补齐 `cmd/athena-worm-trading/main.go` 与独立 schema 工具，支持局部选择；只准备 `worm_trading` 和显式依赖的账户 schema 验证，不构建 Markets、不创建其库或复用其 DSN。

| 配置／资源 | 消费者与规则 |
| --- | --- |
| `ATHENA_WORM_TRADING_LISTEN_ADDRESS`、`ATHENA_WORM_TRADING_PORT` | Trading 默认 loopback:8090，运行器统一解析并显式传参 |
| `ATHENA_WORM_TRADING_SERVER_ADDRESS` | API 指向选定 Trading；无 Markets 地址注入 |
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | 所属服务及 schema 工具连接本实例 `worm_trading`，保留原记录 |
| `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` | Trading 新目录／改变的组合写 RPC 使用权限只读 reader，连接同环境账户库；账户迁移仍由其 owner 工具执行 |
| `ATHENA_WORM_TRADING_SOLANA_RPC_URL` | 沿用 mainnet、genesis、USDC program／decimals 检查 |
| 现有 Trading 预算／限流配置 | 保留 RPC attempt、balance budget、RPC rate／burst、Worm API attempt、position budget／concurrency 的现有配置及默认值 |
| `ATHENA_WORM_TRADING_CATALOG_BUDGET`（内聚设计新增） | 默认 45 秒，总预算包含限流等待；不小于默认 5 秒的上游单次预算 |
| `ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN` | API 与 Trading 的内部 Bearer；新目录同时携带可信账户身份并由 Trading 读取当前权限 |
| `ATHENA_WALLET_SERVER_ADDRESS`、`ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN` | Trading 连接同环境 Wallet 专用 signer，API 不获得 signer token |
| `ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY` | 仅 Trading 使用；已有数据必须复用原 key，缺失／冲突应失败，不能清空记录或换 key 冒充恢复 |

目录生产客户端固定到 Trading 已采用的 Worm 地址，复用未认证工厂限流器；不继承 Markets Base URL、通知开关、通知地址、端口或拟新增内部 token。API 进程不得从父环境继承 Trading signer token 和加密 key。

**就绪与失败**：核心 Wallet 及账户 schema 准备后启动 Trading。先验证配置、数据库与既有恢复，再由 Solana 探测按现有条件报告空名称 gRPC 健康。权限库是已知共享故障域，新目录／组合写请求在运行时读取失败则拒绝；已受理 worker 继续原恢复规则。Trading 的 SERVING 不等于 Worm 目录或真实交易已验收，上游失败独立显示；其他业务不因此停止，访问设置不改写。

**停止**：先关 API／UI 入口，再有界停止 Trading，最后按资源归属处理 Wallet 等自有依赖；保留 Trading 数据、原 key 和未决任务。Markets 专属永久退役按独立设计执行，不放入正常 stop 或 run。

## 4. 启动流程

按有限阶段串行组织，阶段内输出当前服务及耗时；首版不增加并行构建调度器或自动重启机制。

1. **入口提示**：进入启动入口立即打印仓库、实例和“准备本地运行器”。运行类 Make 目标不执行无关的全仓 Git dirty／tag 扫描；需要构建版本信息时，移到已经显示进度的构建阶段执行，不能伪造版本事实。
2. **工具与归属检查**：确认 Linux／WSL、Bash、Go、Docker 可用；全栈检查 `ui/.nvmrc` 对应 Node 与现有 Vite 依赖。确认实例锁、已有 supervisor／资源和端口，无健康检查或端口抢占副作用。
3. **配置检查与构建**：检查所选服务配置和地址一致性，构建独立程序及 schema 工具，日志从开始即落盘。先发现构建错误，再启动应用进程；Go 自身缓存可复用，但每次按当前源码验证构建。
4. **基础设施与 schema**：创建或复用本实例基础设施，执行所选 schema up／verify，准备头像 bucket，保存本实例地址与凭据。共享准备失败时停止本轮，不绕过失败步骤。
5. **核心入口就绪**：启动 Wallet、Notification、Manager，然后 API、UI；核对核心健康与会员／管理员 bootstrap。输出“核心入口可用，业务仍在启动”，此时不能宣布全栈完成。
6. **业务启动**：启动 Trader Sync、Solana、Market Radar、Managed OO、Profit Sharing 和 Worm Trading，逐项检查初始就绪；Markets 不启动。所需核心能力已在核心阶段启动，具体连接与配置按第 3.3 节注入。程序按原规则开始后台工作，即使对应用户访问仍关闭。
7. **结果摘要**：全部所选应用就绪后打印全栈结果、两个页面地址、访问状态、远端依赖状态、日志目录和停止命令。随后前台保持运行、等待 Ctrl+C，并继续报告进程退出事件。

本地配置检查不意味着连接已验证。基础设施创建和进程启动仍可能失败，必须保留当时错误；已存在的同实例 supervisor 不重复启动，报告其归属和状态入口。端口被其他实例或未知程序占用时报告冲突，不自行换端口、杀进程或接管现场。

## 5. 就绪、超时与失败

### 5.1 就绪含义

| 对象 | 初始就绪要求 |
| --- | --- |
| PostgreSQL／Redis／MinIO | 本实例身份匹配，连接与必要准备成功；数据库 schema 和 bucket 已核对 |
| 本地 gRPC 程序 | 进程身份匹配并报告约定的 `SERVING`；Trader Sync 使用已存在的具体内部 service 名，其余按实际注册的健康名称 |
| API | `/healthz` 可用；对应身份模式的会员／管理员 bootstrap 返回合法结构与预期 realm，不要求用户业务已开放 |
| UI | 会员和管理员 HTML 入口及资源可加载，Vite 到本实例 API 的代理可用 |
| 五个远端 Gateway | 只做并行、有界的只读健康连接检查，独立展示可达数；不自动运行带真实业务地址的额度／数据探测 |

`SERVING` 表示本地服务已准备接收调用，不表示行情采集已覆盖、外部供应商长期健康或业务验收通过。Manager 的 `SERVING` 也不等于五个 Gateway 全可达。外部 RPC／Gateway 暂时不可达时，若本地服务仍符合自身就绪语义，显示对应外部异常；不能伪造采集成功，也不自动关闭用户访问设置。

**建议默认时间预算**：运行器编译 5 分钟；单个应用／schema 工具构建 10 分钟；基础设施准备 5 分钟；每个 schema 步骤 120 秒；普通服务就绪 60 秒，UI 120 秒，Notification 180 秒；全栈启动总预算 30 分钟。单次本地探测 1 秒，远端 Gateway 每个最多 3 秒且并行。时间预算明确记录在运行器配置中，到期报告具体阶段，取消已登记的本轮工作并执行相应收尾。

Notification 的较长预算覆盖现有非首次启动至少 60 秒恢复屏障；更长 Retry-After 或未处理旧 sender 仍按既有规则等待／失败，不能缩短屏障来通过本地启动。运行器不自动调用旧 sender 恢复命令。

### 5.2 故障范围

- 工具、全栈配置、构建、共享基础设施／schema 准备失败：返回非零，回收本次已拥有的临时使用者，保留数据与日志。此时不输出核心入口可用。
- 核心程序初始启动失败：无法完成核心环境准备，停止本轮已启动的自有程序及基础设施，保留证据；不能通过跳过该核心服务继续报告全栈成功。
- 核心就绪后某个业务程序启动失败：停止并核验该程序本轮残留，继续尝试其他业务；保留核心和其他成功程序供查看。输出“全栈启动未完成”及失败服务、原因和日志，不把该状态叫作启动成功。访问开关仍保持数据库原值。
- 全栈运行期间任何进程异常退出：立即打印并记录退出事件，健康状态标记不可用，其他独立进程继续运行；不自动重启或重置开关。共享数据库等共同依赖故障仍可能影响多个服务，按各自真实结果报告。
- 用户主动停止或启动总预算到期：取消尚未完成的步骤，执行实例收尾。任何残留清理失败都记录为失败，不能只凭已发信号宣称已停止。

保留前台运行期间，不能用 `make run` 的退出码判断全栈已就绪；自动化应读取状态结果。目标 `runtime-status` 在原实例、资源、进程与日志信息上补充启动阶段、逐服务启动结果、全栈就绪布尔值和失败摘要，并继续执行当前健康核对；停止不能清空本轮已记录的启动或运行失败事实。查询成功与全栈健康分开表达，读取到失败状态本身不应伪装成命令查询失败。

完成启动后，用户正常停止且本轮无启动／进程／清理失败时退出 0；存在上述失败时退出非零。启动过程中 Ctrl+C 按取消标记并返回非零，完成启动后的 Ctrl+C 按正常停止处理。错误与取消不写成验收通过。

## 6. 终端输出与日志

从 Make 入口到运行器编译、镜像准备、构建、迁移和就绪等待均有即时提示。任何进行中的长步骤每 5 秒输出一次阶段、对象、已耗时和日志位置；使用普通追加文本，同时支持人工终端与 AI 日志采集，不依赖只在 TTY 可见的动画。

完整服务日志继续单独保存。启动前的运行器编译日志放在本次专用临时目录并从第一行输出路径；进入实例后，阶段日志和逐服务日志按本轮 run ID 保存在 `.run/instances/<INSTANCE>/`。新启动不覆盖旧日志，临时编译器也必须能响应取消并回收所属子进程。

失败终端显示服务、阶段、退出码／超时原因、日志末尾和完整文件路径。日志尾部限制为最近 30 行，完整日志保留；不输出整份环境配置来代替诊断。

以下仅为**目标输出示例，不是实际启动结果**：

```text
[开始] 仓库 /home/yege/work/athena；实例 full-stack；配置 .env
[构建] 正在编译运行器，已用时 5 秒；日志 /tmp/athena-.../launcher.log
[准备] 复用本实例 PostgreSQL、Redis、MinIO
[核心] 5/5 就绪，业务仍在启动
[业务] 7/7 就绪
[结果] 本期 11 个本地应用已就绪
会员：http://localhost:4000/
管理员：http://localhost:4000/admin/
用户访问：首次未配置的已登记板块默认关闭；后台任务正常运行
远端 Gateway：5/5 健康检查可达（不代表业务探测通过）
Token：接入延期
状态：make runtime-status INSTANCE=full-stack
停止：Ctrl+C，或 make stop INSTANCE=full-stack
日志：/home/yege/work/athena/.run/instances/full-stack/
```

地址必须根据实际 UI 端口、BaseHRef 与实例配置生成；示例端口不是固定承诺。访问状态从访问接口读取：已有管理员设置时显示实际开放／关闭数量，无法读取时明确写无法确认，不能每次启动都声称“默认关闭”。启动摘要只报告环境准备，不写“业务验收通过”。

## 7. 命令、停止与局部运行

保留日常命令：`make run`、`make stop`、`make runtime-status INSTANCE=...`；`INSTANCE` 默认 `full-stack`，启动和停止必须使用同一仓库及实例。前台日志持续运行是预期行为，不代表终端卡死。

成功创建实例时保存本轮不可变运行器的位置；状态与停止入口优先复用经过路径、内容及实例归属校验的该运行器，避免清理已有实例还必须重新编译当前源码。没有可用运行器时才进入带进度和预算的编译入口；无法验证时明确失败，不执行归属不明的文件。`runtime-status` 和 `stop-instance` 继续要求显式 INSTANCE，全栈 `run`／`stop` 的默认值保持一致。

按新注册表扩展 `make build-service SERVICE=...`、`make run-service SERVICE=... INSTANCE=...` 和显式 `run-services` 集合。新增 Wallet／Manager／Solana 等选择项在实现前不能当作当前可用命令。

Solana 纳入同一资源所有者后，停止使用 `ATHENA_RUN_PROFILE` 分支、Goreman／专用 Procfile 的第二套全栈管理路径；后续文档改用统一选择入口。旧 profile 现场仍须用它当前的归属记录和停止入口收尾，不能直接删记录或向新运行器冒认其资源。外部／借用数据库继续只读验证，不自动迁移、seed 或停止。

`make stop`／Ctrl+C 是停止本地环境的真实进程操作，与管理员关闭用户访问不同。停止顺序为 API／UI 用户入口、业务工作者、其使用的 Notification／Wallet／Manager，最后处理自有基础设施；Worm Trading 在 Markets 之前停止，无依赖的同层程序可并行，基础设施必须等使用者退出后再处理。

沿用 PID／PGID、启动时间、boot ID、执行文件、run ID、容器 ID 和标签的归属核验；单服务失败清理复用同一核验与信号函数，只选择本轮失败程序，不能按名称或端口杀进程。服务正常关闭预算沿用声明值，默认 30 秒；实例收尾预算设为 180 秒，包含已存在的强制退出与容器停止等待，结束后检查进程、端口和容器。

保留数据库、volume、配置、凭据、日志及验收证据；不自动执行 `run-reset`，不删除待退役模块数据，不操作五个远端 Gateway 或其他实例资源。无法证实退出时保留失败记录，并给出仍未停止对象及重试停止命令。

人工执行 `make run` 后由用户 Ctrl+C／`make stop` 收尾；AI 为验收启动的临时环境按 [AGENTS.md](../../../AGENTS.md#本地验收环境准备与完成标准)在任务结束时主动停止，只有用户明确指定的保留现场除外。

## 8. 人工与 AI 验收

人工路径保持简短：执行 `make run` → 根据摘要打开管理员入口 → 开放要验收的板块 → 在会员端查看结果 → 按需关闭访问或停止整个本地环境。已有开放设置重启后保持，不要求重复开关。

AI 实施验收覆盖：

1. 空实例与已有数据实例分别启动；11 个应用清单、5 个本期数据库及基础设施归属正确，Trading 与账户权限读取依赖纳入，Markets 不启动、不建库；旧 Token／Sports／Temporal 不被全栈隐式创建或启动。
2. 冷构建、缺 Node／UI 依赖、Docker 不可用、端口冲突、迁移失败和慢启动均有即时输出与有界结果；包含运行器编译阶段的 Ctrl+C。
3. 本地消费者实际连接本实例地址；Manager 与管理员检查连接同一组远端 Gateway；停止测试证明远端资源未被操作。
4. 核心启动失败、业务启动失败和运行期间退出分别符合故障范围；Notification 的真实恢复屏障不被跳过，外部不可用不被当作成功。
5. 独立服务构建不导入无关命令，局部选择只准备必要基础设施；Solana 使用同环境账户／业务 schema，后台扫描按本期运行目标执行。
6. 会员／管理员 bootstrap、真实页面 smoke 与访问开关验收通过；关闭访问后后台继续、重启保存开关与业务数据。页面 200 或端口监听不能替代这组验收。
7. 默认与自定义实例、不同 checkout、根路径与前缀路径、Ctrl+C 和外部停止均能精确收尾；数据库数据及证据保留，共享／其他实例未被影响。
8. Trading 独立构建与局部运行只准备声明存储；自定义端口同步消费者。无 Markets 程序／配置／数据库仍能运行；内部 Bearer、可信账户与服务侧权限正确，API 无 signer token／加密 key。权限库或 Solana 配置错误、就绪超时、上游失败和有界停止分别保留证据；原 key 重启解密与未决任务恢复不变。

本轮仅进行静态设计核对，上述运行结果均待实施验证。服务编排到位不等于全部业务重构完成；Token 与完整删除退役仍按各自范围交付。

## 9. 实施位置与规范

主要落点为 [Makefile](../../../Makefile)、[入口脚本](../../../hack/local-runtime.sh)、[运行器编译脚本](../../../hack/run-local-runtime.sh)、[注册表](../../../internal/devruntime/registry.go)、[全栈清单与数据库准备](../../../internal/devruntime/fullstack.go)、[配置](../../../internal/devruntime/environment.go)、[启动和健康探测](../../../internal/devruntime/launch.go)、[归属与停止](../../../internal/devruntime/runner.go)，以及相应独立命令 main、schema 工具和源码验证。

遵守[服务开发规范](../../developer-guide/service-development-standards.md) SDS-R2／R3／R4／R5／R7／R8：API 不承载新增后台任务、服务独立构建与配置、故障明确、资源按 owner 回收、文档区分目标与现状，验证覆盖实际改动。共享账户数据库的共同故障域与既有事务保持原设计；改动生成源时同步对应消费者。

**确认记录（2026-09-16）**：用户先采用 10 程序清单、核心先可用／业务失败保留核心、按阶段可见输出及有界等待、统一局部运行入口、停止保留数据；随后明确保留 Worm，并确认“统一使用一个开关。然后交易处理方式按照你的建议”。目标扩充为十二程序，本次已补齐 Worm 配置、授权、就绪、失败与停止设计。[删除与清理配套设计](2026-09-16-module-removal-cleanup-design.md)已完成自查及补充，三份设计均尚未实施；Token 专属接入继续延期。

**最新决定（2026-09-16）**：用户采用 Trading 集中承接按需目录、删除 Markets 及其专属数据；历史十二程序确认由当前十一程序目标覆盖。本次只同步目标文档，尚未扩展运行器或实施退役。
