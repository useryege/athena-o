# BSC Swap Indexer Server (47.254.154.128)

本文记录当前独立运行 BSC V2 Swap 交易索引器的服务器。该实例与主 ATHENA
服务和 BSC 入账普通交易索引器分开部署，其他服务通过 gRPC 查询已经完成索引的
钱包 Swap 交易哈希。

## 服务器信息

以下信息于 2026-07-22 核对：

| 项目 | 当前值 |
| --- | --- |
| 服务器 | SHL |
| 公网 IP | `47.254.154.128` |
| 主机名 | `iZgw883p7pkcbe7f09pofeZ` |
| 系统 | Ubuntu 24.04.2 LTS |
| 架构 | Linux x86-64 |
| CPU | 2 vCPU |
| 内存 | 1.58 GiB |
| 系统盘 | 40 GiB；核对时可用约 28 GiB |
| Swap | 未配置 |
| SSH | `root@47.254.154.128:22` |
| 部署目录 | `/opt/athena-bsc-swap-indexer` |
| Docker | Engine 29.6.2、Compose 5.3.1 |

硬件、磁盘和运行指标都是运维快照，不应作为永久容量保证。PostgreSQL 会持续
保存新发现的 Swap 交易且没有历史清理机制，必须定期检查系统盘剩余空间。

## 索引数据语义

索引器只使用以下日志 Topic0 识别交易：

```text
0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822
```

服务首次启动回填最近 30 天，并持续跟随 BSC Mainnet finalized 高度。每轮使用
一个 `eth_getLogs` 请求过滤最多 100 个连续区块。只要交易包含目标 Topic0 日志
就会进入数据集；服务不验证 Router、Factory、Pair、Token、合约字节码或日志
内容，因此数据可以包含其他 V2 协议或发出相同 Topic0 的合约。

同一交易中的多个匹配日志按交易哈希去重。数据库保存交易位置、区块时间和顶层
`tx.from` 钱包地址，不保存 Pair、Token、金额或交易路径。

## 运行服务

Docker Compose 项目运行两个容器：

| 容器 | 镜像 | 网络边界 | 用途 |
| --- | --- | --- | --- |
| `athena-bsc-swap-indexer-indexer-1` | `athena-bsc-swap-indexer:local` | `0.0.0.0:8130`、宿主机 `127.0.0.1:8131` | 区块扫描、gRPC、健康状态和指标 |
| `athena-bsc-swap-indexer-postgres-1` | `postgres:18` | 仅 Compose 内部 `5432` | 保存 Swap 交易和扫描游标 |

PostgreSQL 永久数据卷为：

```text
athena-bsc-swap-indexer-postgres-data
```

远端环境文件位于 `/opt/athena-bsc-swap-indexer/.env`，所有者为
`root:root`，权限为 `0600`。重复部署必须保留该文件和数据卷，不得执行
`docker compose down -v`。

## 服务接口和安全边界

授权调用方使用以下地址连接索引器：

```text
47.254.154.128:8130
```

该端口提供 `BscSwapTransactionService.ListSwapTransactions`。首次请求必须提供
钱包地址和排他的 `before_block_number`；后续请求使用绑定钱包和区块位置的
`page_token`。响应返回交易哈希、下一页令牌和当前已索引边界。

gRPC 当前没有 TLS 或应用层认证。阿里云安全组中的 TCP `8130` 必须只允许实际
调用服务的来源访问，不应向任意公网客户端开放。

容器内 telemetry 监听 `0.0.0.0:8131`，但 Compose 只将其映射到宿主机
`127.0.0.1:8131`。阿里云安全组虽然已经配置 `8131` 放行，公网仍无法直接访问
该端口；`/healthz`、`/readyz` 和 `/metrics` 必须通过 SSH 在目标服务器本机
调用。PostgreSQL 没有宿主机端口映射。

## 当前运行状态

2026-07-22 核对时，Indexer 和 PostgreSQL 均为 `healthy`，连续运行期间没有
容器重启或扫描失败。`healthz` 返回 HTTP 200；首次 30 天回填尚未追平
finalized 高度，因此 `readyz` 返回 HTTP 503，属于预期状态。

核对时 PostgreSQL 数据库约 4.98 GB，Docker 数据卷约 5.7 GB。回填完成后数据
仍会持续增长；40 GiB 系统盘只适合作为当前容量，运维必须持续观察数据库体积
和剩余磁盘，提前安排扩容或容量告警。

## 部署和更新

从 ATHENA 仓库根目录执行：

```bash
make deploy-bsc-swap-indexer-vps \
  REMOTE_HOST=47.254.154.128 \
  BSC_SWAP_INDEXER_ENV_FILE=.env.bsc-swap-indexer
```

部署脚本在本地构建 Linux amd64 镜像，通过 SSH 传输镜像和配置，再重建专用
Compose 容器。它不会删除 PostgreSQL 命名卷，扫描会从数据库中的已提交游标
继续。

完整部署说明见
[BSC Swap Indexer 独立部署](../deploy/bsc-swap-indexer/README.md)。当前运行设计见
[BSC V2 Swap Transactions](design/blockchain-data/bsc-v2-swap-transactions.md)。

## 日常检查

查看容器状态：

```bash
ssh root@47.254.154.128 \
  'cd /opt/athena-bsc-swap-indexer && docker compose --env-file .env ps'
```

从允许访问 `8130` 的主机检查标准 gRPC Health：

```bash
grpc_health_probe -addr=47.254.154.128:8130
```

通过 SSH 检查存活和 readiness：

```bash
ssh root@47.254.154.128 \
  'curl -i http://127.0.0.1:8131/healthz'

ssh root@47.254.154.128 \
  'curl -i http://127.0.0.1:8131/readyz'
```

`healthz` 表示进程存活。首次回填未追平 finalized 高度时，`readyz` 返回
HTTP 503 是预期行为。

查看同步高度、剩余区块和扫描失败次数：

```bash
ssh root@47.254.154.128 \
  'curl -sS http://127.0.0.1:8131/metrics | grep -E "^athena_bsc_swap_(finalized_block|indexed_block|lag_blocks|scan_failures_total) "'
```

检查数据库和磁盘容量：

```bash
ssh root@47.254.154.128 \
  'cd /opt/athena-bsc-swap-indexer && docker compose --env-file .env exec -T postgres psql -U athena -d bsc_swap -Atc "SELECT pg_size_pretty(pg_database_size(current_database()));" && df -h /'
```

查看索引器日志：

```bash
ssh root@47.254.154.128 \
  'cd /opt/athena-bsc-swap-indexer && docker compose --env-file .env logs -f indexer'
```
