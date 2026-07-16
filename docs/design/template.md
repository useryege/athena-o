# Design Document Title

Use this template for the current implemented design of one subsystem or independently understandable capability. Remove all instructional text and placeholders before registering the document in the [design index](README.md).

## Scope

State what the capability owns, what behavior this document explains, and the most important adjacent responsibilities that are outside its boundary.

## Source Locations

Link the implementation entry points and name the important symbols. Include application logic, adapters, persistent storage, configuration, and process wiring when relevant.

| Concern | Source | Key symbols |
| --- | --- | --- |
| Process entry | `path/to/file` | `SymbolName` |

## Architecture

Describe component responsibilities, boundaries, dependencies, and the direction of calls or data movement. Add a small diagram only when it makes those relationships materially clearer.

## Runtime Flow

Describe the current end-to-end execution sequence, including startup, steady-state work, and shutdown. Identify concurrency and transaction boundaries explicitly.

## State / Data

Describe durable state, in-memory state, ownership, lifecycle, uniqueness constraints, and the point at which state transitions become committed.

## Configuration

List the configuration sources, relevant fields, defaults, validation rules, and which behavior each setting controls. Do not copy secrets or unrelated environment variables.

## Invariants

List the conditions the implementation relies on and must preserve across changes.

## Failure Recovery

Explain how dependency failures, partial work, retries, restarts, cancellation, and invalid configuration behave. State where atomicity prevents partial state.

## Observability

Document logs, health and readiness semantics, metrics, status endpoints, and the fields needed to diagnose the capability.

## Change Checklist

- [ ] Component responsibilities and boundaries still match this document.
- [ ] Runtime, concurrency, and transaction flows are current.
- [ ] State, data, interfaces, configuration, dependencies, and invariants are current.
- [ ] Failure recovery, health checks, and observability are current.
- [ ] Source links and named symbols resolve to the implementation.
- [ ] The [design index](README.md) contains the correct entry.
