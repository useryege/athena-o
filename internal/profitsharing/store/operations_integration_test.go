//go:build integration

package store

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/useryege/athena/internal/testutil/pgtest"
)

type submitProposalFixture struct {
	db                    *pgtest.DB
	store                 *SQLStore
	roundID               int64
	proposalID            string
	slug                  string
	authorAccountID       string
	participantAccountIDs []string
}

func TestSQLStoreSubmitProposalNoRows(t *testing.T) {
	fixture := newSubmitProposalFixture(t)
	ctx := context.Background()
	installSubmitDiscardTrigger(t, ctx, fixture)

	proposal, err := fixture.store.SubmitProposal(ctx, fixture.slug, fixture.authorAccountID, 1)
	if proposal != nil {
		t.Fatalf("submit proposal=%+v, want nil", proposal)
	}
	if !stderrors.Is(err, ErrRevisionConflict) {
		t.Fatalf("submit error=%v, want ErrRevisionConflict", err)
	}
	assertDraftProposalUnchanged(t, ctx, fixture)
}

func TestSQLStoreSubmitProposalQueryError(t *testing.T) {
	fixture := newSubmitProposalFixture(t)
	ctx := context.Background()
	installSubmitProbeTrigger(t, ctx, fixture)

	proposal, err := fixture.store.SubmitProposal(ctx, fixture.slug, fixture.authorAccountID, 1)
	if proposal != nil {
		t.Fatalf("submit proposal=%+v, want nil", proposal)
	}
	var pgErr *pgconn.PgError
	if !stderrors.As(err, &pgErr) {
		t.Fatalf("submit error=%v, want wrapped PostgreSQL error", err)
	}
	if pgErr.Code != "P0001" || pgErr.Message != "profit-sharing submit probe" {
		t.Fatalf("PostgreSQL error=(code=%q message=%q), want (code=%q message=%q)", pgErr.Code, pgErr.Message, "P0001", "profit-sharing submit probe")
	}
	if err == nil || !strings.Contains(err.Error(), "change status for profit sharing proposal") {
		t.Fatalf("submit error=%v, want submit status context", err)
	}
	assertDraftProposalUnchanged(t, ctx, fixture)
}

func TestSQLStoreSubmitProposalSuccess(t *testing.T) {
	fixture := newSubmitProposalFixture(t)
	ctx := context.Background()

	proposal, err := fixture.store.SubmitProposal(ctx, fixture.slug, fixture.authorAccountID, 1)
	if err != nil {
		t.Fatalf("submit proposal: %v", err)
	}
	assertSubmittedProposal(t, ctx, fixture, proposal, 2)
}

func TestSQLStoreReopenProposalSuccess(t *testing.T) {
	fixture := newSubmitProposalFixture(t)
	ctx := context.Background()

	submitted, err := fixture.store.SubmitProposal(ctx, fixture.slug, fixture.authorAccountID, 1)
	if err != nil {
		t.Fatalf("submit proposal before reopen: %v", err)
	}
	if submitted == nil || submitted.Revision != 2 {
		t.Fatalf("submitted proposal=%+v, want revision 2", submitted)
	}
	reopened, err := fixture.store.ReopenProposal(ctx, fixture.slug, fixture.authorAccountID, 2)
	if err != nil {
		t.Fatalf("reopen proposal: %v", err)
	}
	if reopened == nil {
		t.Fatal("reopened proposal=nil, want proposal")
	}
	if reopened.ID != fixture.proposalID || reopened.RoundID != fixture.roundID || reopened.AuthorAccountID != fixture.authorAccountID {
		t.Fatalf("reopened proposal identity=%+v, want proposal %q for round %d author %q", reopened, fixture.proposalID, fixture.roundID, fixture.authorAccountID)
	}
	if reopened.Status != ProposalStatusDraft || reopened.Revision != 3 || reopened.SubmittedAt != nil {
		t.Fatalf("reopened proposal state=(status=%q revision=%d submitted_at=%v), want draft revision 3 without submitted_at", reopened.Status, reopened.Revision, reopened.SubmittedAt)
	}
	assertCompleteProposalItems(t, reopened.Items)
	status, revision, submittedAt, itemCount, completeItemCount, basisPointsTotal := proposalState(t, ctx, fixture)
	if status != ProposalStatusDraft || revision != 3 || submittedAt != nil {
		t.Fatalf("stored proposal state=(status=%q revision=%d submitted_at=%v), want draft revision 3 without submitted_at", status, revision, submittedAt)
	}
	assertCompleteItemState(t, itemCount, completeItemCount, basisPointsTotal)
	assertStoredCompleteProposalItems(t, ctx, fixture)
}

func TestSQLStoreSubmitProposalRevisionConflict(t *testing.T) {
	fixture := newSubmitProposalFixture(t)
	ctx := context.Background()

	proposal, err := fixture.store.SubmitProposal(ctx, fixture.slug, fixture.authorAccountID, 2)
	if proposal != nil {
		t.Fatalf("submit proposal=%+v, want nil", proposal)
	}
	if !stderrors.Is(err, ErrRevisionConflict) {
		t.Fatalf("submit error=%v, want ErrRevisionConflict", err)
	}
	assertDraftProposalUnchanged(t, ctx, fixture)
}

