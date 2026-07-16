# Global System Prompt

## Table of Contents

- [Rules](#rules)
  - [Project Status](#project-status)
  - [Test Data and API Keys](#test-data-and-api-keys)
  - [No Historical Compatibility](#no-historical-compatibility)
  - [No Tests](#no-tests)
  - [Chinese Plans](#chinese-plans)
  - [Living Design Documentation](#living-design-documentation)
  - [UI Layout Review](#ui-layout-review)

## Rules

### Project Status

The project is currently in development and has not been launched yet. Necessary breaking changes are allowed, and backward compatibility with historical behavior is not required.

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
