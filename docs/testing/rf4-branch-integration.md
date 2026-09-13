# rf4 两分支集成验收

日期：2026-09-13（Asia/Shanghai）。本记录对应用户批准将以下两个分支合入 `rf4`，实施范围见[集成计划](../superpowers/plans/2026-09-13-rf4-branch-integration.md)。

| 对象 | 验证基线 |
| --- | --- |
| 合并前 rf4 | `61f4abc2` |
| Trader Sync 源分支 | `codex/trader-sync-independent`，`5228d78b` |
| Trader Sync 合并提交 | `355c7dce`，保留源分支提交历史 |
| Solana 源分支 | `codex/solana-discovery`，`53febffe` |
| 本次最终树 | 在上述 Trader Sync 合并之上集成 Solana，并包含下列交叉修复；本记录随 Solana merge commit 提交 |

## 集成结果与修复

- 默认 `make run` / `make stop` 保持 managed 六服务：Trader Sync、API Server、Notification、Wallet、Profit Sharing 和 UI。Solana 仅经显式 `solana-discovery` / `solana-preview` profile 运行。
- 两分支分别使用了账户迁移 `000002`。保留 Trader Sync runtime control 的 `000002`，将 Solana 授权迁移改为 `000003`，重新生成账户 schema contract，保留两组约束和权限行为。
- Solana DSN 优先使用 `ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN`，其次为 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`；移除旧 API DSN 回退和隐式数据库默认值，缺失时拒绝启动。
- Solana preview 要求显式专用 DSN，先调用独立账户迁移命令 `up` / `verify`，成功后直接启动 API；移除对已删除 Trader Sync 启动包装器的调用。测试覆盖迁移失败时不启动 API。
- managed API 配置白名单补入 Solana 服务地址与内部令牌，未向 API 注入扫描器配置。修复自动合并引入的重复 Go import。
- 同步长期需求、设计和运行文档；不改变历史源分支验收材料的适用范围。

独立审查发现原 Solana 预览账户库的版本 `000002` 含义与 rf4 不同。最终处理范围是保留原库和原 worktree，使用新库验证 rf4；文档和显式 DSN 已消除误用旧库的默认路径，审查复核无剩余阻断项。这不表示旧库兼容新 schema。

## 验证结果

以下均为合并后实现的实际验证，最终命令退出码为 0。Go 使用本地工具链与模块缓存，前端使用 Node `v24.14.1`、Yarn `1.22.22`。

| 验证 | 结果与证据 |
| --- | --- |
| `go test ./internal/... ./cmd/... ./util/... ./pkg/... ./tools/... -count=1` | 35 个有测试包通过。设置独立 `SOLANA_TEST_POSTGRES_DSN`，实际执行 Solana PostgreSQL 测试。日志 `.tmp/rf4-integration/merged-go.log`。 |
| 账户及存储集成测试（`-tags=integration`） | `internal/accountstate/schema` 25.301s、`schema/catalog` 1.129s、`internal/tradersync/store` 140.728s、`internal/notification/store` 46.392s；修复重复 import 后 `internal/accountstate/store` 单独重跑 13.564s，通过。使用本次专用临时 PostgreSQL。 |
| 运行入口与配置 | Go test / vet 覆盖 Solana commands、devruntime、local-runtime；`python3 hack/solana-integration_test.py` 5 项通过；shell-local、ShellCheck 和 Solana Goreman 生命周期 fixture 测试通过。未启动真实 Solana 采集。 |
| 前端 Jest | 31 套、361 项通过；`.tmp/rf4-integration/merged-jest.log`。 |
| 前端 lint / build | 类型检查、ESLint 及 12 项 lint 配置测试通过，构建通过；`merged-lint.log`、`merged-build.log`。现有大于 500 kB bundle 提示仍存在。 |
| `make solana-discovery-build` | 成功生成 `.tmp/bin/athena-solana-discovery`；`.tmp/rf4-integration/solana-build.log`。 |
| `make sqlc-local` | 成功，未产生额外 sqlc 差异；`go.mod` / `go.sum` 保持不变。 |
| schema contract 与 `make protogen` | 成功。protogen 的 39 个额外差异仅为 descriptor gzip 字节：逐文件比较解压后的完整 descriptor 与其余源码均一致，恢复这些压缩差异。记录 `protogen.log`、`protogen-semantic-check.json`，Swagger 生成版本为 `dev`。 |
| 隔离浏览器验收 | `make ui-acceptance`：根路径与 `/athena` 路径各 33 项 UI fixture、10 项 live，共 86 项通过，临时环境清理通过。 |
| 新 rf4 环境真实 smoke | `make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://localhost:34000`：会员和管理员应用壳 2 项通过，使用系统 Google Chrome。 |
| 真实 API 边界 | 会员和管理员 bootstrap 均返回 200；Trader Sync subscriptions 返回 200；暂停的 Solana status 返回预期 503，随后 bootstrap 仍为 200。记录 `live-api-checks.json` 及两个 bootstrap JSON。 |

