# 人工审查指南：用户关键操作日志系统

## 审查对象

- 任务标识：`key-operation-logs`
- 轮次：R1
- 仓库／worktree：`/home/yege/work/athena/.worktrees/key-operation-logs`
- 分支：`codex/key-operation-logs`
- 受审产品版本：`66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0`（完整产品提交 SHA；材料提交不改变该产品版本）
- 设计依据：[关键操作日志设计](../../../../superpowers/specs/2026-09-18-key-operation-logs-design.md)、[需求](../../../../requirements/observability/key-operation-logs.md)、[设计说明](../../../../design/observability/operation-logs.md)、[实施计划](../../../../superpowers/plans/2026-09-18-key-operation-logs.md)
- AI 验收记录：[key-operation-logs-acceptance.md](../../../key-operation-logs-acceptance.md)
- 配套报告：[human-report.md](human-report.md)

## 版本核对

在启动服务或执行写操作前，从目标 worktree 执行：

```bash
cd /home/yege/work/athena/.worktrees/key-operation-logs
test "$(git rev-parse 66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0^{commit})" = 66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0
git merge-base --is-ancestor 66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0 HEAD
git diff --exit-code 66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0 HEAD -- cmd internal pkg ui hack Makefile go.mod go.sum sqlc.yaml
```

预期第一、第二条成功，第三条只允许本轮明确列出的后续产品差异；当前页面 scalar/wrapper 修复已包含在 `66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0`。父任务文档差异和 `athena-operation-log-migrate` 预编译产物不属于产品版本核对范围，不能 reset、clean 或覆盖。

## 环境恢复

本轮需要真实开发环境进行复核。若环境已停止，在同一 worktree 执行：

```bash
INSTANCE=key-operation-logs-human-review-r1 make run
make runtime-status INSTANCE=key-operation-logs-human-review-r1
```

入口为会员 `http://localhost:4000/`、管理员 `http://localhost:4000/admin/`；对应 API 为 `http://localhost:8080`，operation-log 服务仅内部监听。使用本地开发身份 `local-user` 与 `local-admin`，不要伪造生产登录或外部凭据。若复用已有实例，先记录实例所属 worktree、状态和日志，不停止其他任务资源。

本轮已有 AI 真实证据来自实例 `key-operation-logs-acceptance`：状态日志 `.superpowers/sdd/2026-09-18-key-operation-logs/task-8-runtime-status.log`，UI smoke 报告在 `.tmp/athena-ui-acceptance/2026-09-18T09-47-39-346Z-b2ba99f2/`。用户可复核该已停止实例的日志与截图；若需重新启动，使用新的 `key-operation-logs-human-review-r1` 实例并在收尾项停止。

## 身份与数据

| 用途 | 身份 | 数据与预期 |
| --- | --- | --- |
| 会员操作 | 本地 `local-user`／MEMBER | 读取自身账户，执行一次 `account.profile.update`，使用当前 profile revision；只修改测试 display name 或保持原值。 |
| 管理员查询 | 本地 `local-admin`／ADMINISTRATOR | 读取 Operation Logs 列表、capture status 和单条 detail；不执行账户权限或业务写操作。 |
| 失败／边界 | 受控请求或既有专项测试 | 使用已保存的 Task 1–7 证据，不为了制造失败而修改真实业务数据。 |

## 检查顺序

### CHK-001：版本与范围

- 设计依据：计划的任务边界和“不得覆盖其他 worktree”规则。
- 操作：执行版本核对命令，阅读验收记录的 V01–V18 表和当前 git status。
- 预期：产品祖先为 `66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0`；只看到已说明的文档／产物差异；未执行 reset、clean、stash。
- 记录：命令输出、实际差异和版本。
- 影响与恢复：只读，无影响。

### CHK-002：真实栈与双 realm bootstrap

- 设计依据：runtime-and-verification、管理员入口设计。
- 操作：启动或复用专用实例，运行 runtime-status，打开会员和管理员入口，观察 bootstrap 与核心服务状态。
- 预期：full-stack-ready；会员为 `local-user`，管理员为 `local-admin`；operation-log 服务与 schema ready；无把 HTTP 200 单独当作验收通过。
- 记录：状态日志、两份 bootstrap 响应和入口截图。
- 影响与恢复：仅启动本轮实例；CHK-011 停止。

### CHK-003：会员关键操作入箱

- 设计依据：V18、Task 4/5 采集契约。
- 操作：会员读取账户 profile revision，执行一次 `PUT /api/v1/account/{id}/profile`，等待入箱；保存响应和 operation id。
- 预期：业务响应成功或明确业务失败；日志采集不改变业务结果；记录可信 MEMBER、account/resource、action code 和 protocol result。
- 记录：请求响应、日志列表中的 operation id、时间。
- 影响与恢复：只使用测试账户；保持或恢复原 display name。

### CHK-004：管理员列表、状态和详情

- 设计依据：V10、V11、V16、Task 6 UI。
- 操作：管理员打开 `/admin/operation-logs`，查看 Capture status、History，筛选 `SUCCEEDED` 或 action code，打开会员操作详情。
- 预期：Projection/Query ready/Persistence 状态独立显示；列表稳定展示 action、actor、resource、outcome；详情显示资源、effect、changes、protocol，不显示请求 body、token、私钥或备注正文。
- 记录：页面截图、列表/detail JSON、URL 和浏览器错误记录。
- 影响与恢复：只读查询。

