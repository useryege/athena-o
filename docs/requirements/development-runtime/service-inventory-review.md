# 本地与生产服务清单核对

> 核对更新：2026-09-17。当前本地十一应用图、局部选择、schema owner 和访问控制已实现并按代码版本 `6b2db2ec8ca5d488a672e68785eea3272af4f50c` 真实验收，见[全栈验收记录](../../testing/full-stack-access-acceptance.md)。生产 Compose 与 Token 旧实现仍按其实际现状记录；本地目标不能反向改写远端部署事实。
>
> 原整组运行控制继续暂停。六个访问开关只限制新用户业务请求，不启停下表应用或后台任务。Token 接入延期；BSC、Sports、Worm Markets 已退役。

## 当前本地十一应用

`FullStackServices()` 是唯一默认图，统一由 `ServiceSpec` 定义 build、配置白名单、schema、监听／消费地址、readiness、核心分类和停止层。局部 `run-service`／`run-services` 只选择请求的应用和最小依赖。

| 分类 | 应用 | 当前本地职责与依赖 | schema owner |
| --- | --- | --- | --- |
| 核心 | `wallet` | Wallet gRPC、加密与 signer；本实例 PostgreSQL | `wallet` |
| 核心 | `notification` | Telegram、poller、恢复与业务通知；账户库 | `account` |
| 核心 | `etherscan-manager` | 调度 Etherscan 请求；连接五个远端 Gateway | 无本地业务库 |
| 核心 | `api-server` | 双 realm、账户／权限、公共 HTTP/gRPC、六板块准入 | `account` |
| 核心 | `ui` | member/admin 两个 HTML/React 入口及 API 代理 | 无数据库 |
| 业务 | `trader-sync` | Polygon/Polymarket 观察、订阅和活动；账户库 | `account` |
| 业务 | `solana-discovery` | finalized 扫描、补全与查询；同一账户库内业务 schema | `account`、`solana-discovery` |
| 业务 | `market-radar` | Polymarket 市场读取与通知 | 无本地业务库 |
| 业务 | `managed-oo` | OO 读取、扫描与通知；独立 PostgreSQL | `managed-oo` |
| 业务 | `profit-sharing` | 轮次、提案、投票；独立 PostgreSQL | `profit-sharing` |
| 业务 | `worm-trading` | 目录、组合、Preview、Run、Cash Out；Wallet、Worm、Solana RPC、账户只读与独立库 | `account`、`worm-trading` |

完整 managed 实例只创建 `athena`、`wallet`、`managed_oo`、`profit_sharing`、`worm_trading` 五个数据库；Solana 与账户共享 `athena`。Market Radar 不新增数据库。当前图不创建 Markets、Token、Temporal、BSC 或 Sports 数据库，也不删除既有残留。

## Token 设计依据与延期范围

**用户决定，2026-09-15**：Token 完整重构后再设计目标服务、运行接入和访问开关。本轮只保留账户的隐藏 Token 模块权限；它不出现在六个 Module Access 设置、会员业务页面或十一应用图中。

