---
name: third-party-pkg-style-guide
description: provide a style guide for building reusable go packages for third-party http services under `util/third-party-name`. use when implementing or standardizing third-party http service integrations so request logic stays out of business code.
---

# Third-Party Pkg Style Guide

## Overview

Use this skill as a style guide when designing or implementing a reusable Go package for a third-party HTTP service.

Keep the package as the transport boundary so business code works with business-facing methods instead of raw HTTP details.

## Intake

Collect the minimum information before coding:

1. Third-party service name and target package path, usually `util/third-party-name`.
2. Required capabilities and the business operations the caller actually needs.
3. Base URL, headers, versioning, and environment differences.
4. Non-functional requirements such as timeout, retry, pagination, idempotency, rate limits, and logging.
5. Error handling expectations.

If the request is still vague, reply with:

```markdown
Current understanding
- Third-party service:
- Target capabilities:
- Package path: `util/third-party-name`

Suggested package design
- Package name:
- Public API:
- Config options:
- Error strategy in `client.go`:

Open questions
- ...

Implementation plan
1. ...
2. ...
3. ...
```

## Design Rules

Default to these rules unless the user explicitly asks for a different shape:

- Put the integration in `util/third-party-name`.
- Start with `config.go` and `client.go` only.
- Expose business-oriented methods such as `NewClient(...)`, `GetUserInfo(...)`, or `DeleteOrder(...)`.
- Keep route building, headers, serialization, retries, pagination, and remote error mapping inside the package.
- Put config options in `config.go` and package-owned request flow in `client.go`.
- Keep local error definitions at the top of `client.go`.
- Define request or response structs immediately above the first method that uses them.
- Model only the endpoints needed now. Do not generate a full SDK unless the user explicitly wants one.

Avoid these patterns:

- Raw platform request assembly in handlers, services, or controllers.
- Public APIs centered on generic transport helpers such as `Do(...)` or `Call(path, method, body)`.
- Wide `map[string]any` contracts when stable typed models are practical.
- Hardcoded secrets, domains, or environment-specific configuration.
- Splitting into extra files before the package is large enough to justify them.

## Implementation Workflow

Use this sequence:

1. Check whether `util` already contains a similar wrapper and avoid duplicate abstractions.
2. Confirm the package name and the public API before splitting files.
3. Start from `references/config.template.md` to shape options and defaults.
4. Build `client.go` from `references/client.template.md`.
5. Keep local errors at the top of `client.go` and place structs near first use.
6. Verify that business code no longer builds raw platform requests directly.

When deciding whether the package should stay compact or grow, read `references/package-blueprint.md`.

## Quality Bar

The resulting package should:

- Hide protocol details behind business-facing methods.
- Preserve enough remote error context to debug failures.
- Map platform failures into errors the caller can reason about.
- Stay small and capability-driven instead of becoming a generic request framework.
- Default to a compact `config.go + client.go` layout unless the task clearly requires more structure.

## Resources

- `references/package-blueprint.md`: compact blueprint for deciding package shape and boundaries
- `references/config.template.md`: option-based config template for `config.go`
- `references/client.template.md`: single-file client template with local errors and near-use struct definitions
