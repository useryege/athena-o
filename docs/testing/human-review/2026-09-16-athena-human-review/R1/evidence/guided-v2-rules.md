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
  - [AI 交付后的人工审查](#ai-交付后的人工审查)
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

### AI 交付后的人工审查

- ATHENA 开发任务完成实现、AI 审阅、约定验证及第六步环境收尾后，使用项目技能 [athena-human-review](.codex/skills/athena-human-review/SKILL.md)，在 `docs/testing/human-review/<任务标识>/R<轮次>/` 交付具体的 `ai-delivery.md`、`review-guide.md` 和可回填的 `human-report.md`。纯咨询、独立只读审查、独立 PR 操作、其他项目及未明确接续的历史已完成任务不自动建立新轮次。
- “本轮 AI 交付完成，待人工审查”只表示该轮 AI 工作、材料和收尾已有实际证据；测试通过、PR 合并、环境停止或通知发送均不表示“最终交付完成”。本轮 AI 阶段存在缺口时如实记录“AI 阶段未完成／存在阻塞”。
- 人工报告在草稿期间保持被审查版本稳定。用户提交完整报告后，即授权 AI 核实并修复已确认设计范围内的问题，再完成审阅、验证、下一轮交付与收尾；报告可如实包含受阻和未执行项。用户明确的仅分析或先审方案限制继续优先；新需求、设计改变和超出既有授权的外部操作仍需用户决定。
- 修复轮保留原始观察和稳定问题编号；AI 验证后标记“待人工复验”，不能代替用户关闭。下一轮复验修复项、受影响流程和历史受阻项；未受影响的历史结果只有在写明来源版本与影响依据时才能沿用。
- 只有用户对准确轮次和版本明确确认通过，必查项、问题处置、AI 证据及环境状态均已交代，才记录“最终交付完成”。人工审查阶段位于第六步 AI 交付与环境收尾之后，该轮可按任务既有方式保留工作区或分支、提交 PR 或已经合并；既有 Git 操作授权只按其原始对象、版本和限制继续适用，不自动扩大到后续发布或合并。

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

The smoke **tool** does not start services; the accepting **agent** must prepare them. Required real smoke left unverified means the AI delivery stage remains incomplete: distinguish implementation, verification, post-delivery human review, and final user confirmation. Task-result notification behavior follows the root `AGENTS.md`; sending it does not prove acceptance or final delivery. Check-only and isolated-only requests do not require starting a development stack.

## ATHENA evidence contract

- Keep member and administrator sessions isolated. For interactive Google or Phantom authentication, use the real browser flow. Harness `storageState` proves only the isolated test identity and is not authentication acceptance.
- Classify evidence by mode: `ui-fixtures` proves page behavior against intercepted responses; `live` proves the ATHENA components and temporary PostgreSQL exercised by the isolated harness, while chain, profile, and Telegram boundaries remain local substitutes; `smoke` proves the current development application's bootstrap and shell only.
- The `a11y` suite checks automated accessibility rules for its existing scenarios. Preserve raw axe results and failing exits; it complements keyboard and manual checks and does not prove that every page is accessible.
- The built-in browser is for interactive inspection. A successful built-in-browser check is not a system-Chrome smoke result or a Playwright regression result.
- Preserve Playwright's native report, trace, screenshot, and attachments under `.tmp/athena-ui-acceptance/<run-id>/`. Report only checks actually executed, including the mode, target, result, blocker, and cleanup status.
- Do not broaden an acceptance-only request into a product fix. When a fix is authorized, preserve the failing evidence, use `systematic-debugging` and `test-driven-development`, then rerun the affected scenario and the smallest relevant regression.

The isolated command cleans only its run-specific harness, process groups, and database container, and preserves its evidence artifacts. The smoke command never owns the user's `make run` environment.

When browser work enters post-delivery human review, use `athena-human-review` after the AI acceptance and task shutdown steps. Its concrete guide records the stopped environment and the exact recovery and stop instructions for the reviewed version. Restoring that environment for the user is a new review-environment action with explicit resource ownership; it does not change an earlier smoke result or make the final human conclusion automatic.


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

## Post-delivery human review boundary

For a development task already using the ATHENA post-delivery review loop, report Git delivery and human review as separate facts. Step six may hand off a worktree or branch, publish a PR, or contain an already merged version according to the task's actual delivery mode. A merged PR can still be awaiting human review. Fixes from a submitted report reuse publish or merge authorization only when its original repository, version, action, and limits cover the follow-up; otherwise keep the new version local and report the missing authority. Human review follows the AI delivery and cleanup step, and neither AI checks nor a merge is final human approval.

A standalone request to create, inspect, ready, or merge PRs keeps this skill's original scope and does not create a human-review round unless the user explicitly connects it to an active ATHENA development delivery.


# Source: .codex/skills/athena-human-review/SKILL.md

---
name: athena-human-review
description: Use when ATHENA development reaches post-delivery human review, when a user submits an ATHENA human review report, when fixes need another review round, or when the user confirms the reviewed version as final.
---

# ATHENA 交付后人工审查

把 AI 阶段完成与用户最终确认分开记录。首次交付必须同时提供可定位的交付报告、可执行指南和可回填报告，不能只给检查建议。

## 触发边界

- ATHENA 开发任务完成实现、AI 审阅、约定验证和第六步环境收尾后，生成 R1 材料。
- 用户明确提交完整人工报告后，处理报告；报告允许包含受阻和未执行项。
- 修复完成并再次收尾后，生成 R2、R3 等材料。
- 用户对准确轮次和版本明确确认通过后，记录最终交付。

纯咨询、独立只读审查、独立 PR 操作、其他项目，以及未明确接续的历史已完成任务不自动进入此闭环。

## 每轮材料契约

从三个实际模板生成文件，保存到 `docs/testing/human-review/<任务标识>/R<轮次>/`：

1. [AI 交付模板](assets/ai-delivery-template.md)生成 `ai-delivery.md`：交付范围、设计依据、准确版本、AI 证据、已知限制、Git 状态、人工状态和环境收尾。
2. [人工审查指南模板](assets/review-guide-template.md)生成 `review-guide.md`：版本核对、适用时的环境恢复、身份和测试数据、逐项操作、可观察预期、证据及恢复方法。
3. [人工报告模板](assets/human-report-template.md)生成 `human-report.md`：AI 预填元信息和真实检查项，结果保持“未执行”，由用户填写观察、问题和结论。

模板中的填写标记只用于作者识别。实际交付必须替换为本轮确定值；不适用项写明原因。AI 不代填用户“通过”。若成果未提交，记录工作区差异和可恢复补丁或等价产物，不能用旧提交的证据代表当前状态。

正式交付前必须实际写入并读回三份文件，核对路径存在和内容完整，再在回复中给出可打开的文件链接。只声称“已生成”、给出尚未落地的路径或复述模板要求不算交付。逐项检查以下契约：

- 三份材料使用同一任务、轮次、设计依据和完整不可缩写的提交 SHA／产物版本；不能用 `111...111`、“最新版本”等代替。
- 指南在任何启动步骤之前写出完整预期版本及核对方法；版本不符即记录受阻。
- 指南中的每个真实 `CHK-*` 都在 `human-report.md` 有且只有一条对应结果行，检查内容已经具体化，初始结果为“未执行”。
- 报告保留可填写的实际观察、证据、问题编号和总体结论；不能用一段泛化回填说明替代逐项结果行。

若当前通道确实只能交付文本或禁止写文件，在回复中直接给出可复制的最小 `human-report.md`：填入真实任务、轮次、完整版本和设计依据，至少预填一条来自本轮指南的实际 `CHK` 行，结果为“未执行”，并保留观察、证据、问题编号、报告状态和总体结论字段。简短回复也必须包含这些材料，不能以篇幅为由省略。

## 四个入口

### 首次交付

R1 覆盖本次交付的完整人工范围。只有 AI 审阅、约定验证、材料和收尾均有证据时，状态才是“本轮 AI 交付完成，待人工审查”；否则写“AI 阶段未完成／存在阻塞”及缺口。文档或技能任务写明无需运行栈，不为人工阶段启动服务。通知只引用根 `AGENTS.md` 的累计时长、固定内容和同任务去重规则。

### 已提交报告

“已提交”可由文件状态或同等自然语言表达。草稿期间保持被审查版本稳定；可协助恢复环境，但不因草稿问题修改产品。若用户要求立即修复，先保存本轮已有观察并明确切换到下一版本。

收到已提交报告后，先用 `receiving-code-review` 核实，再用 `systematic-debugging` 定位；局部修复方案写入交付材料，多任务或跨层修复使用 `writing-plans`，然后对已确认设计范围内的缺陷连续实施。行为修复使用 `test-driven-development`，实施后使用 `requesting-code-review` 和 `verification-before-completion` 完成审阅与验证，并执行仍适用的真实验收。报告提交即包含这部分修复授权，不新增审批。用户明确的仅分析／先审方案限制继续优先；新需求、设计决定及外部操作仍按原授权边界处理。

保留原始观察，问题编号跨轮次不重置。方案拟定或修复进行中不得提前写验证通过；只有实际修复、代码审阅和约定验证的证据均已取得，问题才能标为“AI 已验证，待人工复验”，且不能标为人工关闭。

### 修复轮交付

新轮次列出：原问题复验、受修复影响的关联流程、历史受阻或未执行项，以及可沿用的上轮结果。沿用项注明来源轮次、版本和影响判断，不能伪装成在新版本重新通过。

### 最终人工确认

仅当用户明确确认准确轮次和版本，必查项已通过，问题或例外已有处置，AI 证据与该版本一致，且环境状态已核对，才记录“最终交付完成”。PR 已合并、AI 测试通过、通知已发送都不能替代此确认。人工阶段位于第六步 AI 交付与环境收尾之后；该轮可以按任务既有方式保留工作区或分支、提交 PR 或已经合并。既有 Git 操作授权只按其原始对象、版本和限制继续适用，不能自动扩大到后续发布或合并。


# Source: .codex/skills/athena-human-review/agents/openai.yaml

interface:
  display_name: "ATHENA Human Review"
  short_description: "Manage ATHENA post-delivery human review rounds"
  default_prompt: "Use $athena-human-review to prepare or process the current ATHENA post-delivery human review round."


# Source: .codex/skills/athena-human-review/assets/ai-delivery-template.md

# AI 交付报告：【AI 填写任务名称】

> 本模板中的 `【AI 填写】` 必须在实际交付时替换为确定值；不适用时写明原因。

## 交付身份

- 任务标识：【AI 填写，用于稳定目录名】
- 交付轮次：【AI 填写，R1／R2／R3…】
- AI 阶段状态：【AI 填写：本轮 AI 交付完成，待人工审查／AI 阶段未完成／存在阻塞】
- 人工审查状态：【AI 填写：待人工审查／待人工复验／最终交付完成；只有用户确认后可填最后一项】
- 仓库及 worktree：【AI 填写绝对路径】
- 分支：【AI 填写】
- 提交／产物版本：【AI 填写完整且不可缩写的 SHA、子模块 SHA 或其他不可变版本；不得使用省略号、“最新版本”或可变分支名代替】
- 工作区状态：【AI 填写 clean；或列出未提交差异及可恢复补丁／产物】
- Git 交付状态：【AI 填写未提交／已提交／PR 草稿／已合并等事实，并附可定位信息】

## 范围与设计依据

- 本轮交付范围：【AI 填写】
- 明确不在本轮范围：【AI 填写】
- 已确认需求、设计或会话决定：【AI 填写可定位链接或记录】
- 相对上一轮的变化：【R1 写首次交付；修复轮逐项引用 ISSUE 编号】

## AI 审阅与验证证据

| 检查或命令 | 针对版本 | 结果 | 证据位置 |
| --- | --- | --- | --- |
| 【AI 填写实际执行项】 | 【AI 填写】 | 【AI 填写实际结果】 | 【AI 填写链接、日志或报告】 |

- 未执行或未通过的约定验证：【AI 填写；没有则写“无”】
- 已知限制与阻塞：【AI 填写；没有则写“无”】
- 证据与当前版本差异：【AI 填写；没有则写“无”】

## 问题与修复状态

| 问题编号 | 原始报告 | 核实结论／原因 | 本轮修改 | AI 验证 | 人工状态 |
| --- | --- | --- | --- | --- | --- |
| 【R1 无问题时写“不适用”；修复轮沿用稳定编号】 | 【链接到原始观察】 | 【AI 填写】 | 【AI 填写】 | 【AI 填写】 | 【AI 已验证，待人工复验／用户确认关闭／其他真实状态】 |

## 环境与资源收尾

### 已停止

- 【AI 填写环境、仓库/worktree、实例/profile、停止命令和退出核对；没有则写明本任务未启动运行环境】

### 保留

- 【AI 填写保留原因、归属、地址、仓库/worktree、实例/profile、会话或进程、日志、验收结果和准确停止命令；没有则写“无”】

## 人工审查入口

- [本轮人工审查指南](review-guide.md)
- [本轮预填人工报告](human-report.md)
- 提交方式：完成或受阻后，把 `human-report.md` 标为“已提交”，或在会话中明确提交同等完整内容。
- 下一状态：等待用户对本报告所列准确版本进行人工审查；AI 通过不代表人工通过。

## 三份材料完整性核对

| 材料 | 实际路径 | 已写入并读回 | 内容核对 |
| --- | --- | --- | --- |
| `ai-delivery.md` | 【AI 填写实际路径】 | 【AI 填写是／否】 | 【任务、轮次、完整版本、证据、收尾和状态】 |
| `review-guide.md` | 【AI 填写实际路径】 | 【AI 填写是／否】 | 【启动前版本核对、真实 CHK、操作、预期和收尾】 |
| `human-report.md` | 【AI 填写实际路径】 | 【AI 填写是／否】 | 【每个 CHK 逐行预填、未执行、观察、证据、问题和总体结论】 |

只有三行均核对为“是”才可声称材料已交付；回复中给出三份实际文件的可打开链接。


# Source: .codex/skills/athena-human-review/assets/human-report-template.md

# 人工审查报告： 【AI 预填任务名称】

> AI 预填任务、轮次、准确版本、设计依据和真实检查项。用户填写结果、实际观察、问题和总体结论。AI 不代填用户“通过”。

## 审查身份

- 任务标识：【AI 预填】
- 交付轮次：【AI 预填】
- 仓库／worktree：【AI 预填】
- 分支与提交／产物版本：【AI 预填准确、完整且不可缩写的 SHA 或其他不可变版本】
- 设计依据：【AI 预填链接】
- AI 交付报告：[ai-delivery.md](ai-delivery.md)
- 人工审查指南：[review-guide.md](review-guide.md)
- 实际环境、身份和数据：【AI 预填要求；用户填写实际差异】
- 审查时间：【用户填写】
- 报告状态：草稿
- 总体结论：【用户填写：通过／存在问题／受阻】

## 检查结果

| 检查编号 | 检查内容（AI 预填） | 结果（用户填写） | 实际观察 | 证据 | 问题编号 |
| --- | --- | --- | --- | --- | --- |
| CHK-001 | 【AI 预填与指南一致的具体检查，不得保留泛化占位描述】 | 未执行 |  |  |  |

【AI 为指南中的每个真实检查项预填且只预填一行；检查编号和内容逐项对应，结果初始值均为“未执行”。实际交付不得只写回填说明或用省略号代替结果行。】

## 问题记录

### ISSUE-001：【用户填写简短问题描述】

- 发现轮次：【本轮编号】
- 对应检查：【CHK 编号；指南外发现填“自由检查”】
- 操作步骤：
  1. 【用户填写实际动作】
  2. 【用户填写输入或选择】
- 预期结果：【按指南或设计应看到什么】
- 实际结果：【实际看到了什么】
- 发生情况：【必现／偶发／仅遇到一次／不确定】
- 影响：【是否阻塞后续项目及具体范围】
- 原始证据：【截图、日志、URL、业务记录标识；没有时写“无”】

原始观察提交后保留，不用后续诊断或修复结论覆盖。AI 在交付报告中引用该问题并补充核实、修复与验证状态；下一轮仍使用同一 ISSUE 编号。

## 未执行或受阻项目

| 检查编号 | 状态 | 原因 | 阻塞问题编号 | 可继续的独立检查 |
| --- | --- | --- | --- | --- |
| 【用户填写】 | 【未执行／受阻】 | 【用户填写】 | 【没有则写“无”】 | 【用户填写】 |

## 本轮结论与交接

- 报告提交声明：【用户填写“本轮报告已经提交，请根据上述内容处理”，或明确确认准确版本最终通过】
- 环境状态：【用户填写已停止／按要求保留／需要 AI 协助收尾】
- 保留实例详情：【用途、地址、仓库/worktree、实例、会话／进程、日志和停止命令；没有则写“无”】
- 我明确接受的例外及范围：【没有则写“无”】

提交时把“报告状态”从“草稿”改为“已提交”。包含受阻和未执行项的完整报告也可以提交；如实记录即可。


# Source: .codex/skills/athena-human-review/assets/review-guide-template.md

# 人工审查指南： 【AI 填写任务名称】

> AI 在实际交付时把本模板具体化。命令、地址、身份、数据和预期不得保留通用占位符。

## 审查对象

- 轮次：【AI 填写】
- 仓库／worktree：【AI 填写】
- 分支与准确完整版本：【AI 填写不可缩写的 SHA、子模块 SHA 或其他不可变版本】
- 设计依据：【AI 填写链接】
- 配套报告：[human-report.md](human-report.md)

## 版本核对

在启动任何服务或执行会修改数据的步骤前完成本节。

1. 【AI 填写进入正确 worktree 的命令或操作。】
2. 【AI 填写核对提交、子模块或未提交产物身份的准确命令。】
3. 预期：【AI 重复写出应看到的完整、不可缩写 SHA，适用时包括每个子模块 SHA；未提交成果写出差异摘要和可恢复产物版本。】

版本不符时停止会改变数据的检查，在报告中记为“受阻”，保存实际版本信息；不要强制切换或重置覆盖现有工作。

## 环境恢复

- 是否需要运行环境：【AI 填写需要／不需要及理由】
- 前提与依赖：【AI 填写】
- 启动命令、实例或 profile：【AI 从目标 worktree 和本地运行说明填入已核实命令】
- 日志位置与就绪表现：【AI 填写】
- 入口地址：【AI 填写】
- 版本、前端入口、会员／管理员 bootstrap 等适用就绪检查：【AI 填写】

用户可按本节启动，也可请求 AI 协助恢复。文档、技能或不需要运行栈的任务应明确写“无需启动”，不提供虚构命令。

## 身份与测试数据

| 用途 | 身份／权限 | 测试数据及准备方法 | 预期初始状态 |
| --- | --- | --- | --- |
| 【AI 填写】 | 【AI 填写实际账号、角色或不适用】 | 【AI 填写确定数据和准备方法】 | 【AI 填写】 |

## 检查顺序

### CHK-001：【AI 填写可观察目标】

- 设计依据：【AI 填写具体条目】
- 前置条件：【AI 填写入口、身份、数据状态和依赖】
- 操作步骤：
  1. 【AI 填写可照做的动作及输入值】
  2. 【AI 继续填写；按实际需要增删步骤】
- 明确预期：
  1. 【AI 填写每个关键动作后可直接观察的结果、等待条件或时限】
  2. 【AI 填写刷新、重新进入或后续状态等适用结果】
- 记录方式：【AI 填写通过／失败／受阻／未执行所需截图、日志、URL 或业务标识】
- 影响与恢复：【AI 填写会修改的数据及恢复方法；无影响时写“无”】

【AI 为所有真实检查项复制该结构，并保持 CHK 编号在同一任务内稳定。实际指南不得保留本示例标题或泛化检查内容。每个 CHK 必须在同轮 `human-report.md` 中有且只有一条具体结果行。】

## 修复轮复验范围

| 分类 | 本轮检查 | 原因或沿用依据 |
| --- | --- | --- |
| 修复问题 | 【AI 填写 ISSUE 与 CHK；R1 写“不适用”】 | 【AI 填写】 |
| 受影响流程 | 【AI 填写】 | 【AI 填写影响判断】 |
| 先前受阻／未执行 | 【AI 填写】 | 【AI 填写已恢复的前提】 |
| 沿用历史结果 | 【AI 填写来源轮次、版本和 CHK】 | 【AI 填写为什么未受影响；不得写成在本轮重测通过】 |

## 本轮现场收尾

- 本轮用户或 AI 启动的资源及归属：【AI 填写】
- 准确停止命令：【AI 填写，使用同一仓库、INSTANCE/profile；无需环境时写“不适用”】
- 停止后的核对：【AI 填写进程、端口和容器检查】
- 需要保留时：在报告中写明用途、地址、仓库/worktree、实例、会话／进程、日志和之后的停止命令。

完成独立项目后继续其他检查；受阻项及其依赖项如实保留。整轮完成或无法继续时提交报告，不要把未执行项填写为通过。
