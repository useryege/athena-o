# Profit Sharing

> 设计状态：已实现

## Scope

Profit Sharing owns durable draft and opened rounds, configurable draft rosters
that must contain five accounts before opening,
immutable participant presentation snapshots, proposals, votes, ballots,
runoffs, and administrator lifecycle operations. Every relationship and
self/duplicate decision uses Athena account UUID. Username and display name are
round snapshots used only for readable labels.

[Account Access Control](../identity-access/account-access-control.md) owns
current login, administrator role, and the independent Profit Sharing
entitlement. [Account Credentials](../identity-access/account-credentials.md)
owns UUID identity and immutable username. Profit Sharing does not create
accounts, resolve external identity subjects, or infer role from username.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| API Server facade and canonical participant projection | [internal/server/profitsharing/profitsharing.go](../../../internal/server/profitsharing/profitsharing.go) | `Server`, `requester`, `canonicalParticipantInputs`, `validateParticipants`, `OpenRound` |
| Public authorization boundary | [internal/server/authz.go](../../../internal/server/authz.go) | Profit Sharing RPC requirements |
| Domain service | [internal/profitsharing/service.go](../../../internal/profitsharing/service.go) | `Service` round, participant, proposal, vote, and projection methods |
| Public and internal contracts | [internal/server/profitsharing/profitsharing.proto](../../../internal/server/profitsharing/profitsharing.proto), [internal/profitsharing/profit_sharing.proto](../../../internal/profitsharing/profit_sharing.proto) | `Participant.account_id`, `Proposal.author_account_id`, internal requester fields |
| Persistence | [internal/profitsharing/store/migrations/000001_init.sql](../../../internal/profitsharing/store/migrations/000001_init.sql), [internal/profitsharing/store/queries/profit_sharing.sql](../../../internal/profitsharing/store/queries/profit_sharing.sql) | participant, proposal, proposal item, ballot, and vote UUID columns |
| UUID conversion boundary | [internal/profitsharing/store/account_id.go](../../../internal/profitsharing/store/account_id.go) | `canonicalAccountID`, `accountUUID`, `accountIDFromUUID` |
| Administrator UI | [ui/src/app/admin/pages/profit-sharing-admin.tsx](../../../ui/src/app/admin/pages/profit-sharing-admin.tsx), [ui/src/app/shared/pages/profit-sharing-shared.tsx](../../../ui/src/app/shared/pages/profit-sharing-shared.tsx) | account search, roster editor, readable participant labels |
| Realm-specific browser services | [ui/src/app/shared/services/profit-sharing-service.ts](../../../ui/src/app/shared/services/profit-sharing-service.ts), [ui/src/app/member/profit-sharing-service.ts](../../../ui/src/app/member/profit-sharing-service.ts), [ui/src/app/admin/profit-sharing-service.ts](../../../ui/src/app/admin/profit-sharing-service.ts), [ui/src/app/member/services.ts](../../../ui/src/app/member/services.ts), [ui/src/app/admin/services.ts](../../../ui/src/app/admin/services.ts) | neutral `ProfitSharingReader` and parsers; disjoint `MemberProfitSharingService` and `AdminProfitSharingService` command surfaces |
| Member UI | [ui/src/app/member/pages/profit-sharing.tsx](../../../ui/src/app/member/pages/profit-sharing.tsx) | member rounds, UUID self checks, proposal and vote operations |

## Architecture

Profit Sharing has its own PostgreSQL database. The API Server facade derives
the requester UUID from session claims and the administrator boolean from the
persisted credential record. Public request fields cannot choose either. The
facade also replaces client-supplied username and display name in draft roster
input with current canonical values from `CredentialManager` and
`accountcenter.Manager`.

The internal service receives explicit `requester_account_id` and
`requester_is_admin` only from that trusted facade. It validates canonical UUID,
role, membership, phase, proposal ownership, ballot state, and self-vote rules.
`ListRounds` and `GetRound` are the shared read contract: an administrator uses
its explicit role to obtain the management projection, while an ordinary
account must pass the Profit Sharing entitlement and domain membership rules.
Proposal and vote commands always use the member boundary and reject an
administrator. Round lifecycle commands always require the persisted
administrator role, never a special username.

