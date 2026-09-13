# Trader Sync 独立服务验收记录

> 状态：2026-09-13 按用户要求在验收节点暂停。独立服务、两跳/后端并发、镜像与首轮全栈/浏览器证据已通过；最终代码修复复审通过，持久全栈重启新发现Redis挂载问题，修复及后续真实smoke未完成。整个任务尚未完成，未发送完成邮件。

实施范围见[12项计划](../superpowers/plans/2026-09-13-trader-sync-independent-grpc-service.md)。本次工作区为 `/home/yege/work/athena/.worktrees/trader-sync-independent`，分支 `codex/trader-sync-independent`，基于 `3b1cd556`；未合并或发布到原 `rf4` checkout，也未执行生产部署。

可重复入口和长期边界见[本地运行编排](../design/development-runtime/local-runtime-orchestration.md)、[运行说明](../developer-guide/running-locally.md)、[生产镜像与维护](../../deploy/trader-sync/README.md)。以下日志与结构化证据保留在该工作区的 `.superpowers/sdd/2026-09-13-trader-sync-independent-grpc-service/`（下文简称证据目录）和 `.tmp/`，属于本地验收产物，不包含在源代码提交中。恢复工作先读[暂停交接记录](../superpowers/plans/2026-09-13-trader-sync-independent-grpc-service-checkpoint.md)。

## SDS 规则与实际证据

| 规则 | 已实现边界 | 验证证据 |
| --- | --- | --- |
| SDS-R1 | 独立 TS 拥有采集/订阅/投影/目录，API 无后台 runtime，Notification 独立 sender。 | 三个独立 main、runtime owner 测试；实际 `/proc` 和 Docker 证明 TS-only 只有 TS+自有 PG。 |
| SDS-R2 | 16个内部 gRPC、专用 token、可信 Actor、服务端权限；事务不跨 RPC。 | 52消息完整字段目录、最终 gateway JSON 两跳、真实 PG owner/realm/replay；`task-12a-report.md`、`task-12ac-review.md`。 |
| SDS-R3 | 独立构建、health、schema 工具、TLS 镜像、局部启停。 | `task-6-report.md`、`task-11-report.md`、`task-9-fix-1-report.md`；真实 Ctrl+C、停止/重启、镜像最小文件系统。 |
| SDS-R4 | 每进程白名单、5/15秒调用预算、TS 故障隔离、RPC ready 与 WSS degraded 分开。 | 配置/认证测试、真实 TS fatal 后 API/Notification 继续、WSS中断时 Create/Resume pending；部署与 Make 修复均独立复审通过。 |
| SDS-R5 | checkout/instance 精确归属、持久记录、辅助进程回收、managed/external 边界。 | `task-8-review.md`、`task-8-rereview-1.md`、`task-9-rereview-1.md`；真实 Go/迁移器崩溃回收、双实例与外部借用。真实局部→全栈及全栈→局部 stop/reset，peer进程、资源、数据库和健康快照保持。 |
| SDS-R6 | 单活 generation guard、离线撤权、活动/通知同事务、Notification 自有冻结/许可/结果。 | 真实跨 pool/PG 并发与完整后端 race；独立 TS 在 Notification 缺席时形成1活动和1待发送投递。 |
| SDS-R7 | 长期设计与实际入口/资源/配置同步，历史规格保留。 | 本报告及上述长期文档；25份长期文档的相对文件链接及diff whitespace已校验，最终事实增量继续检查。 |
| SDS-R8 | 后端、proto/gateway、真实实例、UI双base、镜像与失败场景。 | 下列命令及退出结果；真实全栈 smoke 两项通过；最终审阅修复单独列明，不由预检替代。 |

## 契约、事务与后端验证

最终 JSON 覆盖16个RPC的全部请求/响应、52种消息字段目录；不同大整数 sentinel 防止 revision/generation 相互错映。覆盖大于 JS 安全整数的 ID、MaxUint64、精确金额、note未传/null/空值、合法 false/0、未知时间、六区间 P/L、重复时间点、legs presence、空列表与游标。内部 server 调用计数证明实际经过内部 hop。

真实 PG 在 handler 提交后阻断响应，再用相同 request_id/载荷重试 create/pause/note，确认只写一次，版本、时间和行数稳定；不同载荷复用 request_id 拒绝。三种定向错误实现的 overlay RED 均失败，正确实现 GREEN。详情见证据目录 `task-12a-report.md`。

