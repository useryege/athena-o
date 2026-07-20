# BSC Transaction Indexer 独立部署

`athena-bsc-transaction-indexer` 使用独立镜像、独立 Docker Compose 和独立 PostgreSQL 18 数据卷。部署过程不会启动、停止或更新主 ATHENA 服务。

## 前置条件

- 本地安装 Docker、Docker Compose 插件、SSH 和 SCP。
- 目标服务器为 Linux amd64，安装 Docker 与 Docker Compose 插件，并允许部署用户使用 Docker。
- 目标服务器能够访问配置的 BSC Mainnet JSON-RPC 地址。
- 目标服务器的 TCP `8130` 仅对白名单中的主服务器开放。

如需部署到 arm64，可在 Make 命令中传入 `TARGET_ARCH=linux/arm64`。

## 配置

从示例创建本地部署环境文件：

```bash
cp deploy/bsc-transaction-indexer/.env.example .env.bsc-transaction-indexer
```

至少填写：

- `REMOTE_HOST`：目标服务器 IP，也可以在 Make 命令中传入；
- `POSTGRES_PASSWORD`：独立 PostgreSQL 密码；
- `ATHENA_BSC_INBOUND_NODE_RPC_URL`：目标服务器可访问的 BSC 节点 URL。

默认对外提供 `0.0.0.0:8130` gRPC，telemetry 只映射为目标服务器的 `127.0.0.1:8131`。真实环境文件不会提交到 Git，远端副本安装为 `/opt/athena-bsc-transaction-indexer/.env`，权限为 `0600`。

## 一键部署与更新

```bash
make deploy-bsc-transaction-indexer-vps \
  REMOTE_HOST=<目标服务器IP> \
  BSC_INDEXER_ENV_FILE=.env.bsc-transaction-indexer
```

命令会在本地验证 Compose、构建专用 Linux 镜像、检查远端 Docker、上传部署文件、流式传输镜像，并等待 PostgreSQL 和索引器的 liveness healthcheck 通过。

更新代码后重复执行同一命令即可。部署脚本只重新创建容器，不执行 `docker compose down -v`，因此命名卷、扫描游标和已索引交易会保留。

可选覆盖项：

```bash
make deploy-bsc-transaction-indexer-vps \
  REMOTE_HOST=<目标服务器IP> \
  REMOTE_USER=root \
  BSC_INDEXER_REMOTE_APP_DIR=/opt/athena-bsc-transaction-indexer \
  TARGET_ARCH=linux/amd64
```

## 服务检查

主服务使用现有生成的 `BscInboundTransactionServiceClient` 连接：

```text
<目标服务器IP>:8130
```

使用标准 gRPC health probe 检查服务状态：

```bash
grpc_health_probe -addr=<目标服务器IP>:8130
```

查看存活、回填进度和指标：

```bash
ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose exec -T indexer curl -i http://127.0.0.1:8131/healthz'
ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose exec -T indexer curl -i http://127.0.0.1:8131/readyz'
ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose exec -T indexer curl http://127.0.0.1:8131/metrics'
```

初始一个月回填期间，`healthz` 应成功而 `readyz` 可以返回未就绪。gRPC 响应中的 `indexed_through_block` 和 `indexed_through_timestamp` 表示当前可查询的数据边界。

查看容器状态和日志：

```bash
ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env ps'
ssh root@<目标服务器IP> \
  'cd /opt/athena-bsc-transaction-indexer && docker compose --env-file .env logs -f indexer'
```

## 数据备份与恢复

默认数据卷名为 `athena-bsc-transaction-indexer-postgres-data`。可以在本地通过 SSH 创建数据库备份：

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
