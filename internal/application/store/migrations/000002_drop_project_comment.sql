-- +goose Up

DROP TABLE IF EXISTS project_comment;

-- +goose Down

CREATE TABLE IF NOT EXISTS project_comment (
  id BIGSERIAL PRIMARY KEY,
  project_contract BYTEA NOT NULL,
  username TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT project_comment_project_contract_len CHECK (length(project_contract) = 20),
  CONSTRAINT project_comment_username_not_empty CHECK (length(btrim(username)) > 0),
  CONSTRAINT project_comment_content_not_empty CHECK (length(btrim(content)) > 0),
  CONSTRAINT project_comment_content_max_len CHECK (char_length(content) <= 1000),
  CONSTRAINT project_comment_project_fk FOREIGN KEY (project_contract) REFERENCES project(contract)
);

CREATE INDEX IF NOT EXISTS project_comment_project_timeline_idx
  ON project_comment (project_contract, created_at DESC, id DESC);