### CHK-005：身份、权限和敏感字段边界

- 设计依据：V03、V04、V07、V13。
- 操作：阅读 Task 4/5 authz 与 sensitive-field 测试证据；在管理员入口尝试只读查询，确认 member/API key/development 查询边界按当前本地 auth 配置工作。
- 预期：actor 只能来自可信认证结果；管理员查询不接受 API key/development 伪造；详情保留白名单字段，敏感材料不落日志。
- 记录：对应测试日志和实际 HTTP 状态；无法在当前身份复现的项写受阻。
- 影响与恢复：不修改权限和 token。

### CHK-006：分页、过滤和快照

- 设计依据：V10、V11、查询协议附录。
- 操作：在列表使用 outcome、module、actor 过滤和 Next page；保存 snapshot token，打开 detail，再刷新列表。
- 预期：cursor 只由服务签发；筛选／身份／snapshot 参数篡改被拒绝；页面不虚构 total，detail 与同一快照事实一致。
- 记录：请求 URL、状态和返回 JSON。
- 影响与恢复：只读。

### CHK-007：日志故障不阻断业务

- 设计依据：V06、V14、Task 1/4/5 producer 与 recorder。
- 操作：阅读并复核 `task-1-*`、`task-4-*`、`task-5-*` 的 sink failure、budget、panic、response-write 和 retry 证据；不在真实栈中 kill 共享进程制造故障。
- 预期：业务结果仍按真实 handler 返回；日志失败只产生 bounded 状态／UNKNOWN 等事实，不重放用户业务。
- 记录：测试命令、退出码和证据路径。
- 影响与恢复：只读。

### CHK-008：独立服务、迁移和健康边界

- 设计依据：V12、V15、V17、Task 7。
- 操作：阅读 migration up/verify、独立 service listener、TLS readiness、Compose 契约和 devruntime integration 证据；若运行新实例，确认 operation-log 不发布宿主端口且只使用规定 schema/依赖。
- 预期：独立 schema owner、只读 verify、API 不持有 cursor key、内部 bearer/TLS 边界和最小依赖成立；独立命令故障到恢复的全链路若未重测，报告保留“未执行”。
- 记录：日志、端口与依赖快照。
- 影响与恢复：不删除卷；CHK-011 按归属停止。

### CHK-009：桌面、手机、缩放和键盘

- 设计依据：V16、管理员页面设计。
- 操作：在 1440×900、390×844 和 200% 缩放检查列表、筛选、详情抽屉、空态、错误态和键盘焦点。
- 预期：内容不溢出，操作可键盘完成，capture status 与 history 层级清晰；每种结果状态语义保持后端值。
- 记录：每个 viewport 的截图与实际状态。
- 影响与恢复：只读。若本轮不执行某 viewport，报告标为未执行。

### CHK-010：V01–V15 证据审阅

- 设计依据：verification-map.md。
- 操作：逐项阅读 Task 1–5 报告、独立审阅、Go unit/race/vet、真实 PostgreSQL integration 和 migration 证据，核对目录 74/98、状态枚举、原子 wallet selection、savepoint、身份折叠和敏感字段。
- 预期：每项都区分专项自动化证据、真实环境证据和未执行故障注入；不得把单测写成真实业务验收。
- 记录：V01–V15 对照表和缺口。
- 影响与恢复：只读。

### CHK-011：按归属收尾

- 设计依据：运行规则和 Task 8 任务要求。
- 操作：从本 worktree 执行 `INSTANCE=<本轮实例> make stop`；若使用任务专用 PostgreSQL，最后停止 `athena-key-operation-logs-tests`；随后检查进程、4000/8080/8124/56669 端口、容器和卷。
- 预期：本轮启动的实例与容器停止，卷、日志、截图、报告保留；其他 worktree、共享容器和进程保持原样。
- 记录：停止输出和 `task-8-environment-shutdown.log`。
- 影响与恢复：不执行 run-reset、docker prune 或删除卷。

### CHK-012：人工结论边界

- 设计依据：athena-human-review 技能和项目规则。
- 操作：阅读三份 R1 材料和当前报告；用户填写每项结果、问题、总体结论和提交声明。
- 预期：AI 不代填“人工通过”；未执行／受阻项保留原状态；只有用户明确确认准确版本并完成必查项后才能记录最终交付。
- 记录：人工报告提交时间、问题编号和用户结论。
- 影响与恢复：只修改审查材料，不修改产品数据。

## 修复轮复验范围

R1 无先前人工 ISSUE；本轮发现的 UI scalar/wrapper 问题已在 `66e4b5f6c56782ee7df187bdc49f7ca4e4039ca0` 以测试先行修复。后续人工发现问题时保留用户填写的 ISSUE 编号，并在下一轮复验受影响流程。

## 本轮现场收尾

- 本轮资源归属：`key-operation-logs` worktree 启动的 `key-operation-logs-human-review-r1`（若恢复）以及任务专用 PostgreSQL `athena-key-operation-logs-tests`。
- 停止命令：`INSTANCE=<实例名> make stop`；任务专用 PostgreSQL 用 `docker stop athena-key-operation-logs-tests`。
- 停止后的核对：state.json Phase、所属进程、4000/8080/8124/56669 端口、容器状态和保留卷／日志。
- 不属于本轮的进程、容器、worktree 保持原样。
