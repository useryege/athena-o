# 本地与生产服务清单核对

> 核对日期：2026-09-15，工作区 `rf4`。目标成员依据已确认需求与目标设计；现状依据源码、仓库部署配置及本地环境文件，不代表已检查现有服务器的实时健康。
>
> 当前范围：改为[板块访问开关简化方案](business-access-control.md)，原整组运行控制暂停。本文件继续提供进程与环境的静态核对，保留核心分类、六个可控板块与延期接入的 Token、BSC／Sports 删除决定及本地独立进程／存储、使用五个远端 Gateway 的安排；业务默认停止和成员启停要求仅属原方案历史。
>
> 实现状态：访问开关与清单扩展尚未实施，原整组运行控制未实现且已暂停。2026-09-15 静态核对没有操作环境；后续 2026-09-16 删除清理已执行，真实本地验收与逐环境结果见[验收记录](../../testing/module-removal-cleanup-acceptance.md)。

> [本期 make run 配套提案](../../superpowers/specs/2026-09-15-local-full-stack-design.md)已于 2026-09-16 获采用，尚未实施：原五核心／五业务版本已采用；2026-09-16 保留 Worm 后目标覆盖五核心／七业务，共 12 个本地应用，Worm 独立入口、配置、就绪和停止设计已补齐，两服务共用一个访问开关；Token 旧九角色暂不加入，远端 Gateway 单列。下文 16／15 个业务角色仍仅为现状统计。[删除与清理配套设计](../../superpowers/specs/2026-09-16-module-removal-cleanup-design.md)已按确认范围实施，旧实例、共享资源和专属历史数据的实际处理分别见验收记录。

## Token 设计依据与延期范围

**用户决定，2026-09-15**：Token 需要完整重构，目标以已完成的业务文档为依据；当前代码与文档脱离。用户进一步要求，Token 的详细接入设计等其重构完成后再进行。

本轮保留 Token 板块标识及通用访问边界，暂不确定目标进程数量或专属接入与验收。内部任务启停与成员控制不属于本期访问开关。此前的目标职责表从本轮清单移出；下文九个旧角色仅用于记录现状与清理依据，不是未来控制成员。

