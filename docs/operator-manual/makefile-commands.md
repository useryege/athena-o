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
| `PROD_MIGRATE_MODULE` | `all` | 迁移目标模块。可设为 `account-state`、`worm-markets`、`fifa-market-dashboard`、`notification`、`wallet`、`sports-live`、`sports-history`、`managed-oo`、`profit-sharing`、`token` 或 `all`。 |
| `PROD_POSTGRES_VOLUME` | `athena-prod-postgres-data` | PostgreSQL external volume 名称。本地停止、远程部署和远程删除都会删除该 volume。 |
| `PROD_REDIS_VOLUME` | `athena-prod-redis-data` | Redis AOF external volume 名称。本地停止、全新远程部署和远程删除都会删除该 volume；热部署保留。 |
| `PROD_MINIO_VOLUME` | `athena-prod-minio-data` | MinIO external volume 名称。本地停止、全新远程部署和远程删除都会删除该 volume；热部署保留。 |
| `MINIO_IMAGE` | `athena-minio:9e49d5e7a648` | 从固定 MinIO Server commit 构建的镜像名。 |
| `MINIO_MC_IMAGE` | `athena-minio-mc:7394ce0dd2a8` | 从固定 mc commit 构建的一次性初始化镜像名。 |
| `ATHENA_POSTGRES_AUTO_MIGRATE` | 本地默认 `true`，生产 compose 为 `false` | 控制服务启动时是否自动执行 PostgreSQL migration。生产部署脚本会在启动业务服务前显式迁移。 |
| `TARGET_ARCH` | `linux/amd64` | Docker 镜像构建平台。 |

## 环境与工具

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make install-codegen-tools-local` | 安装代码生成需要的工具。 | `make install-codegen-tools-local` |
| `make jwt-secret` | 生成可用于 `ATHENA_JWT_SECRET` 的 HS256 随机签名密钥。 | `make jwt-secret` |
| `make service-password` | 生成可用于 PostgreSQL、Redis 或 MinIO 的随机密码。 | `make service-password` |
| `make wallet-private-key-ciphertext` | 生成可用于 `wallet_private_keys.private_key_ciphertext` 的密文 SQL 表达式。 | `make wallet-private-key-ciphertext` |

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

Google OIDC 登录和 API Key 都使用 Athena 自有 JWT v2。切换认证方案或需要强制
所有浏览器会话与 API Key 失效时，生成新值并替换 `ATHENA_JWT_SECRET`；随后所有
用户需要重新登录，自动化调用方需要重新创建 API Key。

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

## Google OIDC 配置

Google 只负责确认外部身份。任意通过完整 OIDC 校验且邮箱已验证的 Google 用户都可
登录；未知 `sub` 会创建一个 `user-<UUID>` 内部账号，初始没有业务、API Key 或
Profit Sharing 权限。认证启用时必须配置以下变量：

```env
ATHENA_GOOGLE_OIDC_CLIENT_ID='<google-web-client-id>'
ATHENA_GOOGLE_OIDC_REDIRECT_URI='<exact-callback-uri>'
ATHENA_ADMIN_GOOGLE_EMAIL='<administrator-google-email>'
```

在 Google Cloud 中分别创建本地和生产 **Web application** OAuth client，配置
consent screen/audience。要允许任意 Google 用户，Audience 必须为 **External** 且
应用必须发布；Testing 状态仍只允许 Test users。登记完全一致的 authorized
redirect URI：

- 本地：`http://localhost:4000/auth/google/callback`
- 生产：`https://<athena-domain>/auth/google/callback`

生产 URI 必须显式使用 HTTPS。Athena 不会从请求的 `Host`、
`X-Forwarded-Host` 等 header 推断回调地址。普通账号始终按 Google `sub` 查找；
邮箱只用于安全审计，以及未绑定 `admin` 的首次认领。管理员邮箱比较仅去除首尾空白
并忽略大小写，不归并 Gmail 点号或 `+alias`。认领后持久化 `sub` 永久优先，修改
环境邮箱不能替换管理员身份。

