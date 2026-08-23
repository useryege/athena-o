# Athena Makefile 操作手册

本文档说明项目根目录 `Makefile` 中保留的常用命令，偏向日常本地运行、代码生成和生产部署速查。

## 前置依赖

常用命令会依赖以下工具：

- Go：用于代码生成，以及生产镜像内部构建二进制。
- Docker：用于构建生产镜像和运行生产 compose。
- yarn：用于手动在 `ui` 目录运行前端开发命令。

## 命令约定

- Makefile 只保留当前日常使用的入口：本地运行、代码生成、生产部署和数据清理。
- 生产镜像构建通过 Dockerfile 完成，Dockerfile 内部仍会调用 `make athena-all` 构建二进制。
- 所有命令默认在项目根目录执行。

## 常用环境变量

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `PROD_IMAGE` | `athena:local` | 生产部署使用的镜像名。 |
| `PROD_COMPOSE_FILE` | `docker-compose.prod.yml` | 生产 compose 文件路径。 |
| `PROD_ENV_FILE` | `.env.prod` | 生产部署读取的环境变量文件。 |
| `REMOTE_APP_DIR` | `/root/athena` | 远端服务器上的部署目录。 |
| `REMOTE_USER` | `root` | SSH 登录远端服务器使用的用户。 |
| `PROD_LOG_SERVICE` | 空 | 查看生产日志时指定服务名。为空时查看全部服务。 |
| `PROD_MIGRATE_MODULE` | `all` | 迁移目标模块。可设为 `account-access`、`worm-markets`、`fifa-market-dashboard`、`notification`、`wallet`、`sports-live`、`sports-history`、`managed-oo`、`profit-sharing`、`token` 或 `all`。 |
| `PROD_POSTGRES_VOLUME` | `athena-prod-postgres-data` | PostgreSQL external volume 名称。本地停止、远程部署和远程删除都会删除该 volume。 |
| `ATHENA_POSTGRES_AUTO_MIGRATE` | 本地默认 `true`，生产 compose 为 `false` | 控制服务启动时是否自动执行 PostgreSQL migration。生产部署脚本会在启动业务服务前显式迁移。 |
| `TARGET_ARCH` | `linux/amd64` | Docker 镜像构建平台。 |

## 环境与工具

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make install-codegen-tools-local` | 安装代码生成需要的工具。 | `make install-codegen-tools-local` |
| `make password-hash` | 将明文密码转换为 bcrypt hash，用于配置 `.env` 中的 `ATHENA_ACCOUNT_*_PASSWORD_HASH`。 | `make password-hash` |
| `make jwt-secret` | 生成可用于 `ATHENA_JWT_SECRET` 的 HS256 随机签名密钥。 | `make jwt-secret` |
| `make service-password` | 生成可用于 `POSTGRES_PASSWORD` / `REDIS_PASSWORD` 的随机密码。 | `make service-password` |
| `make wallet-private-key-ciphertext` | 生成可用于 `wallet_private_keys.private_key_ciphertext` 的密文 SQL 表达式。 | `make wallet-private-key-ciphertext` |

生成本地账号密码 hash：

```bash
# 交互式输入（推荐，密码不回显）
make password-hash

# 非交互（密码会出现在 shell history，仅适合临时使用）
make password-hash PASSWORD='temporary-password'

# 也可直接运行
go run tools/password-hash/main.go -password 'temporary-password'
```

输出为一行 bcrypt hash（例如 `$2a$10$...`）。写入 `.env` 时请用**单引号**包裹 hash，避免 `$` 被 shell 展开导致登录失败：

```bash
ATHENA_ACCOUNT_LINGJIE_PASSWORD_HASH='$2a$10$...'
```

生成 HS256 JWT secret：

```bash
# 默认生成 base64 编码的 32 字节随机密钥
make jwt-secret

# 也可直接运行
go run tools/jwt-secret/main.go

