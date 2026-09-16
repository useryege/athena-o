# 本地运行编排

> 设计状态：Trader Sync 独立进程、gRPC、schema 工具与实例运行器已实现；Redis持久重启修复、两次真实全栈重启及最终Chrome验收通过，见[验收记录](../../testing/trader-sync-independent-service-acceptance.md)。

> 关联目标已调整为[板块访问开关简化方案](../../requirements/development-runtime/business-access-control.md)：本期只限制用户访问，进程及后台任务继续运行；原整组运行控制、默认停任务和新增 Runtime Control 的设计已暂停。服务清单扩展、核心分类、两个 BSC 索引器与 Sports 删除决定、Worm 保留决定，以及本地独立资源、Manager 使用五个远端 Gateway 的边界继续保留。访问范围及默认／重启规则已确认，尚未实施：首次默认关闭，之后保留管理员设置，重启不改变开关；下文描述现有运行命令与资源管理。

> [make run 全栈启动配套提案](../../superpowers/specs/2026-09-15-local-full-stack-design.md)原十程序版本已于 2026-09-16 获采用；随后保留 Worm，目标扩充为十二程序，Worm 独立入口、配置、鉴权、就绪与停止设计已补齐；两业务服务共用一个 Worm 访问开关。独立入口、按需存储准备、即时进度、核心先就绪及业务失败隔离继续沿用。尚未实施；下文六程序清单与当前失败收尾方式仍是现状，不代表新提案已落地。

## 范围与服务边界

本地运行器提供按服务选择的构建、启动、seed、状态、停止和重置入口，以及显式全栈入口。遵守[服务开发规范 SDS-R1 至 SDS-R8](../../developer-guide/service-development-standards.md)。批准依据为[服务职责与事务](../../superpowers/specs/2026-09-13-trader-sync-service-boundaries-design.md)、[本地运行](../../superpowers/specs/2026-09-13-trader-sync-local-runtime-design.md)和[内部字段契约](../../superpowers/specs/2026-09-13-trader-sync-grpc-contract-design.md)；历史规格保留批准时的事实。

Trader Sync 在独立 `athena-trader-sync` 进程中运行。API Server 保留公共协议、身份/session、账户权限和 facade，通过内部 gRPC 调用 Trader Sync；API 不持有 Collector、Resolver、Projector 或目录刷新 runtime。Notification 自己拥有 Telegram Bot、poller、摘要协调、发送许可与结果。

