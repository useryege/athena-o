# ATHENA 浏览器验收统一实施计划

> 执行方式：subagent-driven-development，按任务实现、测试、独立审查；不再次请求已确认决策。

**目标：** 通过一个命令管理隔离浏览器回归和已有开发环境冒烟，并让项目 skill 正确引导 AI。

**架构：** 薄 Bash 入口定位 WSL Node，Node 编排子进程和产物，Playwright Test 保持唯一测试框架。现有 Go harness 提供随机数据库、真实角色会话、manifest 和 stop 协议。

**技术：** Bash、Node 内置模块、项目本地 Yarn/Playwright、Go、PostgreSQL 16 Docker 镜像。

**规格：** `docs/superpowers/specs/2026-09-13-athena-browser-acceptance-design.md`。

## 全局约束与接口

- 在 `.worktrees/browser-acceptance` 实现。最终只带回本任务改动，保留原根目录 Procfile 修改。不推送、不部署。
- Make 接口：`UI_ACCEPTANCE_MODE=isolated|smoke`（默认 isolated）、`UI_ACCEPTANCE_CHECK_ONLY=1`、`UI_ACCEPTANCE_BASE_URL`（smoke 默认 http://localhost:4000）。不开放任意 Playwright 参数转发，以免破坏有状态 live 套件。
- Node 选择：`ATHENA_UI_ACCEPTANCE_NODE` > PATH > NVM 默认；项目本地 Yarn/Playwright。同一 WSL 运行域，不修改全局 PATH/配置。
- 编排器传给 Playwright：`ATHENA_UI_E2E_MODE=isolated|smoke`、`ATHENA_UI_E2E_BASE_URL`、`ATHENA_UI_E2E_PATH_PREFIX`、`ATHENA_UI_E2E_MANIFEST`（仅 isolated）、`ATHENA_UI_E2E_OUTPUT_DIR`（本 project 绝对输出目录）。
- 配置原生 reporter：list + JSON(`${OUTPUT_DIR}/results.json`) + HTML(`${OUTPUT_DIR}/html`，不自动打开)。测试附件 `${OUTPUT_DIR}/artifacts`。
- isolated 只选择 ui-fixtures/live，smoke 只选择 smoke。isolated 使用 channel: chromium 的配套完整 Chromium（不设 executablePath）；smoke 为 `ATHENA_CHROME_PATH` 或 `/usr/bin/google-chrome`。不静默回退浏览器。
- 命令不自动安装/下载；依赖缺失给 blocker。所有新运行证据置 `.tmp/athena-ui-acceptance/<run-id>`，检查模式不写文件。
- 首批回归沿用完整现有 Trader Sync；无需改 Go／产品源代码。必要测试基础设施修改由根代理裁定。

## Task 1: 统一运行入口与生命周期

**负责文件：** Makefile、新 `hack/ui-acceptance.sh`、新 `ui/scripts/acceptance-runner.mjs` 及其 Node 测试；不编辑 Playwright config/spec 和 skill。

- [x] 先写有意义的失败测试：Node 选择/缺失、check-only 零副作用、模式依赖差异、非法目标、构建/PG/harness/Playwright失败、SIGINT/TERM、清理失败、并发锁与共享资源不受影响。
- [x] 薄 shell 定位 Node；Node 编排用数组 spawn 参数，避免 shell 拼接。`make ui-acceptance` 委托 shell，Node 从项目本地包运行 Yarn/Playwright。
- [x] 预检 selected Node/依赖/浏览器实际存在及可执行。isolated 额外检查 Go/Docker 镜像；使用 `docker.io/library/postgres:${ATHENA_POSTGRES_IMAGE_TAG:-16}`，不 pull。
- [x] isolated 获得仓库锁，build 一次并快照 ui/dist/app。临时 PG 使用唯一 name/label、随机密码、loopback 随机端口，无持久卷，退出删除自身容器及匿名卷。
- [x] 根路径与 /athena 顺序运行：新 dir -> Go harness -> 完整 ui-fixtures -> 完整 live -> stop -> wait。监控 harness 异常退出；启动期限 5min，Go测试30min，正常stop等待30s，然后本轮进程组 TERM/KILL有界回收。
- [x] smoke 仅调用 smoke project，不启动/停止用户环境。默认地址仅允许 HTTP localhost/127.0.0.1/[::1]，不允许用户信息或 query/fragment。
- [x] 结果写 run.json/report.md（含证据边界、各阶段退出/失败、版本、构建摘要、清理）；保留首个失败与清理失败。check-only输出JSON，不建目录。
- [x] 聚焦测试通过、自审，写报告供独立审查。不要运行完整实际回归（根代理统一执行）。

