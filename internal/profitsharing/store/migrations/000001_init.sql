-- +goose Up

CREATE TABLE profit_sharing_round (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  phase TEXT NOT NULL DEFAULT 'draft',
  revision BIGINT NOT NULL DEFAULT 1,
  active_ballot_number INTEGER,
  final_proposal_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  opened_at TIMESTAMPTZ,
  published_at TIMESTAMPTZ,
  closed_at TIMESTAMPTZ,
  CONSTRAINT profit_sharing_round_phase_check
    CHECK (phase IN ('draft', 'collecting', 'voting', 'closed')),
  CONSTRAINT profit_sharing_round_slug_unique
    UNIQUE (slug),
  CONSTRAINT profit_sharing_round_slug_check
    CHECK (BTRIM(slug) <> ''),
  CONSTRAINT profit_sharing_round_revision_check
    CHECK (revision > 0),
  CONSTRAINT profit_sharing_round_active_ballot_check
    CHECK (active_ballot_number IS NULL OR active_ballot_number > 0)
);

CREATE TABLE profit_sharing_participant (
  round_id BIGINT NOT NULL,
  account_id UUID NOT NULL,
  username TEXT NOT NULL,
  display_name TEXT NOT NULL,
  display_order INTEGER NOT NULL,
  baseline_responsibility TEXT NOT NULL DEFAULT '',
  CONSTRAINT profit_sharing_participant_pk
    PRIMARY KEY (round_id, account_id),
  CONSTRAINT profit_sharing_participant_round_fk
    FOREIGN KEY (round_id)
    REFERENCES profit_sharing_round (id)
    ON DELETE CASCADE,
  CONSTRAINT profit_sharing_participant_order_unique
    UNIQUE (round_id, display_order),
  CONSTRAINT profit_sharing_participant_username_check
    CHECK (
      char_length(username) BETWEEN 3 AND 42
      AND username ~ '^[A-Za-z0-9.-]+$'
      AND username ~ '[A-Za-z0-9]'
    ),
  CONSTRAINT profit_sharing_participant_display_name_check
    CHECK (
      char_length(display_name) BETWEEN 1 AND 80
      AND display_name !~ '[[:cntrl:]]'
    ),
  CONSTRAINT profit_sharing_participant_order_check
    CHECK (display_order > 0)
);

CREATE TABLE profit_sharing_proposal (
  id TEXT PRIMARY KEY,
  round_id BIGINT NOT NULL,
  author_account_id UUID NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  anonymous_label TEXT,
  revision BIGINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submitted_at TIMESTAMPTZ,
  CONSTRAINT profit_sharing_proposal_round_author_unique
    UNIQUE (round_id, author_account_id),
  CONSTRAINT profit_sharing_proposal_id_round_unique
    UNIQUE (id, round_id),
  CONSTRAINT profit_sharing_proposal_id_round_author_unique
    UNIQUE (id, round_id, author_account_id),
  CONSTRAINT profit_sharing_proposal_round_author_fk
    FOREIGN KEY (round_id, author_account_id)
    REFERENCES profit_sharing_participant (round_id, account_id)
    ON DELETE CASCADE,
  CONSTRAINT profit_sharing_proposal_label_unique
    UNIQUE (round_id, anonymous_label),
  CONSTRAINT profit_sharing_proposal_status_check
    CHECK (status IN ('draft', 'submitted')),
  CONSTRAINT profit_sharing_proposal_revision_check
    CHECK (revision > 0),
  CONSTRAINT profit_sharing_proposal_label_check
    CHECK (anonymous_label IS NULL OR BTRIM(anonymous_label) <> ''),
  CONSTRAINT profit_sharing_proposal_submission_check
    CHECK (
      (status = 'draft' AND submitted_at IS NULL)
      OR (status = 'submitted' AND submitted_at IS NOT NULL)
    )
);

CREATE TABLE profit_sharing_proposal_item (
  round_id BIGINT NOT NULL,
  proposal_id TEXT NOT NULL,
  participant_account_id UUID NOT NULL,
  responsibility TEXT NOT NULL DEFAULT '',
  basis_points INTEGER,
  CONSTRAINT profit_sharing_proposal_item_pk
    PRIMARY KEY (proposal_id, participant_account_id),
  CONSTRAINT profit_sharing_proposal_item_proposal_fk
    FOREIGN KEY (proposal_id, round_id)
    REFERENCES profit_sharing_proposal (id, round_id)
    ON DELETE CASCADE,
  CONSTRAINT profit_sharing_proposal_item_participant_fk
    FOREIGN KEY (round_id, participant_account_id)
    REFERENCES profit_sharing_participant (round_id, account_id)
    ON DELETE CASCADE,
  CONSTRAINT profit_sharing_proposal_item_basis_points_check
    CHECK (basis_points IS NULL OR basis_points BETWEEN 0 AND 10000)
);

