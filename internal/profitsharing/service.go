package profitsharing

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/profitsharing/apiclient"
	profitsharingstore "github.com/useryege/athena/internal/profitsharing/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	apiclient.UnimplementedProfitSharingServiceServer
	store       *profitsharingstore.SQLStore
	startStopMu sync.Mutex
	started     bool
}

func NewService(store *profitsharingstore.SQLStore) *Service {
	return &Service{store: store}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "profit sharing store is required")
	}
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	s.started = false
	s.startStopMu.Unlock()
	return nil
}

func (s *Service) ListRounds(ctx context.Context, req *apiclient.ListRoundsRequest) (*apiclient.ListRoundsResponse, error) {
	requester, requesterIsAdmin, err := requestIdentity(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	snapshots, err := s.store.ListRounds(ctx, requester, requesterIsAdmin)
	if err != nil {
		return nil, serviceError(err)
	}
	response := &apiclient.ListRoundsResponse{Rounds: make([]*apiclient.RoundSummary, 0, len(snapshots))}
	for i := range snapshots {
		response.Rounds = append(response.Rounds, roundSummaryToProto(&snapshots[i]))
	}
	return response, nil
}

func (s *Service) GetRound(ctx context.Context, req *apiclient.GetRoundRequest) (*apiclient.GetRoundResponse, error) {
	requester, requesterIsAdmin, err := requestIdentity(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	snapshot, err := s.authorizedRound(ctx, req.GetSlug(), requester, requesterIsAdmin)
	if err != nil {
		return nil, err
	}
	return &apiclient.GetRoundResponse{Round: roundToProto(snapshot, requester, requesterIsAdmin)}, nil
}

func (s *Service) CreateRound(ctx context.Context, req *apiclient.CreateRoundRequest) (*apiclient.CreateRoundResponse, error) {
	requester, err := requireAdmin(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	participants, err := participantInputsFromProto(req.GetParticipants())
	if err != nil {
		return nil, err
	}
	created, err := s.store.CreateRound(ctx, req.GetSlug(), req.GetTitle(), participants)
	if err != nil {
		return nil, serviceError(err)
	}
	snapshot, err := s.store.GetRound(ctx, created.Slug, requester)
	if err != nil {
		return nil, serviceError(err)
	}
	return &apiclient.CreateRoundResponse{Round: roundToProto(snapshot, requester, true)}, nil
}

func (s *Service) UpdateRound(ctx context.Context, req *apiclient.UpdateRoundRequest) (*apiclient.UpdateRoundResponse, error) {
	requester, err := requireAdmin(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	participants, err := participantInputsFromProto(req.GetParticipants())
	if err != nil {
		return nil, err
	}
	updated, err := s.store.UpdateRound(ctx, req.GetCurrentSlug(), req.GetSlug(), req.GetTitle(), participants, req.GetExpectedRevision())
	if err != nil {
		return nil, serviceError(err)
	}
	snapshot, err := s.store.GetRound(ctx, updated.Slug, requester)
	if err != nil {
		return nil, serviceError(err)
	}
	return &apiclient.UpdateRoundResponse{Round: roundToProto(snapshot, requester, true)}, nil
}

func (s *Service) OpenRound(ctx context.Context, req *apiclient.OpenRoundRequest) (*apiclient.OpenRoundResponse, error) {
	requester, err := requireAdmin(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	opened, err := s.store.OpenRound(ctx, req.GetSlug(), req.GetExpectedRevision(), req.GetValidatedParticipantAccountIds())
	if err != nil {
		return nil, serviceError(err)
	}
	snapshot, err := s.store.GetRound(ctx, opened.Slug, requester)
	if err != nil {
		return nil, serviceError(err)
	}
	return &apiclient.OpenRoundResponse{Round: roundToProto(snapshot, requester, true)}, nil
}

func (s *Service) PublishRound(ctx context.Context, req *apiclient.PublishRoundRequest) (*apiclient.PublishRoundResponse, error) {
	requester, err := requireAdmin(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	published, err := s.store.PublishRound(ctx, req.GetSlug(), req.GetExpectedRevision())
	if err != nil {
		return nil, serviceError(err)
	}
	snapshot, err := s.store.GetRound(ctx, published.Slug, requester)
	if err != nil {
		return nil, serviceError(err)
	}
	return &apiclient.PublishRoundResponse{Round: roundToProto(snapshot, requester, true)}, nil
}

func (s *Service) CloseBallot(ctx context.Context, req *apiclient.CloseBallotRequest) (*apiclient.CloseBallotResponse, error) {
	requester, err := requireAdmin(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	result, err := s.store.CloseBallot(ctx, req.GetSlug(), req.GetExpectedRevision())
	if err != nil {
		return nil, serviceError(err)
	}
	snapshot, err := s.store.GetRound(ctx, result.Round.Slug, requester)
	if err != nil {
		return nil, serviceError(err)
	}
	return &apiclient.CloseBallotResponse{
		Round:         roundToProto(snapshot, requester, true),
		RunoffCreated: result.RunoffCreated,
	}, nil
}

func (s *Service) UpdateProposal(ctx context.Context, req *apiclient.UpdateProposalRequest) (*apiclient.UpdateProposalResponse, error) {
	requester, err := requireParticipantActor(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	if err := s.authorizeParticipant(ctx, req.GetSlug(), requester); err != nil {
		return nil, err
	}
	items, err := proposalItemInputsFromProto(req.GetItems())
	if err != nil {
		return nil, err
	}
	proposal, err := s.store.UpdateProposal(ctx, req.GetSlug(), requester, req.GetExpectedRevision(), items)
	if err != nil {
		return nil, serviceError(err)
	}
	snapshot, err := s.store.GetRound(ctx, req.GetSlug(), requester)
	if err != nil {
		return nil, serviceError(err)
	}
	return &apiclient.UpdateProposalResponse{Proposal: proposalToProto(proposal, snapshot.Participants, requester, true, true, false, 0, false)}, nil
}

func (s *Service) SubmitProposal(ctx context.Context, req *apiclient.SubmitProposalRequest) (*apiclient.SubmitProposalResponse, error) {
	requester, err := requireParticipantActor(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	if err := s.authorizeParticipant(ctx, req.GetSlug(), requester); err != nil {
		return nil, err
	}
	proposal, err := s.store.SubmitProposal(ctx, req.GetSlug(), requester, req.GetExpectedRevision())
	if err != nil {
		return nil, serviceError(err)
	}
	snapshot, err := s.store.GetRound(ctx, req.GetSlug(), requester)
	if err != nil {
		return nil, serviceError(err)
	}
	return &apiclient.SubmitProposalResponse{Proposal: proposalToProto(proposal, snapshot.Participants, requester, true, true, false, 0, false)}, nil
}

func (s *Service) ReopenProposal(ctx context.Context, req *apiclient.ReopenProposalRequest) (*apiclient.ReopenProposalResponse, error) {
	requester, err := requireParticipantActor(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	if err := s.authorizeParticipant(ctx, req.GetSlug(), requester); err != nil {
		return nil, err
	}
	proposal, err := s.store.ReopenProposal(ctx, req.GetSlug(), requester, req.GetExpectedRevision())
	if err != nil {
		return nil, serviceError(err)
	}
	snapshot, err := s.store.GetRound(ctx, req.GetSlug(), requester)
	if err != nil {
		return nil, serviceError(err)
	}
	return &apiclient.ReopenProposalResponse{Proposal: proposalToProto(proposal, snapshot.Participants, requester, true, true, false, 0, false)}, nil
}

func (s *Service) SubmitVote(ctx context.Context, req *apiclient.SubmitVoteRequest) (*apiclient.SubmitVoteResponse, error) {
	requester, err := requireParticipantActor(req.GetRequesterAccountId(), req.GetRequesterIsAdmin())
	if err != nil {
		return nil, err
	}
	if req.GetBallotNumber() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "ballot_number must be positive")
	}
	if strings.TrimSpace(req.GetProposalId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "proposal_id is required")
	}
	if err := s.authorizeParticipant(ctx, req.GetSlug(), requester); err != nil {
		return nil, err
	}
	vote, err := s.store.SubmitVote(ctx, req.GetSlug(), req.GetBallotNumber(), requester, req.GetProposalId())
	if err != nil {
		return nil, serviceError(err)
	}
	return &apiclient.SubmitVoteResponse{ProposalId: vote.ProposalID, BallotNumber: vote.BallotNumber}, nil
}

func (s *Service) authorizedRound(ctx context.Context, slug, requester string, requesterIsAdmin bool) (*profitsharingstore.RoundSnapshot, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}
	snapshot, err := s.store.GetRound(ctx, strings.TrimSpace(slug), requester)
	if err != nil {
		return nil, serviceError(err)
	}
	if !requesterIsAdmin && !containsParticipant(snapshot.Participants, requester) {
		return nil, status.Error(codes.PermissionDenied, "requester is not a participant in this round")
	}
	return snapshot, nil
}

func (s *Service) authorizeParticipant(ctx context.Context, slug, requester string) error {
	_, err := s.authorizedRound(ctx, slug, requester, false)
	return err
}

func requestIdentity(accountID string, requesterIsAdmin bool) (string, bool, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(accountID))
	if err != nil || parsed == uuid.Nil {
		return "", false, status.Error(codes.Unauthenticated, "requester_account_id must be a valid UUID")
	}
	return parsed.String(), requesterIsAdmin, nil
}

func requireAdmin(accountID string, requesterIsAdmin bool) (string, error) {
	accountID, _, err := requestIdentity(accountID, requesterIsAdmin)
	if err != nil {
		return "", err
	}
	if !requesterIsAdmin {
		return "", status.Error(codes.PermissionDenied, "administrator access is required")
	}
	return accountID, nil
}

func requireParticipantActor(accountID string, requesterIsAdmin bool) (string, error) {
	accountID, _, err := requestIdentity(accountID, requesterIsAdmin)
	if err != nil {
		return "", err
	}
	if requesterIsAdmin {
		return "", status.Error(codes.PermissionDenied, "administrators cannot propose or vote")
	}
	return accountID, nil
}

func participantInputsFromProto(inputs []*apiclient.ParticipantInput) ([]profitsharingstore.ParticipantInput, error) {
	converted := make([]profitsharingstore.ParticipantInput, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			return nil, status.Error(codes.InvalidArgument, "participant cannot be null")
		}
		converted = append(converted, profitsharingstore.ParticipantInput{
			AccountID:              input.GetAccountId(),
			Username:               input.GetUsername(),
			DisplayName:            input.GetDisplayName(),
			DisplayOrder:           input.GetDisplayOrder(),
			BaselineResponsibility: input.GetBaselineResponsibility(),
		})
	}
	return converted, nil
}

func proposalItemInputsFromProto(inputs []*apiclient.ProposalItemInput) ([]profitsharingstore.ProposalItemInput, error) {
	converted := make([]profitsharingstore.ProposalItemInput, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			return nil, status.Error(codes.InvalidArgument, "proposal item cannot be null")
		}
		var basisPoints *int32
		if input.GetBasisPointsSet() {
			value := input.GetBasisPoints()
			basisPoints = &value
		}
		converted = append(converted, profitsharingstore.ProposalItemInput{
			ParticipantAccountID: input.GetParticipantAccountId(),
			Responsibility:       input.GetResponsibility(),
			BasisPoints:          basisPoints,
		})
	}
	return converted, nil
}

func roundToProto(snapshot *profitsharingstore.RoundSnapshot, requester string, requesterIsAdmin bool) *apiclient.Round {
	result := &apiclient.Round{
		Summary:      roundSummaryToProto(snapshot),
		Participants: make([]*apiclient.Participant, 0, len(snapshot.Participants)),
	}
	if snapshot.Round.FinalProposalID != nil {
		result.FinalProposalId = *snapshot.Round.FinalProposalID
	}
	result.OpenedAtUnix = unixTime(snapshot.Round.OpenedAt)
	result.PublishedAtUnix = unixTime(snapshot.Round.PublishedAt)
	result.ClosedAtUnix = unixTime(snapshot.Round.ClosedAt)

	proposalByAuthor := make(map[string]*profitsharingstore.Proposal, len(snapshot.Proposals))
	for i := range snapshot.Proposals {
		proposalByAuthor[snapshot.Proposals[i].AuthorAccountID] = &snapshot.Proposals[i]
	}
	for _, participant := range snapshot.Participants {
		item := &apiclient.Participant{
			AccountId:              participant.AccountID,
			Username:               participant.Username,
			DisplayName:            participant.DisplayName,
			DisplayOrder:           participant.DisplayOrder,
			BaselineResponsibility: participant.BaselineResponsibility,
		}
		if requesterIsAdmin && snapshot.Round.Phase != profitsharingstore.RoundPhaseDraft {
			item.ProposalProgressVisible = true
			item.Submitted = proposalByAuthor[participant.AccountID] != nil && proposalByAuthor[participant.AccountID].Status == profitsharingstore.ProposalStatusSubmitted
		}
		result.Participants = append(result.Participants, item)
	}

	if !requesterIsAdmin && snapshot.Round.Phase == profitsharingstore.RoundPhaseCollecting {
		if proposal := proposalByAuthor[requester]; proposal != nil {
			result.OwnProposal = proposalToProto(proposal, snapshot.Participants, requester, true, true, false, 0, false)
		}
	}
	if snapshot.Ballot != nil && (snapshot.Round.Phase == profitsharingstore.RoundPhaseVoting || snapshot.Round.Phase == profitsharingstore.RoundPhaseClosed) {
		ballot := &apiclient.Ballot{
			Number:           snapshot.Ballot.Number,
			ParticipantCount: int32(len(snapshot.Participants)),
			VotedCount:       snapshot.Ballot.VotedCount,
			Candidates:       make([]*apiclient.Proposal, 0, len(snapshot.Ballot.Candidates)),
		}
		if snapshot.Ballot.CurrentVote != nil {
			ballot.CurrentVoteProposalId = snapshot.Ballot.CurrentVote.ProposalID
		}
		closed := snapshot.Round.Phase == profitsharingstore.RoundPhaseClosed
		for i := range snapshot.Ballot.Candidates {
			proposal := &snapshot.Ballot.Candidates[i]
			voteCount := snapshot.Ballot.VoteCounts[proposal.ID]
			isFinal := snapshot.Round.FinalProposalID != nil && *snapshot.Round.FinalProposalID == proposal.ID
			ballot.Candidates = append(ballot.Candidates, proposalToProto(proposal, snapshot.Participants, requester, closed, false, closed, voteCount, isFinal))
		}
		result.ActiveBallot = ballot
	}
	return result
}

func roundSummaryToProto(snapshot *profitsharingstore.RoundSnapshot) *apiclient.RoundSummary {
	submittedCount := int64(0)
	for _, proposal := range snapshot.Proposals {
		if proposal.Status == profitsharingstore.ProposalStatusSubmitted {
			submittedCount++
		}
	}
	result := &apiclient.RoundSummary{
		Slug:             snapshot.Round.Slug,
		Title:            snapshot.Round.Title,
		Phase:            roundPhaseToProto(snapshot.Round.Phase),
		Revision:         snapshot.Round.Revision,
		ParticipantCount: int32(len(snapshot.Participants)),
		SubmittedCount:   submittedCount,
		CreatedAtUnix:    snapshot.Round.CreatedAt.Unix(),
		UpdatedAtUnix:    snapshot.Round.UpdatedAt.Unix(),
	}
	if snapshot.Ballot != nil {
		result.ActiveBallotNumber = snapshot.Ballot.Number
		result.VotedCount = snapshot.Ballot.VotedCount
	}
	return result
}

func proposalToProto(proposal *profitsharingstore.Proposal, participants []profitsharingstore.Participant, requester string, showAuthor, showRevision, showVoteCount bool, voteCount int64, isFinal bool) *apiclient.Proposal {
	result := &apiclient.Proposal{
		Id:               proposal.ID,
		Status:           proposalStatusToProto(proposal.Status),
		IsOwn:            proposal.AuthorAccountID == requester,
		VoteCountVisible: showVoteCount,
		VoteCount:        voteCount,
		IsFinal:          isFinal,
		Items:            make([]*apiclient.ProposalItem, 0, len(proposal.Items)),
	}
	if showRevision {
		result.Revision = proposal.Revision
	}
	if proposal.AnonymousLabel != nil {
		result.Label = *proposal.AnonymousLabel
	}
	participantByAccount := make(map[string]profitsharingstore.Participant, len(participants))
	for _, participant := range participants {
		participantByAccount[participant.AccountID] = participant
	}
	if showAuthor {
		author := participantByAccount[proposal.AuthorAccountID]
		result.AuthorAccountId = proposal.AuthorAccountID
		result.AuthorUsername = author.Username
		result.AuthorDisplayName = author.DisplayName
	}
	for _, item := range proposal.Items {
		participant := participantByAccount[item.ParticipantAccountID]
		converted := &apiclient.ProposalItem{
			ParticipantAccountId:    item.ParticipantAccountID,
			ParticipantUsername:     participant.Username,
			ParticipantDisplayName:  participant.DisplayName,
			ParticipantDisplayOrder: participant.DisplayOrder,
			Responsibility:          item.Responsibility,
		}
		if item.BasisPoints != nil {
			converted.BasisPoints = *item.BasisPoints
			converted.BasisPointsSet = true
		}
		result.Items = append(result.Items, converted)
	}
	return result
}

func containsParticipant(participants []profitsharingstore.Participant, accountID string) bool {
	for _, participant := range participants {
		if participant.AccountID == accountID {
			return true
		}
	}
	return false
}

func roundPhaseToProto(phase string) apiclient.RoundPhase {
	switch phase {
	case profitsharingstore.RoundPhaseDraft:
		return apiclient.RoundPhase_ROUND_PHASE_DRAFT
	case profitsharingstore.RoundPhaseCollecting:
		return apiclient.RoundPhase_ROUND_PHASE_COLLECTING
	case profitsharingstore.RoundPhaseVoting:
		return apiclient.RoundPhase_ROUND_PHASE_VOTING
	case profitsharingstore.RoundPhaseClosed:
		return apiclient.RoundPhase_ROUND_PHASE_CLOSED
	default:
		return apiclient.RoundPhase_ROUND_PHASE_UNSPECIFIED
	}
}

func proposalStatusToProto(proposalStatus string) apiclient.ProposalStatus {
	switch proposalStatus {
	case profitsharingstore.ProposalStatusDraft:
		return apiclient.ProposalStatus_PROPOSAL_STATUS_DRAFT
	case profitsharingstore.ProposalStatusSubmitted:
		return apiclient.ProposalStatus_PROPOSAL_STATUS_SUBMITTED
	default:
		return apiclient.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED
	}
}

func unixTime(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}

func serviceError(err error) error {
	switch {
	case errors.Is(err, profitsharingstore.ErrRoundNotFound), errors.Is(err, profitsharingstore.ErrProposalNotFound), errors.Is(err, profitsharingstore.ErrBallotNotFound), errors.Is(err, profitsharingstore.ErrCandidateNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, profitsharingstore.ErrRoundAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, profitsharingstore.ErrRevisionConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, profitsharingstore.ErrNotParticipant), errors.Is(err, profitsharingstore.ErrSelfVote):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, profitsharingstore.ErrInvalidParticipants), errors.Is(err, profitsharingstore.ErrInvalidProposal):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, profitsharingstore.ErrInvalidPhase), errors.Is(err, profitsharingstore.ErrIncompleteProposal), errors.Is(err, profitsharingstore.ErrNotAllProposalsSubmitted), errors.Is(err, profitsharingstore.ErrNotAllVotesSubmitted), errors.Is(err, profitsharingstore.ErrNotConfigured):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
