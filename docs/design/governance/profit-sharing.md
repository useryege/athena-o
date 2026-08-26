# Profit Sharing

## Scope

Profit Sharing owns durable draft and opened rounds, participant rosters,
proposals, votes, immutable snapshots, and administrator lifecycle operations.
Athena's account layer separately decides whether an ordinary account is allowed
to use Profit Sharing at all. Membership and entitlement are both required for
member reads and writes.

[Account Access Control](../identity-access/account-access-control.md) owns the
independent `ProfitSharingEnabled` flag and login state. [Account Credentials](../identity-access/account-credentials.md)
owns durable account identity. Profit Sharing does not create accounts or infer
participants from Google identity data.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| API Server facade and participant validation | [internal/server/profitsharing/profitsharing.go](../../../internal/server/profitsharing/profitsharing.go) | `Server`, `OpenRound`, account-directory validation |
| Authorization boundary | [internal/server/authz.go](../../../internal/server/authz.go) | Profit Sharing RPC requirements |
| Domain service | [internal/profitsharing](../../../internal/profitsharing) | round, participant, proposal, vote, and snapshot operations |
| Proto contract | [internal/profitsharing/profit_sharing.proto](../../../internal/profitsharing/profit_sharing.proto) | `ProfitSharingService` |
| Persistence | [internal/profitsharing/store](../../../internal/profitsharing/store) | migrations, queries, transactional store |
| Administrator UI | [ui/src/app/pages/profit-sharing-admin.tsx](../../../ui/src/app/pages/profit-sharing-admin.tsx) | `ProfitSharingAdminPage`, server-side participant search |
| Member UI | [ui/src/app/pages/profit-sharing.tsx](../../../ui/src/app/pages/profit-sharing.tsx) | `ProfitSharingPage` |

## Architecture

Profit Sharing state remains in its own PostgreSQL database. The API Server
facade composes the authenticated Athena account and current access snapshot
before calling the domain service. Administrator lifecycle RPCs use the fixed
administrator boundary. Member RPCs require current login, independent Profit
Sharing entitlement, and the round's durable member relationship.

The participant selector queries the Account API with server-side search and
`profitSharingEligibleOnly=true`. Results include only non-administrator accounts
whose login and Profit Sharing access are enabled; labels prefer Athena profile
display name and verified email.

The public service contains no trusted requester or administrator fields. The
facade derives those values from the authenticated context and supplies the
internal `requester_account` and `requester_is_admin` fields. The domain service
then repeats membership, actor-role, phase, ownership, and ballot checks. Public
rounds are requester-specific flattened projections; candidate authors and live
vote totals remain hidden until their defined lifecycle stage.

## Runtime Flow

1. An administrator builds a draft roster from the durable Athena account
   directory using dynamically registered account IDs.
2. `OpenRound` revalidates the complete roster immediately before the state
   transition. Every participant must still exist, be an ordinary account, have
   a permanent Google binding, have login enabled, and have Profit Sharing
   enabled. Missing, duplicate, blocked, unbound, administrator, or unauthorized
   participants reject the open operation.
3. The Profit Sharing service compares the validated set with the complete
   durable draft roster in its transaction and persists the immutable opened
   snapshot.
4. Member reads and mutations pass through account authorization first and then
   domain membership checks. Entitlement alone cannot read a round in which the
   account is not a participant; membership alone cannot bypass a revoked
   entitlement.
5. Administrator lifecycle operations retain the unique fixed-admin boundary.
   Administrators do not submit ordinary participant proposals or votes.
6. Disabling login blocks the user's session and API Keys and therefore Profit
   Sharing immediately. Disabling only Profit Sharing blocks member operations
   without changing module access, API Keys, or round history.
7. The standalone service starts on port `8108`, marks gRPC health serving after
   its database is ready, and is reached through one reusable API Server
   clientset. An administrator may create a blank draft and replace its slug,
   title, and five participant definitions using the round revision.
8. Opening a valid draft locks it, verifies the revision, title, participant
   metadata, and exact equality with the facade-validated five-account set, then
   creates one proposal and five proposal items for every participant and enters
   `COLLECTING` in one transaction.
9. During collection, a participant reads and revises only their own proposal.
   Submission requires every responsibility and allocation with exactly 10,000
   basis points in total. A proposal may be reopened before publication. The
   administrator sees submission progress but not unpublished contents.
10. After all five submissions, publication assigns stable anonymous labels,
    creates the first ballot with every proposal, and enters `VOTING`.
    Participants can see allocation contents but not authors or live tallies.
11. A vote targets the current ballot and is upserted for that participant. The
    transaction rejects self-voting and rechecks that the ballot is still active,
    so a racing ballot transition cannot redirect a vote.
12. Closing requires all five votes. A unique leader closes the round and records
    the winner. A tie closes the current ballot and atomically opens a runoff
    containing only the highest-tied candidates. The process repeats until one
    proposal wins.
13. A closed projection exposes only final-ballot candidates, aggregate results,
    and the winner; it does not expose eliminated-candidate or cross-ballot
    history. Shutdown marks health not serving, stops gRPC, and closes the pool.

