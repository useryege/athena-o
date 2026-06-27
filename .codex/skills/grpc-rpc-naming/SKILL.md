---
name: grpc-rpc-naming
description: Standardize gRPC/proto RPC method names. Use when Codex needs to design, name, rename, or review `rpc` methods in `.proto` service definitions, especially when choosing CRUD or business-action method names.
---

# gRPC RPC Naming

## Core Rule

Name RPC methods as `PascalCase` + verb-first + resource/action target:

```proto
rpc GetUser(...) returns (...);
rpc ListUsers(...) returns (...);
rpc CreateOrder(...) returns (...);
rpc DisableAccount(...) returns (...);
```

Prefer names that read as a clear operation. Keep IDs, filters, pagination, sort options, and lookup criteria in request fields rather than in the RPC method name.

## Standard Verbs

Use these verbs for common resource operations:

- `Get<Resource>`: get one resource.
- `List<Resources>`: list resources; use plural resource names.
- `Create<Resource>`: create one resource.
- `Update<Resource>`: update one resource.
- `Delete<Resource>`: delete one resource.
- `BatchGet<Resources>`: get multiple resources by explicit identifiers.
- `BatchCreate<Resources>`: create multiple resources.

Examples:

```proto
rpc GetUser(GetUserRequest) returns (GetUserResponse);
rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersResponse);
```

## Business Actions

For domain actions, still start with a precise verb:

```proto
rpc EnableUser(...) returns (...);
rpc DisableUser(...) returns (...);
rpc ArchiveOrder(...) returns (...);
rpc CancelOrder(...) returns (...);
rpc SubmitOrder(...) returns (...);
rpc ApproveApplication(...) returns (...);
rpc RejectApplication(...) returns (...);
```

Prefer the domain verb that matches the state transition or command being requested. Do not force every method into CRUD when a business action is clearer.

## Avoid

Avoid unclear, redundant, or non-standard patterns:

- `UserInfo`: missing a verb.
- `QueryUser`: prefer `GetUser` or `ListUsers`.
- `AddUser`: prefer `CreateUser`.
- `RemoveUser`: prefer `DeleteUser`.
- `GetUserList`: prefer `ListUsers`.
- `GetUserById`: prefer `GetUser`; put `id` in the request.
- `get_user`, `getUser`, `GET_USER`: use PascalCase.

## Review Output

When reviewing or proposing names:

- Return the recommended RPC name first.
- Give a short rationale tied to verb choice, resource plurality, or request-field placement.
- If replacing an existing name, show `OldName -> NewName`.
- Keep recommendations limited to method names unless the user explicitly asks for message, field, or service naming.
