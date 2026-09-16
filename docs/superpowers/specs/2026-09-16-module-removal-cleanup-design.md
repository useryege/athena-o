# 两个 BSC 索引器与 Sports：删除和清理配套设计

> 最新边界（2026-09-16）：用户另行确认删除 Markets 及其专属数据，采用 [Trading 统一承接目录](2026-09-16-worm-trading-market-query-design.md)。本文件仍只执行 BSC／Sports 原删除范围，不自行扩大删库或通知来源；下文 Worm 双服务保留基线仅适用于内聚重构前。执行时按目标分支实际状态调整保留回归，不能为通过旧检查重建 Markets。两个任务的账户迁移、约束和共享生成产物必须协调。

> 状态：2026-09-16 删除范围与历史数据策略已获用户确认，清理设计已完成本轮自查及缺口补充，尚未实施。Trading 保留，Markets 按独立重构退役；本文件删除两个 BSC 索引器、Sports Live／History 和 World Cup Corners 删除；其专属历史数据直接删除。自查补充与覆盖结论见第 8 节，不把设计完成记作实施完成。
>
> 承接[两个 BSC 索引器删除需求](../../requirements/blockchain-data/bsc-indexer-removal.md)、[Sports 删除与 Worm 保留决定](../../requirements/development-runtime/sports-removal.md)、[访问开关设计](2026-09-15-business-access-control-design.md)与 [make run 配套设计](2026-09-15-local-full-stack-design.md)。原 D01–D03 仍暂停，Token 详细接入仍延期。

> 优先实施计划（2026-09-16）：用户要求先规划清理和删除，已形成[九项任务与逐环境退役顺序](../plans/2026-09-16-module-removal-cleanup.md)。本次仅完成计划，代码、部署和数据操作尚未执行；最新十一应用全栈与访问开关另行实施。

## 1. 当前确认与处理方式

本文件形成时，用户曾要求 Worm 双服务保留。最新决定已改为 Markets 独立退役、Trading 承接目录；本文件的 BSC／Sports 四库与六通知来源范围保持，不把新 Markets 退役混入其中。

删除对象的代码、入口、生成产物与实际消费者同时清理；按环境退役旧进程，然后直接删除确认专属的历史数据，不增加备份／归档保留步骤。资源归属核对用于限定删除对象，不重新询问已经确定的数据策略。

本期访问开关只拦截新用户请求，后台照常运行；永久退役则要求删除对象的进程、采集和通知生产退出目标系统。管理员不再出现被删除业务的开放入口。

## 2. 删除清单与保留边界

| 删除对象 | 专属源码与契约 | 数据归属 |
| --- | --- | --- |
| BSC 普通交易索引器 | `cmd/athena-bsc-transaction-indexer`、`internal/bscinbound`，扫描、查询、proto、存储、测试及部署 | `bsc_inbound` |
| BSC Swap 索引器 | `cmd/athena-bsc-swap-indexer`、`internal/bscswap`，扫描、查询、proto、存储、测试及部署 | `bsc_swap` |
| Sports Live | `cmd/athena-sports-live`、`internal/sportslive`、`internal/server/sportslive`，赛事、价格、告警及页面 | `sports_live` |
| Sports History | `cmd/athena-sports-history`、`internal/sportshistory`、`internal/server/sportshistory`，历史同步、刷新、价格及页面 | `sports_history` |
| World Cup Corners | `internal/server/worldcupcorners`、对应客户端、页面和静态数据 | API 内嵌静态数据，未发现独立数据库 |

共四个独立程序和一个 API 内嵌业务；不代表当前本地或远端已经有相同数量的运行实例。

**保留：**

