package profitsharing

import (
	"context"
	"strings"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
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
}

func NewServer(
	clientset profitsharingapiclient.Clientset,
	credentialMgr *accountcredentials.CredentialManager,
	accessController *accountaccess.Controller,
) *Server {
	return &Server{
		clientset:        clientset,
		credentialMgr:    credentialMgr,
		accessController: accessController,
	}
}

func (s *Server) ListRounds(ctx context.Context, _ *profitsharingpkg.ListRoundsRequest) (*profitsharingpkg.ListRoundsResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().ListRounds(ctx, &profitsharingapiclient.ListRoundsRequest{
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
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
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().GetRound(ctx, &profitsharingapiclient.GetRoundRequest{
		Slug:             request.GetSlug(),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.GetRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) CreateRound(ctx context.Context, request *profitsharingpkg.CreateRoundRequest) (*profitsharingpkg.CreateRoundResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().CreateRound(ctx, &profitsharingapiclient.CreateRoundRequest{
		Slug:             request.GetSlug(),
		Title:            request.GetTitle(),
		Participants:     projectParticipantInputs(request.GetParticipants()),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.CreateRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) UpdateRound(ctx context.Context, request *profitsharingpkg.UpdateRoundRequest) (*profitsharingpkg.UpdateRoundResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().UpdateRound(ctx, &profitsharingapiclient.UpdateRoundRequest{
		CurrentSlug:      request.GetCurrentSlug(),
		Slug:             request.GetSlug(),
		Title:            request.GetTitle(),
		Participants:     projectParticipantInputs(request.GetParticipants()),
		ExpectedRevision: request.GetExpectedRevision(),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.UpdateRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) OpenRound(ctx context.Context, request *profitsharingpkg.OpenRoundRequest) (*profitsharingpkg.OpenRoundResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	roundResponse, err := s.clientset.ProfitSharing().GetRound(ctx, &profitsharingapiclient.GetRoundRequest{
		Slug:             request.GetSlug(),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	validatedAccounts, err := s.validateParticipants(roundResponse.GetRound())
	if err != nil {
		return nil, err
	}

	response, err := s.clientset.ProfitSharing().OpenRound(ctx, &profitsharingapiclient.OpenRoundRequest{
		Slug:                         request.GetSlug(),
		ExpectedRevision:             request.GetExpectedRevision(),
		RequesterAccount:             requesterAccount,
		RequesterIsAdmin:             requesterIsAdmin,
		ValidatedParticipantAccounts: validatedAccounts,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.OpenRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) PublishRound(ctx context.Context, request *profitsharingpkg.PublishRoundRequest) (*profitsharingpkg.PublishRoundResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().PublishRound(ctx, &profitsharingapiclient.PublishRoundRequest{
		Slug:             request.GetSlug(),
		ExpectedRevision: request.GetExpectedRevision(),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.PublishRoundResponse{Round: projectRound(response.GetRound())}, nil
}

func (s *Server) CloseBallot(ctx context.Context, request *profitsharingpkg.CloseBallotRequest) (*profitsharingpkg.CloseBallotResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().CloseBallot(ctx, &profitsharingapiclient.CloseBallotRequest{
		Slug:             request.GetSlug(),
		ExpectedRevision: request.GetExpectedRevision(),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.CloseBallotResponse{
		Round:         projectRound(response.GetRound()),
		RunoffCreated: response.GetRunoffCreated(),
	}, nil
}

func (s *Server) UpdateProposal(ctx context.Context, request *profitsharingpkg.UpdateProposalRequest) (*profitsharingpkg.UpdateProposalResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().UpdateProposal(ctx, &profitsharingapiclient.UpdateProposalRequest{
		Slug:             request.GetSlug(),
		ExpectedRevision: request.GetExpectedRevision(),
		Items:            projectProposalItemInputs(request.GetItems()),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.UpdateProposalResponse{Proposal: projectProposal(response.GetProposal())}, nil
}

func (s *Server) SubmitProposal(ctx context.Context, request *profitsharingpkg.SubmitProposalRequest) (*profitsharingpkg.SubmitProposalResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().SubmitProposal(ctx, &profitsharingapiclient.SubmitProposalRequest{
		Slug:             request.GetSlug(),
		ExpectedRevision: request.GetExpectedRevision(),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.SubmitProposalResponse{Proposal: projectProposal(response.GetProposal())}, nil
}

func (s *Server) ReopenProposal(ctx context.Context, request *profitsharingpkg.ReopenProposalRequest) (*profitsharingpkg.ReopenProposalResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.clientset.ProfitSharing().ReopenProposal(ctx, &profitsharingapiclient.ReopenProposalRequest{
		Slug:             request.GetSlug(),
		ExpectedRevision: request.GetExpectedRevision(),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.ReopenProposalResponse{Proposal: projectProposal(response.GetProposal())}, nil
}

func (s *Server) SubmitVote(ctx context.Context, request *profitsharingpkg.SubmitVoteRequest) (*profitsharingpkg.SubmitVoteResponse, error) {
	requesterAccount, requesterIsAdmin, err := requester(ctx)
	if err != nil {
		return nil, err
	}
	roundResponse, err := s.clientset.ProfitSharing().GetRound(ctx, &profitsharingapiclient.GetRoundRequest{
		Slug:             request.GetSlug(),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	activeBallot := roundResponse.GetRound().GetActiveBallot()
	if activeBallot == nil || activeBallot.GetNumber() <= 0 {
		return nil, status.Error(codes.FailedPrecondition, "profit sharing round has no active ballot")
	}

	response, err := s.clientset.ProfitSharing().SubmitVote(ctx, &profitsharingapiclient.SubmitVoteRequest{
		Slug:             request.GetSlug(),
		BallotNumber:     activeBallot.GetNumber(),
		ProposalId:       request.GetProposalId(),
		RequesterAccount: requesterAccount,
		RequesterIsAdmin: requesterIsAdmin,
	})
	if err != nil {
		return nil, err
	}
	return &profitsharingpkg.SubmitVoteResponse{
		ProposalId:   response.GetProposalId(),
		BallotNumber: response.GetBallotNumber(),
	}, nil
}

func requester(ctx context.Context) (string, bool, error) {
	account := utilsession.GetUserIdentifier(ctx)
	if account == "" {
		return "", false, status.Error(codes.Unauthenticated, "authenticated account is missing")
	}
	return account, account == common.AthenaAdminUsername, nil
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

	validated := make([]string, 0, len(round.GetParticipants()))
	seen := make(map[string]struct{}, len(round.GetParticipants()))
	for _, participant := range round.GetParticipants() {
		accountName := participant.GetAccount()
		if strings.TrimSpace(accountName) == "" {
			return nil, status.Error(codes.FailedPrecondition, "profit sharing participant account is empty")
		}
		if accountName == common.AthenaAdminUsername {
			return nil, status.Error(codes.FailedPrecondition, "the administrator cannot be a profit sharing participant")
		}
		if _, exists := seen[accountName]; exists {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q is duplicated", accountName)
		}
		seen[accountName] = struct{}{}

		account, err := s.credentialMgr.Get(accountName)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q is not configured", accountName)
		}
		if !account.HasCapability(accountcredentials.CapabilityLogin) {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q does not have login capability", accountName)
		}
		if !account.HasGoogleBinding() {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q does not have a Google identity binding", accountName)
		}
		access, err := s.accessController.Get(accountName)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q has no access configuration", accountName)
		}
		if !access.LoginEnabled {
			return nil, status.Errorf(codes.FailedPrecondition, "profit sharing participant account %q has login disabled", accountName)
		}
		validated = append(validated, accountName)
	}
	return validated, nil
}

func projectParticipantInputs(inputs []*profitsharingpkg.ParticipantInput) []*profitsharingapiclient.ParticipantInput {
	result := make([]*profitsharingapiclient.ParticipantInput, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, &profitsharingapiclient.ParticipantInput{
			Account:                input.GetAccountName(),
			DisplayName:            input.GetDisplayName(),
			DisplayOrder:           input.GetSortOrder(),
			BaselineResponsibility: input.GetBaselineResponsibility(),
		})
	}
	return result
}

func projectProposalItemInputs(inputs []*profitsharingpkg.ProposalItemInput) []*profitsharingapiclient.ProposalItemInput {
	result := make([]*profitsharingapiclient.ProposalItemInput, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, &profitsharingapiclient.ProposalItemInput{
			ParticipantAccount: input.GetAccountName(),
			Responsibility:     input.GetResponsibility(),
			BasisPoints:        input.GetShareBasisPoints(),
			BasisPointsSet:     input.GetShareBasisPointsSet(),
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
			AccountName:            participant.GetAccount(),
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
			AccountName:         item.GetParticipantAccount(),
			DisplayName:         item.GetParticipantDisplayName(),
			Responsibility:      item.GetResponsibility(),
			ShareBasisPoints:    item.GetBasisPoints(),
			ShareBasisPointsSet: item.GetBasisPointsSet(),
		})
	}
	return &profitsharingpkg.Proposal{
		Id:                proposal.GetId(),
		Label:             proposal.GetLabel(),
		IsOwn:             proposal.GetIsOwn(),
		AuthorAccount:     proposal.GetAuthorAccount(),
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
			AuthorAccount:     proposal.GetAuthorAccount(),
			AuthorDisplayName: proposal.GetAuthorDisplayName(),
			VoteCount:         proposal.GetVoteCount(),
			IsWinner:          proposal.GetIsFinal(),
		})
	}
	return results
}
