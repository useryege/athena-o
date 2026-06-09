# Athena Makefile 常用命令指南

本文档说明项目根目录 `Makefile` 中保留的常用命令，偏向日常本地运行、代码生成、文档和生产部署速查。

## 前置依赖

常用命令会依赖以下工具：

- Go：用于代码生成，以及生产镜像内部构建二进制。
- Docker：用于构建生产镜像和运行生产 compose。
- yarn：用于手动在 `ui` 目录运行前端开发命令。
- mkdocs：用于本地文档预览和文档构建；也可以通过 Docker 目标运行。

## 命令约定

- Makefile 只保留当前日常使用的入口：本地运行、代码生成、文档、生产部署和数据清理。
- 生产镜像构建通过 Dockerfile 完成，Dockerfile 内部仍会调用 `make athena-all` 构建二进制。
- 所有命令默认在项目根目录执行。

## 常用环境变量

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `PROD_IMAGE` | `athena:local` | 生产部署使用的镜像名。 |
| `PROD_COMPOSE_FILE` | `docker-compose.prod.yml` | 生产 compose 文件路径。 |
| `PROD_ENV_FILE` | `./.env` | 生产部署读取的环境变量文件。 |
| `REMOTE_APP_DIR` | `/root/athena` | 远端服务器上的部署目录。 |
| `REMOTE_USER` | `root` | SSH 登录远端服务器使用的用户。 |
| `PROD_LOG_SERVICE` | 空 | 查看生产日志时指定服务名。为空时查看全部服务。 |
| `PROD_MIGRATE_MODULE` | `all` | 生产迁移目标模块。可设为 `application`、`worm`、`solidity`、`notification`、`wallet`、`polymarket` 或 `all`。 |
| `PROD_POSTGRES_VOLUME` | `athena-prod-postgres-data` | 生产 PostgreSQL external volume 名称。普通部署、停止和重启不得删除该 volume。 |
| `CONFIRM_DESTROY_PROD_DATA` | 空 | 删除生产 PostgreSQL volume 的确认开关。只有精确等于 `yes` 时 `prod-destroy-data-remote` 才会执行。 |
| `PROD_RESET_REMOTE_DATA` | 空 | `prod-remote-deploy.sh` 内部开关。为 `yes` 时部署前停止远端 Athena compose 并删除 PostgreSQL volume。 |
| `PROD_RUN_REMOTE_MIGRATIONS` | 空 | `prod-remote-deploy.sh` 内部开关。为 `yes` 时部署启动后自动执行远端 migration。 |
| `ATHENA_POSTGRES_AUTO_MIGRATE` | 本地默认 `true`，生产 compose 为 `false` | 控制服务启动时是否自动执行 PostgreSQL migration。生产环境通过显式迁移命令控制 schema 演进。 |
| `TARGET_ARCH` | `linux/amd64` | Docker 镜像构建平台。 |
| `ATHENA_POSTGRES_DATA_DIR` | `/tmp/athena-local/postgres` | `clean-postgres-data` 删除的本地 PostgreSQL 数据目录。 |
| `ATHENA_REDIS_DATA_DIR` | `/tmp/athena-local/redis` | `clean-postgres-data` 删除的本地 Redis 数据目录。 |

## 环境与工具

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make install-codegen-tools-local` | 安装代码生成需要的工具。 | `make install-codegen-tools-local` |
| `make password-hash` | 将明文密码转换为 bcrypt hash，用于配置 `.env` 中的 `ATHENA_ACCOUNT_*_PASSWORD_HASH`。 | `make password-hash` |

生成本地账号密码 hash：

```bash
# 交互式输入（推荐，密码不回显）
make password-hash

# 非交互（密码会出现在 shell history，仅适合临时使用）
make password-hash PASSWORD='Yudian#2026!'

# 也可直接运行
go run tools/password-hash/main.go -password 'Yudian#2026!'
```

输出为一行 bcrypt hash（例如 `$2a$10$...`）。写入 `.env` 时请用**单引号**包裹 hash，避免 `$` 被 shell 展开导致登录失败：

```bash
ATHENA_ACCOUNT_LINGJIE_PASSWORD_HASH='$2a$10$...'
```

## 代码生成

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make codegen-local` | 在本机执行完整代码生成流程。 | `make codegen-local` |
| `make protogen` | 先准备 vendor，再生成 protobuf 相关代码。 | `make protogen` |

