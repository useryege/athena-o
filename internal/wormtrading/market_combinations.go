package wormtrading

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxMarketCombinationPageSize = 100

func (s *Service) CreateMarketCombination(
	ctx context.Context,
	req *apiclient.CreateMarketCombinationRequest,
) (*apiclient.CreateMarketCombinationResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	items := marketCombinationInputsFromProto(req.GetItems())
	combination, err := s.credentialStore.CreateMarketCombination(ctx, ownerAccountID, req.GetName(), items)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, marketCombinationRPCError(err)
	}
	return &apiclient.CreateMarketCombinationResponse{Combination: marketCombinationToProto(combination)}, nil
}

func (s *Service) GetMarketCombination(
	ctx context.Context,
	req *apiclient.GetMarketCombinationRequest,
) (*apiclient.GetMarketCombinationResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	combinationID, err := normalizeMarketCombinationID(req.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	combination, err := s.credentialStore.GetMarketCombination(ctx, ownerAccountID, combinationID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, marketCombinationRPCError(err)
	}
	return &apiclient.GetMarketCombinationResponse{Combination: marketCombinationToProto(combination)}, nil
}

func (s *Service) ListMarketCombinations(
	ctx context.Context,
	req *apiclient.ListMarketCombinationsRequest,
) (*apiclient.ListMarketCombinationsResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	if req.GetPage() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "page must be positive")
	}
	if req.GetPageSize() <= 0 || req.GetPageSize() > maxMarketCombinationPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "page_size must be between 1 and %d", maxMarketCombinationPageSize)
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	combinations, total, err := s.credentialStore.ListMarketCombinations(ctx, ownerAccountID, req.GetPage(), req.GetPageSize())
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, marketCombinationRPCError(err)
	}
	items := make([]*apiclient.MarketCombination, 0, len(combinations))
	for index := range combinations {
		items = append(items, marketCombinationToProto(&combinations[index]))
	}
	return &apiclient.ListMarketCombinationsResponse{
		Items:    items,
		Total:    total,
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}, nil
}

func (s *Service) UpdateMarketCombination(
	ctx context.Context,
	req *apiclient.UpdateMarketCombinationRequest,
) (*apiclient.UpdateMarketCombinationResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	combinationID, err := normalizeMarketCombinationID(req.GetId())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	items := marketCombinationInputsFromProto(req.GetItems())
	combination, err := s.credentialStore.UpdateMarketCombination(
		ctx,
		ownerAccountID,
		combinationID,
		req.GetName(),
		req.GetExpectedRevision(),
		items,
	)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, marketCombinationRPCError(err)
	}
	return &apiclient.UpdateMarketCombinationResponse{Combination: marketCombinationToProto(combination)}, nil
}

func (s *Service) DeleteMarketCombination(
	ctx context.Context,
	req *apiclient.DeleteMarketCombinationRequest,
) (*apiclient.DeleteMarketCombinationResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	combinationID, err := normalizeMarketCombinationID(req.GetId())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	err = s.credentialStore.DeleteMarketCombination(ctx, ownerAccountID, combinationID, req.GetExpectedRevision())
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, marketCombinationRPCError(err)
	}
	return &apiclient.DeleteMarketCombinationResponse{}, nil
}

func (s *Service) requireMarketCombinationStore() error {
	if !s.isStarted() {
		return status.Error(codes.FailedPrecondition, "Worm Trading service is not started")
	}
	if s.credentialStore == nil {
		return status.Error(codes.FailedPrecondition, "Worm Trading store is not configured")
	}
	return nil
}

func normalizeMarketCombinationAccountID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return "", status.Error(codes.InvalidArgument, "owner_account_id must be a valid non-zero UUID")
	}
	return parsed.String(), nil
}

func normalizeMarketCombinationID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return "", status.Error(codes.InvalidArgument, "id must be a valid non-zero UUID")
	}
	return parsed.String(), nil
}

func marketCombinationInputsFromProto(items []*apiclient.MarketCombinationItemInput) []wormstore.MarketCombinationItemInput {
	converted := make([]wormstore.MarketCombinationItemInput, 0, len(items))
	for _, item := range items {
		if item == nil {
			converted = append(converted, wormstore.MarketCombinationItemInput{})
			continue
		}
		converted = append(converted, wormstore.MarketCombinationItemInput{
			EventConditionID:  item.GetEventConditionId(),
			EventTitle:        item.GetEventTitle(),
			EventLogo:         item.GetEventLogo(),
			MarketConditionID: item.GetMarketConditionId(),
			MarketTitle:       item.GetMarketTitle(),
			MarketLogo:        item.GetMarketLogo(),
			IsYes:             item.GetIsYes(),
			OutcomeLabel:      item.GetOutcomeLabel(),
		})
	}
	return converted
}

func marketCombinationToProto(combination *wormstore.MarketCombination) *apiclient.MarketCombination {
	if combination == nil {
		return nil
	}
	items := make([]*apiclient.MarketCombinationItem, 0, len(combination.Items))
	for _, item := range combination.Items {
		items = append(items, &apiclient.MarketCombinationItem{
			Ordinal:           item.Ordinal,
			EventConditionId:  item.EventConditionID,
			EventTitle:        item.EventTitle,
			EventLogo:         item.EventLogo,
			MarketConditionId: item.MarketConditionID,
			MarketTitle:       item.MarketTitle,
			MarketLogo:        item.MarketLogo,
			IsYes:             item.IsYes,
			OutcomeLabel:      item.OutcomeLabel,
		})
	}
	return &apiclient.MarketCombination{
		Id:             combination.ID,
		Name:           combination.Name,
		Revision:       combination.Revision,
		Items:          items,
		CreatedAt:      combination.CreatedAt.Unix(),
		UpdatedAt:      combination.UpdatedAt.Unix(),
		OwnerAccountId: combination.OwnerAccountID,
	}
}

func marketCombinationRPCError(err error) error {
	switch {
	case errors.Is(err, wormstore.ErrMarketCombinationNotFound):
		return status.Error(codes.NotFound, "market combination not found")
	case errors.Is(err, wormstore.ErrMarketCombinationExists):
		return status.Error(codes.AlreadyExists, "market combination name already exists")
	case errors.Is(err, wormstore.ErrMarketCombinationRevision):
		return status.Error(codes.Aborted, "market combination revision conflict")
	case errors.Is(err, wormstore.ErrInvalidMarketCombination):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "market combination store operation failed")
	}
}
