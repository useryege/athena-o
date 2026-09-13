# Development Toolchain

Athena uses the local toolchain for day-to-day development.

## Local Toolchain

Install the code generation tools:

```bash
make install-codegen-tools-local
```

For UI work, install dependencies directly in the UI directory:

```bash
cd ui
nvm install
nvm use
yarn install
cd ..
```

`ui/.nvmrc` selects the local Node.js version (currently `24.14.1`), while the UI
package accepts `>=24.14.1 <25`. With an existing NVM installation, these commands
install and select that project version without changing the global NVM default.

For local process orchestration, install `goreman` if it is not already present:

```bash
go install github.com/mattn/goreman@latest
```

Start the local stack:

```bash
cd ui
nvm use
cd ..
make run
```

Run these commands in the same shell from the repository root so the local stack
uses the Node.js version selected by `ui/.nvmrc`.

Run generated-code updates when needed:

```bash
make codegen-local
```

## AI 开发检查工具

在当前 Ubuntu 24.04 / WSL 开发环境中，先在同一终端激活项目 Node，再从仓库根目录安装：

```bash
cd ui
nvm use
cd ..
make install-ai-dev-tools
make ai-dev-tools-check
```

安装入口复用项目安装器：ShellCheck `0.11.0`、grpcurl `1.9.4`、govulncheck
`1.7.0` 放在仓库 `dist/`，不修改全局 PATH。发布包校验 SHA256；govulncheck
使用当前 Go 工具链安装，不自动升级 Go。系统安装 `postgresql-client-16`
（可能需要 sudo），仅提供客户端，接受 Ubuntu 仓库的补丁更新。UI 开发依赖
`@axe-core/playwright` 固定为 `4.13.0`，安装时读取已有 Yarn 锁文件。

就绪检查只验证安装状态，不下载依赖，也不代表项目扫描没有发现问题。

| 修改或排查场景 | 入口 | 如何使用结果 |
| --- | --- | --- |
| 修改启动、安装、验收脚本 | `make lint-shell` | 检查 `hack/` 和 `ui/scripts/` 的 Shell 脚本；按文件与规则定位问题 |
| 检查 Go 依赖的已知漏洞 | `make vuln-check` | 分析默认构建条件下的 `./...`，查看漏洞及调用路径；集成测试构建标签不在默认范围内 |
| 检查 UI 无障碍 | `make ui-a11y` | 使用独立隔离验收，查看 axe 原始结果和 Playwright 报告 |
| 排查 gRPC 接口 | `./dist/grpcurl` | 对实际监听地址查询反射或提供 proto 定义，再调用具体方法 |
| 验证 SQL、表结构或连接 | `psql` | 使用明确的开发数据库连接执行查询 |

ShellCheck 与 govulncheck 的扫描日志保存在 `.tmp/ai-dev-tools/`，命令保留失败
退出状态。govulncheck 使用文本输出；不要将其 JSON 模式的零退出码解释成没有漏洞。
网络、依赖加载和运行环境错误应先排查，不能当作扫描成功。

### gRPC 与 PostgreSQL 调试

ATHENA 主服务注册了 gRPC reflection 和标准健康服务。将 `GRPC_ADDR` 设置为
当前实例实际监听的 `host:port` 后，对本地明文端口执行：

```bash
./dist/grpcurl -plaintext "$GRPC_ADDR" list
./dist/grpcurl -plaintext "$GRPC_ADDR" describe grpc.health.v1.Health
./dist/grpcurl -plaintext -d '{"service":""}' "$GRPC_ADDR" grpc.health.v1.Health/Check
```

其他子服务未必开放反射；此时使用 grpcurl 的 `-import-path` 和 `-proto` 指向
仓库对应定义，或使用 `-protoset`。TLS 服务使用相应证书选项，不套用明文示例。

将 `PGSERVICE` 指向本机已配置的开发连接，或使用 `PGHOST`、`PGPORT`、`PGUSER`、
`PGDATABASE` 和密码文件配置，然后执行：

```bash
psql -X -v ON_ERROR_STOP=1 -c 'SELECT current_database(), version();'
psql -X -v ON_ERROR_STOP=1 -c '\dt'
```

客户端安装不会启动数据库服务；数据库仍由现有 Docker / 本地运行流程管理。

### 独立无障碍检查

`make ui-a11y` 固定使用隔离模式，复用浏览器验收的 UI 构建、临时数据库、测试
服务、运行锁和清理逻辑，不需要先运行 `make run`。前置检查可执行：

```bash
UI_ACCEPTANCE_CHECK_ONLY=1 make ui-a11y
```

首批检查成员订阅列表、添加表单、取消确认弹窗和管理员同步状态页面，覆盖桌面与
移动视口、明暗主题，以及根路径和 `/athena`。规则范围为 WCAG 2 A/AA、2.1 A/AA。
各场景保留 axe 原始结果；违规不会阻止后续场景及另一前缀收集，最终仍返回失败。
报告沿用 `.tmp/athena-ui-acceptance/<run-id>/`。

该入口补充现有 `make ui-acceptance` 的功能、键盘和视觉检查。自动扫描只能发现
部分无障碍问题，不能替代实际操作验收。首轮发现的现有问题在本次工具接入中记录，
不通过禁用规则、隐藏元素或批量修改业务代码将结果变成通过。

本机安装版本和首轮发现见 [AI 开发工具验收记录](../testing/ai-dev-tools-readiness.md)。
