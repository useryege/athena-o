# Polygon PoS 自建节点配置与 Hetzner 费用调研

> 调研日期：2026-09-09。
>
> 范围：Polygon PoS 主网普通全节点，为应用提供 HTTP / WebSocket JSON-RPC、链上数据查询和交易广播。
>
> 性质：公开资料与硬件预算参考，尚未选定服务器或部署方案，不构成采购授权或已确认技术设计。
>
> 关联资料：[托管 Polygon RPC 服务与费用调研](requirements/polymarket-copy-trading/hosted-polygon-rpc-providers.md)。

## 结论

采购普通主网 RPC 全节点，可以先按 **16 核 CPU、64–128 GB 内存、约 8 TB 可用本地 NVMe SSD、1 Gbps 网络**估算。这里的存储是实际可用容量，需要扣除 RAID、文件系统和系统使用的空间。

Hetzner AX162-1 配备 48 核 CPU、128 GB ECC 内存和两块 3.84 TB NVMe SSD。按调研时的公开价格，加一个主 IPv4 后，限量款约 **€319/月**，普通款约 **€614/月**，均未含税。这个价位的存储条件依赖将两块磁盘合并使用：RAID0 标称约 7.68 TB，超过官方硬件表的 6 TB 推荐值，但略低于快照指引的 8 TB，不能视为有充足扩容余量的方案。

若需要磁盘镜像与更多容量，应选择更大磁盘配置并另行核价；不能把双盘标称容量直接作为 RAID1 的可用容量。下述费用也不包含归档节点、验证者部署或多节点高可用架构。

## 默认 HTTP 与 WebSocket 端口

应用访问 Polygon PoS 的 EVM JSON-RPC 时，连接的是 Bor 节点。

| 协议 | 默认端口 | 本机连接地址 |
| --- | ---: | --- |
| HTTP JSON-RPC | 8545 | `http://127.0.0.1:8545` |
| WebSocket JSON-RPC | 8546 | `ws://127.0.0.1:8546` |

