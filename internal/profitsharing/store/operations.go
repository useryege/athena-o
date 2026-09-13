package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	profitsharingsqlc "github.com/useryege/athena/internal/profitsharing/store/sqlc"
)

var profitSharingRoundSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (s *SQLStore) CreateRound(ctx context.Context, slug, title string, participants []ParticipantInput) (*Round, error) {
	slug = strings.TrimSpace(slug)
	title = strings.TrimSpace(title)
	if !profitSharingRoundSlugPattern.MatchString(slug) {
		return nil, fmt.Errorf("%w: slug must contain lowercase letters, numbers, and single hyphens", ErrInvalidParticipants)
	}
	participants, err := validateParticipantInputs(participants)
	if err != nil {
		return nil, err
	}
	var created profitsharingsqlc.ProfitSharingRound
	err = s.withTx(ctx, func(queries *profitsharingsqlc.Queries) error {
		row, err := queries.CreateProfitSharingRound(ctx, profitsharingsqlc.CreateProfitSharingRoundParams{
			Slug:  slug,
			Title: title,
		})
		if err != nil {
			if isConstraint(err, "profit_sharing_round_slug_unique") {
				return ErrRoundAlreadyExists
			}
			return fmt.Errorf("create profit sharing round: %w", err)
		}
		if err := createParticipants(ctx, queries, row.ID, participants); err != nil {
			return err
		}
		created = row
		return nil
	})
	if err != nil {
		return nil, err
	}
	converted := roundFromSQLC(created)
	return &converted, nil
}

