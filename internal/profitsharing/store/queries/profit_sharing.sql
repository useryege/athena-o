-- name: CreateProfitSharingRound :one
INSERT INTO profit_sharing_round (slug, title)
VALUES (sqlc.arg(slug)::text, sqlc.arg(title)::text)
RETURNING *;

-- name: GetProfitSharingRound :one
SELECT *
FROM profit_sharing_round
WHERE id = sqlc.arg(round_id)::bigint;

-- name: GetProfitSharingRoundBySlug :one
SELECT *
FROM profit_sharing_round
WHERE slug = sqlc.arg(slug)::text;

-- name: GetProfitSharingRoundForUpdate :one
SELECT *
FROM profit_sharing_round
WHERE id = sqlc.arg(round_id)::bigint
FOR UPDATE;

-- name: GetProfitSharingRoundBySlugForUpdate :one
SELECT *
FROM profit_sharing_round
WHERE slug = sqlc.arg(slug)::text
FOR UPDATE;

-- name: ListProfitSharingRounds :many
SELECT round.*
FROM profit_sharing_round AS round
WHERE sqlc.arg(requester_is_admin)::boolean
   OR EXISTS (
     SELECT 1
     FROM profit_sharing_participant AS participant
     WHERE participant.round_id = round.id
       AND participant.account = sqlc.arg(requester_account)::text
   )
ORDER BY round.created_at DESC, round.id DESC;

-- name: UpdateProfitSharingRoundConfig :one
UPDATE profit_sharing_round
SET slug = sqlc.arg(slug)::text,
    title = sqlc.arg(title)::text,
    revision = revision + 1,
    updated_at = NOW()
WHERE id = sqlc.arg(round_id)::bigint
  AND phase = 'draft'
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: DeleteProfitSharingParticipants :exec
DELETE FROM profit_sharing_participant
WHERE round_id = sqlc.arg(round_id)::bigint;

-- name: BatchCreateProfitSharingParticipants :exec
INSERT INTO profit_sharing_participant (
  round_id,
  account,
  display_name,
  display_order,
  baseline_responsibility
)
SELECT
  sqlc.arg(round_id)::bigint,
  unnest(sqlc.arg(accounts)::text[]),
  unnest(sqlc.arg(display_names)::text[]),
  unnest(sqlc.arg(display_orders)::integer[]),
  unnest(sqlc.arg(baseline_responsibilities)::text[]);

-- name: ListProfitSharingParticipants :many
SELECT *
FROM profit_sharing_participant
WHERE round_id = sqlc.arg(round_id)::bigint
ORDER BY display_order, account;

-- name: GetProfitSharingParticipant :one
SELECT *
FROM profit_sharing_participant
WHERE round_id = sqlc.arg(round_id)::bigint
  AND account = sqlc.arg(account)::text;

-- name: OpenProfitSharingRound :one
UPDATE profit_sharing_round
SET phase = 'collecting',
    revision = revision + 1,
    updated_at = NOW(),
    opened_at = NOW()
WHERE id = sqlc.arg(round_id)::bigint
  AND phase = 'draft'
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: BatchCreateProfitSharingProposals :exec
INSERT INTO profit_sharing_proposal (id, round_id, author_account)
SELECT
  unnest(sqlc.arg(proposal_ids)::text[]),
  sqlc.arg(round_id)::bigint,
  unnest(sqlc.arg(author_accounts)::text[]);

-- name: CreateInitialProfitSharingProposalItems :exec
INSERT INTO profit_sharing_proposal_item (
  round_id,
  proposal_id,
  participant_account,
  responsibility,
  basis_points
)
SELECT
  proposal.round_id,
  proposal.id,
  participant.account,
  participant.baseline_responsibility,
  NULL
FROM profit_sharing_proposal AS proposal
JOIN profit_sharing_participant AS participant
  ON participant.round_id = proposal.round_id
WHERE proposal.round_id = sqlc.arg(round_id)::bigint;

-- name: GetProfitSharingProposal :one
SELECT *
FROM profit_sharing_proposal
WHERE id = sqlc.arg(proposal_id)::text
  AND round_id = sqlc.arg(round_id)::bigint;

-- name: GetProfitSharingProposalByAuthor :one
SELECT *
FROM profit_sharing_proposal
WHERE round_id = sqlc.arg(round_id)::bigint
  AND author_account = sqlc.arg(author_account)::text;

-- name: GetProfitSharingProposalByAuthorForUpdate :one
SELECT *
FROM profit_sharing_proposal
WHERE round_id = sqlc.arg(round_id)::bigint
  AND author_account = sqlc.arg(author_account)::text
FOR UPDATE;

