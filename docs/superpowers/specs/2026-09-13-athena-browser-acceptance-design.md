# ATHENA 浏览器验收入口与 skill 统一

状态：用户已于 2026-09-13 确认方案并要求执行。

## 目标

建立隔离业务回归与已有开发环境冒烟的统一入口。日常探索使用当前可用的内置浏览器；正式回归使用项目 Playwright Test。此次接入现有 Trader Sync 测试和会员／管理员应用壳，不增加全站业务测试或外部浏览器工具。

## 已确认决策

- `make ui-acceptance` 默认执行隔离回归，配套 Chromium，根路径和 `/athena` 各使用全新 harness。
- `make ui-acceptance UI_ACCEPTANCE_MODE=smoke` 使用系统 Chrome 检查真实开发环境，工具本身不启停 `make run`。执行验收的代理按 [项目环境准备规则](../../../AGENTS.md#本地验收环境准备与完成标准)确认目标归属、复用或主动启动环境、排查并重验；验收后默认保留服务运行并报告状态。
- `UI_ACCEPTANCE_CHECK_ONLY=1` 只读预检所选模式，不构建、不创建运行目录、不启动服务。
- smoke 地址默认 `http://localhost:4000`，通过 `UI_ACCEPTANCE_BASE_URL` 覆盖，仅接受 HTTP loopback 地址。隔离地址只取本轮 manifest。
- 使用 WSL Node，依次检查显式 `ATHENA_UI_ACCEPTANCE_NODE`、PATH、NVM 默认版本；满足项目 Node engines。调用项目本地 Yarn/Playwright，不依赖全局 Yarn，不安装或下载依赖。
- 隔离回归预检 Docker、Go、项目依赖、配套浏览器及本地 PostgreSQL 镜像；冒烟不要求 Docker、Go、数据库或镜像。
- 不改变产品 API、认证、业务行为、现有 Go harness 与用户原有 Procfile 修改。

## 隔离环境生命周期

取得仓库级锁，构建 UI 一次，并复制构建快照到本轮目录、记录文件摘要。独立 PostgreSQL 容器使用项目镜像版本（默认 16）、临时凭据、loopback 随机端口、无开发环境持久卷。覆盖容器匿名卷的清理。

复用 `TestUIHarness` 的现有 manifest/stop 契约。每个 prefix 新建目录及数据库，后台启动 `go test -v -tags=integration,uiharness ./internal/tradersync/acceptance -run '^TestUIHarness$' -count=1 -timeout=30m`。等待可解析 manifest 且进程健康，再运行完整 `ui-fixtures`，随后完整 `live`。live 单 worker、零自动重试、无 grep/shard，失败后重新运行必须新环境。任一阶段失败终止后续阶段。

所有成功、失败、中断、超时路径都清理本轮资源。先写 stop 并等待，必要时只终止本轮进程组，再删除本轮容器。禁止 `make stop`、reset、共享端口扫杀和共享卷删除。清理失败不能标记成功；原失败与清理失败都保留。`SIGHUP`、`SIGINT`、`SIGTERM` 分别退出 129、130、143，重复信号只走一次清理。`SIGKILL` 和主机崩溃后的残留仅按运行文档人工核实归属并恢复，不自动回收失效锁。

## Playwright 和冒烟

复用现有配置与测试，新增 smoke project。isolated 使用配套 Chromium；smoke 使用 `ATHENA_CHROME_PATH` 或 `/usr/bin/google-chrome`；相对路径以命令启动目录为基准转为绝对路径，预检、Playwright 与报告统一使用该路径。smoke 只收集自身 spec，不能要求或读取隔离 manifest。

会员与管理员各自独立上下文。断言主文档/必要同源资源、部署 base、应用挂载、真实 bootstrap 及与会话状态对应的登录页或应用壳，无未解释 pageerror。匿名是正常状态，不自动登录、不提交业务修改。监听、断言应覆盖延迟资源失败，不能只看到 HTTP 200 就通过。从导航开始给应用自身的 bootstrap 重试与页面就绪 15 秒；允许中间 HTTP/JSON 失败恢复，持续失败则附上最近响应与解析错误。测试不自行补发请求、刷新页面或增加重试次数；资源稳定后复查最新 bootstrap 和页面状态。realm/角色、未捕获异常和认证流程约束保持严格。

手动创建的 live 角色上下文也须留下失败 trace。保留现有断言；不得因更换浏览器而随意放宽产品指标。

## 输出和技能

新运行证据放在 `.tmp/athena-ui-acceptance/<run-id>/`，prefix/project 分目录。使用 Playwright 原生 JSON、HTML、附件、截图、trace，另有简短机器可读运行摘要和中文 Markdown 总结：模式、覆盖、工具版本、构建摘要、阶段结果、失败原因、清理状态。

fixture 只证明模拟响应下的真实页面；live 证明真实产品组件和临时 PostgreSQL，链、资料、Telegram 是本地替身；smoke 只证明现有开发环境的应用壳。没有执行的检查不能报告通过。

旧 skill 直接替换为 `athena-browser-acceptance`，更新 metadata。删除无外部调用的旧 runner/harness/preflight/reference，无兼容 alias。新 skill 引用维护中的项目命令和文档，保留真实/模拟证据、角色和环境所有权边界；不复制通用工具手册，不覆盖 AGENTS.md。现有独立 UI audit 脚本与历史测试记录不在本次重写范围。

## 验证

先留存旧 skill 错误入口的证据；新脚本先写失败测试，覆盖缺失依赖、Node 选择、目标错误、构建/PG/harness/测试失败、超时、中断、清理和资源所有权。smoke 使用受控 HTTP 场景验证匿名、正常 shell、bootstrap 错误和资源失败，当任务要求真实环境验收时，由代理准备或复用 make run 环境后验证；不能仅因服务未启动而跳过。无法自行解决的阻塞须记录尝试与原因，必要验收未完成不得宣称整个任务完成或发送完成通知。

实际运行两个 prefix 的完整 fixture/live。独立评审检查规格与质量，并以前后场景验证新 skill 的工具选择和证据分类。实现完整且验证结束后，从原仓库根使用默认 .env 发送一次完成邮件。
