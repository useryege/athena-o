---
name: llm-friendly-api-wrapper
description: Build and maintain LLM-friendly third-party API wrappers in ATHENA-style Go modules using llms.txt/OpenAPI docs, public read snapshot collection, typed clients, drift handling, and gated unit/integration tests. Use when integrating a new vendor API or auditing schema drift for an existing wrapper.
---

# LLM Friendly API Wrapper

## Overview

Use this skill to standardize third-party API integration for LLM-friendly projects.

Follow a public-read-first workflow:
1. Sync docs from `llms.txt`.
2. Snapshot real request/response payloads.
3. Implement typed Go wrappers.
4. Guard with unit tests + gated integration tests.
5. Iterate on schema drift.

## Workflow

### 1) Sync docs into local vendor module

Create vendor module under `util/<vendor>/`.

Adopt this structure:
- `util/<vendor>/sync-docs.sh`
- `util/<vendor>/<vendor>-docs/`
- `util/<vendor>/README.md`
- `util/<vendor>/Makefile`

Use `scripts/bootstrap_vendor_layout.sh` to scaffold layout.

Implement doc sync script behavior:
- Fetch `llms.txt` index first.
- Extract markdown doc URLs.
- Rebuild local docs directory on each run.
- Fail fast when no URLs are discovered.

### 2) Extract public read endpoints

Treat local `api-reference/**/*.md` as source of truth.

Apply endpoint inclusion rules:
- Include all `GET` endpoints by default.
- Include `POST` only when semantics are read-only (`list/get/search/check`).
- Exclude write/auth-sensitive endpoints by rule (`trade`, `api-keys`, write `relayer`, private user streams).

Use snapshot command to verify extraction instead of guessing from docs manually.

### 3) Capture live request/response snapshots

Place snapshot command in vendor module:
- `util/<vendor>/cmd-<vendor>-public-read-snapshot/`

Persist snapshots into:
- `util/<vendor>/request-response/latest/`

Snapshot command requirements:
- Discover sample IDs automatically from public endpoints.
- Continue on per-endpoint failure.
- Write per-endpoint `request.json`, `response.json`, `meta.json`.
- Write root `summary.json` and `index.ndjson`.
- Return non-zero exit code if any endpoint fails.

Use snapshots as drift evidence before changing models.

### 4) Implement Go wrapper with ATHENA pattern

Use this client shape:
- `DefaultBaseURL`, `DefaultTimeout`
- `Config`
- exported `Client` interface
- private impl struct
- constructor `NewXxxClient`
- centralized `do()` request function

Design rules:
- Add `context.Context` to every method.
- Escape path params with `url.PathEscape`.
- Encode optional bool/number params as pointers when `false/0` is meaningful.
- Use repeated query keys for list params.
- Keep top-level response strongly typed.
- Use `json.RawMessage` for high-drift nested payloads.
- Return typed `APIError` with HTTP status + raw body (size-limited).

See `references/wrapper-contract.md` for modeling rules.

### 5) Build tests with explicit gates

Write unit tests first:
- URL/query/body encoding
- error decoding
- cancellation behavior
- representative decode paths

Write live integration tests second:
- Skip by default.
- Enable by env gates.
- Use method-level `t.Run` granularity.
- Add `..._LOG_RESPONSE=1` gate to print full payload when needed.

Use `scripts/check_public_read_coverage.sh` to audit docs/client/tests mapping.

### 6) Handle drift as a routine

Run this cadence:
1. `sync-docs`
2. snapshot dry-run
3. snapshot live run
4. compare `request-response/latest`
5. adjust models/tests

Prefer evidence-driven updates from snapshots over speculative schema changes.

See `references/drift-playbook.md` for triage priority.

## Commands Template

Use `scripts/render_make_targets.sh --vendor <vendor> --dry-run` to generate Makefile targets.

Recommended targets:
- `sync-docs`
- `snapshot-public-read`
- `snapshot-public-read-dry`
- `test-unit`
- `test-integration`

## References

Load references on demand:
- `references/polymarket-blueprint.md`
- `references/wrapper-contract.md`
- `references/test-gates-and-commands.md`
- `references/drift-playbook.md`
