# Running Athena Locally

Athena local development no longer requires a cluster. The local stack runs Athena processes with Redis and PostgreSQL provided locally or through Docker.

## Prerequisites

Complete [Development Environment](development-environment.md).
Use Go `1.27.1` as required by `go.mod` for local builds and acceptance. Follow
[Install Go on Linux（WSL）](install-go.md) when preparing a WSL machine.

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
`127.0.0.0/8`, or `::1`. Use an instance without development identities when switching
back to OIDC; stop and explicitly reset only the old instance if discarding its data.
The local runtime defaults to `127.0.0.1`.

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

当前 `make run` 已实现十一应用统一图和[板块访问开关](../requirements/development-runtime/business-access-control.md)。关闭访问只拦截新的用户业务请求，程序、后台任务和通知继续运行；首次缺行默认 CLOSED，之后保留管理员设置，重启不改变开关。Token 接入延期，BSC、Sports 和 Markets 不在当前图。实际根路径与 `/athena` 前缀验收见[全栈记录](../testing/full-stack-access-acceptance.md)。

Start the explicitly selected full stack from this checkout:

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

The full stack contains Wallet, Notification, Etherscan Manager, API Server, UI,
Trader Sync, Solana Discovery, Market Radar, Managed OO, Profit Sharing, and Worm
Trading. It defaults to the `full-stack` instance; use the same explicit `INSTANCE`
and, when applicable, `ENV_FILE` for run, status, and stop. Managed mode creates only
the selected owners' databases. Infrastructure and all application listeners use
loopback addresses; inspect the assigned addresses, readiness, failures, schema results,
Gateway health, logs, and exact stop command with `make runtime-status INSTANCE=full-stack`.
The current default ports are UI `4000`, API `8080`, Notification `8086`, Wallet `8088`,
Manager `8100`, Trader Sync `8122`, Solana `8112`, Market Radar `8092`, Managed OO
`8106`, Profit Sharing `8108`, and Worm Trading `8090`.

For Trader Sync development, select only that service:

```bash
make build-service SERVICE=trader-sync
make run-service SERVICE=trader-sync INSTANCE=ts-dev
# In another terminal, from the same checkout:
make seed-service SERVICE=trader-sync INSTANCE=ts-dev
make runtime-status INSTANCE=ts-dev
make stop-instance INSTANCE=ts-dev
```

This starts only Trader Sync and its own persistent PostgreSQL. Seed prints member and
administrator UUIDs for authorized RPCs; repeated seed preserves existing identities
and revoked grants. To select an integration set, use
`make run-services SERVICES='trader-sync api-server ui' INSTANCE=ts-integration`.
API and Notification do not implicitly start Trader Sync. UI alone requires no database.
The runtime requires Linux/WSL and Bash 5.1+; the selected Go service does not require Node.

Wallet and API use the same development `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` when
selected in the full stack. If overridden, both must receive the same token of at
least 32 bytes. Health remains available when business credentials are rejected.

Selections are positive: `run-service` and `run-services` do not add unselected
applications. Managed mode prepares only the selected schema owners. `DB_MODE=external`
performs read-only verification of the selected databases and never creates, migrates,
seeds, stops, or resets a borrowed database. Direct API routes follow
`ATHENA_SERVER_ROOTPATH`; HTML, public assets, callbacks, and the Vite proxy follow
`ATHENA_SERVER_BASEHREF`. A successful `runtime-status` command means state was read;
check `SelectedReady` or `FullStackReady` rather than treating exit 0 as readiness.

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
`/account/access`. It can use Profile, Access, Help, and Logout, but starts no
business requests until the administrator grants a module or Profit Sharing access. API
Key management appears only when its independent entitlement is enabled.

## Local Data

Each managed instance has persistent volumes named from the canonical checkout path
and instance name. `.run/instances/<instance>/state.json` records actual resources and
addresses. `make stop-instance INSTANCE=ts-dev` and terminal Ctrl+C stop only owned
processes and containers, retaining volumes, credentials, fixture identities and logs.
`make reset-instance INSTANCE=ts-dev` deletes that instance's owned data and requires
it to be stopped first. `make stop` and `make run-reset` select `full-stack` through the
same ownership engine. A new checkout does not import or delete old fixed-name volumes.