三个进程分别创建和关闭自己的 PostgreSQL pool，连接同一个 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` 权威数据库。共享数据库是明确的共同故障域，不意味着共享内存 pool 或跨 RPC 传递事务。Wallet 与 Profit Sharing 的独立数据库只由显式全栈准备。

## 源码入口

| 职责 | 源码 |
| --- | --- |
| 用户命令与前台信号转发 | [Makefile](../../../Makefile)、[run-local-runtime.sh](../../../hack/run-local-runtime.sh)、[运行器 main](../../../cmd/athena-local-runtime/main.go) |
| 独立服务图与全栈图 | [registry.go](../../../internal/devruntime/registry.go)、[fullstack.go](../../../internal/devruntime/fullstack.go) |
| 资源归属、恢复与启停 | [internal/devruntime](../../../internal/devruntime) |
| Trader Sync 入口与生命周期 | [独立 main](../../../cmd/athena-trader-sync/main.go)、[runtime.go](../../../internal/tradersync/runtime.go) |
| 内部 gRPC、权限与公共 facade | [transport](../../../internal/tradersync/transport)、[client](../../../internal/tradersync/apiclient)、[公共 facade](../../../internal/server/tradersync) |
| schema 权威与独立工具 | [accountstate/schema](../../../internal/accountstate/schema)、[athena-account-state-migrate](../../../cmd/athena-account-state-migrate) |
| API 与 Notification 独立入口 | [athena-server](../../../cmd/athena-server/main.go)、[athena-notification](../../../cmd/athena-notification/main.go) |
| 生产镜像、TLS 与维护 | [部署说明](../../../deploy/trader-sync/README.md)、[Compose](../../../docker-compose.prod.yml) |

## 本地命令与最小依赖

运行器支持 Linux/WSL，依赖 Go、Docker 和 Bash 5.1+；只有选择 UI 时才需要项目规定的 Node 24 与已安装的 UI 依赖。命令从目标 checkout/worktree 根目录运行。`.env` 作为数据解析，不以 shell source 执行；已导出的环境变量覆盖文件，显式空值保留。

```bash
make build-service SERVICE=trader-sync
make run-service SERVICE=trader-sync INSTANCE=ts-dev
make seed-service SERVICE=trader-sync INSTANCE=ts-dev
make runtime-status INSTANCE=ts-dev
make stop-instance INSTANCE=ts-dev
# 仅在已停止且确认要丢弃本实例数据时：
make reset-instance INSTANCE=ts-dev
```

`run-service` 默认实例名为服务名；其他实例操作应明确 `INSTANCE`。`run-services` 显式指定集合和实例：

```bash
make run-services SERVICES='trader-sync api-server ui' INSTANCE=ts-integration
```

| 正向选择 | 业务进程 | 必要基础设施 |
| --- | --- | --- |
| `trader-sync` | 独立 Trader Sync | 本实例 PostgreSQL，或显式外部库 |
| `notification` | 独立 Notification | 本实例 PostgreSQL，或显式外部库 |
| `api-server` | 独立 API Server | PostgreSQL、Redis、MinIO 与已准备的私有头像 bucket |
| `ui` | Vite | 无数据库；按配置连接 API |
| 显式全栈 | Trader Sync、API、Notification、UI、Wallet、Profit Sharing | 本实例 PostgreSQL、Redis、MinIO |

API 和 Notification 的局部入口不会启动 Trader Sync。全栈保留八个既有模块数据库的准备和两个 Temporal 空数据库，未选择的历史模块不启动业务进程。Wallet/Profit Sharing 仍使用既有聚合构建入口，这个既有范围不进入 Trader Sync 的独立构建依赖。

`make run` 显式选择全栈图，默认实例名为 `full-stack`，允许通过 `INSTANCE` 指定别名；`make stop` 和 `make run-reset` 对应同一实例的停止与重置。它们使用相同的资源引擎，不能按固定容器名或端口清理其他实例。局部选择使用 `run-service`/`run-services`；全栈只接受 `DB_MODE=managed`，不能借全栈入口向外部库隐式创建其他模块数据库。

## 数据模式与身份

managed 模式为每个 checkout/实例创建独立持久 PostgreSQL。容器、volume 名含规范 checkout 路径与实例的 namespace 哈希；数据库、Redis 和 MinIO 的宿主端口由 Docker 动态分配且只绑定 loopback。业务默认端口仍为 UI4000、API8080、Notification8086、Trader Sync8122、Wallet8088、Profit Sharing8108，并存时必须显式指定不冲突的业务端口。

用 `runtime-status` 查询实际资源、地址、生命周期和各进程退出码；`.run/instances/<instance>/state.json`、日志和 fixture 属于该 checkout。运行器保存稳定凭据文件与不可变执行文件，不因后来重新构建覆盖正在运行进程的身份。

managed 启动顺序为：验证所选配置与端口 → 登记实例 → 构建所选程序 → 准备所需基础设施 → 独立 schema `up`/`verify` → 启动业务进程并检查初始就绪。辅助构建和迁移也登记进程身份与退出结果；运行器异常退出后，停止操作先回收这些使用者，再清理依赖。

seed 是独立的本地操作，不在正常业务启动中隐式创建测试订阅。`seed-service` 输出 `member_id`、`administrator_id` JSON，并保存 `development-fixture.json`。重复 seed 复用持久身份，不重新授予已撤销的 Trader Sync grant。

显式使用已有库：

```bash
make run-service SERVICE=trader-sync INSTANCE=ts-borrower DB_MODE=external ENV_FILE=.env.ts-external
```

文件必须提供 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`；API/Trader Sync 还需显式且一致的内部 token。external 只读验证 schema，不创建、迁移、seed、停止或删除数据库；借用方 reset 被拒绝。允许显式借用另一 managed/full-stack 的库，但原 owner 停库或故障会影响借用方，操作者须协调全部消费者。一个权威库只允许一个活跃 Collector。

## 进程、授权与事务

Trader Sync 内部 gRPC 有 13 个会员方法与 3 个管理员概要方法。API 把已认证账户 UUID 与 realm 转为可信 Actor，同时附带专用服务 token；服务端再次以持久身份、权限和 owner 校验。后台方法也要求服务身份。读调用默认最多 5 秒，写入/目标确认最多 15 秒，并受更早的上游 deadline 限制；不自动重试 mutation。同一 request ID 的相同内容可显式重放原结果，不同内容拒绝。