- Worm Trading 的全部业务与数据，包括 Wallet／账户中的 Worm 签名、凭据再认证、钱包选择、连接、会话、交易授权和 `util/worm`。Markets 在目录承接前仍是现有依赖，其退役与删库按独立设计；本清理不得破坏 Trading，也不得重新引入已删除 Markets。
- API、UI、账户、登录、权限、Wallet、Notification、Etherscan Manager／Gateway。账户、钱包、私钥、头像和资产保留。
- Trader Sync、Token、Solana、Market Radar、Managed OO、Profit Sharing。Token 暂不纳入本期详细接入，不等于删除其代码或数据。
- 其他模块使用的 Polymarket、BSC／EVM、Solana 客户端、市场模型、合约和工具；Worm 的体育市场能力继续保留，不按 `sports` 等关键词整包删除。
- 共享数据库容器、卷、Redis、MinIO、网络和同机 Gateway；资源删除必须精确到已确认所有者。

## 3. 仓库清理契约

### 3.1 服务、API 与权限

1. 删除四个专属命令、服务、后台任务、健康注册、遥测及客户端；清理命令分发、API 构造／关闭、gRPC 与 HTTP gateway 注册，World Cup Corners 一并移除。
2. 旧 HTTP 业务地址进入正常未知 API 处理，旧 gRPC 方法不再注册。Worm 的原始 HTTP 交易、凭据、组合、预览、授权和 Cash Out 接口保留，不得误归为待删除入口。
3. 从账户模型、proto enum、管理员授权、bootstrap、默认授权、SQL 和 schema 契约移除 `sports_live`、`sports_history`、`world_cup_corners` 三类权限。保留其他编号及 Trading 权限；Markets 授权由独立重构清理，协调其迁移最终约束，不复用删除的 proto 编号或名称。
4. 通过现有 schema 管理的一次性清理步骤删除已有账户库中对应三类 `account_module_access` 行并收紧约束，新旧数据库最终结构一致。不重建账户库、不改写其他授权，不额外禁用登录或 API Key；派生状态仍按剩余权限计算。
5. Wallet 通用功能和 Worm 专属签名／授权均保留；不新增空字段、隐藏兼容入口或假服务来承接 Sports。

**账户 schema 的具体处理：**在现有 Goose 序列追加下一未使用版本的清理迁移，同一事务中删除三类失效授权并更新 `account_module_access_module_check` 和 `account_module_access_max_level_check`；其他表、约束及保留模块权限不改写。已应用的历史迁移及版本记录保留，不通过删掉旧版本文件或重建账户库实现清理。新库运行完整迁移后与已有库升级后的最终结构一致；旧迁移中的 Sports 名称属于历史结构记录，不是仍提供业务的入口。

共享账户库启动校验同时检查版本集合和实际 catalog，因此实施必须同步生成 `internal/accountstate/schema/contract.json`：先完成迁移／查询 SQL 和 `sqlc.yaml`，执行对应 sqlc 生成，再用 `make account-state-schema-contract` 从工具自建测试库生成契约；最后构建所有受影响使用者。现场沿现有不兼容 schema 的维护流程，在同库使用者退出后执行 `up` → `verify` → 新版本启动，不能一边用旧权限写入路径一边改约束。维护失败不绕过校验、不回退到会重新提供 Sports 的旧版本。

这沿用[账户 schema 权威](../../../internal/accountstate/schema/schema.go)、[契约生成工具](../../../tools/account-state-schema-contract/main.go)和[已有部署维护流程](../../../deploy/trader-sync/README.md)。核心服务在正常 schema 发布维护中按原流程更新；“核心保留”与“没有管理员停服开关”不代表可以跳过该维护流程。实际同库使用者按环境核对，不能只照搬旧脚本的三个名字遗漏 Solana 或其他已接入进程。

依据：[账户权限模型](../../../internal/accountaccess/access.go)、[账户协议](../../../internal/server/account/account.proto)、[账户 SQL](../../../internal/accountstate/store/queries/account_access.sql)、[账户目录 SQL](../../../internal/accountstate/store/queries/account_directory.sql)、[Wallet 协议](../../../internal/wallet/wallet.proto)与 [API 入口](../../../internal/server/athena-server.go)。

