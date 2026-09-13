# Trader Sync 独立构建与本地运行设计

> 状态：已获用户确认，待实现；本文中的新增入口、命令和配置尚未实现。
>
> 已确认输入：沿用[第 1、2 板块](2026-09-13-trader-sync-service-boundaries-design.md)的职责、内部 gRPC、权限、同库事务和单活 runtime；用户本轮确认默认使用开发实例自己的持久数据库，需要时显式连接已有库。

2026-09-13，用户审阅自审后的方案并确认本板块的独立构建、配置、数据库准备、局部编排、资源所有权与部署接入设计。后续沿用这些决定；该确认不代表服务代码已修改或运行验收已完成。

## 1. 本轮范围与方案选择

目标是让开发者只运行 Trader Sync，直接通过内部 gRPC 开发和验证；需要页面、公开 API 或 Telegram 联调时，再明确选择对应服务。`ATHENA_URL=http://localhost:4000` 用于形成业务链接，不能被解释为 UI 必须在线。

| 方案 | 使用方式与代价 | 选择 |
| --- | --- | --- |
| 本机二进制 + 按需容器化基础设施 | Go 进程便于日志和断点调试，编排器准备最小 PostgreSQL，记录资源所有权 | 已确认为本地默认 |
| 全部使用 Compose 容器开发 | 与部署形态接近，需要额外处理源码挂载、镜像重建和调试端口 | 提供服务镜像作部署验证，不作为日常默认 |
| 手动管理数据库并直接运行二进制 | 适合已有数据库、IDE 和外部 supervisor，但开发者需自行准备依赖 | 保留为明确的直接运行方式 |

本轮设计独立入口、配置、迁移准备、局部编排、停止及镜像接入。内部 proto 具体 DTO/字段映射、完整实施任务拆分与验收脚本仍是后续工作；不重开已确认的业务/事务决策，不借本次改造重写全部业务服务。

