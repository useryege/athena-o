# Running Athena Locally

Athena local development no longer requires a cluster. The local stack runs Athena processes with Redis and PostgreSQL provided locally or through Docker.

## Prerequisites

Complete [Development Environment](development-environment.md).

## Start Local Services

### Configure browser authentication

Create a Google Cloud **Web application** OAuth client for local development. Set the
OAuth consent audience to **External** and publish the application when arbitrary Google
users must be able to register. Google's Testing state admits only configured test users.
Register this authorized redirect URI exactly:

```text
http://localhost:4000/auth/google/callback
```

Set the local client and administrator bootstrap email in `.env`:

```env
ATHENA_GOOGLE_OIDC_CLIENT_ID='<local-web-client-id>'
ATHENA_GOOGLE_OIDC_CLIENT_SECRET='<local-web-client-secret>'
ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE=''
ATHENA_GOOGLE_OIDC_REDIRECT_URI='http://localhost:4000/auth/google/callback'
ATHENA_ADMIN_GOOGLE_EMAIL='<administrator-google-email>'
ATHENA_JWT_SECRET='<at-least-32-byte-signing-secret>'
```

No Google subject or Solana address is configured ahead of time. After Athena verifies an
unknown Google subject or Phantom wallet signature, it creates a shared 15-minute
browser-bound registration ticket and sends the browser to `/register`; it does not create
an account or issue an Athena session. The user selects a permanent, case-preserving
username there. A successful submission atomically creates a UUID `account_id`, the
immutable username, initial profile, and complete access matrix. Ordinary accounts start
with login enabled and no business, API Key, or Profit Sharing access. The UUID is the
internal identity used by sessions, authorization, API Keys, Wallet, and Profit Sharing;
the UI displays `@username` instead.

The Google entry selects the account realm before provider verification. The member entry
always resolves or creates an ordinary Pending persona, including for the configured
administrator email. The administrator entry accepts an unknown identity only when its
verified email matches `ATHENA_ADMIN_GOOGLE_EMAIL`; its submitted account is created with
`administrator=true`, login enabled, and no member modules, API Key, or Profit Sharing
entitlement. The same Google subject may therefore own two independent UUIDs and globally
unique usernames. No username confers a role. A Phantom account is member-only and keeps one
permanent canonical Solana address, with no merge, rebind, transfer, or recovery path.
Email comparison trims surrounding whitespace and ignores case, but does not normalize
Gmail dots or `+alias` values. When
authentication is enabled, missing OIDC settings or administrator email prevent the API
Server from listening. Setting `ATHENA_SERVER_DISABLE_AUTH=true` creates or reuses both
isolated development identities and does not require Google configuration. Requests from
the member application use `local-user` with maximum member grants; requests from the
administrator application use management-only `local-admin`. There is no disabled-auth
role selector. This mode is accepted only when the API Server listens on `localhost`,
`127.0.0.0/8`, or `::1`; reset all local state before switching back to OIDC. The local
Procfile defaults to `127.0.0.1`.

Every request that needs an identity carries exactly one
`X-Athena-Application-Realm: member|admin` header. The two browser applications add it
automatically. Private image GETs and EventSource connections, whose browser APIs cannot
set that header, use `athenaRealm=member|admin`; if a request supplies both transports they
must agree. Missing, duplicated, invalid, or conflicting values never fall back to a
default identity.

The direct client-secret variable is permitted only for local development. To exercise the
file-based path instead, leave it empty, write the secret to a separate `0600` file, and set
`ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE` to that file.

Phantom login needs no App ID, secret, callback, RPC endpoint, or new environment variable.
Install the Phantom desktop browser extension and use its injected Solana provider at
`http://localhost:4000`. Athena asks the extension to sign a server-generated five-minute
Sign-In With Solana message, verifies Ed25519 locally, and never requests a transaction,
network fee, balance, private key, or mnemonic. A Phantom address always creates an
ordinary account and cannot claim the administrator role. Using Google and Phantom creates
two independent accounts even when both belong to the same person.

### Start the stack

Start through the local helper script:

```bash
cd ui
nvm use
cd ..
make run
```

Run these commands in the same shell from the repository root so the helper inherits
the Node.js version selected by `ui/.nvmrc` (currently `24.14.1`). The UI package
accepts Node.js `>=24.14.1 <25`; do not change the global NVM default. On a new
machine, complete the Node.js setup in [Development Environment](development-environment.md)
before starting the stack.

The default local stack exposes:

- API server: `http://localhost:8080`
- UI dev server: `http://localhost:4000`
- Redis: `localhost:6379`
- PostgreSQL: `localhost:5432`

