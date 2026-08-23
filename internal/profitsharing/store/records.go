package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	profitsharingsqlc "github.com/useryege/athena/internal/profitsharing/store/sqlc"
)

func (s *SQLStore) ListRounds(ctx context.Context, requesterAccount string, requesterIsAdmin bool) ([]RoundSnapshot, error) {
	queries, err := s.requireQueries()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProfitSharingRounds(ctx, profitsharingsqlc.ListProfitSharingRoundsParams{
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, fmt.Errorf("list profit sharing rounds: %w", err)
	}
	snapshots := make([]RoundSnapshot, 0, len(rows))
	for _, row := range rows {
		snapshot, err := loadRoundSnapshot(ctx, queries, row, requesterAccount)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func (s *SQLStore) GetRound(ctx context.Context, slug, requesterAccount string) (*RoundSnapshot, error) {
	queries, err := s.requireQueries()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProfitSharingRoundBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoundNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get profit sharing round %q: %w", slug, err)
	}
	snapshot, err := loadRoundSnapshot(ctx, queries, row, requesterAccount)
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func loadRoundSnapshot(ctx context.Context, queries profitsharingsqlc.Querier, row profitsharingsqlc.ProfitSharingRound, requesterAccount string) (RoundSnapshot, error) {
	participants, err := queries.ListProfitSharingParticipants(ctx, row.ID)
	if err != nil {
		return RoundSnapshot{}, fmt.Errorf("list participants for profit sharing round %q: %w", row.Slug, err)
	}
	proposals, err := queries.ListProfitSharingProposals(ctx, row.ID)
	if err != nil {
		return RoundSnapshot{}, fmt.Errorf("list proposals for profit sharing round %q: %w", row.Slug, err)
	}

	snapshot := RoundSnapshot{
		Round:        roundFromSQLC(row),
		Participants: make([]Participant, 0, len(participants)),
		Proposals:    make([]Proposal, 0, len(proposals)),
	}
	for _, participant := range participants {
		snapshot.Participants = append(snapshot.Participants, participantFromSQLC(participant))
	}
	proposalIndex := make(map[string]int, len(proposals))
	for _, proposalRow := range proposals {
		proposal := proposalFromSQLC(proposalRow)
		items, err := queries.ListProfitSharingProposalItems(ctx, profitsharingsqlc.ListProfitSharingProposalItemsParams{
			RoundID:    row.ID,
			ProposalID: proposal.ID,
		})
		if err != nil {
			return RoundSnapshot{}, fmt.Errorf("list items for profit sharing proposal %q: %w", proposal.ID, err)
		}
		proposal.Items = make([]ProposalItem, 0, len(items))
		for _, item := range items {
			proposal.Items = append(proposal.Items, proposalItemFromSQLC(item))
		}
		proposalIndex[proposal.ID] = len(snapshot.Proposals)
		snapshot.Proposals = append(snapshot.Proposals, proposal)
	}

	if !row.ActiveBallotNumber.Valid {
		return snapshot, nil
	}
	ballots, err := queries.ListProfitSharingBallots(ctx, row.ID)
	if err != nil {
		return RoundSnapshot{}, fmt.Errorf("list ballots for profit sharing round %q: %w", row.Slug, err)
	}
	var ballotRow *profitsharingsqlc.ProfitSharingBallot
	for i := range ballots {
		if ballots[i].BallotNumber == row.ActiveBallotNumber.Int32 {
			ballotRow = &ballots[i]
			break
		}
	}
	if ballotRow == nil {
		return RoundSnapshot{}, fmt.Errorf("active ballot %d for profit sharing round %q is missing", row.ActiveBallotNumber.Int32, row.Slug)
	}
	candidates, err := queries.ListProfitSharingBallotCandidates(ctx, profitsharingsqlc.ListProfitSharingBallotCandidatesParams{
		RoundID:      row.ID,
		BallotNumber: ballotRow.BallotNumber,
	})
	if err != nil {
		return RoundSnapshot{}, fmt.Errorf("list active ballot candidates for profit sharing round %q: %w", row.Slug, err)
	}
	ballot := Ballot{
		RoundID:    row.ID,
		Number:     ballotRow.BallotNumber,
		Status:     ballotRow.Status,
		CreatedAt:  ballotRow.CreatedAt.Time,
		ClosedAt:   timestamptzPtr(ballotRow.ClosedAt),
		Candidates: make([]Proposal, 0, len(candidates)),
		VoteCounts: make(map[string]int64),
	}
	for _, candidate := range candidates {
		if index, ok := proposalIndex[candidate.ID]; ok {
			ballot.Candidates = append(ballot.Candidates, snapshot.Proposals[index])
			continue
		}
		ballot.Candidates = append(ballot.Candidates, proposalFromSQLC(candidate))
	}
	ballot.VotedCount, err = queries.CountProfitSharingBallotVotes(ctx, profitsharingsqlc.CountProfitSharingBallotVotesParams{
		RoundID:      row.ID,
		BallotNumber: ballot.Number,
	})
	if err != nil {
		return RoundSnapshot{}, fmt.Errorf("count votes for profit sharing round %q ballot %d: %w", row.Slug, ballot.Number, err)
	}
	if row.Phase == RoundPhaseClosed {
		tallies, err := queries.ListProfitSharingBallotTallies(ctx, profitsharingsqlc.ListProfitSharingBallotTalliesParams{
			RoundID:      row.ID,
			BallotNumber: ballot.Number,
		})
		if err != nil {
			return RoundSnapshot{}, fmt.Errorf("list final tallies for profit sharing round %q: %w", row.Slug, err)
		}
		for _, tally := range tallies {
			ballot.VoteCounts[tally.ProposalID] = tally.VoteCount
		}
	}
	if requesterAccount != "" {
		vote, err := queries.GetProfitSharingVote(ctx, profitsharingsqlc.GetProfitSharingVoteParams{
			RoundID:      row.ID,
			BallotNumber: ballot.Number,
			VoterAccount: requesterAccount,
		})
		if err == nil {
			converted := voteFromSQLC(vote)
			ballot.CurrentVote = &converted
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return RoundSnapshot{}, fmt.Errorf("get current vote for profit sharing round %q: %w", row.Slug, err)
		}
	}
	snapshot.Ballot = &ballot
	return snapshot, nil
}

func roundFromSQLC(row profitsharingsqlc.ProfitSharingRound) Round {
	return Round{
		ID:                 row.ID,
		Slug:               row.Slug,
		Title:              row.Title,
		Phase:              row.Phase,
		Revision:           row.Revision,
		ActiveBallotNumber: int4Ptr(row.ActiveBallotNumber),
		FinalProposalID:    textPtr(row.FinalProposalID),
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
		OpenedAt:           timestamptzPtr(row.OpenedAt),
		PublishedAt:        timestamptzPtr(row.PublishedAt),
		ClosedAt:           timestamptzPtr(row.ClosedAt),
	}
}

func participantFromSQLC(row profitsharingsqlc.ProfitSharingParticipant) Participant {
	return Participant{
		RoundID:                row.RoundID,
		Account:                row.Account,
		DisplayName:            row.DisplayName,
		DisplayOrder:           row.DisplayOrder,
		BaselineResponsibility: row.BaselineResponsibility,
	}
}

func proposalFromSQLC(row profitsharingsqlc.ProfitSharingProposal) Proposal {
	return Proposal{
		ID:             row.ID,
		RoundID:        row.RoundID,
		AuthorAccount:  row.AuthorAccount,
		Status:         row.Status,
		AnonymousLabel: textPtr(row.AnonymousLabel),
		Revision:       row.Revision,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
		SubmittedAt:    timestamptzPtr(row.SubmittedAt),
	}
}

func proposalItemFromSQLC(row profitsharingsqlc.ProfitSharingProposalItem) ProposalItem {
	return ProposalItem{
		RoundID:            row.RoundID,
		ProposalID:         row.ProposalID,
		ParticipantAccount: row.ParticipantAccount,
		Responsibility:     row.Responsibility,
		BasisPoints:        int4Ptr(row.BasisPoints),
	}
}

func voteFromSQLC(row profitsharingsqlc.ProfitSharingVote) Vote {
	return Vote{
		RoundID:               row.RoundID,
		BallotNumber:          row.BallotNumber,
		VoterAccount:          row.VoterAccount,
		ProposalID:            row.ProposalID,
		ProposalAuthorAccount: row.ProposalAuthorAccount,
		CreatedAt:             row.CreatedAt.Time,
		UpdatedAt:             row.UpdatedAt.Time,
	}
}

func int4Ptr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	converted := value.Int32
	return &converted
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	converted := value.String
	return &converted
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	converted := value.Time
	return &converted
}