| 命令/范围 | 实际结果 | 日志 |
| --- | --- | --- |
| `go test -tags=integration ./internal/tradersync/acceptance -count=1` | exit0，107.091s；含100 distinct和10 shared目标受控容量。 | `task-12-acceptance-baseline.log` |
| 下面的完整后端 race 命令 | exit0；TS224.494s、store151.532s、Notification128.142s、acceptance106.235s等全部通过，无race。 | `task-12-backend-race-final.log` |
| `go test -race ./internal/devruntime ./cmd/athena-local-runtime ./tools/trader-sync-dev -count=1` | exit0；devruntime9.696s、CLI1.763s。 | `task-9-fix-1-race.log` |
| `go test -tags integration ./internal/devruntime ./cmd/athena-local-runtime ./tools/trader-sync-dev -count=1 -v` | exit0；111.725s/7.619s。真实seed撤权保留、Ctrl+C、辅助工具崩溃、外部库、API依赖均通过。 | `task-9-fix-1-integration.log` |
| 相同devruntime/CLI/tools范围 `go vet` | exit0，无诊断。 | `task-9-fix-1-vet.log` |

后端集成测试使用独立测试 PostgreSQL 中的随机数据库；测试 admin DSN 由0600的 `test-env.sh` 提供，不连接用户开发数据库：

```bash
go test -race -p 2 -tags=integration \
  ./internal/accountstate/... ./internal/notification/... \
  ./internal/tradersync/... ./internal/server/... ./util/telegram/... \
  ./cmd/athena-trader-sync/commands ./cmd/athena-notification/commands -count=1
```

首次后端 race 未通过，日志 `task-12-backend-race.log` 保留：三个旧摘要测试未使用真实 runtime guard；故障测试的 PostgreSQL backend筛选缺少database范围，误中同一个**专用测试容器**的另一个随机库。修复测试 fixture 和精确数据库筛选，增加第二数据库 owner 存活的回归后，完整命令通过；未通过降低业务 guard 或删测试处理失败。

## 独立进程、来源中断和持久性

```bash
make trader-sync-acceptance
```

该入口运行显式 opt-in 测试，始终附加随机实例后缀，拒绝复用已有目录。它通过真实 Make 命令启动独立二进制；fixture只提供录制链上响应、Profile/Gamma HTTPS与受限loopback代理，不在测试进程内替代 TS runtime。未带 opt-in 的普通 `go test` 跳过该测试，不能算作真实运行通过。

- 正式脚本 exit0，32.743s；产物 `.tmp/trader-sync-independent-acceptance/20260913T130656Z-I3LmIW09/`。
- 新测试定向 race exit0，35.606s（只直接instrument测试进程，Make构建的业务子进程为普通构建；业务race另由上面的完整后端命令覆盖）；产物 `.tmp/trader-sync-independent-acceptance/race-lZ1OoPl8/`。未因此重复完整后端套件。
- 真实 `/proc` 身份、Docker labels、仅TS+PG、seed账户、DB OID、run ID和退出结果都有结构化证据。
- 录制成交形成1条活动，Notification未启动时同事务留下1条pending投递；重启保留活动、订阅和账户，产生新epoch与持久中断。仅在停机期间出现的provider事实未导入，历史区间请求计数为0。
- 阻断WSS后标准health仍SERVING；Create/Resume保持pending_baseline且无currentInterval。恢复WSS后两个订阅回到healthy。
- 两个通过实例已精确Stop+Reset，容器和卷残留为0；日志/证据保留。两轮验收代码自身的失败实例已Stop，保留已退出容器、volume和日志：第一轮混合stdout/stderr误解析JSON，第二轮重启后测试pool仍连旧动态PG端口；修复验收捕获/重连后通过。

另用原 `.env` 中的实际来源启动并保留实例 `ts-acceptance`。授权member/admin内部RPC可用，原HTTP/WSS/cursor/站点配置逐值核对一致。停止重启后 member `fee99519-62cf-4d54-949a-003878ab0a8d`、admin `7f4eb243-5150-45ed-9980-cd4dc159e8f0` 保持，数据库 `athena`/OID16384保持；runtime generation1→2、collector epoch1→3，实际 WSS connected=true。最终修复后再次无reset重启，generation2→3、epoch3→4，实际二进制为 `176a46de`；数据库/OID/账户、配置hash及Docker资源ID保持，授权RPC通过（`final-fix-ts-compare.json`、`final-fix-ts-rpc.json`）。该实例没有订阅，故这只证明真实来源连接与重启，不证明实网交易流量或SLO。

