---
name: external-platform-api-package
description: wrap third-party platform apis into reusable go packages under `util/<platform>`. use when integrating external platforms, saas apis, or third-party http services and the request logic should not be scattered through business code.
---

# External Platform API Package

## Overview

Use this skill to design or implement a reusable Go integration package for an external platform.

Keep the package as the boundary for transport concerns so business code works with typed inputs, typed outputs, and meaningful errors instead of raw HTTP details.

## Intake

Collect the minimum information before coding:

1. Platform name and target package path, usually `util/<platform>`.
2. Required capabilities and the business operations the caller actually needs.
3. Authentication model, base URL, versioning, and environment differences.
4. Non-functional requirements such as timeout, retry, pagination, idempotency, rate limits, and logging.
5. Error handling expectations and whether tests are expected in the same change.

If the request is still vague, reply with:

```markdown
Current understanding
- Platform:
- Target capabilities:
- Package path: `util/<platform>`

Suggested package design
- Package name:
- Public API:
- Key configuration:
- Error strategy:

Open questions
- ...

Implementation plan
1. ...
2. ...
3. ...
```

## Design Rules

Default to these rules unless the user explicitly asks for a different shape:

- Put the integration in `util/<platform>`.
- Expose business-oriented methods such as `NewClient(...)`, `CreateOrder(...)`, `GetUser(...)`, or `VerifyWebhook(...)`.
- Keep route building, auth, signing, headers, serialization, retries, pagination, and remote error mapping inside the package.
- Inject configuration through a constructor, config struct, or options. Prefer supporting `*http.Client` injection for reuse and testing.
- Model only the endpoints needed now. Do not generate a full SDK unless the user explicitly wants one.

Avoid these patterns:

- Raw platform request assembly in handlers, services, or controllers.
- Public APIs centered on generic transport helpers such as `Do(...)` or `Call(path, method, body)`.
- Wide `map[string]any` contracts when stable typed models are practical.
- Hardcoded secrets, domains, or environment-specific configuration.

## Implementation Workflow

Use this sequence:

1. Check whether `util` already contains a similar wrapper and avoid duplicate abstractions.
2. Confirm the package name and the public API before splitting files.
3. Create the smallest useful file layout for the current scope.
4. Implement package-owned concerns such as auth, serialization, and error mapping.
5. Add focused tests when they materially reduce risk.
6. Verify that business code no longer builds raw platform requests directly.

When the package needs more structure or examples, read `references/package-blueprint.md`.

## Quality Bar

The resulting package should:

- Hide protocol details behind typed, business-facing methods.
- Preserve enough remote error context to debug failures.
- Map platform failures into errors the caller can reason about.
- Stay small and capability-driven instead of becoming a generic request framework.

## Resources

- `references/package-blueprint.md`: detailed package blueprint, file responsibilities, and review checklist
- `scripts/validate-skill-english.sh`: validates that `SKILL.md`, `references/`, `scripts/`, and `assets/` stay free of CJK and full-width characters
- `assets/`: future templates or static materials if the skill grows
