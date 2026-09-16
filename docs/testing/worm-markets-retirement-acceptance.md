# Worm Markets 退役与跨层验收

验收日期：2026-09-16。范围为 Tasks 1–10；源码与隔离验收版本截至 `191e4f31`，精确数据库退役工具为 `0022bed4`。原 main default 的账户迁移、真实只读验收、五来源通知维护、专属数据库删除、正常重启和环境收尾均已完成并留证；最终文档和整分支审阅另行记录，不把审阅前状态写成整个任务已经结束。

| 证据层 | 状态与范围 |
| --- | --- |
| 实现 | Trading 独立目录、权限、消费者、worker、账户迁移、Markets 服务/API 清理、本地独立运行和通知维护已集成；Task 9 补成功交易回归并修复签名前后加载步骤遗漏尝试记录的问题，Task 10 增加默认报告、显式 apply 的精确数据库退役工具。 |
| 隔离副作用 | 真实 PostgreSQL、Trading SQLStore、凭据 cipher、目录校验、Solana 交易解析与 Ed25519 签名；Worm、Wallet、Solana 外部边界受控。没有真实下单、平仓或撤凭据。 |
| 真实只读 | 实际开发实例的会员/管理员、桌面/手机；shell smoke 2 项，四场景业务 16 项，最终修复源码重建后 healthy 4 项通过，均 0 skipped/flaky。 |
| 通知与收尾 | 原 main 五来源 report/apply/report 均成功，待发和在途均为 0；没有发送测试通知。两轮 owned 应用、替身和容器均按归属停止，原卷与证据保留。任务结果通知由控制器按整项任务统一处理。 |
| 现场数据 | 已在核实 server、owner、OID、活跃连接和全部保留库后精确删除 `worm_markets`；复查为 `already_absent`，正常重启未重建。九个非目标库保留，Trading／Wallet 全表计数与指纹保持。 |

## 契约与修复

沿用 [Trading 设计](../design/trading/worm-trading.md)、[订单执行](../design/trading/worm-order-execution.md)、[单笔 Cash Out](../design/trading/worm-position-cash-out.md)、[批量 Cash Out](../design/trading/worm-position-cash-out-batches.md) 与 [退役需求](../requirements/development-runtime/worm-markets-removal.md)。未新增 Markets 兼容层，也未修改公共业务状态机。

成功 Open → Wallet 签名 → Finalize → HMAC position confirmation 的新增回归首先失败：`MarkExecutionStepSigning` 和 `RecordExecutionStepSigned` 返回裸步骤映射，丢失此前 Open attempt；后续确认报 `EXECUTION_PLAN_INVALID`。两处改为完整加载步骤及 attempts/isolation，成功路径通过，恢复测试仍禁止重放。没有 SQL、proto、schema 或生成输入变化，本项未运行生成器、未修改生成产物。

首次综合单元测试另暴露 `internal/devruntime` 测试 helper 的 `/proc` 启动读取竞态；定向 20 次复现后，仅让 helper 等待本次进程的确切 run marker，带 1 秒上限。该包及最终完整指定单元命令通过；未修改进程管理产品逻辑。

## 隔离后端与 HTTP 证据

[`internal/wormtrading/retirement_integration_test.go`](../../internal/wormtrading/retirement_integration_test.go) 复用 Task 4 的真实 SQL/凭据基础，新增受限 `GetEvent`/`GetMarket` catalog、合法交易及成功外部回执；每次外部 mutation 在返回前记录计数和校验请求，没有真实网络 fallback。覆盖：

- 新目录参与预览和 fresh preflight；Open 1 次、Finalize 1 次，步骤完成；随后单笔 Close 1 次完成。
- Open/Close 的 `DISPATCHED` 崩溃与 `OUTCOME_UNKNOWN` 恢复均不重复发送；新 Service 从持久状态恢复。暂时读取失败可继续原 Cash Out，catalog 失败暂停的 Run 可恢复同一持久步骤。
- Run、单笔 Cash Out、batch 的钱包 mutation 锁互斥；batch child Close 成功保留真实 Solana adapter 基线，unknown 父任务恢复保留 baseline 并进入 reconciliation。
- USDC 基线 `20000000`、slot `100`：同额新 slot 不完成；多 1 atomic 但旧 slot 不完成；`20000001` 且 slot `101` 才完成 item 并由 worker 完成父 batch；超时暂停并保留 `USDC_CREDIT_NOT_OBSERVED`。
- unknown 使整条 Run 进入 `RECONCILIATION_REQUIRED`。显式 Terminate 后同一钱包/市场仍被 isolation 阻止，另一个市场可新建 Run。未采用早期 inventory 中“同一 unknown Run 内可继续另一 pair”的错误假设。

