package profitsharing

import (
	"context"
	"strconv"
	"strings"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcenter"
	"github.com/useryege/athena/internal/accountcredentials"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	profitsharingapiclient "github.com/useryege/athena/internal/profitsharing/apiclient"
	profitsharingpkg "github.com/useryege/athena/pkg/apiclient/profitsharing"
	utilsession "github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const requiredParticipantCount = 5

type Server struct {
	profitsharingpkg.UnimplementedProfitSharingServiceServer
	clientset        profitsharingapiclient.Clientset
	credentialMgr    *accountcredentials.CredentialManager
	accessController *accountaccess.Controller
	accountCenter    *accountcenter.Manager
}

func NewServer(
	clientset profitsharingapiclient.Clientset,
	credentialMgr *accountcredentials.CredentialManager,
	accessController *accountaccess.Controller,
	accountCenter *accountcenter.Manager,
) *Server {
	return &Server{
		clientset:        clientset,
		credentialMgr:    credentialMgr,
		accessController: accessController,
		accountCenter:    accountCenter,
	}
}

func (s *Server) ListRounds(ctx context.Context, _ *profitsharingpkg.ListRoundsRequest) (*profitsharingpkg.ListRoundsResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().ListRounds(ctx, &profitsharingapiclient.ListRoundsRequest{
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}

	rounds := make([]*profitsharingpkg.Round, 0, len(response.GetRounds()))
	for _, summary := range response.GetRounds() {
		rounds = append(rounds, projectRoundSummary(summary))
	}
	return &profitsharingpkg.ListRoundsResponse{Rounds: rounds}, nil
}

func (s *Server) GetRound(ctx context.Context, request *profitsharingpkg.GetRoundRequest) (*profitsharingpkg.GetRoundResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().GetRound(ctx, &profitsharingapiclient.GetRoundRequest{
		Slug:               request.GetSlug(),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.GetRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) CreateRound(ctx context.Context, request *profitsharingpkg.CreateRoundRequest) (*profitsharingpkg.CreateRoundResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	participants, err := s.canonicalParticipantInputs(ctx, request.GetParticipants())
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().CreateRound(ctx, &profitsharingapiclient.CreateRoundRequest{
		Slug:               request.GetSlug(),
		Title:              request.GetTitle(),
		Participants:       participants,
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	captureProfitSharingRound(ctx, response.GetRound(), "PROFIT_SHARING_ROUND_CREATE")
	return &profitsharingpkg.CreateRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) UpdateRound(ctx context.Context, request *profitsharingpkg.UpdateRoundRequest) (*profitsharingpkg.UpdateRoundResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	participants, err := s.canonicalParticipantInputs(ctx, request.GetParticipants())
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().UpdateRound(ctx, &profitsharingapiclient.UpdateRoundRequest{
		CurrentSlug:        request.GetCurrentSlug(),
		Slug:               request.GetSlug(),
		Title:              request.GetTitle(),
		Participants:       participants,
		ExpectedRevision:   request.GetExpectedRevision(),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	captureProfitSharingRound(ctx, response.GetRound(), "PROFIT_SHARING_ROUND_UPDATE")
	operationlogrecord.CaptureString(ctx, "oldSlug", request.GetCurrentSlug())
	operationlogrecord.CaptureStrings(ctx, "changedFields", profitSharingRoundChangedFields(request))
	return &profitsharingpkg.UpdateRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) OpenRound(ctx context.Context, request *profitsharingpkg.OpenRoundRequest) (*profitsharingpkg.OpenRoundResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	roundResponse, err := s.clientset.ProfitSharing().GetRound(ctx, &profitsharingapiclient.GetRoundRequest{
		Slug:               request.GetSlug(),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	validatedAccountIDs, err := s.validateParticipants(roundResponse.GetRound())
	if err != nil {
		return nil, err
	}

	response, err := s.clientset.ProfitSharing().OpenRound(ctx, &profitsharingapiclient.OpenRoundRequest{
		Slug:                           request.GetSlug(),
		ExpectedRevision:               request.GetExpectedRevision(),
		RequesterAccountId:             requesterAccountID,
		RequesterIsAdmin:               requesterIsAdmin,
		ValidatedParticipantAccountIds: validatedAccountIDs,
	})
	if err != nil {
		return nil, err
	}
	captureProfitSharingRound(ctx, response.GetRound(), "PROFIT_SHARING_ROUND_OPEN")
	return &profitsharingpkg.OpenRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) PublishRound(ctx context.Context, request *profitsharingpkg.PublishRoundRequest) (*profitsharingpkg.PublishRoundResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().PublishRound(ctx, &profitsharingapiclient.PublishRoundRequest{
		Slug:               request.GetSlug(),
		ExpectedRevision:   request.GetExpectedRevision(),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	captureProfitSharingRound(ctx, response.GetRound(), "PROFIT_SHARING_ROUND_PUBLISH")
	return &profitsharingpkg.PublishRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) CloseBallot(ctx context.Context, request *profitsharingpkg.CloseBallotRequest) (*profitsharingpkg.CloseBallotResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().CloseBallot(ctx, &profitsharingapiclient.CloseBallotRequest{
		Slug:               request.GetSlug(),
		ExpectedRevision:   request.GetExpectedRevision(),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	captureProfitSharingRound(ctx, response.GetRound(), "PROFIT_SHARING_BALLOT_CLOSE")
	return &profitsharingpkg.CloseBallotResponse{
		Round:         projectRound(response.GetRound()),
		RunoffCreated: response.GetRunoffCreated(),
	}, nil
}

func (s *Server) UpdateProposal(ctx context.Context, request *profitsharingpkg.UpdateProposalRequest) (*profitsharingpkg.UpdateProposalResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().UpdateProposal(ctx, &profitsharingapiclient.UpdateProposalRequest{
		Slug:               request.GetSlug(),
		ExpectedRevision:   request.GetExpectedRevision(),
		Items:              projectProposalItemInputs(request.GetItems()),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	captureProfitSharingProposal(ctx, response.GetProposal(), "PROFIT_SHARING_PROPOSAL_UPDATE")
	operationlogrecord.CaptureString(ctx, "slug", request.GetSlug())
	operationlogrecord.CaptureStrings(ctx, "changedFields", []string{"items"})
	return &profitsharingpkg.UpdateProposalResponse{Proposal: projectProposal(response.GetProposal())}, nil
}

func (s *Server) SubmitProposal(ctx context.Context, request *profitsharingpkg.SubmitProposalRequest) (*profitsharingpkg.SubmitProposalResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().SubmitProposal(ctx, &profitsharingapiclient.SubmitProposalRequest{
		Slug:               request.GetSlug(),
		ExpectedRevision:   request.GetExpectedRevision(),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	captureProfitSharingProposal(ctx, response.GetProposal(), "PROFIT_SHARING_PROPOSAL_SUBMIT")
	operationlogrecord.CaptureString(ctx, "slug", request.GetSlug())
	return &profitsharingpkg.SubmitProposalResponse{Proposal: projectProposal(response.GetProposal())}, nil
}

func (s *Server) ReopenProposal(ctx context.Context, request *profitsharingpkg.ReopenProposalRequest) (*profitsharingpkg.ReopenProposalResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().ReopenProposal(ctx, &profitsharingapiclient.ReopenProposalRequest{
		Slug:               request.GetSlug(),
		ExpectedRevision:   request.GetExpectedRevision(),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	captureProfitSharingProposal(ctx, response.GetProposal(), "PROFIT_SHARING_PROPOSAL_REOPEN")
	operationlogrecord.CaptureString(ctx, "slug", request.GetSlug())
	return &profitsharingpkg.ReopenProposalResponse{Proposal: projectProposal(response.GetProposal())}, nil
}

func (s *Server) SubmitVote(ctx context.Context, request *profitsharingpkg.SubmitVoteRequest) (*profitsharingpkg.SubmitVoteResponse, error) {
	requesterAccountID, requesterIsAdmin, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	roundResponse, err := s.clientset.ProfitSharing().GetRound(ctx, &profitsharingapiclient.GetRoundRequest{
		Slug:               request.GetSlug(),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	activeBallot := roundResponse.GetRound().GetActiveBallot()
	if activeBallot == nil || activeBallot.GetNumber() <= 0 {
		return nil, status.Error(codes.FailedPrecondition, "profit sharing round has no active ballot")
	}

	response, err := s.clientset.ProfitSharing().SubmitVote(ctx, &profitsharingapiclient.SubmitVoteRequest{
		Slug:               request.GetSlug(),
		BallotNumber:       activeBallot.GetNumber(),
		ProposalId:         request.GetProposalId(),
		RequesterAccountId: requesterAccountID,
		RequesterIsAdmin:   requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	operationlogrecord.CaptureString(ctx, "slug", request.GetSlug())
	operationlogrecord.CaptureInt64(ctx, "ballotId", int64(response.GetBallotNumber()))
	operationlogrecord.CaptureResource(ctx, "ballot", strconv.Itoa(int(response.GetBallotNumber())))
	operationlogrecord.Commit(ctx, "PROFIT_SHARING_VOTE_SUBMIT")
	return &profitsharingpkg.SubmitVoteResponse{
		ProposalId:   response.GetProposalId(),
		BallotNumber: response.GetBallotNumber(),
	}, nil
}

func (s *Server) requester(ctx context.Context) (string, bool, error) {
	accountID := utilsession.AccountID(ctx)
	if accountID == "" {
		return "", false, status.Error(codes.Unauthenticated, "authenticated account is missing")
	}
	if s.credentialMgr == nil {
		return "", false, status.Error(codes.Internal, "profit sharing account directory is not configured")
	}
	account, err := s.credentialMgr.Get(accountID)
	if err != nil {
		return "", false, status.Error(codes.Unauthenticated, "authenticated account is not registered")
	}
	return account.ID, account.Administrator, nil
}

func captureProfitSharingRound(ctx context.Context, round *profitsharingapiclient.Round, effect string) {
	if round == nil || round.GetSummary() == nil {
		return
	}
	summary := round.GetSummary()
	operationlogrecord.CaptureResource(ctx, "round", summary.GetSlug())
	switch effect {
	case "PROFIT_SHARING_ROUND_CREATE", "PROFIT_SHARING_ROUND_OPEN", "PROFIT_SHARING_ROUND_PUBLISH":
		operationlogrecord.CaptureString(ctx, "slug", summary.GetSlug())
		operationlogrecord.CaptureInt64(ctx, "revision", summary.GetRevision())
		operationlogrecord.CaptureString(ctx, "state", profitSharingRoundPhase(summary.GetPhase()))
	case "PROFIT_SHARING_ROUND_UPDATE":
		operationlogrecord.CaptureString(ctx, "newSlug", summary.GetSlug())
		operationlogrecord.CaptureInt64(ctx, "revision", summary.GetRevision())
	case "PROFIT_SHARING_BALLOT_CLOSE":
		operationlogrecord.CaptureString(ctx, "slug", summary.GetSlug())
		if summary.GetActiveBallotNumber() > 0 {
			operationlogrecord.CaptureInt64(ctx, "ballotId", int64(summary.GetActiveBallotNumber()))
		}
		operationlogrecord.CaptureString(ctx, "state", profitSharingRoundPhase(summary.GetPhase()))
	}
	operationlogrecord.Commit(ctx, effect)
}

func captureProfitSharingProposal(ctx context.Context, proposal *profitsharingapiclient.Proposal, effect string) {
	if proposal == nil {
		return
	}
	operationlogrecord.CaptureResource(ctx, "proposal", proposal.GetId())
	operationlogrecord.CaptureInt64(ctx, "revision", proposal.GetRevision())
	if effect != "PROFIT_SHARING_PROPOSAL_UPDATE" {
		operationlogrecord.CaptureString(ctx, "state", profitSharingProposalStatus(proposal.GetStatus()))
	}
	operationlogrecord.Commit(ctx, effect)
}

func profitSharingRoundChangedFields(request *profitsharingpkg.UpdateRoundRequest) []string {
	fields := make([]string, 0, 3)
	if request.GetCurrentSlug() != request.GetSlug() {
		fields = append(fields, "slug")
	}
	if request.GetTitle() != "" {
		fields = append(fields, "title")
	}
	if len(request.GetParticipants()) > 0 {
		fields = append(fields, "participants")
	}
	return fields
}

func profitSharingRoundPhase(value profitsharingapiclient.RoundPhase) string {
	switch value {
	case profitsharingapiclient.RoundPhase_ROUND_PHASE_DRAFT:
		return "draft"
	case profitsharingapiclient.RoundPhase_ROUND_PHASE_COLLECTING:
		return "collecting"
	case profitsharingapiclient.RoundPhase_ROUND_PHASE_VOTING:
		return "voting"
	case profitsharingapiclient.RoundPhase_ROUND_PHASE_CLOSED:
		return "closed"
	default:
		return ""
	}
}

func profitSharingProposalStatus(value profitsharingapiclient.ProposalStatus) string {
	switch value {
	case profitsharingapiclient.ProposalStatus_PROPOSAL_STATUS_DRAFT:
		return "draft"
	case profitsharingapiclient.ProposalStatus_PROPOSAL_STATUS_SUBMITTED:
		return "submitted"
	default:
		return ""
	}
}

func (s *Server) validateParticipants(round *profitsharingapiclient.Round) ([]string, error) {
	if round == nil || round.GetSummary() == nil {
		return nil, status.Error(codes.Internal, "profit sharing round projection is missing")
	}
	if round.GetSummary().GetPhase() != profitsharingapiclient.RoundPhase_ROUND_PHASE_DRAFT {
		return nil, status.Error(codes.FailedPrecondition, "only a draft profit sharing round can be opened")
	}
	if len(round.GetParticipants()) != requiredParticipantCount {
		return nil, status.Errorf(codes.FailedPrecondition, "profit sharing round must contain exactly %d participants", requiredParticipantCount)
	}
	if s.credentialMgr == nil || s.accessController == nil {
		return nil, status.Error(codes.Internal, "profit sharing participant account validation is not configured")
	}

	validatedAccountIDs := make([]string, 0, len(round.GetParticipants()))
	seen := make(map[string]struct{}, len(round.GetParticipants()))
	for _, participant := range round.GetParticipants() {
		accountID := strings.TrimSpace(participant.GetAccountId())
		account, err := s.credentialMgr.Get(accountID)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q is not registered", accountID)
		}
		if account.Administrator {
			return nil, status.Error(codes.FailedPrecondition, "the administrator cannot be a profit sharing participant")
		}
		if !account.HasExternalIdentity() {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q has no external login identity", account.ID)
		}
		if _, exists := seen[account.ID]; exists {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q is duplicated", account.ID)
		}
		seen[account.ID] = struct{}{}

		access, err := s.accessController.Get(account.ID)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q has no access configuration", account.ID)
		}
		if access.Administrator {
			return nil, status.Error(codes.FailedPrecondition, "the administrator cannot be a profit sharing participant")
		}
		if !access.LoginEnabled {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q has login disabled", account.ID)
		}
		if !access.ProfitSharingEnabled {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q is not authorized for Profit Sharing", account.ID)
		}
		validatedAccountIDs = append(validatedAccountIDs, account.ID)
	}
	return validatedAccountIDs, nil
}

func (s *Server) canonicalParticipantInputs(ctx context.Context, inputs []*profitsharingpkg.ParticipantInput) ([]*profitsharingapiclient.ParticipantInput, error) {
	if s.credentialMgr == nil || s.accountCenter == nil {
		return nil, status.Error(codes.Internal, "profit sharing account projection is not configured")
	}
	result := make([]*profitsharingapiclient.ParticipantInput, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			return nil, status.Error(codes.InvalidArgument, "profit sharing participant cannot be null")
		}
		account, err := s.credentialMgr.Get(input.GetAccountId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "profit sharing participant account %q is not registered", input.GetAccountId())
		}
		if account.Administrator {
			return nil, status.Error(codes.InvalidArgument, "the administrator cannot be a profit sharing participant")
		}
		profile, err := s.accountCenter.GetProfile(ctx, account.ID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load canonical profile for profit sharing participant %q: %v", account.ID, err)
		}
		result = append(result, &profitsharingapiclient.ParticipantInput{
			AccountId:              account.ID,
			Username:               account.Username,
			DisplayName:            profile.DisplayName,
			DisplayOrder:           input.GetSortOrder(),
			BaselineResponsibility: input.GetBaselineResponsibility(),
		})
	}
	return result, nil
}

func projectProposalItemInputs(inputs []*profitsharingpkg.ProposalItemInput) []*profitsharingapiclient.ProposalItemInput {
	result := make([]*profitsharingapiclient.ProposalItemInput, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, &profitsharingapiclient.ProposalItemInput{
			ParticipantAccountId: input.GetParticipantAccountId(),
			Responsibility:       input.GetResponsibility(),
			BasisPoints:          input.GetShareBasisPoints(),
			BasisPointsSet:       input.GetShareBasisPointsSet(),
		})
	}
	return result
}

func projectRound(round *profitsharingapiclient.Round) *profitsharingpkg.Round {
	if round == nil {
		return nil
	}
	result := projectRoundSummary(round.GetSummary())
	if result == nil {
		result = &profitsharingpkg.Round{}
	}
	result.Participants = projectParticipants(round.GetParticipants())
	result.MyProposal = projectProposal(round.GetOwnProposal())
	result.WinnerProposalId = round.GetFinalProposalId()

	if ballot := round.GetActiveBallot(); ballot != nil {
		result.BallotNumber = ballot.GetNumber()
		result.VotedCount = ballot.GetVotedCount()
		result.MyVoteProposalId = ballot.GetCurrentVoteProposalId()
		result.Proposals = projectProposals(ballot.GetCandidates())
		result.Results = projectResults(ballot.GetCandidates())
	}
	return result
}

func projectRoundSummary(summary *profitsharingapiclient.RoundSummary) *profitsharingpkg.Round {
	if summary == nil {
		return nil
	}
	return &profitsharingpkg.Round{
		Slug:             summary.GetSlug(),
		Title:            summary.GetTitle(),
		Phase:            profitsharingpkg.RoundPhase(summary.GetPhase()),
		Revision:         summary.GetRevision(),
		ParticipantCount: summary.GetParticipantCount(),
		SubmittedCount:   summary.GetSubmittedCount(),
		VotedCount:       summary.GetVotedCount(),
		BallotNumber:     summary.GetActiveBallotNumber(),
	}
}

func projectParticipants(participants []*profitsharingapiclient.Participant) []*profitsharingpkg.Participant {
	result := make([]*profitsharingpkg.Participant, 0, len(participants))
	for _, participant := range participants {
		proposalStatus := profitsharingpkg.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED
		if participant.GetProposalProgressVisible() {
			proposalStatus = profitsharingpkg.ProposalStatus_PROPOSAL_STATUS_DRAFT
			if participant.GetSubmitted() {
				proposalStatus = profitsharingpkg.ProposalStatus_PROPOSAL_STATUS_SUBMITTED
			}
		}
		result = append(result, &profitsharingpkg.Participant{
			AccountId:              participant.GetAccountId(),
			Username:               participant.GetUsername(),
			DisplayName:            participant.GetDisplayName(),
			BaselineResponsibility: participant.GetBaselineResponsibility(),
			SortOrder:              participant.GetDisplayOrder(),
			ProposalStatus:         proposalStatus,
		})
	}
	return result
}

func projectProposals(proposals []*profitsharingapiclient.Proposal) []*profitsharingpkg.Proposal {
	result := make([]*profitsharingpkg.Proposal, 0, len(proposals))
	for _, proposal := range proposals {
		result = append(result, projectProposal(proposal))
	}
	return result
}

func projectProposal(proposal *profitsharingapiclient.Proposal) *profitsharingpkg.Proposal {
	if proposal == nil {
		return nil
	}
	items := make([]*profitsharingpkg.ProposalItem, 0, len(proposal.GetItems()))
	for _, item := range proposal.GetItems() {
		items = append(items, &profitsharingpkg.ProposalItem{
			ParticipantAccountId:   item.GetParticipantAccountId(),
			ParticipantUsername:    item.GetParticipantUsername(),
			ParticipantDisplayName: item.GetParticipantDisplayName(),
			Responsibility:         item.GetResponsibility(),
			ShareBasisPoints:       item.GetBasisPoints(),
			ShareBasisPointsSet:    item.GetBasisPointsSet(),
		})
	}
	return &profitsharingpkg.Proposal{
		Id:                proposal.GetId(),
		Label:             proposal.GetLabel(),
		IsOwn:             proposal.GetIsOwn(),
		AuthorAccountId:   proposal.GetAuthorAccountId(),
		AuthorUsername:    proposal.GetAuthorUsername(),
		AuthorDisplayName: proposal.GetAuthorDisplayName(),
		Status:            profitsharingpkg.ProposalStatus(proposal.GetStatus()),
		Revision:          proposal.GetRevision(),
		Items:             items,
	}
}

func projectResults(proposals []*profitsharingapiclient.Proposal) []*profitsharingpkg.Result {
	results := make([]*profitsharingpkg.Result, 0, len(proposals))
	for _, proposal := range proposals {
		if !proposal.GetVoteCountVisible() {
			continue
		}
		results = append(results, &profitsharingpkg.Result{
			ProposalId:        proposal.GetId(),
			Label:             proposal.GetLabel(),
			AuthorAccountId:   proposal.GetAuthorAccountId(),
			AuthorUsername:    proposal.GetAuthorUsername(),
			AuthorDisplayName: proposal.GetAuthorDisplayName(),
			VoteCount:         proposal.GetVoteCount(),
			IsWinner:          proposal.GetIsFinal(),
		})
	}
	return results
}
