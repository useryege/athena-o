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



# Source: .codex/skills/athena-browser-acceptance/SKILL.md

---
name: athena-browser-acceptance
description: Use when inspecting ATHENA pages, checking UI accessibility, smoke-testing a local development UI, or running repeatable Playwright UI acceptance and browser regression.
---

# ATHENA Browser Acceptance

Choose the lightest route that produces the evidence the request needs. Read the repository `AGENTS.md`, the affected requirements and design, and [Running Athena Locally](../../../docs/developer-guide/running-locally.md) before acting.

## Choose the route

| Request | Route |
| --- | --- |
| Explore or inspect a page during development | Use the current session's built-in browser when it is available. Inspect the real page, console, and decisive requests. |
| Repeatable regression or isolated frontend/backend acceptance | Run `make ui-acceptance`. This creates fresh root-path and `/athena` harnesses and uses Playwright's paired Chromium. |
| Smoke-test the real local development UI | Prepare or reuse the target development environment below, then run `make ui-acceptance UI_ACCEPTANCE_MODE=smoke` using system Chrome. The command itself never starts or stops the stack. |
| Check UI accessibility, including WCAG contrast or accessible names | Run `make ui-a11y`. This runs the existing axe/Playwright scenarios in an isolated harness; an existing development stack is not a prerequisite. See [coverage and evidence](../../../docs/developer-guide/toolchain-guide.md#独立无障碍检查). |
| Check prerequisites only | Add `UI_ACCEPTANCE_CHECK_ONLY=1` to the selected route; for accessibility, use `UI_ACCEPTANCE_CHECK_ONLY=1 make ui-a11y`. Report prerequisite results only; when full acceptance is required, investigate and resolve prerequisite failures within the authorized scope, then rerun. |

Choose routes for the requested evidence, not as a checklist to run on every task. A documentation-only correction does not itself require browser scans. Availability in a previous checkout does not establish readiness in the current worktree; use the selected route's prerequisite check when needed.

Use `UI_ACCEPTANCE_BASE_URL` only to select an explicit HTTP loopback smoke target. The command discovers WSL Node in the documented order and uses repository-local Yarn and Playwright. It does not install or download dependencies.

## Prepare the real development environment

Follow the local acceptance rule in `AGENTS.md` and the [environment preparation procedure](../../../docs/developer-guide/running-locally.md#prepare-the-development-environment-for-acceptance).

1. Confirm the requested target and its repository/worktree using the running process and runtime state. Reuse a healthy matching stack. A listening port in another worktree is not evidence for this checkout; do not stop an unrelated stack or silently substitute its URL.
2. If the target stack is absent, select the project's Node version and run `make run` from the target repository in a persistent session with captured logs. Necessary local environment preparation is part of authorized acceptance; do not ask again merely because startup is needed. Honor explicit read-only, check-only or no-start instructions instead.
3. Check startup progress, process health, both frontend entries and member/admin bootstrap. Inspect logs when readiness fails; resolve environment issues within scope and retry. Do not repeatedly rerun unchanged failures. Neither an open port nor HTTP 200 proves application readiness; run the actual smoke and inspect its exit result and artifacts.
4. Keep failure evidence. A first connection refusal, failed preflight, time pressure or passing isolated tests does not justify abandoning required smoke. Report a blocker only with concrete attempts and an unresolved cause that cannot be handled autonomously. Do not expand acceptance into unapproved product changes or data resets.
5. When the development, debugging or acceptance task ends, stop the temporary services, previews, test substitutes and owned containers started for this task. This includes completion, cancellation, pause, and ending with a failure or blocker after saving evidence. Keep services available while the task is actively debugging or testing; a single reply is not a task shutdown trigger. From the owning repository, use the matching instance/profile stop command, then stop separately launched helpers through their own shutdown entrypoints. Preserve development databases, volumes and evidence; do not use reset or stop borrowed infrastructure.
6. Preserve services already started by the user and environments used by other active tasks. Retain a task-created environment only when the user explicitly requests it for inspection, debugging or use as the main development environment; that request covers only the specified environment. Verify owned process exit, released ports and stopped containers. Investigate and report cleanup failures. Report stopped and retained environments, and for each retained environment include its reason, target, repository/worktree, instance, session/process, logs, acceptance result and exact stop command. Follow the [task shutdown procedure](../../../docs/developer-guide/running-locally.md#task-shutdown-and-retained-environments); never kill processes of unclear ownership.

The smoke **tool** does not start services; the accepting **agent** must prepare them. Required real smoke left unverified means acceptance remains incomplete: distinguish implementation from verification and do not declare the whole task complete or send its completion email. Check-only and isolated-only requests do not require starting a development stack.

## ATHENA evidence contract

- Keep member and administrator sessions isolated. For interactive Google or Phantom authentication, use the real browser flow. Harness `storageState` proves only the isolated test identity and is not authentication acceptance.
- Classify evidence by mode: `ui-fixtures` proves page behavior against intercepted responses; `live` proves the ATHENA components and temporary PostgreSQL exercised by the isolated harness, while chain, profile, and Telegram boundaries remain local substitutes; `smoke` proves the current development application's bootstrap and shell only.
- The `a11y` suite checks automated accessibility rules for its existing scenarios. Preserve raw axe results and failing exits; it complements keyboard and manual checks and does not prove that every page is accessible.
- The built-in browser is for interactive inspection. A successful built-in-browser check is not a system-Chrome smoke result or a Playwright regression result.
- Preserve Playwright's native report, trace, screenshot, and attachments under `.tmp/athena-ui-acceptance/<run-id>/`. Report only checks actually executed, including the mode, target, result, blocker, and cleanup status.
- Do not broaden an acceptance-only request into a product fix. When a fix is authorized, preserve the failing evidence, use `systematic-debugging` and `test-driven-development`, then rerun the affected scenario and the smallest relevant regression.

The isolated command cleans only its run-specific harness, process groups, and database container, and preserves its evidence artifacts. The smoke command never owns the user's `make run` environment.


# Source: .codex/skills/publish-multi-repo-prs/SKILL.md

---
name: publish-multi-repo-prs
description: Publish coordinated GitHub pull requests for a root repository and its Git submodules by inspecting local and remote state, pushing confirmed commits, skipping empty or duplicate PRs, creating structured draft PRs, checking conflicts and status checks, and marking ready or merging only when authorized. Use when the user asks to create, submit, or resubmit PRs; publish multi-repository or submodule changes; check PR conflicts; or merge conflict-free PRs.
---

# Publish Multi-Repo PRs

Publish related branches from Git repositories and submodules without including unrelated local work or creating redundant PRs.

## Guardrails

- Read and obey the applicable `AGENTS.md` in every repository.
- Treat each submodule as an independent repository with its own remote, branch, PR, and merge result.
- Do not stage, commit, amend, rebase, force-push, clean, stash, or delete worktree changes during PR publishing. If the user also requests a local commit, complete that authorized task separately before continuing this publishing workflow.
- Exclude untracked, unstaged, and staged worktree content from the PR unless it already belongs to a pushed commit. Report excluded local changes.
- Never expose tokens, credentials, `.env` values, or authentication output.
- Follow Superpowers completion verification for the exact commits being published. Use relevant tests, builds, or checks as evidence; preserve unrelated worktree content and avoid modifying source as part of publishing.
- Do not create an empty PR. Do not duplicate an open PR with the same repository, head, and base.
- Do not force-push, enable auto-merge, delete branches, or retarget PRs unless explicitly requested.
- Create draft PRs by default. Mark ready or merge only when the current user request explicitly authorizes that action.

## Tool Selection

1. Prefer the connected GitHub plugin for remote comparison, PR lookup, creation, status inspection, readiness, and merging.
2. Fall back to an authenticated GitHub CLI only when the plugin is unavailable or unauthorized. Verify `gh auth status` before using it.
3. Use local `git` for worktree inspection, branch discovery, submodule traversal, and normal pushes.
4. Stop when neither GitHub channel can access the target private repository.

## Workflow

### 1. Discover repositories

1. Start with the root repository and list initialized submodules recursively.
2. Order repositories from the deepest submodule to the root so child PRs are created and merged before parent gitlink PRs.
3. Use each repository's current branch as `head` unless the user specifies another branch.
4. Use the user-specified `base`; otherwise default to `feature/local-development-run`.
5. Record the repository full name from `origin`, current HEAD SHA, upstream SHA, and worktree status.
6. Stop for detached HEAD, a missing remote, or an ambiguous repository identity.

### 2. Confirm remote state

For every repository:

1. Confirm the head and base branches exist remotely.
2. Compare local HEAD with the remote head.
3. When local commits are ahead and the user asked to publish PRs, push the current branch normally only after confirming the commit scope is intentional. Never include worktree-only changes and never force-push.
4. When local and remote have diverged, stop that repository and report the mismatch.
5. Re-read the remote head SHA after any push and use it as the expected PR head.

### 3. Decide whether a PR is needed

Compare remote `base...head` through GitHub:

- If `ahead_by = 0`, skip the repository and report that the base already contains the head changes.
- If `ahead_by > 0`, continue even when `behind_by > 0`. A previous PR merge commit commonly makes the head appear behind; behind status alone is not a conflict.
- Treat GitHub's computed `mergeable` result, not `behind_by`, as the conflict decision.
- Search for an open PR with the exact repository, head, and base. Reuse it instead of creating a duplicate.

### 4. Draft the PR

Derive the title from only the commits newly ahead of base. Use a concise English Conventional Commit-style title that summarizes the whole new diff.

Write the body in Simplified Chinese with exactly these sections:

```markdown
## 变更内容

- <high-signal changes>

## 变更原因

<why the changes are needed>

## 影响

<developer or user impact>

## 验证

- <validation that actually ran>
- <实际测试命令与结果；未运行时说明未验证范围>
```

- Mention submodule pointer changes and child PRs when relevant.
- Mention excluded untracked or local-only artifacts when they could be mistaken as part of the PR.
- Never claim a validation command ran unless its result is known.
- Report tests only when their execution and results are known for the reviewed changes. Otherwise state that tests were not run; do not infer success from an earlier task or a template.

Create the PR as a draft with maintainer edits enabled. Preserve the exact expected head SHA.

### 5. Check conflicts and checks

1. Re-read the PR after creation. GitHub may initially return `mergeable: false` while calculating; do not immediately classify that transient result as a conflict.
2. Re-read a bounded number of times until GitHub returns a stable mergeability result. Do not wait indefinitely.
3. Stop the repository when GitHub confirms a merge conflict.
4. Inspect combined commit statuses and GitHub Actions/check runs when available:
   - No configured checks: allow the workflow to continue.
   - Pending checks: leave the PR open and report the pending state.
   - Failed or cancelled checks: leave the PR open and report the failure.
   - Successful checks: allow the workflow to continue.

### 6. Mark ready, approve, or merge

Interpret the user's authorization literally:

- "创建 PR" or "提交 PR": create or reuse a draft PR, then stop after reporting its state.
- "检查是否冲突": inspect mergeability and checks without merging.
- "自己审核通过": mark the draft Ready for review. Do not submit `APPROVE` from the PR author's account because GitHub rejects self-approval.
- "合并" or "没冲突自己合并通过": after confirming stable mergeability and no blocking checks, mark Ready and merge.

When merging:

1. Use a merge commit unless the user explicitly requests squash or rebase.
2. Pass the expected head SHA so GitHub rejects a merge if the branch changed during inspection.
3. Merge deepest submodules first and the root repository last.
4. Stop dependent parent merges if a child merge fails.
5. Do not treat repository permission to merge as permission to bypass a confirmed conflict or failed check.

## Final Report

Group repositories by outcome:

- Created or reused: repository, PR number, URL, head, base, and draft/ready state.
- Skipped: repository and why no PR was needed.
- Blocked: repository, conflict/check/auth reason, and the required next action.
- Merged: repository, PR URL, and merge commit SHA.

State explicitly whether any local worktree files were excluded and whether tests or other validation ran.
