---
name: ts-sdk-to-proto
description: defines mandatory prerequisites and scope rules when converting a typescript sdk under node_modules to protocol buffers (.proto). requires an explicit node_modules subdirectory as sdk context and a bounded list of sdk methods or functions to model. use when the user mentions ts-sdk to proto, typescript sdk to protobuf, or generating proto from a node sdk.
---

# TypeScript SDK to Protocol Buffers

## Overview

This skill governs how to turn a bounded subset of a TypeScript/JavaScript SDK (installed under `node_modules`) into `.proto` definitions. Do not author or guess protos until mandatory intake is complete.

## Mandatory intake (blocking)

Collect both items below before reading SDK sources for conversion. If either is missing, stop and ask using [assets/intake-reply-template.md](assets/intake-reply-template.md).

1. **SDK context**: A concrete path to the installed package directory **under `node_modules`** (repository-relative). Examples: `node_modules/@scope/pkg`, `third-party/foo/node_modules/pkg`. Do not rely on package name alone or vague descriptions.
2. **Bounded scope**: An explicit, finite list of symbols to model (class methods, exported functions, or file + symbol). Open-ended requests such as “convert the whole SDK” are invalid; require a narrowed list. Model only listed symbols—do not expand to unlisted exports or transitive APIs.

## Conversion workflow

1. Treat the given `node_modules/...` tree as the **only** authoritative SDK source. Use that package’s `package.json` (`main`, `types`, `exports`) to locate entry points when needed.
2. For **listed symbols only**, derive request/response shapes (including errors and async where relevant) and map them to `service` and `message` definitions. Follow existing project proto conventions for naming and field numbering; if none exist, confirm with the user before inventing patterns.
3. Do not pull in unlisted symbols from the same file or dependency chain.

## Preflight checklist

- [ ] SDK path under `node_modules` is known and specific.
- [ ] Scope is an enumerated list of methods/functions/symbols.
- [ ] No scope creep beyond that list.

## Resources

- [assets/intake-reply-template.md](assets/intake-reply-template.md) — message to send when intake is incomplete.
- `scripts/` — reserved for repeatable automation.
- `references/` — reserved for extended notes, examples, or policies.
