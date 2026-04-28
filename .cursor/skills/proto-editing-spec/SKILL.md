---
name: proto-editing-spec
description: Defines mandatory .proto editing standards for this repository (based on the RepositoryService style), including message naming, field tags, RPC+HTTP annotations, and compatibility constraints. Use when creating or editing proto files, service APIs, field identifiers, or backward-compatible schema changes.
---

# Proto Editing Spec

Summarize proto file coding standards; every future proto edit must follow this specification.

## Scope

- Applies to all `.proto` file edits in this repository.
- Default syntax version is `proto3`.
- Highest priorities: backward compatibility and runtime performance.

## File Layout Order

Organize top-level declarations in this order:

1. `syntax = "proto3";`
2. file-level comments (optional; recommended for service protos)
3. `package ...;`
4. `import ...;`
5. `option go_package = "...";`
6. `message` definitions
7. `service` definitions

## Naming Rules

- `message` names use `PascalCase` (e.g., `RepoQuery`).
- `rpc` names use `PascalCase` verb phrases (e.g., `ListRepositories`).
- Field names use `lowerCamelCase` (e.g., `appProject`, `githubAppID`).
- Keep suffix semantics consistent:
  - `*Query`: read/query input
  - `*Request`: write/mutation input
  - `*Response`: output or empty action response

## Field and Tag Rules

- Never modify existing field tag numbers.
- Never reuse removed tags; protect removed fields with `reserved` when needed.
- New fields must use new tags appended at the tail.
- Choose `int32` / `int64` deliberately based on value range and cross-language compatibility.
- Use `repeated` only when list semantics are required.
- Use fully-qualified external type names when needed to avoid ambiguity.

## Comment Rules

- Use concise `//` comments to describe:
  - message intent
  - non-obvious fields
  - behavior of each `rpc`
- Comments should describe API semantics only, not implementation details.
- Compatibility-transition RPCs must explicitly be marked as `deprecated`.

## Service and HTTP Annotation Rules

- Every externally reachable `rpc` must declare a `google.api.http` mapping.
- Keep HTTP verb semantics consistent:
  - list/read: `get`
  - create/validate: `post`
  - update: `put`
  - delete: `delete`
- Create/update endpoints must explicitly set `body`.
- Paired resources (e.g., normal repo / write repo) should use mirrored route naming.
- Keep compatibility aliases as old RPCs and mark them with `option deprecated = true`.

## Compatibility and Performance Constraints

- Prefer additive schema evolution to avoid breaking wire format.
- Do not change existing field types unless a migration strategy is explicitly requested.
- Keep request/response messages minimal to reduce serialization and network overhead.
- Avoid deep nesting or oversized payloads unless truly required.

## Editing Checklist (Must Pass)

- [ ] Declaration order follows this spec.
- [ ] Naming is correct (`PascalCase` for `message/rpc`, `lowerCamelCase` for fields).
- [ ] Existing field tags remain unchanged.
- [ ] New fields use new tags without breaking compatibility.
- [ ] HTTP annotations are complete with correct method/body settings.
- [ ] Legacy compatibility RPCs are explicitly marked as `deprecated`.
- [ ] RPCs and non-obvious fields have clear comments.
- [ ] No unnecessary payload growth or performance regression is introduced.

## Conflict Resolution

- If a task explicitly belongs to `ts-sdk-to-proto`, follow that skill's stricter rules first.
- For all other proto editing scenarios, this skill is the default standard.