默认端口不代表服务已经启用。调研时 Bor 的默认配置中，`jsonrpc.http.enabled` 与 `jsonrpc.ws.enabled` 均为 `false`，监听主机为 `localhost`。启用对应服务后才能访问；跨机器连接还需要调整监听地址及相应网络访问配置。实际值以部署使用的 Bor 版本和配置为准。[Polygon 官方端口文档](https://docs.polygon.technology/pos/reference/port-management)、[Bor 默认配置](https://github.com/0xPolygon/bor/blob/develop/docs/cli/default_config.toml)

## 主网全节点硬件要求

| 项目 | Polygon 官方最低配置 | 官方推荐配置 | 本次采购建议 |
| --- | ---: | ---: | --- |
| CPU | 8 核 | 16 核 | 16 核或以上 |
| 内存 | 32 GB | 64 GB | 64–128 GB |
| 存储 | 4 TB | 6 TB | 约 8 TB 可用本地 NVMe SSD，并按实际快照与保留策略复核 |
| 网络 | 1 Gbps | 1 Gbps | 1 Gbps |

最低与推荐配置来自 [Polygon 节点前置要求](https://docs.polygon.technology/pos/how-to/prerequisites)。采购建议是本次调研判断，不是额外的官方最低要求，也不代表某个 RPC 吞吐量承诺。

官方快照指引列出的主网 Bor 与 Heimdall 解压数据合计约 4.5 TB，并建议准备 8 TB 磁盘；这与硬件表推荐的 6 TB 存在口径差异。因此不建议仅按 4 TB 最低值采购。官方快照页也建议使用直连存储降低延迟；本次据此优先考虑本地 NVMe SSD。[快照与磁盘指引](https://docs.polygon.technology/pos/how-to/snapshots)

实际占用取决于客户端版本、选用快照、修剪和历史数据保留配置。快照大小会变化，不能将某次快照体积或近似日增长量当作永久容量保证。下载、解压和维护时所需的临时空间也应在部署前核对。

普通全节点需要 Bor 与 Heimdall，两者可以部署在同一台机器上。本次按一台机器估算。若需要查询任意历史区块的完整账户或合约状态，应另行评估归档模式；不能沿用这份普通全节点预算。[Polygon 节点架构](https://docs.polygon.technology/pos/architecture/overview)

## Hetzner 配置与月租

### 价格口径

- 机房：德国 Falkenstein 或芬兰 Helsinki。
- 币种：欧元；全部金额未含 VAT 或其他税费，未做人民币换算。
- 月租：服务器基础价格加一个主 IPv4；主 IPv4 为 €1.70/月，无开通费。
- 网络：按默认 1 Gbps 档位及其不限流量政策计算，不包括 10 Gbps 升级。
- 月租不包含额外备份、备用节点、应用服务器、数据库、其他 RPC/API 服务及运维成本。
- Hetzner 于 2026-06-15 调整了价格与服务器配置，旧页面和历史文章中的价格不能直接用于当前预算。

来源：[官方价格调整表](https://docs.hetzner.com/general/infrastructure-and-availability/price-adjustment/)、[IPv4 定价](https://docs.hetzner.com/general/infrastructure-and-availability/ipv4-pricing/)、[标准化与调价说明](https://docs.hetzner.com/general/infrastructure-and-availability/faq-standardization-and-price-adjustment/)。

### 已核实价格的候选

| 型号 | CPU / 内存 | NVMe 硬盘 | 基础月租 | 含一个 IPv4 的月租 | 一次性开通费 | 首个完整月预算，含开通费 |
| --- | --- | --- | ---: | ---: | ---: | ---: |
| AX162-1-LTD，限量供应 | AMD EPYC 9454P，48 核 / 128 GB DDR5 ECC reg. | 2 × 3.84 TB，Datacenter Edition | €317.30 | **€319.00** | €39.00 | €358.00 |
| AX162-1，普通款 | 同上 | 同上 | €612.30 | **€614.00** | €304.00 | €918.00 |

首个完整月预算按“服务器月租 + IPv4 月租 + 一次性开通费”计算，不是对实际账单周期或按小时计费结果的承诺。限量款的低价依赖供应，是否可订购及最终金额以当时配置器和结算页为准。[AX 系列硬件配置](https://docs.hetzner.com/robot/dedicated-server/server-lines/ax-server/)、[AX162 配置器](https://www.hetzner.com/dedicated-rootserver/ax162/configurator/)、[官方价格表](https://docs.hetzner.com/general/infrastructure-and-availability/price-adjustment/)

48 核 CPU 高于本次全节点建议。列出 AX162 是因为其公开可核价配置同时提供了较大的本地 NVMe 容量，不表示 Polygon 全节点必须使用 48 核 CPU。

### RAID 对可用容量的影响

以下为标称容量，尚未扣除文件系统和系统占用。

| 硬盘配置 | RAID 方式 | 标称可用容量 | 对本次部署的影响 |
| --- | --- | ---: | --- |
| 2 × 3.84 TB | RAID1 镜像 | 约 3.84 TB | 低于官方快照指引中约 4.5 TB 的解压规模，不适合按该规模部署 |
| 2 × 3.84 TB | RAID0 合并 | 约 7.68 TB | 超过 6 TB 推荐值，接近但低于 8 TB 快照建议；需复核当前快照和预留空间 |
| 2 × 1.92 TB | RAID0 合并 | 约 3.84 TB | 即使合并也不足以覆盖上述快照规模 |

RAID0 没有磁盘冗余，任一磁盘故障都可能导致整个阵列的数据不可用，需要恢复或重新同步节点。因此 €319 / €614 的服务器不能直接理解为同价提供约 8 TB 镜像存储。

### 磁盘更充裕、价格尚未核实的候选

AX102-4 配置为 AMD Ryzen 9 7950X3D、16 核 CPU、128 GB DDR5 ECC 内存，以及 **2 × 1.92 TB + 2 × 7.68 TB NVMe SSD**。

若两组硬盘分别做 RAID1，标称可用容量分别约为 1.92 TB 和 7.68 TB，总计约 9.60 TB；这是两个阵列的容量之和，不代表单个数据目录天然可以使用全部空间。部署时还需要确定系统、Bor 与 Heimdall 的存储分配。

这个配置更适合进一步评估磁盘镜像与容量余量，但本次未核实其动态月租及开通费，因此不提供推算报价。[AX 系列硬件配置](https://docs.hetzner.com/robot/dedicated-server/server-lines/ax-server/)、[AX102 配置器](https://www.hetzner.com/dedicated-rootserver/ax102/configurator/)

## 服务商用途限制与未确定事项

Hetzner 通用条款和独立服务器协议明确禁止加密货币挖矿相关用途。本次查阅的公开条文未明确说明“不质押、不参与出块、仅供应用调用的 Polygon RPC 全节点”是否获准，因此不能直接得出允许或全部禁止的结论。购买前应向 Hetzner 书面确认具体用途；本次没有联系服务商。[通用条款第 8.3 条](https://www.hetzner.com/legal/terms-and-conditions/)、[独立服务器协议](https://www.hetzner.com/legal/dedicated-server/)

以下事项尚未核实或决定：

- 目标 RPC 并发、查询类型和历史数据保留深度，以及相应实际性能。
- 部署使用的 Bor / Heimdall 版本、快照、修剪方式和最终可用磁盘空间。
- 限量型号的实际库存、账户适用税率和结算金额。
- AX102-4 的实际月租与开通费。
- 单机恢复时间、备份方式，以及是否需要第二个节点。
- 具体部署方式是否依赖外部 Ethereum RPC，以及是否产生额外费用；本次没有将自建 Ethereum 节点计入预算。

本次工作仅整理官方公开资料与费用计算，没有购买服务器、部署节点或运行性能验证。
