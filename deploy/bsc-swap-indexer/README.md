# BSC Swap Indexer 独立部署

> [删除已确认，尚未实施](../../docs/requirements/blockchain-data/bsc-indexer-removal.md)（2026-09-15）。本服务已移出本地与生产目标服务清单；以下保留现有部署方式供后续退役定位，不作为新增部署指引。本轮仅维护文档，未操作远端实例或数据。

`athena-bsc-swap-indexer` 使用独立可执行文件、Docker 镜像、Docker Compose 和 PostgreSQL 18 数据卷，不依赖或重启主 ATHENA 服务。本说明面向专用服务器 `47.254.154.128`；当前代码交付不会自动修改该服务器。

## 服务边界

索引器从 BSC Mainnet finalized 高度向前回填 30 天，并以固定 100 区块范围执行单次 `eth_getLogs`。唯一过滤条件是：

```text
topic0 = 0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822
```

不校验 Router、Factory、Pair 或 token。数据库只保存交易哈希、区块位置、区块时间和顶层 `tx.from`。

## 服务器准备

目标机需要 64 位 Ubuntu、SSH、Docker Engine、Docker Compose v2，并且能够访问配置的 BSC JSON-RPC 节点。首次部署前可执行：

```bash
make install-docker-vps REMOTE_HOST=47.254.154.128
```

安装脚本不管理 Swap、云安全组或防火墙。TCP `8130` 应只允许实际调用服务的来源访问；telemetry 默认仅绑定远端宿主机 `127.0.0.1:8131`，PostgreSQL 不发布宿主机端口。

## 配置

从仓库根目录准备未提交的环境文件：

```bash
cp deploy/bsc-swap-indexer/.env.example .env.bsc-swap-indexer
```

至少确认：

- `REMOTE_HOST=47.254.154.128`；
- `POSTGRES_PASSWORD` 已设置；
- `ATHENA_BSC_SWAP_NODE_RPC_URL` 可从目标服务器容器访问；
- `BSC_SWAP_INDEXER_GRPC_PORT=8130` 和安全组规则一致。

远端部署目录默认为 `/opt/athena-bsc-swap-indexer`，永久数据卷默认为 `athena-bsc-swap-indexer-postgres-data`。

## 构建与部署

只编译独立二进制：

```bash
make athena-bsc-swap-indexer
```

构建独立镜像：

```bash
make bsc-swap-indexer-build-image
```

在明确需要部署时执行：

```bash
make deploy-bsc-swap-indexer-vps \
  REMOTE_HOST=47.254.154.128 \
  BSC_SWAP_INDEXER_ENV_FILE=.env.bsc-swap-indexer
```

部署脚本会在本地构建 Linux amd64 镜像，通过 SSH 上传 Compose 和环境文件，流式传输镜像，并只重建 `athena-bsc-swap-indexer` Compose 项目。重复部署保留 PostgreSQL 命名卷和扫描游标。

## 运行检查

```bash
ssh root@47.254.154.128 \
  'cd /opt/athena-bsc-swap-indexer && docker compose --env-file .env ps'

ssh root@47.254.154.128 \
  'curl -i http://127.0.0.1:8131/healthz'

ssh root@47.254.154.128 \
  'curl -i http://127.0.0.1:8131/readyz'

ssh root@47.254.154.128 \
  'curl -sS http://127.0.0.1:8131/metrics'
```

首次回填期间 `healthz` 成功而 `readyz` 返回 HTTP 503 属于正常状态。不得执行 `docker compose down -v`，除非明确要永久删除索引数据并重新回填。