## 生产构建

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make prod-build-local` | 构建生产部署使用的本地镜像。 | `make prod-build-local` |

常见用法：

```bash
PROD_IMAGE=athena:local make prod-build-local
```

## 本地运行

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make run` | 通过 `hack/goreman-start.sh` 启动，可使用脚本支持的排除参数。 | `make run` |

## UI

UI 相关命令直接在 `ui` 目录执行，例如 `yarn install`、`yarn start`。

## 文档

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make serve-docs-local` | 使用本机 `mkdocs serve` 预览文档。 | `make serve-docs-local` |
| `make build-docs` | 使用 Docker 构建 MkDocs 文档。 | `make build-docs` |

## 生产部署

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make prod-start-local` | 使用生产 compose 在本机启动服务。 | `make prod-start-local` |
| `make prod-stop-local` | 停止本机生产 compose 服务，不删除 volume。 | `make prod-stop-local` |
| `make prod-destroy-local` | 危险操作：停止本机生产 compose，并删除 PostgreSQL volume。 | `make prod-destroy-local` |
| `make prod-logs-local` | 查看本机生产 compose 日志。 | `make prod-logs-local` |
| `make prod-deploy-remote` | 上传 compose、`.env`、PostgreSQL init 脚本和镜像到远端，并启动服务。 | `make prod-deploy-remote` |
| `make prod-deploy-fresh-remote` | 危险操作：本地构建镜像，清空远端 Athena 容器和 PostgreSQL volume，再全新部署并迁移。 | `PROD_ENV_FILE=.env.prod PROD_IMAGE=athena:local make prod-deploy-fresh-remote` |
| `make prod-start-remote` | 在远端执行生产 compose 启动。 | `make prod-start-remote` |
| `make prod-stop-remote` | 在远端停止生产 compose 服务，不删除 volume。 | `make prod-stop-remote` |
| `make prod-logs-remote` | 查看远端生产 compose 日志。 | `make prod-logs-remote` |
| `make prod-migrate-remote` | 在远端手动执行 PostgreSQL migration。默认迁移全部模块。 | `make prod-migrate-remote` |
| `make prod-migration-status-remote` | 查看远端 PostgreSQL migration 状态。默认查看全部模块。 | `make prod-migration-status-remote` |
| `make prod-db-backup-remote` | 在远端通过 `pg_dumpall` 备份 PostgreSQL 到 `$(REMOTE_APP_DIR)/backups`。 | `make prod-db-backup-remote` |
| `make prod-destroy-data-remote` | 危险操作：停止远端 compose 并删除生产 PostgreSQL volume。必须显式确认。 | `make prod-destroy-data-remote CONFIRM_DESTROY_PROD_DATA=yes` |

### 部署前本地预演

部署远端服务器前，可以先用生产镜像和生产 compose 在本机跑一次。`prod-start-local` 已经是本地生产 compose 启动入口，不需要再手写完整的 `docker compose up -d`。

如果使用 `.env.prod` 作为预演环境文件，至少需要包含：

```env
POSTGRES_PASSWORD=your_postgres_password
REDIS_PASSWORD=your_redis_password
ATHENA_JWT_SECRET=your_jwt_secret
ATHENA_WALLET_ENCRYPTION_KEY=your_wallet_encryption_key
```

`ATHENA_WALLET_ENCRYPTION_KEY` 可用以下命令生成：

```bash
openssl rand -hex 32
```

该 key 用于加密 wallet 相关敏感数据。已有 wallet 数据后不要随意更换，否则旧数据可能无法解密。

推荐预演流程：

```bash
PROD_ENV_FILE=.env.prod make prod-build-local
PROD_ENV_FILE=.env.prod make prod-start-local
```

`prod-start-local` 会强制设置 `ATHENA_SERVER_DISABLE_AUTH=false`，即使环境文件中配置为 `true`，本地生产预演仍会启用服务端认证。

生产 compose 中各后端服务设置了 `ATHENA_POSTGRES_AUTO_MIGRATE=false`，因此本地预演和远程部署一样，需要显式执行数据库迁移：

```bash
PROD_ENV_FILE=.env.prod docker compose -f docker-compose.prod.yml --env-file .env.prod --profile tools run --rm athena-migrate athena up --module all
```

查看日志和访问本地服务：

```bash
PROD_ENV_FILE=.env.prod make prod-logs-local
```

```text
http://127.0.0.1:8080
```

停止本地预演：

```bash
PROD_ENV_FILE=.env.prod make prod-stop-local
```