func newSubmitProposalFixture(t *testing.T) submitProposalFixture {
	t.Helper()
	db := pgtest.New(t, Migrations(), "migrations")
	ctx := context.Background()
	fixture := submitProposalFixture{
		db:              db,
		store:           NewSQLStore(db.Pool),
		proposalID:      uuid.NewString(),
		slug:            "submit-proposal",
		authorAccountID: uuid.NewString(),
	}
	if err := db.Pool.QueryRow(ctx, `
		INSERT INTO profit_sharing_round (slug, title, phase, revision)
		VALUES ($1, 'Submit proposal fixture', 'collecting', 1)
		RETURNING id`, fixture.slug).Scan(&fixture.roundID); err != nil {
		t.Fatalf("create collecting round: %v", err)
	}

	fixture.participantAccountIDs = []string{fixture.authorAccountID, uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for index, accountID := range fixture.participantAccountIDs {
		if _, err := db.Pool.Exec(ctx, `
			INSERT INTO profit_sharing_participant (
				round_id, account_id, username, display_name, display_order, baseline_responsibility
			) VALUES ($1, $2, $3, $4, $5, $6)`,
			fixture.roundID, accountID, fmt.Sprintf("member-%d", index+1), fmt.Sprintf("Member %d", index+1), index+1, fmt.Sprintf("responsibility %d", index+1)); err != nil {
			t.Fatalf("create participant %d: %v", index+1, err)
		}
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO profit_sharing_proposal (id, round_id, author_account_id, status, revision)
		VALUES ($1, $2, $3, 'draft', 1)`, fixture.proposalID, fixture.roundID, fixture.authorAccountID); err != nil {
		t.Fatalf("create draft proposal: %v", err)
	}
	for index, accountID := range fixture.participantAccountIDs {
		if _, err := db.Pool.Exec(ctx, `
			INSERT INTO profit_sharing_proposal_item (
				round_id, proposal_id, participant_account_id, responsibility, basis_points
			) VALUES ($1, $2, $3, $4, 2000)`,
			fixture.roundID, fixture.proposalID, accountID, fmt.Sprintf("responsibility %d", index+1)); err != nil {
			t.Fatalf("create proposal item %d: %v", index+1, err)
		}
	}
	return fixture
}

func installSubmitDiscardTrigger(t *testing.T, ctx context.Context, fixture submitProposalFixture) {
	t.Helper()
	if _, err := fixture.db.Pool.Exec(ctx, `
		CREATE FUNCTION discard_profit_sharing_submit() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			RETURN NULL;
		END;
		$$;
		CREATE TRIGGER discard_profit_sharing_submit
		BEFORE UPDATE ON profit_sharing_proposal
		FOR EACH ROW WHEN (NEW.status = 'submitted')
		EXECUTE FUNCTION discard_profit_sharing_submit()`); err != nil {
		t.Fatalf("install discard submit trigger: %v", err)
	}
}

func installSubmitProbeTrigger(t *testing.T, ctx context.Context, fixture submitProposalFixture) {
	t.Helper()
	if _, err := fixture.db.Pool.Exec(ctx, `
		CREATE FUNCTION probe_profit_sharing_submit() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'profit-sharing submit probe' USING ERRCODE = 'P0001';
		END;
		$$;
		CREATE TRIGGER probe_profit_sharing_submit
		BEFORE UPDATE ON profit_sharing_proposal
		FOR EACH ROW WHEN (NEW.status = 'submitted')
		EXECUTE FUNCTION probe_profit_sharing_submit()`); err != nil {
		t.Fatalf("install submit probe trigger: %v", err)
	}
}

func assertDraftProposalUnchanged(t *testing.T, ctx context.Context, fixture submitProposalFixture) {
	t.Helper()
	status, revision, submittedAt, itemCount, completeItemCount, basisPointsTotal := proposalState(t, ctx, fixture)
	if status != ProposalStatusDraft || revision != 1 || submittedAt != nil {
		t.Fatalf("proposal state=(status=%q revision=%d submitted_at=%v), want draft revision 1 without submitted_at", status, revision, submittedAt)
	}
	assertCompleteItemState(t, itemCount, completeItemCount, basisPointsTotal)
	assertStoredCompleteProposalItems(t, ctx, fixture)
}

func assertSubmittedProposal(t *testing.T, ctx context.Context, fixture submitProposalFixture, proposal *Proposal, revision int64) {
	t.Helper()
	if proposal == nil {
		t.Fatal("submitted proposal=nil, want proposal")
	}
	if proposal.ID != fixture.proposalID || proposal.RoundID != fixture.roundID || proposal.AuthorAccountID != fixture.authorAccountID {
		t.Fatalf("submitted proposal identity=%+v, want proposal %q for round %d author %q", proposal, fixture.proposalID, fixture.roundID, fixture.authorAccountID)
	}
	if proposal.Status != ProposalStatusSubmitted || proposal.Revision != revision || proposal.SubmittedAt == nil {
		t.Fatalf("submitted proposal state=(status=%q revision=%d submitted_at=%v), want submitted revision %d with submitted_at", proposal.Status, proposal.Revision, proposal.SubmittedAt, revision)
	}
	assertCompleteProposalItems(t, proposal.Items)
	status, storedRevision, submittedAt, itemCount, completeItemCount, basisPointsTotal := proposalState(t, ctx, fixture)
	if status != ProposalStatusSubmitted || storedRevision != revision || submittedAt == nil {
		t.Fatalf("stored proposal state=(status=%q revision=%d submitted_at=%v), want submitted revision %d with submitted_at", status, storedRevision, submittedAt, revision)
	}
	assertCompleteItemState(t, itemCount, completeItemCount, basisPointsTotal)
	assertStoredCompleteProposalItems(t, ctx, fixture)
}

func assertCompleteProposalItems(t *testing.T, items []ProposalItem) {
	t.Helper()
	if len(items) != 5 {
		t.Fatalf("proposal item count=%d, want 5", len(items))
	}
	for index, item := range items {
		if item.Responsibility != fmt.Sprintf("responsibility %d", index+1) || item.BasisPoints == nil || *item.BasisPoints != 2000 {
			t.Fatalf("proposal item %d=%+v, want responsibility %q and 2000 basis points", index+1, item, fmt.Sprintf("responsibility %d", index+1))
		}
	}
}

func assertCompleteItemState(t *testing.T, itemCount, completeItemCount, basisPointsTotal int) {
	t.Helper()
	if itemCount != 5 || completeItemCount != 5 || basisPointsTotal != 10000 {
		t.Fatalf("proposal items=(count=%d complete=%d total_basis_points=%d), want (5, 5, 10000)", itemCount, completeItemCount, basisPointsTotal)
	}
}

func assertStoredCompleteProposalItems(t *testing.T, ctx context.Context, fixture submitProposalFixture) {
	t.Helper()
	rows, err := fixture.db.Pool.Query(ctx, `
		SELECT item.participant_account_id::text, item.responsibility, item.basis_points
		FROM profit_sharing_proposal_item AS item
		JOIN profit_sharing_participant AS participant
		  ON participant.round_id = item.round_id
		 AND participant.account_id = item.participant_account_id
		WHERE item.round_id = $1
		  AND item.proposal_id = $2
		ORDER BY participant.display_order, participant.account_id`, fixture.roundID, fixture.proposalID)
	if err != nil {
		t.Fatalf("list stored proposal items: %v", err)
	}
	defer rows.Close()

	index := 0
	for rows.Next() {
		var (
			accountID      string
			responsibility string
			basisPoints    int32
		)
		if err := rows.Scan(&accountID, &responsibility, &basisPoints); err != nil {
			t.Fatalf("scan stored proposal item %d: %v", index+1, err)
		}
		if index >= len(fixture.participantAccountIDs) {
			t.Fatalf("stored proposal item %d=%q, want exactly five items", index+1, accountID)
		}
		if accountID != fixture.participantAccountIDs[index] || responsibility != fmt.Sprintf("responsibility %d", index+1) || basisPoints != 2000 {
			t.Fatalf("stored proposal item %d=(account_id=%q responsibility=%q basis_points=%d), want (%q, %q, 2000)", index+1, accountID, responsibility, basisPoints, fixture.participantAccountIDs[index], fmt.Sprintf("responsibility %d", index+1))
		}
		index++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate stored proposal items: %v", err)
	}
	if index != len(fixture.participantAccountIDs) {
		t.Fatalf("stored proposal item count=%d, want %d", index, len(fixture.participantAccountIDs))
	}
}

func proposalState(t *testing.T, ctx context.Context, fixture submitProposalFixture) (string, int64, *time.Time, int, int, int) {
	t.Helper()
	var (
		status            string
		revision          int64
		submittedAt       *time.Time
		itemCount         int
		completeItemCount int
		basisPointsTotal  int
	)
	if err := fixture.db.Pool.QueryRow(ctx, `
		SELECT proposal.status,
		       proposal.revision,
		       proposal.submitted_at,
		       COUNT(item.*),
		       COUNT(*) FILTER (WHERE BTRIM(item.responsibility) <> '' AND item.basis_points IS NOT NULL),
		       COALESCE(SUM(item.basis_points), 0)
		FROM profit_sharing_proposal AS proposal
		LEFT JOIN profit_sharing_proposal_item AS item
		  ON item.round_id = proposal.round_id
		 AND item.proposal_id = proposal.id
		WHERE proposal.id = $1
		  AND proposal.round_id = $2
		GROUP BY proposal.status, proposal.revision, proposal.submitted_at`, fixture.proposalID, fixture.roundID).Scan(
		&status, &revision, &submittedAt, &itemCount, &completeItemCount, &basisPointsTotal,
	); err != nil {
		t.Fatalf("read proposal state: %v", err)
	}
	return status, revision, submittedAt, itemCount, completeItemCount, basisPointsTotal
}
