# BSC／Sports 清理整合到 rf4

2026-09-16 用户要求先整合到 `rf4`。目标工作区为 `/home/yege/work/athena`，采用本地非快进合并，保留两条分支的历史。

## 范围与基线

- 目标父提交：`4a7773910603d149aab1f31c7eb75fa722806d65`（`rf4`）。
- 来源父提交：`595ae2a0eaa0cb47c06b6fa027f49265f09d0d13`（`codex/module-removal-cleanup`）。
- 共同基线：`264d0dc1fd524dc23b236f939da1d343572bd030`。
- `rf4` 独有的 Worm Markets 退役／Trading 目录承接设计与实施计划保留。本次整合不执行该新计划，不发布远端；`47.245.181.189` 继续按用户要求跳过。

八个冲突文件均为文档。合并后明确区分：BSC／Sports 清理已执行；Markets 退役、十一应用编排及访问开关尚未实施。账户现行为 Sports 迁移 `000004` 和八项模块，后续 Markets 迁移使用下一空号（当前预期 `000005`），完成后才变为七项。独立审阅发现的旧迁移分支与双服务目标残留已修正。

生产源码与清理分支完全一致。新增的一行测试修正位于 `pkg/apiclient/account/access_json_test.go`：旧 JSON fixture 仍包含已删除的编号 2／3／7，导致四种权限标志组合均报 `unknown account data module "2"`；先独立复现，再更新为正式八项模块，保留全部 JSON 往返与独立标志断言。

原根目录两份未跟踪的退役索引器 `.env.bsc-*` 配置移入本次忽略证据目录 `retired-local-configs/`，保留内容、原路径和 SHA256；不提交为活动配置。来源 worktree 与其忽略的验收证据继续保留。

## 本次验证

证据根目录：`/home/yege/work/athena/.superpowers/module-removal-integration/`。

| 验证 | 结果与证据 |
| --- | --- |
| 本地 Go 包 | `go test ./cmd/... ./common/... ./internal/... ./pkg/... ./tools/... ./util/... -count=1` 退出 0；37 个包执行测试、178 个包仅编译，`logs/go-local-final.log` |
| PostgreSQL 集成 | 独立 PostgreSQL 16，账户 schema／store／txgate、通知 store、退役 CLI 共六个有测试包通过，`logs/go-integration.log` |
| 命令构建 | `go build ./cmd/... ./tools/retire-sports-notifications`，`logs/go-build.log` |
| 前端 | Jest 37 套／417 条通过；lint 与生产构建退出 0，`logs/ui-jest.log`、`ui-lint.log`、`ui-build.log` |
| 部署脚本 | `hack/production-compose_test.sh` 与 `hack/deploy-scripts_test.sh` 退出 0，Compose／脚本日志保留 |
| 真实本地环境 | 从整合中的根目录执行 `make run INSTANCE=rf4-module-removal-20260916 ENV_FILE=.superpowers/module-removal-integration/local.env`；六程序 ready，会员／管理员 bootstrap 均返回正确身份与八项模块，`runtime-ready.json`、`bootstrap.json` |
| 系统 Chrome smoke | `http://127.0.0.1:61821` 的会员／管理员应用壳，两条通过、无失败／跳过；报告 `.tmp/athena-ui-acceptance/2026-09-16T03-39-43-542Z-a902dc45/report.md` |

首次扩大全仓 `go test ./...` 未通过，原始结果保存在 `logs/go-all.log`：两个 Etherscan E2E 包分别缺少本地 `127.0.0.1:8100` 服务及 `E2E_LIVE=1`；未把这次运行记为全仓通过，也未为本次整合开启外部 live E2E。另有 `TestReadProcessCapturesKernelIdentity` 一次读取到空 RunID，未改动的测试单独重复 20 次及最终本地 Go 包回归均通过，记录 `logs/runtime-identity-repeat.log`。账户 JSON fixture 失败按上文修正。

本次 smoke 只证明整合工作区的应用壳与 bootstrap；未重新执行完整隔离 UI／Worm 实际读取。相同生产代码在原实施任务的 1002 条浏览器回归及 Worm 只读证据见[原验收记录](module-removal-cleanup-acceptance.md)，与本次验证分开记账。

## 环境收尾

原生实例 namespace 为 `2b340a98dc115747e5f565ba7178e725`。从根目录使用相同 `INSTANCE`／`ENV_FILE` 执行 `make stop`；独立 PostgreSQL 测试容器及 Telegram 本地替身也已停止。运行器 state 为 `stopped`，记录的 23 个进程身份退出，新建四个容器停止，61812 与 61819–61825 共八个端口释放。数据库、数据卷、日志和浏览器报告保留，未执行 reset。

任务前 40 个容器的运行／停止状态全部保持，其他任务服务未接管。精确 ID、挂载、前后状态、进程与端口检查见 `cleanup-verification.json`。本次没有要求继续运行的临时环境。

本次整合累计执行超过十分钟，按根 `AGENTS.md` 在交付前使用默认 `.env` 发送一次结果通知；实际命令退出与邮件结果保存在同一证据目录，不复用原实施任务的通知记录。
