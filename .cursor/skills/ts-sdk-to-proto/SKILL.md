---
name: ts-sdk-to-proto
description: defines mandatory prerequisites and scope rules when converting a typescript sdk under node_modules to protocol buffers (.proto), requires proto field identifiers to match TS SDK field naming per-field (camelCase, PascalCase, etc.), plus mandatory post-generation verification that gRPC request/response match the TS SDK; any mismatch must be reported to the user with possible non-error rationale. requires an explicit node_modules subdirectory as sdk context, a bounded list of sdk methods or functions to model, and the target output .proto path. use when the user mentions ts-sdk to proto, typescript sdk to protobuf, or generating proto from a node sdk.
---

# TypeScript SDK to Protocol Buffers

## Overview

This skill governs how to turn a bounded subset of a TypeScript/JavaScript SDK (installed under `node_modules`) into `.proto` definitions. Do not author or guess protos until mandatory intake is complete.

## Mandatory intake (blocking)

Collect all items below before reading SDK sources for conversion. If any is missing, stop and ask using [assets/intake-reply-template.md](assets/intake-reply-template.md).

1. **SDK context**: A concrete path to the installed package directory **under `node_modules`** (repository-relative). Examples: `node_modules/@scope/pkg`, `third-party/foo/node_modules/pkg`. Do not rely on package name alone or vague descriptions.
2. **Bounded scope**: An explicit, finite list of symbols to model (class methods, exported functions, or file + symbol). Open-ended requests such as “convert the whole SDK” are invalid; require a narrowed list. Model only listed symbols—do not expand to unlisted exports or transitive APIs.
3. **Output proto**: The repository-relative path to the **target `.proto` file** where conversion results must be written (or appended). Typically an `@`-style or plain path reference to an existing `**/*.proto` (e.g. `third-party/opinion/opinion/opinion.proto`). Do not invent a new file path or choose one without the user’s explicit target.

### Examples (intake)

**Template (copy and fill):**

```markdown
Please convert TS SDK → proto.

- **SDK context**:
- **Bounded scope**:
- **Output proto**:
```

## Conversion workflow

1. Treat the given `node_modules/...` tree as the **only** authoritative SDK source. Use that package’s `package.json` (`main`, `types`, `exports`) to locate entry points when needed.
2. For **listed symbols only**, derive request/response shapes (including errors and async where relevant) and map them to `service` and `message` definitions. **Field names must align with the TS SDK** (see [Field naming (mandatory)](#field-naming-mandatory)). Follow existing project proto conventions for **field numbering** and other non-name rules; if none exist, confirm with the user before inventing patterns.
3. Do not pull in unlisted symbols from the same file or dependency chain.
4. **After** `.proto` changes are written, **run verification** (see below). Do not treat the task as finished until verification is done and any findings are communicated.

## Field naming (mandatory)

- **Align with the TS SDK per field**: For every property taken from a TypeScript interface, class field, or response object, the corresponding `.proto` field **must use the same identifier spelling and casing** as in the SDK (e.g. `camelCase`, `PascalCase`, `snake_case` if the SDK uses it). Do not rename for “proto style” alone when that diverges from the source.
- **Consistency**: Prefer **the same field names** as the SDK; only differ when a proto keyword/reserved name forces a change—then document the mapping and call it out in verification.
- **Mixed casing in one message is allowed** when the SDK does the same.

**Example** — if the SDK shape uses `GoodBoy` and `smallGirl`, the proto message should use those exact field names (types depend on the real SDK mapping):

```text
// TS (conceptual)
interface Example {
  GoodBoy: string;
  smallGirl: string;
}

// .proto — names and casing match the SDK
message Example {
  string GoodBoy = 1;
  string smallGirl = 2;
}
```

## RPC request/response constraints (mandatory)

- **No shared RPC envelope messages**: do not reuse the same request or response message type across multiple RPC methods. Define RPC-specific messages even when payload fields are identical. This avoids `RPC_REQUEST_RESPONSE_UNIQUE` style lint failures such as `"pkg.FooResponse" is used as the request or response type for multiple RPCs`.
- **Response naming must follow RPC name**: each RPC response message should be named `<RpcName>Response` (or `<ServiceName><RpcName>Response` if project conventions require service-prefixed names). Avoid generic names like `CancelOrderApiResponse` when the RPC is `CancelOrder`.

## Enum constraints (mandatory)

- **Enum zero value must be UNSPECIFIED**: every proto enum must reserve numeric `0` for an `_UNSPECIFIED` value (for example `SDK_ORDER_SIDE_UNSPECIFIED = 0`). Do not assign business values like `BUY`, `SELL`, `ACTIVE`, etc. to zero.
- **Shift concrete values away from zero**: when fixing or generating enums, assign domain values from `1+` to satisfy lint rules and preserve explicit “unset” semantics.
- **Do not treat UNSPECIFIED as a real business value in adapters**: request parsing/conversion logic must reject or explicitly handle `_UNSPECIFIED` instead of silently mapping it to a concrete SDK enum.

## Post-generation verification (mandatory)

**When:** Immediately after generating or updating the target `.proto` for the scoped symbols.

**Standard:** For each modeled RPC, the gRPC **request** and **response** must match the TypeScript SDK’s observable **inputs** and **outputs** for the corresponding method or function (including wrapper types such as `ApiResponse<T>`, query/path/body splits, and nested `result` payloads). Compare semantics field-by-field: **field names and casing must match the SDK** unless a documented exception applies ([Field naming (mandatory)](#field-naming-mandatory)). Also check optional vs required and flattening vs nesting.

In the same verification pass, ensure RPC envelopes satisfy [RPC request/response constraints (mandatory)](#rpc-requestresponse-constraints-mandatory), especially uniqueness-per-RPC and `<RpcName>Response` naming.

Also verify enums satisfy [Enum constraints (mandatory)](#enum-constraints-mandatory), especially zero-value `_UNSPECIFIED` and non-zero business values.

**If everything aligns:** State briefly that verification passed for the listed symbols.

**If anything does not align:** You **must** report this to the user in the same turn. The report must include:

- Which RPC or message diverges from which SDK symbol.
- What differs (request vs response, missing/extra/renamed fields, structural nesting, type width, etc.).
- A short **hypothesis** for *why* it might still be acceptable—e.g. intentional proto flattening, backward compatibility, gRPC status vs HTTP `errno`, or project-wide naming rules—so the user can judge whether it is an error or a deliberate tradeoff.

Absence of a mismatch does not require a long explanation; a mismatch **always** requires explicit user-visible reporting—do not silently leave drift undocumented.

## Preflight checklist

- [ ] SDK path under `node_modules` is known and specific.
- [ ] Scope is an enumerated list of methods/functions/symbols.
- [ ] Target output `.proto` path is explicit and user-provided.
- [ ] No scope creep beyond that list.

## Postflight checklist

- [ ] Verification ran against the TS SDK for every scoped RPC (request + response).
- [ ] Field names/casing match the SDK per [Field naming (mandatory)](#field-naming-mandatory), or divergences are explicitly reported.
- [ ] RPC request/response message types are unique per RPC and response message naming follows `<RpcName>Response` (or approved service-prefixed form).
- [ ] Every enum uses `_UNSPECIFIED = 0`, business values start at `1+`, and adapter logic does not silently map UNSPECIFIED to a concrete SDK value.
- [ ] Any mismatch is documented to the user with concrete diffs and possible non-error rationale.

## Resources

- [assets/intake-reply-template.md](assets/intake-reply-template.md) — message to send when intake is incomplete.
- `scripts/` — reserved for repeatable automation.
- `references/` — reserved for extended notes, examples, or policies.
