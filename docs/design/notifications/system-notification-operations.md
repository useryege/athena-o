# System Notification Operations

> 设计状态：已实现

## Scope

System Notification Operations owns operational Telegram notifications sent to
the configured test or production group and optional forum Topic. It includes
the internal producer contract, durable topics and deliveries, fair dispatch,
administrator list/detail/test operations, and the shared Notification runtime
view.

Market Radar, Sports Live, Managed OO, and Worm Markets determine their alert
conditions and retain their own source-side alert state. This capability accepts
their system requests and owns delivery afterward. It never selects an ordinary
account or exposes system records to the member application. Account binding and
private-chat delivery are documented in [Account Telegram Notifications](account-telegram-notifications.md).

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Internal system and runtime contracts | [internal/notification/notification.proto](../../../internal/notification/notification.proto) | `SystemNotificationService`, `NotificationRuntimeService` |
| System application logic | [internal/notification/service.go](../../../internal/notification/service.go) | `SendSystemNotification`, `ListSystemNotificationDeliveries`, `GetSystemNotificationDelivery` |
| Durable system store | [internal/notification/store/system_notifications.go](../../../internal/notification/store/system_notifications.go), [internal/notification/store/queries/system_notifications.sql](../../../internal/notification/store/queries/system_notifications.sql), [internal/notification/store/migrations/000001_init.sql](../../../internal/notification/store/migrations/000001_init.sql) | `EnsureSystemNotificationTopic`, system topic/delivery queue operations |
| Fair worker and Telegram gateway | [internal/notification/worker.go](../../../internal/notification/worker.go), [internal/notification/sender.go](../../../internal/notification/sender.go) | `claimFairNotificationBatch`, `processClaimedSystemNotification`, `TelegramSender` |
| Public administrator facade | [internal/server/notification/notification.proto](../../../internal/server/notification/notification.proto), [internal/server/notification/notification.go](../../../internal/server/notification/notification.go) | administrator list/detail/test/runtime HTTP routes |
| Administrator authorization | [internal/server/authz.go](../../../internal/server/authz.go) | `administratorGRPCMethods` notification entries |
| Administrator UI | [ui/src/app/admin/pages/system-notifications.tsx](../../../ui/src/app/admin/pages/system-notifications.tsx), [ui/src/app/admin/pages/system-notification-detail.tsx](../../../ui/src/app/admin/pages/system-notification-detail.tsx), [ui/src/app/admin/pages/service-status.tsx](../../../ui/src/app/admin/pages/service-status.tsx), [ui/src/app/admin/notification-service.ts](../../../ui/src/app/admin/notification-service.ts) | list/filter/detail/test pages and Notification runtime panel |
| Internal authenticated client | [internal/notification/apiclient/apiclient.go](../../../internal/notification/apiclient/apiclient.go), [internal/notification/apiclient/internal_auth.go](../../../internal/notification/apiclient/internal_auth.go) | `Clientset.System`, `Clientset.Runtime`, `NewNotificationClientset` |
| Current alert producers | [internal/marketradar/mover_alerts.go](../../../internal/marketradar/mover_alerts.go), [internal/sportslive/price_alerts.go](../../../internal/sportslive/price_alerts.go), [internal/sportslive/score_alerts.go](../../../internal/sportslive/score_alerts.go), [internal/managedoo/proposed_alerts.go](../../../internal/managedoo/proposed_alerts.go), [internal/managedoo/disputed_alerts.go](../../../internal/managedoo/disputed_alerts.go), [internal/wormmarkets/notifications.go](../../../internal/wormmarkets/notifications.go) | `SendSystemNotification` callers |
| Process and deployment wiring | [cmd/athena-notification/commands/athena_notification.go](../../../cmd/athena-notification/commands/athena_notification.go), [Procfile](../../../Procfile), [docker-compose.prod.yml](../../../docker-compose.prod.yml) | one process, one Bot, chat IDs, shared internal credential |

## Architecture

System and account notifications are logical domains inside one
`athena-notification` deployment. `SystemNotificationService` owns only
send/list/get system operations. `AccountNotificationService` owns private
recipient operations, and `NotificationRuntimeService` reports shared process
state. All three use one database, one Telegram Bot identity, one outbound
sender, one rate interval, and one fair worker.

Producers call `Clientset.System()` with the shared internal Bearer. The API
Server is the only public proxy: administrator sessions can list, inspect, and
queue tests, while member sessions and API Keys cannot read operational records.
The administrator Service Status page reaches the separate shared runtime
facade; runtime state is not a member capability.

## Runtime Flow

1. The Notification process validates its internal Bearer, database, Bot token,
   test/production chat IDs, and Telegram client. It synchronizes the Bot
   profile, verifies long-polling compatibility, then starts the shared poller
   and fair delivery worker before reporting gRPC health `SERVING`.
2. Market Radar, Sports Live, Managed OO, and Worm Markets create an
   authenticated client only when their notification feature is enabled. Their
   alerts call `SystemNotificationService.SendSystemNotification` with source,
   severity, title/body/link, logical `test` or `prod` chat, and Topic label.
3. `SendSystemNotification` validates the request and rendered Telegram length.
   `EnsureSystemNotificationTopic` first reads the durable `(telegram_chat,
   label)` mapping. A missing mapping is serialized with a PostgreSQL advisory
   lock, created through Telegram, and stored with its message-thread ID.
4. The accepted request inserts a `pending` system delivery referencing that
   Topic and immediately returns its delivery ID. Provider I/O occurs only in
   the worker.
5. Each worker pass divides its batch between account and system queues,
   alternates the preferred domain, and interleaves claimed rows. Every provider
   call uses the same `TelegramSender`; the configured send interval is applied
   between all account and system sends.
