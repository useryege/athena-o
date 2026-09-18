-- +goose Up
CREATE SCHEMA IF NOT EXISTS operation_log;
CREATE TABLE operation_log.event (
 event_id uuid PRIMARY KEY,
 ingest_id bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
 operation_id uuid NOT NULL,
 phase text NOT NULL CHECK (phase IN ('START','FINISH')),
 producer_id uuid NOT NULL,
 schema_version integer NOT NULL CHECK (schema_version > 0),
 occurred_at timestamptz NOT NULL,
 received_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 payload jsonb NOT NULL CHECK (octet_length(payload::text) <= 32768),
 payload_hash bytea NOT NULL CHECK (octet_length(payload_hash)=32),
 UNIQUE(operation_id,phase)
);
CREATE INDEX event_received_idx ON operation_log.event(received_at,event_id);
CREATE TABLE operation_log.delivery (
 event_id uuid PRIMARY KEY REFERENCES operation_log.event(event_id),
 state text NOT NULL DEFAULT 'PENDING' CHECK (state IN ('PENDING','PROCESSED','QUARANTINED')),
 failure_count bigint NOT NULL DEFAULT 0 CHECK (failure_count>=0),
 next_attempt_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 processed_at timestamptz,
 reason_code text
);
CREATE INDEX delivery_pending_idx ON operation_log.delivery(next_attempt_at,event_id) WHERE state='PENDING';
CREATE INDEX delivery_quarantined_idx ON operation_log.delivery(event_id) WHERE state='QUARANTINED';
CREATE TABLE operation_log.publication (
 singleton_id integer PRIMARY KEY CHECK(singleton_id=1),
 last_seq bigint NOT NULL DEFAULT 0 CHECK(last_seq>=0),
 last_published_at timestamptz
);
INSERT INTO operation_log.publication(singleton_id) VALUES(1);
CREATE TABLE operation_log.entry_version (
 operation_id uuid NOT NULL,
 visible_from_seq bigint NOT NULL CHECK(visible_from_seq>0),
 visible_to_seq bigint CHECK(visible_to_seq>visible_from_seq),
 started_at timestamptz NOT NULL,
 finished_at timestamptz,
 actor_account_id uuid,
 actor_username text,
 actor_role text NOT NULL CHECK(actor_role IN ('MEMBER','ADMINISTRATOR','UNKNOWN')),
 realm text NOT NULL CHECK(realm IN ('MEMBER','ADMIN','UNKNOWN')),
 credential_kind text NOT NULL CHECK(credential_kind IN ('LOGIN_SESSION','API_KEY','DEVELOPMENT','UNAUTHENTICATED')),
 module_code text NOT NULL,
 action_code text NOT NULL,
 outcome text NOT NULL CHECK(outcome IN ('UNKNOWN','SUCCEEDED','ACCEPTED','FAILED','DENIED','PARTIAL','ACTION_REQUIRED','CANCELLED')),
 observation text NOT NULL CHECK(observation IN ('START_ONLY','FINISH_ONLY','COMPLETE')),
 target_account_id uuid,
 primary_resource_type text,
 primary_resource_id text,
 request_id uuid NOT NULL,
 parent_operation_id uuid,
 business_request_id text,
 duration_ms bigint CHECK(duration_ms>=0),
 detail jsonb NOT NULL CHECK (octet_length(detail::text) <= 16384),
 source_event_ids uuid[] NOT NULL,
 PRIMARY KEY(operation_id,visible_from_seq)
);
CREATE INDEX entry_history_idx ON operation_log.entry_version(operation_id,visible_from_seq DESC);
CREATE UNIQUE INDEX entry_current_idx ON operation_log.entry_version(operation_id) WHERE visible_to_seq IS NULL;
CREATE INDEX entry_sort_idx ON operation_log.entry_version(started_at DESC,operation_id DESC,visible_from_seq);
CREATE INDEX entry_actor_idx ON operation_log.entry_version(actor_account_id,started_at DESC,operation_id DESC);
CREATE INDEX entry_username_idx ON operation_log.entry_version(lower(actor_username),started_at DESC,operation_id DESC);
CREATE INDEX entry_username_prefix_idx ON operation_log.entry_version(lower(actor_username) text_pattern_ops,started_at DESC,operation_id DESC);
CREATE INDEX entry_action_idx ON operation_log.entry_version(module_code,action_code,started_at DESC,operation_id DESC);
CREATE INDEX entry_outcome_idx ON operation_log.entry_version(outcome,started_at DESC,operation_id DESC);
CREATE INDEX entry_resource_idx ON operation_log.entry_version(primary_resource_type,primary_resource_id,started_at DESC,operation_id DESC);
CREATE TABLE operation_log.producer_status (
 producer_id uuid PRIMARY KEY,
 started_at timestamptz NOT NULL,
 last_seen_at timestamptz NOT NULL,
 stopped_at timestamptz,
 snapshot_no bigint NOT NULL CHECK(snapshot_no>=0),
 attempted_events bigint NOT NULL CHECK(attempted_events>=0),
 confirmed_events bigint NOT NULL CHECK(confirmed_events>=0),
 unconfirmed_events bigint NOT NULL CHECK(unconfirmed_events>=0),
 invalid_events bigint NOT NULL CHECK(invalid_events>=0),
 capacity_rejected_events bigint NOT NULL CHECK(capacity_rejected_events>=0),
 in_flight_events bigint NOT NULL CHECK(in_flight_events>=0),
 last_failure_at timestamptz,
 last_failure_code text,
 last_recovered_at timestamptz,
 last_confirmed_at timestamptz,
 persistence_reachable boolean,
 CHECK(attempted_events::numeric=confirmed_events::numeric+unconfirmed_events::numeric+invalid_events::numeric+capacity_rejected_events::numeric+in_flight_events::numeric)
);
-- +goose Down
DROP TABLE operation_log.producer_status;
DROP TABLE operation_log.entry_version;
DROP TABLE operation_log.publication;
DROP TABLE operation_log.delivery;
DROP TABLE operation_log.event;