本地开发可直接设置 `ATHENA_GOOGLE_OIDC_CLIENT_SECRET`。生产必须保持该变量为空，
只使用独立文件：

```env
ATHENA_GOOGLE_OIDC_CLIENT_SECRET=''
ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE='./secrets/google-oidc-client-secret'
```

准备文件时不要把 secret 写入环境文件或提交到 Git：

```bash
mkdir -p secrets
install -m 0600 /secure/source/google-oidc-client-secret secrets/google-oidc-client-secret
```

生产 Compose 将文件以只读 secret 仅挂载到 `athena-server` 的
`/run/secrets/google-oidc-client-secret`；migration 和其他业务容器不会获得文件
内容。本地生产预演让 `athena-server` 使用当前宿主机 UID 读取该用户自己的 `0600`
文件；远程部署脚本把上传副本改为容器 UID/GID `999` 持有并保持 `0600`。设置
`ATHENA_SERVER_DISABLE_AUTH=true` 时保留开发管理员旁路，不要求 Google 配置；
该旁路只允许非 Compose 的开发进程监听 loopback 地址。生产 Compose 固定启用认证，
部署脚本会拒绝该旁路。

虽然其他生产服务仍复用选定的部署 env 文件读取各自业务配置，Compose 会把
`ATHENA_JWT_SECRET`、全部 Google OIDC/管理员邮箱输入以及 `REDIS_PASSWORD` 在所有
非 API Server 容器中显式覆盖为空；只有 `athena-server` 能签发 Athena JWT、访问
持久账号目录或访问认证 Redis。
远端上传后的 `.env` 同样改为当前部署用户持有且权限为 `0600`。

## 代码生成

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make codegen-local` | 在本机执行完整代码生成流程。 | `make codegen-local` |
| `make protogen` | 先准备 vendor，再生成 protobuf 相关代码。 | `make protogen` |

## 生产构建

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make minio-images-local` | 从固定源码 commit 构建 MinIO Server 和 mc 初始化镜像。 | `make minio-images-local` |
| `make prod-build-local` | 构建 MinIO/mc 镜像及生产部署使用的 Athena 镜像。 | `make prod-build-local` |

常见用法：

```bash
PROD_IMAGE=athena:local make prod-build-local
```