| 保留资源 | 当前值 |
| --- | --- |
| 工作区 | `/home/yege/work/athena/.worktrees/trader-sync-independent` |
| 实例 / namespace | `ts-acceptance` / `13aee55fe55c79c78aeaab38a4b8c83d` |
| Trader Sync | `127.0.0.1:28122`，PID536539 |
| PostgreSQL | `127.0.0.1:56711`，本实例持久volume；50472/59035为先前run的旧映射。 |
| supervisor / 持久会话 | PID535729 / session94885 |
| run ID | `abce4e42-b3e0-44f0-9e09-1b14b84641b5` |
| 证据 | 当前 `final-fix-ts-run.log`、`final-fix-after-restart-ts-acceptance.json`、`final-fix-ts-compare.json`；先前 `task-12b-original-restart-*` 保留历史。 |
| 停止 | 从上述工作区执行 `make stop-instance INSTANCE=ts-acceptance`；保留数据，不执行reset。 |

## 故障行为覆盖

| 场景 | 实际断言与证据 |
| --- | --- |
| 两runtime同库 | 第二实例在worker写入前失败，第一owner保持；`TestRuntimeOfflineReadyRejectsSecondBeforeWorkerInitialization` / `TestRuntimeSecondInstanceFailsBeforeAdvancingTokens`。 |
| owner断连/卡死worker | 所有worker取消，旧代新写失败，阻塞期间不能提前释放owner，watchdog使真实进程退出；任务6进程日志与完整race。 |
| TS停止/配置错误 | `TestRealRunnerTraderSyncFatalKeepsAPIAndNotification`、`TestAPIOnlyRunsWithBrokenCollectorConfiguration`；API无效内部凭据映射503，公共session错误映射401。 |
| TS离线撤权 | `TestRuntimeWriteOfflineSummaryPermitAndRevocationAdapters`及真实跨pool事务：权限、订阅、未获许可资格同事务墓碑。 |
| Notification离线/恢复 | 独立TS活动+pending事实；`TestPermitAndActualOutcomeDoNotRequireTraderSyncRuntime`、`TestOutcomeUnknownRemainsTerminal`与既有frozen-summary/两跳pipeline保证冻结结果恢复、unknown终态。 |
| WSS暂时离线 | 上述真实独立进程证据：ready/degraded分离，pending基线与恢复。 |
| external/schema错误 | `TestExternalVerificationFailureCreatesNoChildrenOrDatabase` / `TestExternalHealthyBorrowerStopLeavesDatabaseRunning`：只读verify，无DDL/seed/reset，借用者停止不动库。 |
| 创建失败、supervisor崩溃 | `TestInitialTraderSyncExitRollsBackBatchAndPreservesEvidence` / `TestStartupToolsRemainOwnedAfterSupervisorCrash`：真实Go build/schema迁移使用者退出后才stop数据库，日志/卷保留。 |
| 局部/fullstack双向隔离 | 真实局部实例stop/reset后全栈快照相同；真实全栈stop/reset后ts-acceptance快照相同，已reset目标容器/卷残留0；见下节结构化证据。 |

## 全栈实现回归与退出竞态

任务10的六进程真实就绪/TS fatal后其余五进程仍ready、八旧模块schema准备、schema失败不启动消费者均通过，定向集成60.317s。四方向stop/reset核心测试使用真实自有PGID与Docker命令记录器，验证不操作peer；下面的最终真实双栈演练另验证实际容器，二者不混为同一证据。

最终devruntime全包集成150.867s中，仅 `TestFullStackRealFatalKeepsAPIAndEveryOtherConsumer` 在退出阶段失败：UI线程组正在退出，`/proc`身份已消失，原10ms pidfd等待尚未取得退出事件。其余项目通过。真实pthread用例复现了150ms退出窗口和2秒持续存活的区别；`14bdc996`只在持有pidfd且身份消失时最多等待250ms，必须取得内核退出事件才能视为已退出，超时继续安全拒绝，WaitProcess受调用者ctx约束。没有放宽SameProcess、PGID或TS30秒停止预算。

修后真实pthread五场景GREEN0.884s、相关race4.021s、vet0；原唯一失败的全栈场景连续两次通过33.19s/30.21s，合计63.407s。最终普通devruntime/CLI测试6.802s/0.811s及vet通过。此证据由“原完整集成其余项目通过+唯一失败项修后重验”组成，不把原失败全集日志改称PASS。