# 如需 hex 格式
go run tools/jwt-secret/main.go -format hex
```

输出为一行 secret。写入 `.env` 或 `.env.prod`：

```bash
ATHENA_JWT_SECRET='<generated-secret>'
```

生成数据库和 Redis 密码：

```bash
# 默认生成 32 位字母数字密码
make service-password

# 也可直接运行
go run tools/service-password/main.go

# 如需更长密码
go run tools/service-password/main.go -length 48
```

输出为一行仅包含大小写字母和数字的密码，可直接写入 `.env` 或 `.env.prod`：

```bash
POSTGRES_PASSWORD='<generated-password>'
REDIS_PASSWORD='<generated-password>'
```

生成钱包私钥密文：

```bash
# 交互式输入私钥（推荐，私钥不回显），默认读取 ATHENA_WALLET_ENCRYPTION_KEY
make wallet-private-key-ciphertext

# 只输出 hex，方便手动拼接到 decode('<hex>', 'hex')
make wallet-private-key-ciphertext ARGS="-format hex"

# 也可直接运行
ATHENA_WALLET_ENCRYPTION_KEY='<wallet-encryption-key>' go run tools/wallet-private-key-ciphertext/main.go
```

默认输出为一段可直接写入 SQL 的 BYTEA 表达式：

```sql
decode('<generated-ciphertext-hex>', 'hex')
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
| `make run` | 前台启动本地服务，创建可重建的 PostgreSQL/Redis 容器并复用持久化数据。支持 `ATHENA_RUN_EXCLUDE`。 | `make run` |
| `make stop` | 优雅停止本地服务并删除容器和运行控制状态，保留 PostgreSQL/Redis 数据 volume。 | `make stop` |
| `make run-reset` | 先停止服务，再删除本地容器、数据 volume、运行控制状态和默认临时运行数据。不会重新启动。 | `make run-reset` |

本地 PostgreSQL 和 Redis 数据分别保存在固定命名 volume
`athena-local-postgres-data` 和 `athena-local-redis-data`。前台按
`Ctrl+C` 与从另一终端执行 `make stop` 具有相同的浅层清理语义，后续
`make run` 会创建新容器并挂载原数据。PostgreSQL volume 会记录镜像、
用户、初始数据库、密码和初始化 SQL 的配置指纹；这些初始化设置发生变化
后必须执行 `make run-reset`，避免以新配置静默打开不兼容的旧数据。
Profit Sharing 新增独立的 `profit_sharing` 数据库和 `8108` 端口；首次使用
包含该数据库的初始化配置时同样必须执行 `make run-reset`。本地 Procfile
默认启用 API Server 认证，五个参与账号必须通过登录后才能提交方案或投票。

`make run-reset` 还会清理默认的 `/tmp/athena-local`、各 Athena 服务的
`/tmp/coverage/athena-*` 目录和 `/tmp/coverage/api-server`。通过环境变量
指定到其他位置的自定义临时目录不会被自动删除。

## UI

UI 相关命令直接在 `ui` 目录执行，例如 `yarn install`、`yarn start`。

## 文档

文档直接以仓库内 Markdown 维护。开发前先阅读 `docs/design/README.md` 和相关子系统设计文档；设计级代码变更需要在同一任务中同步更新 Living Design Docs。

## 生产部署

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make prod-start-local` | 创建本地 PostgreSQL volume、执行 migration 并启动生产 compose 服务。 | `make prod-start-local` |
| `make prod-stop-local` | 停止本机生产 compose 服务并删除 PostgreSQL volume。 | `make prod-stop-local` |
| `make prod-logs-local` | 查看本机生产 compose 日志。 | `make prod-logs-local` |
| `make prod-reset-secrets` | 更新生产 env 中的 PostgreSQL、Redis 和 JWT secret。 | `make prod-reset-secrets` |
| `make prod-deploy-remote` | 自动轮换凭据、构建镜像、清空远程数据库并完成全新部署。 | `make prod-deploy-remote` |
| `make prod-hot-deploy-remote` | 构建镜像并热部署后端服务，保留远程 PostgreSQL 数据。 | `make prod-hot-deploy-remote` |
| `make prod-destroy-remote` | 删除远程 Athena 运行资源和 PostgreSQL volume。 | `make prod-destroy-remote` |

### 部署前本地预演

部署远端服务器前，可以先用生产镜像和生产 compose 在本机跑一次。`prod-start-local` 会创建 PostgreSQL volume、启动 PostgreSQL、执行 migration，再启动其余服务。

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
make prod-build-local
make prod-start-local
```

