# Go 1.27.1 升级验收记录

日期：2026-09-13。批准范围为 Go 工具链升级，不包含第三方业务依赖、UI 对比度及存量 Shell/lint 诊断修复。计划见 [Go 升级计划](../superpowers/plans/2026-09-13-go-1.27.1-upgrade.md)。

当前状态：配置、工具检查、源码、完整代码生成、镜像及浏览器验收均已完成。用户已批准 mockery 3.8.0 版本例外，正式安装器和原项目二进制均已升级，代码生成阻塞已解除。独立审查无阻断问题；完成邮件已发送，SMTP 在内置第 2 次尝试接受，Make 退出 0。

## 实现范围

- `/usr/local/go` 已通过官方归档校验并安装 Go 1.27.1；`go.mod` 仅修改 `go` 指令，业务依赖及 `go.sum` 不变。
- 五个 Go 构建阶段均固定 `golang:1.27.1@sha256:f44f6e88636cfb311f9ebace870ded69d943f227bb3cb27d32ffd84ea18c43ea`。
- MinIO Server/mc 源提交不变，默认标签增加 `-go1.27.1`，Make、Compose、启动及部署脚本、安装与运行文档已同步。
- golangci-lint 2.13.2、govulncheck 1.7.0、gopls 0.22.0 使用 Go 1.27.1 编译。govulncheck 安装和检查同时核对精确工具版本、编译 Go 及主机 Go，旧二进制会重建。
- mockery 按用户批准升级为 3.8.0，由现有生成工具安装器安装到 `dist/mockery`；原项目及隔离副本实际二进制均确认为 Go 1.27.1 编译的 3.8.0。
- 在 `.worktrees/go-1.27.1-upgrade` 实施；同步原仓库前逐一核对文件 SHA-256 基线，保留其他任务改动。未提交、推送或部署远端。

## 源码与工具

| 检查 | 实际结果 |
| --- | --- |
| 首次 `go test -mod=readonly ./...` | 失败：缺少默认 Etherscan Manager 服务，且真实查询测试要求 `E2E_LIVE=1`。保留失败日志。 |
| 准备服务后全套 Go 测试 | 退出 0；设置 `E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_MANAGER_ADDR=127.0.0.1:8101 GOTOOLCHAIN=local GOPROXY=off`。包含真实只读交易查询；需要额外开关的限流探针按原设计跳过。 |
| 主服务、Gateway、两个索引器 Make 构建 | 全部退出 0；四个二进制 metadata 均为 Go 1.27.1。 |
| AI 工具回归测试及实际安装/就绪检查 | 退出 0；覆盖旧编译器、缺失、元数据失败、精确版本及主机版本检查，保留扫描失败状态和日志。 |
| golangci-lint 配置校验 | 退出 0。 |
| golangci-lint 全仓分析 | 完成分析、退出 1；1,683 条诊断，未出现分析器崩溃或 Go 1.27 加载错误。未批量修复或禁用规则。 |
| gopls 代表文件检查 | 可运行；保留 `internal/server/athena-server.go:396` 的 tautological-condition 诊断。 |

核心日志位于 `.worktrees/go-1.27.1-upgrade/.superpowers/sdd/go-1.27.1-upgrade/`：`go-test.log`、`go-test-with-services.log`、`go-build.log`、`golangci-lint.log`、`tools-report.md`。

## 镜像与 MinIO

以下五个镜像构建均退出 0。逐个创建临时容器、提取实际运行二进制并执行 `go version -m`，全部确认 Go 1.27.1；提取用容器已清理。

| 镜像 | 二进制 |
| --- | --- |
| `athena:go1.27.1-validation` | `athena` |
| `athena-bsc-transaction-indexer:go1.27.1-validation` | `athena-bsc-transaction-indexer` |
| `athena-bsc-swap-indexer:go1.27.1-validation` | `athena-bsc-swap-indexer` |
| `athena-minio:9e49d5e7a648-go1.27.1` | `minio` |
| `athena-minio-mc:7394ce0dd2a8-go1.27.1` | `mc` |

真实本地启动使用新 MinIO Server/mc 标签，成功初始化私有 `athena-account-avatars` 桶及应用策略。`minio --version` 确认源码提交 `9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a` 与 Go 1.27.1。未重置或删除开发数据卷。

构建日志为上述证据目录中的 `docker-main.log`、`docker-transaction.log`、`docker-swap.log`、`docker-minio.log`；二进制校验见 `image-verification.json`。

## 漏洞复扫

`make vuln-check` 完成扫描，govulncheck 原生退出码为 3，Make 返回 2。可达漏洞 ID 从 37 项变为 9 项，来自 5 个第三方模块；新的报告中没有标准库漏洞路径。

旧报告有 28 个 ID 包含标准库路径，另有 10 个 ID 包含第三方路径，其中 `GO-2026-5026` 同时涉及 `net/http` 和 `x/net`，不能将模块分组直接相加。此次按 ID、Found in 和调用路径对照，不以固定数量作为验收阈值，也不把静态可达性当作已证实可利用。