### 3.2 页面、生成代码与测试

- 删除 `/sports-live`、`/sports-history`、`/world-cup-corners` 的导航、路由、懒加载、页面、专属模型和请求代码。全部 `/worm-trading/*` 页面及其依赖保留。
- 管理员权限、业务状态、帮助与共享账户展示中只移除 Sports 三项；其余 UI 沿已确认主题和组件维护。
- 删除专属 proto、客户端、SQL、迁移和生成配置／产物。共享 `market_intelligence_types.go` 仅移除没有保留消费者的 Sports 类型，Worm 模型保留。
- 同步 `sqlc.yaml` 与 `hack/generate-proto.sh` 的删除对象入口，重新生成受影响的共享代码、deepcopy、Swagger／API 文档；Worm 专属 Swagger 后处理继续保留。不得只删产物、留下生成入口恢复旧功能。
- 删除仅验证废弃功能的用例和工具；混合用例保留 Worm 与其他业务断言，补齐入口消失、保留授权和核心回归测试。依赖库按实际剩余引用整理。

依据：[会员路由](../../../ui/src/app/member/app.tsx)、[页面加载](../../../ui/src/app/member/routes.tsx)、[共享类型](../../../pkg/apis/application/v1alpha1/market_intelligence_types.go)、[sqlc 配置](../../../sqlc.yaml)与 [proto 生成脚本](../../../hack/generate-proto.sh)。实施时沿 [sync-athena-changes](../../../.codex/skills/sync-athena-changes/SKILL.md) 的实际依赖链完成生成和消费者调整。

### 3.3 启动、部署与配置

- 清理两个索引器的 `deploy/` 目录、部署脚本、Make 构建／镜像／部署目标，及专属 `BSC_INDEXER_*`、`BSC_SWAP_INDEXER_*`、`ATHENA_BSC_INBOUND_*`、`ATHENA_BSC_SWAP_*` 配置。
- 从生产 Compose 移除 Sports 两服务、相关 `depends_on`、API 地址、迁移环境与模块目标；从本地运行器移除对应注册、预检、配置、数据库准备和状态项。Worm 两服务及全部必要配置保留。
- 清理初始化 SQL、模块迁移注册及 `.env`／`.env.prod`／模板中的 Sports 专属入口和配置；缺少废弃服务的地址、DSN 和 token 时，保留服务仍应能构建、初始化和启动。
- 最新 `make run` 目标为十一个本地应用，Worm 只保留 Trading；目录承接、独立构建、权限读取与配置见最新内聚设计，访问设计保留 `worm` 标识和 Trading 完整入口。上述均尚未实施；执行本清理按实际分支验证依赖，不重建 Markets 或宣称开关已受控。
- 旧 Token／Temporal 的本地准备缩减按运行配套设计处理，不扩张成本次全仓代码和数据删除。普通启动／停止仍保留数据；永久删除使用独立的一次性退役步骤。

### 3.4 已部署文件与构建残留

- 在记录资源归属并完成停止后，删除明确专属的旧二进制、Compose／启动文件、环境文件、部署临时包及空部署目录；不得先删用于识别和停止资源的文件。仅保留定位、操作日志和验收证据，不把可重新启动旧业务的部署副本当作归档长期保留。
- 两个索引器专属镜像仅在确认没有保留容器引用后移除。Sports 当前与 Worm 等服务共用 `PROD_IMAGE`，不能将该镜像当作 Sports 专属资源删除；用一致的新版本替换，旧共享镜像按原镜像管理规则处理。PostgreSQL 基础镜像及共享网络、缓存不属于专属清理范围。
- 混合 `.env`、部署目录、日志与构建缓存只清理明确属于被删业务的项。保留 Gateway、Worm 和其他服务文件，不执行目录通配删除、Docker prune 或全仓缓存清空。