[`internal/server/retirement_integration_test.go`](../../internal/server/retirement_integration_test.go) 单独覆盖真实 HTTP development proof handler → 内部 Bearer gRPC → Trading → SQL 接受路径。Run、单笔、batch 均持久化 `DEVELOPMENT`、确切 session digest 及当前 access revision `7`；通过实际 controller 撤权到 `8` 后，新 proof HTTP 403 且未进入 gRPC，已受理队列仍可恢复，Run 实际重新 claim。测试把后台调度停在 queue read 边界，避免授权/撤权断言与消费竞态。

这两组证据分层：HTTP 测试证明 proof 签发及 durable 接受/恢复资格；worker 测试以明确的 store fixture binding 验证执行/崩溃恢复。没有声称同一条 HTTP 签发任务在同一个夹具中跑完所有外部副作用，也未用直接 store 授权冒充 HTTP proof。

## UI 与真实环境

新增隔离场景注册在既有 `theme-refactor.spec.ts` 导入的两个模块中，统一 `worm-retirement` 标题过滤。44 次通过 = 22 场景 × root 和 `/athena` 两个部署前缀；覆盖桌面 1440px、手机 390px 下七条路由、组合保存 200/403、保留输入、目录失败保留历史选择、preview 仅准备 frozen Run 并等待独立授权。三个 Jest 文件 29 tests 通过。单一深色、导航和错误状态沿用既定 Inter/JetBrains Mono 组件。

真实目标为 worktree `.worktrees/worm-markets-retirement`，`http://localhost:61901`，owner `worm-retirement-full`、独立 borrower `worm-retirement-live`。浏览器 state 来自应用自身 DisableAuth 的不同 member/admin development 账户；不是 harness 身份，不代表 Google/Phantom 交互登录经过验证。会话文件只在本机保留，不提交。

[`playwright.worm-retirement.config.ts`](../../ui/playwright.worm-retirement.config.ts) 复用基础配置 localhost、系统 Chrome 与报告设置，两个项目分别匹配 `@member`/`@admin`，精确匹配新增只读 spec。它要求：

```bash
# 在 ui/，采用 ui/.nvmrc 的 Node 24.14.1。
ATHENA_UI_E2E_MODE=smoke yarn playwright test --config=playwright.worm-retirement.config.ts
```

调用环境还必须设置 `ATHENA_UI_E2E_BASE_URL`、`ATHENA_UI_E2E_OUTPUT_DIR`、`ATHENA_WORM_RETIREMENT_MEMBER_STATE`、`ATHENA_WORM_RETIREMENT_ADMIN_STATE`（两者为已存在的绝对文件路径）及实际可读的 `ATHENA_WORM_RETIREMENT_EVENT_ID`。`ATHENA_WORM_RETIREMENT_SCENARIO` 取 healthy/provider-down/account-down/restored，`ATHENA_WORM_RETIREMENT_BASELINE` 可保存并比较列表内容。所有浏览器请求限定同源 GET/HEAD/OPTIONS；不会保存组合、连接或交易。

实际只读覆盖 Assets、Combinations、新建编辑器、Executions 四条会员 route；空库没有组合/Run，另外三条 edit/preview/detail 没有伪造实际覆盖。健康与恢复阶段实际目录返回 3 个子市场；编辑器 Add event 只发 GET，并显示市场。provider CONNECT/账户 TCP 代理故障时目录明确 503（API code 14），真实组件显示 alert 和 aria-invalid；旧组合/执行/连接列表、Wallet、bootstrap 仍 200 且 items 不变。账户故障只切断 Trading 新 account reader 代理，不会使 API 自己的账户路径自动失效。