证据：`task-10-report.md`、`task-10-green-integration.log`、`task-10-final-all-integration.log`、`task-10-stop-race-fix-report.md`、`task-10-final-stop-retest.log`。Task10代码提交 `94ddc21b`，`task-10-review.md` 的规格与质量审阅均为 Approved。

## 独立镜像、TLS与部署

任务11阶段镜像 `athena-trader-sync:task11` 的ID为 `sha256:aa9db6020484d8b35f6e172577183f6eebeb878c339bc9ae19f8383bfa4fdb54`。实际构建仅含TS和schema tool，没有UI、seed或聚合main；真实镜像测试使用内部网络和专属PG，验证空业务环境health、正确CA/服务名、错误CA/服务名拒绝、非root和TERM退出0。

证据：`task-11-image-acceptance.log`、`task-11-image-final-build.log`、`task-11-image-final-health.log`。初始完整TLS矩阵与最终源码重建后的health证据分别保留，没有把一次端口监听当作TLS成功。

最终恢复顺序修复 `176a46de` 后，重新执行 `make build-service-image SERVICE=trader-sync TRADER_SYNC_IMAGE=athena-trader-sync:final`，exit0。交付镜像为 `athena-trader-sync:final`，ID `sha256:24759bfbb97428bc78c52b8961d53c7b348b180797f7527fb9b5c03f5d139e46`。`TEST_HEALTH_ONLY=true TRADER_SYNC_IMAGE=athena-trader-sync:final bash hack/trader-sync-image_test.sh` exit0，再证实schema up/verify、空业务环境TLS health、UID/GID999、最小文件系统及TERM退出0；未改变的错误CA/name矩阵沿用前述证据。日志 `final-fix-image-build.log` / `final-fix-image-health.log`，准确镜像/源码关系见 `final-fix-image.json`，精确测试run的容器/network残留均0（`final-fix-image-cleanup.json`）。

`bash hack/trader-sync-deploy_test.sh`、`bash hack/deploy-scripts_test.sh`和真实Compose配置回归均exit0。fake argv测试覆盖镜像选择、save/load、schema兼容只替换TS、不兼容维护停止/确认/up/verify/启动，以及失败不重启；真实Compose解析验证API/Notification消费者白名单、默认与显式空值和凭据隔离。实际Make命令行验证镜像/实例等8个变量的Make/shell表达式按字面传递，无执行副作用。任务9、11的原审阅问题已由 `8b1597bc` / `d94ef3aa` 修复，两个scoped复审Approved。

## UI与真实全栈

项目Node24下 `make ui-acceptance` exit0，报告 `.tmp/athena-ui-acceptance/2026-09-13T12-13-23-523Z-8a65ceb3/report.md`。`/`与`/athena`各通过33项UI fixtures和10项live，cleanup通过。live使用真实领域/PG/内部gRPC/facade/gateway，外界为loopback替身；页面fixture断言与真实网络证据分开。

从本工作区使用 Node24、独立测试配置和 `DB_MODE=managed` 启动 `make run INSTANCE=full-stack`，真实六进程及各自基础设施就绪。测试配置只使用loopback Telegram fixture，实际HTTP/WSS/cursor沿用 `.env`；本次全栈站点和Google回调使用 `localhost:24000`，没有修改原 `.env` 或 `ts-acceptance` 的站点4000配置。

第一轮 run `6d1e74cb-c0b0-4267-8a46-8da6fa7ba999` 完成以下故障演练：

- `make trader-sync-acceptance INSTANCE=ts-peer` exit0，30.835s。新局部实例走真实启动、停止和reset；前后全栈真实进程、资源、数据库、健康快照完全相同。日志 `task-12b-peer-make.log`，artifact `.tmp/trader-sync-independent-acceptance/20260913T132626Z-YqULyTiQ/`，快照 `task-12b-fullstack-before-peer.json` / `task-12b-fullstack-after-peer.json`。
- 新UI实例占用已用24000端口时，Make按预期exit2；原UI PID不变、HTTP仍200。日志 `task-12b-port-conflict.log`。
- 按登记身份/pidfd仅TERM该全栈Trader Sync。会员和管理员TS接口变为503，但两个bootstrap保持200且完整session对象相同；其余五个进程PID和ready保持。证据 `task-12b-ts-outage-http.json`、对应响应体及 `task-12b-ts-fault-signal.log`。
- 仅对本轮新建全栈执行 `make stop INSTANCE=full-stack`、`make run-reset INSTANCE=full-stack`，均exit0；该run会话正常exit0。其容器/卷残留0，局部 `ts-acceptance` 的进程、资源、DB、run和健康快照完全相同。证据 `task-12b-fullstack-reset-isolation.json`、`task-12b-original-preserved-baseline.json` / `task-12b-original-after-fullstack-reset.json`。reset只用于本次隔离演练，保留开发实例的日常停止不执行reset。

