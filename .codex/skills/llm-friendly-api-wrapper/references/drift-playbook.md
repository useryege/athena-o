# Drift Playbook

## Goals

- Detect schema drift early.
- Avoid blind model changes.
- Keep wrappers callable while schema evolves.

## Standard cycle

1. Sync latest docs from llms/OpenAPI sources.
2. Run snapshot dry-run to confirm endpoint extraction.
3. Run snapshot live to collect current payloads.
4. Inspect `summary.json` + failed endpoint logs.
5. Compare payload changes with current models.
6. Update wrapper types/decoders.
7. Re-run unit + integration tests.

## Triage priority

Priority 0:
- Method unusable (4xx from wrong request shape)
- Decoder panic or hard unmarshal failure

Priority 1:
- Top-level field type changes
- Required field missing/new field impacts business logic

Priority 2:
- Deep nested optional fields drift
- Cosmetic ordering/extra metadata changes

## Adaptation strategy

- For top-level contract changes, update strong types promptly.
- For unstable nested objects, migrate to `json.RawMessage` or optional structs.
- Keep request construction exact and deterministic.
- Record drift rationale in code comments only when non-obvious.

## Evidence standards

Do not change models from assumptions.

Always attach evidence from:
- snapshot payloads, or
- direct curl samples, or
- repeatable integration test output.
