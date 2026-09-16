# 两个 BSC 索引器删除需求

> 需求状态：已确认，2026-09-15。用户明确表示两个索引器已不再需要，要求删除并先维护文档。
>
> 实现状态：删除尚未实施。本轮仅更新文档，源码、构建入口和部署配置仍在；未检查或操作远端运行实例，未处理数据库及数据卷。

## 目标与范围

删除以下两个独立索引服务及其专属配套能力：

| 服务 | 现有能力 | 源码入口 |
| --- | --- | --- |
| `athena-bsc-transaction-indexer` | BSC 入账普通交易扫描、持久化与查询 | [命令](../../../cmd/athena-bsc-transaction-indexer)、[模块](../../../internal/bscinbound) |
| `athena-bsc-swap-indexer` | BSC V2 Swap-topic 交易扫描、持久化与查询 | [命令](../../../cmd/athena-bsc-swap-indexer)、[模块](../../../internal/bscswap) |

两者从本地与生产的目标服务清单中移除，不纳入后续 `make run` 全服务启动或管理员板块访问开关。此前将两者合并为“BSC 链上索引”业务组的建议撤回；无需继续评审其核心／业务分类或启停方式。整组控制需求以 [R18](../development-runtime/business-group-control.md#confirmed) 记录此决定。

## 后续删除实施需覆盖的内容

以下是需求范围与检查依据，尚未执行：

- 两个命令入口、专属扫描和查询模块、服务注册、健康及遥测代码，以及专属测试。
- 两个模块内的 proto、生成客户端、SQL 查询、迁移和生成存储代码；同步清理实际存在的生成配置与调用引用。
- [普通交易部署目录](../../../deploy/bsc-transaction-indexer)、[Swap 部署目录](../../../deploy/bsc-swap-indexer)及各自 Dockerfile、Compose、配套说明。
- [普通交易部署脚本](../../../hack/deploy-bsc-transaction-indexer.sh)和 [Swap 部署脚本](../../../hack/deploy-bsc-swap-indexer.sh)。
- [Makefile](../../../Makefile) 中的 `athena-bsc-transaction-indexer`、`athena-bsc-swap-indexer`、`bsc-transaction-indexer-build-image`、`bsc-swap-indexer-build-image`、`deploy-bsc-transaction-indexer-vps`、`deploy-bsc-swap-indexer-vps`，以及仅供两者使用的 `BSC_INDEXER_*`、`BSC_SWAP_INDEXER_*` 配置。
- 两者专属的环境配置示例、忽略规则、构建／测试引用与其他残留入口。共享依赖需核对实际消费者，只有确实不再使用的内容才随之清理。
- 当前设计、开发、运行说明及跨业务引用，避免仍将两者作为可部署服务或后续开发依赖。历史计划和验收记录保留其发生时的事实。

删除时直接清理专属能力，不为旧入口增加兼容层或替代业务组。服务边界和独立构建运行的验证按[服务开发规范](../../developer-guide/service-development-standards.md)中的适用规则执行。

## 与其他能力的边界

- Etherscan Manager / Gateway 继续属于常开核心服务，不受板块访问开关控制，持续提供共享能力。
- 本次仅删除上述两个独立索引器。Token 的 ETH 首版范围、未来 BSC 接入方向及其他模块使用的 BSC／EVM 通用能力不因本决定取消。
- Token 钱包历史研究与 Trader Sync 后续设计不再依赖这两个索引器。现有 Managed OO、Token 聚合合约和其他服务的功能范围保持各自需求定义。
- [通用 Docker 安装脚本](../../../hack/install-docker-vps.sh)等共享工具不作为索引器专属文件删除。

本轮源码引用核对未发现其他业务的直接 Go 调用方；这一结论仅覆盖所检索的仓库源码，不代表已检查仓库外消费者或远端运行状态。删除决定来自用户确认。

## 既有部署与数据记录

[普通交易服务器记录](../../bsc-transaction-indexer-server.md)与 [Swap 服务器记录](../../bsc-swap-indexer-server.md)保留部署目录、实例及数据库资源位置，供后续退役工作定位。记录中的核对日期属于历史事实，本轮未重新验证。

2026-09-15 的[服务清单核对](../development-runtime/service-inventory-review.md)发现，两台索引器主机也位于当前 Etherscan Gateway 配置池中。后续退役必须只清理索引器所属资源，保留同机 Gateway；本删除决定不包括停用整台主机或清理其共享资源。

部署退役属于后续删除工作的范围，当前状态为待实施；停用实例与历史数据处置应分别记录结果。本轮不执行远端停服、目录清理或数据库／数据卷删除，也不把更新文档或日后删除仓库源码记作远端退役已完成。2026-09-16 用户已确认历史数据直接删除：退役消费者后删除 `bsc_inbound`、`bsc_swap` 数据库及确认专属的数据卷，不设置导出备份或归档保留步骤。详见[删除与清理配套设计](../../superpowers/specs/2026-09-16-module-removal-cleanup-design.md)。配套设计已完成本轮自查，补齐账户 schema、部署产物清理和失败重入，覆盖仓库、旧实例、直接删库及同机 Gateway 保护；尚未实施。

## 后续完成标准

| 检查项 | 预期结果 |
| --- | --- |
| 仓库能力删除 | 两个专属程序、模块、API 和配套构建／部署入口已移除，无悬空调用或生成配置 |
| 启动与控制范围 | 本地与生产目标清单不包含两个索引器，管理员不出现对应访问开关或启停入口 |
| 其他服务 | 实际受影响的构建与验证通过，共享核心能力和其他业务范围保持成立 |
| 文档一致性 | 活跃文档不再引导部署或依赖两个索引器，历史记录能够区分实施前后的事实 |
| 运行资源 | 既有部署退役结果与数据处置状态有独立记录；两个专属数据库／卷按已确认策略直接删除，未完成项如实列明 |

以上为未来实施的验收要求。本轮只进行文档内容、链接与格式检查。

## 关联现状设计

- [BSC 入账普通交易索引现状](../../design/blockchain-data/bsc-inbound-normal-transactions.md)。
- [BSC V2 Swap 索引现状](../../design/blockchain-data/bsc-v2-swap-transactions.md)。
- [业务板块整组启停需求](../development-runtime/business-group-control.md)。

2026-09-16 已按用户优先顺序整理[删除清理实施计划](../../superpowers/plans/2026-09-16-module-removal-cleanup.md)，覆盖仓库清理、本地验收、现场退役与直接删库、最终核验。当前仅完成规划，尚未执行删除。