本节未逐项指定目录的日志位于 `.tmp/rf4-integration/`。账户与运行入口子任务的原始输出保存在会话工具记录，未保存为独立日志文件；其中账户集成首轮为 session `97503` / chunk `6fa609`，重复 import 修复后绿色重跑为 session `30004` / chunk `b92813`，sqlc 为 chunk `68fbff`，contract 生成为 chunk `cb0634`。

浏览器原始证据：

- 隔离验收：`.tmp/athena-ui-acceptance/2026-09-13T15-20-23-350Z-82f5a0af/`，含 `run.json`、`report.md` 和各阶段日志。
- 真实 smoke：`.tmp/athena-ui-acceptance/2026-09-13T15-23-18-930Z-44a06036/`，含 `run.json`、`report.md`、`smoke/runner.log`。

隔离 live 验收使用真实产品组件与临时 PostgreSQL，链、资料和 Telegram 使用本地替身。真实 smoke 证明已运行环境的会员与管理员应用壳；未声称它覆盖所有业务流程。Solana 主网采集与原预览保持暂停，本次没有重新执行其主网验收。历史证据见 [Solana 验收](solana-discovery.md)和 [Trader Sync 独立服务验收](trader-sync-independent-service-acceptance.md)。

失败及修复过程：账户迁移最初因重复版本失败，改号后暴露 contract 不匹配，生成 contract 后通过；账户存储重复 import 单独修复并重跑通过。首次宽范围 Go 验证遇到旧 ignored vendor 缺少测试依赖，且进入无关外部 E2E，停止本次测试进程后改用模块缓存并限定上述产品代码目录。新 runtime 首次端口预检失败，后续确认目标端口可绑定并重试，完整启动和 smoke 均通过；保留 `runtime-initial-port-failure.log`，不据此推断具体端口占用原因。

## 保留的本地验收环境

- 仓库：`/home/yege/work/athena`，分支 `rf4`，实例 `rf4-integration`。
- 会员入口：<http://localhost:34000>；管理员入口：<http://localhost:34000/admin/>。
- 启动命令：`make run INSTANCE=rf4-integration ENV_FILE=.tmp/rf4-integration/runtime.env`。
- 持久会话：`60535`；supervisor PID `1672008`。验收时 API `1674287`、Trader Sync `1674182`、Notification `1674211`、Wallet `1674234`、Profit Sharing `1674198`、UI `1674265` 均 ready。
- 运行日志：`.tmp/rf4-integration/runtime.log`；实例日志与状态：`.run/instances/rf4-integration/`；验收状态快照：`.tmp/rf4-integration/runtime-status.json`。
- 本地 Notification 使用 `127.0.0.1:34909` Telegram 替身，脚本 `.tmp/rf4-integration/telegram-fixture.py`，会话 `13138`，PID `1649171`。替身随环境保留，本次未用它向外发送 Telegram 消息。
- 从上述仓库执行 `make stop INSTANCE=rf4-integration` 停止本次 managed 实例，保留其开发数据库。独立 Telegram 替身在不再需要该环境时，可先确认该 PID 仍属于上述脚本，再结束该进程。

验收使用的额外临时 PostgreSQL 容器 `athena-rf4-merge-test-pg-20260913` 已核对任务/checkout 标签及无活动客户端后清理，记录 `.tmp/rf4-integration/test-postgres-cleanup.json`。隔离浏览器自建数据库也已清理。原根目录服务、Trader Sync 源 worktree 的服务以及原 Solana 库均保持原样，未执行 reset。

## 服务规范映射

参照[服务开发规范](../developer-guide/service-development-standards.md)：

| 规则 | 本次证据 |
| --- | --- |
| SDS-R1、R2、R4 | Trader Sync 与 Solana 独立服务，API 通过客户端调用；Go RPC/config/auth 测试、实际 Trader Sync 查询及 Solana 停机故障隔离检查。 |
| SDS-R3、R5 | 独立 Solana 构建、profile 路由及生命周期 fixture；managed 六服务实际启动、资源归属记录及停止命令；旧服务和数据保留。 |
| SDS-R6 | 合并账户迁移与 schema contract，账户授权、存储及事务集成测试；旧 Solana 数据迁入不在本次范围。 |
| SDS-R7、R8 | [集成计划](../superpowers/plans/2026-09-13-rf4-branch-integration.md)、[运行说明](../developer-guide/running-locally.md#solana-discovery-local)、本记录及保留的历史验收。 |

本次为本地分支合并与集成验证，未执行远程 push 或生产部署。完成通知在最终合并提交与祖先关系核对之后按项目规则执行，其实际发送结果以交付回复为准。
