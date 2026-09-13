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
  - [Plan Implementation Completion Email](#plan-implementation-completion-email)
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

### 本地验收环境准备与完成标准

- 已授权的本地验收包含必要的环境检查、准备、启动、排查和重验。服务未启动是待处理的前置条件，不能仅凭首次连接拒绝、预检失败或其他测试通过就结束必要的真实环境验收。
- 先确认目标地址、运行进程及其所属仓库/worktree。正确且健康的已有环境直接复用；未启动时按[本地运行说明](docs/developer-guide/running-locally.md#prepare-the-development-environment-for-acceptance)选择项目 Node 版本，从目标仓库执行 `make run`，保存持久运行会话及日志。这些必要操作无须再次请求用户确认。
- smoke 命令本身不启停开发服务；执行验收的代理负责准备环境。不得把工具的职责边界解释成代理不能执行 `make run`。启动后检查进程、前端入口和会员/管理员 bootstrap，再执行真实 smoke 并检查退出结果和报告；端口监听或 HTTP 200 不等于验收通过。
- 失败时保留证据，检查相关日志，在已授权范围内解决环境问题并重验。只有确实无法自行解决的外部依赖、凭据、权限或需要用户决定的问题，才报告阻塞，写明尝试、原因和未完成项；验收任务本身不授权扩大为产品行为修改。
- 必要的真实验收未通过时，区分“实现完成”和“验收未完成”，不得宣称整个任务完成，也不得发送整个任务的完成通知。隔离测试、受控场景和预检不能替代真实环境验收。
- 本次启动的开发服务在验收结束后默认保留运行，交付时说明地址、仓库/worktree、会话或进程、日志、验收结果及从该仓库执行 `make stop` 的停止方式。已有服务保持原样；不得自动执行 `make run-reset`、删除数据卷或终止归属不明的进程。
- 用户明确要求只读、仅预检或不启动服务时遵循该限制，仅报告实际检查结果。规则文档修改、纯隔离测试不因此自动扩大为启动真实环境。

### Chinese Plans

When outlining or listing a plan (implementation steps, task breakdown, approach summary, etc.), write the plan in Chinese (简体中文).

### Plan Implementation Completion Email

After a user-confirmed plan has been fully implemented, including all required code, configuration, documentation, and verification, send exactly one completion email from the repository root before returning the final response:

```bash
make notify-task-complete \
  TASK_NOTIFICATION_SUBJECT='任务完成：<简短任务名称>' \
  TASK_NOTIFICATION_BODY='已完成：<核心成果>；验证：<验证结果>。'
```

- Do not send this notification when only the plan has been written, or when implementation is incomplete, blocked, failed, or cancelled.
- Write a short subject and body in Simplified Chinese. Do not include passwords, tokens, API keys, or any other secrets.
- Use the default `.env` notification configuration; this rule does not switch to `.env.prod`.
- Wait for the Make command to finish before returning the final response.
- If the notification still fails after the command's built-in retries, keep the implementation task complete but report the notification failure and a credential-safe error summary in the final response. Do not claim that the email was sent.

### Impeccable Integration

- Use Superpowers to guide the development process and Impeccable for UI/UX expertise under `ui/`. Read the relevant requirements, `docs/design/web-ui/` documents, and current React/CSS implementation as context.
- Reuse the design approval obtained through Superpowers. Layout diagrams and visual previews may support that design discussion without a separate repository-specific layout gate.
- Derive any Impeccable context files from the repository's requirements, designs, and approved task spec; keep those references aligned with the project facts.
- Treat automatic Impeccable hook findings as review input. Do not weaken confirmed behavior, accessibility requirements, established brand decisions, or repository-specific UI conventions merely to clear a generic detector rule.

### 本地图片路径规则

- 读取用户提供的本地图片前，必须先将路径转换为 WSL2 可用的格式。
- 禁止将原始 Windows 路径（如 `C:\Users\name\AppData\Local\Temp\image.png`）直接传给 Linux 图片工具。
- 应转换为对应的 WSL 路径，例如 `/mnt/c/Users/name/AppData/Local/Temp/image.png`；必要时使用 `wslpath -u`。
- 读取前必须确认转换后的路径真实存在。
- Windows 8.3 短路径（如 `FUNDCO~1`）必要时应解析为完整用户目录名称。
- 对 Windows 临时目录中的图片，需要区分：
  - 路径格式不正确；
  - 临时文件已经被删除。
- 转换后的文件存在时，使用其绝对 WSL 路径读取。
- 文件不存在时，应告知用户临时文件可能已经过期，并请用户重新上传或附加图片。
