# rf4 分支集成实施计划

> 执行方式：按 `subagent-driven-development` 分工推进运行入口、账户 schema 和文档整合，主代理负责集成验证、提交及交付。

**目标：** 将用户已批准的 `codex/trader-sync-independent` 和 `codex/solana-discovery` 合入 `rf4`，保留两分支实现并修正它们共用的运行和账户边界。

**架构：** Trader Sync 继续为独立 gRPC 服务，默认 managed 全栈保持六服务。Solana 使用显式局部 profile 与独立发现服务，API 仅代理查询；账户 schema 由独立迁移命令准备，业务进程只验证。

**技术栈：** Go、gRPC、PostgreSQL、React、Bash 与现有本地运行器。

**设计依据：** [Trader Sync 独立服务](../../design/trading/trader-sync-activity-alerts.md)、[本地运行编排](../../design/development-runtime/local-runtime-orchestration.md)、[Solana 设计总览](../../design/solana-intelligence/README.md)。本次合并决定已由用户批准，不新增设计审批。

## 约束

- Solana 采集与预览保持暂停，原候选、游标和待补全队列保留；测试不恢复主网扫描。
- 原 Solana 预览账户 schema 与 `rf4` 不兼容，原库继续由保留的原 worktree 和版本管理；本次使用新数据库验收，不运行旧库的 `up` / reset，不新增历史兼容路径。旧数据转入 `rf4` 需单独明确数据迁移范围。
- 默认 managed 全栈为 Trader Sync、API Server、Notification、Wallet、Profit Sharing 和 UI，不把 Solana 隐式加入。
- 共用账户库统一为 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`；保留不同业务的事务归属、连接池所有权和权限复查。
- Solana preview 使用专用数据库，先执行账户 schema `up` / `verify` 后直达 API，不经 Trader Sync 旧包装器。
- 保留源分支历史验收和 spec；长期文档同步当前集成事实，新验证单独记录。
- 本次不扩大 Solana 首版或全站 UI 视觉确认范围，不改变已确认的 Nansen 单一深色目标。

## 任务一：合并与共用入口整合

涉及 `Makefile`、`Procfile`、`Procfile.solana-preview`、Solana 命令配置及运行脚本。

- [x] 检查工作区与分支状态，按用户授权完成两个分支在 `rf4` 中的代码整合。
- [x] 保留 managed 六服务入口与显式 Solana profile，解决入口文本冲突。
- [x] 核对 Solana 数据库回退、preview schema 准备和 API 启动顺序；用相应 shell 行为测试及配置测试覆盖交叉影响。

## 任务二：账户 schema 与消费者一致

涉及 `internal/accountstate/store/migrations/`、`internal/accountstate/schema/`、账户授权代码及测试、生成消费者。

- [x] 保留 Trader Sync runtime control 与 Solana 授权两组 schema 变更，消除重复迁移编号并核对验证版本。
- [x] 核对 Solana READ/NONE、完整账户矩阵及普通账户默认 NONE，验证已有 Trader Sync schema 上的升级结果。
- [x] 执行受影响 Go 包、真实 PostgreSQL schema/事务测试和对应 UI 契约测试，记录命令、退出结果与限制。

## 任务三：文档、验收与交付

涉及 `docs/developer-guide/running-locally.md`、Solana 长期需求和设计、需求/设计索引及本次集成验收记录。

- [x] 解决运行文档冲突，统一默认六服务、显式 Solana profile、共享 DSN 和 schema 准备说明。
- [x] 将长期文档从“仅源分支实现”更新为 `rf4` 集成事实，保留采集暂停、历史验收范围与后续未决研究规则。
- [x] 核对改动文件链接、冲突标记和 `git diff --check`；验证受影响构建、运行入口和必要真实环境验收，保持 Solana 停机。

本次实现与验证结果见[集成验收记录](../../testing/rf4-branch-integration.md)。Go 及相关 PostgreSQL 集成测试、31 套 361 项前端测试、静态检查与构建、Solana 独立构建、86 项隔离浏览器测试和新环境的 2 项真实 smoke 已通过；原 Solana worktree、数据库、游标与待补全队列保持原样。

交付顺序：主代理在核对最终差异与验证证据后完成最后的合并提交，检查两个源分支均为 `rf4` 的祖先及工作区状态，再按项目规则发送一次完成通知。通知在最后合并提交后执行，发送结果以命令退出结果和最终交付说明为准。