第二轮全栈 run `603fb553-fdd9-4055-9be8-e40100d54bfe` 当时已启动并通过验收，随后为验证最终修复的持久重启而正常停止；`task-12b-final-delivery-fullstack.json`核对真实进程/资源/DB/health，`task-12b-final-delivery-ts.json`再次核对独立实例。实际UI24000代理下，会员/管理员bootstrap和TS公开查询四项HTTP200，两个HTML入口200，session均已认证；见 `task-12b-final-http.json`及响应体。

```bash
PATH=/home/yege/.nvm/versions/node/v24.14.1/bin:$PATH \
  make ui-acceptance UI_ACCEPTANCE_MODE=smoke \
  UI_ACCEPTANCE_BASE_URL=http://127.0.0.1:24000
```

真实系统Chrome149 smoke exit0，会员/管理员两项通过（4.8s），cleanup通过，无基础设施失败；完整执行日志 `task-12b-final-smoke.log`，报告 `.tmp/athena-ui-acceptance/2026-09-13T13-31-14-818Z-939154b8/report.md`。smoke证明入口和会话；上面的真实TS查询/503故障演练证明内部服务路径，变更与精度由两跳/PG测试覆盖，不把shell smoke外推为交易SLO。

以下为首轮smoke通过时的历史快照；这些业务进程现在已停止，端口不作为当前可用地址。暂停状态见后节。

| 全栈验收历史资源 | 当时的值 |
| --- | --- |
| 实例 / namespace | `full-stack` / `d18893a9ea0eb3f2f2fcf0d542861c26` |
| UI / API | `http://127.0.0.1:24000`，PID236670 / `http://127.0.0.1:28080`，PID236694 |
| Trader Sync | `127.0.0.1:8122`，PID236608 |
| Notification / Wallet / Profit Sharing | `127.0.0.1:28086`，PID236636 / `127.0.0.1:28088`，PID236657 / `127.0.0.1:28108`，PID236623 |
| PG / Redis / MinIO | `127.0.0.1:58087` / `127.0.0.1:64821` / `127.0.0.1:64822`，各自持久volume。 |
| supervisor / 持久会话 | PID234952 / session83310（已正常结束）。 |
| 运行配置 / 日志 | 证据目录 `task-12b-fullstack.env`（0600）/ `task-12b-fullstack-final-run.log`。 |
| 停止 | 从本工作区执行 `make stop`；保留数据。 |

本次全栈曾依赖Telegram替身 `http://127.0.0.1:39131`，历史PID4126952 / session35842。暂停只读检查发现该PID和监听已不存在，退出原因没有证据；没有在暂停阶段重新启动。旧脚本、日志和身份记录保留为 `task-12b-telegram-fixture.py` / `.log` / `.json`，恢复全栈前需重新启动并记录新身份。不得使用旧PID操作新的未知进程。

## 最终审阅修复与复验

全分支审阅检查基线 `3b1cd556` 至实现/首轮证据提交，发现两个需要修复的问题和两处文档旧表述。已由同一修复批次 `176a46de` 处理，唯一限定范围的复审结论为规格与质量均 Approved：

- **故障测试数据库范围**：command测试的owner-loss SQL原先只按固定advisory key筛选cluster级 `pg_locks`。现限定当前数据库OID及 `objsubid=1`；在独占临时PG中，原查询真实终止第二个随机库owner而得到RED，修复后第二owner保持可用。未对开发集群运行该未限定查询。
- **启动恢复屏障**：原实现只同步恢复NULL-epoch快照，旧bound epoch逐账户清理由Collector启动后异步执行。现由已经带runtime guard的 `RecoverPending` 先同步收尾旧epoch、再恢复NULL-epoch，全部完成后才启动三个worker和开放RPC；保留Collector重连清理职责。读取view虽已能展示中断，不能替代已批准的固定初始化顺序。
- **恢复回归**：真实PG/gRPC覆盖 unbound/old_epoch × publish/owner_loss。`pg_blocking_pids`确认账户锁真实阻塞后检查health NOT_SERVING和全部16RPC Unavailable；释放锁后检查baseline持久failed/ended及旧epoch订阅interrupted，再验证ready。恢复中丢失owner会及时取消，阻塞attempt保持pending，另一库owner仍有效。两个old_epoch场景在原代码均得到真实RED，修复后四场景race通过。
- 两处长期文档已同步为Runtime整组所有权覆盖Projector，以及 `.run/instances/<instance>/state.json` 为实际归属记录主路径。