The administrator participant selector queries the Account API with
`profitSharingEligibleOnly=true`. The server returns registered non-
administrator accounts with login and Profit Sharing enabled. Browser labels
prefer profile display name and always include `@username`; UUID is retained as
the selection value and React/domain key.

The browser exposes the two command sets from separate application roots. The
member application owns `/profit-sharing/*`; the administrator application owns
`/admin/profit-sharing/*`. They may share neutral round DTO normalization and
presentation components, but neither application imports the other realm's
commands or route tree.

## Runtime Flow

1. An administrator reads the management projection and creates or edits a
   draft roster with distinct account UUIDs, display order, and baseline
   responsibilities. A draft may remain incomplete while it is being prepared.
   The API Server ignores the submitted presentation strings and snapshots each
   current immutable username and current display name from account state.
2. `OpenRound` reloads the draft projection and revalidates every UUID. Each must
   still be registered, ordinary, distinct, login enabled, Profit Sharing
   enabled, and backed by a valid Google or Solana external login identity. It
   sends the exact validated UUID set to the domain service.
3. The domain transaction compares that set with the complete durable roster,
   verifies revision and draft fields, locks the round, and opens only on exact
   equality. It creates one proposal per participant and one proposal item for
   each `(proposal, participant_account_id)` pair before entering `COLLECTING`.
4. During collection, an entitled ordinary participant reads and revises only
   the proposal whose `author_account_id` matches the request UUID. Submission
   requires every responsibility and allocation, totaling exactly 10,000 basis
   points. A proposal may be reopened before publication.
5. After all five submissions, publication assigns stable anonymous labels,
   creates ballot one with every proposal, and enters `VOTING`. Members can see
   allocations but not authors or live tallies.
6. Voting upserts one selection per `(round, ballot, voter_account_id)`. The
   transaction rejects selection whose proposal author UUID equals voter UUID
   and rechecks the active ballot so a racing transition cannot redirect a vote.
7. Closing requires five votes. A unique leader closes the round and records
   the winning proposal. A tie closes the current ballot and atomically creates
   a runoff containing only the highest-tied proposal UUIDs. Runoffs repeat
   until one winner exists.
8. A closed projection exposes final-ballot candidates, aggregate results,
   readable author snapshots, and the winner, without exposing eliminated-
   candidate or cross-ballot history.
9. Disabling login immediately blocks sessions and therefore member operations.
   Disabling only Profit Sharing blocks member RPCs without rewriting roster or
   historical snapshots. Re-enabling access resumes eligible membership. The
   administrator's fixed Profit Sharing entitlement remains disabled; its read
   and lifecycle access comes solely from explicit administrator authorization.
10. The standalone service starts on port 8108, marks gRPC health serving after
    its database is ready, and shuts down by setting health not serving,
    gracefully stopping gRPC, and closing PostgreSQL.

## State / Data

`profit_sharing_participant` is keyed by `(round_id, account_id UUID)` and stores
immutable `username`, `display_name`, display order, and baseline
responsibility. Profile or email changes after the snapshot do not rewrite it.

| Table | Durable responsibility |
| --- | --- |
| `profit_sharing_round` | Unique slug, title, phase, revision, active ballot, winner proposal, and lifecycle times. |
| `profit_sharing_participant` | UUID roster identity plus username/display-name snapshot, order, and baseline responsibility. |
| `profit_sharing_proposal` | One proposal per `(round_id, author_account_id UUID)`, status, blind label, revision, and timestamps. |
| `profit_sharing_proposal_item` | One responsibility and nullable basis-point allocation per proposal and `participant_account_id UUID`. |
| `profit_sharing_ballot` | Numbered open or closed ballot per round. |
| `profit_sharing_ballot_candidate` | Proposal subset for an initial or runoff ballot. |
| `profit_sharing_vote` | One selection per ballot and `voter_account_id UUID`, with proposal-author UUID for database self-vote enforcement. |

The current entitlement remains in account-state, not this database. There is
no process-local workflow cache or scheduler. Round and proposal revisions,
row locks, foreign keys, and unique constraints are the commit/concurrency
boundaries. Restart reconstructs all workflow state from PostgreSQL.

## Configuration