`prod-stop-local` 只会停止并移除 compose 容器，不会删除 PostgreSQL volume。如果需要重置预演数据，请使用 `prod-destroy-local`。

如需一键停止本地生产 compose，并永久删除其 PostgreSQL 数据：

```bash
PROD_ENV_FILE=.env.prod make prod-destroy-local
```

`prod-destroy-local` 会删除 compose 容器、孤立容器、网络和 `PROD_POSTGRES_VOLUME` 指定的 PostgreSQL volume，但保留本地构建的 `PROD_IMAGE` 镜像。该命令可重复执行。

### 远程部署

以下示例使用：

```env
REMOTE_HOST=47.245.181.189
REMOTE_USER=root
REMOTE_APP_DIR=/root/athena
PROD_POSTGRES_VOLUME=athena-prod-postgres-data
```

部署前先确认远端 Docker 和 Docker Compose v2 可用：

```bash
ssh root@47.245.181.189 'docker --version && docker compose version'
```

如果远端提示 `docker: 'compose' is not a docker command`，说明缺少 Docker Compose v2 插件。优先尝试安装系统包：

```bash
ssh root@47.245.181.189 '
set -e
if command -v apt-get >/dev/null 2>&1; then
  apt-get update
  apt-get install -y docker-compose-plugin
elif command -v yum >/dev/null 2>&1; then
  yum install -y docker-compose-plugin
elif command -v dnf >/dev/null 2>&1; then
  dnf install -y docker-compose-plugin
else
  echo "Unsupported package manager; install Docker Compose plugin manually."
  exit 1
fi
docker compose version
'
```

如果系统包不存在，可手动安装 Compose CLI 插件：

```bash
ssh root@47.245.181.189 '
set -e
mkdir -p /usr/local/lib/docker/cli-plugins
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="aarch64" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac
curl -SL "https://github.com/docker/compose/releases/download/v5.1.2/docker-compose-linux-${ARCH}" -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
docker compose version
'
```

推荐远程部署流程：

```bash
PROD_ENV_FILE=.env.prod PROD_IMAGE=athena:local make prod-build-local
PROD_ENV_FILE=.env.prod PROD_IMAGE=athena:local make prod-deploy-remote
PROD_ENV_FILE=./.env.prod make prod-migrate-remote
PROD_ENV_FILE=./.env.prod PROD_LOG_SERVICE=athena-server make prod-logs-remote
```

`prod-deploy-remote` 会把 `$(PROD_ENV_FILE)` 上传到远端并命名为 `.env`，同时上传 `docker-compose.prod.yml`、`hack/postgres/init` 和本地 `$(PROD_IMAGE)` 镜像。

如果需要清空远端 Athena 数据并从零部署，可以使用一键全新部署命令：

```bash
PROD_ENV_FILE=.env.prod PROD_IMAGE=athena:local make prod-deploy-fresh-remote
```

`prod-deploy-fresh-remote` 会先构建本地生产镜像，然后停止远端 `$(REMOTE_APP_DIR)` 下 compose 管理的 Athena 容器、删除 `$(PROD_POSTGRES_VOLUME)` 指向的 PostgreSQL volume、重新上传部署文件和镜像、启动服务，并自动执行 `athena up --module all`。该命令名称本身即表示确认清空 Athena 数据，不需要再传 `CONFIRM_DESTROY_PROD_DATA=yes`。它不会删除远端非 Athena compose 管理的容器或其他 Docker volume。

远端迁移、日志、启停目标会通过 `. $(PROD_ENV_FILE)` 读取环境变量。使用 `.env.prod` 时建议写成 `PROD_ENV_FILE=./.env.prod`，避免 `/bin/sh` 找不到不带 `/` 的 dot 文件。

验证远端服务：

```bash
ssh root@47.245.181.189 'cd /root/athena && PROD_POSTGRES_VOLUME=athena-prod-postgres-data docker compose -f docker-compose.prod.yml --env-file .env ps'
ssh root@47.245.181.189 'curl -sS http://127.0.0.1:8080/api/version'
```

默认生产 compose 将 `athena-server` 绑定到远端 `127.0.0.1:8080`。本机访问时可打开 SSH 隧道：

```bash
ssh -L 8080:127.0.0.1:8080 root@47.245.181.189
```

然后访问：

```text
http://127.0.0.1:8080
```

如果日志出现 `ATHENA_ADMIN_PASSWORD_HASH is not set`，表示服务生成了临时 admin 密码，重启后会变化。生产环境建议配置固定的 `ATHENA_ADMIN_PASSWORD_HASH`。