## 本地运行

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make run` | 前台启动本地服务，创建可重建的 PostgreSQL、Redis、MinIO 容器并复用持久化数据。支持 `ATHENA_RUN_EXCLUDE`。 | `make run` |
| `make stop` | 优雅停止本地服务并删除容器和运行控制状态，保留 PostgreSQL、Redis、MinIO 数据 volume。 | `make stop` |
| `make run-reset` | 先停止服务，再删除本地容器、数据 volume、运行控制状态和默认临时运行数据。不会重新启动。 | `make run-reset` |

本地 PostgreSQL、Redis 和私有头像对象分别保存在固定命名 volume
`athena-local-postgres-data`、`athena-local-redis-data` 和
`athena-local-minio-data`。前台按
`Ctrl+C` 与从另一终端执行 `make stop` 具有相同的浅层清理语义，后续
`make run` 会创建新容器并挂载原数据。PostgreSQL volume 会记录镜像、
用户、初始数据库、密码和初始化 SQL 的配置指纹；这些初始化设置发生变化
后必须执行 `make run-reset`，避免以新配置静默打开不兼容的旧数据。
MinIO 首次运行会从固定源码 commit 构建 Server/mc 镜像，并初始化
`athena-account-avatars` 私有 bucket 和最小权限应用账号。本地 API、Console
默认只绑定 `127.0.0.1:9000` 和 `127.0.0.1:9001`。
Profit Sharing 新增独立的 `profit_sharing` 数据库和 `8108` 端口；首次使用
包含该数据库的初始化配置时同样必须执行 `make run-reset`。本地 Procfile
默认启用 API Server 认证。普通用户首次登录创建 Pending 动态账号，管理员授予独立
Profit Sharing 权限并把账号加入轮次后，用户才能提交方案或投票；`admin` 首次按
配置邮箱认领，此后始终按持久化 Google `sub` 保持管理员身份。

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
| `make prod-start-local` | 创建本地 PostgreSQL/Redis/MinIO volume、执行 migration、初始化私有 bucket 并启动生产 compose 服务。 | `make prod-start-local` |
| `make prod-stop-local` | 停止本机生产 compose 服务并删除 PostgreSQL/Redis/MinIO volume。 | `make prod-stop-local` |
| `make prod-logs-local` | 查看本机生产 compose 日志。 | `make prod-logs-local` |
| `make prod-reset-secrets` | 更新生产 env 中的 PostgreSQL、Redis、MinIO root、头像应用凭据和 JWT secret。 | `make prod-reset-secrets` |
| `make prod-deploy-remote` | 自动轮换凭据、构建镜像、清空远程数据库并完成全新部署。 | `make prod-deploy-remote` |
| `make prod-hot-deploy-remote` | 构建镜像并热部署后端服务，保留远程 PostgreSQL、Redis 和 MinIO 数据。 | `make prod-hot-deploy-remote` |
| `make prod-destroy-remote` | 删除远程 Athena 运行资源及 PostgreSQL/Redis/MinIO volume。 | `make prod-destroy-remote` |

### 部署前本地预演

部署远端服务器前，可以先用生产镜像和生产 compose 在本机跑一次。`prod-start-local` 会创建 PostgreSQL/Redis/MinIO volume、启动 PostgreSQL、执行 migration、初始化私有头像 bucket，再启动其余服务。Redis 使用 AOF 持久化未过期的会话撤销记录。

如果使用 `.env.prod` 作为预演环境文件，至少需要包含：

```env
POSTGRES_PASSWORD=your_postgres_password
REDIS_PASSWORD=your_redis_password
MINIO_ROOT_PASSWORD=your_minio_root_password
ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID=your_avatar_access_key
ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY=your_avatar_secret_key
ATHENA_JWT_SECRET=your_at_least_32_byte_jwt_secret
ATHENA_WALLET_ENCRYPTION_KEY=your_wallet_encryption_key
ATHENA_GOOGLE_OIDC_CLIENT_ID=your_production_web_client_id
ATHENA_GOOGLE_OIDC_CLIENT_SECRET=
ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE=./secrets/google-oidc-client-secret
ATHENA_GOOGLE_OIDC_REDIRECT_URI=https://athena.example.com/auth/google/callback
ATHENA_ADMIN_GOOGLE_EMAIL=owner@example.com
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

`prod-start-local` 会强制设置 `ATHENA_SERVER_DISABLE_AUTH=false`，即使环境文件中配置为
`true`，本地生产预演仍会启用服务端认证。运行前必须创建
`./secrets/google-oidc-client-secret` 并配置真实的 client ID、生产预演回调 URI 和
管理员 Google 邮箱；缺失配置会使 API Server 拒绝启动。首次验证登录会认领
`admin` 或创建 Pending 动态账号，不需要预先提取 `sub`。
Compose 会同时使用 `$(PROD_ENV_FILE)` 做变量插值和容器 `env_file` 注入，不会回退
读取仓库根目录的 `.env`。`prod-reset-secrets` 会把该文件权限收紧为 `0600`。

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

`prod-stop-local` 会删除 compose 容器、孤立容器、网络，以及
`PROD_POSTGRES_VOLUME`、`PROD_REDIS_VOLUME` 和 `PROD_MINIO_VOLUME` 指定的三个
volume，但保留本地
构建的 Athena、MinIO 和 mc 镜像。下一次启动会重新创建空数据库和私有 bucket。

### 远程部署

以下示例使用：

