# Development Toolchain

Athena uses Go `1.27.1` from `go.mod` and the local toolchain for day-to-day development.

## Local Toolchain

After changing the Go toolchain, rebuild Go-installed tools. The pinned
`golangci-lint` version is `2.13.2`; `govulncheck` stays at `1.7.0` but must be
compiled with Go `1.27.1`. Verify the compiler and compiled tool version with
`go version` and `go version -m dist/govulncheck`.

Install the code generation tools:

```bash
make install-codegen-tools-local
```

The code generation installer pins `mockery` to `3.8.0`, which supports Go
`1.27.1`. Reinstall these tools after upgrading Go; rebuilding the older
`mockery 3.6.1` release does not resolve its Go 1.27 type-loading failure.

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

Local orchestration uses the repository's Go instance runtime on Linux/WSL with Bash5.1+ and Docker. It does not require Goreman. A selected Go service uses `make run-service SERVICE=trader-sync`; only the explicit full stack or a set containing UI needs Node.

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
使用当前 Go `1.27.1` 工具链安装，并检查可执行文件的编译 Go 版本；切换 Go 后应重新
安装，即使 `govulncheck` 自身仍为 `1.7.0`。安装器不自动升级 Go。系统安装 `postgresql-client-16`
（可能需要 sudo），仅提供客户端，接受 Ubuntu 仓库的补丁更新。UI 开发依赖
`@axe-core/playwright` 固定为 `4.13.0`，安装时读取已有 Yarn 锁文件。

就绪检查只验证安装状态，不下载依赖，也不代表项目扫描没有发现问题。

### 按任务选择工具

AI 通过终端调用这些项目命令即可。先确定需要回答的问题，再选择能提供对应证据的
工具和检查范围；局部修改先检查相关文件或包。纯文档错别字修改通常只需核对差异，
不需要启动浏览器、数据库或全仓扫描。用户明确要求的检查和任务方案中的必要验收
仍按要求执行。

工具选择表同时覆盖首批新增工具和项目已有工具，不是固定执行顺序：