-- name: ListProfitSharingProposals :many
SELECT *
FROM profit_sharing_proposal
WHERE round_id = sqlc.arg(round_id)::bigint
ORDER BY anonymous_label NULLS LAST, id;

-- name: ListProfitSharingProposalItems :many
SELECT item.*
FROM profit_sharing_proposal_item AS item
JOIN profit_sharing_participant AS participant
  ON participant.round_id = item.round_id
 AND participant.account = item.participant_account
WHERE item.round_id = sqlc.arg(round_id)::bigint
  AND item.proposal_id = sqlc.arg(proposal_id)::text
ORDER BY participant.display_order, participant.account;

-- name: DeleteProfitSharingProposalItems :exec
DELETE FROM profit_sharing_proposal_item
WHERE round_id = sqlc.arg(round_id)::bigint
  AND proposal_id = sqlc.arg(proposal_id)::text;

-- name: BatchCreateProfitSharingProposalItems :exec
INSERT INTO profit_sharing_proposal_item (
  round_id,
  proposal_id,
  participant_account,
  responsibility,
  basis_points
)
SELECT
  sqlc.arg(round_id)::bigint,
  sqlc.arg(proposal_id)::text,
  unnest(sqlc.arg(participant_accounts)::text[]),
  unnest(sqlc.arg(responsibilities)::text[]),
  NULLIF(unnest(sqlc.arg(basis_points_values)::integer[]), -1);

-- name: UpdateProfitSharingProposalDraft :one
UPDATE profit_sharing_proposal
SET revision = revision + 1,
    updated_at = NOW()
WHERE id = sqlc.arg(proposal_id)::text
  AND round_id = sqlc.arg(round_id)::bigint
  AND author_account = sqlc.arg(author_account)::text
  AND status = 'draft'
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: SubmitProfitSharingProposal :one
UPDATE profit_sharing_proposal
SET status = 'submitted',
    revision = revision + 1,
    updated_at = NOW(),
    submitted_at = NOW()
WHERE id = sqlc.arg(proposal_id)::text
  AND round_id = sqlc.arg(round_id)::bigint
  AND author_account = sqlc.arg(author_account)::text
  AND status = 'draft'
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: ReopenProfitSharingProposal :one
UPDATE profit_sharing_proposal
SET status = 'draft',
    revision = revision + 1,
    updated_at = NOW(),
    submitted_at = NULL
WHERE id = sqlc.arg(proposal_id)::text
  AND round_id = sqlc.arg(round_id)::bigint
  AND author_account = sqlc.arg(author_account)::text
  AND status = 'submitted'
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: CountSubmittedProfitSharingProposals :one
SELECT COUNT(*)::bigint
FROM profit_sharing_proposal
WHERE round_id = sqlc.arg(round_id)::bigint
  AND status = 'submitted';

-- name: AssignProfitSharingProposalLabel :one
UPDATE profit_sharing_proposal
SET anonymous_label = sqlc.arg(anonymous_label)::text,
    updated_at = NOW()
WHERE id = sqlc.arg(proposal_id)::text
  AND round_id = sqlc.arg(round_id)::bigint
  AND status = 'submitted'
  AND anonymous_label IS NULL
RETURNING *;

-- name: CreateProfitSharingBallot :one
INSERT INTO profit_sharing_ballot (round_id, ballot_number)
VALUES (sqlc.arg(round_id)::bigint, sqlc.arg(ballot_number)::integer)
RETURNING *;

-- name: GetActiveProfitSharingBallotForUpdate :one
SELECT ballot.*
FROM profit_sharing_ballot AS ballot
JOIN profit_sharing_round AS round
  ON round.id = ballot.round_id
 AND round.active_ballot_number = ballot.ballot_number
WHERE ballot.round_id = sqlc.arg(round_id)::bigint
FOR UPDATE OF ballot;

-- name: BatchCreateProfitSharingBallotCandidates :exec
INSERT INTO profit_sharing_ballot_candidate (round_id, ballot_number, proposal_id)
SELECT
  sqlc.arg(round_id)::bigint,
  sqlc.arg(ballot_number)::integer,
  proposal_id
FROM unnest(sqlc.arg(proposal_ids)::text[]) AS proposal_id;

-- name: ListProfitSharingBallotCandidates :many
SELECT proposal.*
FROM profit_sharing_ballot_candidate AS candidate
JOIN profit_sharing_proposal AS proposal
  ON proposal.id = candidate.proposal_id
 AND proposal.round_id = candidate.round_id
WHERE candidate.round_id = sqlc.arg(round_id)::bigint
  AND candidate.ballot_number = sqlc.arg(ballot_number)::integer
