---
name: implement-ts-grpc-service
description: implements a typescript grpc-js server that wraps a typescript sdk behind a user-chosen *.proto service—adapting grpc requests to sdk calls and sdk results back to proto messages—so other languages can use sdk behavior over grpc; work stays anchored to the ts package directory that owns the proto. use when the user selects a concrete .proto under a ts project and asks to implement the service as thin sdk bridging, not reimplemented sdk logic.
---

# Implement TypeScript gRPC Service (from Proto + SDK)

## Overview

**Primary scenario:** The user has **selected a concrete `*.proto` file** and asked you to **implement that proto’s gRPC `service`** (server handlers, registration, codegen, and related wiring). The same file **resides under a TypeScript project directory** in the repository—a package root with `package.json`, `tsconfig`, and normal Node/TS layout—so all work is **anchored to that TS project**: follow its existing `proto:gen` (or add one consistently), generated output paths, imports, and file layout instead of creating a second, parallel structure elsewhere.

**Out of scope as the main guide:** The proto is not under any TS package; or the user only wants `.proto` or API design **without** a TypeScript gRPC server—unless they explicitly expand the task to include this package-bound implementation.

**Architectural role:** The `service` in that `.proto` is the **gRPC façade** over **selected capabilities of a TS SDK** installed in the same package (or reachable from it). The point is to expose SDK behavior to **other runtimes and languages** via gRPC instead of requiring each caller to use the SDK directly.

**What each RPC implementation does:** Treat the handler as a **thin adapter**: (1) **transform** the incoming proto request into the shapes and arguments the SDK method expects; (2) **call** the SDK; (3) **transform** the SDK return value (and any uniform success/error envelope the SDK uses) into the outgoing proto response; (4) map failures to appropriate gRPC status. **Do not reimplement** the SDK’s business logic inside the server—keep behavior in the SDK and keep the TS layer to wiring, validation at the boundary, and mapping.

## Mandatory intake (blocking)

Do not implement handlers, codegen, or wiring until **both** the **`.proto`** and the **TypeScript package directory** are available as concrete context.

1. **Proto file (required):** The user must provide the **`*.proto` in a form you can read**—typically a **repository-relative path** to the file, or the file **open/attached in the editor** so imports, `service` definitions, and messages are inspectable. Naming an RPC or a service **without** the actual `.proto` source is not enough.
2. **TS project directory (required):** The user must provide the **repository-relative path to the TypeScript package root** that will own the server implementation—the directory that contains (or will contain) `package.json`, `tsconfig`, `src/`, `cmd/`, etc. Do not guess a folder from repo layout alone; the user must point at the intended **context directory** so you align `proto:gen`, imports, and file placement with that package. If either the proto or this directory is missing from context, stop and ask using [assets/intake-reply-template.md](assets/intake-reply-template.md).

<!-- TBD: additional numbered prerequisites (SDK path, RPC scope, codegen, …) -->

### Copy-paste template (user)

```markdown
Implement TS gRPC service:

- **Proto** (required): repository-relative path to `*.proto`, or confirm the file is open in the editor.
- **TS package directory** (required): repository-relative path to the TS project root (e.g. `third-party/foo/`).
- <!-- TBD: further fields -->
```

## Workflow

1. **Regenerate protobuf outputs:** From the **TS package root** (the directory from mandatory intake item 2—the folder that contains `package.json`), run **`npm run proto:gen`**. That refreshes the generated TypeScript / `@grpc/grpc-js` artifacts from the current `.proto` files so types and `*Service` definitions match the contract **before** you add or change RPC handlers. If `proto:gen` is missing or the script fails, stop and fix or align `package.json` / `protoc` inputs with the user instead of hand-editing `src/gen/**`.

2. **Mirror this repository’s gRPC sidecar pattern, then close gaps:** Read how **this codebase** turns a `.proto` `service` into a runnable `@grpc/grpc-js` server—entrypoint (`cmd/server` or equivalent), `addService` + generated `*Service`, thin handlers, SDK client factory, request/response mappers, and gRPC error helpers. Use [references/opinion-sidecar-layout.md](references/opinion-sidecar-layout.md) and the **target TS package’s** existing files as the live template; match naming, file split, and error mapping before writing new code. Then **implement the RPCs that are still missing** (replace stubs such as `UNIMPLEMENTED`, empty bodies, or TODOs) by wiring proto request/response types to the SDK using the **Overview** adapter rules—without changing the `.proto` unless the user asked to.

<!-- TBD: further workflow steps -->

## Hard rules

<!-- TBD: normative bullets -->

## Verification and reporting

<!-- TBD: when; pass criteria; failure report -->

## Preflight checklist

- [ ] <!-- TBD -->
- [ ] <!-- TBD -->

## Postflight checklist

- [ ] <!-- TBD -->
- [ ] <!-- TBD -->

## Resources

- [assets/intake-reply-template.md](assets/intake-reply-template.md) — <!-- TBD -->
- [references/opinion-sidecar-layout.md](references/opinion-sidecar-layout.md) — <!-- TBD -->

