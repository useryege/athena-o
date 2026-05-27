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
| `PROD_CLEAR_DATA` | `false` | 远端生产部署时是否删除 compose volumes。只有精确等于 `true` 时生效。 |
| `REMOTE_APP_DIR` | `/root/athena` | 远端服务器上的部署目录。 |
| `REMOTE_USER` | `root` | SSH 登录远端服务器使用的用户。 |
| `PROD_LOG_SERVICE` | 空 | 查看生产日志时指定服务名。为空时查看全部服务。 |
| `TEST_MODULE` | 空 | 指定要运行的 Go 测试包。为空时运行全部非 E2E 测试。 |
| `TARGET_ARCH` | `linux/amd64` | Docker 镜像构建平台。 |
| `ATHENA_*` | 多个默认值 | 控制本地、E2E、端口、数据目录、认证等运行参数。可先执行 `make print-env-vars` 查看部分配置。 |

## 环境与工具

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make print-env-vars` | 打印 Makefile 中常用环境变量和路径。 | `make print-env-vars` |
| `make install-tools-local` | 安装本地开发、测试、代码生成需要的工具。 | `make install-tools-local` |

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

常见远端部署流程：

```bash
PROD_IMAGE=athena:local make prod-build-local
make prod-deploy-remote
```

查看指定服务日志：

```bash
PROD_LOG_SERVICE=athena-server make prod-logs-remote
```

### 数据清理风险

默认部署不会删除远端 PostgreSQL 数据：

```bash
make prod-deploy-remote
```

如果明确需要清空远端 compose volumes，可以使用：

```bash
PROD_CLEAR_DATA=true make prod-deploy-remote
```

注意：该命令会在远端执行 `docker compose down --volumes`。当前 `docker-compose.prod.yml` 中声明了 `postgres-data`，因此这会清空远端 PostgreSQL 数据，并在下次启动时重新执行 `hack/postgres/init` 初始化脚本。

## 清理

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make clean` | 删除 `dist`，并清理 VSCode `debug.test` 文件。 | `make clean` |
| `make clean-postgres-data` | 删除本地 PostgreSQL 和 Redis 数据目录。 | `make clean-postgres-data` |

注意：`make clean-postgres-data` 会执行 `sudo rm -rf "$(ATHENA_POSTGRES_DATA_DIR)" "$(ATHENA_REDIS_DATA_DIR)"`，默认会删除 `/tmp/athena-local/postgres` 和 `/tmp/athena-local/redis`。
