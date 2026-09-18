# SDD ledger — plan: docs/superpowers/plans/2026-09-18-key-operation-logs.md

任务开始：2026-09-18 02:11:46 UTC。通知未发送。所有执行累计，子任务不发邮件。
工作区：/home/yege/work/athena/.worktrees/key-operation-logs；分支 codex/key-operation-logs；基线 033c6e81；根 rf4 干净、未修改。
已读根 AGENTS.md，rg 全树只找到根规则。已读 using-superpowers/Codex reference、brainstorming（批准已满足）、using-git-worktrees、writing-plans、TDD、SDD、sync-athena-changes。

## 实施前核对

| 任务／共享关系 | 生产／消费及一致性结论 |
| --- | --- |
| 1 | event、ingest、record 分责，测试目录数量和行为，命名由实现报告固化；无数据库依赖 |
| 2 | 只新增 operation_log schema，任务文本与验证 public 不变一致 |
| 3 | 内外协议 DTO 和 Viewer 一致；五公共／三内部方法；默认时间仅首次 |
| 4 | 37 项在拦截器唯一收集，facade 只观察事实；没有第二个 gateway wrapper |
| 5 | 61 位置包含失败初始化和登录子动作；同请求共享预算 |
| 6 | 三来源独立与稳定快照一致；scope 变化清理全部旧响应 |
| 7 | 最小依赖与全栈选择分开，保持资源归属 |
| 8 | V01–V18、AI 审閱、收尾、人工材料分别举证，不互相替代 |
| 1→2 | Sink Append／PublishStatus 使用 event.Event／ingest.Status，store 实现，不反向依赖 |
| 1→4、5 | recorder 通过 context 传递，默认 UNKNOWN／共享 request budget，hook nil-safe |
| 2→3 | store 投影和 schema 供 query/runtime 使用；协议包不导入 API runtime |
| 3→4 | facade/client 先生成后注册；API 持有 producer/client，不持有日志 runtime |
| 3→6 | proto 类型化 DTO 是 UI 来源；UI 不做隐式数字转换 |
| 4→5 | 共用服务器字段／context；任务顺序执行，不并行写这些文件 |
| 2、3、4→7 | 迁移 owner、服务入口、配置名落地后再接入 registry |
| 1–7→8 | 汇总覆盖与实际验收，再更新长期文档及人工材料 |

## 状态

计划已完成自查；用户已授权继续，不再次询问执行方式。按 SDD 单实现代理串行，各任务独立审阅。主代理同时准备后续任务上下文、验证环境及文档。

Task 1: in progress — implementer /root/event_capture，BASE 0f8d762ac415acd5b6cec5cb9c94a1349744907e。曾被中断，02:20 UTC 原代理恢复，尚无重复派发。
追加基线：server/...、account schema 真实 integration、TypeScript 均 exit 0，证据见 environment.md／integration-notes.md。
任务 5 文件定位精化：真实 CredentialManager 在 internal/accountcredentials/manager.go，不在 util/session；属批准 hook 范围，无语义或范围变更。

## 2026-09-18 02:24 UTC 中断收尾

执行收到两次 turn_aborted；第一次恢复后再次中断，现停止推进并保存进度，未将实施授权取消或视为外部阻塞。
- 已提交中文计划 0f8d762a；尚无产品实现提交。
- Task 1 implementer /root/event_capture 当前 interrupted；internal/operationlog/event 下四个未提交源／测试文件保留，尚未经任务审阅、不可视为完成。
- 部分源码备份：.tmp/key-operation-logs/interrupted/partial-event-foundation.tar.gz。
- Task 2–8 未实施，V01–V18 未完成，真实 ATHENA 栈未启动，人工审查材料未到交付阶段。
- 前述基线测试和隔离浏览器 preflight 成功不等于新功能验收。
- 本任务 PostgreSQL 正在按环境记录精确停止，数据卷和日志保留，最终状态见 environment.md。
- 累计执行约 12 分钟，主代理本次结束前发送固定通用通知一次；结果另记。