修复后的关联验证命令：

```bash
go test -race -tags integration \
  ./cmd/athena-trader-sync/commands ./internal/tradersync ./internal/tradersync/store \
  -run 'TestRuntime|TestObservation|TestCollector|TestBaseline' -count=1 -v
```

exit0，三个包18.615s / 110.990s / 37.283s；独立TS构建也exit0。日志及逐条exitcode见 `final-fix-logs/`，完整说明 `final-fix-report.md`。没有schema/proto变化，没有重跑无关全仓库套件。专供I1 RED使用的临时PG按精确container ID和标签清理完成，共享测试PG容器保持运行，关联集成测试仅使用其中的随机测试库。最终镜像证据见上节；新的独立TS受控Make验收也exit0，35.256s，artifact `.tmp/trader-sync-independent-acceptance/20260913T135143Z-LcPH2V4l/`，日志 `final-fix-controlled.log`。该临时实例已精确stop/reset；真实来源的保留TS重启通过。

## 暂停时的持久全栈重启问题

用户要求找合适节点暂停后，选择在故障证据已保存、数据保留且未开始新修复时收尾。原fullstack正常stop exit0；最终代码下原配置run在Redis启动失败，Make exit2（`final-fix-fullstack-run.log`）。8个旧模块schema准备已成功，业务未启动，启动失败回滚后该实例为stopped，三个owned持久volume均保留。没有reset、重新seed或改动原root及其他实例。

精确Redis容器 `d9e75b6aad544e55f1711624ce3518b265ee052d5d13df9165e81a7c27a8a4b0` 的重复 `docker container start <ID>` 稳定失败：Docker Desktop WSL缓存的文件bind源不存在；其源 `.run/instances/full-stack/redis.conf` 本身仍存在且为0600。`prepareAPIInfrastructure`每次调用 `SaveSecret`，后者以atomic rename替换同内容文件，而后复用旧容器；该inode变化是需在隔离fixture进一步验证的直接原因假设。精确argv、daemon错误、labels/mount/inode证据见 `final-fix-redis-start-reproduce.json`、`final-fix-redis-mount-diagnosis.json`。

待恢复时先按 `redis-restart-fix-brief.md` 在新独立fixture完成RED/GREEN；最小修复应保持未变配置文件的inode和0600语义、保留变值原子写入及路径约束。修复后对已坏且明确stopped的本任务Redis容器做精确保卷恢复，再验证两次真实持久重启及新全栈Chrome smoke。该修复尚未派发或实现；不会通过reset清空数据回避问题。完整暂停现场与继续顺序见 `final-fix-runtime-pause-report.md` 和正式[交接记录](../superpowers/plans/2026-09-13-trader-sync-independent-grpc-service-checkpoint.md)。

## 本次操作事故与恢复

2026-09-13约20:55:47，一项shell负向测试从临时checkout执行旧生命周期脚本，执行前漏设fake Docker PATH。旧脚本按固定名删除原 `rf4` checkout 的 `athena-postgres`、`athena-redis`、`athena-minio` 容器，三个持久volume保留；API/Notification随后因数据库断连退出。UI、Wallet、Profit Sharing继续运行。新TS实例和专用测试库未受影响。这是本次测试操作错误，不能作为有效隔离验证。

发现后立即告知用户并暂停相关任务。确认原supervisor PID307846/RPC8555及原启动脚本和volume fingerprint后，只通过该supervisor恢复三个基础设施，再恢复API/Notification。PG使用原数据目录并跳过初始化，40业务表/13数据库可读，RedisPONG、MinIO200。Notification拒绝未确认停止的旧sender：从登记身份、原supervisor退出日志和`/proc`确认PID335374确实退出后，仅对incarnation `a16f6843-8cd1-4611-996a-08998905fca1`运行原CLI恢复，exit0；该恢复命令不调用Telegram。随后恢复原Notification并保留60秒恢复屏障。

