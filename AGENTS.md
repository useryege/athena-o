# Global System Prompt

## Table of Contents

- [Rules](#rules)
  - [Project Status](#project-status)
  - [Requirement-Driven Architecture](#requirement-driven-architecture)
  - [Test Data and API Keys](#test-data-and-api-keys)
  - [No Historical Compatibility](#no-historical-compatibility)
  - [No Tests](#no-tests)
  - [Chinese Plans](#chinese-plans)
  - [Living Design Documentation](#living-design-documentation)
  - [UI Layout Review](#ui-layout-review)
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

### Living Design Documentation

`docs/design/` contains the repository-internal, English-language explanation of the currently implemented design for developers and AI agents.

- Before planning or implementing a change, read `docs/design/README.md` and every design document relevant to the affected subsystem or capability.
- Update the relevant design document in the same change when implementation changes component responsibilities, boundaries, runtime flow, state machines, data models, interface contracts, configuration defaults, dependency relationships, failure recovery, health checks, or observability.
- When adding a subsystem or independently understandable capability, create a document from `docs/design/template.md` and register it in `docs/design/README.md`.
- Purely internal refactors, formatting changes, copy edits, and generated-file updates that do not change design semantics do not require a design-document update.
- Describe only the current implementation. Replace obsolete content instead of retaining compatibility notes, change histories, future plans, or deprecated designs as an archive.
- Link to actual source paths and name the important symbols instead of copying large code sections into documentation.
- Executable code is the source of truth. If code and documentation disagree, inspect the code and correct the documentation in the same task.
- The implementer is responsible for keeping the affected design documents synchronized; documentation maintenance is not a separate follow-up task.

### UI Layout Review

When a task involves UI design, page layout, interaction structure, visual hierarchy, or other frontend interface changes, generate a Markdown layout diagram first and submit it to the user for review before implementation.

- The Markdown layout diagram should show the page structure, major regions, control placement, state or interaction entry points, and responsive differences when relevant.
- Begin code implementation only after the user confirms the layout diagram.
- Minor style tweaks, copy changes, or non-visual logic changes do not require a layout diagram unless the user explicitly asks for one.

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
