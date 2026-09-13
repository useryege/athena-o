# AI 开发工具就绪与首轮扫描

日期：2026-09-13。范围：第一批工具安装、独立检查入口及真实运行验证；现有代码和依赖发现仅记录，不在本次批量修复。用法见[开发工具链](../developer-guide/toolchain-guide.md#ai-开发检查工具)，批准范围见[实现计划](../superpowers/plans/2026-09-13-ai-dev-tools.md)。

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