通知已发送一次：make notify-task-complete exit 0；SMTP accepted attempt 1/3；固定标题／正文，日志 .tmp/key-operation-logs/interrupted/notification.log。恢复同一任务不要重复发送本次已完成的通知。
最终根仓库核对：rf4 在 033c6e81457c03326dc0d729d6696e0ff5c4010c，git status --short 为空。

## 2026-09-18 02:27 UTC 继续实施

用户询问暂停原因；主代理承认将工具中断误判为任务暂停，原实施授权持续有效。继续原任务，不重新设计、不重复已完成计划／基线检查。Task 1 原代理已恢复。此前固定通知已发送，按同任务去重不重复发送。暂停等待用户期间不计执行时间。

主代理正在同步 docs/requirements/{README.md,observability/key-operation-logs.md} 与 docs/design/{README.md,observability/operation-logs.md} 的恢复实施状态；这些未提交文件不是 Task 1 代理所有，后续汇总提交。历史规格原件不改。
浏览器前置检查追加：system Google Chrome 149.0.7827.53、smoke CHECK_ONLY exit 0；仅就绪，不是环境验收。

Task 1: implementation bc5fb40ff5a53b8ef85a9a977bc35c06d4e7460d, DONE_WITH_CONCERNS. Actual report task-1-report.md. Dynamic per-entry state/stage/warningCode mappings are designated adapter scope for Tasks 4/5; reviewer independently checking foundation contract. Race three packages / vet / diff check evidence stored task-1-green.log. Review agent /root/review_event_capture reviewing full Task 1 range (134938 bytes). Not yet marked complete.

Task 1: fix round 1/5 started, FIX_BASE bc5fb40f. Independent review /root/review_event_capture: 3 Important (producer lifecycle observations/final status; non-nullable JSON null rejection; Access Revision case alias). Root verified lifecycle omission and accepts targeted reviewer reproductions; no design change. Original implementer /root/event_capture resumed with task-1-review.md. Cross-task pending state/reason/effect and resource domains carried to Tasks 4/5, not marked complete prematurely.

Ruling: Task 5 的 wallet_selection.replace 测试按实际单 RPC/SQL 原子事务检验 SUCCEEDED／拒绝／UNKNOWN，不人为制造“移除部分后添加失败”；PARTIAL 仍用于真实已确认子效果 — 批准规范要求准确事实、原子批量不得虚构部分成功，实际 internal/wormtrading/store/wallet_selections.go:80..248 在同一事务提交，规格中的多步例子不适用当前路径 — 若下游实际另有事务外效果未被当前源码定位，将需补观察和回归；不改变业务事务。计划同步校正真实 CredentialManager 路径，历史规格/目录原件不改。

只读辅助 /root/capture_source_mapping（sol/high）在独立读取 37 gRPC 动作真实提交点和枚举，输出 grpc-capture-map.md；不是实现代理或额外 review seat，不写代码、不执行测试／业务操作。当前唯一实现代理仍是 Task 1 fix /root/event_capture；主代理完成 Task 5 源码 hook／Task 8 环境准备。

Task 1: fix round 1/5 (3 addressed, 0 open; commits bc5fb40f..9c90cc12). Scoped re-review clean; task-1-rereview.md.
Task 1: complete (commits 0f8d762a..9c90cc12, review clean). Deferred integration domain checks carried in integration-notes.md; no current Task 1 gaps.
Task 2: starting, BASE 9c90cc12ccccb624a2c6e96455444d50197669bc; storage implementer dispatch pending.

Task 2 implementer /root/log_storage (astra/high), running. Real PostgreSQL ready at 127.0.0.1:56669; root retains ownership.

只读辅助 /root/http_capture_source_mapping (sol/high) 梳理 61 native HTTP/auth entries，输出 http-capture-map.md。只有 log_storage 修改产品代码。主代理新增并维护 docs/testing/key-operation-logs-acceptance.md，Task1证据已复制到 .tmp/key-operation-logs/task-1；长期设计改为基础层已落地而非“均尚未实现”。以上文档暂不提交，以免干扰 Task2 review range。

