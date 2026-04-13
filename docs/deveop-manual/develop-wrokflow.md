# Development Workflow

```mermaid
flowchart TD
    %% Development Phase
    A[Requirements Review<br/>Clarification / Estimation / Scheduling]
    B[Technical Solution Breakdown]
    C[Test Case Design<br/>Unit Tests / Integration Tests / E2E Tests]
    D[Implementation]
    E[Validation Testing<br/>Functional / Regression / Fix Verification]
    F{Acceptance Criteria Met?}

    %% Release Phase
    G[Pre-release Verification]
    H[Canary Release]
    I{Canary Stable?}
    J[Gradual Full Rollout]

    %% Post-release Phase
    K[Post-release Monitoring]
    L[Retrospective & Follow-up Actions]

    A --> B
    B --> C
    C --> D
    D --> E
    E --> F

    F -- No --> C
    F -- Yes --> G

    G --> H
    H --> I

    I -- No --> E
    I -- Yes --> J

    J --> K
    K --> L
```

## Notes

- The delivery loop includes `Test Case Design`, `Implementation`, and `Validation Testing`.
- The team repeats this loop until the feature meets the agreed acceptance criteria.
- `Validation Testing` should confirm functional correctness, regression safety, and readiness for release.
- After the feature passes acceptance, it moves through pre-release, canary rollout, and gradual full rollout with monitoring at each stage.