四场景每场景 4 tests 全通过（member/admin × 两尺寸），另有初始 shell smoke 2 tests 和最终修复源码重建 healthy 4 tests。手机成功/失败截图由控制器人工查看。最初三轮 locator 失败及 provider 代理环境覆盖导致的无效故障均保留，未计为通过；校准 exact 图标菜单名、Profile region 和 mobile Primary navigation 范围，没有修改产品 UI 来迎合测试。

## Task 10 原 main 现场退役

现场操作只针对原 main default full-stack，namespace `ac252d3201cc3c6d872334f8e6a1bcf5`；没有部署远端或操作其他实例。main/rf4 先本地快进已审阅的工具提交 `0022bed4`，没有 stash、reset 或 push，主工作区另四个未提交文件的 SHA256 前后相同。配置只删除两项 Markets 通知键；Trading 原本没有稳定 key 且 33 张业务表均为空，因此初始化新 key，不是轮换或旧凭据解密验证。Wallet key、Wallet 内部／签名 token 与 Trading 内部 token 保持。

账户 schema 迁移 `0` 到 `5` 及 verify 全部通过。原两个账户的 access revision 从 `1` 更新为 `2`，模块矩阵各由八项变为七项；原 16 条 grants 中两条 Markets grant 删除，其余 14 条 grant 与 flags 逐项不变，Markets 权限没有转换成 Trading 权限。

通知工具对固定五个 `worm-markets.*` 来源执行 report/apply/report，三次均退出 0；`pending=0`、`sending=0`、`cancelled=0`、`retired_total=0` 且 `counts_verified=true`。既有 delivery 0、attempt 0、Topic 1、consumed 1、两个 sender 记录及 Bot `next_update_id=487823039` 保持；没有测试消息或替代消息外发。

数据库工具的 apply 前报告经控制器核对：目标 `worm_markets`、OID `17152`、owner `athena`、server identity `7685665987292844070`、活跃连接 0，并以九个真实 DSN 完整核验全部保留库。完全相同参数追加 `--apply` 后于 11:13:35 UTC 精确 DROP，复查返回 `already_absent`。原 1000 条 market 与 2969 条 price history 随专属库直接删除，没有归档或转存。保留库为 `postgres`、`athena`、`temporal`、`temporal_visibility`、`worm_trading`、`wallet`、`managed_oo`、`profit_sharing`、`token`；Trading 与 Wallet 全表 count/fingerprint 在删除后和正常重启后均与原值一致。

最终 preservation audit 退出 0，逐项确认 16 个历史迁移源文件、28 条现行 HTTP 路径和七条 Trading 路由未因退役操作改变；Markets 的三条旧 HTTP 路径继续为 404。

删除前真实 main shell smoke 为 2 passed；新业务只读 spec 为 4 passed。三条已删除 Markets HTTP 路由实际返回 404。正常 stop 后第二次 `make run` 连接相同 server identity，只看到九个保留库，未重建 Markets。重启后的首次真实浏览器套件为 3 passed / 1 failed：desktop `/account/profile` 停在 Loading Athena，bootstrap 请求在 5 秒内未返回；失败窗口没有 gRPC handler 记录，UI proxy 也没有 error，具体 pre-gRPC 延迟原因未定位。未修改产品或测试；随后 proxy/direct bootstrap 分别在 4ms/1ms 返回 200，同一套件复验为 4 passed / 0 skipped / 0 flaky，耗时 17.177s。该复验只证明随后运行成功，不声称首次超时已定位或修复。

真实 Trading 库为空，因此现场只能覆盖四条可用会员路由；三条历史详情路由仍由 Task 9 受控 fixture 覆盖。空库也没有旧 Trading 凭据可供解密验证，没有执行真实 Worm 下单、平仓或撤凭据。正常运行确实新增 sender 实例并推进 poller/liveness、Trader Sync 索引和 epoch/control 记录，因此保留结论限定为已核对的业务表指纹、权限、迁移和通知事实，不声称共享数据库逐字节未变。