func (s *SQLStore) UpdateRound(ctx context.Context, currentSlug, nextSlug, title string, participants []ParticipantInput, expectedRevision int64) (*Round, error) {
	currentSlug = strings.TrimSpace(currentSlug)
	nextSlug = strings.TrimSpace(nextSlug)
	title = strings.TrimSpace(title)
	if currentSlug == "" || !profitSharingRoundSlugPattern.MatchString(nextSlug) {
		return nil, fmt.Errorf("%w: current_slug is required and slug must contain lowercase letters, numbers, and single hyphens", ErrInvalidParticipants)
	}
	participants, err := validateParticipantInputs(participants)
	if err != nil {
		return nil, err
	}
	var updated profitsharingsqlc.ProfitSharingRound
	err = s.withTx(ctx, func(queries *profitsharingsqlc.Queries) error {
		row, err := lockRoundBySlug(ctx, queries, currentSlug)
		if err != nil {
			return err
		}
		if row.Phase != RoundPhaseDraft {
			return ErrInvalidPhase
		}
		if row.Revision != expectedRevision {
			return ErrRevisionConflict
		}
		updated, err = queries.UpdateProfitSharingRoundConfig(ctx, profitsharingsqlc.UpdateProfitSharingRoundConfigParams{
			Slug:             nextSlug,
			Title:            title,
			RoundID:          row.ID,
			ExpectedRevision: expectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRevisionConflict
		}
		if err != nil {
			if isConstraint(err, "profit_sharing_round_slug_unique") {
				return ErrRoundAlreadyExists
			}
			return fmt.Errorf("update profit sharing round %q: %w", currentSlug, err)
		}
		if err := queries.DeleteProfitSharingParticipants(ctx, row.ID); err != nil {
			return fmt.Errorf("delete participants for profit sharing round %q: %w", currentSlug, err)
		}
		return createParticipants(ctx, queries, row.ID, participants)
	})
	if err != nil {
		return nil, err
	}
	converted := roundFromSQLC(updated)
	return &converted, nil
}

func (s *SQLStore) OpenRound(ctx context.Context, slug string, expectedRevision int64, validatedAccountIDs []string) (*Round, error) {
	slug = strings.TrimSpace(slug)
	validatedAccountIDs, err := normalizeValidatedAccountIDs(validatedAccountIDs)
	if err != nil {
		return nil, err
	}
	var opened profitsharingsqlc.ProfitSharingRound
	err = s.withTx(ctx, func(queries *profitsharingsqlc.Queries) error {
		row, err := lockRoundBySlug(ctx, queries, slug)
		if err != nil {
			return err
		}
		if row.Phase != RoundPhaseDraft {
			return ErrInvalidPhase
		}
		if row.Revision != expectedRevision {
			return ErrRevisionConflict
		}
		if strings.TrimSpace(row.Title) == "" {
			return fmt.Errorf("%w: round title is required before opening", ErrInvalidParticipants)
		}
		participants, err := queries.ListProfitSharingParticipants(ctx, row.ID)
		if err != nil {
			return fmt.Errorf("list participants for profit sharing round %q: %w", slug, err)
		}
		if err := validateOpenParticipants(participants, validatedAccountIDs); err != nil {
			return err
		}
		proposalIDs := make([]string, len(participants))
		authorAccountIDs := make([]pgtype.UUID, len(participants))
		for i, participant := range participants {
			proposalIDs[i] = uuid.NewString()
			authorAccountIDs[i] = participant.AccountID
		}
		if err := queries.BatchCreateProfitSharingProposals(ctx, profitsharingsqlc.BatchCreateProfitSharingProposalsParams{
			ProposalIds:      proposalIDs,
			RoundID:          row.ID,
			AuthorAccountIds: authorAccountIDs,
		}); err != nil {
			return fmt.Errorf("create proposals for profit sharing round %q: %w", slug, err)
		}
		if err := queries.CreateInitialProfitSharingProposalItems(ctx, row.ID); err != nil {
			return fmt.Errorf("create initial proposal items for profit sharing round %q: %w", slug, err)
		}
		opened, err = queries.OpenProfitSharingRound(ctx, profitsharingsqlc.OpenProfitSharingRoundParams{
			RoundID:          row.ID,
			ExpectedRevision: expectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRevisionConflict
		}
		if err != nil {
			return fmt.Errorf("open profit sharing round %q: %w", slug, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	converted := roundFromSQLC(opened)
	return &converted, nil
}

func (s *SQLStore) UpdateProposal(ctx context.Context, slug, authorAccountID string, expectedRevision int64, items []ProposalItemInput) (*Proposal, error) {
	slug = strings.TrimSpace(slug)
	authorAccountID, err := canonicalAccountID(authorAccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: proposal author account ID is invalid", ErrInvalidProposal)
	}
	authorUUID, err := accountUUID(authorAccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: proposal author account ID is invalid", ErrInvalidProposal)
	}
	var updated profitsharingsqlc.ProfitSharingProposal
	var persistedItems []profitsharingsqlc.ProfitSharingProposalItem
	err = s.withTx(ctx, func(queries *profitsharingsqlc.Queries) error {
		round, err := lockRoundBySlug(ctx, queries, slug)
		if err != nil {
			return err
		}
		if round.Phase != RoundPhaseCollecting {
			return ErrInvalidPhase
		}
		participants, err := queries.ListProfitSharingParticipants(ctx, round.ID)
		if err != nil {
			return fmt.Errorf("list participants for profit sharing round %q: %w", slug, err)
		}
		items, err = validateProposalItems(items, participants)
		if err != nil {
			return err
		}
		proposal, err := lockProposalByAuthor(ctx, queries, round.ID, authorAccountID)
		if err != nil {
			return err
		}
		if proposal.Status != ProposalStatusDraft {
			return ErrInvalidPhase
		}
		if proposal.Revision != expectedRevision {
			return ErrRevisionConflict
		}
		if err := queries.DeleteProfitSharingProposalItems(ctx, profitsharingsqlc.DeleteProfitSharingProposalItemsParams{
			RoundID:    round.ID,
			ProposalID: proposal.ID,
		}); err != nil {
			return fmt.Errorf("delete items for profit sharing proposal %q: %w", proposal.ID, err)
		}
		if err := createProposalItems(ctx, queries, round.ID, proposal.ID, items); err != nil {
			return err
		}
		updated, err = queries.UpdateProfitSharingProposalDraft(ctx, profitsharingsqlc.UpdateProfitSharingProposalDraftParams{
			ProposalID:       proposal.ID,
			RoundID:          round.ID,
			AuthorAccountID:  authorUUID,
			ExpectedRevision: expectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRevisionConflict
		}
		if err != nil {
			return fmt.Errorf("update profit sharing proposal %q: %w", proposal.ID, err)
		}
		persistedItems, err = queries.ListProfitSharingProposalItems(ctx, profitsharingsqlc.ListProfitSharingProposalItemsParams{
			RoundID:    round.ID,
			ProposalID: proposal.ID,
		})
		if err != nil {
			return fmt.Errorf("read updated profit sharing proposal %q: %w", proposal.ID, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	converted := proposalFromSQLC(updated)
	for _, item := range persistedItems {
		converted.Items = append(converted.Items, proposalItemFromSQLC(item))
	}
	return &converted, nil
}

func (s *SQLStore) SubmitProposal(ctx context.Context, slug, authorAccountID string, expectedRevision int64) (*Proposal, error) {
	return s.changeProposalStatus(ctx, slug, authorAccountID, expectedRevision, true)
}

func (s *SQLStore) ReopenProposal(ctx context.Context, slug, authorAccountID string, expectedRevision int64) (*Proposal, error) {
	return s.changeProposalStatus(ctx, slug, authorAccountID, expectedRevision, false)
}

func (s *SQLStore) changeProposalStatus(ctx context.Context, slug, authorAccountID string, expectedRevision int64, submit bool) (*Proposal, error) {
	slug = strings.TrimSpace(slug)
	authorAccountID, err := canonicalAccountID(authorAccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: proposal author account ID is invalid", ErrInvalidProposal)
	}
	authorUUID, err := accountUUID(authorAccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: proposal author account ID is invalid", ErrInvalidProposal)
	}
	var changed profitsharingsqlc.ProfitSharingProposal
	var persistedItems []profitsharingsqlc.ProfitSharingProposalItem
	err = s.withTx(ctx, func(queries *profitsharingsqlc.Queries) error {
		round, err := lockRoundBySlug(ctx, queries, slug)
		if err != nil {
			return err
		}
		if round.Phase != RoundPhaseCollecting {
			return ErrInvalidPhase
		}
		proposal, err := lockProposalByAuthor(ctx, queries, round.ID, authorAccountID)
		if err != nil {
			return err
		}
		if proposal.Revision != expectedRevision {
			return ErrRevisionConflict
		}
		if submit {
			if proposal.Status != ProposalStatusDraft {
				return ErrInvalidPhase
			}
			participants, listErr := queries.ListProfitSharingParticipants(ctx, round.ID)
			if listErr != nil {
				return fmt.Errorf("list participants for profit sharing round %q: %w", slug, listErr)
			}
			persistedItems, err = queries.ListProfitSharingProposalItems(ctx, profitsharingsqlc.ListProfitSharingProposalItemsParams{
				RoundID:    round.ID,
				ProposalID: proposal.ID,
			})
			if err != nil {
				return fmt.Errorf("list items for profit sharing proposal %q: %w", proposal.ID, err)
			}
			if err := validateCompleteProposal(persistedItems, participants); err != nil {
				return err
			}
			changed, err = queries.SubmitProfitSharingProposal(ctx, profitsharingsqlc.SubmitProfitSharingProposalParams{
				ProposalID:       proposal.ID,
				RoundID:          round.ID,
				AuthorAccountID:  authorUUID,
				ExpectedRevision: expectedRevision,
			})
		} else {
			if proposal.Status != ProposalStatusSubmitted {
				return ErrInvalidPhase
			}
			changed, err = queries.ReopenProfitSharingProposal(ctx, profitsharingsqlc.ReopenProfitSharingProposalParams{
				ProposalID:       proposal.ID,
				RoundID:          round.ID,
				AuthorAccountID:  authorUUID,
				ExpectedRevision: expectedRevision,
			})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRevisionConflict
		}
		if err != nil {
			return fmt.Errorf("change status for profit sharing proposal %q: %w", proposal.ID, err)
		}
		if persistedItems == nil {
			persistedItems, err = queries.ListProfitSharingProposalItems(ctx, profitsharingsqlc.ListProfitSharingProposalItemsParams{
				RoundID:    round.ID,
				ProposalID: proposal.ID,
			})
			if err != nil {
				return fmt.Errorf("list items for profit sharing proposal %q: %w", proposal.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	converted := proposalFromSQLC(changed)
	for _, item := range persistedItems {
		converted.Items = append(converted.Items, proposalItemFromSQLC(item))
	}
	return &converted, nil
}

func (s *SQLStore) PublishRound(ctx context.Context, slug string, expectedRevision int64) (*Round, error) {
	slug = strings.TrimSpace(slug)
	var published profitsharingsqlc.ProfitSharingRound
	err := s.withTx(ctx, func(queries *profitsharingsqlc.Queries) error {
		round, err := lockRoundBySlug(ctx, queries, slug)
		if err != nil {
			return err
		}
		if round.Phase != RoundPhaseCollecting {
			return ErrInvalidPhase
		}
		if round.Revision != expectedRevision {
			return ErrRevisionConflict
		}
		participants, err := queries.ListProfitSharingParticipants(ctx, round.ID)
		if err != nil {
			return fmt.Errorf("list participants for profit sharing round %q: %w", slug, err)
		}
		proposals, err := queries.ListProfitSharingProposals(ctx, round.ID)
		if err != nil {
			return fmt.Errorf("list proposals for profit sharing round %q: %w", slug, err)
		}
		if len(participants) != requiredParticipantCount || len(proposals) != len(participants) {
			return ErrNotAllProposalsSubmitted
		}
		for _, proposal := range proposals {
			if proposal.Status != ProposalStatusSubmitted {
				return ErrNotAllProposalsSubmitted
			}
		}
		// Proposal IDs are UUIDv4 values created at OpenRound. Sorting those random
		// values produces a random assignment while the persisted label remains stable.
		sort.Slice(proposals, func(i, j int) bool { return proposals[i].ID < proposals[j].ID })
		proposalIDs := make([]string, 0, len(proposals))
		for i, proposal := range proposals {
			if !proposal.AnonymousLabel.Valid {
				if _, err := queries.AssignProfitSharingProposalLabel(ctx, profitsharingsqlc.AssignProfitSharingProposalLabelParams{
					AnonymousLabel: proposalLabel(i),
					ProposalID:     proposal.ID,
					RoundID:        round.ID,
				}); err != nil {
					return fmt.Errorf("assign anonymous label to profit sharing proposal %q: %w", proposal.ID, err)
				}
			}
			proposalIDs = append(proposalIDs, proposal.ID)
		}
		if _, err := queries.CreateProfitSharingBallot(ctx, profitsharingsqlc.CreateProfitSharingBallotParams{
			RoundID:      round.ID,
			BallotNumber: 1,
		}); err != nil {
			return fmt.Errorf("create initial ballot for profit sharing round %q: %w", slug, err)
		}
		if err := queries.BatchCreateProfitSharingBallotCandidates(ctx, profitsharingsqlc.BatchCreateProfitSharingBallotCandidatesParams{
			RoundID:      round.ID,
			BallotNumber: 1,
			ProposalIds:  proposalIDs,
		}); err != nil {
			return fmt.Errorf("create initial ballot candidates for profit sharing round %q: %w", slug, err)
		}
		published, err = queries.PublishProfitSharingRound(ctx, profitsharingsqlc.PublishProfitSharingRoundParams{
			RoundID:          round.ID,
			ExpectedRevision: expectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRevisionConflict
		}
		if err != nil {
			return fmt.Errorf("publish profit sharing round %q: %w", slug, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	converted := roundFromSQLC(published)
	return &converted, nil
}

func (s *SQLStore) SubmitVote(ctx context.Context, slug string, ballotNumber int32, voterAccountID, proposalID string) (*Vote, error) {
	slug = strings.TrimSpace(slug)
	voterAccountID, err := canonicalAccountID(voterAccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: voter account ID is invalid", ErrInvalidProposal)
	}
	voterUUID, err := accountUUID(voterAccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: voter account ID is invalid", ErrInvalidProposal)
	}
	proposalID = strings.TrimSpace(proposalID)
	var persisted profitsharingsqlc.ProfitSharingVote
	err = s.withTx(ctx, func(queries *profitsharingsqlc.Queries) error {
		round, err := lockRoundBySlug(ctx, queries, slug)
		if err != nil {
			return err
		}
		if round.Phase != RoundPhaseVoting || !round.ActiveBallotNumber.Valid || round.ActiveBallotNumber.Int32 != ballotNumber {
			return ErrInvalidPhase
		}
		if _, err := queries.GetProfitSharingParticipant(ctx, profitsharingsqlc.GetProfitSharingParticipantParams{
			RoundID:   round.ID,
			AccountID: voterUUID,
		}); errors.Is(err, pgx.ErrNoRows) {
			return ErrNotParticipant
		} else if err != nil {
			return fmt.Errorf("get voter in profit sharing round %q: %w", slug, err)
		}
		ballot, err := queries.GetActiveProfitSharingBallotForUpdate(ctx, round.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBallotNotFound
		}
		if err != nil {
			return fmt.Errorf("lock active ballot for profit sharing round %q: %w", slug, err)
		}
		if ballot.Status != "open" || ballot.BallotNumber != ballotNumber {
			return ErrInvalidPhase
		}
		candidates, err := queries.ListProfitSharingBallotCandidates(ctx, profitsharingsqlc.ListProfitSharingBallotCandidatesParams{
			RoundID:      round.ID,
			BallotNumber: ballotNumber,
		})
		if err != nil {
			return fmt.Errorf("list ballot candidates for profit sharing round %q: %w", slug, err)
		}
		var selected *profitsharingsqlc.ProfitSharingProposal
		for i := range candidates {
			if candidates[i].ID == proposalID {
				selected = &candidates[i]
				break
			}
		}
		if selected == nil {
			return ErrCandidateNotFound
		}
		if selected.AuthorAccountID == voterUUID {
			return ErrSelfVote
		}
		persisted, err = queries.UpsertProfitSharingVote(ctx, profitsharingsqlc.UpsertProfitSharingVoteParams{
			RoundID:                 round.ID,
			BallotNumber:            ballotNumber,
			VoterAccountID:          voterUUID,
			ProposalID:              selected.ID,
			ProposalAuthorAccountID: selected.AuthorAccountID,
		})
		if err != nil {
			if isConstraint(err, "profit_sharing_vote_no_self_vote_check") {
				return ErrSelfVote
			}
			return fmt.Errorf("submit vote in profit sharing round %q: %w", slug, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	converted := voteFromSQLC(persisted)
	return &converted, nil
}

func (s *SQLStore) CloseBallot(ctx context.Context, slug string, expectedRevision int64) (*CloseBallotResult, error) {
	slug = strings.TrimSpace(slug)
	var result CloseBallotResult
	err := s.withTx(ctx, func(queries *profitsharingsqlc.Queries) error {
		round, err := lockRoundBySlug(ctx, queries, slug)
		if err != nil {
			return err
		}
		if round.Phase != RoundPhaseVoting || !round.ActiveBallotNumber.Valid {
			return ErrInvalidPhase
		}
		if round.Revision != expectedRevision {
			return ErrRevisionConflict
		}
		ballot, err := queries.GetActiveProfitSharingBallotForUpdate(ctx, round.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBallotNotFound
		}
		if err != nil {
			return fmt.Errorf("lock active ballot for profit sharing round %q: %w", slug, err)
		}
		if ballot.Status != "open" {
			return ErrInvalidPhase
		}
		participants, err := queries.ListProfitSharingParticipants(ctx, round.ID)
		if err != nil {
			return fmt.Errorf("list participants for profit sharing round %q: %w", slug, err)
		}
		voteCount, err := queries.CountProfitSharingBallotVotes(ctx, profitsharingsqlc.CountProfitSharingBallotVotesParams{
			RoundID:      round.ID,
			BallotNumber: ballot.BallotNumber,
		})
		if err != nil {
			return fmt.Errorf("count votes for profit sharing round %q: %w", slug, err)
		}
		if voteCount != int64(len(participants)) {
			return ErrNotAllVotesSubmitted
		}
		tallies, err := queries.ListProfitSharingBallotTallies(ctx, profitsharingsqlc.ListProfitSharingBallotTalliesParams{
			RoundID:      round.ID,
			BallotNumber: ballot.BallotNumber,
		})
		if err != nil {
			return fmt.Errorf("list tallies for profit sharing round %q: %w", slug, err)
		}
		if len(tallies) == 0 {
			return ErrCandidateNotFound
		}
		maxVotes := tallies[0].VoteCount
		tiedProposalIDs := make([]string, 0, len(tallies))
		for _, tally := range tallies {
			if tally.VoteCount != maxVotes {
				break
			}
			tiedProposalIDs = append(tiedProposalIDs, tally.ProposalID)
		}
		if _, err := queries.CloseProfitSharingBallot(ctx, profitsharingsqlc.CloseProfitSharingBallotParams{
			RoundID:      round.ID,
			BallotNumber: ballot.BallotNumber,
		}); err != nil {
			return fmt.Errorf("close ballot %d for profit sharing round %q: %w", ballot.BallotNumber, slug, err)
		}
		if len(tiedProposalIDs) == 1 {
			closed, err := queries.CloseProfitSharingRound(ctx, profitsharingsqlc.CloseProfitSharingRoundParams{
				FinalProposalID:  tiedProposalIDs[0],
				RoundID:          round.ID,
				ExpectedRevision: expectedRevision,
			})
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrRevisionConflict
			}
			if err != nil {
				return fmt.Errorf("close profit sharing round %q: %w", slug, err)
			}
			result.Round = roundFromSQLC(closed)
			return nil
		}
		nextBallotNumber := ballot.BallotNumber + 1
		if _, err := queries.CreateProfitSharingBallot(ctx, profitsharingsqlc.CreateProfitSharingBallotParams{
			RoundID:      round.ID,
			BallotNumber: nextBallotNumber,
		}); err != nil {
			return fmt.Errorf("create runoff ballot for profit sharing round %q: %w", slug, err)
		}
		if err := queries.BatchCreateProfitSharingBallotCandidates(ctx, profitsharingsqlc.BatchCreateProfitSharingBallotCandidatesParams{
			RoundID:      round.ID,
			BallotNumber: nextBallotNumber,
			ProposalIds:  tiedProposalIDs,
		}); err != nil {
			return fmt.Errorf("create runoff candidates for profit sharing round %q: %w", slug, err)
		}
		advanced, err := queries.AdvanceProfitSharingRunoffBallot(ctx, profitsharingsqlc.AdvanceProfitSharingRunoffBallotParams{
			NextBallotNumber: nextBallotNumber,
			RoundID:          round.ID,
			ExpectedRevision: expectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRevisionConflict
		}
		if err != nil {
			return fmt.Errorf("advance profit sharing round %q to runoff: %w", slug, err)
		}
		result.Round = roundFromSQLC(advanced)
		result.RunoffCreated = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SQLStore) withTx(ctx context.Context, fn func(*profitsharingsqlc.Queries) error) error {
	pool, err := s.requirePool()
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin profit sharing transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	if err := fn(profitsharingsqlc.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit profit sharing transaction: %w", err)
	}
	committed = true
	return nil
}

func lockRoundBySlug(ctx context.Context, queries *profitsharingsqlc.Queries, slug string) (profitsharingsqlc.ProfitSharingRound, error) {
	row, err := queries.GetProfitSharingRoundBySlugForUpdate(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return profitsharingsqlc.ProfitSharingRound{}, ErrRoundNotFound
	}
	if err != nil {
		return profitsharingsqlc.ProfitSharingRound{}, fmt.Errorf("lock profit sharing round %q: %w", slug, err)
	}
	return row, nil
}

func lockProposalByAuthor(ctx context.Context, queries *profitsharingsqlc.Queries, roundID int64, authorAccountID string) (profitsharingsqlc.ProfitSharingProposal, error) {
	authorUUID, err := accountUUID(authorAccountID)
	if err != nil {
		return profitsharingsqlc.ProfitSharingProposal{}, ErrProposalNotFound
	}
	row, err := queries.GetProfitSharingProposalByAuthorForUpdate(ctx, profitsharingsqlc.GetProfitSharingProposalByAuthorForUpdateParams{
		RoundID:         roundID,
		AuthorAccountID: authorUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return profitsharingsqlc.ProfitSharingProposal{}, ErrProposalNotFound
	}
	if err != nil {
		return profitsharingsqlc.ProfitSharingProposal{}, fmt.Errorf("lock profit sharing proposal for account %q: %w", authorAccountID, err)
	}
	return row, nil
}

func validateParticipantInputs(inputs []ParticipantInput) ([]ParticipantInput, error) {
	normalized := make([]ParticipantInput, len(inputs))
	accountIDs := make(map[string]struct{}, len(inputs))
	orders := make(map[int32]struct{}, len(inputs))
	for i, input := range inputs {
		accountID, err := canonicalAccountID(input.AccountID)
		if err != nil {
			return nil, fmt.Errorf("%w: participant account ID is invalid", ErrInvalidParticipants)
		}
		input.AccountID = accountID
		input.Username = strings.TrimSpace(input.Username)
		input.DisplayName = strings.TrimSpace(input.DisplayName)
		input.BaselineResponsibility = strings.TrimSpace(input.BaselineResponsibility)
		if input.Username == "" {
			return nil, fmt.Errorf("%w: participant %q username is required", ErrInvalidParticipants, input.AccountID)
		}
		if input.DisplayName == "" {
			return nil, fmt.Errorf("%w: participant %q display name is required", ErrInvalidParticipants, input.AccountID)
		}
		if input.DisplayOrder <= 0 {
			return nil, fmt.Errorf("%w: participant %q display order must be positive", ErrInvalidParticipants, input.AccountID)
		}
		if _, exists := accountIDs[input.AccountID]; exists {
			return nil, fmt.Errorf("%w: duplicate participant account ID %q", ErrInvalidParticipants, input.AccountID)
		}
		if _, exists := orders[input.DisplayOrder]; exists {
			return nil, fmt.Errorf("%w: duplicate participant display order %d", ErrInvalidParticipants, input.DisplayOrder)
		}
		accountIDs[input.AccountID] = struct{}{}
		orders[input.DisplayOrder] = struct{}{}
		normalized[i] = input
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].DisplayOrder < normalized[j].DisplayOrder })
	return normalized, nil
}

func normalizeValidatedAccountIDs(accountIDs []string) ([]string, error) {
	if len(accountIDs) != requiredParticipantCount {
		return nil, fmt.Errorf("%w: exactly %d validated accounts are required", ErrInvalidParticipants, requiredParticipantCount)
	}
	normalized := make([]string, len(accountIDs))
	seen := make(map[string]struct{}, len(accountIDs))
	for i, accountID := range accountIDs {
		canonicalID, err := canonicalAccountID(accountID)
		if err != nil {
			return nil, fmt.Errorf("%w: validated account ID is invalid", ErrInvalidParticipants)
		}
		if _, exists := seen[canonicalID]; exists {
			return nil, fmt.Errorf("%w: duplicate validated account ID %q", ErrInvalidParticipants, canonicalID)
		}
		seen[canonicalID] = struct{}{}
		normalized[i] = canonicalID
	}
	return normalized, nil
}

func validateOpenParticipants(participants []profitsharingsqlc.ProfitSharingParticipant, validatedAccountIDs []string) error {
	if len(participants) != requiredParticipantCount || len(validatedAccountIDs) != requiredParticipantCount {
		return fmt.Errorf("%w: exactly %d participants are required", ErrInvalidParticipants, requiredParticipantCount)
	}
	validated := make(map[string]struct{}, len(validatedAccountIDs))
	for _, accountID := range validatedAccountIDs {
		if _, duplicate := validated[accountID]; duplicate {
			return fmt.Errorf("%w: duplicate validated account ID %q", ErrInvalidParticipants, accountID)
		}
		validated[accountID] = struct{}{}
	}
	configured := make(map[string]struct{}, len(participants))
	for _, participant := range participants {
		accountID := accountIDFromUUID(participant.AccountID)
		if accountID == "" {
			return fmt.Errorf("%w: participant account ID is invalid", ErrInvalidParticipants)
		}
		if strings.TrimSpace(participant.Username) == "" {
			return fmt.Errorf("%w: participant %q username is required before opening", ErrInvalidParticipants, accountID)
		}
		if strings.TrimSpace(participant.DisplayName) == "" {
			return fmt.Errorf("%w: participant %q display name is required before opening", ErrInvalidParticipants, accountID)
		}
		if strings.TrimSpace(participant.BaselineResponsibility) == "" {
			return fmt.Errorf("%w: participant %q baseline responsibility is required before opening", ErrInvalidParticipants, accountID)
		}
		if _, duplicate := configured[accountID]; duplicate {
			return fmt.Errorf("%w: duplicate configured account ID %q", ErrInvalidParticipants, accountID)
		}
		configured[accountID] = struct{}{}
		if _, ok := validated[accountID]; !ok {
			return fmt.Errorf("%w: configured account ID %q was not validated", ErrInvalidParticipants, accountID)
		}
	}
	if len(configured) != len(validated) {
		return fmt.Errorf("%w: validated accounts do not match configured participants", ErrInvalidParticipants)
	}
	return nil
}

func validateProposalItems(items []ProposalItemInput, participants []profitsharingsqlc.ProfitSharingParticipant) ([]ProposalItemInput, error) {
	if len(items) != len(participants) {
		return nil, fmt.Errorf("%w: proposal must contain one item for every participant", ErrInvalidProposal)
	}
	byAccountID := make(map[string]ProposalItemInput, len(items))
	for _, item := range items {
		accountID, err := canonicalAccountID(item.ParticipantAccountID)
		if err != nil {
			return nil, fmt.Errorf("%w: proposal participant account ID is invalid", ErrInvalidProposal)
		}
		item.ParticipantAccountID = accountID
		item.Responsibility = strings.TrimSpace(item.Responsibility)
		if _, duplicate := byAccountID[item.ParticipantAccountID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate proposal participant %q", ErrInvalidProposal, item.ParticipantAccountID)
		}
		if item.BasisPoints != nil && (*item.BasisPoints < 0 || *item.BasisPoints > 10000) {
			return nil, fmt.Errorf("%w: basis points for %q must be between 0 and 10000", ErrInvalidProposal, item.ParticipantAccountID)
		}
		byAccountID[item.ParticipantAccountID] = item
	}
	normalized := make([]ProposalItemInput, 0, len(participants))
	for _, participant := range participants {
		accountID := accountIDFromUUID(participant.AccountID)
		item, ok := byAccountID[accountID]
		if !ok {
			return nil, fmt.Errorf("%w: missing proposal participant %q", ErrInvalidProposal, accountID)
		}
		normalized = append(normalized, item)
	}
	return normalized, nil
}

func validateCompleteProposal(items []profitsharingsqlc.ProfitSharingProposalItem, participants []profitsharingsqlc.ProfitSharingParticipant) error {
	if len(items) != len(participants) {
		return fmt.Errorf("%w: proposal must contain one item for every participant", ErrIncompleteProposal)
	}
	expected := make(map[string]struct{}, len(participants))
	for _, participant := range participants {
		expected[accountIDFromUUID(participant.AccountID)] = struct{}{}
	}
	var total int64
	for _, item := range items {
		accountID := accountIDFromUUID(item.ParticipantAccountID)
		if _, ok := expected[accountID]; !ok {
			return fmt.Errorf("%w: unexpected participant %q", ErrIncompleteProposal, accountID)
		}
		delete(expected, accountID)
		if strings.TrimSpace(item.Responsibility) == "" {
			return fmt.Errorf("%w: responsibility for %q is required", ErrIncompleteProposal, accountID)
		}
		if !item.BasisPoints.Valid {
			return fmt.Errorf("%w: basis points for %q are required", ErrIncompleteProposal, accountID)
		}
		total += int64(item.BasisPoints.Int32)
	}
	if len(expected) != 0 {
		return fmt.Errorf("%w: proposal is missing participants", ErrIncompleteProposal)
	}
	if total != 10000 {
		return fmt.Errorf("%w: basis points must total 10000", ErrIncompleteProposal)
	}
	return nil
}

func createParticipants(ctx context.Context, queries *profitsharingsqlc.Queries, roundID int64, participants []ParticipantInput) error {
	if len(participants) == 0 {
		return nil
	}
	params := profitsharingsqlc.BatchCreateProfitSharingParticipantsParams{
		RoundID:                  roundID,
		AccountIds:               make([]pgtype.UUID, 0, len(participants)),
		Usernames:                make([]string, 0, len(participants)),
		DisplayNames:             make([]string, 0, len(participants)),
		DisplayOrders:            make([]int32, 0, len(participants)),
		BaselineResponsibilities: make([]string, 0, len(participants)),
	}
	for _, participant := range participants {
		accountID, err := accountUUID(participant.AccountID)
		if err != nil {
			return fmt.Errorf("create profit sharing participant: %w", err)
		}
		params.AccountIds = append(params.AccountIds, accountID)
		params.Usernames = append(params.Usernames, participant.Username)
		params.DisplayNames = append(params.DisplayNames, participant.DisplayName)
		params.DisplayOrders = append(params.DisplayOrders, participant.DisplayOrder)
		params.BaselineResponsibilities = append(params.BaselineResponsibilities, participant.BaselineResponsibility)
	}
	if err := queries.BatchCreateProfitSharingParticipants(ctx, params); err != nil {
		return fmt.Errorf("create profit sharing participants: %w", err)
	}
	return nil
}

func createProposalItems(ctx context.Context, queries *profitsharingsqlc.Queries, roundID int64, proposalID string, items []ProposalItemInput) error {
	params := profitsharingsqlc.BatchCreateProfitSharingProposalItemsParams{
		RoundID:               roundID,
		ProposalID:            proposalID,
		ParticipantAccountIds: make([]pgtype.UUID, 0, len(items)),
		Responsibilities:      make([]string, 0, len(items)),
		BasisPointsValues:     make([]int32, 0, len(items)),
	}
	for _, item := range items {
		accountID, err := accountUUID(item.ParticipantAccountID)
		if err != nil {
			return fmt.Errorf("create profit sharing proposal item: %w", err)
		}
		params.ParticipantAccountIds = append(params.ParticipantAccountIds, accountID)
		params.Responsibilities = append(params.Responsibilities, item.Responsibility)
		basisPoints := int32(-1)
		if item.BasisPoints != nil {
			basisPoints = *item.BasisPoints
		}
		params.BasisPointsValues = append(params.BasisPointsValues, basisPoints)
	}
	if err := queries.BatchCreateProfitSharingProposalItems(ctx, params); err != nil {
		return fmt.Errorf("create items for profit sharing proposal %q: %w", proposalID, err)
	}
	return nil
}

func proposalLabel(index int) string {
	label := ""
	for index >= 0 {
		label = string(rune('A'+index%26)) + label
		index = index/26 - 1
	}
	return "Proposal " + label
}

func isConstraint(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == constraint
}
