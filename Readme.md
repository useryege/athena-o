# Athena Makefile 常用命令指南

本文档说明项目根目录 `Makefile` 中常用命令的使用方式，偏向日常开发、测试、构建和部署速查。

## 前置依赖

常用命令会依赖以下工具：

- Go：用于本地构建、测试、代码生成。
- Docker：用于构建镜像、启动容器化测试工具、运行生产 compose。
- kubectl：用于本地和 E2E 环境的 Kubernetes 命名空间与资源操作。
- yarn：用于 `ui` 目录依赖安装、检查和构建。
- mkdocs：用于本地文档预览和文档构建；也可以通过 Docker 目标运行。

## 命令约定

- 带 `-local` 后缀的目标通常直接使用本机工具链执行，例如 `make build-local`。
- 不带 `-local` 的构建、测试、生成目标通常会先构建 `athena-test-tools` 镜像，再在测试工具容器中执行，例如 `make test`。
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
| `ATHENA_POSTGRES_AUTO_MIGRATE` | 本地默认 `true`，生产 compose 为 `false` | 控制服务启动时是否自动执行 PostgreSQL migration。生产环境通过显式迁移命令控制 schema 演进。 |
| `TEST_MODULE` | 空 | 指定要运行的 Go 测试包。为空时运行全部非 E2E 测试。 |
| `TARGET_ARCH` | `linux/amd64` | Docker 镜像构建平台。 |
| `ATHENA_*` | 多个默认值 | 控制本地、E2E、端口、数据目录、认证等运行参数。可先执行 `make print-env-vars` 查看部分配置。 |

## 环境与工具

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make print-env-vars` | 打印 Makefile 中常用环境变量和路径。 | `make print-env-vars` |
| `make install-tools-local` | 安装本地开发、测试、代码生成需要的工具。 | `make install-tools-local` |
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
| `make abigen-local` | 生成 Solidity ABI 相关 Go 代码。 | `make abigen-local` |
| `make manifests-local` | 更新 Kubernetes manifests。 | `make manifests-local` |

## 构建

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make build-local` | 在本机编译全部 Go 代码，不包含 E2E 包。 | `make build-local` |
| `make cli-local` | 构建本地 `athena` CLI 到 `dist/athena`。 | `make cli-local` |
| `make image` | 构建 Athena Docker 镜像，可配合 `DOCKER_PUSH=true` 推送。 | `make image` |
| `make prod-build-local` | 构建生产部署使用的本地镜像。 | `make prod-build-local` |

常见用法：

```bash
PROD_IMAGE=athena:local make prod-build-local
TARGET_ARCH=linux/amd64 make image
```

## 测试与检查

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make test-local` | 运行本机单元测试。 | `make test-local` |
| `make test-race-local` | 使用 Go race detector 运行单元测试。 | `make test-race-local` |
| `make lint-local` | 运行 `golangci-lint`。 | `make lint-local` |
| `make pre-commit-local` | 在本机执行代码生成、构建、lint、测试。 | `make pre-commit-local` |

只测试某个包：

```bash
TEST_MODULE=./internal/application/... make test-local
```

## 本地运行

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make start-local` | 准备依赖并通过 `goreman` 启动本地 Athena。 | `make start-local` |
| `make run` | 通过 `hack/goreman-start.sh` 启动，可使用脚本支持的排除参数。 | `make run` |

`make start-local` 会清理并重建 `/tmp/athena-local`，适合需要完整本地环境时使用。

## UI

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make dep-ui-local` | 在 `ui` 目录执行 `yarn install`。 | `make dep-ui-local` |
| `make lint-ui-local` | 在 `ui` 目录执行 `yarn lint`。 | `make lint-ui-local` |
| `make build-ui` | 通过 Docker 构建 UI，并更新 `ui/dist/app`。 | `make build-ui` |

## 文档

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make serve-docs-local` | 使用本机 `mkdocs serve` 预览文档。 | `make serve-docs-local` |
| `make build-docs` | 使用 Docker 构建 MkDocs 文档。 | `make build-docs` |

## E2E

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make start-e2e-local` | 在本机启动 E2E 所需服务和资源。 | `make start-e2e-local` |
| `make test-e2e-local` | 运行 E2E 测试。需要先启动 E2E 服务。 | `make test-e2e-local` |

E2E 测试超时时间可通过 `ATHENA_E2E_TEST_TIMEOUT` 调整：

```bash
ATHENA_E2E_TEST_TIMEOUT=120m make test-e2e-local
```

## 生产部署

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make prod-start-local` | 使用生产 compose 在本机启动服务。 | `make prod-start-local` |
| `make prod-stop-local` | 停止本机生产 compose 服务，不删除 volume。 | `make prod-stop-local` |
| `make prod-logs-local` | 查看本机生产 compose 日志。 | `make prod-logs-local` |
| `make prod-deploy-remote` | 上传 compose、`.env`、PostgreSQL init 脚本和镜像到远端，并启动服务。 | `make prod-deploy-remote` |
| `make prod-start-remote` | 在远端执行生产 compose 启动。 | `make prod-start-remote` |
| `make prod-stop-remote` | 在远端停止生产 compose 服务，不删除 volume。 | `make prod-stop-remote` |
| `make prod-logs-remote` | 查看远端生产 compose 日志。 | `make prod-logs-remote` |
| `make prod-migrate-remote` | 在远端手动执行 PostgreSQL migration。默认迁移全部模块。 | `make prod-migrate-remote` |
| `make prod-migration-status-remote` | 查看远端 PostgreSQL migration 状态。默认查看全部模块。 | `make prod-migration-status-remote` |
| `make prod-db-backup-remote` | 在远端通过 `pg_dumpall` 备份 PostgreSQL 到 `$(REMOTE_APP_DIR)/backups`。 | `make prod-db-backup-remote` |
| `make prod-destroy-data-remote` | 危险操作：停止远端 compose 并删除生产 PostgreSQL volume。必须显式确认。 | `make prod-destroy-data-remote CONFIRM_DESTROY_PROD_DATA=yes` |

`prod-deploy-remote` 会执行以下操作：

1. 检查远端 Docker 和 Docker Compose 是否可用。
2. 创建远端部署目录 `$(REMOTE_APP_DIR)`。
3. 确保生产 PostgreSQL external volume 存在，默认名为 `athena-prod-postgres-data`。
4. 上传 `docker-compose.prod.yml`、`.env` 和 `hack/postgres/init`。
5. 将本地 `$(PROD_IMAGE)` 镜像传输到远端。
6. 执行 `docker compose up -d` 启动服务。

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

唯一允许删除生产 PostgreSQL volume 的入口是：

```bash
make prod-destroy-data-remote CONFIRM_DESTROY_PROD_DATA=yes
```

该命令会在远端停止 compose，并删除 `$(PROD_POSTGRES_VOLUME)` 指向的 Docker volume。默认 volume 名为 `athena-prod-postgres-data`。这是不可逆的危险操作，执行前必须确认已经完成备份。

## 清理

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make clean` | 删除 `dist`，并清理 VSCode `debug.test` 文件。 | `make clean` |
| `make clean-postgres-data` | 删除本地 PostgreSQL 和 Redis 数据目录。 | `make clean-postgres-data` |

注意：`make clean-postgres-data` 会执行 `sudo rm -rf "$(ATHENA_POSTGRES_DATA_DIR)" "$(ATHENA_REDIS_DATA_DIR)"`，默认会删除 `/tmp/athena-local/postgres` 和 `/tmp/athena-local/redis`。
