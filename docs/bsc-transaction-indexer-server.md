# BSC Transaction Indexer Server (LXM)

> 2026-09-16 现场核验：`47.245.183.140` 上无索引器容器、卷、专属镜像、部署目录或自动启动残留；仅保留 PostgreSQL 基础镜像及仍为 SERVING 的 Etherscan Gateway。本轮未执行删除；历史数据库已随原独占卷不存在，不能将其记作本轮 DROP。详细机器身份与证据见[验收记录](testing/module-removal-cleanup-acceptance.md)。主机和 Gateway 保留。

本文保留 BSC 入账普通交易索引器的历史服务器与部署记录，供后续退役定位资源。
该服务采用独立部署和 gRPC 查询接口；以下状态以各节原核对日期为准，不代表本轮已确认仍在运行或存在主服务调用方。

## 服务器信息

以下信息于 2026-07-21 核对：

| 项目 | 核对时的值 |
| --- | --- |
| 服务器 | LXM |
| 公网 IP | `47.245.183.140` |
| 主机名 | `iZgw8g4vw9m7flvr3ltbkbZ` |
| 系统 | Ubuntu 24.04.2 LTS |
| 架构 | Linux x86-64 |
| CPU | 2 vCPU |
| 内存 | 1.58 GiB |
| 系统盘 | 40 GiB；核对时可用约 31 GiB |
| Swap | 未配置 |
| SSH | `root@47.245.183.140:22` |
| 部署目录 | `/opt/athena-bsc-transaction-indexer` |
| Docker | Engine 29.6.2、Compose 5.3.1 |

硬件和磁盘数字是运维快照，不应作为永久容量保证。PostgreSQL 数据会持续累积，
需要通过 `df -h /var/lib/docker` 定期检查剩余空间。

## 运行服务

Docker Compose 项目运行两个容器：

| 容器 | 镜像 | 网络边界 | 用途 |
| --- | --- | --- | --- |
| `athena-bsc-transaction-indexer-indexer-1` | `athena-bsc-transaction-indexer:local` | `0.0.0.0:8130`、宿主机 `127.0.0.1:8131` | 区块扫描、gRPC、健康状态和指标 |
| `athena-bsc-transaction-indexer-postgres-1` | `postgres:18` | 仅 Compose 内部 `5432` | 保存交易和扫描游标 |

PostgreSQL 永久数据卷为：

```text
athena-bsc-transaction-indexer-postgres-data
```

远端环境文件位于 `/opt/athena-bsc-transaction-indexer/.env`，所有者为
`root:root`，权限为 `0600`。重复部署必须保留该文件和数据卷，不得执行
`docker compose down -v`。

## 服务接口和配置

授权调用方使用以下地址连接索引器：

```text
47.245.183.140:8130
```

该端口提供 `BscInboundTransactionService`，当前只查询成功、finalized、空
calldata 且严格大于 `0.01 BNB` 的顶层入账普通交易。查询可以通过
`before_position` 指定排他的区块号和交易索引截止点；`before_position` 与
`page_token` 不能同时提供。生产索引器不保存内部转账、Trace、BEP-20 转账或
Swap。

当前扫描配置：

```env
ATHENA_BSC_INBOUND_NODE_RPC_URL=ws://88.99.103.60:8546
ATHENA_BSC_INBOUND_SCAN_BATCH_SIZE=200
ATHENA_BSC_INBOUND_FETCH_CONCURRENCY=16
ATHENA_BSC_INBOUND_GRPC_LISTEN_ADDRESS=0.0.0.0:8130
ATHENA_BSC_INBOUND_TELEMETRY_LISTEN_ADDRESS=0.0.0.0:8131
```

容器内 telemetry 监听 `0.0.0.0:8131`，但 Compose 只将其映射到宿主机
`127.0.0.1:8131`，公网无法直接访问。PostgreSQL 没有宿主机端口映射。

gRPC 当前没有 TLS 或 token 验证。TCP `8130` 必须由云安全组或服务器防火墙
限制为仅允许授权调用方访问，不应面向任意公网客户端开放。

## 历史部署和更新（入口已删除，不再执行）

以下为删除前部署记录，当前命令已移除；退役操作使用[清理手册](operator-manual/module-removal-retirement.md)：

```bash
make deploy-bsc-transaction-indexer-vps \
  REMOTE_HOST=47.245.183.140 \
  BSC_INDEXER_ENV_FILE=.env.bsc-transaction-indexer
```

部署脚本在本地构建 Linux amd64 镜像，通过 SSH 传输镜像和配置，再重建专用
Compose 容器。它不会删除 PostgreSQL 命名卷，扫描会从数据库中的已提交游标
继续。

当 gRPC 契约和主 ATHENA Collector 同时更新时，必须先部署索引器并确认新契约
可用，再部署调用 `before_position` 的主服务。旧索引器会忽略新请求字段，不能
满足创建交易之前的严格查询语义。

完整的安装、部署、备份和恢复说明见
BSC Transaction Indexer 独立部署（历史路径 `deploy/bsc-transaction-indexer/README.md`，基线 `264d0dc1`）。
当前运行设计见
[BSC Inbound Normal Transactions](design/blockchain-data/bsc-inbound-normal-transactions.md)。

## 日常检查

查看容器状态：

```bash
ssh root@47.245.183.140 \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env ps'
```

检查存活和 readiness：

```bash
ssh root@47.245.183.140 \
  'curl -i http://127.0.0.1:8131/healthz'

ssh root@47.245.183.140 \
  'curl -i http://127.0.0.1:8131/readyz'
```

`healthz` 表示进程存活。首次回填未追平 finalized 高度时，`readyz` 返回
HTTP 503 是预期行为。

查看实时同步高度、剩余区块和错误次数：

```bash
ssh root@47.245.183.140 \
  'curl -sS http://127.0.0.1:8131/metrics | grep -E "^athena_bsc_inbound_(finalized_block|indexed_block|lag_blocks|scan_failures_total) "'
```

查看索引器日志：

```bash
ssh root@47.245.183.140 \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env logs -f indexer'
```
