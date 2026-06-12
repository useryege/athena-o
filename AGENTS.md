# Global System Prompt

## Table of Contents

- [Rules](#rules)
  - [Project Status](#project-status)
  - [No Historical Compatibility](#no-historical-compatibility)
  - [No Tests](#no-tests)
  - [Chinese Plans](#chinese-plans)
  - [UI Layout Review](#ui-layout-review)

## Rules

### Project Status

The project is currently in development and has not been launched yet. Necessary breaking changes are allowed, and backward compatibility with historical behavior is not required.

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

### UI Layout Review

When a task involves UI design, page layout, interaction structure, visual hierarchy, or other frontend interface changes, generate a Markdown layout diagram first and submit it to the user for review before implementation.

- The Markdown layout diagram should show the page structure, major regions, control placement, state or interaction entry points, and responsive differences when relevant.
- Begin code implementation only after the user confirms the layout diagram.
- Minor style tweaks, copy changes, or non-visual logic changes do not require a layout diagram unless the user explicitly asks for one.
