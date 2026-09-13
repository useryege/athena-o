# AI 开发工具就绪与首轮扫描

日期：2026-09-13。范围：第一批工具安装、独立检查入口及真实运行验证；现有代码和依赖发现仅记录，不在本次批量修复。用法见[开发工具链](../developer-guide/toolchain-guide.md#ai-开发检查工具)，批准范围见[实现计划](../superpowers/plans/2026-09-13-ai-dev-tools.md)。

下文保留首轮扫描基线；后续 Go 1.27.1 升级见 [Go 升级验收记录](go-1.27.1-upgrade.md)，第三方依赖修复及最新漏洞复扫见 [第三方依赖验收记录](third-party-vulnerabilities.md)；共用状态标签对比度修复见[后续 UI 修复验收](#后续-ui-对比度修复验收)。

## 安装状态

| 工具 | 实测版本 | 安装位置 |
| --- | --- | --- |
| ShellCheck | 0.11.0 | `dist/shellcheck` |
| grpcurl | 1.9.4 | `dist/grpcurl` |
| govulncheck | 1.7.0 | `dist/govulncheck` |
| psql | 16.15（Ubuntu 16.15-0ubuntu0.24.04.1） | `/usr/bin/psql` |
| @axe-core/playwright | 4.13.0 | `ui/node_modules/@axe-core/playwright` |

首次安装与重复安装均成功，`make ai-dev-tools-check` 五项就绪。Node 24.14.1、Yarn 1.22.22、Go 1.25.5 保持原版本；Yarn 锁文件仅增加 axe 包及其 axe-core 依赖。

本机 sudo 未缓存凭证，PostgreSQL 客户端通过 WSL 的 root 用户执行 Ubuntu apt 安装。未修改 sudo 策略或用户权限，未安装 PostgreSQL 服务端；仓库安装器保留常规 sudo 安装与明确的手工安装提示。

## 实际功能验证

- 临时本地 gRPC 服务：反射列举与健康服务定义查询成功，健康 RPC 返回 `SERVING`；进程已退出。
- 临时 Docker PostgreSQL 16：psql 执行只读事务，`SELECT 1`、版本和系统表查询成功；容器及匿名卷已删除。客户端为16.15，本地缓存镜像的服务端为16.14。
- 默认隔离、默认冒烟和独立 a11y 三种前置检查均返回 `status: ready`。前置检查不代表已有开发服务可访问。
- 安装脚本语法、针对新增脚本的 ShellCheck，以及缺失提示/日志/退出码行为测试通过。
- 原始连接和前置检查证据保存在 `.tmp/ai-dev-tools/2026-09-13-validation/`。

## ShellCheck 首轮发现

`make lint-shell` 完成扫描并返回非零：14个文件共49条诊断，包含4条 warning、43条 info、2条 style，无 error 级诊断。这些是静态分析提示，不等同于49个已证实的运行故障。

| 规则 | 数量 | 后续核查方向 |
| --- | --- | --- |
| SC2029 | 31 | SSH 命令参数由本地还是远端展开是否符合意图 |
| SC2295 | 6 | 参数展开中的模式字符与引用 |
| SC1091 | 4 | 动态 source 的静态分析路径 |
| SC2034 | 3 | 未使用变量；包含原有工具版本配置的单文件误报 |
| SC2181 | 2 | 直接检查命令状态 |
| SC2016 | 1 | 单引号中的变量展开意图 |
| SC2329 | 1 | 间接调用函数是否被静态分析识别 |
| SC2154 | 1 | 与原有动态 source 相关的未赋值提示 |

新增安装器、命令脚本和脚本行为测试的定向 ShellCheck 检查通过。原有脚本及配置提示未被批量改写或全局屏蔽。完整日志路径见本地 `.superpowers/sdd/ai-dev-tools/shell-first-run.log` 的首行；原始扫描日志保存在 `.tmp/ai-dev-tools/shellcheck-*.log`。

后续按用户批准范围修复 ShellCheck 提示及远端命令引用，过程和最新结果见
[ShellCheck 修复验收记录](shellcheck-cleanup.md)。上文保留工具接入时的49条历史基线。

## Go 漏洞首轮发现

`make vuln-check` 完成默认构建条件的 `./...` 扫描，govulncheck 原生退出码为3（Make 将失败汇总为2）。报告指出37项可达漏洞；另有14项导入包、30项所需模块中的发现，当前代码未显示调用对应漏洞路径。可达分析不等于已经证明可利用。

| 来源 | 可达发现数 | 扫描报告中的最高修复版本 |
| --- | --- | --- |
| Go 标准库 | 27 | Go 1.25.13 |
| golang.org/x/image | 3 | v0.45.0 |
| github.com/go-git/go-git/v5 | 3 | v5.19.1 |
| google.golang.org/grpc | 1 | v1.82.1 |
| golang.org/x/text | 1 | v0.39.0 |
| github.com/xuri/excelize/v2 | 1 | v2.11.0 |
| golang.org/x/net | 1 | v0.55.0 |

修复版本来自本次工具报告，尚未对升级兼容性做验证；本次未升级 Go 或上述依赖。完整漏洞 ID、调用路径和修复版本保存在 `.tmp/ai-dev-tools/govulncheck-*.log`；最新漏洞数据会随后续扫描变化。


## 无障碍扫描与运行器验证

最终完整运行：`2026-09-13T05-20-04-272Z-a3669cc6`。UI 构建成功；32个用例全部执行并保存32份原始 axe JSON，其中28通过、4失败，零跳过、零 flaky。根路径和 `/athena` 各14通过、2失败。运行器正确记录扫描违规，基础设施失败为空，临时服务、数据库容器和运行锁清理成功。

4个失败场景均为管理员 Service Status 的浅色主题，覆盖两个视口及两个前缀。每个场景检测到3个绿色状态标签违反 `color-contrast`：文字 `#389e0d`、背景 `#f6ffed`，比值3.37，低于普通文字要求的4.5。axe 将影响级别标记为 serious。该产品问题未在本次修复。

成员列表、添加表单及取消弹窗的最终扫描全部通过。接入过程中修复了扫描时机：等待弹窗入场样式实际移除，避免将动画过渡中的颜色当作最终状态；未禁用动画、过滤规则或修改产品样式。前两轮调查数据保留供追溯，不作为最终存量问题清单。

运行器39项测试全部通过，包括默认项目选择、违规后继续第二前缀，以及配置/浏览器启动错误中止并清理。只有带原始 axe 违规附件的断言失败才作为扫描发现收集；缺少报告或基础设施故障不会被描述成无违规。E2E TypeScript 检查、格式检查和差异检查通过；独立审查发现的失败分类问题已修复并通过定向复审。

最终原始产物：

- [运行摘要](../../.tmp/athena-ui-acceptance/2026-09-13T05-20-04-272Z-a3669cc6/report.md)
- [根路径 Playwright 报告](../../.tmp/athena-ui-acceptance/2026-09-13T05-20-04-272Z-a3669cc6/root/a11y/html/index.html)
- [前缀路径 Playwright 报告](../../.tmp/athena-ui-acceptance/2026-09-13T05-20-04-272Z-a3669cc6/athena/a11y/html/index.html)

上述原始产物为本机 Git 忽略目录中的证据，复制仓库或清理缓存后需重新运行命令生成；本文保留此次验证结论。


## 后续 UI 对比度修复验收

日期：2026-09-13。范围为已批准的[共用状态标签修复](../superpowers/plans/2026-09-13-ui-status-contrast.md)，上文保留工具接入时的历史失败。

`StatusTag` 仅在 `positive && !negative` 时添加成功状态类；浅色文字由 `#389e0d` 改为 `#237804`，背景仍为 `#f6ffed`。axe 实测三个问题标签的对比度由 3.37:1 升至 5.43:1（直接计算为 5.4379:1，四舍五入为 5.44:1），12px 字号不变，超过 4.5:1 要求。深色主题继续使用现有配色。未修改状态语义、组件接口、背景、边框、字号、依赖或 axe 规则。

| 检查 | 结果与证据 |
| --- | --- |
| 修改前完整 `make ui-a11y` | 28通过、4失败；均为两前缀、两视口的浅色服务状态标签 `color-contrast`，每场景3个节点。构建成功，清理通过。 |
| 修改后完整 `make ui-a11y` | 32/32通过；根路径和 `/athena` 各16通过，零跳过、零 flaky、零 axe 违规，构建及清理通过。 |
| TypeScript | `node ui/node_modules/typescript/bin/tsc --noEmit --project ui/src/app` 通过。 |
| 页面抽查 | Accounts、Etherscan Gateways、Trader Sync 各明暗主题；Service Status 加桌面/移动。修复前后各10张截图，共比较70个标签，仅15处预期的浅色共用成功文字/对比度变化。负面、中性、深色及 Trader Sync 局部覆盖均不变。 |
| 隔离 `make ui-acceptance` | 86/86通过；两前缀各33个 ui-fixtures、10个 live，零跳过、零 flaky；构建成功、临时服务/数据库/锁清理通过，命令退出0。 |
| 真实 `make ui-acceptance UI_ACCEPTANCE_MODE=smoke` | 2/2通过，系统 Chrome，目标 `http://localhost:4000`。会员/管理员 bootstrap 分别为本地会员与管理员身份，两应用 shell 均通过；命令退出0。 |

抽查由浏览器拦截 API、加载当前根工作区的真实前端资源，属于 `ui-fixtures` 证据，不是对真实业务数据或认证流程的验收。截图目视检查服务状态页的明暗主题和移动布局；before 原始样式未采集几何字段，因此不将截图检查描述为自动像素/尺寸回归。深色成功标签的计算对比度保持7.04:1。自动无障碍结论仅覆盖现有32个场景，不代表全站检查。

本次追加的 Prettier 检查仍提示两个源文件原有格式问题（修改前根工作区也失败），涉及未修改的 JSX 闭合排版和媒体查询缩进；没有批量格式化或将该检查宣称为通过。任务差异检查通过。

本机原始产物（Git 忽略目录，清理后需重新运行）：

- [修改前 a11y 摘要](../../.worktrees/ui-status-contrast/.tmp/athena-ui-acceptance/2026-09-13T07-27-59-160Z-e726f8cf/report.md)
- [修改后 a11y 摘要](../../.worktrees/ui-status-contrast/.tmp/athena-ui-acceptance/2026-09-13T07-30-54-877Z-a016a1d4/report.md)，包含32份原始 axe 结果。
- [隔离功能验收摘要](../../.worktrees/ui-status-contrast/.tmp/athena-ui-acceptance/2026-09-13T07-32-19-260Z-3ef7202e/report.md)
- [真实 smoke 摘要](../../.tmp/athena-ui-acceptance/2026-09-13T07-33-58-458Z-0ccb3039/report.md)
- [抽查说明及重跑命令](../../.tmp/ui-status-contrast/probe-notes.md)、[修改前样式](../../.tmp/ui-status-contrast/before/status-tag-probe.json)、[修改后样式](../../.tmp/ui-status-contrast/after/status-tag-probe.json)、[70个标签比较结果](../../.tmp/ui-status-contrast/probe-comparison.json)。

真实环境复用 `/home/yege/work/athena` 的已有服务：Goreman PID `307846`、Vite PID `308228`，原持久会话 `29856`，日志 `.tmp/third-party-vulnerabilities/runtime.log`。本次仅在对比基线无并发修改后回写两处 UI 源文件，未重启服务或改动数据；服务继续运行。需要停止时从该根目录执行 `make stop`。