适用规范：[SDS-R3 独立构建运行](../../developer-guide/service-development-standards.md#sds-r3)、[SDS-R4 配置与故障隔离](../../developer-guide/service-development-standards.md#sds-r4)、[SDS-R5 资源所有权](../../developer-guide/service-development-standards.md#sds-r5)，以及 SDS-R1、R2、R6、R7、R8 在前一份设计中明确的边界和验证要求。

## 2. 独立构建与最小依赖

### 入口

- 新增 `cmd/athena-trader-sync/main.go`，只装配 Trader Sync 命令及其依赖，直接构建 `go build -o dist/athena-trader-sync ./cmd/athena-trader-sync`。不把新命令加入根 `cmd/main.go` 的分发器。
- API、Notification 参与局部联调时，也通过各自目录下的独立 `main.go` 构建；这些薄入口可以调用其现有 command 包。其他无关服务的现有聚合入口不因此全部重写。
- 运行已构建二进制，避免以 `go run` 包装器作为需要监管的业务进程。IDE 可直接启动相同 main；二进制不暗中启动容器、API、迁移器或其他服务。
- 服务镜像使用 `deploy/trader-sync/Dockerfile`，只构建上述 Go 入口，不经过根 Dockerfile 的 UI 和聚合构建阶段。镜像提供业务命令及标准 health 探测子命令。
- 已提交的 proto/sqlc 等生成源码是正常构建输入；只在对应生成源变更时执行生成，不把全仓 `codegen-local`、UI 构建或其他服务构建绑到单服务 build。

独立构建允许已确认的类型、纯逻辑和窄存储适配器依赖：例如 Trader Sync 活动事务调用 Notification 的入队 store，Notification 摘要读取借自身 pool 使用 Trader Sync store。禁止的是导入其他服务的 command/完整 runtime 来借用其进程与生命周期，不把这些必要的同库 adapter 误判为待消除的耦合。

### 依赖图

| 选定服务 | 必要本地依赖 | 不自动加入的服务 |
| --- | --- | --- |
| `trader-sync` | PostgreSQL 中的完整权威 `account-state` schema | API、Notification、UI、Redis、MinIO、其他业务服务 |
| `notification` | 同实例同库；既有 Telegram/站点配置及 Notification 自己声明的依赖 | Trader Sync、API、UI |
| `api-server` | 现有账户库、Redis、头像存储等自身基础设施与配置 | Trader Sync、Notification、Wallet 等远程业务进程 |
| `ui` | 前端工具链及其页面所使用的 API 入口 | 不隐式启动整套业务服务；联调命令显式选择 API |

Trader Sync 声明外部 HTTP/WSS 节点、Profile/Gamma 等已有来源。正常采集依赖它们提供事实；服务不自建 Polygon 节点。临时网络故障沿用重连、资料 unavailable 和中断规则，不把所有外部来源探通作为查询/暂停/取消 RPC 的共同就绪门槛。

API 的“远程客户端配置”与“远程进程必须在线”分开：连接 Trader Sync 使用非阻塞客户端创建，服务离线只使相关调用不可用。局部联调不会为了避免一个未启动服务的连接失败而补启动其他业务进程。

## 3. 开发实例与目标命令

**开发实例**是同一 checkout 下的一组进程、基础设施和持久数据，例如 `trader-sync`、`ts-integration`。实例不是业务服务，也不是生产数据库拆分：同一实例内 API、Trader Sync、Notification 始终连接同一份 `athena` 数据库，各自创建 pool。

下面都是拟实现的命令，不能按当前可用命令执行：

| 命令 | 目标语义 |
| --- | --- |
| `make build-service SERVICE=trader-sync` | 仅构建选定服务及其代码依赖，不启动资源 |
| `make run-service SERVICE=trader-sync` | 默认 `INSTANCE=trader-sync`；准备此实例的持久 PostgreSQL、初始化/校验 schema，前台监管一个 Trader Sync |
| `make run-services SERVICES="api-server trader-sync notification" INSTANCE=ts-integration` | 明确选定进程，计算必要基础设施的并集，同一 schema 只准备一次；UI 需要时显式加入 |
| `make runtime-status INSTANCE=trader-sync` | 显示 checkout、实例、运行批次、进程/资源归属、地址、就绪与退出状态、日志位置 |
| `make stop-instance INSTANCE=trader-sync` | 只停止该实例拥有的进程和受管容器，保留持久数据及日志 |
| `make reset-instance INSTANCE=trader-sync` | 只对已停止实例删除其拥有的持久数据；实例仍在运行或使用外部库时拒绝；不自动重启 |
| `make seed-service SERVICE=trader-sync INSTANCE=trader-sync` | 独立开发工具，显式准备该受管库的最小账户/权限 fixture，输出用于授权 RPC 的账户 ID |

`SERVICE`/`SERVICES` 来自明确的服务注册表；未知项和循环依赖在创建资源前失败。各服务登记构建入口、必要基础设施、配置白名单、健康检查和停止预算。本次保证上述联调服务通过独立入口启动；完整栈的其他已有进程按当前明确清单登记，其构建差距继续标注，不趁本次全部拆改。尚未符合独立入口规则的其他服务不能对外宣称已支持局部运行；未知名称不退回全栈。

默认前台运行并同时保存各进程日志；终端的 `Ctrl+C` 与 `stop-instance` 走同一停止协议。需要持久开发会话时，由终端/IDE/执行工具保存前台 supervisor 会话，不为此增加后台驻留管理 daemon。

同实例只允许一个运行 supervisor。重复启动显示已有归属和状态，不抢占、不替换；改变服务集合需要先显式停止，再用新集合启动，持久库保留，Trader Sync 重启中断可见。本轮不提供热增删进程或不中断切换。

### 外部库模式

`make run-service SERVICE=trader-sync INSTANCE=ts-existing DB_MODE=external` 必须显式提供 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`。该模式不启动、迁移、停止或删除数据库，只连接并校验；已存在同库活跃 runtime 时安全失败，不接管它的本地资源。

外部库首次准备使用独立的 `make account-state-migrate`，显式指定同一 DSN，在该库使用方已按部署/维护流程停止后运行。不能把连接失败、缺表或不兼容解释为允许 reset。外部库模式不运行 seed，不因 `.env` 恰好有 DSN 就自动进入外部模式。

允许显式复用已有全栈开发库；数据库仍由原 owner 管理。外部调用者的 stop/reset 没有该库的清理权限，但原 owner 的停库、重建和故障会影响所有借用者，不能宣称跨实例共享数据库仍有数据/故障隔离。原 owner 的维护需要协调共享使用者；本轮不引入跨实例借用租约或阻止 owner 停库的全局管理器。需要统一启停保障的联调服务应放入同一实例。

默认 `DB_MODE=managed` 下，DSN 由实例拥有的 PostgreSQL 生成并注入，环境文件中的共享 DSN不用于选库；启动摘要明确说明使用的是哪个实例的数据。模式属于实例配置，已有实例不能通过改一个环境变量原地更换数据归属；另建实例或在停止后执行显式重新配置。

### 最小测试身份

seed 工具只允许已核验归属的本地受管实例，不进入生产镜像或业务 RPC。它通过账户存储创建完整 member/admin aggregate，为测试 member 配置 Trader Sync grant；不绑定 Telegram、不创建真实目标订阅、不伪造活动。

fixture 用持久身份标识保证重复调用不重复创建；已有账户的 grant、暂停或撤权测试结果不得被再次 seed 静默重置。内部 gRPC 仍要求服务凭据、可信 Actor 及数据库内授权。隔离验收可以准备自己的 fixture，服务不增加 `skip-auth` 或伪造管理员的模式。

## 4. 配置归属与加载

### 同库定位

将现有 `ATHENA_SERVER_POSTGRES_DSN` 直接更名为 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`，API、Trader Sync、Notification、account-state 迁移器统一读取该名字，删除旧名字的读取和注入，不保留兼容 fallback。它表示共享 schema 的位置，不表示 pool 归 API 所有。

本地同实例由编排器统一注入；部署由同一配置来源注入三进程。不能分别为 Trader Sync 和 Notification 生成不同数据库。account-state 专用 loader 强制 DSN 非空，不能落入当前 `postgres.DSN()` 的 localhost 默认值。更名须同步 `.env` 模板、真实本地配置、Compose、部署脚本、测试和实际读取者。已有 HTTP/WSS、cursor key 和 `ATHENA_URL` 的值继续使用。

### Trader Sync 与 API 客户端

| 配置 | 读取者 | 默认值与校验 |
| --- | --- | --- |
| `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` | 三进程、迁移器 | 必须明确指向目标库；局部受管模式由 runner 注入 |
| `ATHENA_TRADER_SYNC_LISTEN_ADDRESS` | Trader Sync | `127.0.0.1:8122`；合法监听地址；端口冲突失败，不杀占用进程 |
| `ATHENA_TRADER_SYNC_SERVER_ADDRESS` | API | 由编排/部署显式提供可连接地址，不能拿 `0.0.0.0` 当目标 |
| `ATHENA_TRADER_SYNC_HTTP_URL`、`ATHENA_TRADER_SYNC_WSS_URL` | Trader Sync | 必填合法绝对 URL；沿用已选的成对 provider，无自动切换 |
| `ATHENA_TRADER_SYNC_PROXY_URL` | Trader Sync | 空串直连；本地 runner 中未定义时沿用 WSL gateway 默认；二进制/生产未定义时直连 |
| `ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY` / `_FILE` | Trader Sync | 稳定非空 key；不随 run 自动轮换，沿用已生成的开发值 |
| `ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN` / `_FILE` | API、Trader Sync | 独立随机凭据，至少 32 字节，不含空白/控制字符；不得等于 cursor key |
| `ATHENA_URL` | Trader Sync、Notification、API 现有使用者 | 保留现有绝对站点 URL 校验；当前开发为 `http://localhost:4000`，不探测站点是否运行 |
| `ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES` | Trader Sync | 默认 100，正整数 |
| `ATHENA_TRADER_SYNC_GRPC_TRANSPORT` | API、Trader Sync | 默认 `tls`；仅显式 `loopback-insecure` 允许本机明文 |
| `ATHENA_TRADER_SYNC_TLS_CERT_FILE`、`ATHENA_TRADER_SYNC_TLS_KEY_FILE` | Trader Sync | TLS 模式必填且可加载；仅服务端获得私钥 |
| `ATHENA_TRADER_SYNC_TLS_CA_FILE`、`ATHENA_TRADER_SYNC_TLS_SERVER_NAME` | API、独立 health 探测命令 | 校验服务端证书与名称；CA 未提供时用系统信任库，禁止跳过证书校验 |
| `ATHENA_TRADER_SYNC_SHUTDOWN_TIMEOUT` | Trader Sync、runner | 默认 30 秒，正 duration；是总停止预算，不是每个组件各 30 秒 |

内部读/状态 5 秒、解析/写 15 秒继续使用上一板块建议预算，在 API 客户端与服务端分别强制上限及父级取消；本轮作为代码默认常量，不再增加重复的部署环境变量。

`NAME` 与 `NAME_FILE` 是互斥来源，同时设置报配置错误；文件按明确规则去掉一个末尾换行，其他空白按对应凭据规则校验。进程加载一次，凭据或证书轮换需要重启对应进程。Notification 不获得 Trader Sync 的内部 token、TLS 私钥、链节点或 cursor 配置。

本机 runner 默认明确向选定的 API/Trader Sync 注入 `loopback-insecure`，同时校验服务监听与客户端目标均为 loopback；用户显式选择 TLS 时使用其证书配置。数据库是否外部复用不改变 gRPC 的网络边界。直接二进制及生产默认 TLS，跨主机绝不允许明文；TLS 模式的 Bearer credential 要求安全传输，不复用现有无条件明文的 gRPC 工厂。

### 本地加载和故障表现

1. runner 从显式 `ENV_FILE`（默认根 `.env`）读取变量，已导出的 shell 变量优先；保留“未定义”和“已定义为空”的区别。dotenv 解析不执行 shell 命令。
2. 再按选定的受管/外部模式形成实例 DSN、地址等编排输出；禁止外部 DSN被误用于受管资源。WSL 专用代理逻辑移到 Trader Sync 的局部启动处，API 不再执行该 helper。
3. 按进程白名单注入其配置及必要工具环境，不让 Goreman 的全量 dotenv 自动把所有服务凭据交给每个进程。直接运行二进制的操作者负责显式环境注入，业务 loader 仍只读取自身字段。
4. Trader Sync 配置格式错误在分配业务资源前退出；缺外部连接或临时网络不可用按业务来源状态处理。API 不读取或校验 Trader Sync 的 HTTP/WSS/HMAC。
5. API 的 Trader Sync 客户端配置缺失/非法、证书错误或内部认证失败，记录此依赖的配置/连接故障，相关 facade 返回 503；不终止其他 API 模块，也不退回明文。构造失败时安装携带安全错误类别的不可用依赖实现，不能让 command/NewServer 把这个局部错误返回为整个 API 的启动失败。公共用户自身认证错误仍返回原有 401。

本地 runner 给同实例 API/Trader Sync 注入相同的专用内部 token；没有显式 token 时仅在受管开发实例首次创建时生成并持久保存，后续启动复用。生产与外部库模式要求显式配置，不自动生成。直接日志不输出 token；状态记录包含来源和就绪结果，不包含秘密值。

## 5. 数据库准备与 schema 校验

- 增加只导入 `internal/accountstate/store/migrations` 及通用数据库工具的独立 schema 命令，例如 `cmd/athena-account-state-migrate/main.go`，提供 `up` 和只读 `verify`。不导入 `internal/migration` 的全模块注册表，不复制迁移文件。
- 受管实例仅创建 `athena` 库及必要数据库角色，不挂载当前用于全部业务数据库的初始化目录。使用独立命名的持久 volume；PostgreSQL 端口仅绑定 loopback，可由 Docker 分配后读取真实映射。
- 默认保留 PostgreSQL 的正常持久写配置，不复制现有脚本的 `fsync=off` 等性能设置。`stop` 后复用同一 volume，绝不把每次启动变成新测试库。
- 局部 runner 显式执行 schema `up/verify` 后才启动使用该库的业务进程；API、Trader Sync、Notification 的目标入口只连接及 verify，不隐式执行 DDL，也不依赖 API 先运行。全栈与生产启动入口同步增加相同前置步骤。
- 迁移锁使用同库固定的 account-state 锁标识，不能根据 DSN 的 hostname/端口拼锁名；所有调用权威迁移集的路径使用同一标识。通过 localhost 与容器名访问同一库时仍串行。
- verify 同时核对迁移版本和当前 schema 契约（本次 runtime control 及使用到的表、列、约束）；缺失、较新、不完整或不兼容 schema 均明确失败，不自动修表或 reset。现有初始化 SQL原地修改不等于对已有库生效；不兼容的开发库使用新实例或在明确停止后显式重建。
- verify 在只读事务中读取已有迁移记录与 PostgreSQL 系统目录，缺表直接失败；不能调用可能初始化版本表的 Goose status/ensure 路径来冒充只读校验。
- 迁移/验证有独立总预算，默认 120 秒；数据库未就绪重试和锁等待共用该预算。超时保留数据库及日志，不通过启动业务进程掩盖迁移失败。

外部库复用意味着共享故障域和写入真实的开发状态；外部模式只隔离本次进程生命周期，不能声称业务数据也被隔离。运行与 schema 命令应输出目标库的非秘密身份，便于检查配置一致性。

## 6. 生命周期与资源所有权

### 实例记录

实例命名空间由 checkout 的规范化绝对路径和 `INSTANCE` 共同确定；`INSTANCE` 只接受受限字符，不能当任意路径。状态写入 `.run/instances/<instance>/`，资源名包含稳定的 checkout/instance 标识，避免不同 worktree 共用固定 `athena-postgres` 或 `athena-local-postgres-data`。

每次运行另有随机 run ID。状态记录服务集合、二进制路径及构建信息、supervisor/子进程 PID、启动标识、进程组、Docker resource ID/labels、端口、日志路径、owned/borrowed 属性和退出结果。凭据单独保存在受限访问的配置文件中，不混入状态或命令行参数。

所有变更实例状态的命令使用同一短期互斥文件锁，不能在整个前台运行期间占住它而阻止 stop；存活 supervisor 的登记用于拒绝第二次启动。资源创建前先登记意图，创建时写 owner/instance/run labels，随后原子更新记录。这样 supervisor 在创建资源与保存结果之间崩溃，也可通过精确 labels 找回本次资源。

### 启动

1. 检查服务注册表、配置、工具和实例锁；构建选定入口，任何失败均不启动其他业务服务。
2. 核验已有资源的 checkout/instance 所有者与初始化配置；相符则复用，冲突则失败。不能因名称、端口或“看起来是 ATHENA”就接管资源。
3. 准备受管 PostgreSQL并执行 schema 准备；外部模式只做只读 verify。给所有选定的同库使用者注入同一连接位置。
4. 启动 Trader Sync，沿用上一板块的 NOT_READY → 所有权接替提交 → 恢复 → worker 初始化 → SERVING 顺序。第二个同库 runtime 在任何 worker 工作前被拒绝。
5. 输出实例与业务状态。运行成功至少要取得实际服务就绪证据；进程存在或端口监听不足以宣布就绪。WSS 未连接可以是 RPC ready / collector degraded。

数据库容器就绪等待有 120 秒上限；schema 阶段使用其独立 120 秒预算。服务注册表声明各自启动就绪预算，Trader Sync 初始设为 60 秒，从进程启动计时，涵盖接替与恢复；超时停止本次启动并保留证据。这些是可验证的初始控制预算，不是已测得的业务时延承诺。

一次新启动在返回初始成功前失败，回收本批次新启动的进程和受管容器，保留持久 volume、日志及失败状态。已核实存在的借用资源保持原样。

### 停止、运行故障与崩溃恢复

正常停止首先关闭业务准入并报告 NOT_SERVING，在同一总预算内取消请求/worker并 join，最后关闭所有权 session、pool 和 transport。不能提前解锁来让停止“看起来完成”。运行时自身也设置最终退出 watchdog，直接运行二进制同样具有有界停止。

runner 到预算后仅强制结束本次核验身份的进程组；PID 校验结合 OS boot/start 标识、记录的 executable/checkout 和运行批次，不能只依据 PID 或监听端口。随后确认进程退出，才停止它拥有的依赖容器；容器清理由独立的有限预算约束，失败明确留作未清理资源。

多服务联调初始就绪后，Trader Sync fatal 只结束自身并记录失败，不使 supervisor 终止 API、Notification 或共享 PostgreSQL。用户显式 `stop-instance` 才停止整个实例。基础设施只有在所有本次使用者都停止后才回收；共享数据库自身故障仍可能同时影响多个进程。

supervisor/终端异常消失后，`runtime-status`/`stop-instance` 先验证持久记录和实际进程，继续回收可证明拥有的资源。记录过期或身份不符时保留资源并报告，不扩大扫描、杀进程或删除卷。重新启动不能通过删除状态文件或清空业务表解除单活锁；真实旧进程退出后由数据库 session/generation 协议恢复并展示中断。

`reset-instance` 面向整个受管实例的数据，名称中不使用“只重置 Trader Sync”误导共享数据的范围。它要求本实例全部进程已停止、资源归属完整且数据库为受管模式；外部数据库一律不支持 reset。若该库被显式借给其他运行环境，其 owner 必须先协调停止共享使用者；不能以当前没有数据库连接推断没有借用者。保留本轮日志，不调用 `make run-reset`，也不触及其他实例拥有的卷、coverage 或临时目录。

## 7. 全栈、部署与可观测性接入

现有 `make run` 是明确启动完整开发栈的入口；保留其用户用途，在内部显式选择完整进程图并接入独立 Trader Sync。局部命令不能包装 `make run` 再靠 exclude 裁剪。

本次必须同步修改旧全栈清理逻辑：全栈只监管自己的实例，不能按已知端口或同 checkout 的二进制名杀掉另一个局部实例。`make stop`/`make run-reset` 的全栈资源范围继续在长期文档中明确，加入本次所有权保护；不能只保护局部 stop 而让旧 stop 反过来破坏局部运行。

生产 Compose 增加独立 `athena-trader-sync` 服务和 `TRADER_SYNC_IMAGE`，显式监听 `0.0.0.0:8122`，API 使用 `athena-trader-sync:8122`。名称与现有部署脚本的 `athena-` 服务选择规则一致。Trader Sync 不依赖 API/Notification 的启动健康；API 也不以 Trader Sync 必须健康作为整个 API 的启动条件。两者各自依赖已成功完成的权威 schema 准备及数据库。

生产内部链路使用 TLS：Trader Sync 挂载服务端证书/私钥，API 挂载 CA 并校验服务名；专用 token 作为两进程各自的只读 secret 挂载。没有公网端口发布，不注入 UI、钱包 signer 或 Telegram 凭据。默认镜像命令直接启动 Trader Sync。容器 health 子命令连接自身 `127.0.0.1:8122`，仍使用 TLS，挂载同一信任 CA并按证书中的 `athena-trader-sync` 服务名验证；它只加载探测地址/传输配置，查询标准 health，不加载业务凭据或启动业务组件。

独立镜像的构建、检查、save/load 传输及每处 Compose 调用的镜像变量注入纳入现有部署脚本，不能只写一个未被部署消费的 Dockerfile。schema 未变化时可仅停止并替换 Trader Sync。account-state schema 变化时采用维护顺序：停止并确认全部同库使用者退出（至少 API、Trader Sync、Notification）→执行权威 up/verify→启动新版进程；必须调整现有“先迁移再 recreate”的路径。迁移失败保留数据库和日志，不启动未经验证的业务版本。迁移锁只协调迁移器，不能用它或 Trader Sync runtime 锁阻塞 API/Notification 的正常离线事务。

不承诺滚动无中断。Compose 的停止宽限应大于 30 秒业务预算并容纳容器回收，例如初始 40 秒。

标准 health 可不带业务 token，仅返回生命周期状态；业务运行详情仍经内部认证与管理员 Actor 查询。进程日志带 service/instance/run、runtime generation/collector epoch 和阶段信息；状态与日志不以零值填补不可达证据。RPC reflection 不作为默认开放条件，开发调用可使用内部 proto 或生成的 descriptor。

## 8. 实施后必须取得的证据

| 场景 | 必须证明 |
| --- | --- |
| 独立构建 | Trader Sync 的 Go 依赖图和镜像构建不导入 API command/runtime 或无关服务实现，不执行 Node/UI 构建；受控存储适配器按批准边界存在 |
| 最小启动 | 只有目标进程与受管 PostgreSQL；无需 API/Notification/UI/Redis/MinIO，仍能使用 fixture 完成带身份的订阅 RPC |
| 配置隔离 | 清除或故意弄错无关凭据不影响 TS；TS 配置错误不再使 API 退出；内部 token 错误呈 503 且公共 session 保留 |
| 数据与迁移 | run/stop/run 保留订阅；独立实例数据分离；外部模式没有 DDL/建容器；等价 DSN并发迁移仍互斥，旧/缺失 schema 被拒绝 |
| 单活与就绪 | 同库第二实例被拒绝；WSS 断开时查询/暂停/取消仍可用；重启恢复并显示中断，不补遗漏；沿用上一板块 generation 并发测试 |
| 资源所有权 | 两个隔离实例及旧全栈并存、端口冲突、PID 重用、重复 stop、创建中崩溃；不杀其他实例进程/删除其卷；借用方 stop 不操作外部库，原 owner 停库对借用方的已声明影响如实展示 |
| 有界停止与局部故障 | 阻塞 worker/请求达到预算后确认退出，没有提前释放 owner；TS fatal 后 API/Notification 存活；清理失败留下明确证据 |
| 镜像与联调 | 独立容器 TLS/health、生效 secret 注入、API facade、同库撤权、Notification 独立消费；真实验收与隔离测试分别记录 |

本轮只写设计，不启动服务，也不声称这些新入口已经通过运行测试。实现时还须同步[本地运行说明](../../developer-guide/running-locally.md)、[长期本地编排设计](../../design/development-runtime/local-runtime-orchestration.md)、[Trader Sync 长期设计](../../design/trading/trader-sync-activity-alerts.md)、命令文档和部署使用者。

## 当前源码依据

- [聚合入口](../../../cmd/main.go)、[Makefile](../../../Makefile)、[根 Dockerfile](../../../Dockerfile)：当前构建为何仍耦合全服务/UI。
- [局部运行差距](../../design/development-runtime/local-runtime-orchestration.md)、[当前运行脚本](../../../hack/local-runtime.sh)、[PostgreSQL 脚本](../../../hack/start-postgres-with-password.sh)、[Procfile](../../../Procfile)：全栈图、固定资源、持久配置和清理行为。
- [Trader Sync 配置](../../../internal/tradersync/config.go)、[当前 API 组合](../../../internal/server/trader_sync_runtime.go)、[WSL helper](../../../hack/trader-sync-local.sh)：配置及资源归属的迁出点。
- [账户 pool 与开发身份](../../../internal/accountstate/store/sql_store.go)、[Notification pool](../../../internal/notification/store/sql_store.go)、[权威迁移](../../../internal/accountstate/store/migrations/000001_init.sql)：同库、独立 pool 和 fixture 的依据。
- [汇总迁移注册](../../../internal/migration/modules.go)、[PostgreSQL helper](../../../util/db/postgres/postgres.go)：局部迁移构建依赖及当前 DSN 字符串锁标识问题。
- [当前 gRPC 客户端](../../../util/grpc/client.go)、[Compose](../../../docker-compose.prod.yml)、[部署脚本](../../../hack/prod-remote-deploy.sh)：明文工厂、独立镜像与 TLS 注入的改造点。

## 自审记录（2026-09-13）

已对照源码，并分别交叉审阅资源生命周期、构建/配置与部署消费者；这是设计审阅，未运行新增服务。

| 发现 | 自审修订 |
| --- | --- |
| 独立 build 的表述可能误伤已批准的存储适配器 | 明确允许类型、纯逻辑和窄 store 依赖，禁止其他 runtime/command 耦合 |
| 本地默认 TLS 缺证书、容器默认 loopback 无法互联 | runner 显式 loopback 明文，生产 `athena-trader-sync` 使用内部地址和 TLS |
| API 构造客户端失败可能仍退出整个进程 | 明确安装不可用依赖，由 facade 返回局部 503 |
| DSN fallback、按 DSN 地址取迁移锁、伪只读 status | 强制共享 DSN、同库固定锁、只读事务 verify |
| 外部复用与“任何 stop 都无影响”的承诺矛盾 | 明确借用方无清理权、原 owner 保有生命周期及共享故障域，不虚构借用保护 |
| 现有部署先迁移再替换仍会遇到旧业务写入 | schema 变化先停全部同库使用者，迁移成功后启动；业务事务不增加 runtime 锁 |

本板块经上述自审修订后已获用户确认。完整 gRPC 拆分的字段契约、实施计划和实际运行验收仍不能据此标为完成。