`prod-deploy-remote` 会执行以下操作：

1. 检查远端 Docker 和 Docker Compose 是否可用。
2. 创建远端部署目录 `$(REMOTE_APP_DIR)`。
3. 确保生产 PostgreSQL external volume 存在，默认名为 `athena-prod-postgres-data`。
4. 通过一次 `tar | ssh` 上传 `docker-compose.prod.yml`、`.env` 和 `hack/postgres/init`。
5. 将本地 `$(PROD_IMAGE)` 镜像传输到远端。
6. 执行 `docker compose up -d` 启动服务。

`prod-deploy-fresh-remote` 在以上流程前会先执行远端清理，并在启动后执行 migration：

1. 如果远端 `$(REMOTE_APP_DIR)/docker-compose.prod.yml` 存在，执行 `docker compose down --remove-orphans`。
2. 删除并重新创建 `$(PROD_POSTGRES_VOLUME)`。
3. 执行 `docker compose --profile tools run --rm athena-migrate athena up --module all`。

`prod-stop-remote` 只执行 `docker compose down`，不会携带 `--volumes`，因此不会删除生产 PostgreSQL 数据。

### 生产数据库安全模型

生产 PostgreSQL 数据保存在 external volume 中，默认名称是：

```text
athena-prod-postgres-data
```

`docker-compose.prod.yml` 中的 `postgres-data` 指向该 external volume。日常部署、停止、重启都不应该删除它。

`hack/postgres/init` 只用于 PostgreSQL 数据目录首次初始化时创建 database，例如 `application`、`wallet`、`solidity` 等。它不负责后续表结构演进。生产环境的 schema 演进由独立迁移命令控制。

生产 compose 中各后端服务设置了：

```env
ATHENA_POSTGRES_AUTO_MIGRATE=false
```

因此生产服务启动时只连接数据库，不自动执行 goose migration。需要修改表结构时，必须手动运行 `prod-migrate-remote` 或底层 `athena-migrate` 命令。

### 推荐生产流程

普通后端代码部署：

```bash
make prod-build-local
make prod-deploy-remote
```

包含数据库结构变更的部署：

```bash
make prod-build-local
make prod-deploy-remote
make prod-db-backup-remote
make prod-migration-status-remote
make prod-migrate-remote
make prod-start-remote
make prod-logs-remote
```

只迁移单个模块：

```bash
make prod-migrate-remote PROD_MIGRATE_MODULE=application
```

查看指定服务日志：

```bash
PROD_LOG_SERVICE=athena-server make prod-logs-remote
```

### 迁移命令底层说明

生产迁移服务通过同一个 `athena` 镜像运行。底层使用 `ATHENA_BINARY_NAME=athena-migrate` 选择迁移命令入口。

支持的命令形式：

```bash
athena up --module all
athena up --module wallet
athena status --module all
```

当前迁移命令覆盖以下模块：

- `application`
- `worm`
- `solidity`
- `notification`
- `wallet`
- `polymarket`

远端 Makefile 目标会通过 compose 工具服务执行迁移，例如：

```bash
docker compose -f docker-compose.prod.yml --env-file .env --profile tools run --rm athena-migrate athena up --module all
```

### 数据清理风险

日常生产部署绝不清库。以下命令都不会删除生产 PostgreSQL volume：

```bash
make prod-deploy-remote
make prod-start-remote
make prod-stop-remote
```

允许删除生产 PostgreSQL volume 的入口有两个：

```bash
make prod-destroy-data-remote CONFIRM_DESTROY_PROD_DATA=yes
PROD_ENV_FILE=.env.prod PROD_IMAGE=athena:local make prod-deploy-fresh-remote
```

`prod-destroy-data-remote` 会在远端停止 compose，并删除 `$(PROD_POSTGRES_VOLUME)` 指向的 Docker volume。`prod-deploy-fresh-remote` 会删除该 volume 后立即全新部署并自动迁移。默认 volume 名为 `athena-prod-postgres-data`。这些都是不可逆的危险操作，执行前必须确认已经完成备份。

## 清理

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make clean-postgres-data` | 删除本地 PostgreSQL 和 Redis 数据目录。 | `make clean-postgres-data` |

注意：`make clean-postgres-data` 会执行 `sudo rm -rf "$(ATHENA_POSTGRES_DATA_DIR)" "$(ATHENA_REDIS_DATA_DIR)"`，默认会删除 `/tmp/athena-local/postgres` 和 `/tmp/athena-local/redis`。