There is no mandatory first-run reset. Schema preparation is explicit in the runtime:
managed instances run the independent `athena-account-state-migrate up` and `verify`
before business startup; the API, Trader Sync and Notification binaries only verify.
An incompatible existing database fails clearly; do not reset unrelated instances.
Reset intentionally removes that instance's accounts, grants, subscriptions, activities,
notification attempts, sessions and avatars. Full-stack reset also removes its module
data. It does not revoke remote credentials or resolve unknown remote operations.

To borrow an existing account-state database, use `DB_MODE=external` with an explicit
`ENV_FILE` containing `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` and the appropriate internal
credentials. External mode performs read-only schema verification and never creates,
migrates, seeds, stops or deletes the borrowed database. External reset is rejected.
Borrowing another managed instance's database is allowed; stopping its owner affects
all borrowers. Full-stack mode accepts only managed databases.

Configuration files are parsed as data. Exported values override the file, including
explicit empty values. Independent processes receive their own configuration allowlist.
For port variables, TLS, sender recovery, and ownership details see
[Local Runtime Orchestration](../design/development-runtime/local-runtime-orchestration.md).

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

1. Confirm the target repository/worktree and URL. Inspect `make runtime-status` and `.run/instances/<instance>/state.json`, together with the process command and working directory. Use `.run/athena-local-runtime/supervisor.state` only as evidence for a legacy environment where that file still exists. Reuse a healthy matching stack. If another worktree or an unrelated process owns the required port, preserve it and investigate the documented configuration; do not kill it or silently test the wrong checkout.
2. If the target stack is absent, select Node and start it **from the target repository**:

   ```bash
   cd ui
   nvm use
   cd ..
   make run
   ```

   Use a persistent terminal/session for the duration of development and acceptance, capture its output to a task-specific log under `.tmp/`, and record its identifier and resource ownership. Stop task-created temporary environments when the task ends according to the procedure below. Do not change the global Node default. Resolve prerequisite problems using the documented setup within the authorized scope; the smoke tool's no-install policy does not prohibit the agent from preparing the environment.
