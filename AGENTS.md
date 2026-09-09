# Global System Prompt

## Table of Contents

- [Rules](#rules)
  - [Project Status](#project-status)
  - [Requirement-Driven Architecture](#requirement-driven-architecture)
  - [Test Data and API Keys](#test-data-and-api-keys)
  - [No Historical Compatibility](#no-historical-compatibility)
  - [No Tests](#no-tests)
  - [Chinese Plans](#chinese-plans)
  - [Plan Implementation Completion Email](#plan-implementation-completion-email)
  - [Design-Gated Development](#design-gated-development)
  - [需求与技术设计讨论](#需求与技术设计讨论)
  - [UI Layout Review](#ui-layout-review)
  - [Impeccable Integration](#impeccable-integration)
  - [本地图片路径规则](#本地图片路径规则)

## Rules

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

### No Tests

Do not plan, propose, add, or run tests (unit tests, integration tests, end-to-end tests, etc.) unless the user explicitly asks for them. We trust the code we write.

### Chinese Plans

When outlining or listing a plan (implementation steps, task breakdown, approach summary, etc.), write the plan in Chinese (简体中文).

### Plan Implementation Completion Email

After a user-confirmed plan has been fully implemented, including all required code, configuration, documentation, and explicitly authorized verification, send exactly one completion email from the repository root before returning the final response:

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

### Design-Gated Development

- 修改业务逻辑、开展后端开发或设计前端 UI 前，先阅读 `docs/requirements/README.md`、`docs/design/README.md`、相关能力文档和实际源码，明确当前阶段、范围与预期行为。聊天记录不是跨任务事实来源；影响后续工作的确认结果必须先写回文档。
- 后端功能固定按“需求目标设计 → 后端技术设计 → 明确派发实现 → 设计一致性审查”推进。详细操作遵循 `docs/developer-guide/design-led-backend-development.md` 和仓库技能 `$athena-backend-feature-workflow`。
- 需求文档必须由用户明确确认并标记为 `已确认`，才能进入后端技术设计；仍为 `讨论中` 时不得开始技术设计或修改后端源码。
- 技术设计必须由用户明确确认并标记为 `已确认待实现`，才能作为实现依据。确认设计本身不构成实现授权；只有用户另行明确派发实现任务后，才能修改后端源码。
- 实现必须遵循已确认需求和技术设计。完成后在同一任务内同步文档、实际源码链接和设计状态，并按需求逐项进行设计一致性审查；文档维护不是事后独立补写任务。
- 业务规则、权限、业务状态、范围或可观察行为发生实质变化时，退回需求阶段并重新确认。组件边界、接口或数据契约、数据模型、事务与并发、核心流程或基础设施发生实质变化时，退回技术设计阶段并重新确认。
- 实现中发现实质偏差时，只继续不受偏差影响的部分；先把拟议变化和影响写回相应文档，等待重新确认，并由用户明确授权继续变更后的实现，再修改依赖该变化的源码。原实现授权不自动覆盖变化后的范围。
- 命名、辅助函数拆分和局部代码组织由实现者自主决定，只要不改变已确认行为或技术方案。纯格式化、注释、文案，以及源输入、契约和设计语义均未变化的生成文件刷新等机械修改，豁免上述设计确认流程；修改 SQL、Proto、ABI 等生成源不属于豁免。
- 用户明确要求修复、且现有代码只是偏离已确认设计时，可以直接按该设计修复。若期望行为未被已确认需求或设计定义，则必须先回到相应阶段补充并确认。
- 可执行代码是当前运行行为的事实来源。代码与标记为 `已实现` 的设计不一致时，先判断偏差来源；在已明确授权的实现或修复任务中同步修复代码或文档，单独审查任务只报告偏差而不擅自修改。不保留失效说明、历史兼容层或废弃方案。
- 前端 UI 在需求对齐后继续遵循 [UI Layout Review](#ui-layout-review)；本后端门禁不替代其布局确认要求。

### 需求与技术设计讨论

需求讨论和技术设计阶段，对需要用户决策的未确认要点，优先提供选择题，帮助用户理解取舍并作出明确选择。

- 提问前先查阅相关需求、设计文档和实际源码，区分已确认结论、待决策事项和缺失信息。不要重复询问已确认且未发生实质变化的事项；命名、辅助函数拆分等可自主决定的实现细节，按既有规则自行处理。
- 每道题聚焦一个决策，简述背景及其对需求或设计的影响，通常提供 2–3 个清晰、可比较的可行选项。单选题的选项应互斥；允许组合时说明组合关系，不为凑数量编造选项。用户可以提出选项之外的方案。
- 有充分依据时，将推荐项放在首位并标注“推荐”，结合当前目标、已确认约束和实际代码说明推荐理由，同时简述各选项的主要收益、代价或适用条件。推荐应帮助用户判断取舍，不能只给结论或将个人偏好表述为客观事实。
- 无法形成有意义的选项，或缺乏可靠推荐依据时，明确说明缺失的信息及其如何影响判断，并请求用户补充最少必要的上下文，例如业务场景、具体实例、优先级、规模或边界条件。已有可行选项但暂不能推荐时，可以先列出选项并说明推荐待补充信息后确定；不要强行推荐或用未经确认的假设替用户决定。
- 分轮收敛问题，每轮优先处理 1–3 个会影响当前阶段结论的关键事项；存在依赖关系时，先确认前置问题，再讨论后续选择。等待答复时，可继续当前阶段内不依赖该答案的工作。
- 将用户明确选择的结论及必要理由同步到对应需求或设计文档，并保留尚未解决的要点。推荐项、界面默认选项和用户未回复均不视为确认；单个要点的选择不等于整份需求或技术设计通过确认，也不构成实现授权，阶段推进仍遵循 [Design-Gated Development](#design-gated-development)。

### UI Layout Review

When a task involves UI design, page layout, interaction structure, visual hierarchy, or other frontend interface changes, first align the requirements and documents under [Design-Gated Development](#design-gated-development), then complete the UI design and generate a Markdown layout diagram for user review before implementation.

- The Markdown layout diagram should show the page structure, major regions, control placement, state or interaction entry points, and responsive differences when relevant.
- Begin code implementation only after the requirements and documents are aligned, the UI design is complete, and the user confirms the layout diagram. Reuse an already-confirmed design when the scope and expected behavior remain unchanged.
- Minor style tweaks, copy changes, or non-visual logic changes do not require a layout diagram unless the user explicitly asks for one.

### Impeccable Integration

- Impeccable is a UI/UX workflow aid, not an independent source of product or design truth. `docs/requirements/` and `docs/design/` remain authoritative, and repository instructions and confirmed documents override Impeccable defaults or detector findings.
- Scope Impeccable work to the frontend under `ui/`. Read the relevant requirements, `docs/design/web-ui/` documents, and current React/CSS implementation before using it; its own shaping or approval steps do not replace [Design-Gated Development](#design-gated-development) or [UI Layout Review](#ui-layout-review).
- Do not silently run `$impeccable init` or `$impeccable document`, and do not create `PRODUCT.md` or `DESIGN.md` as competing records. If Impeccable needs those files, first propose how they derive from the authoritative repository documents and wait for the user's explicit confirmation.
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