The Wallet gRPC process binds `127.0.0.1:8088` by default. The Procfile supplies
the same explicit development-only `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` to Wallet
and the API Server, and the API Server attaches it to every internal Wallet RPC.
If you override this value while running either process separately, use the same
token of at least 32 bytes for both processes. Missing or mismatched credentials
leave the standard Wallet health probe available but reject every business and
private-key RPC.

Use the CLI against the local API with:

```bash
export ATHENA_SERVER=127.0.0.1:8080
```

Open `http://localhost:4000` for interactive login. Vite proxies `/auth/*` to the API
Server, including Google callback, Phantom challenge/verification, shared registration,
and username availability. The callback URI must continue to use port `4000`; its
`http://localhost:4000` origin is also the trusted SIWS domain/URI. Athena never derives
either trust boundary from `Host` or forwarded headers.

With `ATHENA_SERVER_DISABLE_AUTH=true`, both local applications are immediately usable:

- Member: `http://localhost:4000/`
- Administrator: `http://localhost:4000/admin/`

They may remain open at the same time because each request selects its own development
identity. For direct API calls, include the matching header, for example:

```bash
curl -H 'X-Athena-Application-Realm: member' http://localhost:4000/api/v1/app/bootstrap
curl -H 'X-Athena-Application-Realm: admin' http://localhost:4000/api/v1/app/bootstrap
```

An unknown identity first lands on `/register`. Google registrations display verified
email; Phantom registrations display a copyable Solana address. After the user chooses an
available permanent username and account creation succeeds, the new account lands on
`/account/access`. It can use Profile, Appearance, Access, Help, and Logout, but starts no
business requests until the administrator grants a module or Profit Sharing access. API
Key management appears only when its independent entitlement is enabled.

## Local Data

Redis, PostgreSQL, and MinIO run in disposable Docker containers backed by the
fixed `athena-local-redis-data`, `athena-local-postgres-data`, and
`athena-local-minio-data` volumes. Ordinary `make stop` removes the containers
but preserves those volumes; `make run-reset` removes both containers and owned
data volumes.

The current schema uses UUID account IDs throughout account state, Profit Sharing, Wallet,
and related service boundaries. Before first use of this implementation, run
`make run-reset`, then `make run`. Reset removes accounts, administrator binding, profiles,
access grants, API Keys, sessions, OAuth transactions, Phantom challenges, shared
registration tickets, Profit Sharing references, and avatars; there is no compatibility
import. The current Wallet schema also replaces the former system-wallet/seed model with
owner-only EVM and Solana wallets, so an existing local volume must be reset before its
next startup; the application never performs that destructive reset automatically.

Supported variables:

- `ATHENA_SERVER_DISABLE_AUTH` default in the Procfile: `false`; set `true` only for the
  loopback dual-identity development mode described above
- `ATHENA_REDIS_PORT` default: `6379`
- `ATHENA_REDIS_IMAGE_TAG` default: `8.2.3`
- `ATHENA_POSTGRES_PORT` default: `5432`
- `ATHENA_POSTGRES_IMAGE_TAG` default: `16`
- `ATHENA_POSTGRES_INIT_DIR` default: `hack/postgres/init`
- `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` default in the Procfile:
  `athena-local-wallet-internal-auth-token-2026`

Etherscan API keys are documented separately in
[Etherscan Configuration](../etherscan-configuration.md).

## UI Changes

Run the UI only:

```bash
cd ui
nvm use
yarn start
```

The dev server listens on port `4000`. Override the host with `ATHENA_YARN_HOST`.

### Browser acceptance

Use the repository entry point for repeatable browser checks. It runs entirely in WSL,
uses the project's local Yarn and Playwright packages, and never installs or downloads
dependencies:

```bash
make ui-acceptance
```

The default `isolated` mode builds the UI once, then runs the complete maintained
`ui-fixtures` and `live` projects against fresh root-path and `/athena` harnesses. The run
owns one temporary PostgreSQL container; each harness uses a fresh database. Browser tests
use Playwright's paired Chromium. This
mode does not use, stop, or reset a development stack started with `make run`.

To smoke-test the real development UI with system Chrome, prepare or reuse its environment as described below, then run:

```bash
make ui-acceptance UI_ACCEPTANCE_MODE=smoke
```

Smoke mode defaults to `http://localhost:4000`. Set `UI_ACCEPTANCE_BASE_URL` to another
explicit HTTP loopback URL. It checks the member and administrator bootstrap and application
shells in separate browser contexts. Anonymous or login-page results are valid session
states; the smoke does not perform Google or Phantom login and does not submit business
changes. The smoke command itself never starts or stops `make run`; the agent performing acceptance is responsible for preparing the target environment.