Source map complete: grpc-capture-map.md covers 37/37 gRPC rows; it identifies resource-shape/primary-ID/catalog field mismatches, gateway probe non-durable acceptance, post-commit projection read hazards, ballot composite identity, Trader Sync desired-vs-observed state, and successful no-ops. Carry to Task 4/5 implementation; no design approval pause because these are existing-source facts within approved “evidence only” rule.

## 2026-09-18 Task 2 复审与修复完成

- Task 2 基础提交：`4bb6525b feat(operation-log): persist and publish operation history`。
- 审阅修复保留在当前工作区，待提交：detail 16 KiB CHECK 与 schema contract 同步；历史版本 INSERT 移除 upsert；projector 增加单条 savepoint 隔离、可信身份折叠；修正已验证 partial identity → complete FINISH；修正 START/FINISH ProducerID 不一致隔离；catalog dump 改为只读 contract golden 断言。
- TDD 红绿证据：`task-2-identity-completion-red.log` / `task-2-identity-completion-green.log`、`task-2-producer-mismatch-red.log` / `task-2-producer-mismatch-green.log`。
- 最终验证证据：`task-2-integration-final.log`、`task-2-race-final.log`、`task-2-unit-final.log`、`task-2-vet-final.log`、`task-2-cli-build-final.log`、`task-2-sqlc-final.log`、`task-2-schema-rerun-green.log`；真实 PostgreSQL 为 `athena-key-operation-logs-tests` / `127.0.0.1:56669`。
- 独立复审已通过：未发现 Critical/Important/Minor；确认 Task 2 可进入 Task 3。Task 3 仍需实现 projector runtime loop/backoff、查询协议及完整 V10–V18 验收。

## 2026-09-18 Task 3 开始

Task 2 已在 `4bb6525b` 基础上由 `5edced00` 完成修复提交并通过独立复审。Task 3 implementer `/root/task3_query_service` 已派发，brief 为 `.superpowers/sdd/2026-09-18-key-operation-logs/task-3-brief.md`，基线 `5edced00`；等待实现、任务审阅及必要修复。
- Task 2 follow-up capacity review found and fixed a real Important: `entry_version.detail` now allows 65536 bytes so the full normalized view can retain legal 32 KiB events plus receipt metadata; added 100-resource PostgreSQL regression (`task-2-large-event-red.log` / `task-2-large-event-green.log`), synchronized catalog contract, and committed `ba5be08a`. Task 3 implementer was instructed not to stage these Task 2 files.

## 2026-09-18 Task 3 review round

初轮 Task 3 独立审阅发现 C1 内部 List 丢失 from/to、C2 DTO 缺少 presence，以及 I1–I7 错误映射、运行状态、健康、权限、字段、消息容量和证据缺口。实现代理随后因模型容量错误中断；主代理在保留其已暂存 proto presence 改动的基础上完成修复并提交 `8411d169`。修复包含两套 proto 的 Nullable wrapper 与生成物、内部时间范围、详情稳定错误、快照时间、查询预算、ERROR 状态、健康状态轮询、账户库故障映射、公共 DTO 字段、facade 状态接口管理员校验及 RPC 64 KiB 边界。

Task 3 fix 验证证据：`task-3-fix-unit.log`、`task-3-fix-race.log`、`task-3-fix-race-store.log`、`task-3-fix-integration.log`、`task-3-fix-vet.log`、`task-3-fix-build.log`、`task-3-fix-diff-check.log`；真实 PostgreSQL 使用 `athena-key-operation-logs-tests` / `127.0.0.1:56669`。等待第二轮独立审阅，Task 3 尚未标记完成。

## 2026-09-18 Task 3 最终复审与提交

