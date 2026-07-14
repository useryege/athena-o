# Service-Core Local Surfaces

## Contents

- Source of truth
- Default services
- Playwright discovery
- Runtime operations
- Artifact conventions

## Source of truth

Read the repository-root `AGENTS.md` and `resource/doc/local-development.md` before use. They override this reference. Obtain local credentials from `AGENTS.md`; never copy them into a runner or report.

## Default services

| Surface | Default address |
|---|---|
| API health | `http://127.0.0.1:6776/base/health` |
| Swagger | `http://127.0.0.1:6776/swagger/index.html` |
| Admin login | `http://127.0.0.1:5173/admin/login` |
| Team login | `http://127.0.0.1:5174/login` |
| Official website | `http://127.0.0.1:5175` |
| MySQL | `127.0.0.1:23306` |
| Redis | `127.0.0.1:26379` |
| MinIO API | `http://127.0.0.1:19000` |
| Goreman RPC | `127.0.0.1:8555` |

Honor `SERVICE_CORE_*` overrides when present.

## Playwright discovery

1. Prefer `<repo>/e2e/node_modules/playwright` and read its package version.
2. Prefer `/usr/bin/google-chrome`; allow `SERVICE_CORE_ACCEPTANCE_CHROME_BIN` override.
3. Run Chrome headless. Do not download browsers or install dependencies during acceptance.
4. When the Codex in-app browser rejects the WSL workspace mapping, use this direct WSL Playwright path rather than treating the product as broken.
5. Set `SERVICE_CORE_ACCEPTANCE_TARGETS` to a comma-separated subset such as `api,admin,mysql,redis,minio` when the changed feature does not involve every frontend. The preflight default checks both frontends and all core services.

## Runtime operations

- Inspect: `$HOME/go/bin/goreman -p 8555 run status` when `goreman` is not on PATH.
- Restart only the changed process: `goreman -p 8555 run restart api|admin-ui|team-ui`.
- Stop an environment started by Codex with `make stop`, then verify owned ports and `service-core-local-*` containers exited.
- Preserve a user-started environment unless the user requests cleanup.

## Artifact conventions

- Runner: `.tmp/<feature>-acceptance.mjs`
- Run directory: `.tmp/<feature>-<UTC timestamp>/`
- Required: `final-report.md`, `report.json`, role traces, critical screenshots.
- Optional: read-only database snapshot and redacted request/response summaries.
- Never stage `.tmp` artifacts.
