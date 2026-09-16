# Source: AGENTS.md

# Global System Prompt

## Table of Contents

- [Rules](#rules)
  - [服务独立性与开发边界](#服务独立性与开发边界)
  - [Project Status](#project-status)
  - [Requirement-Driven Architecture](#requirement-driven-architecture)
  - [Test Data and API Keys](#test-data-and-api-keys)
  - [No Historical Compatibility](#no-historical-compatibility)
  - [Superpowers Development Workflow](#superpowers-development-workflow)
  - [本地验收环境准备与完成标准](#本地验收环境准备与完成标准)
  - [Chinese Plans](#chinese-plans)
  - [Task Result Email](#task-result-email)
  - [Impeccable Integration](#impeccable-integration)
  - [本地图片路径规则](#本地图片路径规则)

## Rules

### 服务独立性与开发边界

新服务或服务边界改造必须先阅读并遵守[服务开发规范](docs/developer-guide/service-development-standards.md)。该规范以 `SDS-R1` 至 `SDS-R8` 定义服务边界、API、独立构建运行、授权、事务、本地编排和验证证据；设计与 PR 必须按变更适用范围引用相应规则和证据。

既有实现的差距不得新增耦合；无关的小修复不因此强制全系统重构。纯解析器或内部工具库无需为了该规范拆成服务。

### Project Status

The project is currently in development and has not been launched yet. Necessary breaking changes are allowed, and backward compatibility with historical behavior is not required.

### Requirement-Driven Architecture

The current service architecture is designed to satisfy the current requirements and must not be treated as a permanent constraint. Future requirements may expose limitations in the existing architecture or make those requirements inconvenient to implement. When that happens:

- Prioritize satisfying the requirements over preserving the existing architecture.
- Freely perform breaking refactors or replace existing component boundaries, data flows, and technology choices when needed to provide a clear and effective implementation.
- Introduce additional infrastructure or technology stacks, including Kafka, Redis, RabbitMQ, or other appropriate systems, whenever the requirements justify them.
- Do not treat the current architecture or technology stack as immutable.

### Test Data and API Keys

This repository is in development. All data in this repository is test data, and any API keys are keys used for testing only. Do not redact, mask, obfuscate, or otherwise desensitize these values.

### No Historical Compatibility

Do not preserve historical compatibility. When implementing changes:

- Do not add migration paths, feature flags, dual code paths, or shims solely to keep old behavior working.
- Do not retain deprecated APIs, fields, or data formats unless they are still actively used in the current design.
- Prefer direct replacement and cleanup over gradual deprecation.
- Breaking changes are acceptable when they simplify the codebase or align with the current design.

### Superpowers Development Workflow

- Use the upstream Superpowers skills in `.agents/skills/` as the development workflow for this repository. At the start of a task, read and follow [using-superpowers](.agents/skills/using-superpowers/SKILL.md), including its Codex platform reference, then use the applicable skills.
- Follow Superpowers for brainstorming, design approval, planning, implementation, debugging, code review, and completion. Apply the user's current request and existing authorization; do not ask again for an unchanged decision already made in the task.
- Follow [test-driven-development](.agents/skills/test-driven-development/SKILL.md) for behavior changes and [verification-before-completion](.agents/skills/verification-before-completion/SKILL.md) before claiming success. Relevant tests and verification are part of implementation; they do not require a separate request. Honor explicit task-specific exceptions from the user.
- Read the relevant [requirements](docs/requirements/README.md), [designs](docs/design/README.md), and actual source as project context. Preserve business decisions, unresolved questions, and scope; keep affected long-term documents consistent with the resulting implementation. Document statuses describe facts, not additional workflow gates.
- Use Superpowers' default `docs/superpowers/specs/` and `docs/superpowers/plans/` locations when its selected workflow calls for written artifacts. Project-local worktrees belong in `.worktrees/`; temporary Superpowers execution state belongs in `.superpowers/`.
- Project-specific skills supply domain knowledge and repository operations alongside Superpowers. Installation details and the skill map are in [Superpowers Development](docs/developer-guide/superpowers-development.md).
- 选择开发、调试或验证工具时，查阅[按任务选择工具](docs/developer-guide/toolchain-guide.md#按任务选择工具)，根据当前问题和影响范围选用，并在当前工作区确认所选工具可用。工具清单不是每个任务的必跑清单；用户或任务方案已经要求的验证仍须完成。

### 本地验收环境准备与完成标准

- 已授权的本地验收包含必要的环境检查、准备、启动、排查和重验。服务未启动是待处理的前置条件，不能仅凭首次连接拒绝、预检失败或其他测试通过就结束必要的真实环境验收。
- 先确认目标地址、运行进程及其所属仓库/worktree。正确且健康的已有环境直接复用；未启动时按[本地运行说明](docs/developer-guide/running-locally.md#prepare-the-development-environment-for-acceptance)选择项目 Node 版本，从目标仓库执行 `make run`，保存持久运行会话及日志。这些必要操作无须再次请求用户确认。
- smoke 命令本身不启停开发服务；执行验收的代理负责准备环境。不得把工具的职责边界解释成代理不能执行 `make run`。启动后检查进程、前端入口和会员/管理员 bootstrap，再执行真实 smoke 并检查退出结果和报告；端口监听或 HTTP 200 不等于验收通过。
- 失败时保留证据，检查相关日志，在已授权范围内解决环境问题并重验。只有确实无法自行解决的外部依赖、凭据、权限或需要用户决定的问题，才报告阻塞，写明尝试、原因和未完成项；验收任务本身不授权扩大为产品行为修改。
- 必要的真实验收未通过时，区分“实现完成”和“验收未完成”，不得宣称整个任务完成；达到下文任务结果邮件的时长条件时，仍须发送固定的通用通知，未完成项及阻塞或失败原因只在当前工作会话中说明，不写入邮件。隔离测试、受控场景和预检不能替代真实环境验收。
- 开发、调试或验收任务结束时，默认停止本任务启动的临时服务、预览、测试替身及所属容器，保留数据库、数据卷、日志、截图和验收报告。完成、取消、暂停，以及保存证据后以失败或阻塞结束任务，均须收尾；任务仍在持续调试或验收时可保持运行，不以单次回复结束作为停服时点。
- 用户在任务前启动的服务、其他任务正在使用的环境和借用的共享基础设施保持原样。长期主开发环境须由用户明确指定；只有用户明确要求继续查看或保留现场调试时，才保留本任务中对应的环境，不能据此保留所有分支和测试实例。
- 收尾由执行任务的代理主动完成：先核对仓库/worktree、实例和资源归属，再从同一仓库使用带相同 `INSTANCE` / profile 的 `make stop`、`make stop-instance` 或对应工具的停止入口；独立测试替身和预览也须按记录停止。不得自动执行 `make run-reset`、删除数据卷、停止借用的基础设施或终止归属不明的进程。
- 停止后核对所属进程退出、端口释放和容器停止；收尾失败须保留日志、排查并报告未停止项，不得仅凭停止命令已执行就宣称完成。交付时列明已停止和仍保留的环境；保留项注明用户要求或已有环境归属，并提供地址、仓库/worktree、实例、会话或进程、日志、验收结果及准确停止命令。
- 用户明确要求只读、仅预检或不启动服务时遵循该限制，仅报告实际检查结果。规则文档修改、纯隔离测试不因此自动扩大为启动真实环境。

### Chinese Plans

When outlining or listing a plan (implementation steps, task breakdown, approach summary, etc.), write the plan in Chinese (简体中文).

### Task Result Email

任何类型的任务，只要累计执行时间超过十分钟（严格大于 600 秒），都必须在任务结束时、最终回复前，从仓库根目录调用一次以下命令发送结果邮件。此规则适用于计划、实施、调试、审阅、研究及文档等任务，不要求进入 Plan 模式或事先确认实施计划。

- 从开始处理任务起记录执行时间，包括分析、工具运行、测试、验证和环境收尾；等待用户回复或任务暂停期间不计入。同一任务跨轮次继续执行时累计计时，并行子任务的重叠时间只计一次。
- 累计执行时间不超过十分钟时不发送。超过十分钟后继续执行，等任务结束再通知；不在到达时限时发送进度邮件，也不定期重复通知。
- 任务完成、失败、取消，或因阻塞结束本次执行时，达到时长条件都须发送相同的通用通知。邮件仅告知本次开发工作已结束，不表示任务成功或全部实施完成；实际状态、成果、验证结果、未完成项及原因只在当前工作会话中说明。
- 同一任务由主代理统一发送一次，子代理不单独发送；保留通知结果，避免跨轮次重复调用。

```bash
make notify-task-complete \
  TASK_NOTIFICATION_SUBJECT='开发工作通知' \
  TASK_NOTIFICATION_BODY='本次开发工作已结束，请返回工作会话查看。'
```

- 标题和正文必须逐字使用上述固定文案，不添加任务名称、项目或产品名称、业务领域、功能描述、技术栈、实现细节、文件路径、分支或提交信息、链接、日志、验证结果、错误原因、执行耗时或任何凭据；不得通过概括、缩写或代号间接透露正在开发什么。历史计划或示例中的详细邮件模板不得沿用。
- 发件人显示名称使用通用的 `Development Notification`，不包含项目名称；不得添加项目签名、附件或其他任务内容。此限制适用于对外邮件，不妨碍在当前工作会话中如实汇报细节。
- 使用默认 `.env` 通知配置，不切换到 `.env.prod`。
- 等待 Make 命令结束后再给出最终回复。
- 若命令内置重试耗尽后仍失败，保持任务实际结果，在最终回复中说明通知失败并提供不含凭据的错误摘要，不得声称邮件已发送。
