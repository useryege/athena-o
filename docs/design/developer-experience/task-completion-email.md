# Task Completion Email

> 设计状态：已实现

## Scope

Task Completion Email owns the explicit developer command that sends a short
plain-text notification after an Athena task has finished. The caller provides
the subject and body for each invocation, while the implementation fixes the
Tencent Exmail SMTP endpoint and the destination mailbox.

This capability does not run automatically after other Make targets, determine
whether a task succeeded, schedule notifications, or provide a general-purpose
email client. HTML, attachments, CC, BCC, dynamic recipients, and arbitrary
SMTP servers are outside its boundary.

The agent's notification policy is defined in
[AGENTS.md](../../../AGENTS.md#task-result-email): any task with more than ten
minutes of cumulative execution receives one result notification when execution
ends, regardless of Plan mode or success. The agent tracks time, excludes user
response waits and pauses, and reports the actual outcome. The command itself
does not measure task duration or enforce this policy.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Command implementation | [tools/task-completion-email/main.go](../../../tools/task-completion-email/main.go) | `main`, `run`, `loadEnvironment`, `readConfig`, `buildMessage`, `sendWithRetry`, `sendAttempt` |
| Developer entry point | [Makefile](../../../Makefile) | `notify-task-complete` |
| Operator interface | [docs/operator-manual/makefile-commands.md](../../operator-manual/makefile-commands.md) | `make notify-task-complete` |

## Architecture

The root Make target starts the Go command without interpolating the subject,
body, or SMTP credentials into a shell command. The command reads invocation
content and SMTP credentials from its environment, optionally supplements
missing settings from one env file, validates the complete input, constructs a
single RFC 5322/MIME message, and submits it directly to Tencent Exmail over an
implicit TLS connection.

The delivery boundary is intentionally fixed in code:

- SMTP endpoint: `smtp.exmail.qq.com:465`.
- Authentication: SMTP PLAIN over TLS.
- Recipient: `2687665142@qq.com`.
- Message format: UTF-8 `text/plain`.

No Athena service, database, queue, or runtime container participates in the
flow. The process owns only the current invocation and exits after delivery or
terminal failure.

## Runtime Flow

1. The caller runs `make notify-task-complete` and supplies
   `TASK_NOTIFICATION_SUBJECT` and `TASK_NOTIFICATION_BODY`.
2. The command selects `.env` by default, or the path named by
   `TASK_NOTIFICATION_ENV_FILE`. Values already present in the process
   environment take precedence over values from the file. An absent default
   `.env` is allowed so a fully configured process environment can be used;
   failure to load an explicitly selected file is an error.
3. It requires the SMTP username, SMTP client-specific password, subject, and
   body, and rejects subjects containing newline characters before any network
   attempt.
4. It generates one `Message-ID` and serializes one UTF-8 plain-text message for
   the fixed recipient.
5. For each delivery attempt, it creates a new connection with a ten-second
   timeout, negotiates implicit TLS 1.2 or newer, authenticates, and submits the
   message.
6. A retryable SMTP or network failure starts a fresh connection. There are at
   most three total attempts, with one second before the second attempt and two
   seconds before the third.
7. Once the SMTP server accepts the message data, delivery is successful. A
   later `QUIT` failure does not trigger another send. The process prints a
   success indication and exits with status zero.

The flow is synchronous and single-threaded. It creates no background process
and does not modify the task whose completion is being reported.

## State / Data

The command has no durable state. Credentials, subject, body, serialized
message, attempt count, and generated `Message-ID` exist only for the lifetime
of the process. The same serialized message and `Message-ID` are reused across
all attempts in one invocation so a recipient or mail system can recognize
retry duplicates.

SMTP acceptance is the commit point visible to this process. Acceptance means
Tencent Exmail has taken responsibility for the message; the tool does not
track downstream mailbox delivery or read receipts.

## Configuration

| Setting | Default | Behavior |
| --- | --- | --- |
| `ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_USERNAME` | None | Required Tencent Exmail account used for SMTP authentication and the From address. |
| `ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_PASSWORD` | None | Required Tencent Exmail client-specific password used only for SMTP authentication. |
| `TASK_NOTIFICATION_SUBJECT` | None | Required per-invocation mail subject; newline characters are invalid. |
| `TASK_NOTIFICATION_BODY` | None | Required per-invocation plain-text UTF-8 body. |
| `TASK_NOTIFICATION_ENV_FILE` | `.env` | Selects the env file used to supplement the process environment; set it to `.env.prod` explicitly when production credentials are intended. |

The SMTP endpoint, port, TLS mode, minimum TLS version, and recipient are not
configuration inputs. Process-environment values always win over env-file
values, including when an alternate file is selected.

## Invariants

- Sending occurs only through an explicit `make notify-task-complete` invocation;
  no other Make target depends on it.
- Every accepted message is addressed only to `2687665142@qq.com` through
  `smtp.exmail.qq.com:465` using implicit TLS 1.2 or newer.
- The caller controls only the subject and plain-text body, not message routing
  or SMTP transport settings.
- Header values cannot contain newlines, preventing caller input from injecting
  additional mail headers.
- A single invocation performs no more than three SMTP delivery attempts and
  reuses one `Message-ID` for all of them.
- Diagnostic output does not include the SMTP password or serialized
  authentication payload.

## Failure Recovery

Failure to load an explicitly selected env file, missing required values,
invalid content, and message-construction errors fail before network access and
are not retried. The command reports a concise error and exits nonzero. A
missing default `.env` alone is not an error when the required values already
exist in the process environment.

Connection, TLS, authentication, and message-submission failures are retried up
to three total attempts with one-second and two-second backoff intervals. Each
attempt uses a new connection and a new ten-second deadline. When all attempts
fail, the process reports a credential-safe error and exits nonzero.

A connection can fail after the server accepted the message but before the
client observed that acceptance. Retrying such an ambiguous failure can produce
duplicate mail. Reusing the `Message-ID` makes the duplicates identifiable but
does not provide exactly-once delivery. A failure after confirmed message
acceptance, including a failed `QUIT`, is not retried.

## Observability

The command writes invocation-level success or failure to the terminal and uses
its exit status as the Make contract: zero means the SMTP server accepted the
message, and nonzero means validation or all allowed delivery attempts failed.
Retry diagnostics identify the attempt and failure stage without printing
credentials or message content.

There are no metrics, health endpoints, persistent delivery records, or inbox
confirmation checks. Operators diagnose failures from the command output and
Tencent Exmail account state.

## Change Checklist

- [ ] The Make target remains explicit and independent of all other targets.
- [ ] Fixed SMTP, TLS, authenticated-sender, and recipient boundaries match the implementation.
- [ ] Env-file precedence and required invocation inputs remain current.
- [ ] MIME serialization and header-injection safeguards remain current.
- [ ] Attempt count, timeouts, backoff, acceptance point, and duplicate-delivery risk remain current.
- [ ] Terminal output and exit behavior remain credential-safe.
- [ ] Source links resolve, and the [design index](../README.md) contains the current summary.
