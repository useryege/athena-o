# BSC Transaction Indexer 独立部署

> [删除已确认，尚未实施](../../docs/requirements/blockchain-data/bsc-indexer-removal.md)（2026-09-15）。本服务已移出本地与生产目标服务清单；以下保留现有部署方式供后续退役定位，不作为新增部署指引。本轮仅维护文档，未操作远端实例或数据。

`athena-bsc-transaction-indexer` 使用独立可执行文件、Docker 镜像、Docker Compose 和 PostgreSQL 18 数据卷。部署、更新和重启该服务不会操作主 ATHENA 服务。

## 远端服务器依赖

远端服务器必须具备：

- 64 位 Linux，默认目标架构为 `linux/amd64`；
- SSH Server 和 Bash；
- `mkdir`、`install`、`rm` 等标准 coreutils；
- 正常运行的 Docker Engine 和 Docker CLI；
- Docker Compose v2 插件，命令形式为 `docker compose`；
- Compose 支持 `up --wait`、`--wait-timeout` 和 `--force-recreate`；
- 能够从 Docker Hub 拉取 `postgres:18`；
- 能够访问配置的 BSC Mainnet JSON-RPC 节点；
- 足够的 Docker 镜像和 PostgreSQL 永久数据存储空间。

远端不需要安装 Go、Git、PostgreSQL 客户端、ATHENA 源代码或私有镜像仓库。索引器镜像由本地构建，然后通过 `docker save | ssh docker load` 传输。

ATHENA 提供面向 Ubuntu root 服务器的一键安装命令：

```bash
make install-docker-vps REMOTE_HOST=<目标服务器IP>
```

该命令通过 SSH 使用 Docker 官方 APT 仓库安装 Engine、CLI、Buildx 和 Compose v2。Docker daemon、Compose 和 `up --wait` 已完整可用时，命令直接成功退出，不重复安装或升级。发现残缺 Docker 安装或冲突包时，命令会停止并报告问题，不会自动卸载或覆盖远端组件。

安装脚本要求目标系统为 Ubuntu，并要求 `REMOTE_USER`（默认 `root`）在远端的 UID 为 `0`；不支持交互式 `sudo`。它不会创建 Swap、修改防火墙或运行 `hello-world`。

也可以按照 Docker 官方说明手动安装：

