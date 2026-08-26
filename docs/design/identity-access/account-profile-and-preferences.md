# Account Profile and Preferences

## Scope

Account Profile and Preferences owns each durable account's display name,
display-only Standard/Pro tier, private avatar metadata, and System/Light/Dark
theme choice. It provides uncached reads and separate optimistic-concurrency
boundaries for profile and preference state.

[Google OIDC Login](google-oidc-login.md) creates an initial ordinary-account
profile from the verified email. Later Google logins may update identity audit
email but never overwrite display name, tier, avatar, or theme. [Account
Credentials](account-credentials.md) owns identity and API Keys, while [Account
Access Control](account-access-control.md) owns authorization.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Domain state and validation | [internal/accountcenter/types.go](../../../internal/accountcenter/types.go) | `Profile`, `Preferences`, `AvatarMetadata`, `Tier`, `ThemeMode` |
| Application boundary | [internal/accountcenter/manager.go](../../../internal/accountcenter/manager.go) | `Manager`, `GetProfile`, `UpdateDisplayName`, `UpdateTier`, `ReplaceAvatar`, `DeleteAvatar`, `UpdatePreferences` |
| Shared PostgreSQL adapter | [internal/accountstate/store/sql_store.go](../../../internal/accountstate/store/sql_store.go) | `SQLStore`, `AccountExists`, `GetProfile`, `UpdateProfile`, `GetPreferences`, `UpdatePreferences` |
| Schema and queries | [internal/accountstate/store/migrations/000001_init.sql](../../../internal/accountstate/store/migrations/000001_init.sql), [internal/accountstate/store/queries/account_center.sql](../../../internal/accountstate/store/queries/account_center.sql) | `account_profile`, `account_preferences` |
| API projection and mutations | [internal/server/account/account.go](../../../internal/server/account/account.go), [internal/server/account/account.proto](../../../internal/server/account/account.proto) | `ToAPIAccountProfile`, `ToAPIAccountPreferences`, `UpdateAccountProfile`, `UpdateAccountTier`, `UpdateAccountPreferences` |
| Browser surfaces | [ui/src/app/pages/account-center.tsx](../../../ui/src/app/pages/account-center.tsx) | `AccountCenterPage` |

## Architecture

`accountcenter.Manager` validates account existence through the durable
`athena_account` parent. It delegates every
read and CAS mutation to the shared account-state PostgreSQL adapter and keeps no
profile cache. Profile and preferences have independent revisions, so a theme
write cannot conflict with a display-name or avatar write.

Ordinary users may change their own display name, avatar, and preferences.
Administrators may view account-safe identity/profile data and change an
ordinary account's display-only tier; they do not receive another user's private
preferences or API Key metadata.

## Runtime Flow

1. The database migration creates the fixed administrator profile. Preferences
   default to System through `Manager` and are inserted by the first successful
   preferences CAS. An ordinary OIDC provisioning transaction creates the
   account, access rows, all ten module rows, and initial profile together. Its
   display name is initialized from the verified email only once.
2. A read first verifies the account parent and then returns current PostgreSQL
   state. Missing preference state uses its explicit persisted/default boundary;
   it is not derived from Google.
3. A profile or preference update validates the complete request and expected
   revision. The SQL CAS advances exactly one revision on success. A stale
   revision returns a conflict without modifying other account state.
4. Avatar upload validates and stores the private object before the profile CAS.
   Successful replacement schedules the prior object for collection; failed CAS
   leaves the unreferenced new object eligible for orphan collection. Raw avatar
   delivery authorizes the viewer and resolves the current metadata reference.
5. Session and bootstrap projections combine current profile/preferences with
   current access and safe identity metadata. They never include Google subject,
   API Key JTI, or bearer material.

## State / Data

`account_profile` and `account_preferences` reference `athena_account` by foreign
key. Profile owns display name, tier, optional avatar object metadata, and a
positive revision. Preferences owns theme and its own positive revision.

The identity directory's `verified_email` is separate from the profile display
name. Email may change on a successful Google login; profile fields remain user-
or administrator-controlled. Deleting or disabling an account is not a profile
operation, and dynamic accounts are retained permanently.

Avatar bytes remain private S3 objects. Public Account projections expose only
an authenticated Athena URL with the profile revision as a cache-busting query,
not an object-store credential or key.

## Configuration

Profile/preferences have no per-account environment settings. Avatar object
store settings are documented in [Account Avatar Storage](account-avatar-storage.md).
Theme defaults to System and ordinary accounts start at the standard display
tier unless the current schema explicitly supplies another value.

## Invariants

- Every profile and preference row belongs to a durable Athena account.
- Google email, name, and avatar are not synchronized after provisioning.
- Profile and preference revisions advance independently by CAS.
- Ordinary callers mutate only their own profile/preferences; administrator tier
  mutation follows the fixed administrator boundary.
- Safe account/session projections exclude subjects, private preferences of
  other users, API Key metadata, and object-store secrets.

## Failure Recovery

Provisioning rolls back account, access, module, and initial profile state as one
transaction. A stale revision or SQL failure leaves the corresponding row
unchanged. Avatar object-write and profile-CAS failures are reconciled by the
orphan collector; a referenced object remains available until replacement
commits.

## Observability

Mutation failures identify the Athena account and operation without logging
avatar bytes, Google subjects, JWTs, or API Key material. Profile reads are not a
health dependency beyond the shared account-state PostgreSQL connection.

## Change Checklist

- [ ] Dynamic account parent and provisioning semantics remain current.
- [ ] Profile, preferences, and avatar CAS boundaries remain independent.
- [ ] Google identity data does not overwrite Athena display state.
- [ ] Authorization and public projections remain current.
- [ ] The [design index](../README.md) contains the current summary.