Profit Sharing eligibility has no per-user environment variables. Account UUID,
username, display name, role, and eligibility come through the API Server.

| Setting | Behavior |
| --- | --- |
| `ATHENA_PROFIT_SHARING_LISTEN_ADDRESS` | Listener address, default `0.0.0.0`. |
| `ATHENA_PROFIT_SHARING_LISTEN_PORT` | Listener port, default `8108`; the local full-stack runtime supplies `--port` from `ATHENA_PROFIT_SHARING_PORT`. |
| `ATHENA_PROFIT_SHARING_POSTGRES_DSN` | Connection for the owned `profit_sharing` database. |
| `ATHENA_POSTGRES_AUTO_MIGRATE` | Enables embedded local migration; production runs migration before service startup. |
| `ATHENA_PROFIT_SHARING_SERVER_ADDRESS` | API Server internal gRPC target, default `127.0.0.1:8108` and service DNS in Compose. |

## Invariants

- Member RPCs require current login, Profit Sharing entitlement, and UUID round
  membership.
- Open accepts exactly five distinct, registered, login-enabled, Profit-Sharing-
  enabled ordinary UUID accounts.
- Role and self identity are never inferred from username, display name, email,
  Solana address, or another external identity subject.
- Username/display-name snapshots are presentation only and cannot change a
  relationship or historical round.
- The administrator owns lifecycle actions and cannot act as a participant.
- Administrator role does not satisfy the Profit Sharing member entitlement;
  member proposal and vote commands reject an administrator even when the
  administrator can read the same round through the management projection.
- Revocation does not delete or rewrite durable round history.
- Proposal items and votes are keyed and de-duplicated by UUID; self-vote is
  enforced both in service logic and PostgreSQL.
- Collection privacy, blind voting, author/tally visibility, revision CAS, and
  runoff rules remain phase-consistent.

## Web presentation and draft ownership

The member list/detail follow the approved v20 layouts and the administrator
list/detail follow v19, using the v23 shared dark theme and relative type sizes.
Compact lists keep native round links and full slugs. Collection remains sealed;
voting remains anonymous; closed rounds display the published result.

Local drafts and confirmation dialogs belong to the account UUID, issuer, realm,
and round slug. Changing that identity or route discards their local state.
The member shell refreshes authorization on
`ACCOUNT_PROFIT_SHARING_ACCESS_DENIED`, so revoked entitlement removes proposal
and vote editing promptly. Failed writes retain an editable draft; revision
conflicts discard it and reload the authoritative proposal.

Administrators can create a draft with zero to five participants, including when
the eligible account directory is empty. Opening collection still requires exactly
five distinct eligible accounts. An unfiltered directory refresh replaces the
known eligibility set so a removed account cannot leave the Open action enabled.

## Failure Recovery

Account directory or profile failure prevents canonical draft projection.
Eligibility failure occurs before the domain open transition. The expected
round revision and exact UUID-set comparison prevent a concurrent roster edit
from opening stale participants.

Domain validation, query, cancellation, or commit failure rolls back its
multi-row mutation. Revision mismatches return `Aborted`; phase/precondition
failures return `FailedPrecondition`; malformed input returns `InvalidArgument`;
membership and self-vote failures return `PermissionDenied`. A later access
change is not distributed into an existing workflow transaction, so every
subsequent member request rechecks current account access.

## Observability

Authorization denials distinguish login, Profit Sharing entitlement, role, and
round membership. Logs use account UUID, round slug/ID, proposal ID, and ballot
number; external identity subjects, JWTs, and API Key JTIs are excluded.
Standard gRPC
health and Service Status report process reachability. Durable timestamps and
revisions are the workflow diagnostic record.

## Change Checklist

- [ ] UUID requester, membership, proposal, item, voter, and self-vote semantics remain current.
- [ ] Canonical username/display-name snapshotting remains current.
- [ ] Independent entitlement and open-round revalidation remain current.
- [ ] Durable workflow phases, revisions, locks, and runoff rules remain current.
- [ ] UI labels avoid presenting UUID as the user's public identity.
- [ ] Member and administrator UIs remain in separate route and command graphs.
- [ ] The [design index](../README.md) contains the current summary.
