# Account and Wallet Avatar Storage

## Scope

Account and Wallet Avatar Storage owns shared image validation, private
S3-compatible object storage, authenticated native HTTP delivery, replacement
compensation, and orphan collection for account-profile and custodial-wallet
avatars. It also owns the pinned MinIO server/client images and the local and
production volume lifecycle.

[Account Profile and Preferences](account-profile-and-preferences.md) owns the
account-profile revision and avatar metadata reference. [Wallet Ownership and
Custody](wallet-ownership.md) owns wallet avatar kind, preset, uploaded-object
metadata, and Wallet revision. The API Server owns their deliberately different
authorization rules: an account avatar is readable or writable by its owner or
an administrator, while a wallet avatar is always restricted to the exact
wallet owner with no administrator bypass. The browser never receives S3
credentials, endpoint, bucket name, or object key.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| S3-compatible adapter | [internal/accountavatar/store.go](../../../internal/accountavatar/store.go) | `Config`, `Store`, `Object`, `ObjectInfo`, `Put`, `Get`, `Stat`, `Delete`, `List` |
| Shared image validation | [internal/avatarimage/image.go](../../../internal/avatarimage/image.go) | `Validate`, `Image`, `DefaultMaxBytes`, `MaxDimension`, `MaxPixels` |
| Account HTTP resource | [internal/server/accountavatarhttp/handler.go](../../../internal/server/accountavatarhttp/handler.go), [internal/server/accountavatarhttp/garbage_collector.go](../../../internal/server/accountavatarhttp/garbage_collector.go) | `Handler.Upload`, `Handler.Download`, `Handler.Delete`, `RunGarbageCollector` |
| Wallet HTTP resource | [internal/server/walletavatarhttp/handler.go](../../../internal/server/walletavatarhttp/handler.go), [internal/server/walletavatarhttp/garbage_collector.go](../../../internal/server/walletavatarhttp/garbage_collector.go) | `Handler.Upload`, `Handler.Download`, `Handler.Delete`, `DeleteObjectBestEffort`, `RunGarbageCollector` |
| Authentication and process wiring | [internal/server/account_avatar.go](../../../internal/server/account_avatar.go), [internal/server/wallet_avatar.go](../../../internal/server/wallet_avatar.go) | `newPrivateAvatarStore`, `authenticateAccountAvatarHTTP`, `authenticateWalletAvatarHTTP`, route registration |
| Pinned object-store images | [deploy/minio/Dockerfile.server](../../../deploy/minio/Dockerfile.server), [deploy/minio/Dockerfile.mc](../../../deploy/minio/Dockerfile.mc) | MinIO commit `9e49d5e7a648`, mc commit `7394ce0dd2a8` |
| Private bucket initialization | [deploy/minio/init-avatar-bucket.sh](../../../deploy/minio/init-avatar-bucket.sh) | bucket creation, anonymous-access removal, application IAM policy |
| Local runtime | [hack/start-minio.sh](../../../hack/start-minio.sh), [hack/local-runtime.sh](../../../hack/local-runtime.sh) | pinned image bootstrap, `athena-local-minio-data`, stop/reset ownership checks |
| Production runtime | [docker-compose.prod.yml](../../../docker-compose.prod.yml), [hack/prod-remote-deploy.sh](../../../hack/prod-remote-deploy.sh) | `minio`, `minio-init`, `PROD_MINIO_VOLUME` |

## Architecture

```mermaid
flowchart LR
    B["Authenticated browser or API Key"] --> A["Account avatar HTTP handler"]
    B --> W["Wallet avatar HTTP handler"]
    A --> P["Account profile CAS"]
    W --> C["Owner-scoped Wallet metadata CAS"]
    A --> S["Private S3-compatible bucket"]
    W --> S
    G1["Account orphan collector"] --> P
    G1 --> S
    G2["Wallet orphan collector"] --> C
    G2 --> S
```

`accountavatar.Store` is a narrow AWS SDK v2 adapter targeting one fixed private
bucket with static bucket-scoped credentials and configurable path-style
addressing. `newPrivateAvatarStore` validates the common configuration and is
used to construct the account and wallet handlers. Client construction performs
no startup network probe, so object-store availability remains isolated from
non-avatar APIs.

Both handlers use `avatarimage.Validate`, but their metadata and authorization
are independent. Account routes accept a canonical target account UUID and use
Account Center profile CAS. Wallet routes accept only a positive wallet ID,
derive the authenticated account UUID, require Wallet module access, and invoke
trusted owner-scoped Wallet metadata methods. Neither route accepts an object
key or owner UUID from public input.