CREATE TABLE profit_sharing_ballot (
  round_id BIGINT NOT NULL,
  ballot_number INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  closed_at TIMESTAMPTZ,
  CONSTRAINT profit_sharing_ballot_pk
    PRIMARY KEY (round_id, ballot_number),
  CONSTRAINT profit_sharing_ballot_round_fk
    FOREIGN KEY (round_id)
    REFERENCES profit_sharing_round (id)
    ON DELETE CASCADE,
  CONSTRAINT profit_sharing_ballot_number_check
    CHECK (ballot_number > 0),
  CONSTRAINT profit_sharing_ballot_status_check
    CHECK (status IN ('open', 'closed')),
  CONSTRAINT profit_sharing_ballot_closed_check
    CHECK (
      (status = 'open' AND closed_at IS NULL)
      OR (status = 'closed' AND closed_at IS NOT NULL)
    )
);

CREATE TABLE profit_sharing_ballot_candidate (
  round_id BIGINT NOT NULL,
  ballot_number INTEGER NOT NULL,
  proposal_id TEXT NOT NULL,
  CONSTRAINT profit_sharing_ballot_candidate_pk
    PRIMARY KEY (round_id, ballot_number, proposal_id),
  CONSTRAINT profit_sharing_ballot_candidate_ballot_fk
    FOREIGN KEY (round_id, ballot_number)
    REFERENCES profit_sharing_ballot (round_id, ballot_number)
    ON DELETE CASCADE,
  CONSTRAINT profit_sharing_ballot_candidate_proposal_fk
    FOREIGN KEY (proposal_id, round_id)
    REFERENCES profit_sharing_proposal (id, round_id)
    ON DELETE CASCADE
);

CREATE TABLE profit_sharing_vote (
  round_id BIGINT NOT NULL,
  ballot_number INTEGER NOT NULL,
  voter_account_id UUID NOT NULL,
  proposal_id TEXT NOT NULL,
  proposal_author_account_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profit_sharing_vote_pk
    PRIMARY KEY (round_id, ballot_number, voter_account_id),
  CONSTRAINT profit_sharing_vote_voter_fk
    FOREIGN KEY (round_id, voter_account_id)
    REFERENCES profit_sharing_participant (round_id, account_id)
    ON DELETE CASCADE,
  CONSTRAINT profit_sharing_vote_candidate_fk
    FOREIGN KEY (round_id, ballot_number, proposal_id)
    REFERENCES profit_sharing_ballot_candidate (round_id, ballot_number, proposal_id)
    ON DELETE CASCADE,
  CONSTRAINT profit_sharing_vote_proposal_author_fk
    FOREIGN KEY (proposal_id, round_id, proposal_author_account_id)
    REFERENCES profit_sharing_proposal (id, round_id, author_account_id)
    ON DELETE CASCADE,
  CONSTRAINT profit_sharing_vote_no_self_vote_check
    CHECK (voter_account_id <> proposal_author_account_id)
);

ALTER TABLE profit_sharing_round
  ADD CONSTRAINT profit_sharing_round_final_proposal_fk
  FOREIGN KEY (final_proposal_id, id)
  REFERENCES profit_sharing_proposal (id, round_id);

CREATE INDEX profit_sharing_round_phase_created_idx
  ON profit_sharing_round (phase, created_at DESC, id DESC);

CREATE INDEX profit_sharing_proposal_round_status_idx
  ON profit_sharing_proposal (round_id, status, id);

CREATE INDEX profit_sharing_vote_candidate_idx
  ON profit_sharing_vote (round_id, ballot_number, proposal_id);

-- +goose Down

ALTER TABLE profit_sharing_round
  DROP CONSTRAINT IF EXISTS profit_sharing_round_final_proposal_fk;

DROP TABLE IF EXISTS profit_sharing_vote;
DROP TABLE IF EXISTS profit_sharing_ballot_candidate;
DROP TABLE IF EXISTS profit_sharing_ballot;
DROP TABLE IF EXISTS profit_sharing_proposal_item;
DROP TABLE IF EXISTS profit_sharing_proposal;
DROP TABLE IF EXISTS profit_sharing_participant;
DROP TABLE IF EXISTS profit_sharing_round;