- Task 3 运行时／传输边界提交：`5929a540`；查询协议、presence、公共 JSON 展平、稳定 reason、管理员错误映射、详情字段清理与 readiness recovery 提交：`2b932e37`。
- 第三轮独立审阅：`task-3-final-review.md`，Critical=0；确认前两轮发现的 DTO shape、错误分类、详情重复字段、投影 readiness 和 gateway nullable 遗漏均已修复。
- 通过证据：scoped unit、race、vet、CLI build、真实 PostgreSQL query/store/schema integration、bufconn bearer/容量、TLS 临时 CA、gateway 标量 JSON、readiness refresh 单测及 `git diff --check`；具体最终命令列于 `task-3-report.md`。
- V17 仍待 Task 8 以独立服务命令 listener 接真实 schema/account 依赖，完成故障→`NOT_SERVING`→恢复→`SERVING` 的端到端证据；当前代码路径已有 bounded Verify/readiness refresh，不能把单测替代该验收。
- Task 3 已实现代码并提交，后续任务按用户要求串行继续；Task 8 汇总 V10–V18 和真实 ATHENA 验收状态。

## 2026-09-18 Task 4 完成（提交前）

- Task 4 gRPC lifecycle/capture 实现已完成：`internal/server/authz.go`、`internal/server/athena-server.go`、`internal/server/operation_log.go`、operationlog forwarding 以及 37 个 facade 入口的 typed hooks。
- 独立复审 `task-4-final-review.md`：Critical=0、Important=0、Minor=0；复审确认 account access enum JSON、认证 marker、交互管理员查询、资源 ID 严格校验和 Trader Sync 持久 desired state 均已修正。
- 复审保留的事实缺口：Telegram 删除 attempt 的 response 无 ID/status；contract blocklist create response 无下游 code hash；gateway probe 接受点仅进程内调度；wallet batch 无持久 batch ID。详见 `task-4-report.md`。
- 验证证据：`task-4-unit.log`、`task-4-race.log`、`task-4-vet.log`、`task-4-build.log`、`task-4-integration.log`、`task-4-diff-check.log`，均退出 0；真实 PostgreSQL 为 `athena-key-operation-logs-tests` / `127.0.0.1:56669`。
- Task 5–8、V01–V18、真实 ATHENA 环境验收和人工审查尚未完成。

## 2026-09-18 Task 5/6 继续执行

- Task 5 HTTP、认证、账户和 Worm 采集已实现，覆盖 catalog 的 61 个 HTTP/auth positions；development／Phantom 失败结果复审后按 DENIED／FAILED／UNKNOWN 修正。独立复审记录在 `task-5-final-review.md`，Critical=0、Important=0。
- Task 5 专项证据：`task-5-race.log`、`task-5-vet.log`、`task-5-build.log`、`task-5-integration.log`；受影响 Go 包、真实 PostgreSQL store/schema integration、diff check 均通过。
- Task 6 管理员 operation-log 列表、详情和独立 capture status UI 已实现；`task-6-final-review.md` 复审为 Critical=0、Important=0，`yarn lint` 和 `task-6-build.log` 通过。真实浏览器验收仍留给 Task 8。
- 当前未提交父任务文档、`athena-operation-log-migrate`、`docs/testing/key-operation-logs-acceptance.md` 保持不混入 Task 5/6 提交；当前产品提交待完成 staged boundary 核对。

## 2026-09-18 Task 7 完成

- Task 7 已实现独立 operation-log devruntime 服务、account/operation-log schema owner、专用 migration all 路由、独立 Dockerfile/Compose 服务、TLS health probe、secret 隔离和最小服务依赖；API 不接收 cursor key。
- Task 7 独立复审 `task-7-final-review.md`：Critical=0、Important=0。复审期间修复 Compose DSN 分叉、account-state 维护归属、TLS CA healthcheck 和 devruntime TLS readiness；保留一个分离 instance 未共享 token 时的 Minor 使用边界。
- 验证证据：`task-7-unit-final2.log`、`task-7-race-final2.log`、`task-7-vet-final2.log`、`task-7-compose-final.log`、`task-7-shellcheck-final.log`、`task-7-migration-up.log`、`task-7-migration-verify.log`、`task-7-service.log`、`task-7-devruntime-integration.log`；报告见 `task-7-report.md`。
- Task 8 尚未开始：V01–V18 全量汇总、真实 ATHENA 环境验收、整分支审阅、环境收尾和人工审查材料均未完成。