依据：[Makefile](../../../Makefile)、[生产 Compose](../../../docker-compose.prod.yml)、[生产部署预检](../../../hack/prod-remote-deploy.sh)、[本地全栈](../../../internal/devruntime/fullstack.go)、[初始化 SQL](../../../hack/postgres/init/00-databases.sql)与 [模块 schema 注册](../../../internal/migration/modules.go)。

## 4. 历史数据：直接删除，已确认

消费者退出、残留连接消除、资源归属核对后，直接删除以下专属数据库及确认专属的数据卷／存储。**不导出备份，不为这些业务保留离线历史副本。**

| 数据库 | 删除内容 |
| --- | --- |
| `bsc_inbound` | 普通交易索引、扫描进度及专属历史记录 |
| `bsc_swap` | Swap 索引、扫描进度及专属历史记录 |
| `sports_live` | 赛事、价格、告警状态及专属历史记录 |
| `sports_history` | 历史赛事、价格、同步进度及专属记录 |

实际库名和挂载以各目标环境核对为准。共享 PostgreSQL 中只删除明确的库，不删除共享容器或整个卷。`worm_markets`、`worm_trading`、账户库、Wallet 库及其他业务库不在删除清单内。

World Cup Corners 的静态数据随源码删除。账户库三类失效授权按第 3.1 节清理；共享 Notification 账本不属于本次四个专属历史库，仍按第 5 节保留真实发送记录。外部账户、凭据、订单、持仓和钱包资产不自动撤销或处置。

实施记录只保留定位信息、对象名称、执行时间、结果、日志和验证证据；这些退役证据不作为保留业务历史数据库的替代安排。正常 `make run`／`make stop` 不执行永久删除。

## 5. 停止旧工作与 Sports 旧通知

先切断待删除的用户业务入口，再停止各环境全部对应生产者和后台进程，移除旧部署清单、重启策略或定时任务中的自动拉起来源。Trading 继续按原规则运行，其交易与数据保留；Markets 的生产者、五个通知来源和专属数据由独立退役流程处理。

Sports Live 已入队消息不会随生产者退出自动消失。只处理以下六个精确来源，执行前再核对实际生产者与记录，不使用 `polymarket.*` 等宽泛匹配：

- `polymarket.sports-live-score`。
- `polymarket.sports-live-price-alert-85-15`、`polymarket.sports-live-price-alert-90-10`、`polymarket.sports-live-price-alert-95-5`、`polymarket.sports-live-price-alert-97-3`、`polymarket.sports-live-price-alert-99-1`。

一次性处理规则：

1. 所有 Sports 生产者退出后，在 Notification 自有存储流程中，以行锁／条件更新将对应 `pending` 消息改为现有终态 `cancelled`，记录退役原因。沿用发送许可的同一行锁边界；业务清理通过一次性维护操作完成，不在每次 Notification 启动时扫描或清空历史队列。
2. 已取得许可的 `sending` 等待现有有界发送结果；已在途消息仍可能送达。变为可重试 `pending` 的继续取消；`sent`、`failed`、`unknown` 和发送尝试保留真实事实，不盲目重试未知发送。
3. 有界复查至旧来源没有可发送 `pending` 或在途 `sending`；超时或旧发送者归属不明时记录未完成项，按现有恢复规则处理，不伪造取消或成功。
4. Notification 保留为核心服务，除既有 schema 发布维护外按原规则运行；Worm、Trader Sync 和其他来源、用户绑定、Bot 偏移、Topic 与已送达外部消息保留。恢复服务时遵守既有 sender 恢复屏障；清理完成前仍可能发生已经获许可的发送或重试，不能以一次 pending 更新宣称旧来源已完全收尾。

依据：[Sports 比分](../../../internal/sportslive/score_alerts.go)、[Sports 价格](../../../internal/sportslive/price_alerts.go)、[发送许可事务](../../../internal/notification/store/attempts.go)和[发送 SQL](../../../internal/notification/store/queries/delivery_attempts.sql)。这不改变日常“关闭访问后通知照常运行”的规则，也不新增常驻控制服务。