## Runtime Flow

1. MinIO starts against its named `/data` volume. The one-shot mc image waits
   for readiness, creates the configured bucket, removes anonymous access,
   creates or updates the application user, and attaches a bucket-scoped
   list/get/put/delete policy. Repeated initialization is idempotent.
2. API Server startup validates endpoint, region, bucket, access key, secret
   key, endpoint URL, path-style setting, and maximum upload bytes, then
   constructs both avatar handlers. Missing or malformed configuration prevents
   startup; MinIO reachability is not probed.
3. Both multipart upload resources limit the complete request and image bytes,
   inspect image configuration before pixel allocation, and fully decode the
   image. JPEG, PNG, and structurally valid non-animated WebP are accepted up to
   2 MiB, 4,096 pixels on either edge, and at most 16,000,000 total pixels. WebP
   extended-canvas dimensions must match its VP8 or VP8L frame. SVG, GIF,
   animated WebP, truncated content, and content-type-only claims are rejected.
4. `PUT /api/v1/account/{id}/avatar` requires multipart `file` and
   `expectedRevision`. After owner-or-administrator authorization, it verifies
   the profile revision, writes a candidate under
   `objects/<sha256(account UUID)>/<random UUID>`, and commits key, content type,
   ETag, and size through profile CAS.
5. `PUT /api/v1/wallets/{id}/avatar` requires Wallet `READ_WRITE` and multipart
   `file` plus `expectedRevision`. It first resolves the wallet through the
   authenticated owner UUID, writes a candidate under
   `wallet-avatars/<sha256(owner UUID)>/<random UUID>`, and commits uploaded-
   object metadata through Wallet revision CAS. The commit clears any preset.
6. After an ambiguous metadata update, each handler re-reads its durable
   reference with an independent bounded context. A committed candidate is
   preserved and returned; a confirmed unreferenced candidate is deleted best
   effort; an unreconcilable candidate remains for grace-period collection.
   Successful replacement commits first and then deletes the prior object best
   effort.
7. `GET /api/v1/account/{id}/avatar` repeats account owner-or-administrator
   authorization and resolves the current profile reference.
   `GET /api/v1/wallets/{id}/avatar` requires Wallet `READ`, exact owner lookup,
   and an uploaded-object reference. Both stream persisted content type, size,
   ETag, `nosniff`, and private revalidation cache headers; matching
   `If-None-Match` returns 304. Missing wallet image bytes are presented by the
   UI as the deterministic default avatar.
8. Account-avatar deletion clears the profile reference through profile CAS. Wallet
   avatar reset requires Wallet `READ_WRITE`, clears uploaded metadata and any
   preset through Wallet CAS, and returns the deterministic default state.
   Setting a Wallet preset through the public Wallet API likewise replaces an
   upload reference and triggers best-effort object cleanup.
9. Separate collectors run once when the API Server context starts and then
   every 24 hours. Each loads its own durable references, lists only its object
   prefix, and deletes unreferenced objects after a 24-hour grace period. A
   reference-load or listing failure stops that collection pass without deleting
   anything.
10. Ordinary local stop removes the MinIO container but retains its volume.
    `make run-reset` removes the owned MinIO volume. Production hot deploy
    retains the configured external volume, while a full fresh deployment owns
    its complete new object state.

## State / Data

The private bucket contains validated original bytes in two disjoint key spaces:

- account avatars: `objects/<sha256(account UUID)>/<random UUID>`;
- wallet avatars: `wallet-avatars/<sha256(owner UUID)>/<random UUID>`.

Object names contain no username, display name, wallet address, source file
name, or extension. S3 supplies content type, content length, ETag, and last-
modified time. The same bucket and application credential serve both prefixes,
but neither HTTP authorization policy uses object-key structure as proof of
ownership.

PostgreSQL metadata is the live-reference source of truth. Account references
are committed in `account_profile`; wallet references are committed in the
owner-scoped `wallets` row and are mutually exclusive with an avatar preset. An
object-store write alone is only a candidate. MinIO data lives in
`athena-local-minio-data` locally and `PROD_MINIO_VOLUME` in production.

## Configuration