恢复后原API8080、Notification8086和新TS28122的gRPC health均SERVING，原UI4000/APIhealthz为200，原UI/Wallet/ProfitSharing PID保持；三容器均挂载原volume。中断约8分钟；volume保留与恢复可读不能证明中断瞬间未持久化数据零损失。没有reset、删除卷或修改原checkout源代码/配置。新测试改用先隔离PATH的命令替身；默认全栈实现替换旧固定名清理为精确归属引擎。

证据：`task-10-red-shell.log`、`task-10-incident.md`、`task-10-incident-sender-recovery.log`、`task-10-incident-health.log`、`task-10-incident-restored-resources.json`；原日志 `/home/yege/work/athena/.tmp/third-party-vulnerabilities/runtime.log`。另一次早期schema生成工具排查只涉及专用测试容器：admin DSN未实际切换到随机DB，已改为新admin库、DSN roundtrip/数据库身份断言及重新生成正确contract，未触及用户库。

## 证据限制与剩余项

录制/合成provider、loopback Telegram、有限本机测试不证明公网供应商静默漏推完整性、100个真实持续活跃目标、长期稳定性或端到端公开时效SLO。单活采集重启有可见中断，不承诺多副本HA或无中断滚动升级。没有执行真实Telegram测试投递或生产发布。

必需剩余项：Redis持久重启缺陷的隔离复现/修复/审阅、原fullstack保卷恢复与两次重启、新binary全栈公开接口/真实Chrome smoke、最终整体审阅与交付事实收口。用户要求暂停，未开始新修复，未发送完成邮件；原I1/I2修复及限定范围复审已经通过。

## 实施裁定记录

以下按记录顺序保留实现中的具体裁定；没有改变单活采集、同库事务或已批准业务范围。

1. 计划任务1、3、8按互不重叠文件并行实现，git提交用同一短期锁串行；审阅diff按任务提交/文件范围生成 — 用户批准计划允许并行，当前developer要求主动并行 — 如果文件边界判断错误需合并重验。

2. task-brief脚本仅识别英文Task标题，本计划中文标题用等价Python按任务标题抽取，保留完整任务与全局约束 — 不修改通用skill脚本 — 抽取错误会造成任务范围遗漏，已检查12个brief。

3. 进程清理采用逐成员pidfd核验，Manager内部承载状态与资源操作，公开计划接口保持；run使用带run标记的前台supervisor子命令 — 避免PID/PGID复用及/proc/environ运行时setenv不生效 — 如果成员身份不能证明，清理报错并保留证据，代价是需人工排查残留。

4. proto生成使用任务私有GOPATH并复用模块cache，固定当前Swagger版本输入 — 旧脚本会写GOPATH/src且可能指向其他checkout — 若未隔离会误改其他任务，此项已通知生成实现者。

5. task2在生成链对新增TS内部client字段/constructor进行确定性的ClientConnInterface适配，并正常重新生成 — 现有gogo grpc插件只生成具体*ClientConn，无法消费计划要求的无连接Unavailable adapter — 若转换规则失配则生成应明确失败，避免手改生成物/泄漏真实空连接。

6. task4在Collector既有Acquire之后、worker之前接入guarded store；task6再整体迁走Acquire/Close；纯AccessRevocationAdapter安装如编译需要前移task4 — 不引入临时第二owner或旧无guard写fallback，保持任务内Collector验证可运行 — 需检查并发开始前完成赋值，最终生命周期仍归task6。

7. 内部Actor.account_id只接受规范小写36位非零UUID，拒绝大写/URN/空白；API resolver以parsedUUID.String()形成可信Actor — spec明确非规范拒绝，优先于brief“现有规范化”歧义和txgate宽松输入 — 内部非规范调用明确ACTOR_INVALID，公开真实ID已规范化不受影响。

8. 任务5依赖已审阅任务3，趁任务8修复并行实施schema；不依赖任务8，不接其不稳定API — 缩短独立分支等待，task8仍未标完成，task9须等reviewclean — 若文件边界冲突要协调重验。

9. 固定MigrationLockName的跨库互不阻塞指PostgreSQL advisory数据库隔离；保留现有goose全局FS在单进程内串行，mutex改可取消channel以遵守总deadline，不扩大Provider重构 — 独立迁移进程不同DB不互锁，避免为消除既有全局设置改全系统迁移库 — 同进程多库迁移仍串行需报告清楚。