## 6. 旧部署退役与数据删除顺序

### 6.1 索引器资源定位

下表来自历史部署记录与 Compose 默认值，**本轮未连接远端核实**；执行时以实际 project、容器标签、挂载及配置覆盖值确认。

| 对象 | 历史主机／目录 | 默认专属卷 |
| --- | --- | --- |
| 普通交易索引器 | `47.245.183.140`，`/opt/athena-bsc-transaction-indexer` | `athena-bsc-transaction-indexer-postgres-data` |
| Swap 索引器 | `47.254.154.128`，`/opt/athena-bsc-swap-indexer` | `athena-bsc-swap-indexer-postgres-data` |

两台主机都在已记录的 Gateway 配置池内。停止索引器项目及确认专属的 PostgreSQL 容器，再删除专属卷；保留同机 Gateway、主机和共享资源。禁止全机 Docker 清理、磁盘清空或按端口批量杀进程。

### 6.2 各环境执行顺序

1. 准备已验证的代码、生成产物、镜像和 schema 工具，记录 checkout／版本、实例或 Compose project、进程／容器 ID、数据库、挂载、日志和自动启动来源。归属不明对象不进入删除集合，记录待核对原因。
2. 进入既有发布流程，先撤下旧用户业务入口及旧权限写入版本，停止 Sports／索引器的专属生产者；保留用于识别和停止资源的运行器与清单。逐项消除旧容器、进程及自动拉起来源，不退役 Worm 或核心服务。
3. 账户库结构变更按第 3.1 节的维护流程执行 `up` → `verify`，然后启动配置、权限、接口与页面一致的新版本及保留使用者；按第 5 节取消 Sports 旧队列并核对发送收尾，验证旧入口不能提供业务。此处是正常部署维护，不新增管理员停服按钮。
4. 新版本保留能力与共享依赖验证通过后，核对目标库没有消费者或残留连接，执行第 4 节直接删除，并清理第 3.4 节的专属部署残留。不得先删共享数据再排查归属，不使用全局 `run-reset`。
5. 验证旧进程退出、所属监听释放、后台写入停止、四个专属库／卷及专属部署入口不再存在，Worm、核心能力及同机 Gateway 正常。共享 API 仍监听时按业务路由归属核对；资源不存在的结论来自实际查询，不仅来自删除命令退出码。

仓库删除、运行退役和数据删除分别记录。不能将源码删除、停服成功或文档更新写成数据已清空。详见[普通交易服务器记录](../../bsc-transaction-indexer-server.md)、[Swap 服务器记录](../../bsc-swap-indexer-server.md)及[服务清单](../../requirements/development-runtime/service-inventory-review.md)。

### 6.3 一次性操作、失败与重入

使用一次性维护工具／操作清单，不建立后台清理服务或新增管理员页面。每个环境的清单记录资源 owner、对象类型、精确名称／标识、删除前状态、处理结果和核验依据；先只读列出待处理对象，再对同一清单执行，已有范围确认不重复变成新的业务审批。

- 同一环境一次只执行一份清理；执行前复核数据库身份、容器 ID／标签及卷挂载，若与清单不符则停止该环境的后续破坏性步骤，不能按相似名称猜测资源。
- 所有步骤有有限 deadline；数据库锁与 SQL 操作设超时。发现残留连接先定位并按归属停止所属使用者，不使用强制踢掉未知连接的方式掩盖遗漏。独占索引器卷只在其所有使用者退出后删除，共享 PostgreSQL 仅执行精确删库。
- 账户清理迁移内部原子提交；不同数据库、Docker 资源与主机之间不承诺一次事务。失败或中断时保留已成功结果，停止依赖失败步骤的后续删除，输出非零结果及未完成对象。未完成不能写成全环境退役完成。
- 再次执行从实际状态核对开始；同一已识别资源确认不存在时记作已完成，有条件的权限／队列清理可以安全重入。新出现或标识改变的对象重新核对归属，不直接沿旧名称删除。
- 四库数据已经删除后不自动重建、不恢复旧服务，不承诺无备份的数据回滚。修复保留服务继续采用一致的新版本；本次删除设计不增加旧业务兼容路径。