| Setting | Behavior |
| --- | --- |
| `ATHENA_ACCOUNT_AVATAR_S3_ENDPOINT` | Required HTTP(S) S3 endpoint with no credentials, query, fragment, or path; local default is `http://127.0.0.1:9000`, production uses `http://minio:9000`. |
| `ATHENA_ACCOUNT_AVATAR_S3_REGION` | Signing and MinIO region; defaults to `us-east-1`. |
| `ATHENA_ACCOUNT_AVATAR_S3_BUCKET` | Private bucket shared by account and wallet avatar prefixes; defaults to `athena-account-avatars`. |
| `ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID` | Required bucket-scoped application access key. |
| `ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY` | Required bucket-scoped application secret key. |
| `ATHENA_ACCOUNT_AVATAR_S3_PATH_STYLE` | Selects AWS SDK path-style addressing; defaults to `true`. |
| `ATHENA_ACCOUNT_AVATAR_MAX_BYTES` | Upload byte limit shared by both surfaces; defaults to and cannot exceed 2 MiB. Lower positive values are accepted; invalid or oversized values fall back to 2 MiB. |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | Initialization-only administrator credential; production password is required and generated by `prod-reset-secrets`. |
| `ATHENA_MINIO_API_PORT`, `ATHENA_MINIO_CONSOLE_PORT` | Local loopback bindings; default to `9000` and `9001`. Production exposes neither port to the host. |
| `ATHENA_MINIO_IMAGE`, `ATHENA_MINIO_MC_IMAGE` | Optional local overrides for the repository-built pinned images. |
| `PROD_MINIO_VOLUME` | External production volume name; defaults to `athena-prod-minio-data`. |

The repository images build MinIO Server from commit
`9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a` and mc from commit
`7394ce0dd2a80935aded936b09fa12cbb3cb8096`. The server commit is the
[official `RELEASE.2025-10-15T17-29-55Z` release](https://github.com/minio/minio/releases/tag/RELEASE.2025-10-15T17-29-55Z).

## Invariants

- The browser and public metadata APIs never receive the MinIO endpoint,
  credentials, bucket, or durable object key.
- The bucket has no anonymous policy; the application user is limited to bucket
  location, listing, and object get/put/delete.
- Account-avatar access allows the matching account UUID or persisted
  administrator. Wallet-avatar access always requires exact owner UUID and never
  grants an administrator bypass.
- Wallet avatar reads require Wallet `READ`; uploads, preset replacement, and
  reset require `READ_WRITE`. API Keys may use these safe metadata operations
  when their account entitlements allow them.
- Stored bytes have passed format, animation, byte, edge-dimension, at-most
  16,000,000-pixel, and full-decode validation.
- A PostgreSQL CAS is the live-reference commit point. Compensation and
  collection never intentionally delete a currently referenced object.
- Ordinary stop and production hot deploy retain their exact named MinIO
  volume. Explicit reset and full deployment replace the complete object state.

## Failure Recovery

Invalid or missing S3 configuration fails API startup before the listener
opens. MinIO unavailability after construction does not affect Wallet listing,
remark or preset edits, private-key reveal, Account Center metadata, or other
API capabilities. Avatar reads and writes return an unavailable or not-found
response as appropriate; wallet presentation falls back to its deterministic
default.

A failed candidate write leaves PostgreSQL unchanged. A failed metadata update
is reconciled before compensation so an ambiguous commit does not cause a live
object to be deleted. Failed old-object cleanup is logged and repaired by the
prefix-specific collector after the grace period. The Compose health check
still gates initial private-bucket setup, and repeated initialization reconstructs
IAM from retained MinIO state.

## Observability

MinIO logs are streamed by Goreman locally and retained by Compose in
production. Lifecycle logs identify image builds, volume creation/removal,
readiness failure, and private-bucket initialization. Avatar handler logs use
account UUID and, for wallet operations, wallet ID when diagnosing storage,
streaming, reconciliation, or cleanup. They exclude usernames, wallet addresses,
image bytes, object credentials, and private keys. Each collector reports
reference/list failures, per-object deletion failures, and deletion/scanned
counts when it removes objects.

## Change Checklist

- [ ] Shared store configuration and image validation match both handlers.
- [ ] Account owner-or-administrator and Wallet exact-owner policies remain distinct.
- [ ] Both CAS, candidate reconciliation, replacement, and reset flows remain ordered as documented.
- [ ] Private delivery, ETag behavior, and cache headers remain current.
- [ ] Prefix-specific collectors protect referenced and grace-period objects.
- [ ] MinIO images, IAM policy, volume lifecycle, and configuration remain aligned.
- [ ] Source links and named symbols resolve to the implementation.
- [ ] The [design index](../README.md) contains the correct entry.
