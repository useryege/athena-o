# Superpowers 开发方式

ATHENA 使用 [obra/superpowers](https://github.com/obra/superpowers) 的完整上游技能集推进开发。具体行为以仓库内的原版 `SKILL.md` 为准；本文说明安装、入口和项目资料的位置，不另设一套审批流程。

## 安装与加载

- 完整的 14 个技能及其脚本、参考资料安装在 [`.agents/skills/`](../../.agents/skills/)，与 Impeccable 并列，通过 Codex 的仓库技能发现机制加载。
- 本次固定上游提交 [`b36e0829c6d0140e93cfef2ca599b1b07d4a7797`](https://github.com/obra/superpowers/tree/b36e0829c6d0140e93cfef2ca599b1b07d4a7797)，该提交的插件清单版本为 `6.3.0`。技能内容未做本地改写；来源记录见 [superpowers.json](../../.agents/superpowers.json)，许可证见 [MIT](../../.agents/licenses/superpowers-MIT.txt)。
- 这是随仓库分发的完整技能安装，不是 Codex 插件市场的全局安装。上游该版本的 Codex 插件声明仅加载 `skills/`，`hooks` 为空。入口由根目录 [AGENTS.md](../../AGENTS.md) 指向 [using-superpowers](../../.agents/skills/using-superpowers/SKILL.md)。
- 新安装的技能在下一轮对话可用。直接调用使用 `$using-superpowers`、`$brainstorming` 等名称；上游文中的 `superpowers:<name>` 对应本仓库同名 `<name>` 技能。
- Codex 工具映射见 [codex-tools.md](../../.agents/skills/using-superpowers/references/codex-tools.md)。工具和模型以当前会话实际提供的能力为准。

Codex 的仓库技能发现方式见 [OpenAI 官方技能文档](https://learn.chatgpt.com/docs/build-skills)。

## 日常入口

| 工作 | 上游技能 |
| --- | --- |
| 识别任务需要的技能 | [using-superpowers](../../.agents/skills/using-superpowers/SKILL.md) |
| 澄清目标、比较方案、审阅设计 | [brainstorming](../../.agents/skills/brainstorming/SKILL.md) |
| 把设计拆成可执行任务 | [writing-plans](../../.agents/skills/writing-plans/SKILL.md) |
| 按任务实现和审查 | [subagent-driven-development](../../.agents/skills/subagent-driven-development/SKILL.md)、[executing-plans](../../.agents/skills/executing-plans/SKILL.md) |
| 并行处理独立问题 | [dispatching-parallel-agents](../../.agents/skills/dispatching-parallel-agents/SKILL.md) |
| 行为实现与故障修复 | [test-driven-development](../../.agents/skills/test-driven-development/SKILL.md)、[systematic-debugging](../../.agents/skills/systematic-debugging/SKILL.md) |
| 发起与处理审查 | [requesting-code-review](../../.agents/skills/requesting-code-review/SKILL.md)、[receiving-code-review](../../.agents/skills/receiving-code-review/SKILL.md) |
| 工作隔离、验证和交付 | [using-git-worktrees](../../.agents/skills/using-git-worktrees/SKILL.md)、[verification-before-completion](../../.agents/skills/verification-before-completion/SKILL.md)、[finishing-a-development-branch](../../.agents/skills/finishing-a-development-branch/SKILL.md) |
| 改进技能 | [writing-skills](../../.agents/skills/writing-skills/SKILL.md) |
| ATHENA 页面检查、冒烟与浏览器回归 | [athena-browser-acceptance](../../.codex/skills/athena-browser-acceptance/SKILL.md) |

由 `brainstorming` 按任务选取 Spike、Bounded 或 Architectural 路径。局部修改不必一律形成完整 spec 和 plan；需要书面方案的任务按上游技能创建相应文件。方案批准、任务执行和审查遵循所选技能，沿用用户已经明确给出的决定与授权。

## 项目资料与任务产物

| 位置 | 用途 |
| --- | --- |
| [`docs/requirements/`](../requirements/README.md) | 长期业务目标、范围、规则和待决问题 |
| [`docs/design/`](../design/README.md) | 长期架构、契约、关键流程和实际源码索引 |
| `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md` | Superpowers 需要的任务设计说明 |
| `docs/superpowers/plans/YYYY-MM-DD-<topic>.md` | Superpowers 需要的实现计划 |
| `.superpowers/` | 临时执行记录、子代理交接和可视化会话，Git 忽略 |
| `.worktrees/` | 项目内隔离工作树，Git 忽略 |

任务 spec 和 plan 引用相关长期文档，记录本次要改变的内容。完成实现时同步受影响的长期文档与源码链接。已有的 `讨论中`、`已确认`、`设计中`、`已确认待实现`、`已实现` 状态继续描述真实情况，不作为额外的阶段启动条件；切换流程不代表批准尚未决定的业务规则，也不代表已有方案已经实现。

## 项目技能的职责

Superpowers 负责开发方法。Impeccable 负责 `ui/` 下的 UI/UX 能力，并复用任务设计讨论与批准结果。`grpc-rpc-naming`、`sync-athena-changes` 继续提供 RPC 命名、生成源与消费者同步知识；多仓库 PR 技能提供相应操作支持。[ATHENA 浏览器验收技能](../../.codex/skills/athena-browser-acceptance/SKILL.md)按请求在当前可用的内置浏览器检查、真实本地开发环境冒烟和隔离 Playwright 回归之间选择入口，并如实区分三者的证据。真实验收按 [AGENTS.md 的环境准备规则](../../AGENTS.md#本地验收环境准备与完成标准)复用或主动启动目标环境；smoke 工具不启停服务不免除代理的准备责任，验收后默认保留服务运行。相关测试与验证遵循 Superpowers。

本次切换移除了自制后端阶段路由技能、独立的需求/设计/另行实现门禁、额外的前端布局审批门槛，以及默认禁止测试的规定。开发期允许破坏性重构、中文计划、完成邮件等项目约定继续由 `AGENTS.md` 管理。

## 工具与技能配合

[开发工具链的选择表](toolchain-guide.md#按任务选择工具)是工具场景与命令的统一入口，
由根目录 `AGENTS.md` 引导 AI 按需查阅。技能负责判断任务需要的证据，CLI 和报告提供
实际结果；已安装不表示每个任务都要调用，也不保证新 worktree 已准备相同依赖。

- `systematic-debugging` 排查启动脚本、RPC 或 SQL 问题时，分别使用 ShellCheck、grpcurl、psql。
- `athena-browser-acceptance` 按请求选择浏览器检查、隔离回归、真实 smoke 或独立 `make ui-a11y`；Impeccable 结合相关 UI/UX 目标判断哪些检查有用。无障碍预检使用 `UI_ACCEPTANCE_CHECK_ONLY=1 make ui-a11y`，不代表页面扫描通过。
- Go 依赖安全检查使用 `make vuln-check`；`verification-before-completion` 应区分工具就绪、扫描成功执行和扫描没有发现问题。

选择性使用针对任务关联性和检查范围，不免除已经要求的验证。复用现有项目技能，
保持上游 Superpowers / Impeccable 原文；版本与命令细节留在工具链文档和安装器中。

## 升级

升级时先选定上游提交，完整更新来源记录列出的技能目录及其资源，保留原版内容、执行权限和许可证。核对 Codex 技能发现结果、上游文件一致性、脚本可执行性及项目说明中的链接，再更新来源提交。若新增或移除上游技能，同步目录清单和本文入口；不要只更新单个 `SKILL.md` 而遗漏配套文件。