10. task5为后续测试保留一个明确任务拥有的随机空admin数据库，安全更新0600 test-env.sh及test-postgres.json；先前postgres管理库41个migration对象保留证据，最终删除整个专用测试容器 — 避免fixture的admin为空前置受事故影响，不需要清理未知对象或用户DB — 清理必须等全部测试结束按container身份确认。

11. 任务6和7按固定接口并行，任务5已提交API稳定但审阅仍需闭环，依赖任务不得最终验收完成前跳过其发现 — 6拥有TSruntime/collector与directmains，7拥有publicfacade/API/Notificationcommands；旧组合源已复制task-6-legacy-runtime.go.txt供6迁移，避免7删除影响阅读 — 并行编译窗口需协调，最终服务组合/build在双方稳定后验证，设计边界不变。

12. 实例实际运行的service/supervisor二进制采用每run或内容寻址的不可变产物路径，不直接长期执行会被其他实例build覆盖的共享dist目标 — Linux覆盖运行中inode使/proc/exe追加(deleted)，破坏严格身份核验并阻止自己的stop — Build公共目标可保留dist，但Run必须snapshot/cache稳定文件，新增并行build后旧实例仍可stop的真实测试；不能放宽SameProcess。

13. external显式TS token只要求选定API/TS读者，Notification-only不读TSsecret，UI-only不启动PG/要求DSN — spec配置归属表及最小依赖优先于task9“external必须DSN/token”的泛述 — 选定DB使用者与RPC读者分别校验，避免重建不相关服务配置耦合。

14. 统一运行日志标记为ATHENA_LOCAL_RUNTIME_INSTANCE +已有ATHENA_LOCAL_RUNTIME_RUN_ID；后者由Manager.Spawn在exec前追加，service白名单不要预填冲突值 — 与严格进程身份同一来源，避免另一套ATHENA_RUNTIME_*别名 — directbinary未设置明确unmanaged。

15. task6恢复期间health必须真正返回NOT_SERVING，不能仅bind listener后让Check等待deadline；先稳定Service句柄注册RPC/health，ready=false拦截业务，再owner/guard/compose恢复完成同步发布后ready — 对齐批准的可观察启动阶段，第二实例仍不初始化业务worker — 需要race和阻塞恢复健康测试，禁止两套listener切换。

16. task9 API-only需自身Redis与头像存储等完整基础设施，不能因lazy client构造暂未连接就省略；task10复用 — 服务独立可运行包含自身必要行为，非业务远程进程可选不代表本地基础设施可省 — 需bootstrap/账户与必要bucket准备证据。

17. 任务12拆成有界契约/幂等12A与剩余真实运行/文档整合12B，12A可与runner/deploy独立并行；root提前执行已有集中后端和UI验证 — 文件与依赖边界清晰，减少等待但不降低任一原12验收 — 若共享fixture有改动影响已有证据须重验受影响入口。

18. Task12A契约测试和12C集中回归fixture修复同属任务12后端验收，完成后由一个独立reviewer联合审阅两份精确提交包 — 避免为同一测试集整合重复上下文/占用席位，仍保留每个要求与fix的明确verdict — 如果任何产品或跨层修复扩展范围，review必须覆盖新增具体风险，不省略独立审阅。

19. 全栈入口本轮只支持DB_MODE=managed，external同库联调使用run-service/run-services；fullstack external在资源/DDL前拒绝 — 批准external协议只定义权威account-state库，未定义8旧模块库借用验证，避免对外部PG隐式CREATE/migrate — 若用户后续需要全栈外部库，需要单独明确多个库的配置与维护权责。

20. 保留本次工作区运行日志和验收证据目录，完成后不按通用skill删除execution workspace — 项目AGENTS要求交付实际日志/证据并保留开发服务，报告也指向本地证据 — 代价是占用本地磁盘，可在用户不再需要验收证据时单独清理。

21. 最终审阅确认启动恢复必须遵守批准spec第45行，旧bound epoch逐账户清理同步在workers/ready前完成；observation view的读取正确性不能替代该初始化顺序 — 最小补齐recover barrier并扩展真实PG/gRPC恢复测试 — 若判断过严，代价仅为恢复完成前延后业务ready，与已批准要求一致。

22. 最终新binary真实持久重启暴露Redis文件bind失效，不能用reset后新建实例的成功替代持久重启证明；继续任务须最小修复并保卷复验 — 当前用户要求在合适节点暂停，因此停在证据完整、数据保留、修复尚未派发的节点 — 未完成项及恢复前置状态均保留，整个任务不宣称完成。