## State / Data

Rounds, rosters, snapshots, proposals, and votes are durable Profit Sharing
rows. Participant identity is the immutable Athena internal account name such as
`user-<UUID>`, never email or Google subject. Dynamic accounts are permanent;
blocking one does not rewrite historical round records.

| Table | Durable responsibility |
| --- | --- |
| `profit_sharing_round` | Unique slug, title, `draft` / `collecting` / `voting` / `closed` phase, revision, active ballot, winner, and lifecycle times. |
| `profit_sharing_participant` | Round-local Athena account, display name, stable order, and baseline responsibility. |
| `profit_sharing_proposal` | One UUID proposal per participant, draft/submitted state, blind label, revision, author, and timestamps. |
| `profit_sharing_proposal_item` | One responsibility and nullable basis-point allocation per proposal and participant. |
| `profit_sharing_ballot` | Numbered open or closed ballots. |
| `profit_sharing_ballot_candidate` | Candidate subset for an initial or runoff ballot. |
| `profit_sharing_vote` | One upsertable selection per ballot and voter, with author data for database self-vote enforcement. |

The independent entitlement lives in account-state `account_access`, not the
Profit Sharing database. There is no process-local workflow cache or background
scheduler.

Round and proposal revisions provide optimistic concurrency. Multi-row
mutations lock affected rows and commit atomically. Reads assemble a requester-
specific durable snapshot; restart requires no in-memory workflow recovery.

## Configuration

Profit Sharing account eligibility has no per-user environment variables. The
administrator and ordinary participants originate from the durable Athena
account directory.

| Setting | Behavior |
| --- | --- |
| `ATHENA_PROFIT_SHARING_LISTEN_ADDRESS` | Listener address, default `0.0.0.0`. |
| `ATHENA_PROFIT_SHARING_LISTEN_PORT` | Listener port, default `8108`; Procfile may supply `--port` from `ATHENA_PROFIT_SHARING_PORT`. |
| `ATHENA_PROFIT_SHARING_POSTGRES_DSN` | Connection for the owned `profit_sharing` database. |
| `ATHENA_POSTGRES_AUTO_MIGRATE` | Enables embedded local migration; production uses the migration command before services start. |
| `ATHENA_PROFIT_SHARING_SERVER_ADDRESS` | API Server internal gRPC target, default `127.0.0.1:8108` and service DNS in Compose. |

A fresh deployment/reset starts with only the unbound administrator account and
no ordinary participants. Users must log in and receive explicit Profit Sharing
access before an administrator can open a round containing them.

## Invariants

- A member RPC requires current login, Profit Sharing entitlement, and current
  round membership.
- `OpenRound` accepts only registered, Google-bound, login-enabled,
  Profit-Sharing-enabled ordinary accounts.
- Email and Google subject are never roster identity.
- The fixed administrator owns lifecycle actions and cannot act as an ordinary
  participant.
- Revoking access does not delete or rewrite durable round history.
- API Key access and product-module access do not imply Profit Sharing access.
- A proposal always contains one item for every participant; submission requires
  complete responsibilities, complete shares, and exactly 10,000 basis points.
- Collection is author-private, voting uses blind labels, and author/tally
  visibility follows phase. A participant cannot vote for their own proposal.
- Closing requires all five participant votes; a tie creates a runoff containing
  exactly the highest-tied candidates.
- Round/proposal revisions, row locks, foreign keys, uniqueness constraints, and
  database self-vote enforcement remain aligned with service checks.

## Failure Recovery

An eligibility failure occurs before the open request reaches the domain state
transition. A concurrent access change is evaluated from the current API Server
snapshot at validation time; subsequent member requests always recheck current
access. Domain transaction failure rolls back round, roster, proposal, vote, or
snapshot changes according to the service's transaction boundary.

Account database unavailability fails eligibility closed. It does not alter an
existing Profit Sharing round. All multi-row domain mutations roll back on
validation, query, cancellation, or commit failure. Revision mismatches return
`Aborted`; phase/precondition failures return `FailedPrecondition`; malformed
input returns `InvalidArgument`; membership and self-vote failures return
`PermissionDenied`.

Facade eligibility validation and the domain transition are separate calls. The
expected round revision and exact roster comparison prevent a concurrent roster
edit from opening stale data. A later entitlement change is not distributed in
that transaction, so every subsequent member request rechecks current access.
The reconnecting internal client leaves durable state unchanged while the
service is unavailable.

## Observability

Authorization denials distinguish account login, Profit Sharing entitlement,
and domain membership failures. Logs use Athena account IDs and round/domain
identifiers; Google subjects, JWTs, and API Key JTIs are excluded. The standalone
process publishes standard gRPC health, and Service Status reports an unreachable
dependency. Durable timestamps and revisions are the workflow diagnostic record;
there is no separate workflow metrics endpoint.

## Change Checklist

- [ ] Independent entitlement and membership composition remain current.
- [ ] Open-round participant revalidation remains current.
- [ ] Dynamic account search and labels remain current.
- [ ] Durable workflow state and reset semantics remain current.
- [ ] The [design index](../README.md) contains the current summary.
