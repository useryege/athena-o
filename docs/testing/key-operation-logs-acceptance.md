# 用户关键操作日志：实施与验收记录

## 当前事实

- 工作分支：`codex/key-operation-logs`
- 独立 worktree：`/home/yege/work/athena/.worktrees/key-operation-logs`
- 当前版本（Task 8 UI 状态兼容修复）：`66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0`；其父提交包含 Task 1–7 的已审阅实现。
- 根仓库及其他 worktree 未被本任务修改。父任务文档仍保留为未提交差异，未被本次产品提交覆盖。
- 设计依据：[`docs/superpowers/specs/2026-09-18-key-operation-logs-design.md`](../superpowers/specs/2026-09-18-key-operation-logs-design.md)、其附录、[`docs/requirements/observability/key-operation-logs.md`](../requirements/observability/key-operation-logs.md)、[`docs/design/observability/operation-logs.md`](../design/observability/operation-logs.md) 和 [`docs/superpowers/plans/2026-09-18-key-operation-logs.md`](../superpowers/plans/2026-09-18-key-operation-logs.md)。

## 已实现内容

Task 1–7 已实现并分别完成独立审阅：类型化事件与 74 个动作／98 个采集入口、持久收件箱与幂等投影、查询／分页／管理员鉴权协议、37 个 gRPC 入口、61 个 HTTP／认证入口、管理员列表与详情 UI，以及独立 operation-log runtime、schema owner、迁移、TLS 健康探针和最小依赖。审阅报告分别位于 `.superpowers/sdd/2026-09-18-key-operation-logs/task-{1,2,3,4,5,6,7}-*review.md`（Task 1/2 的报告名称按目录实际文件为准）。

Task 8 发现并修复一个前端事实映射问题：capture status 的 `persistenceReachable`、`confirmedEvents`、`inFlightEvents` 和 `lastFailureCode` 可能以 JSON scalar 返回，页面此前只读取 `.value` wrapper。`66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0` 增加兼容解包函数、回归测试并修正页面展示。

## V01–V18 结论

| 编号 | 当前结论 | 证据与边界 |
| --- | --- | --- |
| V01 | 专项通过；真实全入口回归仍有限 | Task 1 catalog/allowlist 测试、Task 4 的 37 个 gRPC hook 复审、Task 5 的 61 个 HTTP/auth route 复审；目录为 74 动作／98 入口。 |
| V02 | 专项通过；未覆盖所有生产重试故障注入 | recorder／producer 重试、gRPC-Web 去重和 HTTP middleware 单一观察由 Task 1/4/5 测试与复审覆盖。 |
| V03 | 专项通过 | session、API key、development／Phantom 身份绑定及 stale credential 排除由 Task 4/5 测试与复审覆盖。 |
| V04 | 专项通过 | 认证、管理员／模块准入、Origin、再认证和 CAS 保留业务结果的相关 handler 测试与 Task 4/5 复审通过。 |
| V05 | 专项通过 | 注册 commit/publish/login 分段、取消／revoke 失败和 PARTIAL 映射见 Task 5 报告与 `task-5-race.log`。 |
| V06 | 专项通过 | SUCCEEDED、ACCEPTED、FAILED、DENIED、PARTIAL、UNKNOWN、写回失败、panic 和原子 wallet selection 事实由 Task 1/4/5 测试与复审覆盖。 |
| V07 | 专项通过 | event allowlist、detail 限制、敏感 token／私钥／正文排除和 UI detail 清理由 Task 1/4/5/6 复审覆盖。 |
| V08 | 通过 | 真实任务 PostgreSQL 上的 schema、入箱幂等、乱序、savepoint 隔离和身份折叠证据：`task-2-integration-final.log`、`task-2-store-green.log`、`task-2-race-final.log`。 |
| V09 | 通过 | projector 竞争、重启／积压、低 ID 晚提交和状态错误映射由 Task 2/3 的 store/query/runtime 测试与真实 PostgreSQL 证据覆盖；完整命令故障注入仍属于运行环境边界。 |
| V10 | 通过 | 固定快照、keyset cursor、跨页身份补全和详情一致性见 Task 3 query integration 证据及 `task-3-report.md`。 |
| V11 | 通过 | cursor 签名、过期、筛选和身份／会话绑定见 Task 3 transport/query 测试与管理员 facade 复审。 |
| V12 | 通过 | operation-log schema owner、只读 verify、`all` 路由和 public schema 不变见 `task-7-migration-up.log`、`task-7-migration-verify.log`、`task-7-devruntime-integration.log`。 |
| V13 | 通过 | 内部 bearer、持久管理员／login-enabled 查询边界和 API key 拒绝见 Task 3/4 transport 与 authz 测试。 |
| V14 | 专项通过；未完成全量故障注入 | 200/400 ms 预算、容量、取消、关闭和日志故障隔离见 Task 1/4/5 race/unit 证据；真实全栈故障注入未作为本轮独立场景执行。 |
| V15 | 专项通过；运行时恢复证据有限 | producer/serviceEpoch、unknown/stale、不重置历史事实见 Task 1/2/3 测试；运行实例 readiness 通过，但未完成独立命令的数据库故障→恢复全链路注入。 |
| V16 | 部分通过 | 真实 Chrome 1440×900 管理员 operation-log 列表、状态、历史、SUCCEEDED 结果、详情抽屉和无浏览器错误通过；smoke 会员／管理员壳通过。手机、200% 缩放、八类结果及全部错误／迟到 scope 场景未逐项执行。 |
| V17 | 部分通过 | 独立服务构建、迁移、TLS readiness、最小依赖和真实全栈启动通过；尚未完成独立命令 listener 在 schema 故障后恢复 `NOT_SERVING→SERVING` 的真实注入。 |
| V18 | 部分通过 | 真实会员 `account.profile.update`→管理员列表／detail→浏览器 UI 核对通过；日志服务停机入箱／恢复及完整真实认证边界未执行。 |