后续根据[Token 需求与流程](../token/README.md)、[重构设计包](../../superpowers/specs/2026-09-10-token-first-block-design.md)及重构后的实际实现补齐接入清单。本期仅讨论用户访问开关，Token 详细适配继续延期；原设计依据见[历史记录](../../superpowers/specs/2026-09-15-business-group-control-design.md#token-target)。

## 当前代码与部署核对

下表只描述删除清理后保留的当前实现，供差距与清理核对使用。Token 重构和 Trader Sync 手动交易等目标能力不能冒充已存在的进程；现有进程清单也不能反向限定目标设计。

| 业务组 | 当前业务进程 | 本轮核对到的共享或外部依赖 | 当前默认本地全栈 | 当前生产 Compose |
| --- | --- | --- | --- | --- |
| Trader Sync | `athena-trader-sync` | 同环境账户 PostgreSQL；Polygon HTTP／WSS 和 Polymarket Gamma；业务通知由共享 Notification 的对应工作承担 | 已纳入 | 已纳入 |
| Token | Token API、Chain Processor、六类 Collector、Profile Builder，共 9 个 | Token PostgreSQL；按链配置的 EVM 节点与合约；Ave；源码与普通交易 Collector 经 Etherscan Manager 调用 Gateway | 未纳入业务进程，仅有数据库准备 | 9 个均已纳入 |
| Solana | `athena-solana-discovery`，进程内含扫描、补全与查询 | Solana RPC；同一数据库内的 Solana 业务表和账户权限表，查询需要内部鉴权 | 未纳入；有单独 profile，现有采集仍保持暂停 | 未纳入 |
| Market Radar | `athena-market-radar` | Polymarket Gamma；告警使用 Notification；当前业务读模型主要在内存中 | 未纳入 | 已纳入 |
| Managed OO | `athena-managed-oo` | 自有 PostgreSQL；Polygon RPC、Polymarket Gamma；告警使用 Notification | 未纳入业务进程，仅有数据库准备 | 已纳入 |
| Profit Sharing | `athena-profit-sharing` | 自有 PostgreSQL；公共入口通过核心 API 的身份、账户与权限能力 | 已纳入 | 已纳入 |
| Worm Markets | `athena-worm-markets` | 自有 PostgreSQL、Worm API、共享 Notification | 未纳入业务进程，仅有数据库准备 | 已纳入 |
| Worm Trading | `athena-worm-trading` | 自有 PostgreSQL、Worm Markets、Wallet、Worm API 与 Solana RPC | 未纳入业务进程，仅有数据库准备 | 已纳入 |

原[统一运行控制架构](../../superpowers/specs/2026-09-15-business-group-control-design.md)建议的 `athena-runtime-control` 尚未实现且已暂停；新的用户访问开关不新增该进程。

当前保留能力共对应 **16 个业务进程角色**。生产 Compose 包含其中 15 个，缺少 Solana；数字包含待重构的 Token 旧九角色，不是目标部署数量，也不含核心进程、第三方服务、部署工具或副本数。本期访问开关不控制这些后台进程或任务的运行。

### Token 旧实现的九个角色：差距与清理依据

| 成员角色 | 当前命令或配置 |
| --- | --- |
| Token API | `athena-token-api` |
| Chain Processor | `athena-token-chain-processor` |
| Ave Collector | `athena-token-collector --data-type ave` |
| Chain State Collector | `athena-token-collector --data-type chain_state` |
| Wallet Asset State Collector | `athena-token-collector --data-type wallet_asset_state` |
| Simulation Result Collector | `athena-token-collector --data-type simulation_result` |
| Contract Code Source Collector | `athena-token-collector --data-type contract_code_source` |
| Wallet Normal Transactions Collector | `athena-token-collector --data-type wallet_normal_transactions` |
| Profile Builder | `athena-token-profile-builder` |

旧六类 Collector 复用同一个命令入口，以不同参数运行。目标研究运行时已明确不沿用固定六类任务、整项目终态屏障或单个不可变画像，`simulation_result` 不进入新运行模型；这些旧入口需在重构中按目标替换或清理。Etherscan Manager／Gateway 属于共享核心服务。

服务健康仍需按实际能力判断，不能只检查进程存在；健康信息与本期访问开关独立。本期六个可控板块包含 Worm，Token 接入延期。Worm Trading 实际调用 Worm Markets 与核心 Wallet；已纳入运行配套的配置、就绪和停止设计。Worm 两服务共用一个访问设置，服务故障不自动改写该设置。

## 核心与基础设施范围

| 核心能力 | 本地目标及现状依据 | 生产目标及现状依据 |
| --- | --- | --- |
| UI、API Server | 本地 Vite 与 API，各自使用本实例配置 | API 提供已构建 UI 资源，不要求运行 Vite 开发服务器 |
| 登录、账户、权限 | 核心 API 与同环境账户库中的能力，不因功能名称新增进程 | 同一核心能力边界，使用生产自身配置与存储 |
| Wallet、Notification | 当前默认全栈已有本实例进程，持续提供核心能力 | Compose 已有两者的独立进程 |
| Etherscan Manager | 目标由本地运行器启动本地 Manager；当前全栈尚未纳入 | Compose 已有独立 Manager |
| Etherscan Gateway | 已确认使用现有五个远端 Gateway，本地不另起 Gateway；远端进程不属于本地运行器资源 | 继续使用现有五个独立远端 Gateway，保持常开核心能力 |
| PostgreSQL、Redis、MinIO | 沿用按 checkout／INSTANCE 隔离的本地持久资源；业务库按实际成员补齐 | 使用生产自身资源；不把本地数据库或缓存接入生产业务 |
| 账户／业务 schema 工具、MinIO 初始化 | 启动准备任务，完成后退出；不列为常驻业务组 | 保留部署准备职责，不伪装为持续运行的核心服务 |

第三方 Polygon／Solana／EVM RPC、Polymarket、Ave、Etherscan、Nansen、LLM、Telegram 和身份提供方是外部依赖。`make run` 准备本环境进程与必要连接，不启动这些供应商的程序。本地进程和数据隔离也不代表第三方 API 额度自动隔离，具体配置必须如实展示。

<a id="gateway-scope"></a>
## Etherscan Gateway：运行范围已确认

### 已核对事实

- `.env` 与 `.env.prod` 中的 `ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS` 均列出同一组五个地址；`ETHERSCAN_GATEWAY_IPS` 对应相同主机。Manager 的业务转发与管理员 Gateway 检查使用不同配置入口，后续必须一致指向选定环境范围。
- 五个 Gateway 的部署方式与位置见[服务器记录](../../etherscan-gateway-servers.md)：独立主机上的 systemd 服务，当前主生产 Compose 不启动这些进程。
- Manager 只调度请求，不负责启停或部署 Gateway。Gateway 的常开要求不等于本地 `make run` 可以控制远端机器。
- `47.254.154.128` 与 `47.245.183.140` 同时出现在 Gateway 和 BSC 索引器服务器记录中。索引器删除仅清理其所属资源，必须保留同机 Gateway，不能据此停用整台主机或清理共享资源。
- 原静态核对仅检查文件；后续清理任务对两台索引器同机 Gateway 执行了只读健康核验，均为 SERVING，其余 Gateway 不由该结果代替。文件配置可能被运行时环境变量覆盖，不能当作已验证的实际连接状态。

### 已确认安排

**用户决定，2026-09-15**：用户回复“採用”，确认本地启动自己的业务服务、核心服务、数据库和 Etherscan Manager，Gateway 继续使用配置中的五个远端实例；本地启停与清理不操作远端 Gateway，API 额度是否独立取决于密钥配置。

| 项目 | 已确认的运行边界 |
| --- | --- |
| 本地进程与存储 | UI、API、Wallet、Notification、Etherscan Manager 及保留业务（含 Worm 两服务）的进程由本地实例管理；PostgreSQL、Redis、MinIO 与生产隔离；访问首次默认关闭，后续保留管理员设置，重启不重置；后台不随访问开关停止 |
| 生产进程与存储 | 使用生产自身的业务、核心进程与存储，访问配置与本地相互独立 |
| 远端 Gateway | 本地和生产各自的 Manager 连接现有五个远端 Gateway；本地不新增 Gateway 进程 |
| 管理员网关检查 | 本地状态查询和探测指向同一组已选择的远端 Gateway，不把远端进程显示成本地启动的进程 |
| 启停与资源清理 | 远端 Gateway 常开，不提供业务启停按钮；本地 `make run`、停止、重启和清理不部署、停止、重启或清理这些远端实例 |
| API key 与额度 | 由各环境的实际密钥配置决定，不因 Manager 或数据库独立就宣称第三方额度隔离 |

五个远端地址是当前配置池，部署位置见上文核对事实和服务器记录。本地编排扩展实施仍须检查连接、鉴权和配置一致性；该次设计确认没有执行远端检查或修改既有部署；后续删除任务的实际核验单独见验收记录。

## 原整组运行控制的接入差距（历史，当前不推进）

1. 当前 `FullStackServices()` 只选择 Trader Sync、Profit Sharing、Notification、Wallet、UI 和 API Server。补齐保留业务进程及 Manager，并为所有业务实现初始化前的默认停止和常驻管理通道；不能直接启动旧命令后再关闭采集。
2. Solana 需接入统一实例编排与生产部署，明确同环境账户库、业务表、schema 与内部鉴权；现有 profile 的“启动即扫描”不能直接作为目标默认行为。本轮不恢复既有暂停采集。
3. Token 详细接入设计按用户要求推迟到其重构完成后，再核对实际成员、依赖、配置与就绪／收尾证据。本轮保留组级范围与通用接入要求，不固定沿用旧九成员，也不要求先改造旧采集与画像入口。
4. 原时点曾将 BSC、Worm、Sports 全部列为删除；2026-09-16 已修订为仅删除 BSC／Sports，Worm 两服务、数据库、配置和依赖全部保留。删除对象的专属历史数据直接删除；后续实际代码、运行和数据处理已分别记录在验收记录。
5. 本轮在 `cmd`、`internal`、主生产 Compose 和 Procfile 未发现 Temporal 运行消费者，仅本地全栈创建两个 Temporal 数据库。目标清单不据此新增 Temporal 常驻服务；初始化残留和旧数据分别处理。
6. 服务清单、API 客户端地址、数据库准备、权限／业务准入、健康与控制展示必须同步。本地与生产共享分组语义，运行资源和控制状态按环境隔离。

## 源码与设计依据

- [本地全栈](../../../internal/devruntime/fullstack.go)、[服务注册](../../../internal/devruntime/registry.go)、[基础设施准备](../../../internal/devruntime/infrastructure.go)、[现有运行设计](../../design/development-runtime/local-runtime-orchestration.md)。
- [生产 Compose](../../../docker-compose.prod.yml)、[Token Collector 命令](../../../cmd/athena-token-collector/commands/athena-token-collector.go)、[采集与 Profile](../../design/token-intelligence/collection-profile.md)。
- [Trader Sync runtime](../../../internal/tradersync/runtime.go)、[Solana 命令](../../../cmd/athena-solana-discovery/commands/command.go)、[Solana 当前运行说明](../../developer-guide/running-locally.md)。
- [Market Radar](../../design/market-intelligence/market-radar.md)、[Managed OO](../../design/market-intelligence/managed-oo.md)、[Profit Sharing](../../design/governance/profit-sharing.md)、[Etherscan Manager](../../design/blockchain-data/etherscan-manager.md)。

当前证据限于静态源码与配置核对。新访问开关的接入与验证范围以[简化方案](business-access-control.md)为准；原进程控制的默认停止与真实启停验证不纳入本期。
