# Package Blueprint

Use this reference when implementing a reusable Go wrapper for an external platform.

The goal is to keep transport details inside `util/<platform>` and expose a small, business-facing API to the rest of the codebase.

## Minimal Layout

Start with the smallest layout that matches the current scope:

```text
util/<platform>/
├── config.go
└── client.go
```

Keep the layout compact unless the user explicitly asks for more structure.

Do not create files just to match a template. Add files only when they improve clarity.

## File Selection

Use the smallest set of files that keeps responsibilities clear:

- `client.go`: always present for client state, shared dependencies, and constructors
- `config.go`: add when constructor input grows beyond a few simple arguments

## Template Catalog

Read the specific template only when you need that file:

- `client.go` -> `client.template.md`
- `config.go` -> `config.template.md`

## Default Boundaries

Keep these concerns inside the package:

- Route and query construction
- Header assembly
- Local error definitions
- Request and response struct definitions near first use
- Request and response serialization
- Remote error mapping
- Protocol concerns such as retry, timeout, pagination, and idempotency behavior

Keep business code limited to typed inputs, typed outputs, and package-defined errors.

## Review Checklist

Use this checklist before considering the integration complete:

- The package lives in `util/<platform>`.
- Business code no longer contains raw platform request construction.
- Configuration is injected instead of hardcoded.
- Public methods represent business capabilities, not protocol primitives.
