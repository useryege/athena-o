package store

import (
	"github.com/useryege/athena/internal/serviceschema"
	"github.com/useryege/athena/util/db/postgres"
)

// Schema declares the existing migration and columns consumed by this service.
func Schema() serviceschema.Spec {
	return serviceschema.Spec{Module: "profit-sharing", DSN: func() string { return postgres.DSN("ATHENA_PROFIT_SHARING_POSTGRES_DSN", "profit_sharing") }, Migrations: migrations, Relations: map[string][]string{
		"public.profit_sharing_round":            {"id", "slug", "title", "phase", "revision", "active_ballot_number", "final_proposal_id", "created_at", "updated_at", "opened_at", "published_at", "closed_at"},
		"public.profit_sharing_participant":      {"round_id", "account_id", "username", "display_name", "display_order", "baseline_responsibility"},
		"public.profit_sharing_proposal":         {"id", "round_id", "author_account_id", "status", "anonymous_label", "revision", "created_at", "updated_at", "submitted_at"},
		"public.profit_sharing_proposal_item":    {"round_id", "proposal_id", "participant_account_id", "responsibility", "basis_points"},
		"public.profit_sharing_ballot":           {"round_id", "ballot_number", "status", "created_at", "closed_at"},
		"public.profit_sharing_ballot_candidate": {"round_id", "ballot_number", "proposal_id"},
		"public.profit_sharing_vote":             {"round_id", "ballot_number", "voter_account_id", "proposal_id", "proposal_author_account_id", "created_at", "updated_at"},
	}}
}
