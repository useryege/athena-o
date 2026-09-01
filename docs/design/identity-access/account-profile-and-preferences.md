# Account Profile and Preferences

## Scope

Account Profile and Preferences owns each UUID account's editable display name,
display-only Standard/Pro tier, private avatar metadata, and System/Light/Dark
theme. It exposes uncached reads and independent optimistic-concurrency
boundaries for public profile and private preference state.

The immutable username belongs to [Account Credentials](account-credentials.md)
and appears read-only as `@username`. The shared registration reached from
[Google OIDC Login](google-oidc-login.md) or [Solana Wallet
Authentication](solana-wallet-authentication.md) initializes both username and
display name only when registration commits. Later external logins never
overwrite username, display name, avatar, tier, or theme; Google may refresh
only verified-email audit data.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Domain state and validation | [internal/accountcenter/types.go](../../../internal/accountcenter/types.go) | `Profile`, `Preferences`, `AvatarMetadata`, `Tier`, `ThemeMode` |
| Application boundary | [internal/accountcenter/manager.go](../../../internal/accountcenter/manager.go) | `Manager`, `GetProfile`, `UpdateDisplayName`, `UpdateTier`, `ReplaceAvatar`, `DeleteAvatar`, `UpdatePreferences` |
| PostgreSQL adapter | [internal/accountstate/store/sql_store.go](../../../internal/accountstate/store/sql_store.go) | `AccountExists`, `GetProfile`, `UpdateProfile`, `GetPreferences`, `UpdatePreferences` |
| Schema and queries | [internal/accountstate/store/migrations/000001_init.sql](../../../internal/accountstate/store/migrations/000001_init.sql), [internal/accountstate/store/queries/account_center.sql](../../../internal/accountstate/store/queries/account_center.sql) | `account_profile`, `account_preferences` |
| Registration aggregate | [internal/accountstate/store/queries/account_directory.sql](../../../internal/accountstate/store/queries/account_directory.sql) | `CreateOrdinaryAccount`, `CreateAdministratorAccount`, development member/administrator creation |
| API projection and mutations | [internal/server/account/account.go](../../../internal/server/account/account.go), [internal/server/account/account.proto](../../../internal/server/account/account.proto) | `ToAPIAccountProfile`, `ToAPIAccountPreferences`, `UpdateAccountProfile`, `UpdateAccountTier`, `UpdateAccountPreferences` |
| Browser surfaces and realm facades | [ui/src/app/shared/pages/account-center.tsx](../../../ui/src/app/shared/pages/account-center.tsx), [ui/src/app/admin/pages/admin-accounts.tsx](../../../ui/src/app/admin/pages/admin-accounts.tsx), [ui/src/app/shared/services/accounts-service.ts](../../../ui/src/app/shared/services/accounts-service.ts), [ui/src/app/admin/accounts-service.ts](../../../ui/src/app/admin/accounts-service.ts) | `AccountCenterPage`, `AdminAccountsPage`, neutral `SelfAccountService`, management-only `AdminAccountsService` |
| Neutral browser presentation | [ui/src/app/shared/account-presentation.tsx](../../../ui/src/app/shared/account-presentation.tsx), [ui/src/app/shared/theme.ts](../../../ui/src/app/shared/theme.ts), [ui/src/app/shared/validation.ts](../../../ui/src/app/shared/validation.ts) | account avatar/tier labels, theme conversion, profile validation shared without cross-realm page imports |

## Architecture

`accountcenter.Manager` accepts a stable account UUID and first validates that
the `athena_account` parent exists. It delegates every read and CAS mutation to
the shared account-state PostgreSQL adapter and keeps no profile cache. A
profile revision covers display name, tier, and avatar reference; a separate
preferences revision covers theme, so a theme write cannot conflict with a
profile write.

Username and display name are intentionally distinct. Username is permanent,
case-preserving public identity. Display name is 1–80 UTF-8 characters without
control characters and may be changed. Profile/avatar routes use UUID even
though labels use display name and `@username`.

