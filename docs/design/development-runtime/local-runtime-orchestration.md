# 本地运行编排

> 设计状态：十一应用统一 `ServiceSpec`、按选择准备、分阶段就绪、持久证据和有界停止已实现。代码版本 `6b2db2ec8ca5d488a672e68785eea3272af4f50c` 的根路径和 `/athena` 前缀真实运行均为十一项 ready；最终证据与限制见[全栈验收记录](../../testing/full-stack-access-acceptance.md)。

## 范围与服务边界

`make run` 的当前全栈包含五个核心应用和六个业务应用：Wallet、Notification、Etherscan Manager、API Server、UI；Trader Sync、Solana Discovery、Market Radar、Managed OO、Profit Sharing、Worm Trading。Token、Temporal、Markets、BSC 和 Sports 不在当前全栈，也不因历史数据库或生产 Compose 配置被隐式加入。

运行器也支持局部正向选择。`run-service`／`run-services` 只构建、配置、准备和启动显式选择的应用及其最小基础设施，不补齐全栈成员。每个服务保持独立 main、配置白名单、监听地址、健康探针、schema owner 和停止层；API 通过公开契约与业务服务交互，不把业务实现重新聚合进 `cmd/main.go`。该边界落实 [SDS-R1、R3、R4、R5](../../developer-guide/service-development-standards.md#sds-r1)。

## 源码入口

| 职责 | 源码 |
| --- | --- |
| Make 入口与保存运行器 | [Makefile](../../../Makefile)、[run-local-runtime.sh](../../../hack/run-local-runtime.sh)、[saved-local-runtime.py](../../../hack/saved-local-runtime.py) |
| 十一应用、配置、schema 与停止层 | [registry.go](../../../internal/devruntime/registry.go)、[fullstack.go](../../../internal/devruntime/fullstack.go) |
| 准备、启动、就绪、状态与停止 | [internal/devruntime](../../../internal/devruntime) |
| 账户 schema owner | [athena-account-state-migrate](../../../cmd/athena-account-state-migrate) |

## 命令与实例归属

命令必须从目标 checkout 根目录执行，并在 run/status/stop 中使用同一 `INSTANCE`、`DB_MODE` 和需要的 `ENV_FILE`：

```bash
make run INSTANCE=full-stack
make runtime-status INSTANCE=full-stack
make stop INSTANCE=full-stack

make run-service SERVICE=solana-discovery INSTANCE=solana-discovery
make run-services SERVICES='trader-sync api-server ui' INSTANCE=ts-integration
make stop-instance INSTANCE=ts-integration
```

`make solana-discovery-run`／`make solana-discovery-stop` 是上述统一 owner 的别名，默认 `INSTANCE=solana-discovery`。旧 `hack/solana-local.sh` profile 只由旧 owner 停止；不能用新 owner 接管旧状态，也不能用旧 stop 停新实例。旧现场如需停止，使用原脚本及原实例名。

状态保存在 `.run/instances/<instance>/`。owner 绑定 canonical checkout、instance、namespace、进程身份、不可变保存程序摘要和容器标签。stop/status 优先使用已保存运行器，即使源码损坏或当前缺少 Go 工具也能核查或停止；保存程序摘要、路径或归属不符时拒绝执行。停止保留数据库卷、凭据、日志和状态；`reset-instance` 只允许对已停止且明确要丢弃数据的本实例执行。

## 最小依赖与 schema owner

managed 模式按选择创建最小基础设施和业务库。当前全栈只创建五个 PostgreSQL 数据库：账户／Solana 共用 `athena`，以及 `wallet`、`managed_oo`、`profit_sharing`、`worm_trading`。Market Radar 无新数据库；Token、Temporal、Markets、BSC 和 Sports 数据库不创建，也不删除历史残留。

所有消费者启动前完成所选 schema 的 `up` 与 `verify`。账户 authority 先 `up`／`verify`，Solana 在同一明确账户 DSN 上准备 `solana_discovery` schema；Wallet、Managed OO、Profit Sharing 和 Worm Trading 各自使用独立 owner。业务进程只打开并验证结构，不在启动时迁移。每个 schema 操作预算 120 秒，单个服务构建预算 10 分钟。

`DB_MODE=external` 只对显式所选 owner 执行只读 verify，不创建数据库、版本表、seed、自有容器或卷，也不停止借用基础设施。运行器保存无凭据端点与配置指纹，并拒绝同实例悄然切换目标。共享账户库是故障域，不等于共享进程内连接池或跨 RPC 事务。

## 配置与网络边界

所有业务监听默认显式绑定 loopback。导出的环境变量覆盖 `ENV_FILE`，包括显式空值；每个进程只接收注册表白名单。内部 token、cursor HMAC、Wallet signer 和数据库加密凭据按用途分离并在实例内持久复用，显式值与保存值冲突时失败。

注册表把所选应用地址注入其消费者。Trader Sync 的监听地址包含端口；Market Radar、Managed OO、Profit Sharing 使用各自 `*_LISTEN_PORT`，其中 Profit Sharing 不消费通用 `*_PORT`。Trading 与 API 同选时自动注入 Trading 地址。API 的直接 HTTP 探测由 `ATHENA_SERVER_ROOTPATH` 控制；公开 UI HTML、资源、回调和代理路径由 `ATHENA_SERVER_BASEHREF` 控制，Vite 剥离部署前缀后代理到 API。不能把直接 API 地址错误地统一加 BaseHRef。

Etherscan Manager 连接配置中的五个远端 Gateway。运行器仅执行标准 gRPC Health 只读探测并逐地址保存结果，不创建、部署、停止或修改远端 Gateway；Gateway 健康与 Manager 进程就绪相互独立。

## 分阶段就绪与失败模型

启动顺序为核心依赖（Wallet、Notification、Manager）→ API → UI → 六业务。同层可并行，但遵守依赖。`CoreUsable`、`SelectedReady` 和 `FullStackReady` 分开记录；只有完整十一项全部 ready 才有 `FullStackReady=true`。`runtime-status` 命令成功只表示状态可读，不表示全栈 ready。

API 就绪检查真实 health、member/admin bootstrap、realm、会话和角色。UI 检查两入口、部署根、真实脚本／样式／预载资源及代理 bootstrap；HTML 200 或返回 SPA HTML 的资源 200 不足以 ready。普通应用 ready 预算 60 秒、UI 120 秒、Notification 180 秒；托管基础设施 5 分钟、总启动 30 分钟，launcher 外层 300 秒并留 5 秒强制结束尾限。

核心、构建、共享准备或启动总期限失败会清理本轮资源并非零退出。业务初始失败记录失败、停止该业务，保留可用核心并继续其他业务。运行期进程即使以 code 0 意外退出，也撤销 ready 并进入 degraded。Stage 历史、探针、退出码、失败和 cleanup 结果持久保存；后续成功探测不抹掉原失败。

仅当 API 被选择时，启动摘要才用 admin realm 只读获取 Module Access 设置；认证启用但没有管理员会话时报告不可读，不伪造 CLOSED，也不改变设置。局部选择未包含 API 时不额外引入 API 依赖。

## 停止与恢复

实例 stop 的总预算为 180 秒。停止层为 UI/API → 六业务 → Wallet/Notification/Manager → 自有基础设施；同层一起发 TERM，helper 和 supervisor 共享同一持久 `StopDeadline`，不能各自重新计时。每服务预算仍有效，保留总预算的一小部分用于 KILL 后确认退出。

每次发信号前后重新验证 PID、start ticks、boot、run、exe、pidfd、祖先关系和容器标签。不能证明归属的迟生孤儿会记录明确失败并保留证据，不按模糊 PID/PGID 猜测清理。停止后核对记录进程退出、端口释放和自有容器停止；卷、日志和报告保留，外部数据库与远端 Gateway 不动。

## 验证证据与限制

实际验收包含根路径与 `/athena` 前缀各一次十一应用 ready、双 realm Chrome、94／92 次真实 API 请求、实际 GUI 开关、两次重启持久化、Solana 与 Notification 后台连续性、局部选择和失败隔离，以及最终资源归属审计；详见[验收记录](../../testing/full-stack-access-acceptance.md)。五 Gateway 只做健康读取；部署服务器规则未远端实测，三项业务服务内部鉴权未新增，Token 接入延期。测试通过和本地资源停止不等于人工最终审查通过。