- [Install Docker Engine](https://docs.docker.com/engine/install/)
- [Install the Docker Compose plugin](https://docs.docker.com/compose/install/linux/)

Ubuntu 或 Debian 在配置好 Docker 官方软件源后，需要安装的包为：

```bash
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

## 用户权限

默认使用 `root@服务器IP` 部署。远端用户必须能够：

- 写入 `/opt/athena-bsc-transaction-indexer`；
- 直接执行 Docker 命令，不依赖交互式 `sudo`；
- 创建 Docker 容器、网络和命名卷；
- 绑定宿主机端口 `8130` 和 `8131`。

如使用非 root 用户，该用户必须加入 `docker` 用户组，并对配置的 `BSC_INDEXER_REMOTE_APP_DIR` 有写权限。部署脚本不会自动调用 `sudo`。

部署前可以执行：

```bash
ssh root@<目标服务器IP> '
  docker --version
  docker compose version
  docker compose up --help | grep -- --wait
  mkdir -p /opt/athena-bsc-transaction-indexer
'
```

## 网络和端口

| 方向 | 地址或端口 | 用途 |
| --- | --- | --- |
| 本地到远端 | TCP 22 | SSH、SCP 和镜像传输 |
| 主服务到索引器 | TCP 8130 | BSC 查询 gRPC |
| 远端本机 | `127.0.0.1:8131` | health、readiness 和 Prometheus metrics |
| 索引器到 BSC 节点 | 配置的 RPC 地址 | 区块和 Receipt 读取 |
| 远端到 Docker Hub | HTTPS | 首次拉取 PostgreSQL 18 镜像 |

PostgreSQL 只存在于 Compose 内部网络，不映射宿主机端口。TCP `8130` 应通过云安全组或 Docker `DOCKER-USER` 防火墙链限制为仅允许主服务器访问。

### BSC 节点运行在同一台宿主机

容器内的 `127.0.0.1` 指向索引器容器自身。因此，如果 BSC 节点运行在索引服务器宿主机上，不能配置：

```env
ATHENA_BSC_INBOUND_NODE_RPC_URL=http://127.0.0.1:8545
```

应使用容器能够访问的宿主机私网地址，例如：

```env
ATHENA_BSC_INBOUND_NODE_RPC_URL=http://10.0.0.12:8545
```

同时需要确保节点 RPC 监听地址和防火墙允许 Docker 网络访问。

## 硬件和存储

部署脚本不强制 CPU 或内存下限。生产环境建议：

- 使用 SSD 或 NVMe；
- 为首次一个月回填预留足够的 CPU、内存和 RPC 容量；
- 持续监控 Docker 数据目录剩余空间；
- 根据永久累积的交易数据扩展磁盘容量。

PostgreSQL 18 数据保存在命名卷：

```text
athena-bsc-transaction-indexer-postgres-data
```

可通过 `BSC_INDEXER_POSTGRES_VOLUME` 修改卷名。卷挂载到 PostgreSQL 18 官方镜像要求的 `/var/lib/postgresql`。

## 配置

在 ATHENA 仓库根目录执行：

```bash
cp deploy/bsc-transaction-indexer/.env.example .env.bsc-transaction-indexer
```

至少填写：

- `REMOTE_HOST`：目标服务器 IP，也可以通过 Make 参数传入；
- `POSTGRES_PASSWORD`：独立 PostgreSQL 密码；
- `ATHENA_BSC_INBOUND_NODE_RPC_URL`：目标服务器内的索引器容器可以访问的 BSC 节点 URL。

默认端口配置：

```env
BSC_INDEXER_GRPC_BIND_ADDRESS=0.0.0.0
BSC_INDEXER_GRPC_PORT=8130
BSC_INDEXER_TELEMETRY_BIND_ADDRESS=127.0.0.1
BSC_INDEXER_TELEMETRY_PORT=8131
```

真实环境文件 `.env.bsc-transaction-indexer` 不提交到 Git。远端副本安装为 `/opt/athena-bsc-transaction-indexer/.env`，文件权限为 `0600`。

## 一键部署和更新

```bash
make deploy-bsc-transaction-indexer-vps \
  REMOTE_HOST=<目标服务器IP> \
  BSC_INDEXER_ENV_FILE=.env.bsc-transaction-indexer
```

部署命令会：

1. 验证本地 Docker、SSH、环境文件和 Compose；
2. 构建独立的 Linux 索引器镜像；
3. 检查远端 Docker Engine 和 Compose；
4. 上传 Compose 和环境文件；
5. 通过 SSH 流式传输索引器镜像；
6. 启动 PostgreSQL 18 和索引器；
7. 等待两个容器的 liveness healthcheck 通过；
8. 输出 gRPC 地址、readiness 和日志命令。

可选覆盖部署参数：

```bash
make deploy-bsc-transaction-indexer-vps \
  REMOTE_HOST=<目标服务器IP> \
  REMOTE_USER=root \
  BSC_INDEXER_REMOTE_APP_DIR=/opt/athena-bsc-transaction-indexer \
  TARGET_ARCH=linux/amd64
```

更新代码后重复执行同一个部署命令即可。部署脚本不会执行 `docker compose down -v`，命名卷、扫描游标和已索引交易会保留。

## 服务检查

主服务通过以下地址建立 gRPC 连接：

```text
<目标服务器IP>:8130
```

使用标准 gRPC health probe 检查：

```bash
grpc_health_probe -addr=<目标服务器IP>:8130
```

查看存活状态、索引 readiness 和 Prometheus 指标：

```bash
ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose exec -T indexer curl -i http://127.0.0.1:8131/healthz'

ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose exec -T indexer curl -i http://127.0.0.1:8131/readyz'

ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose exec -T indexer curl http://127.0.0.1:8131/metrics'
```

初始一个月回填期间，`healthz` 应成功，而 `readyz` 可以返回未就绪。gRPC 响应中的 `indexed_through_block` 和 `indexed_through_timestamp` 表示当前可查询的数据边界。

查看容器状态和日志：

```bash
ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env ps'

ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env logs -f indexer'
```

## 数据备份和恢复

创建 PostgreSQL 备份：

```bash
ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env exec -T postgres sh -c '\''pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB"'\''' \
  > bsc-inbound.sql
```

恢复前先停止索引器，导入备份后再启动：

```bash
ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env stop indexer'

ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env exec -T postgres sh -c '\''psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'\''' \
  < bsc-inbound.sql

ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env start indexer'
```

不要删除命名卷，除非明确要永久清除全部索引数据并重新回填。
