# 第三方 Go 依赖漏洞修复验收记录

日期：2026-09-13。计划见 [第三方依赖修复计划](../superpowers/plans/2026-09-13-third-party-vulnerabilities.md)。本任务承接 [Go 1.27.1 升级](go-1.27.1-upgrade.md)，不重复升级 Go，不改变 Node、前端依赖、业务接口、授权或导出格式。

状态：代码、依赖、文档和必要验收已完成，最终独立复核通过。已从原仓库调用一次完成通知命令，SMTP 在第 1 次尝试接受，Make 退出 0；见 [完成通知日志](../../.tmp/third-party-vulnerabilities/completion-email.log)。

## 原因与修复

前次扫描的 9 项可达漏洞来自 5 个第三方模块。静态调用可达不等于已经证明可利用：头像上传实际解码外部图片，gRPC 的 HTTP/2 传输实际运行；Excelize 只用于生成 XLSX，当前没有外部工作簿导入；go-git 只被未注册的 Git 错误转换器引用，报告路径经过包初始化。

| 模块 | 旧版 → 新版 | 原可达漏洞 |
| --- | --- | --- |
| `golang.org/x/image` | v0.25.0 → v0.45.0 | GO-2026-6222、5061、4961 |
| `google.golang.org/grpc` | v1.80.0 → v1.82.1 | GO-2026-6061 |
| `golang.org/x/text` | v0.37.0 → v0.41.0 | GO-2026-5970 |
| `github.com/xuri/excelize/v2` | v2.10.1 → v2.11.0 | GO-2026-5960 |
| `github.com/go-git/go-git/v5` | v5.17.0 → 移除 | GO-2026-5496、4910、4909 |

`util/grpc/errors.go` 删除 Git 专用转换器、两种 Git 拦截器和导入；`internal/server/athena-server.go` 删除对应两行已注释注册。通用和 K8s 错误处理保留，不以硬编码 Git 错误字符串替代依赖。

必要的最小版本选择连带升级包括：`x/tools` 0.48.0、`x/mod` 0.38.0、`x/sync` 0.22.0（由 `x/text` 要求）；`x/net` 0.57.0（由 `x/tools` 要求）；`x/crypto` 0.54.0、`x/term` 0.45.0（由 `x/net` 等要求）；`x/sys` 0.47.0（由 `x/image` 等要求）；`genproto/googleapis/api` 的 20260414 版本（由 gRPC 要求）；`mscfb` 1.0.7（由 Excelize 要求）。tidy 删除 go-git 的无用间接依赖，补列现有 go-jsonnet 测试使用的 `sergi/go-diff`；没有执行整体依赖升级。

## 验证证据

任务证据目录：`.worktrees/third-party-vulns/.superpowers/sdd/2026-09-13-third-party-vulnerabilities/`。使用 Go 1.27.1、`GOTOOLCHAIN=local`。

- 完整 `go test -mod=readonly -count=1 ./...` 退出 0，包含最终新增测试；使用新构建的 Etherscan Manager，地址 `127.0.0.1:18101`，并设置 `E2E_LIVE=1`。真实上游只读交易查询通过；需要额外开关的外部限流探针按原设计跳过。见 `go-test-final.log`。
- 主服务、Gateway、两个索引器本地 Make 构建退出 0，见 `build-local.log`。
- `make codegen-local` 在一次性副本退出 0，末尾 vendor 清理完成，见 `codegen-local.log`；不交付无关生成文件差异。
- Excel 导出使用空、普通、Unicode 固定数据回读临时 XLSX，验证 16 列内容、唯一且活动的 Activity 表、冻结窗格及列宽。篡改冻结位置的临时变异检查按预期失败；生产导出实现未改动。
- 头像正常 JPEG、PNG、有损/扩展/无损 WebP 和格式、字节、尺寸、像素、截断、动画、画布与帧不一致检查通过。官方恶意 VP8L 样本另用无网络、只读 Docker 和 30 秒超时运行：旧版在 256 MiB 限制下分配 178,266,352 字节并按预期失败；新版在 128 MiB 限制下分配 7,360 字节并通过。默认测试跳过该高资源样本，显式运行需 `ATHENA_AVATAR_BOUNDED_WEBP_TEST=1` 和硬资源限制。原样样本、哈希及许可证在 `internal/avatarimage/testdata/`。
- Wallet gRPC 使用真实 TCP、实际 `CreateGRPC` 和项目客户端验证健康、正常状态查询、鉴权拒绝、处理中取消和超时；普通及 race 测试通过。超时严格要求客户端 `DeadlineExceeded`，服务端接受合法的取消/截止竞态。独立审查还执行了 3 次 race 回归，见 `grpc-regressions.md`、`final-review.md`。
- 三个 Docker 镜像均构建成功：`athena:third-party-vulns-validation`、`athena-bsc-transaction-indexer:third-party-vulns-validation`、`athena-bsc-swap-indexer:third-party-vulns-validation`。逐个提取实际二进制验证 Go 1.27.1、gRPC 1.82.1、相应 x/image/x/text 版本且无 go-git，提取容器已删除；见 `docker-build-results.json`、`binary-verification.json`。MinIO 镜像没有重建。
- 代码生成副本的 `go.mod/go.sum` 与交付源码完全相同；40 个 protobuf 描述符解压后逐字节相同，其他 Go 文本不变；生成 Mock 编译通过。Mock 模板、Swagger 版本和 CLI 文档存在生成漂移，其中 CLI 文档会移除原有手写开发说明并产生空白格式问题。全部留在一次性副本，未同步任何生成差异。见 `codegen-review.md`。
- golangci-lint 完成全仓分析，退出 1；本次重新采集的基线和最终报告均为 1,684 条诊断，按文件/规则/诊断文本比较新增 0 条。新测试曾有 1 条 gofumpt 提示，修正后复扫归零；不批量修复或屏蔽存量规则。前次 Go 升级记录的 1,683 条是历史扫描，本次使用同轮 1,684 条基线比较。见 `lint-comparison.json`、`lint-final.json`。
- 完整隔离 UI 验收：根路径和 `/athena` 各 33 项 ui-fixtures 与 10 项 live，共 86 项通过，临时服务、数据库和锁清理成功。见 [隔离报告](../../.worktrees/third-party-vulns/.tmp/athena-ui-acceptance/2026-09-13T06-40-02-659Z-3c5f272e/report.md)。live 的链、资料和 Telegram 边界为本地替身。
- 真实系统 Chrome smoke：会员和管理员应用壳 2 项通过，无基础设施失败。见 [真实 smoke 报告](../../.tmp/athena-ui-acceptance/2026-09-13T06-51-04-177Z-02a1d9ff/report.md)。此检查不证明 Google/Phantom 交互登录或全量业务回归。