API 本身的认证失败仍为公共 401；内部 token、Actor 契约或依赖故障变为对应 facade 的 503，避免错误地退出用户登录。Trader Sync 停止不关闭已就绪的 API/Notification；三者同库不可达则分别进入自身失败边界。

运行期写事务先取得 runtime generation 的 `FOR SHARE` guard，再取得账户或业务行锁；接管以 `FOR UPDATE` 更新 generation。已取得 guard 的旧事务可完成，新发起的旧 generation 写入被拒绝。RuntimeSession 持有独占 advisory session，启动先安装 guard 再恢复记录，失权取消全部 Trader Sync 工作者。

同库原子不变量继续保留：API 账户撤权在调用方事务中使订阅和未获许可资格失效，即使 TS 离线也可提交；Trader Sync 活动、资格快照、普通通知入队同事务；Notification 摘要冻结、发送许可与结果由它的独立 pool/受控 adapter 完成，不依赖 TS runtime guard。已获许可尝试可以结束，未知结果不自动重发，重新授权不复活旧资格。RPC 期间不持有数据库事务、pool 借用或进程锁。

## 就绪、停止与恢复

Trader Sync 配置与只读 schema 验证后先开放同一 gRPC server 的 `NOT_SERVING` health，再取得 RuntimeSession、安装 guard、恢复持久状态、启动工作者，最后开放业务准入并发布 `SERVING`。WSS 暂时离线时 RPC 可用、Collector 显示 degraded，创建/恢复订阅保持待基线；不把连接失败等同于整个 API 失败。

单活采集不提供多副本 HA 或零中断滚动升级。重启、provider 切换与断线建立新的可观察实时边界，展示中断，不补查未收到的历史成交。已可靠持久化且仍合格的候选按原 generation/区间继续处理。目录刷新使用两次受 guard 保护的事务；第二事务持目录行锁执行有界 HTTP 并结算，不另开无 guard 的清理事务。

正常 Ctrl+C、TERM 与 `stop-instance` 进入同一停止协议。Trader Sync 总停止预算 30 秒：先停止业务准入、取消和 join 工作，再由 owner 关闭 transport、session 与 pool；超过预算由进程 watchdog 退出。运行器按 PID/PGID、启动时间、boot ID、执行文件、run ID 与 ancestry 核实使用者，以 pidfd 发信号，停止使用者后才处理精确 ID/标签匹配的容器。未知归属或未完成清理保留证据和失败状态，不凭名称抢占端口。

初始就绪失败会回收本次拥有的运行资源并保留持久数据和日志。全部初始就绪后，单个业务进程 fatal 记录退出状态，其他业务继续运行。停止保留 volume、fixture、token 与证据；reset 只针对已停止的 managed 实例，删除其拥有的数据，不能作为跨实例清理工具。运行器不清理全局 `/tmp/coverage` 或其他 checkout 的 scratch。