| 第三方模块 | 剩余可达 ID | 报告中的最高修复版本 |
| --- | --- | --- |
| `golang.org/x/image` | GO-2026-6222、5061、4961 | v0.45.0 |
| `google.golang.org/grpc` | GO-2026-6061 | v1.82.1 |
| `golang.org/x/text` | GO-2026-5970 | v0.39.0 |
| `github.com/xuri/excelize/v2` | GO-2026-5960 | v2.11.0 |
| `github.com/go-git/go-git/v5` | GO-2026-5496、4910、4909 | v5.19.1 |

这些业务依赖未在本任务升级；修复版本来自本次扫描，并未完成其兼容性验收。报告另有 10 项导入包及 30 项所需模块中的发现，未显示当前代码调用对应漏洞路径。原始报告与逐 ID 对照保存在 `vuln-check.log`、`vuln-comparison.json`。

## 代码生成兼容修复

全部 Go 生成工具均已用 Go 1.27.1 重编译。首次使用 mockery 3.6.1 时，隔离副本中的 `make codegen-local` 在 mockgen 失败：`package "io" without types`。旧 Go 编译的同版 mockery 也复现，项目 `util/io` 包自身测试通过。

逐项执行 `gogen`、`protogen`、`sqlc-local`、`clientgen`、`clidocsgen` 和额外 `abigen-local` 均通过；`clidocsgen` 首次因副本缺少 `ui/dist` 失败，补齐已有嵌入资产后通过。40 个 protobuf 文件仅有 gzip 字节变化，解压后的 descriptor 40/40 相同。无关生成结果未同步原仓库。

用户批准后，正式安装器固定 mockery 3.8.0；原项目执行 `GOTOOLCHAIN=local GOFLAGS=-mod=readonly bash hack/installers/install-codegen-go-tools.sh` 退出 0，`dist/mockery version` 和编译 metadata 分别确认 3.8.0、Go 1.27.1。日志为 `.tmp/go-1.27.1-upgrade/install-codegen-tools-final.log`，metadata 见同目录 `mockery-final-metadata.txt`。

隔离副本同步最终安装器并真实重装全部生成工具后，再次执行完整 `make codegen-local` 退出 0、末尾 vendor 清理成功，`go test -mod=readonly ./util/io/mocks` 通过。最终 Mock 生成结果与候选验证相同，差异为测试辅助函数标记及 `interface{}` 到 `any` 的等价模板变化。生成结果未带回交付源码，项目业务依赖及 `go.sum` 不变。完整证据为 `codegen-report.md`、`install-codegen-go-tools-final.log`、`codegen-local-final-v3.8.0.log`。

## 浏览器与保留的本地环境

- 完整隔离验收：根路径和 `/athena` 各 33 项 ui-fixtures、10 项 live，共 86 项通过；临时 PostgreSQL、harness 和锁清理成功。证据：[隔离验收报告](../../.worktrees/go-1.27.1-upgrade/.tmp/athena-ui-acceptance/2026-09-13T05-51-23-002Z-caca1925/report.md)。live 的链、资料及 Telegram 边界仍为本地替身。
- 真实 smoke：`http://localhost:4000`，会员及管理员应用壳通过，无基础设施失败。证据：[smoke 报告](../../.tmp/athena-ui-acceptance/2026-09-13T05-57-43-141Z-22c89c52/report.md)。不包含 Google/Phantom 交互登录或全量业务验收。
- 从原仓库 `/home/yege/work/athena` 在同一终端激活 Node 24.14.1，再运行 `GOTOOLCHAIN=local GOFLAGS=-mod=readonly ATHENA_RUN_PORT_CLEANUP=false make run`。关闭端口清理仅用于保留独立 Etherscan 验证进程；启动前已确认本项目必要端口空闲。
- 原 `.env` 的 disabled-auth 开发模式保持不变；会员 `local-user` 和管理员 `local-admin` 分别 bootstrap 成功。实际 API Server、Wallet、Notification、Profit Sharing 进程均确认工作目录为原仓库、二进制 Go 为 1.27.1。
- 开发环境保留运行：会话 `30485`，Goreman PID `4135663`；[运行日志](../../.tmp/go-1.27.1-upgrade/runtime.log)，[进程校验](../../.tmp/go-1.27.1-upgrade/runtime-go-processes.json)。会员入口 `/`，管理员入口 `/admin/`。从同一仓库执行 `make stop` 可停止并保留数据卷。
- 独立测试 Manager 使用 8100/8101，分别由会话 `44377`/`20180` 启动，日志位于 worktree `.tmp/go-1.27.1-upgrade/`。8101 使用真实配置并通过只读查询；8100 仅做健康/状态/参数验证，未测试上游业务。

现有 UI 对比度、Shell 诊断及第三方依赖发现保持原修复边界。已从原仓库调用一次完成通知命令；首次连接 EOF 后内置重试成功，SMTP 接受邮件。证据：[完成通知日志](../../.tmp/go-1.27.1-upgrade/completion-email.log)。
