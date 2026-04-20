---
name: sdk-grpc-parity-e2e
description: Defines how to write end-to-end tests that prove an official HTTP/SDK client and a thin gRPC sidecar return semantically identical payloads. Use when the user asks to generate or add E2E tests for a gRPC RPC in TypeScript sidecar code, when adding or reviewing parity tests between upstream SDK responses and grpc-js unary responses, for third-party sidecar validation, or when comparing SDK output to wrapped gRPC output field-by-field.
---

# SDK vs gRPC response parity (E2E)

## When to use

- The user wants **E2E tests for a specific gRPC RPC** implemented in TS (thin sidecar / grpc-js), not generic unit tests of mappers alone.
- Same scope as the description: parity coverage, sidecar validation, or field-by-field SDK vs gRPC comparison.

## Overview

Parity tests assert that **the same logical response** reaches callers whether they use the official SDK directly or the repository’s gRPC wrapper. The **source of truth** for “what the gRPC API should return” is not the raw SDK JSON; it is **the sidecar’s own mapping** (`parse*` + SDK call + `to*Response`). Do not compare unrelated shapes (e.g. SDK `result` vs flat proto fields) without applying the same mapper the server uses.

## Mandatory intake (blocking)

1. **Sidecar mapping entrypoints** — For each RPC under test: the proto request parser (`parse*Request`), the SDK method and its argument type, and the response mapper (`to*Response`). If these are missing or ambiguous, stop and locate them in the sidecar implementation before writing assertions.
2. **Runtime config parity** — Tests that instantiate the SDK must load the **same** environment as the running gRPC process (e.g. `import 'dotenv/config'` before `loadRuntimeConfig()`). Missing env vars cause false failures unrelated to parity.
3. **gRPC address** — A single convention for the test client (e.g. `OPINION_E2E_GRPC_ADDR` with a documented default port matching `GRPC_PORT`).

## Workflow

1. **Construct one proto request** — Same object you send to `OpinionServiceClient` (or equivalent).
2. **Derive the SDK query** — `query = parse*Request(protoRequest)` so the test uses **identical** parsing rules as the server.
3. **Call the SDK** — `createOpinionClient(loadRuntimeConfig().sdk)` (or project equivalent). Prefer a **fresh** client in the test rather than a process-wide singleton unless the repo standard says otherwise.
4. **Handle SDK failure modes** — If the SDK returns `errno !== 0`, do not blindly assert success. When upstream blocks certain regions, align skip behavior with gRPC: detect the same policy text in `errmsg` and `t.skip` instead of failing the suite. Re-throw unexpected SDK throws with `{ cause }` for debugging.
5. **Build the oracle** — `expected = to*Response(sdkResponse)` using the **same** function the sidecar uses after a successful SDK call.
6. **Call gRPC** — Unary call with the **same** proto request; wrap callback style in a Promise; `client.close()` in `finally`.
7. **Handle gRPC errors** — Map `UNAVAILABLE` to a clear “start the server” error; map likely upstream region blocks to `t.skip` when the sidecar surfaces them as `INTERNAL` with known message patterns; rethrow other errors.
8. **Assert** — Compare `expected` and `actual`. Prefer **protobuf wire equality**: `Message.encode(a).finish()` vs `encode(b).finish()` with `assert.deepStrictEqual` on the `Uint8Array` values. Alternative: `assert.deepStrictEqual` on full message objects if deserialization is stable. **Avoid** relying on `toJSON` alone when generated code omits default fields.

## Hard rules / standards

- **Never** compare raw SDK response bodies to gRPC messages without applying the sidecar’s `to*Response` to the SDK result first (unless the test explicitly documents a different contract).
- **Always** derive SDK inputs from `parse*Request(proto)` when that is what the server uses.
- **Document** in the test file header when the scenario performs **two upstream HTTP calls** (SDK + sidecar); list snapshots may drift and cause rare flakes—use small `limit`/fixed `page` when possible and mention the limitation in assertion messages on failure.
- **Share** region-block heuristics between SDK-path (`errno` + `errmsg`) and gRPC-path (`INTERNAL` + details/message) via one regex or helper so wording stays aligned.

## Verification and reporting

- **When**: Immediately after adding or changing a parity test or mapper.
- **Pass**: Encoded bytes (or agreed deep-equal) match for `expected` vs `actual` under stable upstream conditions.
- **If fail**: Report whether the mismatch is likely **mapping drift** (code bug) vs **upstream snapshot drift** (two requests). Include whether `errno`/gRPC status indicated region or availability issues.

## Preflight checklist

- [ ] Parser, SDK method, and `to*Response` for this RPC are identified.
- [ ] Test loads env the same way as the sidecar process.
- [ ] gRPC listen address and test client address conventions match.

## Postflight checklist

- [ ] Parity assertion uses wire bytes or documented deep-equal strategy.
- [ ] Region / UNAVAILABLE behaviors are handled without masking real regressions.
- [ ] Flaky risk from double-fetch is noted in comments or failure messages.

## Resources

- `scripts/` — Placeholder for future automation (e.g. codegen checks); empty until added.
- `references/opinion-sidecar-example.md` — Concrete file paths and patterns from the Athena `third-party/opinion` package.
- `assets/` — Placeholder for templates (e.g. test file boilerplate); empty until added.