## Task 2: Playwright 模式、应用壳冒烟和 trace

**负责文件：** `ui/playwright.config.ts`、新 `ui/e2e/shell-smoke.spec.ts`、所需配置/冒烟测试辅助文件、`ui/e2e/trader-sync-live.spec.ts` 仅trace生命周期改动；不编辑 Make/runner/skill。

- [x] 先写配置和冒烟的失败测试。smoke 用临时本地HTTP场景验证真实 Playwright导航和断言，不只测试自造布尔函数。
- [x] 采用全局接口按mode声明projects；smoke不得收集需要manifest的live文件。检查目标loopback，isolated必须127.0.0.1。单worker、零retry、无parallel。
- [x] isolated不设executablePath；smoke显式系统Chrome。原生reporter写入输出契约，失败截图trace保留。
- [x] smoke 独立member/admin上下文，默认1440x900；验证两HTML入口/base、React挂载、必要同源资源、真实bootstrap结构和会话状态对应内容。匿名合法，不登录、不写业务、不访问外部身份提供方。
- [x] 新增受控场景：正常本地身份shell、匿名登录、bootstrap失败/无效、资源404、pageerror、服务不可达，确认失败有证据且非0。
- [x] 为手动创建的live角色上下文保留失败trace：使用已验证的 Playwright 原生 retain-on-failure 管理 trace，afterEach 关闭仍存活上下文；验证提前关闭的 context 也保留证据，避免双重停止和掩盖原失败；不改变业务断言。
- [x] 运行聚焦测试与e2e TypeScript检查，自审并写报告；完整原有回归由根代理执行。

## Task 3: 项目 skill 与维护文档

**负责文件：** 旧/新验收skill目录、docs/developer-guide/superpowers-development.md、running-locally.md；不编辑script/config/spec/产品。

- [x] 留存旧skill实际不适配证据：不存在的包路径、错误服务/登录/响应式政策；在只读模拟任务中比较引导。
- [x] 重命名为athena-browser-acceptance，更新SKILL.md/agents/openai.yaml，删除无外部调用的旧assets/scripts/references，无alias。
- [x] SKILL只保留路由与ATHENA特有契约，指向统一make命令；日常探索按当前工具可用性选内置浏览器，正式回归使用命令。适配Google/Phantom，不把测试storageState当认证验收。
- [x] 维护文档解释3种调用、Node/Wsl依赖、浏览器差异、准备依赖命令、产物与证据边界。历史记录不改写。
- [x] 运行skill静态校验和链接检查；根代理安排独立行为验证与审查。记录变更和验证报告。

## Task 4: 集成验证与交付

- [x] 审查Task1/2/3的规格与质量；解决接口差异，完成统一命令check-only。
- [x] 运行脚本测试、受控smoke、e2e类型检查，再使用统一命令实际跑根路径和/athena全套 fixture/live。
- [x] 若当前make run可用，执行真实系统Chrome smoke；不可用则保留真实blocker，受控smoke证据单列。
- [x] 独立验证skill行为：页面探索、正式回归、缺失Node、用户环境所有权、fixture/live证据、只报告范围。
- [x] 最终独立审查并修复必要问题；验证git差异、资源清理、产物及文档。
- [x] 通过git apply --check将仅本次改动带回原工作区，再应用并核对Procfile未变。
- [x] 从原仓库根发送一次中文完成邮件，等命令结束后交付简洁结果和报告路径。

## 已执行的验证

- 编排器进程边界测试 23/23；受控 smoke 12/12；原生 manual context trace 2/2；E2E TypeScript 通过。
- 使用统一命令在全新环境运行：根路径 fixture 33/33、live 10/10；`/athena` fixture 33/33、live 10/10，共 86/86，零跳过、零 flaky。
- 成功运行证据在独立 worktree 的 `.tmp/athena-ui-acceptance/2026-09-12T18-56-12-580Z-47d25c19/`。两个数据库不同；cleanup 通过，无本轮容器与锁残留；原有 7 个 Docker 容器 ID/状态未变。
- 当前 `localhost:4000` 没有启动，两入口 smoke 均得到连接拒绝；这属于实际环境 blocker。证据在 `2026-09-12T18-58-05-062Z-860a0a49/`，未启停用户环境。受控 smoke 成功不能替代该实际结果。
- 初轮真实运行揭示重复 trace 所有权冲突；独立审查复现慢资源漏报、认证 popup 绕过与 Go 私有代理绕过。修正后分别验证原场景，并用全新隔离环境完整复验。没有放宽业务断言。