| 修改或排查场景 | 入口 | 如何使用结果 |
| --- | --- | --- |
| 理解 Go 代码、定位类型或引用 | `gopls`、编辑器语言服务 | 导航和诊断；不能代替运行测试 |
| 修改 Go 实现或修复回归 | 对应包的 `go test`、`golangci-lint run` | 先验证受影响包；新增诊断与已有问题分别判断 |
| 修改 UI ESLint 配置或应用隔离导入规则 | `cd ui && yarn lint` | 先用真实 ESLint 配置检查内存回归用例，再执行 TypeScript 与全量 ESLint；记录见 [ESLint files 匹配验收](../testing/eslint-config-matching.md) |
| 修改启动、安装、验收脚本 | `./dist/shellcheck -x -P . <脚本路径>`；需要全仓扫描时用 `make lint-shell` | 检查相关 Shell 脚本；结合实际展开和运行行为判断诊断 |
| 升级 Go/依赖或检查已知漏洞 | `make vuln-check` | 分析默认构建条件下的 `./...`，查看漏洞及调用路径；集成测试构建标签不在默认范围内 |
| 检查页面、交互或浏览器回归 | [athena-browser-acceptance](../../.codex/skills/athena-browser-acceptance/SKILL.md) | 按所需证据选择交互检查、隔离 `make ui-acceptance` 或真实 smoke |
| 检查 UI 无障碍 | `make ui-a11y` | 使用独立隔离验收，查看 axe 原始结果和 Playwright 报告 |
| 验证 DOM 交互测试工具接入 | `cd ui && yarn test:dom` | 运行 Testing Library／user-event 表单样例，不代表现有业务组件已迁移 |
| 验证视觉比对工具接入 | `cd ui && yarn test:visual` | 运行静态样例的基线比较；不能代替产品页面验收 |
| 验证 Go Fuzz 工具接入 | `make test-fuzz` | 限时探索分页游标样例，保留失败输入；不能据此推断所有输入已覆盖 |
| 排查 gRPC 接口 | `./dist/grpcurl` | 对实际监听地址查询反射或提供 proto 定义，再调用具体方法 |
| 验证 SQL、表结构或连接 | `psql` | 使用明确的开发数据库连接执行查询 |
| 排查本地容器或启动环境 | `docker`、`docker compose`、`make runtime-status` / `make run-service` / `make run` | 先确认目标仓库和进程归属；按任务需要准备环境，遵守 [本地运行规则](running-locally.md#prepare-the-development-environment-for-acceptance) |
| 修改 SQL、Proto、API 类型或合约等生成源 | [sync-athena-changes](../../.codex/skills/sync-athena-changes/SKILL.md)、对应生成入口 | 定位所需 sqlc、Protobuf、mockery、abigen 等工具及消费者；需要全流程时执行 `make codegen-local` |

已有安装记录表示当时的本机状态。新机器或新 worktree 的 `dist/`、`ui/node_modules/`
可能尚未准备；在实际目标工作区检查所选工具，例如 `./dist/grpcurl -version`。
需要核对首批五项工具的整体状态时运行 `make ai-dev-tools-check`；浏览器前置条件
用所选入口加 `UI_ACCEPTANCE_CHECK_ONLY=1`，例如 `UI_ACCEPTANCE_CHECK_ONLY=1 make ui-a11y`。
仅因工具存在或历史报告通过，不必重装工具或重跑全部检查。

版本要求以 `go.mod`、`ui/.nvmrc`、`ui/package.json`、`hack/tool-versions.sh`
及对应安装器为准。就绪检查失败后先核对工作目录、Node 选择和可执行文件位置，
按实际缺项使用现有安装入口。工具就绪、扫描完成、扫描发现问题、真实验收通过
是不同结论；保留日志与退出状态，按当前任务范围处理发现。

ShellCheck 与 govulncheck 的扫描日志保存在 `.tmp/ai-dev-tools/`，命令保留失败
退出状态。govulncheck 使用文本输出；不要将其 JSON 模式的零退出码解释成没有漏洞。
网络、依赖加载和运行环境错误应先排查，不能当作扫描成功。

### 测试工具基础接入

新增能力是 React Testing Library／user-event、Playwright 视觉基线比较和 Go 原生
Fuzz。首批仅提供工具样例，不代表产品交互、视觉或业务流程已完成验收；不替代
`make ui-acceptance`、`make ui-a11y` 或真实环境 smoke。

#### 安装与执行

沿用项目 Node、Yarn 和 Playwright 安装入口，不需要启动业务服务或数据库：

```bash
cd ui
nvm use
yarn install --frozen-lockfile
yarn playwright:install chromium
cd ..
GOTOOLCHAIN=local go mod download
make test-ai-tools
```

React Testing Library `16.3.3`、user-event `14.6.7` 和所需的 DOM Testing Library
`10.4.1` 固定在 UI 开发依赖与锁文件中，沿用 Jest、ts-jest 和当前自定义 jsdom
环境。Go Fuzz 直接使用 `go.mod` 指定的工具链，没有额外 Go 依赖。

| 入口 | 验证范围 |
| --- | --- |
| `cd ui && yarn test:dom` | 独立 React 表单的标签定位、输入、点击、键盘提交及测试间清理；也纳入默认 Jest 集合 |
| `cd ui && yarn test:visual` | 固定静态区域与已提交的视觉基线比较，不构建或访问 ATHENA 页面 |
| `cd ui && yarn test:visual:update` | 显式生成或更新视觉基线，更新结果须审阅后提交 |
| `make test-fuzz` | 分页游标的单个 Fuzz 样例，探索 10 秒，使用两个 worker |
| `make test-ai-tools` | 按 DOM、视觉、Fuzz 顺序运行；任一步失败立即退出，后续步骤不运行 |

上表测试入口不自动下载依赖，也不会更新视觉基线。`make test-fuzz` 禁止模块和
工具链自动下载，并以只读模块模式运行；缺失 Go 缓存时先显式执行安装段中的
`go mod download`。需要先完成安装，缺失工具或浏览器
时按实际错误补齐；`make ai-dev-tools-check` 仍只检查原有五项工具的就绪状态。

#### 视觉基线与报告

独立视觉配置使用锁文件对应的 Chromium、固定视口、浅色主题、locale 和时区，
截图时关闭动画。样例通过 `page.setContent()` 渲染固定内容和内联样式，不发起
业务或外部网络请求。首批只维护 WSL/Linux 基线；截图生成与比较须使用相同的
浏览器版本、系统和字体环境。

普通执行遇到基线缺失或差异会失败，不会创建或改写基线。基线 PNG 随样例提交到
Git；更新时检查实际截图和差异，不能为了消除失败而自动接受变化。Playwright
升级或渲染环境变化后，也应先核对原因再显式更新。

JSON、HTML、失败截图、差异图和失败 trace 保存在 `.tmp/ai-test-tools/visual/`。
这是单次运行的工作目录，下一次执行可能覆盖，需保留的失败证据先复制到独立目录；
同一 worktree 内不要并发运行此工具样例。产品验收仍使用原来的配置与报告目录。

#### Fuzz 种子与复现

`FuzzCursorRoundTrip` 调用真实的分页游标编解码，验证非负快照号往返精确、跨账户
解码拒绝以及负数输入拒绝。它仅是工具接入的最小示例，不表示已经覆盖所有游标、
金额或事件输入。普通 `go test` 只执行种子和已保存的失败样本，不持续生成新输入。

```bash
# 仅执行种子回归。
go test ./internal/tradersync -run '^FuzzCursorRoundTrip$' -count=1
# 显式延长同一目标的探索时间；一次只选择一个 Fuzz 目标。
go test ./internal/tradersync -run '^$' -fuzz '^FuzzCursorRoundTrip$' -fuzztime=60s -parallel=2
# 将 Go 输出中的具体样本名代入，单独复现失败。
go test ./internal/tradersync -run '^FuzzCursorRoundTrip/<样本名>$' -count=1
```

Go 将最小失败样本写入对应包的 `testdata/fuzz/FuzzCursorRoundTrip/`，并打印复现
命令。保留样本和失败日志，修复经过授权的问题后将样本作为回归证据提交；不要删除
样本或削弱断言来获得通过。限时探索成功只代表本次探索未发现失败。

本次接入的正向、负向和回归结果见[工具基础接入验收](../testing/ai-test-tools-foundation.md)。

### Shell 脚本回归

`make lint-shell` 扫描 `hack/` 和 `ui/scripts/` 下全部 `.sh`，包含辅助库和测试。
修改 Shell 行为时，按影响范围运行以下本地回归：

```bash
bash hack/ssh-command_test.sh
bash hack/deploy-scripts_test.sh
bash hack/shell-local_test.sh
bash hack/ai-dev-tools_test.sh
bash hack/trader-sync-local_test.sh
```

前两个测试使用本地 SSH/SCP/Docker 等命令替身，验证参数经过远端 shell 解析后的
实际值、二进制输入、退出码，以及四个部署脚本的操作顺序；不会执行真实远端部署。
本地脚本测试覆盖特殊路径、清理退出码和资源归属、Temporal 重试次数与间隔。
涉及浏览器验收入口时，在同一终端激活 `ui/.nvmrc` 的 Node 后执行
`node --test ui/scripts/acceptance-runner.test.mjs`。

部署脚本通过 `hack/lib/ssh-command.sh` 的 `ssh_exec host command args...`
传递参数。固定脚本文本与配置值分离；不要把配置值直接拼入远端命令，也不要用
脚本标准输入通道覆盖 tar 或镜像数据流。ShellCheck 对动态外部 source、信号回调
等已确认分析边界采用带原因的局部说明，不在全局关闭规则。验收记录见
[ShellCheck 修复](../testing/shellcheck-cleanup.md)。

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

当前检查覆盖主题重构的主页面、关键状态和弹窗，以及 Trader Sync 共享回归，覆盖桌面与
移动视口、系统 light／dark 输入下始终保持单一深色，以及根路径和 `/athena`。规则范围为 WCAG 2 A/AA、2.1 A/AA。
局部验证使用 `UI_ACCEPTANCE_GREP` 并在报告标记 filtered；最终无过滤运行先取消该变量。
具体版本、匹配数量与人工复核边界见[主题重构验收](../testing/web-ui-theme-refactor-acceptance.md)。
各场景保留 axe 原始结果；违规不会阻止后续场景及另一前缀收集，最终仍返回失败。
报告沿用 `.tmp/athena-ui-acceptance/<run-id>/`。

该入口补充现有 `make ui-acceptance` 的功能、键盘和视觉检查。自动扫描只能发现
部分无障碍问题，不能替代实际操作验收。首轮发现的现有问题在本次工具接入中记录，
不通过禁用规则、隐藏元素或批量修改业务代码将结果变成通过。

本机安装版本和首轮发现见 [AI 开发工具验收记录](../testing/ai-dev-tools-readiness.md)。