旧实现仍包含 Token API、Chain Processor、Ave、Chain State、Wallet Asset State、Simulation Result、Contract Code Source、Wallet Normal Transactions 六类 Collector，以及 Profile Builder，共九个角色；生产 Compose 仍按旧部署事实记录这些角色。本地运行器不创建 Token 服务或数据库，API health 中存在旧 Token 行也不构成十一应用 ready 条件。旧九角色是重构与清理依据，不是当前本地目标。后续顺序和边界见[Token 需求](../token/token.md#旧版归档清理与新版开发)。

## 本地与生产边界

| 项目 | 本地开发 | 生产现状／边界 |
| --- | --- | --- |
| 应用图 | 上述十一应用；Markets、BSC、Sports 不恢复，Token 延期 | Compose 按仓库当前配置；Token 旧九角色与缺少 Solana 的事实不由本地改造自动改变 |
| PostgreSQL／Redis／MinIO | 按 canonical checkout 与 `INSTANCE` 隔离；只准备所选 owner | 使用生产自身资源；本地运行器不连接或清理生产存储 |
| 五个 Etherscan Gateway | 作为外部依赖，只做标准 gRPC Health 读取 | 独立远端常开服务，Manager 不负责其部署和生命周期 |
| 访问控制 | 六键按环境持久化，首次缺行 CLOSED，重启保持 | 使用各自环境存储；本地设置不复制到生产 |
| 网络入口 | 开发服务默认 loopback；公开 UI 经 API 代理 | 已确认外部用户统一经 API、服务器规则隔离业务端口；本轮未远端实测规则 |

第三方 Polygon、Solana、Polymarket、Worm、Etherscan、Telegram 和身份提供方都是外部依赖。实例隔离不表示第三方凭据或额度自动隔离。

## Etherscan Gateway 运行范围

`.env`／环境配置的 Manager 和 API Gateway 集合必须规范化为同一组五个 `IP:6776` 地址。运行器只在选中 Manager 或 API 的相关阶段并行执行每地址 3 秒的 gRPC Health，保存各自结果；Gateway 失败与 Manager readiness 分开。

本地 run、stop、reset 不能创建、部署、停止、重启或清理远端 Gateway，也不执行额度或业务调用。当前真实验收仅证明配置的五项 Health SERVING；不证明所有 Etherscan 业务、远端部署或服务器规则已经验收。部署位置与共享主机边界见[服务器记录](../../etherscan-gateway-servers.md)。

## schema、配置与服务健康

- managed 模式只为所选 owner 建库并在任何消费者启动前执行独立 `up`／`verify`；业务 main 只验证结构。
- external 模式只读 verify，不创建库、版本表、seed、容器或卷，也不停止借用数据库。
- 运行器持久保存内部凭据并拒绝冲突；Trading 地址在所选 API 中自动注入，Profit Sharing 只消费 `ATHENA_PROFIT_SHARING_LISTEN_PORT`。
- 进程 ready、板块 OPEN 和账户权限彼此独立。Trading `SERVING` 不证明供应商或真实交易可用；Solana gRPC ready 不证明扫描已追平。
- Market Radar、Managed OO、Profit Sharing 本轮未新增内部 Bearer、Actor 协议或账户连接池；其外部用户访问依赖统一 API 和已确认的部署网络边界。Solana 与 Trading 既有内部鉴权保留。

## 历史运行控制与退役范围

原 `athena-runtime-control`、组成员启停、运行代次、默认停任务和通知暂停从未实现，当前也不推进。Markets、BSC、Sports 的源码、权限及入口按各自退役记录处理；Worm Trading 保留，`worm` 只表示 Trading。普通 stop/reset 不执行历史永久删库。

Temporal 在本轮目标源码中没有常驻消费者；旧数据库残留不是新增 Temporal 服务的依据。生产 Compose、旧 Token 和历史现场要单独核对，不能用本地十一应用成功替代。

## 源码与验证依据

- [默认图](../../../internal/devruntime/fullstack.go)、[统一注册表](../../../internal/devruntime/registry.go)、[本地编排设计](../../design/development-runtime/local-runtime-orchestration.md)。
- [访问控制需求](business-access-control.md)、[2026-09-17 修订技术方案](../../superpowers/specs/2026-09-17-full-stack-access-reassessment-design.md)。
- [全栈真实验收](../../testing/full-stack-access-acceptance.md)：根路径／前缀十一 ready、局部选择、五库、五 Gateway Health、访问开关、重启和资源归属。
- [Sports 退役](sports-removal.md)、[Worm Markets 退役](worm-markets-removal.md)及其各自验收记录。

本轮证据不包含远端生产部署、服务器规则实测、Gateway 生命周期操作、真实交易或 Token 新版接入。人工最终审查状态不由本清单推断。