6. A system send resolves the configured group chat ID, requires the persisted
   Topic message-thread ID, renders escaped Telegram HTML, and records the
   provider message ID on success. Failures either schedule a retry or become
   terminal after the configured attempt cap.
7. `GET /api/v1/admin/system-notification-deliveries` pages and filters current
   system records. The detail endpoint resolves one positive delivery ID. The
   test endpoint always queues an informational `admin-ui` notification to the
   configured test chat under the submitted Topic label.
8. `/admin/notifications` renders the list, filters, compact mobile cards,
   provider-facing detail, and single-flight test modal. The administrator
   Service Status page reads `/api/v1/admin/notification-runtime/status` on
   entry, manual refresh, and a ten-second interval.

## State / Data

- `system_notification_topics` is keyed by `(telegram_chat, label)` and stores a
  positive Telegram forum `message_thread_id`. Logical chat is constrained to
  `test` or `prod`.
- `system_notification_deliveries` stores source, severity, title/body/link,
  Telegram channel, logical chat, Topic label, status, provider result, retry
  attempts, next-attempt time, and durable claim metadata. The Topic pair is a
  foreign key to the system topic table.
- System delivery states are `pending`, `sent`, and `failed`. A pending row with
  attempts greater than zero contributes to the runtime retry count.

`SystemNotificationDeliveryItem` and `SystemNotificationDeliveryDetail` are the
shared API projections used by both internal and public contracts. The account
delivery table and private Telegram identifiers are not included in these
projections.

## Configuration

| Setting | Behavior |
| --- | --- |
| `ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN` | Required Bearer shared by Notification, API Server, and the four current system producers. |
| `ATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID` | Concrete group chat behind logical `TELEGRAM_CHAT_TEST`. |
| `ATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID` | Concrete group chat behind logical `TELEGRAM_CHAT_PROD`. |
| `ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN`, `..._API_URL`, `..._TIMEOUT_SECONDS` | Shared Telegram Bot identity, API endpoint, and request timeout. |
| `ATHENA_NOTIFICATION_POSTGRES_DSN` | Database that owns both logical notification domains. |
| `ATHENA_NOTIFICATION_WORKER_SEND_INTERVAL`, `..._POLL_INTERVAL`, `..._BATCH_SIZE`, `..._MAX_ATTEMPTS`, `..._LOCK_TIMEOUT` | Shared rate interval, polling cadence, fair batch size, retry cap, and stale-lock recovery. |
| Producer-specific `..._NOTIFICATION_ENABLED` and `..._NOTIFICATION_SERVER_ADDRESS` | Enable each producer and select the Notification gRPC target. |

Production Compose clears the internal credential from unrelated services and
injects it only into Notification, the API Server, Market Radar, Sports Live,
Managed OO, and Worm Markets. It also clears the Telegram Bot token and concrete
group IDs from every non-Notification application container; producers and the
API Server know only the authenticated gRPC address. Production secret reset
rotates the internal credential independently from Wallet, Wallet signer, and
Worm Trading credentials.

## Invariants

- System operations never accept or infer an ordinary account UUID and never
  call `AccountNotificationService`.
- Member sessions and API Keys cannot list system deliveries, inspect details,
  queue a test, or read shared runtime state.
- Logical system chat and Topic label resolve to one durable positive Telegram
  thread ID before a delivery row is inserted.
- System and account queue claims are interleaved and every provider send shares
  one Bot sender and send interval; neither queue can permanently starve the
  other.
- Provider errors do not change producer-owned alert state inside the
  Notification database; source services retain their documented acceptance
  boundary.
- The old generic topic/delivery tables and Notifications account module are not
  part of the current schema or authorization model.

## Failure Recovery

Invalid Bot, chat, database, or internal-token configuration prevents local
construction or startup. A missing webhook-free polling configuration also
prevents health from becoming `SERVING`. A producer with a syntactically valid
but mismatched credential receives `Unauthenticated`; no system row is accepted.

Claimed rows keep lock owner/time and return to eligibility after the configured
lock timeout. Telegram 429 responses use the provider retry interval; other
temporary failures use bounded exponential delay. A terminal system failure
retains the adapter's bounded safe error category for administrator diagnosis;
raw provider descriptions, response bodies, request URLs, and Bot credentials
never cross into the stored error. Successful provider send followed by an
unavailable database commit remains an external-provider ambiguity and is
diagnosed from the retained pending/attempt metadata and Telegram group state.

Forum Topic creation is serialized among Notification requests. Telegram Topic
creation and the following database insert cross an external transaction
boundary; an insert/commit failure leaves the database without that mapping and
the next request retries Topic creation.

## Observability

Standard gRPC health reports lifecycle readiness. The shared runtime RPC reports
Bot availability/ID/username, poller active state, last successful poll and
handled update timestamps, system and account pending/retry/failed counts, and
unreachable bindings. Only administrators can view its public projection in
Service Status.

System list/detail records expose source, severity, target logical chat, Topic,
channel, status, provider message/error, creation time, and send time. Logs use
delivery IDs and logical system chat metadata; they never include the internal
Bearer or Telegram Bot token.

## Change Checklist

- [ ] System, account, and runtime service boundaries remain separate inside one process.
- [ ] Every current producer uses only authenticated `SendSystemNotification` and retains its source-side acceptance semantics.
- [ ] Topic serialization, durable delivery state, fair claiming, shared rate interval, and retries remain current.
- [ ] Administrator list, filters, detail, test send, and runtime panel match the public contract and authorization map.
- [ ] System records and operator code remain absent from the member bundle and member APIs.
- [ ] Configuration injection, secret rotation, health, runtime fields, and credential-safe logging remain synchronized.
- [ ] Source links resolve and the [design index](../README.md) contains the correct entry.