```env
REMOTE_HOST=47.245.181.189
REMOTE_USER=root
REMOTE_APP_DIR=/root/athena
PROD_POSTGRES_VOLUME=athena-prod-postgres-data
PROD_REDIS_VOLUME=athena-prod-redis-data
PROD_MINIO_VOLUME=athena-prod-minio-data
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

该命令会先更新 `$(PROD_ENV_FILE)` 中的 PostgreSQL、Redis、MinIO root、
头像 bucket 应用凭据和 JWT secret，再构建 Athena、固定源码 MinIO 和 mc
镜像。构建成功后，依次停止远端旧服务，删除并重建
`$(PROD_POSTGRES_VOLUME)`、`$(PROD_REDIS_VOLUME)` 与
`$(PROD_MINIO_VOLUME)`，上传 Compose、环境文件和
Google OIDC client secret 文件以及 PostgreSQL init 脚本，传输三个镜像，执行
migration，初始化私有 bucket，最后启动全部服务并输出容器状态。部署脚本会在
上传前拒绝直接环境变量形式的 client secret、空 client/管理员邮箱、
非 HTTPS 生产回调 URI、少于 32 字节的 JWT signing secret 或空 Google secret
文件，也拒绝生产环境使用 `ATHENA_SERVER_DISABLE_AUTH=true`。Docker 构建上下文会
排除所有 `.env` 文件和 `secrets/` 目录，避免部署凭据进入镜像构建缓存。

**每次远程部署都会永久删除已有 PostgreSQL、Redis 和 MinIO 数据，并轮换
PostgreSQL、Redis、MinIO 和 JWT 凭据，不会自动备份。** 动态账号、管理员绑定、
Profile、权限、Profit Sharing 引用、头像、会话和 API Key 都不会迁移。所有用户
必须重新使用 Google 登录，管理员重新授权；需要自动化访问的账号必须在获得 API Key
权限后创建新的 v2 Key。migration 或 bucket 初始化失败时不会启动 API Server。

如需只手动更新生产凭据文件而不部署：

```bash
make prod-reset-secrets
```

后端代码小幅修改时，可以保留现有数据库并热部署：

```bash
make prod-hot-deploy-remote
```

该命令会构建并传输三个新镜像，覆盖远端 Compose、环境文件和 Google OIDC
client secret 文件，等待 PostgreSQL 和 Redis 就绪，随后启动并等待 MinIO，幂等确保
精确的 `profit_sharing` 数据库和私有头像 bucket 存在，再在现有数据上执行
migration，然后强制重建全部 Athena
后端服务并最后重建 `athena-server`。PostgreSQL、Redis 和 MinIO 的三个持久化
volume 数据都会保留；Compose 仅在配置变化要求时重建对应 stateful container，
不会删除 volume。任一指定 volume 不存在时命令直接终止，避免意外创建空状态。
依赖初始化或 migration 失败时不会进入应用重建阶段。

热部署不会自动轮换 PostgreSQL、Redis、MinIO 或 JWT secret。它会校验并重新
上传当前独立的 Google OIDC client secret，然后短暂重启
Athena 服务，不保证零停机；适用于代码更新和兼容性数据库 migration，不用于
修改现有持久化服务凭据。

一键删除：

```bash
make prod-destroy-remote
```

该命令会删除远端 Compose 容器、孤立容器、网络、
`$(PROD_POSTGRES_VOLUME)`、`$(PROD_REDIS_VOLUME)` 和
`$(PROD_MINIO_VOLUME)`。命令可重复执行，不需要
额外确认参数；不存在的 volume 会被忽略，但任何仍存在的 volume 删除失败都会使
命令失败，不会继续宣称销毁完成或在全新部署中复用旧数据。远端部署文件和已加载的
三个镜像会保留。

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

如果日志报告 Google OIDC client、redirect URI 或管理员邮箱为空，表示认证配置未
完成。正常生产部署不会生成临时管理员凭据，也没有密码兜底入口；修正配置并重启
`athena-server`，不要通过关闭认证绕过问题。

远端日志和状态不再提供独立 Makefile 目标，可直接使用 SSH：

```bash
ssh root@47.245.181.189 'cd /root/athena && docker compose -f docker-compose.prod.yml --env-file .env ps'
ssh root@47.245.181.189 'cd /root/athena && docker compose -f docker-compose.prod.yml --env-file .env logs -f athena-server'
```