ORDER BY proposal.anonymous_label, proposal.id;

-- name: PublishProfitSharingRound :one
UPDATE profit_sharing_round
SET phase = 'voting',
    active_ballot_number = 1,
    revision = revision + 1,
    updated_at = NOW(),
    published_at = NOW()
WHERE id = sqlc.arg(round_id)::bigint
  AND phase = 'collecting'
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: GetProfitSharingVote :one
SELECT *
FROM profit_sharing_vote
WHERE round_id = sqlc.arg(round_id)::bigint
  AND ballot_number = sqlc.arg(ballot_number)::integer
  AND voter_account = sqlc.arg(voter_account)::text;

-- name: UpsertProfitSharingVote :one
INSERT INTO profit_sharing_vote (
  round_id,
  ballot_number,
  voter_account,
  proposal_id,
  proposal_author_account
)
VALUES (
  sqlc.arg(round_id)::bigint,
  sqlc.arg(ballot_number)::integer,
  sqlc.arg(voter_account)::text,
  sqlc.arg(proposal_id)::text,
  sqlc.arg(proposal_author_account)::text
)
ON CONFLICT (round_id, ballot_number, voter_account) DO UPDATE
SET proposal_id = EXCLUDED.proposal_id,
    proposal_author_account = EXCLUDED.proposal_author_account,
    updated_at = NOW()
RETURNING *;

-- name: CountProfitSharingBallotVotes :one
SELECT COUNT(*)::bigint
FROM profit_sharing_vote
WHERE round_id = sqlc.arg(round_id)::bigint
  AND ballot_number = sqlc.arg(ballot_number)::integer;

-- name: ListProfitSharingBallotTallies :many
SELECT
  candidate.proposal_id,
  COUNT(vote.voter_account)::bigint AS vote_count
FROM profit_sharing_ballot_candidate AS candidate
LEFT JOIN profit_sharing_vote AS vote
  ON vote.round_id = candidate.round_id
 AND vote.ballot_number = candidate.ballot_number
 AND vote.proposal_id = candidate.proposal_id
WHERE candidate.round_id = sqlc.arg(round_id)::bigint
  AND candidate.ballot_number = sqlc.arg(ballot_number)::integer
GROUP BY candidate.proposal_id
ORDER BY vote_count DESC, candidate.proposal_id;

-- name: CloseProfitSharingBallot :one
UPDATE profit_sharing_ballot
SET status = 'closed',
    closed_at = NOW()
WHERE round_id = sqlc.arg(round_id)::bigint
  AND ballot_number = sqlc.arg(ballot_number)::integer
  AND status = 'open'
RETURNING *;

-- name: AdvanceProfitSharingRunoffBallot :one
UPDATE profit_sharing_round
SET active_ballot_number = sqlc.arg(next_ballot_number)::integer,
    revision = revision + 1,
    updated_at = NOW()
WHERE id = sqlc.arg(round_id)::bigint
  AND phase = 'voting'
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: CloseProfitSharingRound :one
UPDATE profit_sharing_round
SET phase = 'closed',
    final_proposal_id = sqlc.arg(final_proposal_id)::text,
    revision = revision + 1,
    updated_at = NOW(),
    closed_at = NOW()
WHERE id = sqlc.arg(round_id)::bigint
  AND phase = 'voting'
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: ListProfitSharingBallots :many
SELECT *
FROM profit_sharing_ballot
WHERE round_id = sqlc.arg(round_id)::bigint
ORDER BY ballot_number;

-- name: ListProfitSharingClosedTallies :many
SELECT
  ballot.ballot_number,
  proposal.id AS proposal_id,
  proposal.author_account,
  proposal.anonymous_label,
  COUNT(vote.voter_account)::bigint AS vote_count
FROM profit_sharing_ballot AS ballot
JOIN profit_sharing_ballot_candidate AS candidate
  ON candidate.round_id = ballot.round_id
 AND candidate.ballot_number = ballot.ballot_number
JOIN profit_sharing_proposal AS proposal
  ON proposal.id = candidate.proposal_id
 AND proposal.round_id = candidate.round_id
LEFT JOIN profit_sharing_vote AS vote
  ON vote.round_id = candidate.round_id
 AND vote.ballot_number = candidate.ballot_number
 AND vote.proposal_id = candidate.proposal_id
WHERE ballot.round_id = sqlc.arg(round_id)::bigint
  AND ballot.status = 'closed'
GROUP BY ballot.ballot_number, proposal.id, proposal.author_account, proposal.anonymous_label
ORDER BY ballot.ballot_number, vote_count DESC, proposal.anonymous_label, proposal.id;