`prod-start-local` 会强制设置 `ATHENA_SERVER_DISABLE_AUTH=false`，即使环境文件中配置为 `true`，本地生产预演仍会启用服务端认证。

生产 compose 中各后端服务设置了 `ATHENA_POSTGRES_AUTO_MIGRATE=false`。`prod-start-local` 会在启动业务服务前自动执行 `athena up --module $(PROD_MIGRATE_MODULE)`，默认迁移全部模块；迁移失败时命令会终止并保留 PostgreSQL 容器，便于排查。

查看日志和访问本地服务：

```bash
make prod-logs-local
```

```text
http://127.0.0.1:8080
```

停止本地预演：

```bash
make prod-stop-local
```

`prod-stop-local` 会删除 compose 容器、孤立容器、网络和 `PROD_POSTGRES_VOLUME` 指定的 PostgreSQL volume，但保留本地构建的 `PROD_IMAGE` 镜像。下一次启动会重新创建空数据库并执行 migration。

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

一键部署：

```bash
make prod-deploy-remote
```

该命令会先更新 `$(PROD_ENV_FILE)` 中的 `POSTGRES_PASSWORD`、`REDIS_PASSWORD` 和 `ATHENA_JWT_SECRET`，再构建本地镜像。构建成功后，依次停止远端旧服务、删除并重建 `$(PROD_POSTGRES_VOLUME)`、上传 `docker-compose.prod.yml`、`.env` 和 PostgreSQL init 脚本、传输镜像、启动 PostgreSQL、执行 `athena up --module $(PROD_MIGRATE_MODULE)`，最后启动全部服务并输出容器状态。

**每次远程部署都会永久删除已有 PostgreSQL 数据，并轮换 PostgreSQL、Redis 和 JWT secret，不会自动备份。** JWT secret 轮换后旧登录 Token 会失效。migration 失败时不会启动业务服务，PostgreSQL 容器会保留以便排查。

如需只手动更新生产凭据文件而不部署：

```bash
make prod-reset-secrets
```

后端代码小幅修改时，可以保留现有数据库并热部署：

```bash
make prod-hot-deploy-remote
```

该命令会构建并传输新镜像，覆盖远端 `docker-compose.prod.yml` 和 `.env`，确认 PostgreSQL 就绪并幂等确保精确的 `profit_sharing` 数据库存在，再在现有数据上执行 migration，然后强制重建全部 Athena 后端服务并最后重建 `athena-server`。PostgreSQL、Redis 和 PostgreSQL volume 不会停止或删除；如果指定的 volume 不存在，命令会直接终止，避免意外创建空数据库。数据库创建、连接或 migration 失败时，当前业务容器保持运行且不会进入重建阶段。

热部署不会自动轮换 PostgreSQL、Redis 或 JWT secret。它会短暂重启 Athena 服务，不保证零停机；适用于代码更新和兼容性数据库 migration，不用于修改现有 PostgreSQL 或 Redis 凭据。

一键删除：

```bash
make prod-destroy-remote
```

该命令会删除远端 Compose 容器、孤立容器、网络和 `$(PROD_POSTGRES_VOLUME)`。命令可重复执行，不需要额外确认参数；远端 `$(REMOTE_APP_DIR)` 内的部署文件和已加载的 `$(PROD_IMAGE)` 镜像会保留。

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

远端日志和状态不再提供独立 Makefile 目标，可直接使用 SSH：

```bash
ssh root@47.245.181.189 'cd /root/athena && docker compose -f docker-compose.prod.yml --env-file .env ps'
ssh root@47.245.181.189 'cd /root/athena && docker compose -f docker-compose.prod.yml --env-file .env logs -f athena-server'
```