3. Observe startup output and process health. Check `/` and `/admin/`, then `/api/v1/app/bootstrap` separately with `X-Athena-Application-Realm: member` and `admin`. Verify the expected realm and session response; anonymous/login states are valid. A listening port or HTTP 200 alone is insufficient. Run the smoke command above and inspect its exit result and native report to verify the actual application shells.
4. Preserve startup and smoke failures, inspect the relevant logs, resolve environment issues within scope and rerun the affected checks. Do not automatically reset data, delete volumes, change product behavior or repeatedly retry an unchanged failure. An unresolved external dependency, credential, permission or user decision must be reported with attempted actions and remaining verification; passing isolated tests does not close a required real smoke check.
5. At task completion, cancellation, pause, or an end caused by failure/blockers, save evidence and follow [task shutdown](#task-shutdown-and-retained-environments). The accepting agent performs this cleanup; the smoke command does not own the development stack. Report the actual acceptance and cleanup results separately.

If the user requests only a read-only inspection, prerequisite check or no service startup, honor that boundary. A rules-only change or isolated-only test does not require starting a development stack. Required real acceptance that remains unverified prevents claiming the AI delivery stage complete. Task-result notification behavior remains governed by the fixed-content, elapsed-time and deduplication rules in the root `AGENTS.md`; a notification does not prove acceptance or final delivery.

### Task shutdown and retained environments

The [project rule](../../AGENTS.md#本地验收环境准备与完成标准) defaults to stopping task-created temporary environments. Keep them running while development, debugging or acceptance is in progress; finishing one conversation reply does not end an active task. When the task ends, apply the following ownership rules:

| Environment | Action at task end |
| --- | --- |
| Temporary development or branch acceptance stack started for this task | Stop its services and owned containers; retain development databases, volumes, logs and reports. |
| Preview server or test substitute started for this task | Stop it through its own recorded shutdown entrypoint; preserve mockups and evidence. A helper not managed by `make stop` still needs cleanup. |
| User-started environment, environment used by another active task, or borrowed infrastructure | Leave it unchanged; stop only this task's own consumers. |
| Environment explicitly requested by the user for continued inspection, debugging or use as the main development environment | Retain that environment and its required helpers; stop other task-created temporary instances. A port number alone does not designate a permanent main environment. |

Before stopping, verify the checkout, instance/profile and recorded process/container identities. Run `make stop INSTANCE=<name>` for a managed full stack, `make stop-instance INSTANCE=<name>` for a selected-service instance, or the matching profile/tool stop command from its owning checkout. For a legacy environment, use its recorded owner and shutdown entrypoint. Stop dependent services before separately launched test substitutes. Never use `make run-reset`, delete development volumes, stop borrowed infrastructure, or kill an unidentified process to tidy up a task.

Confirm owned processes exited, their ports were released and their containers stopped. Save and investigate cleanup failures, and report any remaining resources instead of claiming shutdown. Failure evidence does not require a live server: after saving it, task-created temporary resources are stopped unless the user explicitly requested retaining the debugging environment.

The handoff lists stopped and retained environments. For retained items, record the user's request or existing ownership, URL, checkout, instance/profile, session or PID, logs, acceptance result and exact stop command. Keeping data and evidence does not imply keeping the service running. The agent must invoke shutdown; the runtime does not detect a completed conversation task.

### Restore an environment for post-delivery human review

The AI delivery step stops task-owned temporary environments as described above. Human review may therefore start later from a stopped state. The concrete `review-guide.md` for that task and round must identify the exact repository/worktree and immutable version, include a version check, and fill in the applicable start command, `INSTANCE` or profile, log location, URL, identity, test data, readiness checks and stop command. Do not leave generic placeholders in a delivered guide.

The user may run those instructions or ask the AI to restore the environment. Such a request authorizes the necessary startup, readiness checks and environment troubleshooting for the specified review target; it does not authorize resetting data, switching over unrelated worktrees or changing product behavior. Preserve a version, identity or data mismatch as a blocked review prerequisite before making changes that could erase the observed state.

Record who started each resource. User-started review resources remain user-owned. When the AI prepares an environment for continued human inspection, retain only the specified environment and report its URL, checkout, instance/profile, session or PID, logs and exact stop command; other task-owned resources still follow normal shutdown. If the user asks the AI to end the review environment, stop and verify it through the same ownership procedure above.

Documentation, skill and other review tasks that do not exercise a runtime must say that no environment is required. They do not start a stack merely to populate the human-review materials.

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
Prepare Go `1.27.1` and the module cache separately before the first run;
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

A running instance uses immutable binaries. After changing a selected service, stop
and restart that instance to rebuild it and reuse its persistent database:

```bash
make stop-instance INSTANCE=ts-dev
make run-service SERVICE=trader-sync INSTANCE=ts-dev
```

For a full stack, use the same `INSTANCE` with `make stop` and `make run`. The runtime
has no implicit hot restart of all dependent services. Use a small selected service set
or [IDE debugging](debugging-locally.md) for frequent iteration; do not use Goreman against
the descriptive Procfile.

## Production-Like Local Run

The Athena and MinIO Docker build stages use the pinned Go `1.27.1` image.
The default MinIO Server and mc tags carry a `-go1.27.1` suffix so a build
cannot reuse images compiled with the previous Go toolchain.

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

The independent Trader Sync process requires `ATHENA_TRADER_SYNC_HTTP_URL`,
`ATHENA_TRADER_SYNC_WSS_URL`, `ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY` (or its `_FILE`
variant), and `ATHENA_URL`. Supply the verified Polygon137 endpoints from the
[provider document](../requirements/polymarket-copy-trading/hosted-polygon-rpc-providers.md#已取得的开发候选端点).
Keep the cursor key stable across restarts. `ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES`
optionally sets the positive source-job limit (default100), not a subscription quota
or a throughput claim. The API receives only the internal gRPC client configuration;
it does not require provider endpoints or run a Collector.

API and Trader Sync share an internal token of at least32 non-whitespace bytes, distinct
from the cursor key. Managed runtime generates and reuses it when absent; external mode
requires it explicitly. Direct values and corresponding `_FILE` values are mutually
exclusive. All three account-state consumers use `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`,
with independent pools and no legacy alias. Their business startup never runs migrations.

`ATHENA_TRADER_SYNC_PROXY_URL` controls TS HTTP/WSS, Gamma/Profile and directory requests.
An explicitly empty string means direct access. After dotenv parsing and exported-value
merging, only a still-unset key on WSL gets gateway port10809. TS does not inherit global
proxy or Token proxy settings. Restart Trader Sync after changing the HTTP/WSS provider
pair; interruptions remain visible and missing historical trades are not backfilled.

The binary defaults to TLS. The local runtime explicitly selects `loopback-insecure`
only when transport is unset. Health can run without loading business configuration:

```bash
go run ./cmd/athena-trader-sync health --target 127.0.0.1:8122 --transport loopback-insecure
```

Production uses the [independent image, TLS files and schema maintenance protocol](../../deploy/trader-sync/README.md),
with per-process configuration rather than a shared service `env_file`. Notification
owns its Bot and dispatch lifecycle; stopped-sender recovery is independent of Trader Sync.
Detailed validation and limits are recorded in the
[independent service acceptance report](../testing/trader-sync-independent-service-acceptance.md).
<a id="solana-discovery-local"></a>
## Solana 发现的局部运行

旧 Solana profile 的暂停是原现场事实，不限制本轮新实例：Solana Discovery 已纳入当前十一应用默认图，也可以由统一 managed 实例运行器单独选择。启动扫描会访问实际主网 RPC；请为任务使用独立 `INSTANCE`，并以同一 owner 的 status/stop 收尾。访问设置 CLOSED 只限制新的用户查询，不停止后台扫描。

独立入口不构建 API 或启动其他业务：

```bash
make solana-discovery-build
make solana-discovery-run INSTANCE=solana-discovery
# 另一个终端，同一 checkout：
make runtime-status INSTANCE=solana-discovery
make solana-discovery-stop INSTANCE=solana-discovery
```

两个 Make 别名分别委托统一的 `run-service SERVICE=solana-discovery` 与 `stop-instance`，默认实例名为 `solana-discovery`。managed 模式创建本实例 PostgreSQL，先准备账户 schema，再在同一明确 DSN 中执行 Solana `schema up`／`verify`；业务 main 只验证。external 模式要求显式 DSN，只读 verify，不迁移、创建或停止借用数据库。配置使用 `ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN`、`ATHENA_SOLANA_DISCOVERY_RPC_URL` 和持久内部 token。

旧 `hack/solana-local.sh` 的 `solana-discovery`／`solana-preview` 状态、进程和数据不迁移到新 owner。只需停止旧现场时，仍从其原 checkout 使用 `bash hack/solana-local.sh stop solana-discovery` 或 `bash hack/solana-local.sh stop solana-preview`；不要用旧脚本停止新实例，也不要让新入口接管旧状态。

如另一 checkout 已在运行，可用新的 managed 实例预览 Solana + API + UI；它会创建自己的账户数据库、Redis 和 MinIO，不要指向另一实例的账户库：

保留的原 `athena_solana_preview` 来自 Solana 源分支，账户 schema 与 `rf4` 不兼容：两者的账户迁移版本 `000002` 分别代表 Solana 授权和 Trader Sync runtime control。该库、候选、游标及补全队列保持不动；不得用 `rf4` 的自动 `up` 或 reset 尝试恢复。以后明确要求恢复原预览时，使用保留的原 worktree 和原分支版本管理原库。若需要将旧数据转入 `rf4`，须单独明确数据迁移范围；本次集成不提供历史 schema 兼容路径。

```bash
# 先按 ui/.nvmrc 选择 Node 24.14.1
make run-services SERVICES='solana-discovery api-server ui' INSTANCE=solana-preview
make runtime-status INSTANCE=solana-preview
# 另一个终端，同一 checkout：
make stop-instance INSTANCE=solana-preview
```

局部选择不启动 Trader Sync、Notification、Wallet 或其他业务；未选服务的调用按真实状态返回不可用。地址和端口使用统一注册表变量，可从 runtime status 读取。原 `ATHENA_RUN_PROFILE=solana-preview`／`solana-discovery` 的 start 已封闭，避免生成第二套 owner；保留旧现场只使用上文原脚本 stop。

Solana预览为公共节点采样设置每范围1 slot、并发1、每秒请求预算1；有积压时页面照实展示。服务常规默认每范围4 slots；更高容量节点可通过自身配置调整。

## Standalone Worm Trading

Worm Trading builds directly from `cmd/athena-worm-trading` and runs independently
of API, UI and other business services (SDS-R3/R5). Supply a task-specific `ENV_FILE`
with a stable `ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY`, an internal token of
at least 32 bytes in `ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN`, the configured
`ATHENA_WALLET_SERVER_ADDRESS` and matching
`ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN`, and
`ATHENA_WORM_TRADING_SOLANA_RPC_URL`. Wallet is an explicitly borrowed signer;
this command does not launch it. Solana readiness checks its mainnet genesis hash
and slot. The official Worm provider is fixed and does not participate in startup
readiness; its failures affect calls that require the provider, while persisted
combinations and history remain readable.

```bash
make build-service SERVICE=worm-trading
make run-service SERVICE=worm-trading INSTANCE=worm-retirement DB_MODE=managed ENV_FILE=/absolute/path/trading.env
make runtime-status INSTANCE=worm-retirement
make stop-instance INSTANCE=worm-retirement
```

Managed mode prepares only the instance account database and `worm_trading`, using
the independent account migration owner and `athena-worm-trading-migrate up` then
`verify`. Trading startup only opens and verifies both databases. It never migrates
or chooses a fallback database. `verify` uses a read-only repeatable-read transaction:
the complete effective embedded migration version set must match and all required
Trading relations must exist. This is a version/relation check, not a full catalog
fingerprint. An empty, older or future schema is rejected without creating goose
metadata. To prepare an explicitly selected database outside managed runtime:

```bash
ATHENA_WORM_TRADING_POSTGRES_DSN='postgres://user:password@localhost/worm_trading' \
  go run ./cmd/athena-worm-trading-migrate up --timeout=120s
ATHENA_WORM_TRADING_POSTGRES_DSN='postgres://user:password@localhost/worm_trading' \
  go run ./cmd/athena-worm-trading-migrate verify --timeout=120s
```

For `DB_MODE=external`, provide both `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` and
`ATHENA_WORM_TRADING_POSTGRES_DSN`. Only verification runs; no borrowed schema or
container is created, migrated or stopped. Use a distinct instance when switching
DB mode or database identity. For isolated Trading work, external mode can borrow
the full-stack account database while using its own Trading database and Wallet
signer; record the owner and borrower separately. The current default graph already
includes Worm Trading and injects its selected address into API automatically, so
manual address wiring is only needed for a separate instance.

The runtime forwards explicit Trading configuration keys only, including RPC
attempt/balance/rate settings, `ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT`
(default 5s), `ATHENA_WORM_TRADING_CATALOG_BUDGET` (default 45s, positive and at
least the attempt timeout), position budget/concurrency and ordinary proxy
variables. `ATHENA_WORM_TRADING_LISTEN_ADDRESS` defaults to `127.0.0.1`, port to
8090. When Trading and API are selected together, API receives the selected Trading
instance address. Invalid budgets, incompatible schemas and missing keys/tokens
fail startup. Shutdown drains gRPC for at most 10 seconds before cancelling
transport, then stops existing workers and closes the command-owned Trading and
account pools. Transport shutdown does not reverse an external request.

For `GetOrderEventCatalog`, `CreateMarketCombination` and
`UpdateMarketCombination`, the Trading catalog budget starts at RPC entry and
covers the current account-access read, catalog work and final storage under
the same deadline. The API adds a fixed 60-second transport deadline for these
three calls only; an earlier caller deadline or cancellation takes precedence.
Even when the service catalog budget is configured above 60 seconds, these HTTP
requests remain capped by the API transport deadline. Writes are not retried
automatically; this does not change Open/Close or other RPC budgets.

`make build-service-image SERVICE=worm-trading` builds the minimal
`deploy/worm-trading/Dockerfile`, containing only the service and its migration
binary. `WORM_TRADING_IMAGE` defaults to `athena-worm-trading:local`; production
build and local/remote delivery use that image for both Compose entries. The
migration tool is under the `tools` profile. Production schema preparation calls
its `up`/`verify`; account maintenance includes Trading as an account consumer.
`make stop-instance` retains this instance's volumes and evidence and leaves the
configured Wallet/Solana and external databases alone.

## Standalone service schema preparation

Wallet, Etherscan Manager, Market Radar, Managed OO and Profit Sharing each have
an independent `cmd/athena-<service>` main. Build them without the aggregate
`cmd/main.go` (SDS-R3):

```bash
go build ./cmd/athena-wallet ./cmd/athena-etherscan-manager ./cmd/athena-market-radar ./cmd/athena-managed-oo ./cmd/athena-profit-sharing ./cmd/athena-solana-discovery
```

Wallet, Managed OO, Profit Sharing and Solana expose `schema up` and `schema
verify`, each accepting `--timeout=120s`. These commands use only the database
configuration and exit without requiring business credentials, starting RPC or
contacting upstream business services.

| Binary | Explicit database configuration | Owned schema |
| --- | --- | --- |
| `athena-wallet` | `ATHENA_WALLET_POSTGRES_DSN` | Wallet database, `public` |
| `athena-managed-oo` | `ATHENA_MANAGED_OO_POSTGRES_DSN` | Managed OO database, `public` |
| `athena-profit-sharing` | `ATHENA_PROFIT_SHARING_POSTGRES_DSN` | Profit Sharing database, `public` |
| `athena-solana-discovery` | `ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN`, falling back to `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` | `solana_discovery`, in the account database |

The first three retain the existing PostgreSQL connection defaults when their
explicit DSN is unset. Solana requires one of its two explicit DSNs. Prepare
only the selected service's database, for example:

```bash
go run ./cmd/athena-wallet schema up --timeout=120s
go run ./cmd/athena-wallet schema verify --timeout=120s
go run ./cmd/athena-wallet --address=127.0.0.1 --port=8088
```

Run `up` only when this deployment owns schema preparation. Borrowed databases
use `verify` only. Normal startup now always connects and verifies; it does not
migrate, even with `ATHENA_POSTGRES_AUTO_MIGRATE=true`. Verification uses a
read-only transaction and rejects missing relations/columns and mismatched
effective Goose migration versions without creating a metadata table. Solana
reuses its existing idempotent SQL migrations and verifies only its own required
relations/columns; it does not introduce Goose metadata in `public`.

For Solana's shared database, prepare account `up` → account `verify` → Solana
`schema up` → Solana `schema verify` → account `verify`, with the existing account
migration tool. Runtime verifies both the account contract and Solana schema on
its one owned connection pool before starting the scanner or RPC listener.

The five new entry points and Solana handle INT/TERM during database connection
and serving. They stop accepting transport connections, cancel/join background
work, drain RPC and close owned resources within one 30-second shutdown budget.
A blocked drain or cleanup forces transport closure and exits with a shutdown
error. This does not change the persisted business access setting or stop any
external Gateway. Existing Wallet and Solana internal credentials remain
required for service startup; Manager retains its API key, Gateway address and
Gateway authentication configuration. Use each binary's `--help` for its own
business configuration.