11:27–11:28 UTC 最终先停止第二次 owned Trading borrower，再从原 main 执行 `make stop INSTANCE=full-stack`；六个应用、supervisor 和三个 owned 容器均停止，全部应用与动态基础设施端口释放，三个卷保留。通知 helper 在核对 PID 的 cwd/startTicks 后以 SIGTERM 结束，61907 释放；启动会话均正常退出，helper 的 143 符合信号终止。现场恢复任务前停止状态，没有 reset、删卷或停止归属不明资源。11:34 UTC 又核对隔离测试 PostgreSQL 的容器 ID、名称、挂载和实际 63533 映射后停止容器、释放端口并保留卷；首次 guard 因把空 HostConfig 动态端口误认作固定映射而在 mutation 前失败，改按 NetworkSettings 核对后才执行正确停止。至此本任务全部临时环境均已停止。

## 验证命令与结果

Task 9 的 Go 命令从 worktree 根目录执行，`ATHENA_TEST_PG_ADMIN_DSN` 指向已核实归属的隔离 PostgreSQL `127.0.0.1:63533`，各 pgtest 创建独立库；Task 9 未执行 `go test ./...`。Task 6 曾额外启动一次超出 brief 的 `go test ./... -count=1` 宽范围探测，因无关 `internal/etherscanmanager` 测试超过两分钟无新输出而停止该任务拥有的测试进程，不计为通过证据；各任务要求的指定范围已独立通过。

```bash
go test ./internal/wormtrading/... ./internal/server/... ./internal/accountaccess ./internal/devruntime ./internal/notification/... -count=1
go test -tags=integration ./internal/wormtrading/... ./internal/accountstate/store ./internal/accountstate/schema ./internal/notification/store ./internal/devruntime -count=1
go test -race -tags=integration ./internal/wormtrading ./internal/server -run TestRetirement -count=1 -json
go test -race -tags=integration ./internal/wormtrading -run TestRetirementBatchCloseAndUSDCCreditGate -count=1
go build ./cmd/athena-worm-trading ./cmd/athena-worm-trading-migrate ./cmd/athena-server
git diff --check
```

全部最终退出 0。race JSON 中 Trading 19、server 4 个 pass 事件（包括子测试，不能当作 23 个顶层测试）；补充父 batch 终态后受影响测试再次带 race 通过。首次完整 unit 命令因上述测试竞态失败，修复后的完整命令通过。集成日志最慢 devruntime 235.376s，未因运行时间截断。

```bash
# ui/；Node 24.14.1，Yarn 1.22.22
yarn tsc --noEmit --project ./src/app
yarn build
yarn jest --runInBand --coverage=false --runTestsByPath src/app/member/pages/worm-trading-scope.test.tsx src/app/member/pages/worm-combination-delete-scope.test.tsx src/app/member/pages/worm-execution-scope.test.tsx
yarn eslint e2e/worm-trading-retirement.spec.ts playwright.worm-retirement.config.ts e2e/theme-refactor/worm-assets-combinations.ts e2e/theme-refactor/worm-executions.ts
# worktree 根目录
make ui-acceptance UI_ACCEPTANCE_GREP='worm-retirement'
make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://localhost:61901
./dist/shellcheck -x -P . hack/deploy-scripts_test.sh hack/lib/account-state-deploy.sh hack/prod-remote-deploy.sh hack/prod-start-local.sh hack/production-compose_test.sh hack/worm-trading-deploy_test.sh
bash hack/worm-trading-deploy_test.sh
bash hack/trader-sync-deploy_test.sh
bash hack/deploy-scripts_test.sh
```

以上最终通过。UI build 有既存 bundle size advisory。ESLint 仅对四个实际适用 e2e/config 文件执行成功；三个 Jest 文件受现有 ignore，不声称其已 lint。第一次 ShellCheck 使用不存在的猜测路径失败，随后按 Tasks 6/7 实际脚本路径检查通过。

## SDS-R1–R8 证据映射