开发任务的生命周期由执行代理负责，运行器不感知对话任务完成。按 [AGENTS.md](../../../AGENTS.md#本地验收环境准备与完成标准)，任务完成、取消、暂停或以失败/阻塞结束时，代理默认调用对应实例的停止入口，并关闭本任务独立启动的预览和测试替身；上文保留 fixture 指数据和文件，不代表保留辅助进程。用户已有环境、其他任务正在使用的环境和借用基础设施保持原样；只有用户明确要求时才按指定范围保留本任务环境及必要依赖。核对退出并保留数据与证据，具体操作和交付信息见[任务收尾说明](../../developer-guide/running-locally.md#task-shutdown-and-retained-environments)。

Notification 一个进程承载系统、账户和 runtime 三个域，拥有一个 Bot、一个 poller 和一个公平 dispatcher。它只读验证同库 schema，先取得 sender 登记，再构造 Bot/summary 并启动轮询。正常停止先 cancel/join，再保存 graceful stop、恢复未决结果并关闭 sender session/pool。

异常停止后须先从对应 supervisor/container/主机确认**已登记旧 sender 进程确实退出**，再执行：

```bash
athena-notification --recover-stopped-sender=<incarnation-UUID>
```

该命令只恢复指定 incarnation 的 attempt 并退出，不调用 Telegram。无法确认成功的已消费许可转为 unknown；unknown/sent/failed/cancelled 不复活。非首次 sender 启动保留完整 60 秒 monotonic 恢复屏障，未解除的 Retry-After 可能更久。端口空闲或 advisory lock 消失本身不能代替旧进程停止证明。

## 配置边界

| 配置 | 消费者与含义 |
| --- | --- |
| `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` | API、TS、Notification、schema tool 各自打开同一权威库；没有旧变量别名或默认 localhost 回退。managed 运行器注入自己的 DSN。 |
| `ATHENA_TRADER_SYNC_HTTP_URL`、`...WSS_URL` | 仅 TS：Polygon137 同一 provider；开发端点见[来源文档](../../requirements/polymarket-copy-trading/hosted-polygon-rpc-providers.md#已取得的开发候选端点)。切换时成对修改并重启 TS。 |
| `ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY` / `..._FILE` | 仅 TS：稳定游标签名密钥；与服务 token 不同。 |
| `ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN` / `..._FILE` | TS 与 API：至少32字节、不含空白/控制字符。managed 缺省首次生成并持久复用，external 必须显式配置。 |
| `ATHENA_TRADER_SYNC_LISTEN_ADDRESS` / `...SERVER_ADDRESS` | TS 监听与 API 连接地址，独立局部运行默认 `127.0.0.1:8122`。 |
| `ATHENA_TRADER_SYNC_GRPC_TRANSPORT` | 二进制默认 `tls`；本地运行器仅在未设置时显式选 `loopback-insecure`，它拒绝非 loopback 地址。 |
| `ATHENA_TRADER_SYNC_TLS_CERT_FILE`、`...TLS_KEY_FILE` | 仅 TS 服务端证书与私钥。 |
| `ATHENA_TRADER_SYNC_TLS_CA_FILE`、`...TLS_SERVER_NAME` | API/health 客户端的 CA 与证书名称；TLS 不能静默降级。 |
| `ATHENA_TRADER_SYNC_PROXY_URL` | 仅 TS 的 HTTP/WSS、Gamma/Profile、目录请求；明确空串直连。仅本地 WSL 且变量真正未定义时使用 gateway:10809。 |
| `ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES` | TS source job 正整数上限，默认100；不是业务配额或吞吐结论。 |
| `ATHENA_URL` | 可信公开站点地址；用于 API/Notification 链接等。默认开发站点配置为 `http://localhost:4000`。 |
| `ATHENA_NOTIFICATION_*` | Notification 自身 Bot/poller/worker 配置；内部 token 按调用者分配。TS 不读取 Bot 凭据。 |
| `ATHENA_UI_PORT`、`ATHENA_SERVER_PORT`、`ATHENA_NOTIFICATION_PORT` | 运行器所选业务端口；多个实例并存时显式区分。 |

同一 secret 的直接变量和 `_FILE` 互斥，文件只允许末尾一个 LF，不静默裁剪其他空白。配置按进程白名单传递，API 不获得 TS provider/cursor/私钥，Notification 不获得 TS 内部凭据。状态和命令参数不保存秘密；本地秘密文件与日志权限为0600。同内容普通配置文件保留inode并收紧0600，避免重复启动时使Docker Desktop文件挂载失效；内容变化和符号链接仍使用原子替换。

Google OIDC、Phantom、realm、disabled-auth 和头像业务保持原权限规则。公开 origin 与 callback 必须匹配实际站点，不能从 Host 推导。disabled-auth 仅限 loopback，会员/管理员分别选择持久 `local-user`/`local-admin` aggregate；缺失或冲突 realm 不认证。切换正常认证应使用没有开发身份的独立实例，需丢弃旧实例时先停止再显式 reset。Wallet signer capability 与 Worm Trading 私有凭据仍不授予 API 或无关进程。

## 生产与验证证据

独立镜像只构建 TS 与 account-state schema tool，不经过 UI 或聚合 main。生产使用 TLS 和按服务挂载的秘密文件。schema 兼容时只替换 TS；不兼容时，确认维护窗口和全部外部同库使用者停止，再停止三个本栈消费者 → 确认退出 → `up` → `verify` → 启动。迁移失败不启动业务。详见[生产部署说明](../../../deploy/trader-sync/README.md)。

运行器状态展示所选进程、依赖、退出码、日志和实际地址；Trader Sync runtime 另列采集连接、epoch、待基线、积压与中断。`SERVING`、端口监听或 HTTP200 不等同于业务验收通过。独立构建、真实进程、契约、故障/事务、TLS 和浏览器证据以及本次测试操作事故见[独立服务验收记录](../../testing/trader-sync-independent-service-acceptance.md)。