### Prepare the development environment for acceptance

Follow the [project acceptance rule](../../AGENTS.md#本地验收环境准备与完成标准). Required local acceptance includes preparing its environment; connection refusal alone is not a final blocker.

1. Confirm the target repository/worktree and URL. Inspect the process command, working directory and `.run/athena-local-runtime/supervisor.state` where present. Reuse a healthy matching stack. If another worktree or an unrelated process owns the required port, preserve it and investigate the documented configuration; do not kill it or silently test the wrong checkout.
2. If the target stack is absent, select Node and start it **from the target repository**:

   ```bash
   cd ui
   nvm use
   cd ..
   make run
   ```

   Use a persistent terminal/session and capture its output to a task-specific log under `.tmp/`. Keep the session alive after acceptance and record its identifier. Do not change the global Node default. Resolve prerequisite problems using the documented setup within the authorized scope; the smoke tool's no-install policy does not prohibit the agent from preparing the environment.
3. Observe startup output and process health. Check `/` and `/admin/`, then `/api/v1/app/bootstrap` separately with `X-Athena-Application-Realm: member` and `admin`. Verify the expected realm and session response; anonymous/login states are valid. A listening port or HTTP 200 alone is insufficient. Run the smoke command above and inspect its exit result and native report to verify the actual application shells.
4. Preserve startup and smoke failures, inspect the relevant logs, resolve environment issues within scope and rerun the affected checks. Do not automatically reset data, delete volumes, change product behavior or repeatedly retry an unchanged failure. An unresolved external dependency, credential, permission or user decision must be reported with attempted actions and remaining verification; passing isolated tests does not close a required real smoke check.
5. Leave the development service running after acceptance, whether it passed or failed. Report the URL, repository/worktree, session or process identifier, log location and smoke result. Explain that `make stop` from that same repository stops the stack and removes its containers while retaining data volumes. Never stop a pre-existing service just to tidy up acceptance.

If the user requests only a read-only inspection, prerequisite check or no service startup, honor that boundary. A rules-only change or isolated-only test does not require starting a development stack. Required real acceptance that remains unverified prevents claiming full task completion or sending its completion notification.

### Prerequisite checks

Use the read-only prerequisite check before either mode:

```bash
make ui-acceptance UI_ACCEPTANCE_CHECK_ONLY=1
make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_CHECK_ONLY=1
```

Select the project Node.js version in the shell before running the command:

```bash
cd ui
nvm use
cd ..
make ui-acceptance
```

The command selects WSL Node from `ATHENA_UI_ACCEPTANCE_NODE`, then `PATH` (including the
version selected by `nvm use`), then the NVM default installation. It does not use a Windows
Node runtime. Isolated mode additionally
requires Docker, Go, the project dependencies, the paired Playwright Chromium, and the
configured PostgreSQL image. Go is selected from `ATHENA_UI_ACCEPTANCE_GO`, then `PATH`,
then `/usr/local/go/bin/go`. Smoke mode requires system Chrome from `ATHENA_CHROME_PATH` or
`/usr/bin/google-chrome`. Relative `ATHENA_CHROME_PATH` values are resolved against the
command's starting directory; prerequisite checks and Playwright use the same absolute
path, including paths containing spaces. A missing prerequisite is reported as a blocker without changing
the machine.

The isolated command sets `GOTOOLCHAIN=local`, `GOPROXY=off`, and `GONOPROXY=none` for its Go processes.
Prepare the required Go toolchain and module cache separately before the first run;
missing cached modules fail the harness stage without downloading dependencies.

Dependency preparation is an explicit development setup action, separate from acceptance:

```bash
cd ui
nvm install
nvm use
yarn install
yarn playwright:install chromium
```

`yarn install` installs the versions locked by the UI project.
`yarn playwright:install chromium` downloads the Playwright-paired Chromium used by
isolated mode. System Chrome for smoke mode is supplied by the workstation and is not
installed by the repository command.

Each real run writes native Playwright JSON and HTML reports, traces, screenshots,
attachments, a machine-readable summary, and a Chinese summary below
`.tmp/athena-ui-acceptance/<run-id>/`. Fixture results cover simulated API responses; live
results cover the isolated ATHENA components and temporary PostgreSQL while chain, profile,
and Telegram dependencies remain local substitutes; smoke results cover only the existing
development bootstrap and application shells. Interactive inspection with an available
built-in browser is useful during development, but it is not a system-Chrome smoke run or a
repeatable Playwright regression.

### Browser acceptance readiness and interruption

Smoke observes the application's own bootstrap retries. It allows up to 15 seconds from
navigation for a valid bootstrap and the corresponding member or administrator shell to
be ready; transient HTTP errors or malformed JSON do not fail an otherwise recovered
application. Persistent failure or an invalid final session still fails. Each test attaches
`bootstrap-attempts` diagnostics with HTTP status, realm, parsing errors and session status.
The command does not initiate extra bootstrap requests, reload pages, or retry failed tests.
Role/realm mismatches, uncaught page errors, required resource failures, and attempted
identity-provider flows remain failures. The latest bootstrap and shell are checked again
after required resources settle.

`SIGHUP` (terminal hangup), `SIGINT`, and `SIGTERM` enter the same bounded cleanup path and
exit with codes 129, 130, and 143 respectively. Cleanup finishes before exit, including
stopping this run's process groups, harness and PostgreSQL container and releasing its lock.
Repeated signals do not launch another cleanup. Cleanup errors remain in the run report
alongside the original failure. Smoke never stops the development environment.

`SIGKILL`, a machine crash, or an interrupted Docker daemon can leave resources behind.
There is no automatic stale-lock reclamation. Recover manually in this order:

1. Read `.athena-ui-acceptance.lock/owner.json` under the path returned by
   `git rev-parse --git-common-dir` (or under `.tmp/athena-ui-acceptance/` outside a Git
   repository). Record its `pid` and `runId`. Locate the matching
   `.tmp/athena-ui-acceptance/<runId>/` in the worktree that started the run.
2. Check the owner PID's start time, command and working directory; PID existence alone
   does not establish identity because PIDs can be reused. If the original runner is
   still alive, send it `SIGTERM` and wait for its normal cleanup.
3. If the runner is gone, inspect the run's logs and process table to identify its surviving
   harness and child process groups. Write `stop` into each existing `root/harness/` or
   `athena/harness/` directory and allow graceful shutdown. Only send `SIGTERM`, followed
   by `SIGKILL` when necessary, to process groups whose ownership you have confirmed from
   their command, working directory, and harness environment/logs. Do not kill by shared
   service port or a partial process-name match; leave ambiguous processes untouched.
4. Find containers using the exact label `io.athena.ui-acceptance.run=<runId>`. Inspect
   each container's full ID and label before removing that ID with `docker rm --force -v`.
   This removes its anonymous database volume; never remove shared development volumes.
5. After confirming the original runner and its owned resources are gone, re-read
   `owner.json` and confirm its `runId` is unchanged. Remove that owner file and the now-empty
   lock directory. Preserve reports and logs, then rerun acceptance. If ownership cannot
   be established, keep the lock and investigate instead of clearing it speculatively.

## Backend Changes

When `make run` is running, restart a process with `goreman`:

```bash
goreman run restart api-server
```

Process names are listed in the root `Procfile`.

## Production-Like Local Run

For a Docker Compose run that mirrors production deployment more closely:

```bash
make prod-build-local
make prod-start-local
```

Production-like Compose requires `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` in its env
file. `make prod-reset-secrets` generates a new 40-character value; both the
Wallet and API Server containers receive it explicitly, while unrelated
containers receive an empty override.

Stop it with:

```bash
make prod-stop-local
```

### Trader Sync configuration

The API process requires `ATHENA_TRADER_SYNC_HTTP_URL`,
`ATHENA_TRADER_SYNC_WSS_URL`, `ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY`, and
`ATHENA_URL`. Supply the HTTP/WSS endpoints you have verified for Polygon 137;
missing settings fail startup instead of selecting another provider. Keep the
cursor key stable across restarts. `ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES`
optionally sets the positive source-job limit (default 100); it is not a target
subscription quota or a throughput claim.

`ATHENA_TRADER_SYNC_PROXY_URL` controls only the API's Trader Sync source,
Gamma/Profile, and directory requests. A present empty string explicitly means
direct access. The local Procfile runs `hack/trader-sync-local.sh` after Goreman
has loaded `.env`: inherited values, including empty values, take priority over
`.env`; only a still-unset key on WSL gets the default gateway port 10809. This
helper does not change the existing Token proxy policy. Trader Sync transports
ignore global proxy variables, including values loaded back from `.env`.

Production Compose passes these keys through its selected service `env_file`
without an `environment` interpolation default that would shadow that file.
The WSL helper is not used in containers. The notification process consumes
`ATHENA_URL` to configure the shared summary source before starting its existing
dispatcher. Trader Sync proxy settings do not configure Telegram. Explicit
stopped-sender recovery remains independent of Trader Sync source settings.