| 规则 | 本项证据 |
| --- | --- |
| R1 边界 | Trading 为独立进程；API 不持有 Trading runtime。真实 owner/borrower 配置和最终二进制记录；本次 owned 新实例没有 Markets 进程/库。 |
| R2 API | HTTP proof 经内部 Bearer gRPC 到 Trading；受限目录 RPC、当前 access revision 与撤权断言。 |
| R3 独立生命周期 | 三目标独立 build；实际 Trading SERVING、无 Bearer 业务 RPC 拒绝；持 active health Watch 停止成功 12.286 秒。 |
| R4 配置/故障 | provider/账户 owned 代理四场景共 24 次 GET；明确 503，仅目录受影响；恢复成功。共享账户 PG 是声明依赖，没有声称跨库原子撤权。 |
| R5 所有权 | borrower 先停止，owner 保持运行；随后 owner 六进程/三容器及三个 helper 按归属停止；未 reset、未删卷。helper 停止后的首次即时端口绑定断言因释放竞态失败，随后复核端口释放成功；最终证据采用复核结果并保留首次失败。 |
| R6 事务/恢复 | SQLStore 锁互斥、attempt unknown、不重放、USDC evidence；账户及通知同库撤权/发送许可并发集成套件通过。 |
| R7 设计 | 上述长期需求与设计已同步实现及原 main 现场事实；首次重启浏览器超时、空库覆盖和共享库正常运行变化均作为边界保留。 |
| R8 消费者/验证 | 指定 Go/unit/integration/race、UI fixture/Jest/build/lint、真实只读/故障与 shell 回归均有实际日志。 |

## 本机证据与资源收尾

本机工作证据根为 `.superpowers/sdd/2026-09-16-worm-trading-market-query/`，不提交运行凭据或 storageState：

- `t9-go-unit-final.log`、`t9-go-integration.log`、`t9-retirement-race.jsonl`、`t9-batch-final-boundary.log`、`t9-go-build.log`、`t9-jest-first.log`、`t9-tsc.log`、`t9-ui-build.log`、`t9-eslint-final.log`、`t9-shellcheck-final.log`、`t9-shell-regression.log`。
- 隔离 UI：`.tmp/athena-ui-acceptance/2026-09-16T10-29-52-232Z-50771224/report.md`；真实 shell：`.tmp/athena-ui-acceptance/2026-09-16T10-15-52-303Z-da5228a9/report.md`。
- 真实业务：`.tmp/athena-ui-acceptance/worm-retirement-real-{healthy,provider-down,account-down,restored}-4/`；最终 `.tmp/athena-ui-acceptance/worm-retirement-real-healthy-final/`，包括报告、trace、截图、route coverage 和 GET 结果。
- `task-9-live-evidence.md`、`evidence/task-9-fault-probe.json`、`evidence/task-9-real-browser-4.json`、`evidence/task-9-real-browser-final.json`、`evidence/task-9-final-runtime-source.json`、`evidence/task-9-final-trading-stop.json`、`evidence/task-9-full-stack-final-stop.json`、`evidence/task-9-helpers-stopped.json`。

最终 Trading 源码 `execution_runs.go` SHA256 为 `c9073fba9495686c2a213221095198b88b8ca51ac1812959b42a6cf497cd61b1`，实际 binary SHA256 为 `5481ef5fd89dfea608bcece3883a695aa069f7d1533d063855c565331423fe84`。最终 PID 1305291 已退出，61906 释放。`make stop-instance INSTANCE=worm-retirement-live` exit 0，12.286 秒；health Watch 总寿命 32.65 秒包含此前等待，不是停止耗时。

`make stop INSTANCE=worm-retirement-full` exit 0，六应用退出，61900–61905、58986/58988/58989 释放，三个 owned 容器停止且卷保留。三个 helper 核对 cwd/startTicks 后终止，61907–61909 释放；首次立即检查退出的短暂竞态及后续成功复核均有记录。

Task 10 现场证据位于同一证据根下的 `task-10-field-evidence.md`、`evidence/field-*.json`、`evidence/task-10-field-*.json` 及对应 stop log。最终原 main 的六应用、supervisor、三个容器、两个 field borrower、通知 helper 和隔离测试 PG 均已停止，现场及测试端口释放，所属卷保留。没有遗留本任务启动的运行实例。