## Task 8 真实环境证据

实例由指定 worktree 启动：`INSTANCE=key-operation-logs-acceptance make run`。启动与运行状态：`.superpowers/sdd/2026-09-18-key-operation-logs/task-8-runtime-status.log`；状态文件位于 `.run/instances/key-operation-logs-acceptance/state.json`。本实例报告 full-stack-ready，API `127.0.0.1:8080`，UI `127.0.0.1:4000`，operation-log `127.0.0.1:8124`，运行时 PostgreSQL `127.0.0.1:50462`。

- Chrome smoke：`.superpowers/sdd/2026-09-18-key-operation-logs/task-8-ui-smoke.log`，2 tests passed（会员与管理员 bootstrap／应用壳）。原始报告：`.tmp/athena-ui-acceptance/2026-09-18T09-47-39-346Z-b2ba99f2/report.md`。
- 会员 bootstrap／管理员 bootstrap：`.superpowers/sdd/2026-09-18-key-operation-logs/task-8-member-account-before.json` 及 smoke attachment。
- 真实会员操作响应：`task-8-member-profile-update-success.headers`、`task-8-member-profile-update-success.json`。
- 管理员日志列表轮询与 detail：`task-8-admin-operation-logs-polls.log`、`task-8-admin-operation-detail.json`；可见 `account.profile.update`、`MEMBER`、`DEVELOPMENT`、`SUCCEEDED`、可信 account/resource、`changedFields` 和 `protocolResult`。
- 管理员 UI：`task-8-operation-log-ui-final.log`、`task-8-operation-log-ui.png`、`task-8-operation-log-detail-ui.png`；列表、capture status、Reachable、History、结果和详情 Resources/Protocol/Reason 均可见，无 page error 或 operation-log HTTP 失败。
- 修复后的 R2 真实浏览器回归：`task-8-r2-operation-log-ui.log`；会员新操作进入列表，详情请求 URL 明确带 `snapshot_token`，列表快照与 detail 读取保持一致。R2 收尾：`task-8-r2-shutdown.log`。

## 代码与测试证据

- Go 单测：`task-8-go-unit.log`，相关 operationlog/server/auth/migration/devruntime/cmd 包通过。
- Go race：`task-8-go-race.log`，相关包通过，无 DATA RACE。
- Go vet：`task-8-go-vet.log`，退出码 0。
- 真实 PostgreSQL integration：`task-8-pg-integration.log`，operationlog store/schema/query 通过，DSN 为任务专用 `127.0.0.1:56669`。
- UI Jest：`task-8-ui-test.log`；修复回归为 `task-8-operation-log-ui-green-final.log`。
- UI lint/build：`task-8-ui-lint-final2.log`、`task-8-ui-build-final2.log`。
- 变更边界：`git diff --check` 通过；完整分支审阅结果见 `task-8-branch-final-review.md`（完成后补入）。

## 环境收尾

真实 ATHENA 实例与任务专用 PostgreSQL 已在最终复核后停止。停止使用同一 worktree 执行：

```bash
INSTANCE=key-operation-logs-acceptance make stop
```

收尾核对结果：state 为 `stopped`，监督进程为空，4000/8080/8124/50462/50468/50469/56669 均已释放，`athena-key-operation-logs-tests` 为 `Exited (0)`；数据卷 `athena-key-operation-logs-tests-data`、`.run/instances/key-operation-logs-acceptance/` 日志、截图和报告均保留。证据：`task-8-environment-stop.log`、`task-8-task-postgres-stop.log`、`task-8-environment-shutdown.log`。其他 worktree／容器不停止。

## 人工审查

R1 材料目录：`docs/testing/human-review/key-operation-logs/R1/`。AI 交付报告、审查指南和预填人工报告会列出当前完整版本、V01–V18 的实际边界、每个 CHK 项和未执行项；人工报告保持草稿，不能由 AI 代填“通过”。
