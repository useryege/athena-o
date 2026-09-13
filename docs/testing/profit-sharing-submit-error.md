# 分润提案提交错误传递修复验收

日期：2026-09-13。状态：已完成；实现、真实数据库回归、lint 对比、独立审查及交付均已通过。范围为 [Go lint 后续方案](../superpowers/plans/2026-09-13-go-lint-followup.md)的首批，仅修复历史诊断 `G0554`；其他批次未实施。

## 根因与实现边界

[SQLStore 提交路径](../../internal/profitsharing/store/operations.go#L285)在 submit 分支使用 `participants, err :=` 创建内层错误变量，提交查询写入该变量，而分支外检查外层错误。零行结果可能被当成成功；SQL 语句错误可能只在 Commit 时返回，丢失原始查询错误。

本批按已批准方案将参与者列表错误独立命名为 `listErr`，使提交结果进入既有 `pgx.ErrNoRows → ErrRevisionConflict` 映射及原始错误包装。不改变公共 API、数据库结构、SQL、生成文件、依赖或 lint 配置。

## 测试环境与方法

- 工作区：`.worktrees/profit-sharing-submit-error`，分支 `codex/profit-sharing-submit-error`；以当前根目录源码及既有未提交改动为基线，不提交或推送。
- Go 1.27.1；golangci-lint 2.13.2（由 Go 1.27.1 编译）；`GOTOOLCHAIN=local`、`GOFLAGS=-mod=readonly`。
- PostgreSQL 16 使用本机已有镜像 `sha256:be01cf82fc7dbba824acf0a82e150b4b360f3ff93c6631d7844af431e841a95c`；任务容器 `athena-profit-sharing-submit-error-test`，本次端口 `127.0.0.1:60431`，数据位于容器 tmpfs。该端口是本次运行记录，不是项目固定配置。
- 使用 `ATHENA_TEST_PG_ADMIN_DSN`（必须含 `sslmode=disable`），由 `pgtest.New` 为每例创建并清理独立数据库；直接调用真实 SQLStore，故障触发器只存在于测试数据库。
- 每例包含 collecting round、5 个 UUID 参与者、revision=1 的 draft proposal 和 5 条完整分配，各占 2,000 基点。

## 验证记录

实现前 `go test ./internal/profitsharing/...` 退出 0；这些包原来没有测试，结果仅证明基线编译通过。完整 lint 基线为 1,684 条，与上一轮历史快照相比新增 0、消失 0；扫描期间 802 个 Go/模块/配置文件哈希不变。

### 红绿回归

先运行两项故障回归，旧代码退出 1：零行用例错误返回非 nil 的零值提案；语句错误用例无法通过 `errors.As` 取得原始 `*pgconn.PgError`，只返回 `commit unexpectedly resulted in rollback`。随后将参与者列表错误改名，执行完整回归：

```bash
GOTOOLCHAIN=local GOFLAGS=-mod=readonly go test -v -tags=integration -count=1 \
  ./internal/profitsharing/store -run 'TestSQLStore(Submit|Reopen)Proposal'
GOTOOLCHAIN=local GOFLAGS=-mod=readonly go test ./internal/profitsharing/...
```

命令实际在上述专属数据库 DSN 下运行。五项集成测试全部执行、通过，无跳过；普通分润包命令退出 0。普通命令不含 integration 标签，其结果仅证明相关包编译通过。

| 用例 | 断言与结果 |
| --- | --- |
| `TestSQLStoreSubmitProposalNoRows` | 测试触发器对 submitted 更新返回 NULL；返回 nil proposal 和 `ErrRevisionConflict`，提案状态、版本、提交时间与全部条目不变。通过。 |
| `TestSQLStoreSubmitProposalQueryError` | 测试触发器抛出 `P0001` 和固定消息；返回 nil proposal、保留原始 PgError 和提交状态上下文，持久数据不变。通过。 |
| `TestSQLStoreSubmitProposalSuccess` | 返回正确提案与全部条目；数据库状态 submitted、revision=2、设置提交时间。通过。 |
| `TestSQLStoreReopenProposalSuccess` | 正常提交后重新打开，状态 draft、revision=3、清空提交时间，全部条目保留。通过。 |
| `TestSQLStoreSubmitProposalRevisionConflict` | expected revision 不匹配，返回原冲突错误，数据不变。通过。 |

这是受控故障复现和真实存储事务回归，不是已发生业务事故的证据；不涉及浏览器、远端业务或全系统端到端验收。

### lint 对比与变更范围

使用原 `.golangci.yaml`、默认构建条件、原规则及以下命令进行完整前后扫描：

```bash
GOTOOLCHAIN=local GOFLAGS=-mod=readonly golangci-lint run \
  --timeout=10m --modules-download-mode=readonly \
  --output.json.path=<本轮证据目录>/lint-after.json \
  --output.text.path=<本轮证据目录>/lint-after.log ./...
```

- 扫描前 1,684 条，扫描后 **1,683 条**。按文件、规则、诊断文本及重复数量比较，新增 0、消失 1；唯一消失项为 `operations.go` 的 `ineffassign: ineffectual assignment to err`（历史 `G0554`）。
- 额外使用 `--build-tags=integration` 扫描 `./internal/profitsharing/store/...`，新增测试文件诊断为 0；包内仍有 4 条基线提示（3 条 goimports、1 条 perfsprint），未修改。
- 两种 lint 扫描退出 1 均来自存量诊断，无类型加载失败、分析器崩溃或超时；`gomodguard` 弃用警告保持原状。此处不宣称全仓 lint 已清零。
- 802 个既有 Go/模块/配置文件中，仅 `operations.go` 的 3 行发生修改；另新增一个集成测试文件。SQL、生成文件、模块依赖和 lint 配置未变。

## 证据与交付

- [红测](../../.tmp/profit-sharing-submit-error/red.log)、[最终五项回归](../../.tmp/profit-sharing-submit-error/green.log)、[普通包验证](../../.tmp/profit-sharing-submit-error/package-tests.log)。
- [实施前扫描](../../.tmp/profit-sharing-submit-error/lint-before.json)、[修复后扫描](../../.tmp/profit-sharing-submit-error/lint-after.json)、[诊断差异](../../.tmp/profit-sharing-submit-error/lint-comparison.json)、[带 integration 标签的扫描](../../.tmp/profit-sharing-submit-error/lint-integration.json)。

[代码任务审查](../../.tmp/profit-sharing-submit-error/task-1-review.md)的 Spec 与 Quality 均通过，无 Critical、Important 或 Minor 问题。基线比对后仅回写本次 5 个文件至根工作区；回写后的[五项数据库回归](../../.tmp/profit-sharing-submit-error/root-integration.log)全部通过，无跳过。

[资源清理记录](../../.tmp/profit-sharing-submit-error/database-cleanup.log)确认 `pgtest` 创建的数据库剩余 0，本任务专属容器已移除，其他既有运行容器保持运行。源码修改未提交或推送；worktree 保留用于复核。[最终交付审查](../../.tmp/profit-sharing-submit-error/final-review.md)通过，无 Critical、Important 或 Minor 问题。

本机日志存放在 `.tmp/profit-sharing-submit-error/`，由 Git 忽略；清理后需重新运行才能获得原始日志。历史 1,684 条分类 JSON 保留，不因本次修复重写。

按仓库规范从根目录调用一次完成通知，命令退出 0，SMTP 首次尝试接受邮件；见[通知日志](../../.tmp/profit-sharing-submit-error/completion-email.log)。
