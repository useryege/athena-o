# ESLint 规则匹配修复与 Go lint 分类分析实施计划

用户已批准本方案。执行流程为 Superpowers subagent-driven-development；先完成 ESLint，再执行 Go lint 全量扫描和分类。当前 Go 代码不修复。

## 全局约束

- 只修复 ESLint files 匹配，不升级依赖，不变更受限导入模式、错误级别或测试文件忽略策略。
- Go 阶段仅扫描、分析和编写后续方案；不改 Go 源码、模块文件、生成文件和 lint 配置，不使用 --fix。
- 不启动浏览器、Docker、开发服务，不操作真实远端或业务数据。使用 Node 24.14.1 和项目现有工具。
- 隔离 worktree 保留开始时的源码状态；基线比对后仅回写本任务文件，保留其他任务改动，不提交或推送。

## Task 1: ESLint 配置、回归和使用说明

文件：ui/eslint.config.mjs、ui/package.json、新增 ui/scripts/eslint-config.test.mjs、docs/developer-guide/toolchain-guide.md、新增 docs/testing/eslint-config-matching.md。

- [x] TDD：先写加载真实 ESLint 配置的 Node 原生内存用例，验证旧规则漏报时测试失败，保存红日志。
- [x] 所有 files 中 ./src/ 改为 src/；保持受限导入模式、错误级别和忽略策略原样。
- [x] 测试从自身路径确定 UI 根目录；不依赖调用 cwd，不创建违规业务文件，不复制生产规则来替代真实配置。
- [x] 负例覆盖 member→admin、admin→member、shared/session/components→两端，.ts/.tsx、嵌套目录与既有一至三层相对路径模式；正例覆盖同端、共用模块和第三方包，验证既有测试文件仍忽略。需确认具体 no-restricted-imports 错误和非解析失败。
- [x] yarn lint 先运行 node --test scripts/eslint-config.test.mjs，再执行原 TypeScript 和 ESLint 命令；不更改锁文件。
- [x] Node24.14.1 下完成配置回归、从仓库根目录调用回归、完整 yarn lint、相关差异检查；使用文档和验收记录写明根因、结果、范围。

## Task 2: Go lint 全量扫描、分类与后续方案

- [x] 记录源码状态与工具版本，使用原配置扫描 ./...，保留 JSON、日志、真实退出码。完整结束且有诊断是有效扫描，不要求清零。
- [x] 以新结果为基线，与同轮历史1,684条按文件/规则/诊断文本对比，不以行号变化误判新增。
- [x] 按行为风险、等价清理、API/配置弃用、需保留/暂缓分类；对可能影响行为的诊断追踪来源和调用方，标明已确认、条件性风险或待验证，不能把提示全部当bug。
- [x] 输出 docs/testing/go-lint-triage.md，给出数量、根因、证据、具体位置和边界；行为风险逐项或按同一调用契约归组，机械风格项按规则分组。
- [x] 输出 docs/superpowers/plans/2026-09-13-go-lint-followup.md，按风险给出下一批修复的明确范围、方式和验收；涉及业务决策、大改造或高成本的项列为需确认，不直接实施。

## Task 3: 最终审查与交付

- [x] 独立任务审查与全任务最终审查，核对差异、链接、代码不变约束和扫描证据。
- [x] 基线核对后同步本任务文件回根，验证与已验收版本一致。
- [x] 两阶段完成后从根调用一次 make notify-task-complete，等待命令结果并如实报告。命令退出 0，SMTP 首次尝试接受邮件；见 [完成通知日志](../../../.tmp/eslint-go-lint/completion-email.log)。