## 7. 文档与验收

活跃文档移除被删业务的使用指引；保留历史设计、视觉批准和验收记录的事实，并注明 Sports 退役及 Worm 保留。失去源码目标的链接改为历史路径说明或可定位的版本链接。

| 验收范围 | 必需证据 |
| --- | --- |
| 仓库及生成 | 四个程序和世界杯角球无构建／注册／生成入口，无悬空调用；Worm 源码和消费者保留 |
| API／页面／授权 | Sports 旧入口不可用，三类权限清理；Worm 页面、业务授权、签名和连接能力回归通过 |
| 启动和配置 | 无废弃配置／数据库时保留服务可启动；本地空实例和已有数据实例验证；Worm 补入目标运行清单 |
| 旧任务和部署 | 所有对应生产者退出、自动拉起消除、Sports 旧队列不可再发送；Worm 与共享服务正常 |
| 历史数据 | 四个精确专属数据库及确认专属卷已删除，无备份／归档保留步骤；Worm、账户、钱包及共享存储保留 |
| 核心与外部资源 | 登录、账户、Wallet、Notification、Etherscan、受影响其他业务及同机 Gateway 有实际验证证据 |
| 失败与重入 | 错误归属、残留连接、SQL／删除失败、部分成功后中断和再次执行均有明确结果；不会扩大删除集合、重建已删数据或误删共享镜像 |

实施时按影响完成 Go 构建／测试、对应 SQL 与 API 生成、前端类型／构建和真实页面验收；部署和数据删除另取现场证据。遵守[服务开发规范](../../developer-guide/service-development-standards.md) SDS-R1 至 R8 的适用边界，保留账户与通知同库事务，不新增服务或跨库协调协议。

## 8. 本轮设计自查结论

2026-09-16 用户再次确认当前删除内容并要求检查完整性。本次对照账户迁移与严格 schema 校验、通知许可／结果事务、Worm 的体育市场依赖、生产共享镜像和索引器部署文件检查，补齐了三项原先不够具体的安排：

| 自查发现 | 文档补齐位置 |
| --- | --- |
| 仅写“清理权限”不足以保证已有账户库可以升级，新旧 schema 版本／catalog 必须一致 | 第 3.1 节追加迁移、契约生成、既有维护顺序和失败不绕过校验 |
| 只移除服务定义与数据库会遗留可启动文件或误删共享镜像 | 第 3.4 节专属文件／镜像清理与共享产物边界，第 6.2 节执行先后 |
| 跨库／主机删除可能部分成功，需要中断后继续且不能宣称可回滚 | 第 6.3 节精确清单、超时、失败退出、实际状态核验与安全重入 |

**清理设计覆盖已补齐，没有需要用户再决定的删除范围或历史数据策略。**具体环境的容器 ID、数据库覆盖名称、连接占用和维护时间属于实施前现场核对；不能用本次静态自查替代现场验证。

最新启动与访问接入由配套设计负责：十一程序目标、Trading 配置／鉴权／就绪／停止、其 `worm` 开关和完整用户入口；Markets 的目录承接与退役另见最新内聚设计。三份配套设计均尚未实施。清理实施的前提是保留 Worm 的现有能力通过回归，不要求先实现新增访问开关；若与新全栈／访问控制合并交付，则必须完成对应新增范围，不能沿用原五板块／十程序的完整性结论。

**当前实际状态：**仅完成设计自查、文档补充及静态检查；未删除源码、启动环境、连接远端、执行生成器或操作数据。清理设计、代码实施、现场退役与整套启动／访问设计的完成状态分别记录。