Both frontend applications expose the same current-account profile, appearance,
access, Help, and logout capabilities under their own route roots. Only the
member application exposes Security and API Key management. Administrators may
also view safe identity/profile data and change an ordinary account's display-
only tier and profile through Account Admin; they do not receive another user's
private preferences or API Key metadata. Tier never grants authorization.

## Runtime Flow

1. A successful Google or Solana-wallet username registration creates
   `athena_account`, access, all module rows, profile, and preferences together.
   Profile display name is initialized to the exact chosen username, tier to
   Standard, theme to System, and both revisions to one. The disabled-auth
   development aggregate uses the same complete state shape with
   username/display name `local-user` for role `member` or `local-admin` for
   role `administrator`.
2. Reads require the UUID parent and require an already-persisted positive-
   revision row. Missing or malformed profile/preferences state is treated as
   an invalid aggregate, not synthesized from email or username.
3. A profile or preference update validates the complete proposed value and
   expected revision. Its SQL CAS advances exactly one revision on success. A
   stale revision returns a stable conflict and changes nothing.
4. Avatar upload validates and stores a private candidate object before profile
   CAS. Successful replacement commits its reference before old-object cleanup;
   failed or ambiguous CAS is reconciled through a read and orphan collection.
5. Session and bootstrap projections combine UUID, immutable username, current
   profile/preferences, current access, role, and safe provider-specific
   identity presentation. Google subject, API Key JTI, wallet signatures,
   bearer material, and avatar object key remain private.
6. Member Account Center uses `/account/*`; Administrator Account Center uses
   `/admin/account/*`. Both show `@username` in a disabled input and permit
   editing only display name and other mutable profile fields. The administrator
   directory detail adds a copyable Technical account ID; neither self-service
   surface presents UUID as the user's public name.

## State / Data

`account_profile.account_id UUID` and `account_preferences.account_id UUID` are
primary keys and foreign keys to `athena_account`. Profile owns display name,
tier, optional avatar object metadata, and revision. Preferences owns theme and
its own revision. Neither table stores username, email, subject, or role.

The account directory's immutable `username`, provider binding, and mutable
Google-only `verified_email` remain separate. An email change on successful
Google login does not alter any profile field; a Solana address is never used to
initialize or overwrite display name. An account can be disabled without
deleting or rewriting presentation state.

Avatar bytes remain private S3 objects. Public profile projections expose only
an authenticated UUID route such as `/api/v1/account/{id}/avatar?v={revision}`.
The revision is a cache buster; it is not an object-store key or credential.

## Configuration

Profile and preferences have no per-account environment settings. Theme starts
as System and tier starts as Standard. Object-store configuration belongs to
[Account Avatar Storage](account-avatar-storage.md).

## Invariants

- Every profile and preferences row belongs to the same durable UUID account.
- Username is read-only identity presentation; display name is the editable
  profile label and starts equal to username.
- Google email, name, and avatar and Solana-wallet metadata never synchronize
  into Athena presentation state after registration.
- Profile and preference revisions advance independently by CAS.
- Tier is presentation metadata and cannot grant a module or role.
- Safe public/session projections exclude subject, private preferences of other
  users, API Key metadata, and object-store secrets.
- Self-service profile and theme behavior is shared across frontend realms,
  while API Key management remains member-only.

## Failure Recovery

Registration rolls back account, access, modules, profile, and preferences as
one database statement if any inserted component fails. A stale revision or SQL
failure leaves its aggregate unchanged. Avatar write/CAS failures are reconciled
by bounded re-read, compensation, and the orphan collector; no candidate becomes
live until its UUID profile reference commits.

## Observability

Mutation and avatar failures identify account UUID and operation without logging
avatar bytes, external identity subjects, wallet signatures, JWTs, or API Key
material. Profile reads add no health dependency beyond the shared
account-state PostgreSQL connection.

## Change Checklist

- [ ] UUID account parent and registration defaults remain current.
- [ ] Immutable username and editable display-name presentation remain separate.
- [ ] Profile, preferences, and avatar CAS boundaries remain independent.
- [ ] External identity data does not overwrite Athena display state.
- [ ] Authorization and public projections remain current.
- [ ] The [design index](../README.md) contains the current summary.