## 原工作区交付与保留环境

仅将本次 16 个源码、测试样本和文档文件同步到 `/home/yege/work/athena`，每个目标先比对任务开始时的哈希/符号链接基线；保留其他任务全部未提交改动。没有提交、推送或远端部署。独立审查未发现阻断代码或规格问题。

原工作区的旧 `vendor/` 已按最终依赖重新生成。默认模块选择实测从原工作区 vendor 读取 x/image 0.45.0、gRPC 1.82.1、Excelize 2.11.0、x/text 0.41.0；未设置 `-mod=readonly` 的头像、Wallet、Excel 包测试通过。原工作区的四个本地二进制也使用该 vendor 重新构建，见 `.tmp/third-party-vulnerabilities/build-root-vendor.log`；当前模块文件已不会配合旧 vendor 使用。

确认进程归属后，从原工作区执行 `make stop`，再在同一终端激活 Node 24.14.1，运行 `GOTOOLCHAIN=local GOFLAGS='' ATHENA_RUN_PORT_CLEANUP=false make run`。关闭端口清理用于保留独立验证 Manager；应用必要端口由本次拥有的旧进程释放。原 `.env` 的 disabled-auth 开发模式保持不变。

- 会员入口：`http://localhost:4000/`；管理员入口：`http://localhost:4000/admin/`；API：`http://localhost:8080`。两个 realm 的 bootstrap 都确认已认证。
- 环境保留运行：会话 `29856`，Goreman PID `307846`；API PID `335347`、Wallet `335372`、Notification `335374`、Profit Sharing `335345`。这些进程工作目录均为原仓库，实际二进制模块元数据确认新 gRPC/x/image 且不包含 go-git；见 `.tmp/third-party-vulnerabilities/runtime-processes.json`。
- [运行日志](../../.tmp/third-party-vulnerabilities/runtime.log)。从 `/home/yege/work/athena` 执行 `make stop` 可停止本地应用并保留数据卷。本次没有删除开发数据卷，7 个无关容器 ID 保持不变。
- 只读 Etherscan 验证 Manager 在隔离工作区保留运行：`127.0.0.1:18101`，会话 `85279`，PID `168358`；日志位于任务证据目录的 `manager-runtime.log`。主工作区 `make stop` 不管理此独立进程；不再需要时可对该 PID 发送 TERM。之前任务的 8100/8101 验证进程未作改动。

## 剩余发现与范围

原工作区最终 `GOTOOLCHAIN=local GOFLAGS='' make vuln-check` 退出 0，原 9 项可达漏洞均消失，可达漏洞为 0；见 [最终复扫日志](../../.tmp/third-party-vulnerabilities/vuln-check-final.log)。详细扫描另有 2 项导入包级、4 项模块级发现；扫描器未显示当前默认构建调用对应漏洞函数。这不代表这些依赖不存在漏洞，也不覆盖自定义构建标签或未来新增调用。

| 层级 | ID | 当前模块与修复版本 |
| --- | --- | --- |
| 导入包 | GO-2026-5841 | `klauspost/compress` v1.18.6；修复 v1.18.7 |
| 导入包 | GO-2026-5158 | `go.opentelemetry.io/otel` v1.43.0；修复 v1.44.0 |
| 模块 | GO-2026-6355、6354 | `x/crypto` v0.54.0；修复 v0.56.0 |
| 模块 | GO-2026-6303 | `x/crypto` v0.54.0；修复 v0.55.0 |
| 模块 | GO-2026-5932 | `x/crypto` v0.54.0；本次扫描尚无修复版 |

上述非可达项按已确认范围保留，未过滤扫描结果或关闭功能。UI 对比度、存量 Shell/lint 问题仍属于其他任务。官方条目可通过 [Go 漏洞数据库](https://pkg.go.dev/vuln/) 按上述完整编号查询。
